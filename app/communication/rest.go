package communication

import (
	"encoding/json"
	"fmt"
	"go_app/config"
	"go_app/utils"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	deviceIPs   = config.AnalyzerIPs
	restLock    sync.RWMutex
	metersLock  sync.RWMutex
	restData    = make(map[string][]map[string]interface{})
	metersData  = make(map[string][]map[string]interface{})
	client      = &http.Client{Timeout: 2 * time.Second}
	deviceState sync.Map // key: "REST device_3" / "METERS device_3", value: bool (true=offline, false=online)
)

// RunRestCommunication starts both measurement and meter polling goroutines
func RunRestCommunication() {
	for i := 1; i <= len(deviceIPs); i++ {
		id := i
		utils.Go(fmt.Sprintf("REST device_%d measurements", id), func() { fetchAndStoreRESTData(id) })
		utils.Go(fmt.Sprintf("METERS device_%d", id), func() { fetchAndStoreMetersData(id) })
	}
}

// --- MEASUREMENTS ---

func fetchAndStoreRESTData(deviceID int) {
	url := fmt.Sprintf("http://%s/api/v1/measurements", deviceIPs[deviceID-1])
	key := fmt.Sprintf("device_%d", deviceID)
	stateKey := fmt.Sprintf("REST %s", key)

	firstLog := true

	for {
		func() {
			defer utils.Catch(fmt.Sprintf("fetchAndStoreRESTData(device_%d) iteration", deviceID))()

			success := false
			for attempt := 1; attempt <= 3; attempt++ {
				if firstLog {
					utils.LogMessage(fmt.Sprintf("[%s] first conn attempt %d", stateKey, attempt))
				}

				req, _ := http.NewRequest("GET", url, nil)
				req.Header.Set("Accept-Encoding", "identity")

				resp, err := client.Do(req)
				if err != nil {
					time.Sleep(config.IntervalRestData)
					continue
				}

				func() {
					defer resp.Body.Close()

					if resp.StatusCode < 200 || resp.StatusCode >= 300 {
						return
					}

					var response map[string]interface{}
					if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
						return
					}
					if response == nil {
						return
					}

					timestamp := time.Now().UTC().Format(time.RFC3339Nano)
					if ts, ok := response["timestamp"]; ok {
						timestamp = normalizeTimestamp(ts)
					}

					items, ok := response["items"].([]interface{})
					if !ok {
						return
					}

					formatted := make([]map[string]interface{}, 0, len(items))
					for _, raw := range items {
						if item, ok := raw.(map[string]interface{}); ok {
							formatted = append(formatted, map[string]interface{}{
								"id":        item["id"],
								"value":     item["value"],
								"unit":      item["unit"],
								"timestamp": timestamp,
							})
						}
					}

					restLock.Lock()
					restData[key] = formatted
					restLock.Unlock()
					success = true
				}()

				if success {
					break
				}
				time.Sleep(config.IntervalRestData)
			}

			updateDeviceState(stateKey, success)
		}()

		firstLog = false
		time.Sleep(config.IntervalRestData)
	}
}

// --- METERS ---

func fetchAndStoreMetersData(deviceID int) {
	url := fmt.Sprintf("http://%s/api/v1/meters", deviceIPs[deviceID-1])
	key := fmt.Sprintf("device_%d", deviceID)
	stateKey := fmt.Sprintf("METERS %s", key)

	firstLog := true

	for {
		func() {
			defer utils.Catch(fmt.Sprintf("fetchAndStoreMetersData(device_%d) iteration", deviceID))()

			success := false
			for attempt := 1; attempt <= 3; attempt++ {
				if firstLog {
					utils.LogMessage(fmt.Sprintf("[%s] first conn attempt %d", stateKey, attempt))
				}

				req, _ := http.NewRequest("GET", url, nil)
				req.Header.Set("Accept-Encoding", "identity")

				resp, err := client.Do(req)
				if err != nil {
					time.Sleep(config.IntervalRestData)
					continue
				}

				func() {
					defer resp.Body.Close()

					if resp.StatusCode < 200 || resp.StatusCode >= 300 {
						return
					}

					var response map[string]interface{}
					if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
						return
					}
					if response == nil {
						return
					}

					timestamp := time.Now().UTC().Format(time.RFC3339Nano)
					if ts, ok := response["timestamp"]; ok {
						timestamp = normalizeTimestamp(ts)
					}

					items, ok := response["items"].([]interface{})
					if !ok {
						return
					}

					formatted := make([]map[string]interface{}, 0, len(items))
					for _, raw := range items {
						if item, ok := raw.(map[string]interface{}); ok {
							id := item["id"]
							if s, ok := id.(string); ok {
								id = strings.ReplaceAll(s, "-", "_")
							}
							formatted = append(formatted, map[string]interface{}{
								"id":        id,
								"value":     item["value"],
								"unit":      item["unit"],
								"timestamp": timestamp,
							})
						}
					}

					metersLock.Lock()
					metersData[key] = formatted
					metersLock.Unlock()
					success = true
				}()

				if success {
					break
				}
				time.Sleep(config.IntervalRestData)
			}

			updateDeviceState(stateKey, success)
		}()

		firstLog = false
		time.Sleep(config.IntervalRestData)
	}
}

// --- State logger helper ---

func updateDeviceState(stateKey string, success bool) {
	prev, _ := deviceState.LoadOrStore(stateKey, !success)
	if success {
		if prev.(bool) { // wcześniej było offline
			utils.LogMessage(fmt.Sprintf("[%s] is ONLINE again", stateKey))
			deviceState.Store(stateKey, false)
		}
	} else {
		if !prev.(bool) { // wcześniej było online
			utils.LogMessage(fmt.Sprintf("[%s] went OFFLINE", stateKey))
			deviceState.Store(stateKey, true)
		}
	}
}

// --- Read helpers ---

func GetRestData() map[string][]map[string]interface{} {
	restLock.RLock()
	defer restLock.RUnlock()
	return copyMapOfSlices(restData)
}

func GetMetersData() map[string][]map[string]interface{} {
	metersLock.RLock()
	defer metersLock.RUnlock()
	return copyMapOfSlices(metersData)
}

func copyMapOfSlices(input map[string][]map[string]interface{}) map[string][]map[string]interface{} {
	output := make(map[string][]map[string]interface{})
	for k, v := range input {
		output[k] = copySlice(v)
	}
	return output
}

func copySlice(input []map[string]interface{}) []map[string]interface{} {
	out := make([]map[string]interface{}, len(input))
	for i, item := range input {
		entry := make(map[string]interface{})
		for k, v := range item {
			entry[k] = v
		}
		out[i] = entry
	}
	return out
}
