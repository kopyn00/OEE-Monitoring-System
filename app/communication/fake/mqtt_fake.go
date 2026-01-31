package communication

import (
	"math/rand"
	"sync"
	"time"
)

var (
	lastElementTime    = time.Now()
	lastSpeedSignal    = time.Now()
	predkoscOn         = false
	elementOn          = false
	mqttFakeMutex      sync.Mutex
	nextElementDelayMs = rand.Intn(10000) + 2000

	totaliser1 float64 = 9451.0
	totaliser2 float64 = 2239.0
)

// GenerateMockMQTTData symuluje dane MQTT z dynamicznymi impulsami i elementami
func GenerateMockMQTTData() map[string]map[string]interface{} {
	mqttFakeMutex.Lock()
	defer mqttFakeMutex.Unlock()

	now := time.Now()
	timestamp := now.UTC().Format(time.RFC3339)

	if time.Since(lastSpeedSignal) > 200*time.Millisecond {
		predkoscOn = !predkoscOn
		lastSpeedSignal = now
	}

	if time.Since(lastElementTime) > time.Duration(nextElementDelayMs)*time.Millisecond {
		elementOn = true
		lastElementTime = now
		nextElementDelayMs = rand.Intn(10000) + 2000
	} else {
		elementOn = false
	}

	// Inkrementacja totaliserów — symulacja rzeczywistego przyrostu
	totaliser1 += 0.1
	totaliser2 += 0.05

	//utils.LogMessage(fmt.Sprintf("NOWY TIMESTAMP: %s", timestamp))

	return map[string]map[string]interface{}{
		"port1": {
			"maszyna_on/off":  false,
			"Elementy":        elementOn,
			"Predkosc_sygnal": predkoscOn,
			"is_valid":        true,
			"timestamp":       timestamp,
		},
		"port2": {
			"Dlugosc":   22648,
			"Szerokosc": 1286,
			"Wysokosc":  2205,
			"is_valid":  true,
			"timestamp": timestamp,
		},
		"port3": {
			"device_status": 0,
			"flow":          204,
			"is_valid":      true,
			"pressure":      669,
			"temperature":   2930,
			"timestamp":     timestamp,
			"totaliser":     totaliser1,
		},
		"port4": {
			"device_status": 0,
			"flow":          41,
			"is_valid":      true,
			"pressure":      668,
			"temperature":   2690,
			"timestamp":     timestamp,
			"totaliser":     totaliser2,
		},
	}
}

// ResetMQTTGeneratorState resetuje stan symulacji MQTT
func ResetMQTTGeneratorState() {
	mqttFakeMutex.Lock()
	defer mqttFakeMutex.Unlock()

	lastElementTime = time.Now()
	lastSpeedSignal = time.Now()
	nextElementDelayMs = rand.Intn(10000) + 2000
	predkoscOn = false
	elementOn = false
	totaliser1 = 9451.0
	totaliser2 = 2239.0
}

// ForceMQTTGeneratorUpdate wymusza aktualizację stanu generatora
func ForceMQTTGeneratorUpdate() {
	mqttFakeMutex.Lock()
	defer mqttFakeMutex.Unlock()

	lastSpeedSignal = time.Now().Add(-1 * time.Second)
	lastElementTime = time.Now().Add(-10 * time.Second)
}
