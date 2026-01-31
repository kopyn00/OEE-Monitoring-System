package config

import (
	"os"
	"time"
)

// Environment-based configuration
var (
	DbHost     = getEnv("DB_HOST", "localhost")
	DbPort     = getEnv("DB_PORT", "5432")
	DbUser     = getEnv("DB_USER", "admin")
	DbPassword = getEnv("DB_PASSWORD", "admin")
	DbName     = getEnv("TEST_DB_NAME", "test_monitoring_db_go")

	MqttBroker = getEnv("MQTT_BROKER", "10.10.22.10")
	MqttPort   = getEnv("MQTT_PORT", "1883")

	MqttTopics = []string{
		"balluff/cmtk/master1/iolink/devices/port1/data/fromdevice",
		"balluff/cmtk/master1/iolink/devices/port2/data/fromdevice",
		"balluff/cmtk/master1/iolink/devices/port3/data/fromdevice",
		"balluff/cmtk/master1/iolink/devices/port4/data/fromdevice",
		"balluff/cmtk/master2/iolink/devices/port0/data/fromdevice",
		"balluff/cmtk/master2/iolink/devices/port1/data/fromdevice",
		"balluff/cmtk/master2/iolink/devices/port2/data/fromdevice",
	}

	FlowPorts = []string{
	"master1/port3",
	"master1/port4",
	"master2/port0",
	"master2/port1",
	"master2/port2",
	}

	AnalyzerIPs = []string{
	getEnv("ANALYZER_IP01", "192.168.1.130"),
	getEnv("ANALYZER_IP02", "192.168.1.131"),
	getEnv("ANALYZER_IP03", "192.168.1.132"),
	}

	JsonWithBackup = map[string]bool{
    OeeFilePath:      true,
    SummaryFilePath:  true,
	}
)

const (
	// --- Ustawienia produkcyjne i urządzeń ---
	ProductionCycleDefault = 14.0 // domyślny cykl produkcji [elementy/min]
	FlowDeviceCount        = 5    // liczba urządzeń przepływowych (flow meters)
	MeasurementDeviceCount = 3    // liczba analizatorów energii elektrycznej

	// --- Interwały odczytu i aktualizacji danych ---
	IntervalMQTTData          = 50 * time.Millisecond  // okres odświeżania danych z MQTT
	IntervalRestData          = 100 * time.Millisecond // okres odświeżania danych z REST
	FlowUpdateInterval        = 10 * time.Second       // zapis danych przepływowych (flow) do DB/JSON
	MeasurementUpdateInterval = 10 * time.Second       // zapis danych pomiarowych (measurements) do DB/JSON
	MetersUpdateInterval      = 5 * time.Second        // zapis danych licznikowych (meters) do DB/JSON
	OEEUpdateInterval         = 5 * time.Second        // częstotliwość aktualizacji wskaźników OEE

	// --- Parametry obliczeń OEE ---
	AirFactor             = 1.0    // współczynnik przeliczeniowy powietrza (skalowanie totalisera)
	ImpulsyNaObrot        = 8      // liczba impulsów odpowiadających jednemu obrotowi czujnika
	ElementWindow         = 10     // długość bufora historii liczby elementów (ostatnie 10 odczytów) (nieużywany w aktualnej logice)
	IdleTimeoutSeconds    = 10      // po ilu sekundach braku elementów rozpoczyna się zliczanie postoju
	MaxChangeoverDuration = 10 * 60 // maksymalny czas (s), który może być zaliczony jako przezbrojenie zamiast zwykłego postoju

	// --- Ścieżki do plików ---
	SummaryFilePath         = "logs/summary.json"              // podsumowania zmian
	OeeFilePath             = "logs/oee.json"                  // dane OEE (stan bieżący)
	MqttOeeFilePath         = "logs/mqttOEE.json"              // surowe dane MQTT dla OEE
	MqttFlowFilePath        = "logs/mqttFlow.json"             // dane przepływów (flow) z MQTT
	MeasurementFilePath     = "logs/measurements.json"         // dane pomiarowe z REST
	MetersFilePath          = "logs/meters.json"               // dane licznikowe z REST
	SystemLogPath           = "logs/system.log"                // log systemowy aplikacji
	DefaultJsonFile         = "logs/system_report.json"        // plik JSON domyślny (nieużywany w aktualnej logice)
)

// CycleRule defines rules for dynamic cycle assignment
type CycleRule struct {
	MaxLength int
	MaxWidth  int
	CycleLPM  float64
}

// From excel file cycles
var CycleTable = []CycleRule{
	{MaxLength: 600, MaxWidth: 9999, CycleLPM: 15.0},    // <600 mm → 4s
	{MaxLength: 800, MaxWidth: 9999, CycleLPM: 12.875},  // <800 mm → ~4.66s
	{MaxLength: 1200, MaxWidth: 9999, CycleLPM: 12.0},   // <1200 mm → 5s
	{MaxLength: 99999, MaxWidth: 9999, CycleLPM: 7.06},  // >=1200 mm → 8.5s
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}