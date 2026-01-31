package communication

import (
	"go_app/config"
	"go_app/utils"
	"math"
	"math/rand"
	"strconv"
	"time"
)

func GenerateMockRestData() map[string][]map[string]interface{} {
	existing := utils.LoadFromJSONMapArray(config.FakeMeasurementFilePath)
	if len(existing) == 0 {
		return generateEmptyMeasurement()
	}

	updated := make(map[string][]map[string]interface{})
	for deviceID, records := range existing {
		newRecords := make([]map[string]interface{}, 0, len(records))
		for _, rec := range records {
			id := rec["id"].(string)
			value := utils.ToFloat(rec["value"])
			unit := rec["unit"].(string)

			// ±0.01% zmiana
			delta := value * 0.0001
			newValue := value + (rand.Float64()*2-1)*delta

			newRecords = append(newRecords, map[string]interface{}{
				"id":        id,
				"value":     round(newValue),
				"unit":      unit,
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			})
		}
		updated[deviceID] = newRecords
	}
	return updated
}

func GenerateMockMetersData() map[string][]map[string]interface{} {
	existing := utils.LoadFromJSONMapArray(config.FakeMetersFilePath)
	if len(existing) == 0 {
		return generateEmptyMeters()
	}

	updated := make(map[string][]map[string]interface{})
	for deviceID, records := range existing {
		newRecords := make([]map[string]interface{}, 0, len(records))
		for _, rec := range records {
			id := rec["id"].(string)
			unit := rec["unit"].(string)
			value := utils.ToFloat(rec["value"])

			// Dodaj losowy narastający wzrost (symulacja energii)
			increment := rand.Float64() * 0.1
			newValue := value + increment

			newRecords = append(newRecords, map[string]interface{}{
				"id":        id,
				"value":     round(newValue),
				"unit":      unit,
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			})
		}
		updated[deviceID] = newRecords
	}
	return updated
}

func generateEmptyMeasurement() map[string][]map[string]interface{} {
	mock := make(map[string][]map[string]interface{})
	for i := 1; i <= config.MeasurementDeviceCount; i++ {
		key := "device_" + strconv.Itoa(i)
		mock[key] = []map[string]interface{}{
			{"id": "u1", "value": 230.0, "unit": "V", "timestamp": time.Now().UTC().Format(time.RFC3339)},
			{"id": "i1", "value": 5.0, "unit": "A", "timestamp": time.Now().UTC().Format(time.RFC3339)},
		}
	}
	return mock
}

func generateEmptyMeters() map[string][]map[string]interface{} {
	mock := make(map[string][]map[string]interface{})
	for i := 1; i <= config.MeasurementDeviceCount; i++ {
		key := "device_" + strconv.Itoa(i)
		mock[key] = []map[string]interface{}{
			{"id": "t4_ea_pos", "value": 100.0, "unit": "kWh", "timestamp": time.Now().UTC().Format(time.RFC3339)},
			{"id": "t4_es", "value": 80.0, "unit": "kVarh", "timestamp": time.Now().UTC().Format(time.RFC3339)},
		}
	}
	return mock
}

func round(v float64) float64 {
	return math.Round(v*1000) / 1000
}
