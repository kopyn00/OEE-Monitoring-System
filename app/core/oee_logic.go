package core

import (
	"fmt"
	"go_app/config"
	"go_app/utils"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type CyclePeriod struct {
	StartTime      time.Time
	EndTime        time.Time
	CycleLPM       float64
	ElementCounter int
	WorkSeconds    float64
}

type OeeFileFlat struct {
	Timestamp     string        `json:"timestamp"`
	OEE           OeeSection    	`json:"oee"`
	Product       OeeProduct    `json:"product"`
	Internal      OeeInternal   `json:"internal"`
	HelpersAir    HelpersAir    `json:"helpers_air"`
	HelpersEnergy HelpersEnergy `json:"helpers_energy"`
}

type OeeProduct struct {
	DlugoscCalc   float64 `json:"dlugosc_calc"`
	SzerokoscCalc float64 `json:"szerokosc_calc"`
	WysokoscCalc  float64 `json:"wysokosc_calc"`
	Cykl          float64 `json:"cykl"`
}

// Uwaga: start_measurement / element_last_time przeniesione do internal
type OeeInternal struct {
	StartMeasurement      	string        `json:"start_measurement"`
	ElementLastTime       	string        `json:"element_last_time"`
	ImpulsesCount         	int           `json:"impulses_count"`
	CurrentCycleElementCnt	int           `json:"current_cycle_element_cnt"`
	CurrentCycleStart     	string        `json:"current_cycle_start"`
	CurrentCycleValue     	float64       `json:"current_cycle_value"`
	CycleHistory          	[]CyclePeriod `json:"cycle_history"`
	CurrentCycleWorkSeconds	float64 `json:"current_cycle_work_seconds"`
	PrevElement            	bool          `json:"prev_element"`
	PrevSpeed              	bool          `json:"prev_speed"`
	PauseStartTime         	*string       `json:"pause_start_time"`
	TotalPause             	float64       `json:"total_pause"`
	AirBaseline            	float64       `json:"air_baseline"`    // litry (L)
	EnergyBaseline         	float64       `json:"energy_baseline"` // W
	FirstElementDetected   	bool          `json:"first_element_detected"`
	LastCycle              	float64       `json:"last_cycle"`
	LastWydajnosc          	float64       `json:"last_wydajnosc"`
	LastWydajnoscFinal     	float64       `json:"last_wydajnosc_final"`
	LastDostepnosc         	float64       `json:"last_dostepnosc"`
	LastCycleFinal         	float64       `json:"last_cycle_final"`
	ElementsUsed           	int           `json:"elements_used"`
	OeeTemp                	float64       `json:"oee_temp"`
	WydajnoscTemp          	float64       `json:"wydajnosc_temp"`
	DostepnoscTemp         	float64       `json:"dostepnosc_temp"`
}

type HelpersAir struct {
	Baseline             float64            `json:"baseline"`
	Factor               float64            `json:"factor"`
	TotalRawBeforeFactor float64            `json:"total_raw_before_factor"`
	TotalCurrentM3        float64            `json:"total_current_M3"`
	PortsRaw             map[string]float64 `json:"-"` // dynamiczne klucze: air_port_masterX/portY_raw
}

type HelpersEnergy struct {
	Baseline       float64            `json:"baseline"`
	TotalCurrentW float64            `json:"total_current_W"`
	DevicesW      map[string]float64 `json:"-"` // dynamiczne klucze: energy_device_N_ea_pos_total_W
}

type OeeSection struct {
	CzasPomiaru       		float64 `json:"czas_pomiaru"`
	CzasPracy         		float64 `json:"czas_pracy"`
	CzasPostoju       		float64 `json:"czas_postoju"`
	CzasPrzezbrojenia 		float64 `json:"czas_przezbrojenia"`
	CzasPrzezbrojeniaTemp 	float64 `json:"czas_przezbrojenia_temp"`
	IloscElementow    		int     `json:"ilosc_elementow"`
	Dostepnosc        		float64 `json:"dostepnosc"`
	Wydajnosc         		float64 `json:"wydajnosc"`
	Jakosc            		float64 `json:"jakosc"`
	OEE               		float64 `json:"oee"`

	// KPI kosztowe w "oee"
	PowietrzeL  float64 `json:"powietrze_L"`
	EnergyW    	float64 `json:"energia_W"`
	M3naSzt      float64 `json:"M3_na_szt"`
	WNaSzt     	float64 `json:"W_na_szt"`

	// Status przerzucony do "oee"
	StatusMaszyny bool `json:"status_maszyny"`
	StatusPracy   bool `json:"status_pracy"`

	PredkoscObrotnica float64 `json:"predkosc_obrotnica"`
}

var (
	calcLock               sync.Mutex
	impulsesCount          int
	lastImpulse            = time.Now().UTC()
	prevSpeed              bool
	prevSignal             bool
	prevElement            bool
	lastSaved              int
	firstRunFlag           atomic.Bool
	cycleJustChanged       atomic.Bool
	OeeReadyChan           = make(chan struct{}, 1)
	shouldStoreToDB        = false
	lastElementCountOEE    int
	lastWydajnosc          float64
	lastElementCount       int
	elementHistory         = []int{}
	startWydajnoscOnce     sync.Once
	startDostepnoscOnce    sync.Once
	firstElementDetected   bool
	lastPomiar             float64
	lastPostoj             float64
	lastDostepnosc         float64
	lastCycle              float64 = 0.0
	resetRequested         atomic.Bool
	cycleHistory                    = []CyclePeriod{}
	currentCycleStartTime  time.Time = time.Now().UTC()
	currentCycleElementCnt int
	currentCycleValue      float64 = config.ProductionCycleDefault
	startCostOnce          sync.Once
	energyBaselineW       float64
	airBaselineMeters3      float64
	costBaselineSet        atomic.Bool
	lastWorkTick            time.Time
	currentCycleWorkSeconds float64

	CalculatedData = map[string]interface{}{
		"Predkosc_obrotnica":            0.0,
		"ilosc_elementow":               0,
		"czas_pracy":                    0.0,
		"czas_postoju":                  0.0,
		"czas_przezbrojenia":            0.0,
		"czas_pomiaru":                  0.0,
		"Dlugosc_calc":                  0.0,
		"Szerokosc_calc":                0.0,
		"Wysokosc_calc":                 0.0,
		"status_maszyny":                false,
		"cykl":                          config.ProductionCycleDefault,
		"dostepnosc_temp":               100.0,
		"wydajnosc_temp":                100.0,
		"jakosc_temp":                   100.0,
		"oee_temp":                      100.0,
		"dostepnosc":                    0.0,
		"wydajnosc":                     0.0,
		"jakosc":                        0.0,
		"oee":                           0.0,
		"TotalPause_internal":           0.0,
		"PauseStartTime_internal":       "",
		"ElementLastTime_internal":      "",
		"StartMeasurement_internal":     "",
		"lastWydajnosc_internal":        0.0,
		"lastDostepnosc_internal":       0.0,
		"lastCycle_internal":            0.0,
		"lastElementCount_internal":     0,
		"firstElementDetected_internal": false,
		"impulsesCount_internal":        0,
		"prevSpeed_internal":            false,
		"prevElement_internal":          false,
		"lastWydajnoscFinal_internal":   0.0,
		"lastCycleFinal_internal":       0.0,
		"czas_przezbrojenia_temp":       0.0,
		"status_pracy":                  false,
		"energia_W":                    0.0, // suma W od początku zmiany
		"powietrze_L":                   0.0, // suma litrów od początku zmiany
		"W_na_szt":                     0.0, // energia na sztukę (narastająco)
		"M3_na_szt":                      0.0, // powietrze na sztukę (narastająco)
		"energyBaseline_internal":       0.0,
		"airBaseline_internal":          0.0,
	}

	CzasPomiarowy = struct {
		StartMeasurement         time.Time
		ElementLastTime          time.Time
		PauseStartTime           *time.Time
		TotalPause               float64
		PauseStartTotal          float64
		PauseStartChangeoverTemp float64
	}{
		StartMeasurement: time.Now().UTC(),
		ElementLastTime:  time.Now().UTC(),
		PauseStartTime:   nil,
		TotalPause:       0.0,
	}
)

// --- Gettery zgodne ze starym (root) i nowym (oee.{...}) layoutem ---

func getOeeFloat(data map[string]interface{}, key string) float64 {
	if oee, ok := data["oee"].(map[string]interface{}); ok {
		if v, ok := oee[key]; ok {
			return utils.ToFloat(v)
		}
	}
	return utils.ToFloat(data[key])
}

func getOeeInt(data map[string]interface{}, key string) int {
	if oee, ok := data["oee"].(map[string]interface{}); ok {
		if v, ok := oee[key]; ok {
			return utils.ToInt(v)
		}
	}
	return utils.ToInt(data[key])
}

func getBool(data map[string]interface{}, key string) bool {
	if oee, ok := data["oee"].(map[string]interface{}); ok {
		if v, ok := oee[key]; ok {
			return utils.ToBool(v)
		}
	}
	return utils.ToBool(data[key])
}

func ScheduleReset() {
	resetRequested.Store(true)
}

func IsResetScheduled() bool {
	return resetRequested.Load()
}

func ClearResetFlag() {
	resetRequested.Store(false)
}

func ResetOeeStateAndFile(path string) {
	ResetOeeState() // ma własny lock

	var of OeeFileFlat
	calcLock.Lock()
	now := time.Now().UTC()

	// jeżeli helpery już są, użyj ich w pierwszej kolejności
	var ha, he map[string]interface{}
	if m, ok := CalculatedData["helpers_air"].(map[string]interface{}); ok {
		ha = m
	}
	if m, ok := CalculatedData["helpers_energy"].(map[string]interface{}); ok {
		he = m
	}

	// bezpieczny getter: spróbuj z mapy m[key], potem z CalculatedData[fallbackKey], inaczej 0
	fFrom := func(m map[string]interface{}, key string, fallbackKey string) float64 {
		if m != nil {
			if v, ok := m[key]; ok && v != nil {
				return utils.ToFloat(v)
			}
		}
		if v, ok := CalculatedData[fallbackKey]; ok && v != nil {
			return utils.ToFloat(v)
		}
		return 0
	}

	var pauseStrPtr *string
	if CzasPomiarowy.PauseStartTime != nil {
		s := CzasPomiarowy.PauseStartTime.UTC().Format(time.RFC3339)
		pauseStrPtr = &s
	}

	of = OeeFileFlat{
		Timestamp: now.Format(time.RFC3339Nano),
		OEE: OeeSection{
			CzasPomiaru:           utils.ToFloat(CalculatedData["czas_pomiaru"]),
			CzasPracy:             utils.ToFloat(CalculatedData["czas_pracy"]),
			CzasPostoju:           utils.ToFloat(CalculatedData["czas_postoju"]),
			CzasPrzezbrojenia:     utils.ToFloat(CalculatedData["czas_przezbrojenia"]),
			CzasPrzezbrojeniaTemp: utils.ToFloat(CalculatedData["czas_przezbrojenia_temp"]),
			IloscElementow:        utils.ToInt(CalculatedData["ilosc_elementow"]),
			Dostepnosc:            utils.ToFloat(CalculatedData["dostepnosc"]),
			Wydajnosc:             utils.ToFloat(CalculatedData["wydajnosc"]),
			Jakosc:                utils.ToFloat(CalculatedData["jakosc"]),
			OEE:                   utils.ToFloat(CalculatedData["oee"]),
			PowietrzeL:            utils.ToFloat(CalculatedData["powietrze_L"]),
			EnergyW:               utils.ToFloat(CalculatedData["energia_W"]),
			M3naSzt:               utils.ToFloat(CalculatedData["M3_na_szt"]),
			WNaSzt:                utils.ToFloat(CalculatedData["W_na_szt"]),
			StatusMaszyny:         utils.ToBool(CalculatedData["status_maszyny"]),
			StatusPracy:           utils.ToBool(CalculatedData["status_pracy"]),
			PredkoscObrotnica:     utils.ToFloat(CalculatedData["Predkosc_obrotnica"]),
		},
		Product: OeeProduct{
			DlugoscCalc:   utils.ToFloat(CalculatedData["Dlugosc_calc"]),
			SzerokoscCalc: utils.ToFloat(CalculatedData["Szerokosc_calc"]),
			WysokoscCalc:  utils.ToFloat(CalculatedData["Wysokosc_calc"]),
			Cykl:          utils.ToFloat(CalculatedData["cykl"]),
		},
		Internal: OeeInternal{
			StartMeasurement:       CzasPomiarowy.StartMeasurement.UTC().Format(time.RFC3339),
			ElementLastTime:        CzasPomiarowy.ElementLastTime.UTC().Format(time.RFC3339),
			ImpulsesCount:          impulsesCount,
			CurrentCycleElementCnt: currentCycleElementCnt,
			CurrentCycleStart:      currentCycleStartTime.UTC().Format(time.RFC3339),
			CurrentCycleValue:      currentCycleValue,
			CycleHistory:           cycleHistory,
			PrevElement:            prevElement,
			PrevSpeed:              prevSpeed,
			PauseStartTime:         pauseStrPtr,
			TotalPause:             CzasPomiarowy.TotalPause,
			AirBaseline:            airBaselineMeters3,
			EnergyBaseline:         energyBaselineW,
			FirstElementDetected:   firstElementDetected,
			LastCycle:              lastCycle,
			LastWydajnosc:          lastWydajnosc,
			LastWydajnoscFinal:     utils.ToFloat(CalculatedData["lastWydajnoscFinal_internal"]),
			LastDostepnosc:         lastDostepnosc,
			LastCycleFinal:         utils.ToFloat(CalculatedData["lastCycleFinal_internal"]),
			ElementsUsed:           utils.ToInt(CalculatedData["elements_used"]),
			OeeTemp:                utils.ToFloat(CalculatedData["oee_temp"]),
			WydajnoscTemp:          utils.ToFloat(CalculatedData["wydajnosc_temp"]),
			DostepnoscTemp:         utils.ToFloat(CalculatedData["dostepnosc_temp"]),
			CurrentCycleWorkSeconds: currentCycleWorkSeconds,
		},
		HelpersAir: HelpersAir{
			Baseline:             fFrom(ha, "baseline",                "airBaseline_internal"),
			Factor:               config.AirFactor,
			TotalRawBeforeFactor: fFrom(ha, "total_raw_before_factor", "helpers_air_total_raw_before_factor"),
			TotalCurrentM3:        fFrom(ha, "total_current_M3",        "helpers_air_total_current_M3"),
			PortsRaw:             map[string]float64{},
		},
		HelpersEnergy: HelpersEnergy{
			Baseline:      fFrom(he, "baseline",        "energyBaseline_internal"),
			TotalCurrentW: fFrom(he, "total_current_W", "helpers_energy_total_current_W"),
			DevicesW:      map[string]float64{},
		},
	}

	// detale portów/urządzeń tylko jeśli są nie-nil
	if ha != nil {
		for k, v := range ha {
			if strings.HasPrefix(k, "air_port_") && strings.HasSuffix(k, "_raw") && v != nil {
				of.HelpersAir.PortsRaw[k] = utils.ToFloat(v)
			}
		}
	}
	if he != nil {
		for k, v := range he {
			if strings.HasPrefix(k, "energy_device_") && strings.HasSuffix(k, "_ea_pos_total_W") && v != nil {
				of.HelpersEnergy.DevicesW[k] = utils.ToFloat(v)
			}
		}
	}
	calcLock.Unlock()

	if err := saveOeeFlat(of, path); err != nil {
		utils.LogMessage("[OEE] Reset save error: " + err.Error())
		return
	}
	utils.LogMessage("[OEE] Reset state saved to OEE file (flat)")
}

func CalculateData(mqtt map[string]map[string]interface{}) {
	calcLock.Lock()
	defer calcLock.Unlock()

	port1 := mqtt["master1/port1"]
	port2 := mqtt["master1/port2"]
	// if len(port1) == 0 || len(port2) == 0 {
	// 	return
	// }

	now := time.Now().UTC()
	CalculatedData["status_maszyny"] = utils.ToBool(port1["maszyna_on/off"])
	CalculatedData["timestamp"] = now.Format(time.RFC3339Nano)

	updateImpulseCount(port1)
	detectElement(port1, now)
	updateDimensions(port2)
	updateCycleFromDimensions()
	updateElementHistory()
	updateMeasurementTimes(now)
	updateIdleTime(now)
	updateSpeed(now)
	updateDimensions(port2)
	checkIfShouldStore()
	updateStubbedMetrics()
	UpdateFinalOeeMetrics()

	startWydajnoscOnce.Do(func() {
		StartWydajnoscTempUpdater(config.OeeFilePath, 10*time.Second)
	})
	startDostepnoscOnce.Do(func() {
		StartDostepnoscTempUpdater(config.OeeFilePath, 10*time.Second)
	})
	startCostOnce.Do(func() {
		StartCostUpdater(config.MetersFilePath, config.MqttFlowFilePath, 10*time.Second)
	})

	select {
	case OeeReadyChan <- struct{}{}:
	default:
	}
}

func determineCycleRate(length, width float64) float64 {
	for _, entry := range config.CycleTable {
		if length <= float64(entry.MaxLength) && width <= float64(entry.MaxWidth) {
			return entry.CycleLPM
		}
	}
	return config.ProductionCycleDefault
}

func updateCycleFromDimensions() {
	d := utils.ToFloat(CalculatedData["Dlugosc_calc"])
	s := utils.ToFloat(CalculatedData["Szerokosc_calc"])
	newCycle := determineCycleRate(d, s)

	if math.Abs(newCycle-currentCycleValue) > 0.01 {
		now := time.Now().UTC()

		// zamknij poprzedni okres cyklu i przenieś skumulowany czas pracy
		cycleHistory = append(cycleHistory, CyclePeriod{
			StartTime:      currentCycleStartTime,
			EndTime:        now,
			CycleLPM:       currentCycleValue,
			ElementCounter: currentCycleElementCnt,
			WorkSeconds:    currentCycleWorkSeconds, // KLUCZOWE
		})

		// rozpocznij nowy okres
		currentCycleStartTime   = now
		currentCycleValue       = newCycle
		currentCycleElementCnt  = 0
		currentCycleWorkSeconds = 0
		lastWorkTick            = now // uniknij „dociążenia” poprzednim dt
		cycleJustChanged.Store(true)
	}

	CalculatedData["cykl"] = newCycle
}

func updateImpulseCount(port map[string]interface{}) {
	speedSignal := utils.ToBool(port["Predkosc_sygnal"])
	if speedSignal && !prevSpeed {
		impulsesCount++
	}
	prevSpeed = speedSignal
}

func updateMeasurementTimes(now time.Time) {
	CalculatedData["czas_pomiaru"] = now.Sub(CzasPomiarowy.StartMeasurement).Seconds()
}

func detectElement(port map[string]interface{}, now time.Time) {
	elementSignal := utils.ToBool(port["Elementy"])
	if elementSignal && !prevElement {
		CalculatedData["ilosc_elementow"] = utils.ToInt(CalculatedData["ilosc_elementow"]) + 1
		currentCycleElementCnt++
		if !firstElementDetected {
			firstElementDetected = true
		} else {
			CzasPomiarowy.ElementLastTime = now
		}
	}
	prevElement = elementSignal
}

func updateIdleTime(now time.Time) {
	currentCount := utils.ToInt(CalculatedData["ilosc_elementow"])
	currentSignal := prevElement
	idleDuration := now.Sub(CzasPomiarowy.ElementLastTime).Seconds()
	cycle := utils.ToFloat(CalculatedData["cykl"])

	// --- Wykrywanie zbocza narastającego ---
	risingEdge := !prevSignal && currentSignal
	prevSignal = currentSignal

	// --- Pierwszy element nie wykryty → wszystko stoi ---
	if !firstElementDetected {
		pomiar := utils.ToFloat(CalculatedData["czas_pomiaru"])
		CalculatedData["czas_postoju"] = pomiar
		CalculatedData["czas_pracy"] = 0.0
		CalculatedData["czas_przezbrojenia"] = 0.0
		CalculatedData["czas_przezbrojenia_temp"] = 0.0
		CalculatedData["status_pracy"] = false
		lastCycle = cycle

		// inicjalizacja znacznika dla akumulacji czasu pracy
		if lastWorkTick.IsZero() {
			lastWorkTick = now
		}
		return
	}

	// --- START PAUZY ---
	if !currentSignal && idleDuration >= float64(config.IdleTimeoutSeconds) {
		if CzasPomiarowy.PauseStartTime == nil {
			start := CzasPomiarowy.ElementLastTime.Add(
				time.Duration(config.IdleTimeoutSeconds) * time.Second,
			)
			CzasPomiarowy.PauseStartTime = &start
			CzasPomiarowy.PauseStartTotal = CzasPomiarowy.TotalPause
			CzasPomiarowy.PauseStartChangeoverTemp = utils.ToFloat(CalculatedData["czas_przezbrojenia"])
		}
	}

	// --- KONIEC PAUZY – wykryto nowy element (tylko przy zboczu narastającym) ---
	if risingEdge && currentCount != lastElementCount {
		if CzasPomiarowy.PauseStartTime != nil {
			// snapshot wartości przed resetem
			ps := *CzasPomiarowy.PauseStartTime
			startEpoch := float64(ps.Unix())
			endEpoch := float64(now.Unix())
			dur := now.Sub(ps).Seconds()
			changeoverTemp := dur

			if endEpoch < startEpoch {
				startEpoch, endEpoch = endEpoch, startEpoch
			}

			if math.Abs(cycle-lastCycle) > 0.01 {
				// --- PRZEZBROJENIE POTWIERDZONE ---
				if dur <= config.MaxChangeoverDuration {
					CalculatedData["czas_przezbrojenia"] =
						CzasPomiarowy.PauseStartChangeoverTemp + changeoverTemp
					CalculatedData["czas_postoju"] = CzasPomiarowy.PauseStartTotal

					// wywołanie gorutyny z użyciem snapshotów
					utils.Go("AdjustIdleToChangeover", func() {
						AdjustIdleToChangeover(startEpoch, endEpoch, changeoverTemp)
					})
				} else {
					// zbyt długie – traktujemy jako zwykły postój
					CzasPomiarowy.TotalPause += dur
					CalculatedData["czas_przezbrojenia_temp"] = 0.0
				}
			} else {
				// zwykła pauza
				CzasPomiarowy.TotalPause += dur
				CalculatedData["czas_przezbrojenia_temp"] = 0.0
			}

			// reset stanu pauzy
			CzasPomiarowy.PauseStartTime = nil
			CzasPomiarowy.PauseStartTotal = 0
			CzasPomiarowy.PauseStartChangeoverTemp = 0
			lastElementCount = currentCount
			lastCycle = cycle
		}

		// aktualizacja tymczasowego przezbrojenia
		CalculatedData["czas_przezbrojenia_temp"] =
			utils.ToFloat(CalculatedData["czas_przezbrojenia"])
	}

	// --- ZLICZANIE PAUZY (równolegle czas_postoju i czas_przezbrojenia_temp) ---
	if CzasPomiarowy.PauseStartTime != nil {
		pausedNow := now.Sub(*CzasPomiarowy.PauseStartTime).Seconds()
		CalculatedData["czas_postoju"] =
			CzasPomiarowy.PauseStartTotal + pausedNow
		CalculatedData["czas_przezbrojenia_temp"] =
			CzasPomiarowy.PauseStartChangeoverTemp + pausedNow
	} else {
		CalculatedData["czas_postoju"] = CzasPomiarowy.TotalPause
	}

	// --- LICZENIE CZASU PRACY ---
	pomiar := utils.ToFloat(CalculatedData["czas_pomiaru"])
	przezbrojenie := utils.ToFloat(CalculatedData["czas_przezbrojenia"])
	postoj := utils.ToFloat(CalculatedData["czas_postoju"])
	czasPracy := pomiar - postoj - przezbrojenie
	if czasPracy < 0 {
		czasPracy = 0
	}
	CalculatedData["czas_pracy"] = czasPracy

	// --- STATUS PRACY ---
	isCountingWork := firstElementDetected && CzasPomiarowy.PauseStartTime == nil
	CalculatedData["status_pracy"] = isCountingWork

	// --- (NOWE) Akumulacja czasu pracy tylko gdy faktycznie pracujemy ---
	if lastWorkTick.IsZero() {
		lastWorkTick = now
	}
	dt := now.Sub(lastWorkTick).Seconds()
	if dt < 0 {
		dt = 0
	}
	if isCountingWork {
		currentCycleWorkSeconds += dt
	}
	lastWorkTick = now
}

func updateSpeed(now time.Time) {
	if now.Sub(lastImpulse) >= time.Second {
		obroty := float64(impulsesCount) / config.ImpulsyNaObrot
		CalculatedData["Predkosc_obrotnica"] = obroty * 60
		impulsesCount = 0
		lastImpulse = now
	}
}

func updateDimensions(port map[string]interface{}) {
	CalculatedData["Dlugosc_calc"] = utils.ToFloat(port["Dlugosc"])/10 - 20
	CalculatedData["Szerokosc_calc"] = utils.ToFloat(port["Szerokosc"])/10 + 100
	CalculatedData["Wysokosc_calc"] = utils.ToFloat(port["Wysokosc"])/100 - 5.5
}

func checkIfShouldStore() {
	ilosc := utils.ToInt(CalculatedData["ilosc_elementow"])
	if ilosc > lastSaved && ilosc > 0 {
		shouldStoreToDB = true
		lastSaved = ilosc
	}
}

func updateStubbedMetrics() {
	CalculatedData["dostepnosc_temp"] = clamp(calculateDostepnoscTemp())
	CalculatedData["wydajnosc_temp"] = clamp(calculateWydajnoscTemp())
	CalculatedData["jakosc_temp"] = clamp(calculateJakoscTemp())
	CalculatedData["oee_temp"] = clamp(calculateOEETemp())
}

func StartDostepnoscTempUpdater(path string, interval time.Duration) {
	utils.Go("OEE DostepnoscTempUpdater", func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			func() {
				defer utils.Catch("OEE DostepnoscTempUpdater iteration")()

				data := utils.LoadFromJSON(path)

				calcLock.Lock()
				currPomiar := getOeeFloat(data, "czas_pomiaru")
				currPostoj := getOeeFloat(data, "czas_postoju")

				deltaPomiar := currPomiar - lastPomiar
				deltaPostoj := currPostoj - lastPostoj

				if deltaPomiar <= 0 {
					lastDostepnosc = 0.0
				} else {
					lastDostepnosc = (deltaPomiar - deltaPostoj) / deltaPomiar
				}

				lastPomiar = currPomiar
				lastPostoj = currPostoj
				calcLock.Unlock()
			}()
		}
	})
}

func StartWydajnoscTempUpdater(path string, interval time.Duration) {
	utils.Go("OEE WydajnoscTempUpdater", func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			func() {
				defer utils.Catch("OEE WydajnoscTempUpdater iteration")()

				// Jeśli właśnie zmienił się cykl, pomiń jedną iterację (bez zmian).
				if cycleJustChanged.Load() {
					cycleJustChanged.Store(false)
					return
				}

				// Odczytaj „chwilowy” stan z pliku
				data := utils.LoadFromJSON(path)

				// --- policz liczbę sztuk w interwale
				calcLock.Lock()
				currCount := getOeeInt(data, "ilosc_elementow")
				delta := currCount - lastElementCountOEE
				if delta < 0 {
					delta = 0 // osłona na reset/licznik wstecz
				}
				lastElementCountOEE = currCount

				// cykl i status_pracy z bieżącego stanu
				cyklLpm := utils.ToFloat(CalculatedData["cykl"])
				isWorking := getBool(data, "status_pracy")
				calcLock.Unlock()

				// --- oczekiwane sztuki liczymy WYŁĄCZNIE jeśli maszyna faktycznie pracuje
				var expected float64
				if isWorking && cyklLpm > 0 {
					expected = (cyklLpm / 60.0) * interval.Seconds()
				} else {
					expected = 0
				}

				// „chwilowa” wydajność z ostatniego okna czasu
				calcLock.Lock()
				if expected <= 0 {
					lastWydajnosc = 0.0
				} else {
					lastWydajnosc = float64(delta) / expected
				}
				calcLock.Unlock()
			}()
		}
	})
}

func updateElementHistory() {
	ilosc := utils.ToInt(CalculatedData["ilosc_elementow"])
	elementHistory = append(elementHistory, ilosc)
	if len(elementHistory) > config.ElementWindow {
		elementHistory = elementHistory[1:]
	}
}

func calculateDostepnoscTemp() float64 {
	return math.Round(lastDostepnosc*10000) / 10000
}

func calculateWydajnoscTemp() float64 {
	return math.Round(lastWydajnosc*10000) / 10000
}

func calculateJakoscTemp() float64 {
	return 1.0
}

func calculateOEETemp() float64 {
	dostepnosc := utils.ToFloat(CalculatedData["dostepnosc_temp"])
	wydajnosc := utils.ToFloat(CalculatedData["wydajnosc_temp"])
	jakosc := utils.ToFloat(CalculatedData["jakosc_temp"])
	return math.Round(dostepnosc*wydajnosc*jakosc*10000) / 10000
}

func calculateDostepnosc() float64 {
	czasPomiaru := utils.ToFloat(CalculatedData["czas_pomiaru"])
	czasPostoju := utils.ToFloat(CalculatedData["czas_postoju"])
	czasPrzezbrojenia := utils.ToFloat(CalculatedData["czas_przezbrojenia"])

	aktywnyCzas := czasPomiaru - czasPostoju - czasPrzezbrojenia
	if czasPomiaru <= 0 {
		return 0.0
	}
	return math.Round((aktywnyCzas/czasPomiaru)*10000) / 10000
}

func calculateWydajnosc() float64 {
	now := time.Now().UTC()

	var totalExpected float64
	var totalActual   float64

	// Zsumuj oczekiwaną produkcję tylko z realnego czasu pracy w zamkniętych okresach
	for _, p := range cycleHistory {
		if p.CycleLPM > 0 && p.WorkSeconds > 0 {
			totalExpected += p.WorkSeconds / (60.0 / p.CycleLPM)
			totalActual   += float64(p.ElementCounter)
		}
	}

	// Bieżący, otwarty okres cyklu
	currWork := currentCycleWorkSeconds

	// Dolicz ewentualne sekundy pracy, które upłynęły od lastWorkTick do „teraz”
	isWorking := firstElementDetected && CzasPomiarowy.PauseStartTime == nil
	if isWorking && !lastWorkTick.IsZero() {
		dt := now.Sub(lastWorkTick).Seconds()
		if dt > 0 {
			currWork += dt
		}
	}

	if currentCycleValue > 0 && currWork > 0 {
		totalExpected += currWork / (60.0 / currentCycleValue)
		totalActual   += float64(currentCycleElementCnt)
	}

	if totalExpected <= 0 {
		return 0.0
	}
	return math.Round((totalActual/totalExpected)*10000) / 10000
}

func calculateJakosc() float64 {
	// Zakładamy 100% jakości
	return 1.0
}

func calculateOEE() float64 {
	d := calculateDostepnosc()
	w := calculateWydajnosc()
	j := calculateJakosc()
	return math.Round(d*w*j*10000) / 10000
}

func UpdateFinalOeeMetrics() {
	CalculatedData["dostepnosc"] = clamp(calculateDostepnosc())
	CalculatedData["wydajnosc"] = clamp(calculateWydajnosc())
	CalculatedData["jakosc"] = clamp(calculateJakosc())
	CalculatedData["oee"] = clamp(calculateOEE())
}

func ResetOeeState() {
	calcLock.Lock()
	defer calcLock.Unlock()

	CalculatedData["ilosc_elementow"] = 0
	CalculatedData["czas_pracy"] = 0.0
	CalculatedData["czas_postoju"] = 0.0
	CalculatedData["czas_przezbrojenia"] = 0.0
	CalculatedData["czas_przezbrojenia_temp"] = 0.0
	CalculatedData["czas_pomiaru"] = 0.0
	CalculatedData["dostepnosc"] = 0.0
	CalculatedData["wydajnosc"] = 0.0
	CalculatedData["jakosc"] = 0.0
	CalculatedData["oee"] = 0.0
	CalculatedData["energia_W"] = 0.0
	CalculatedData["powietrze_L"] = 0.0
	CalculatedData["W_na_szt"] = 0.0
	CalculatedData["M3_na_szt"] = 0.0

	energyBaselineW = 0
	airBaselineMeters3 = 0
	costBaselineSet.Store(false)

	CzasPomiarowy.StartMeasurement = time.Now().UTC()
	CzasPomiarowy.ElementLastTime = time.Now().UTC()
	CzasPomiarowy.PauseStartTime = nil
	CzasPomiarowy.TotalPause = 0.0

	impulsesCount = 0
	prevSpeed = false
	prevElement = false
	lastSaved = 0
	lastElementCountOEE = 0
	lastWydajnosc = 0.0
	lastElementCount = 0
	elementHistory = []int{}
	lastPomiar = 0.0
	lastPostoj = 0.0
	lastDostepnosc = 0.0
	firstElementDetected = false

	// Reset cyklu
	cycleHistory = []CyclePeriod{}
	currentCycleStartTime = time.Now().UTC()
	currentCycleElementCnt = 0
	currentCycleValue = config.ProductionCycleDefault
	currentCycleWorkSeconds = 0
	lastWorkTick = time.Now().UTC()

	utils.LogMessage("[OEE] OEE data reset after shift ended")
}

func LoadOeeFromJSONFile(path string) {
	data := utils.LoadFromJSON(path)
	if len(data) == 0 {
		utils.LogMessage("[OEE] Failed to load oee.json – no data or corrupted file")
		ResetOeeStateAndFile(path)
		return
	}

	calcLock.Lock()
	defer calcLock.Unlock()

	oeeMap, ok := data["oee"].(map[string]interface{})
	if !ok {
		utils.LogMessage("[OEE] Invalid layout – missing `oee` section")
		ResetOeeStateAndFile(path)
		return
	}

	// --- OEE ---
	CalculatedData["czas_pomiaru"]             = utils.ToFloat(oeeMap["czas_pomiaru"])
	CalculatedData["czas_pracy"]               = utils.ToFloat(oeeMap["czas_pracy"])
	CalculatedData["czas_postoju"]             = utils.ToFloat(oeeMap["czas_postoju"])
	CalculatedData["czas_przezbrojenia"]       = utils.ToFloat(oeeMap["czas_przezbrojenia"])
	CalculatedData["czas_przezbrojenia_temp"]  = utils.ToFloat(oeeMap["czas_przezbrojenia_temp"])
	CalculatedData["ilosc_elementow"]          = utils.ToInt(oeeMap["ilosc_elementow"])
	CalculatedData["dostepnosc"]               = utils.ToFloat(oeeMap["dostepnosc"])
	CalculatedData["wydajnosc"]                = utils.ToFloat(oeeMap["wydajnosc"])
	CalculatedData["jakosc"]                   = utils.ToFloat(oeeMap["jakosc"])
	CalculatedData["oee"]                      = utils.ToFloat(oeeMap["oee"])
	CalculatedData["powietrze_L"]              = utils.ToFloat(oeeMap["powietrze_L"])
	CalculatedData["energia_W"]                = utils.ToFloat(oeeMap["energia_W"])
	CalculatedData["M3_na_szt"]                 = utils.ToFloat(oeeMap["M3_na_szt"])
	CalculatedData["W_na_szt"]                 = utils.ToFloat(oeeMap["W_na_szt"])
	CalculatedData["status_maszyny"]           = utils.ToBool(oeeMap["status_maszyny"])
	CalculatedData["status_pracy"]             = utils.ToBool(oeeMap["status_pracy"])

	// --- PRODUCT ---
	if prod, ok := data["product"].(map[string]interface{}); ok {
		CalculatedData["Dlugosc_calc"]   = utils.ToFloat(prod["dlugosc_calc"])
		CalculatedData["Szerokosc_calc"] = utils.ToFloat(prod["szerokosc_calc"])
		CalculatedData["Wysokosc_calc"]  = utils.ToFloat(prod["wysokosc_calc"])
		CalculatedData["cykl"]           = utils.ToFloat(prod["cykl"])
	}

	// --- INTERNAL ---
	if in, ok := data["internal"].(map[string]interface{}); ok {
		// czasy
		if s, _ := in["start_measurement"].(string); s != "" {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				CzasPomiarowy.StartMeasurement = t
			}
		}
		if s, _ := in["element_last_time"].(string); s != "" {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				CzasPomiarowy.ElementLastTime = t
			}
		}
		if s, _ := in["pause_start_time"].(string); s != "" {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				CzasPomiarowy.PauseStartTime = &t
			}
		} else {
			CzasPomiarowy.PauseStartTime = nil
		}
		CzasPomiarowy.TotalPause = utils.ToFloat(in["total_pause"])

		// liczniki/cykl
		impulsesCount          = utils.ToInt(in["impulses_count"])
		currentCycleElementCnt = utils.ToInt(in["current_cycle_element_cnt"])
		currentCycleValue      = utils.ToFloat(in["current_cycle_value"])
		if s, _ := in["current_cycle_start"].(string); s != "" {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				currentCycleStartTime = t
			}
		}
		currentCycleWorkSeconds = utils.ToFloat(in["current_cycle_work_seconds"])

		// historia cykli
		cycleHistory = []CyclePeriod{}
		if arr, ok := in["cycle_history"].([]interface{}); ok {
			for _, it := range arr {
				if m, ok := it.(map[string]interface{}); ok {
					st, _ := time.Parse(time.RFC3339, fmt.Sprint(m["StartTime"]))
					en, _ := time.Parse(time.RFC3339, fmt.Sprint(m["EndTime"]))
					cyc := utils.ToFloat(m["CycleLPM"])
					cnt := utils.ToInt(m["ElementCounter"])
					ws  := utils.ToFloat(m["WorkSeconds"]) // NOWE
				
					cycleHistory = append(cycleHistory, CyclePeriod{
						StartTime: st, EndTime: en, CycleLPM: cyc, ElementCounter: cnt, WorkSeconds: ws,
					})
				}
			}
		}

		// flagi/ostatnie
		prevElement          = utils.ToBool(in["prev_element"])
		prevSpeed            = utils.ToBool(in["prev_speed"])
		firstElementDetected = utils.ToBool(in["first_element_detected"])
		lastCycle            = utils.ToFloat(in["last_cycle"])
		lastWydajnosc        = utils.ToFloat(in["last_wydajnosc"])
		lastDostepnosc       = utils.ToFloat(in["last_dostepnosc"])

		CalculatedData["lastWydajnoscFinal_internal"] = utils.ToFloat(in["last_wydajnosc_final"])
		CalculatedData["lastCycleFinal_internal"]     = utils.ToFloat(in["last_cycle_final"])
		CalculatedData["elements_used"]               = utils.ToInt(in["elements_used"])
		CalculatedData["oee_temp"]                    = utils.ToFloat(in["oee_temp"])
		CalculatedData["wydajnosc_temp"]              = utils.ToFloat(in["wydajnosc_temp"])
		CalculatedData["dostepnosc_temp"]             = utils.ToFloat(in["dostepnosc_temp"])

		// baseline'y kosztów
		energyBaselineW  = utils.ToFloat(in["energy_baseline"])
		airBaselineMeters3 = utils.ToFloat(in["air_baseline"])
		CalculatedData["energyBaseline_internal"] = energyBaselineW
		CalculatedData["airBaseline_internal"]    = airBaselineMeters3
		if energyBaselineW != 0 || airBaselineMeters3 != 0 {
			costBaselineSet.Store(true)
		} else {
			costBaselineSet.Store(false)
		}
	}

	// --- HELPERS (wyłącznie do UI) ---
	if ha, ok := data["helpers_air"].(map[string]interface{}); ok {
		CalculatedData["helpers_air"] = ha
	}
	if he, ok := data["helpers_energy"].(map[string]interface{}); ok {
		CalculatedData["helpers_energy"] = he
	}

	// --- metryki "na sztukę" ---
	els := utils.ToInt(CalculatedData["ilosc_elementow"])
	if els > 0 {
		eW := utils.ToFloat(CalculatedData["energia_W"])
		airL := utils.ToFloat(CalculatedData["powietrze_L"])
		CalculatedData["W_na_szt"] = math.Round((eW/float64(els))*1000) / 1000
		CalculatedData["M3_na_szt"] = math.Round((airL/float64(els))*1000) / 1000
	}

	utils.LogMessage("[OEE] Loaded data")
}

// StartCostUpdater – liczy energię (W) i powietrze (L) narastająco oraz wskaźniki "na sztukę".
func StartCostUpdater(metersPath, flowPath string, interval time.Duration) {
	utils.Go("OEE CostUpdater", func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			func() {
				defer utils.Catch("OEE CostUpdater iteration")()
				updateCostMetrics(metersPath, flowPath)
			}()
		}
	})
}


func updateCostMetrics(metersPath, flowPath string) {
	meters := utils.LoadFromJSON(metersPath)
	flows := utils.LoadFromJSON(flowPath)

	// Energia: suma W + szczegóły per analizator
	totalW, energyParts, haveE := sumEnergyWFromMeters(meters)

	// Powietrze: suma RAW (przed skalowaniem) + szczegóły per port
	rawAirSum, airParts, haveA := sumAirTotaliserMeters3(flows)
	factor := config.AirFactor
	totalAir := rawAirSum * factor // po przeliczeniu do L

	if !(haveE || haveA) {
		return
	}

	// Pierwszy odczyt – baseline i helpers
	if !costBaselineSet.Load() {
		if haveE {
			energyBaselineW = totalW
		}
		if haveA {
			airBaselineMeters3 = totalAir
		}
		costBaselineSet.Store(true)

		calcLock.Lock()
		he := map[string]interface{}{
			"baseline":         energyBaselineW,
			"total_current_W": totalW,
		}
		for k, v := range energyParts {
			he[k] = v
		}
		CalculatedData["helpers_energy"] = he

		ha := map[string]interface{}{
			"baseline":                airBaselineMeters3,
			"total_current_M3":         totalAir,
			"total_raw_before_factor": rawAirSum,
			"factor":                  factor,
		}
		for k, v := range airParts {
			ha[k] = v
		}
		CalculatedData["helpers_air"] = ha

		CalculatedData["energyBaseline_internal"] = energyBaselineW
		CalculatedData["airBaseline_internal"] = airBaselineMeters3
		calcLock.Unlock()
		return
	}

	// Narastająco od baseline
	energyW := clamp(totalW - energyBaselineW)
	airMeters3 := clamp(totalAir - airBaselineMeters3)

	// Sztuki
	calcLock.Lock()
	elements := utils.ToInt(CalculatedData["ilosc_elementow"])
	calcLock.Unlock()

	var WPerPiece, lPerPiece float64
	if elements > 0 {
		WPerPiece = energyW / float64(elements)
		lPerPiece = airMeters3 / float64(elements)
	}

	// Zapis KPI + helpers do JSON
	calcLock.Lock()

	CalculatedData["energia_W"] = energyW
	CalculatedData["powietrze_L"] = airMeters3
	CalculatedData["W_na_szt"] = WPerPiece
	CalculatedData["M3_na_szt"] = lPerPiece
	CalculatedData["elements_used"] = elements

	he := map[string]interface{}{
		"baseline":         energyBaselineW,
		"total_current_W": totalW,
	}
	for k, v := range energyParts {
		he[k] = v
	}
	CalculatedData["helpers_energy"] = he

	ha := map[string]interface{}{
		"baseline":                airBaselineMeters3,
		"total_current_M3":         totalAir,
		"total_raw_before_factor": rawAirSum,
		"factor":                  factor,
	}
	for k, v := range airParts {
		ha[k] = v
	}
	CalculatedData["helpers_air"] = ha

	CalculatedData["energyBaseline_internal"] = energyBaselineW
	CalculatedData["airBaseline_internal"] = airBaselineMeters3
	calcLock.Unlock()
}

// --- SUMATORY ---

// Zwraca: suma_W, szczegóły, found
// Szczegóły: "energy_device_1_ea_pos_total_W", ...
func sumEnergyWFromMeters(data map[string]interface{}) (float64, map[string]float64, bool) {
	var sumW float64
	found := false
	details := map[string]float64{}

	targets := map[string]bool{"device_1": true, "device_2": true, "device_3": true}

	for devKey, v := range data {
		if !targets[devKey] {
			continue
		}
		arr, ok := v.([]interface{})
		if !ok || len(arr) == 0 {
			details["energy_"+devKey+"_ea_pos_total_W"] = 0
			continue
		}

		var got bool
		for _, it := range arr {
			m, ok := it.(map[string]interface{})
			if !ok {
				continue
			}
			id := strings.ToLower(fmt.Sprint(m["id"]))
			if id != "ea_pos_total" {
				continue
			}
			unit := strings.ToLower(fmt.Sprint(m["unit"])) // "w" / "kw" / ...
			val := utils.ToFloat(m["value"])
					
			factor := 1.0
			if unit == "kw" {
			    factor = 1000.0
			}
			W := val * factor
			sumW += W
			details["energy_"+devKey+"_ea_pos_total_W"] = W
			got = true
			break
		}
		if got {
			found = true
		} else {
			details["energy_"+devKey+"_ea_pos_total_W"] = 0
		}
	}
	return sumW, details, found
}

// sumAirTotaliserMeters3 zwraca sumę RAW (bez przelicznika) z portów powietrza
// iterując wyłącznie po config.FlowPorts – spójnie z SHIFT.
// details zawiera klucze: "air_port_<port>_raw" → wartość raw.
// Skalowanie (np. do L) rób w updateCostMetrics() przez AirFactor.
func sumAirTotaliserMeters3(data map[string]interface{}) (float64, map[string]float64, bool) {
	var sum float64
	anyFound := false
	details := map[string]float64{}

	// iteruj po dokładnie tych samych portach co w SHIFT
	for _, port := range config.FlowPorts {
		v, ok := data[port]
		if !ok {
			details["air_port_"+port+"_raw"] = 0
			continue
		}

		var (
			val   float64
			found bool
		)

		// format: obiekt { totaliser | totalizer : <number> }
		if m, ok := v.(map[string]interface{}); ok {
			if m["totaliser"] != nil {
				val = utils.ToFloat(m["totaliser"])
				found = true
			} else if m["totalizer"] != nil {
				val = utils.ToFloat(m["totalizer"])
				found = true
			}
		} else if arr, ok := v.([]interface{}); ok && len(arr) > 0 {
			// format: tablica rekordów, bierz ostatni wpis
			if last, ok := arr[len(arr)-1].(map[string]interface{}); ok {
				if last["totaliser"] != nil {
					val = utils.ToFloat(last["totaliser"])
					found = true
				} else if last["totalizer"] != nil {
					val = utils.ToFloat(last["totalizer"])
					found = true
				}
			}
		}

		if found {
			sum += val
			anyFound = true
			details["air_port_"+port+"_raw"] = val
		} else {
			details["air_port_"+port+"_raw"] = 0
		}
	}

	return sum, details, anyFound
}

// --- pomocnicze ---

func GetCalculatedData() map[string]interface{} {
	calcLock.Lock()
	defer calcLock.Unlock()

	data := copyMap(CalculatedData)

	data["TotalPause_internal"] = CzasPomiarowy.TotalPause
	if CzasPomiarowy.PauseStartTime != nil {
		data["PauseStartTime_internal"] = CzasPomiarowy.PauseStartTime.UTC().Format(time.RFC3339)
	} else {
		data["PauseStartTime_internal"] = nil
	}
	data["ElementLastTime_internal"] = CzasPomiarowy.ElementLastTime.UTC().Format(time.RFC3339)
	data["StartMeasurement_internal"] = CzasPomiarowy.StartMeasurement.UTC().Format(time.RFC3339)
	data["lastWydajnosc_internal"] = lastWydajnosc
	data["lastDostepnosc_internal"] = lastDostepnosc
	data["lastCycle_internal"] = lastCycle
	data["lastElementCount_internal"] = lastElementCount
	data["firstElementDetected_internal"] = firstElementDetected
	data["impulsesCount_internal"] = impulsesCount
	data["prevSpeed_internal"] = prevSpeed
	data["prevElement_internal"] = prevElement
	data["cycleHistory_internal"] = cycleHistory
	data["currentCycleStart_internal"] = currentCycleStartTime.UTC().Format(time.RFC3339)
	data["currentCycleElementCnt_internal"] = currentCycleElementCnt
	data["currentCycleValue_internal"] = currentCycleValue
	data["energyBaseline_internal"] = energyBaselineW
	data["airBaseline_internal"] = airBaselineMeters3
	return data
}

func ShouldStoreToDB() bool {
	calcLock.Lock()
	defer calcLock.Unlock()
	return shouldStoreToDB
}

func ResetStoreFlag() {
	calcLock.Lock()
	defer calcLock.Unlock()
	shouldStoreToDB = false
}

func IsFirstRun() bool {
	return firstRunFlag.Load()
}

func MarkFirstRunDone() {
	firstRunFlag.Store(false)
}

func copyMap(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{})
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func clamp(val float64) float64 {
	if val < 0 {
		return 0
	}
	return val
}

// Serializer: zapisuje OeeFileFlat w nowym layoutcie
// Zbuduj strukturę z bieżącego stanu i zapisz w nowym układzie
func SaveOeeFlat(path string) {
	var of OeeFileFlat

	calcLock.Lock()
	now := time.Now().UTC()

	// wyciągnij helpery jeśli już są policzone przez updateCostMetrics()
	var ha, he map[string]interface{}
	if m, ok := CalculatedData["helpers_air"].(map[string]interface{}); ok {
		ha = m
	}
	if m, ok := CalculatedData["helpers_energy"].(map[string]interface{}); ok {
		he = m
	}

	// lokalny bezpieczny getter: najpierw z mapy m[key], potem z CalculatedData[fallbackKey], na końcu 0
	fFrom := func(m map[string]interface{}, key string, fallbackKey string) float64 {
		if m != nil {
			if v, ok := m[key]; ok && v != nil {
				return utils.ToFloat(v) // tu już nie trafimy nil
			}
		}
		if v, ok := CalculatedData[fallbackKey]; ok && v != nil {
			return utils.ToFloat(v)
		}
		return 0
	}

	var pauseStrPtr *string
	if CzasPomiarowy.PauseStartTime != nil {
		s := CzasPomiarowy.PauseStartTime.UTC().Format(time.RFC3339)
		pauseStrPtr = &s
	}

	of = OeeFileFlat{
		Timestamp: now.Format(time.RFC3339Nano),
		OEE: OeeSection{
			CzasPomiaru:           utils.ToFloat(CalculatedData["czas_pomiaru"]),
			CzasPracy:             utils.ToFloat(CalculatedData["czas_pracy"]),
			CzasPostoju:           utils.ToFloat(CalculatedData["czas_postoju"]),
			CzasPrzezbrojenia:     utils.ToFloat(CalculatedData["czas_przezbrojenia"]),
			CzasPrzezbrojeniaTemp: utils.ToFloat(CalculatedData["czas_przezbrojenia_temp"]),
			IloscElementow:        utils.ToInt(CalculatedData["ilosc_elementow"]),
			Dostepnosc:            utils.ToFloat(CalculatedData["dostepnosc"]),
			Wydajnosc:             utils.ToFloat(CalculatedData["wydajnosc"]),
			Jakosc:                utils.ToFloat(CalculatedData["jakosc"]),
			OEE:                   utils.ToFloat(CalculatedData["oee"]),
			PowietrzeL:            utils.ToFloat(CalculatedData["powietrze_L"]),
			EnergyW:               utils.ToFloat(CalculatedData["energia_W"]),
			M3naSzt:               utils.ToFloat(CalculatedData["M3_na_szt"]),
			WNaSzt:                utils.ToFloat(CalculatedData["W_na_szt"]),
			StatusMaszyny:         utils.ToBool(CalculatedData["status_maszyny"]),
			StatusPracy:           utils.ToBool(CalculatedData["status_pracy"]),
			PredkoscObrotnica:     utils.ToFloat(CalculatedData["Predkosc_obrotnica"]),
		},
		Product: OeeProduct{
			DlugoscCalc:   utils.ToFloat(CalculatedData["Dlugosc_calc"]),
			SzerokoscCalc: utils.ToFloat(CalculatedData["Szerokosc_calc"]),
			WysokoscCalc:  utils.ToFloat(CalculatedData["Wysokosc_calc"]),
			Cykl:          utils.ToFloat(CalculatedData["cykl"]),
		},
		Internal: OeeInternal{
			StartMeasurement:       CzasPomiarowy.StartMeasurement.UTC().Format(time.RFC3339),
			ElementLastTime:        CzasPomiarowy.ElementLastTime.UTC().Format(time.RFC3339),
			ImpulsesCount:          impulsesCount,
			CurrentCycleElementCnt: currentCycleElementCnt,
			CurrentCycleStart:      currentCycleStartTime.UTC().Format(time.RFC3339),
			CurrentCycleValue:      currentCycleValue,
			CurrentCycleWorkSeconds: currentCycleWorkSeconds,
			CycleHistory:           cycleHistory,
			PrevElement:            prevElement,
			PrevSpeed:              prevSpeed,
			PauseStartTime:         pauseStrPtr,
			TotalPause:             CzasPomiarowy.TotalPause,
			AirBaseline:            airBaselineMeters3,
			EnergyBaseline:         energyBaselineW,
			FirstElementDetected:   firstElementDetected,
			LastCycle:              lastCycle,
			LastWydajnosc:          lastWydajnosc,
			LastWydajnoscFinal:     utils.ToFloat(CalculatedData["lastWydajnoscFinal_internal"]),
			LastDostepnosc:         lastDostepnosc,
			LastCycleFinal:         utils.ToFloat(CalculatedData["lastCycleFinal_internal"]),
			ElementsUsed:           utils.ToInt(CalculatedData["elements_used"]),
			OeeTemp:                utils.ToFloat(CalculatedData["oee_temp"]),
			WydajnoscTemp:          utils.ToFloat(CalculatedData["wydajnosc_temp"]),
			DostepnoscTemp:         utils.ToFloat(CalculatedData["dostepnosc_temp"]),
		},
		HelpersAir: HelpersAir{
			Baseline:             fFrom(ha, "baseline",                "airBaseline_internal"),
			Factor:               config.AirFactor,
			TotalRawBeforeFactor: fFrom(ha, "total_raw_before_factor", "helpers_air_total_raw_before_factor"),
			TotalCurrentM3:        fFrom(ha, "total_current_M3",        "helpers_air_total_current_M3"),
			PortsRaw:             map[string]float64{},
		},
		HelpersEnergy: HelpersEnergy{
			Baseline:      fFrom(he, "baseline",        "energyBaseline_internal"),
			TotalCurrentW: fFrom(he, "total_current_W", "helpers_energy_total_current_W"),
			DevicesW:      map[string]float64{},
		},
	}

	// skopiuj szczegóły portów/urządzeń tylko jeśli wartości są nie-nil
	if ha != nil {
		for k, v := range ha {
			if strings.HasPrefix(k, "air_port_") && strings.HasSuffix(k, "_raw") && v != nil {
				of.HelpersAir.PortsRaw[k] = utils.ToFloat(v)
			}
		}
	}
	if he != nil {
		for k, v := range he {
			if strings.HasPrefix(k, "energy_device_") && strings.HasSuffix(k, "_ea_pos_total_W") && v != nil {
				of.HelpersEnergy.DevicesW[k] = utils.ToFloat(v)
			}
		}
	}
	calcLock.Unlock()

	_ = saveOeeFlat(of, path)
}

// Serializer: zapis OeeFileFlat w nowym layoutcie do JSON
func saveOeeFlat(of OeeFileFlat, path string) error {
	// helpers_air jako płaska mapa (baseline/factor/sumy + dynamiczne porty)
	helpersAir := map[string]interface{}{
		"baseline":                of.HelpersAir.Baseline,
		"factor":                  of.HelpersAir.Factor,
		"total_raw_before_factor": of.HelpersAir.TotalRawBeforeFactor,
		"total_current_M3":         of.HelpersAir.TotalCurrentM3,
	}
	for k, v := range of.HelpersAir.PortsRaw {
		helpersAir[k] = v
	}

	// helpers_energy jako płaska mapa (baseline/suma + dynamiczne device’y)
	helpersEnergy := map[string]interface{}{
		"baseline":         of.HelpersEnergy.Baseline,
		"total_current_W": of.HelpersEnergy.TotalCurrentW,
	}
	for k, v := range of.HelpersEnergy.DevicesW {
		helpersEnergy[k] = v
	}

	// finalna struktura JSON (docelowy layout)
	out := map[string]interface{}{
		"timestamp":      of.Timestamp,
		"oee":            of.OEE,
		"product":        of.Product,
		"internal":       of.Internal,
		"helpers_air":    helpersAir,
		"helpers_energy": helpersEnergy,
	}

	// utils.SaveToJSON nie zwraca błędu – zapisujemy i zwracamy nil
	utils.SaveToJSON(out, path)
	return nil
}
