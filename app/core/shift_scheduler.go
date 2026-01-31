package core

import (
	"fmt"
	"go_app/config"
	"go_app/utils"
	"strconv"
	"sync"
	"time"
	"strings"
)

// Lokalne godziny granic zmian (Polska)
var shiftTimes = []string{"06:00", "14:00", "22:00"}

// var shiftTimes []string
// func init() {
//     for h := 0; h < 24; h++ {
//         for m := 0; m < 60; m += 2 {
//             shiftTimes = append(shiftTimes, fmt.Sprintf("%02d:%02d", h, m))
//         }
//     }
// }

// Pamięć dla totaliserów (baseline na początek zmiany + ostatnia wartość)

func getPreviousShiftRange(shiftStartUTC time.Time) (time.Time, time.Time) {
	end := shiftStartUTC
	start := end.Add(-8 * time.Hour)
	return start, end
}

// func getPreviousShiftRange(shiftStartUTC time.Time) (time.Time, time.Time) {
// 	end := shiftStartUTC
// 	start := end.Add(-2 * time.Minute)
// 	return start, end
// }

var (
	totaliserStart = make(map[int]float64)
	totaliserLast  = make(map[int]float64)
	totaliserLock  sync.Mutex
	energyStart = make(map[int]float64)
	energyLast  = make(map[int]float64)
	energyLock  sync.Mutex
)

type Summary struct {
	DataUtworzenia   string                        `json:"data_utworzenia"`
	StartZmiany      string                        `json:"start_zmiany"`
	KoniecZmiany     string                        `json:"koniec_zmiany"`
	OEE              OeeSectionSummary             `json:"oee"`
	ElementsPerCycle map[string]int                `json:"elements_per_cycle"`
	Energy           EnergySection                 `json:"energy"`
	Totaliser        TotaliserSection              `json:"totaliser"`
	Analizator       map[string]map[string]float64 `json:"analizator"`
}

type OeeSectionSummary struct {
	CzasPracy         float64 `json:"czas_pracy"`
	CzasPostoju       float64 `json:"czas_postoju"`
	CzasPrzezbrojenia float64 `json:"czas_przezbrojenia"`
	CzasPomiaru       float64 `json:"czas_pomiaru"`
	IloscElementow    int     `json:"ilosc_elementow"`
	Dostepnosc        float64 `json:"dostepnosc"`
	Wydajnosc         float64 `json:"wydajnosc"`
	Jakosc            float64 `json:"jakosc"`
	OEE               float64 `json:"oee"`
	W_NaSzt            float64 `json:"W_na_szt"`
    M3_NaSzt           float64 `json:"M3_na_szt"`
}

type TotaliserSection struct {
	PerPort map[string]float64 `json:"per_port"`
	Start   map[string]float64 `json:"start"`
	Last    map[string]float64 `json:"last"`
	SumM3    float64            `json:"sum_M3"`
	LNaSzt  float64            `json:"M3_na_szt"`
}

type EnergySection struct {
	PerDeviceWh map[string]float64 `json:"per_device_Wh"`
	Start       map[string]float64 `json:"start"`
	Last        map[string]float64 `json:"last"`
	SumWh       float64            `json:"sum_W"`
	WhNaSzt     float64            `json:"W_na_szt"`
}

// StartShiftScheduler – pętla granic zmian
func StartShiftScheduler() {
	utils.Go("SHIFT Scheduler", func() {
		// Używaj stałej strefy PL niezależnie od ustawień kontenera
		loc, err := time.LoadLocation("Europe/Warsaw")
		if err != nil {
			loc = time.Local
			utils.LogMessage("[SHIFT] using time.Local (failed to load Europe/Warsaw): " + err.Error())
		}

		for {
			func() {
				defer utils.Catch("SHIFT iteration")()

				nowLocal := time.Now().In(loc)
				nextUTC := findNextShiftTimeUTC(nowLocal, loc)

				// Poprzednia zmiana: [prevStart, prevEnd=nextUTC)
				prevStartUTC, prevEndUTC := getPreviousShiftRange(nextUTC)

				utils.LogMessage("[SHIFT] Prev shift (UTC): start=" +
					prevStartUTC.Format(time.RFC3339Nano) + ", end=" + prevEndUTC.Format(time.RFC3339Nano))

				// Czekaj do granicy — zabezpieczenie na ujemne/dziwne czasy (np. zmiana czasu)
				until := time.Until(nextUTC)
				if until < 0 {
					until = 0
				} else if until > 26*time.Hour { // ochronnie przy zaburzeniach zegara/DST
					until = 26 * time.Hour
				}
				time.Sleep(until)

				// Domknięcie poprzedniej zmiany
				if err := executeShiftSummary(prevStartUTC, prevEndUTC, true); err != nil {
					utils.LogMessage("[SHIFT] Summary write FAILED, OEE NOT reset: " + err.Error())
				} else {
					ResetOeeStateAndFile(config.OeeFilePath)
				}

				// Baseline’y na nową zmianę
				setTotaliserBaselines()
				setEnergyBaselines()

				now := time.Now().UTC()
				utils.LogMessage(fmt.Sprintf("[SHIFT] Boundary passed – local: %s, UTC: %s",
					now.In(loc).Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)))
			}()
		}
	})
}

func findNextShiftTimeUTC(nowLocal time.Time, loc *time.Location) time.Time {
	const layout = "2006-01-02 15:04"
	today := nowLocal.Format("2006-01-02")

	for _, t := range shiftTimes {
		candidateLocal, _ := time.ParseInLocation(layout, today+" "+t, loc)
		if candidateLocal.After(nowLocal) {
			return candidateLocal.UTC()
		}
	}

	// jutro, pierwsza zmiana
	tomorrow := nowLocal.Add(24 * time.Hour).Format("2006-01-02")
	nextLocal, _ := time.ParseInLocation(layout, tomorrow+" "+shiftTimes[0], loc)
	return nextLocal.UTC()
}

// executeShiftSummary: zapisuje podsumowanie poprzedniej zmiany (JSON + DB).
func executeShiftSummary(startUTC, endUTC time.Time, isShiftEnd bool) error {
	s := Summary{
		DataUtworzenia:   endUTC.Format(time.RFC3339Nano),
		StartZmiany:      startUTC.Format(time.RFC3339Nano),
		KoniecZmiany:     endUTC.Format(time.RFC3339Nano),
		OEE:              OeeSectionSummary{},
		Energy:           EnergySection{PerDeviceWh: map[string]float64{}, Start: map[string]float64{}, Last: map[string]float64{}},
		Totaliser:        TotaliserSection{PerPort: map[string]float64{}, Start: map[string]float64{}, Last: map[string]float64{}},
		Analizator:       map[string]map[string]float64{},
		ElementsPerCycle: map[string]int{},
	}

	// --- odczyt źródeł ---
	oee    := utils.LoadFromJSON(config.OeeFilePath)
	meters := utils.LoadFromJSONMapArray(config.MetersFilePath)
	flow   := utils.LoadFromJSONMap(config.MqttFlowFilePath)

	fillOeeSectionSummary(&s.OEE, oee)

	// policz elementy per cykl (history + bieżący okres)
	s.ElementsPerCycle = extractElementsPerCycleFixed(oee)

	// kontrola zgodności sumy z oee.ilosc_elementow
	total := 0
	for _, v := range s.ElementsPerCycle { total += v }
	if s.OEE.IloscElementow > 0 && total != s.OEE.IloscElementow {
		utils.LogMessage(fmt.Sprintf("[SHIFT_SUMMARY] elements_per_cycle total=%d != ilosc_elementow=%d", total, s.OEE.IloscElementow))
	}

	// reszta bez zmian
	fillMeterAnalizator(&s.Analizator, meters)
	fillFlowTotaliser(&s.Totaliser, flow, isShiftEnd, &s.OEE)
	fillEnergy(&s.Energy, meters, isShiftEnd, &s.OEE)

	// --- zapis ---
	utils.SaveToJSON(s, config.SummaryFilePath)
	SaveShiftSummaryToDB(config.SummaryFilePath)
	return nil
}

func mapCycleLPMToLabelFixed(lpm float64) string {
	for i, rule := range config.CycleTable {
		if rule.CycleLPM == lpm {
			return fmt.Sprintf("cykl%d", i)
		}
	}
	return "unknown"
}

func extractElementsPerCycleFixed(oee map[string]interface{}) map[string]int {
	out := map[string]int{
		"cykl0": 0,
		"cykl1": 0,
		"cykl2": 0,
		"cykl3": 0,
	}

	internal, _ := oee["internal"].(map[string]interface{})
	if internal == nil {
		return out
	}

	// 1) Historia cykli
	if rawHist, ok := internal["cycle_history"].([]interface{}); ok {
		for _, it := range rawHist {
			if row, ok := it.(map[string]interface{}); ok {
				lpm := utils.ToFloat(row["CycleLPM"])
				cnt := utils.ToInt(row["ElementCounter"])
				if cnt > 0 {
					out[mapCycleLPMToLabelFixed(lpm)] += cnt
				}
			}
		}
	}

	// 2) Bieżący cykl
	if cval, ok := internal["current_cycle_value"]; ok {
		if ccnt, ok2 := internal["current_cycle_element_cnt"]; ok2 {
			cnt := utils.ToInt(ccnt)
			if cnt > 0 {
				out[mapCycleLPMToLabelFixed(utils.ToFloat(cval))] += cnt
			}
		}
	}

	// 3) Log różnicy względem globalnej ilości elementów
	total := 0
	for _, v := range out { total += v }
	if oeeMap, ok := oee["oee"].(map[string]interface{}); ok {
		target := utils.ToInt(oeeMap["ilosc_elementow"])
		if target > 0 && total != target {
			utils.LogMessage(fmt.Sprintf(
				"[SHIFT_SUMMARY] elements_per_cycle total=%d != oee.ilosc_elementow=%d (różnica %d)",
				total, target, target-total))
		}
	}

	return out
}

func fillOeeSectionSummary(dst *OeeSectionSummary, oee map[string]interface{}) {
	section := oee
	if inner, ok := oee["oee"].(map[string]interface{}); ok {
		section = inner
	}

	dst.CzasPracy         = utils.ToFloat(section["czas_pracy"])
	dst.CzasPostoju       = utils.ToFloat(section["czas_postoju"])
	dst.CzasPrzezbrojenia = utils.ToFloat(section["czas_przezbrojenia"])
	dst.CzasPomiaru       = utils.ToFloat(section["czas_pomiaru"])
	dst.IloscElementow    = utils.ToInt(section["ilosc_elementow"])
	dst.Dostepnosc        = utils.ToFloat(section["dostepnosc"])
	dst.Wydajnosc         = utils.ToFloat(section["wydajnosc"])
	dst.Jakosc            = utils.ToFloat(section["jakosc"])
	dst.OEE               = utils.ToFloat(section["oee"])
	dst.W_NaSzt            = utils.ToFloat(section["W_na_szt"])
	dst.M3_NaSzt           = utils.ToFloat(section["M3_na_szt"])
}

func fillMeterAnalizator(dst *map[string]map[string]float64, meters map[string][]map[string]interface{}) {
	root := *dst
	n := len(config.AnalyzerIPs)
	if n <= 0 { n = 1 }

	for i := 1; i <= n; i++ {
		deviceKey := "device_" + strconv.Itoa(i)
		out := map[string]float64{}
		if entries, ok := meters[deviceKey]; ok {
			for _, entry := range entries {
				id, _ := entry["id"].(string)
				val   := utils.ToFloat(entry["value"])
				out[id] = val
			}
		}
		root[deviceKey] = out
	}
}

func fillFlowTotaliser(dst *TotaliserSection, flow map[string]map[string]interface{}, isShiftEnd bool, oee *OeeSectionSummary) {
	totaliserLock.Lock()
	defer totaliserLock.Unlock()

	perPort := map[string]float64{}
	startMap := map[string]float64{}
	lastMap  := map[string]float64{}
	totalSum := 0.0

	for i, port := range config.FlowPorts {
		idx := i + 1
		k := strconv.Itoa(idx)

		currentVal := 0.0
		if f, ok := flow[port]; ok {
			currentVal = utils.ToFloat(f["totaliser"]) * config.AirFactor
		}

		if _, ok := totaliserStart[idx]; !ok {
			// Fallback z poprzedniego summary
			var startFromSummary float64
			if lastSummary := utils.LoadFromJSON(config.SummaryFilePath); lastSummary != nil {
				if t, ok := lastSummary["totaliser"].(map[string]interface{}); ok {
					if sm, ok := t["start"].(map[string]interface{}); ok {
						if v, ok := sm[k]; ok {
							startFromSummary = utils.ToFloat(v)
						}
					}
				}
			}
			if startFromSummary != 0 {
				totaliserStart[idx] = startFromSummary
			} else {
				totaliserStart[idx] = currentVal
			}
		}

		startVal := totaliserStart[idx]
		diff := currentVal - startVal
		if diff < 0 { diff = 0 }

		perPort[k]         = diff
		startMap[k]        = startVal
		lastMap[k]         = currentVal
		totaliserLast[idx] = currentVal
		totalSum          += diff

		if isShiftEnd {
			totaliserStart[idx] = currentVal
		}
	}

	dst.PerPort = perPort
	dst.Start   = startMap
	dst.Last    = lastMap
	dst.SumM3    = totalSum
	if oee.IloscElementow > 0 {
		dst.LNaSzt = totalSum / float64(oee.IloscElementow)
	} else {
		dst.LNaSzt = 0.0
	}
}

// + import "strings"

func fillEnergy(dst *EnergySection, meters map[string][]map[string]interface{}, isShiftEnd bool, oee *OeeSectionSummary) {
	energyLock.Lock()
	defer energyLock.Unlock()

	perDevice := map[string]float64{}
	startMap  := map[string]float64{}
	lastMap   := map[string]float64{}

	lastSummary := utils.LoadFromJSON(config.SummaryFilePath)
	totalWh := 0.0

	for i := 1; i <= 3; i++ {
		k := strconv.Itoa(i)
		deviceKey := "device_" + k

		currentValWh := 0.0
		if entries, ok := meters[deviceKey]; ok {
			var latestTime time.Time
			for _, ent := range entries {
				id, _ := ent["id"].(string)
				if strings.ToLower(id) != "ea_pos_total" {
					continue
				}

				// --- Normalizacja jednostki do Wh ---
				unit := strings.ToLower(fmt.Sprint(ent["unit"])) // "wh" / "kwh" / (inne?)
				val  := utils.ToFloat(ent["value"])
				switch unit {
				case "kwh":
					val = val * 1000.0
				case "wh", "":
					// val bez zmian (Wh)
				default:
					// Jeśli przyjdzie moc ("w"/"kw") albo coś innego, pomiń tę próbkę,
					// bo sekcja Summary liczy ENERGIĘ (Wh), a nie moc.
					continue
				}

				// --- Najświeższa próbka po timestamp ---
				if tsRaw, ok := ent["timestamp"].(string); ok {
					if ts, err := time.Parse(time.RFC3339, tsRaw); err == nil && ts.After(latestTime) {
						latestTime = ts
						currentValWh = val
						continue
					}
				}
				// fallback gdy brak/parsing timestampu
				currentValWh = val
			}
		}

		// Ustal baseline (start) – niezależny od OEE
		if _, ok := energyStart[i]; !ok {
			if v, ok := lastSummary["energy"].(map[string]interface{}); ok {
				if vs, ok := v["start"].(map[string]interface{}); ok {
					if sv, ok := vs[k]; ok {
						energyStart[i] = utils.ToFloat(sv)
					}
				}
			}
			if _, ok := energyStart[i]; !ok {
				energyStart[i] = currentValWh
			}
		}

		startVal := energyStart[i]
		diffWh := currentValWh - startVal
		if diffWh < 0 {
			diffWh = 0
		}

		perDevice[k]  = diffWh
		startMap[k]   = startVal
		lastMap[k]    = currentValWh
		energyLast[i] = currentValWh
		totalWh      += diffWh

		if isShiftEnd {
			energyStart[i] = currentValWh
		}
	}

	dst.PerDeviceWh = perDevice
	dst.Start       = startMap
	dst.Last        = lastMap
	dst.SumWh       = totalWh
	if oee.IloscElementow > 0 {
		dst.WhNaSzt = totalWh / float64(oee.IloscElementow)
	} else {
		dst.WhNaSzt = 0.0
	}
}


func setTotaliserBaselines() {
	flow := utils.LoadFromJSONMap(config.MqttFlowFilePath)

	totaliserLock.Lock()
	defer totaliserLock.Unlock()

	for i, port := range config.FlowPorts {
		idx := i + 1
		currentVal := 0.0
		if f, ok := flow[port]; ok {
			currentVal = utils.ToFloat(f["totaliser"]) * config.AirFactor
		}
		totaliserStart[idx] = currentVal
		totaliserLast[idx]  = currentVal
	}
}

func setEnergyBaselines() {
	meters := utils.LoadFromJSONMapArray(config.MetersFilePath)

	energyLock.Lock()
	defer energyLock.Unlock()

	// Opcjonalny fallback z poprzedniego summary (nowa struktura z sekcją "energy")
	lastSummary := utils.LoadFromJSON(config.SummaryFilePath)
	var energyLastFromSummary map[string]interface{}
	if e, ok := lastSummary["energy"].(map[string]interface{}); ok {
		if lm, ok := e["last"].(map[string]interface{}); ok {
			energyLastFromSummary = lm
		}
	}

	for i := 1; i <= 3; i++ {
		key := "device_" + strconv.Itoa(i)
		currentVal := 0.0

		if entries, ok := meters[key]; ok {
			// Jeśli wpisy mają timestamp (RFC3339), wybierz najnowszy; w przeciwnym razie ostatni z listy
			var latestTime time.Time
			found := false
			for _, e := range entries {
				if id, _ := e["id"].(string); id == "ea_pos_total" {
					// próbuj po timestamp
					if tsRaw, ok := e["timestamp"].(string); ok {
						if ts, err := time.Parse(time.RFC3339, tsRaw); err == nil {
							if !found || ts.After(latestTime) {
								latestTime = ts
								currentVal = utils.ToFloat(e["value"])
								found = true
								continue
							}
						}
					}
					// fallback: brak/parsowanie timestampu
					currentVal = utils.ToFloat(e["value"])
					found = true
				}
			}
		}

		// Dodatkowy fallback: brak bieżącej próbki -> użyj ostatniej znanej z summary["energy"]["last"][i]
		if currentVal == 0.0 && energyLastFromSummary != nil {
			if v, ok := energyLastFromSummary[strconv.Itoa(i)]; ok {
				currentVal = utils.ToFloat(v)
			}
		}

		energyStart[i] = currentVal
		energyLast[i] = currentVal
	}
}