package core

import (
	"database/sql"
	"fmt"
	"go_app/config"
	"go_app/utils"
	"strconv"
	"strings"
	"time"
	"math"
	"runtime/debug"

	_ "github.com/lib/pq"
)

// --- stan danych dla logów ---
var (
	lastMeasurementsOK bool
	lastMetersOK       bool
	lastFlowOK         bool
	lastOeeOK          bool
	lastShiftOK        bool
)

func getConnection() (*sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DbHost, config.DbPort, config.DbUser, config.DbPassword, config.DbName)
	return sql.Open("postgres", dsn)
}

func SaveMeasurementsToDB(filename string) {
	defer func() {
		if r := recover(); r != nil {
			utils.LogMessage(fmt.Sprintf("[PANIC] SaveMeasurementsToDB: %v", r))
		}
	}()

	data := utils.LoadFromJSON(filename)
	if len(data) == 0 {
		if lastMeasurementsOK {
			utils.LogMessage("[MEASUREMENTS] No data in measurements.json or error reading from file")
			lastMeasurementsOK = false
		}
		return
	}
	if !lastMeasurementsOK {
		utils.LogMessage("[MEASUREMENTS] Data restored in measurements.json")
		lastMeasurementsOK = true
	}

	db, err := getConnection()
	if err != nil {
		utils.LogMessage("[DB] Connection error: " + err.Error())
		return
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		utils.LogMessage("[DB] Transaction begin error: " + err.Error())
		return
	}
	defer tx.Commit()

	for key, items := range data {
		deviceID := extractDeviceID(key)
		if deviceID == 0 {
			continue
		}

		row := make(map[string]interface{})
		var timestamp time.Time

		arr, ok := items.([]interface{})
		if !ok {
			continue
		}

		for _, item := range arr {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			id, ok := m["id"].(string)
			if !ok {
				continue
			}
			value := utils.ToFloat(m["value"])
			row[id] = value

			if tsStr, ok := m["timestamp"].(string); ok && tsStr != "" {
				if t, err := time.Parse(time.RFC3339Nano, tsStr); err == nil {
					timestamp = t
				}
			}
		}

		if timestamp.IsZero() {
			continue
		}

		query := `
			INSERT INTO measurements (
				timestamp, device_id, f, u1, u2, u3, u12, u23, u31,
				i1, i2, i3, i_n, p1, p2, p3, q1, q2, q3,
				s1, s2, s3, pf1, pf2, pf3, p, q, s, pf
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9,
				$10, $11, $12, $13, $14, $15, $16,
				$17, $18, $19, $20, $21, $22, $23,
				$24, $25, $26, $27, $28, $29
			)
			ON CONFLICT DO NOTHING`

		_, err := tx.Exec(query,
			timestamp, deviceID,
			row["f"], row["u1"], row["u2"], row["u3"], row["u12"], row["u23"], row["u31"],
			row["i1"], row["i2"], row["i3"], row["in"], row["p1"], row["p2"], row["p3"],
			row["q1"], row["q2"], row["q3"], row["s1"], row["s2"], row["s3"],
			row["pf1"], row["pf2"], row["pf3"], row["p"], row["q"], row["s"], row["pf"],
		)
		if err != nil {
			utils.LogMessage(fmt.Sprintf("[DB] Error inserting measurements for device_%d: %v", deviceID, err))
			continue
		}
	}
}

func extractDeviceID(key string) int {
	if strings.HasPrefix(key, "device_") {
		numStr := strings.TrimPrefix(key, "device_")
		if num, err := strconv.Atoi(numStr); err == nil {
			return num
		}
	}
	return 0
}

func SaveMetersToDB(filename string) {
	defer func() {
		if r := recover(); r != nil {
			utils.LogMessage(fmt.Sprintf("[PANIC] SaveMetersToDB: %v", r))
		}
	}()

	data := utils.LoadFromJSON(filename)
	if len(data) == 0 {
		if lastMetersOK {
			utils.LogMessage("[METERS] No data in meters.json or error reading from file")
			lastMetersOK = false
		}
		return
	}
	if !lastMetersOK {
		utils.LogMessage("[METERS] Data restored in meters.json")
		lastMetersOK = true
	}

	db, err := getConnection()
	if err != nil {
		utils.LogMessage("[DB] Connection error: " + err.Error())
		return
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		utils.LogMessage("[DB] Transaction error: " + err.Error())
		return
	}
	defer tx.Commit()

	tables := map[string][]string{
		"meters_total_temp": {"ea_pos_total", "ea_neg_total", "er_pos_total", "er_neg_total", "es_total", "er_total", "ea_pos", "ea_neg", "er_pos", "er_neg", "es", "er", "e_runtime"},
		"meters_t1_temp":    {"t1_ea_pos", "t1_ea_neg", "t1_er_pos", "t1_er_neg", "t1_es", "t1_er", "t1_runtime"},
		"meters_t2_temp":    {"t2_ea_pos", "t2_ea_neg", "t2_er_pos", "t2_er_neg", "t2_es", "t2_er", "t2_runtime"},
		"meters_t3_temp":    {"t3_ea_pos", "t3_ea_neg", "t3_er_pos", "t3_er_neg", "t3_es", "t3_er", "t3_runtime"},
		"meters_t4_temp":    {"t4_ea_pos", "t4_ea_neg", "t4_er_pos", "t4_er_neg", "t4_es", "t4_er", "t4_runtime"},
	}

	for deviceKey, rawEntries := range data {
		deviceID := extractDeviceID(deviceKey)
		if deviceID == 0 {
			utils.LogMessage(fmt.Sprintf("[WARNING] Skipped invalid device_id in key: %s", deviceKey))
			continue
		}

		entries, ok := rawEntries.([]interface{})
		if !ok {
			continue
		}

		fieldMap := map[string]float64{}
		var timestamp time.Time

		for _, entry := range entries {
			e, ok := entry.(map[string]interface{})
			if !ok {
				continue
			}
			id, _ := e["id"].(string)
			value := utils.ToFloat(e["value"])
			fieldMap[id] = value

			if tsStr, ok := e["timestamp"].(string); ok && timestamp.IsZero() {
				if t, err := time.Parse(time.RFC3339Nano, tsStr); err == nil {
					timestamp = t
				}
			}
		}
		if timestamp.IsZero() {
			timestamp = time.Now().UTC()
		}

		for tableName, fields := range tables {
			placeholders := make([]string, len(fields)+2)
			args := make([]interface{}, len(fields)+2)
			columns := make([]string, len(fields)+2)

			columns[0], columns[1] = "timestamp", "device_id"
			args[0], args[1] = timestamp, deviceID
			placeholders[0], placeholders[1] = "$1", "$2"

			for i, field := range fields {
				columns[i+2] = field
				args[i+2] = fieldMap[field]
				placeholders[i+2] = fmt.Sprintf("$%d", i+3)
			}

			query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON CONFLICT DO NOTHING",
				tableName,
				strings.Join(columns, ", "),
				strings.Join(placeholders, ", "))

			_, err := tx.Exec(query, args...)
			if err != nil {
				utils.LogMessage(fmt.Sprintf("[DB] Error inserting into %s for device_%d: %v", tableName, deviceID, err))
				continue
			}
		}
	}
}

func SaveFlowDataToDB(filename string) {
	defer func() {
		if r := recover(); r != nil {
			utils.LogMessage(fmt.Sprintf("[PANIC] SaveFlowDataToDB: %v", r))
		}
	}()

	data := utils.LoadFromJSON(filename)
	if len(data) == 0 {
		if lastFlowOK {
			utils.LogMessage("[FLOW] No data in mqttFlow.json or error reading from file")
			lastFlowOK = false
		}
		return
	}
	if !lastFlowOK {
		utils.LogMessage("[FLOW] Data restored in mqttFlow.json")
		lastFlowOK = true
	}

	db, err := getConnection()
	if err != nil {
		utils.LogMessage("[DB] Connection error: " + err.Error())
		return
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		utils.LogMessage("[DB] Transaction error: " + err.Error())
		return
	}
	defer tx.Commit()

	portMapping := map[string]int{
		"master1/port3": 1,
		"master1/port4": 2,
		"master2/port0": 3,
		"master2/port1": 4,
		"master2/port2": 5,
	}

	var globalTimestamp time.Time
	if port3, ok := data["master1/port3"].(map[string]interface{}); ok {
		if tsStr, ok := port3["timestamp"].(string); ok {
			if t, err := time.Parse(time.RFC3339Nano, tsStr); err == nil {
				globalTimestamp = t
			}
		}
	}
	if globalTimestamp.IsZero() {
		globalTimestamp = time.Now().UTC()
	}

	for port, deviceID := range portMapping {
		var flow, pressure, temperature, totaliser float64

		if entryRaw, ok := data[port]; ok {
			if entry, ok := entryRaw.(map[string]interface{}); ok {
				flow = utils.ToFloat(entry["flow"])
				pressure = utils.ToFloat(entry["pressure"])
				temperature = utils.ToFloat(entry["temperature"])
				totaliser = utils.ToFloat(entry["totaliser"]) * config.AirFactor
			}
		}

		query := `
			INSERT INTO flow_data (timestamp, device_id, flow, pressure, temperature, totaliser)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT DO NOTHING`

		_, err := tx.Exec(query, globalTimestamp, deviceID, flow, pressure, temperature, totaliser)
		if err != nil {
			utils.LogMessage(fmt.Sprintf("[DB] Error inserting flow_data for device %d: %v", deviceID, err))
			continue
		}
	}
}

func SaveOeeTempToDB(filename string) {
	defer func() {
		if r := recover(); r != nil {
			utils.LogMessage(fmt.Sprintf("[PANIC] SaveOeeTempToDB: %v", r))
		}
	}()

	data := utils.LoadFromJSON(filename)
	if len(data) == 0 {
		if lastOeeOK {
			utils.LogMessage("[OEE_TEMP] No data in oee.json or error reading from file")
			lastOeeOK = false
		}
		return
	}
	if !lastOeeOK {
		utils.LogMessage("[OEE_TEMP] Data restored in oee.json")
		lastOeeOK = true
	}

	db, err := getConnection()
	if err != nil {
		utils.LogMessage("[DB] Connection error: " + err.Error())
		return
	}
	defer db.Close()

	// nested float/bool/int
	nf := func(keys ...string) float64 {
		var cur any = data
		for _, k := range keys {
			m, ok := cur.(map[string]any)
			if !ok {
				return 0
			}
			cur = m[k]
		}
		return utils.ToFloat(cur)
	}
	nb := func(keys ...string) bool {
		var cur any = data
		for _, k := range keys {
			m, ok := cur.(map[string]any)
			if !ok {
				return false
			}
			cur = m[k]
		}
		return utils.ToBool(cur)
	}
	ni := func(keys ...string) int {
		var cur any = data
		for _, k := range keys {
			m, ok := cur.(map[string]any)
			if !ok {
				return 0
			}
			cur = m[k]
		}
		return utils.ToInt(cur)
	}

	query := `
		INSERT INTO oee_temp (
			timestamp, predkosc_obrotnica, czas_pracy, czas_postoju,
			czas_pomiaru, czas_przezbrojenia,
			status_maszyny, ilosc_elementow,
			dlugosc_calc, szerokosc_calc, wysokosc_calc,
			dostepnosc, wydajnosc, jakosc, cykl, oee,
			czas_przezbrojenia_temp, status_pracy, W_na_szt, M3_na_szt
		) VALUES (
			now(), $1, $2, $3, $4,
			$5, $6,
			$7, $8,
			$9, $10, $11,
			$12, $13, $14, $15, $16,
			$17, $18, $19
		)
		ON CONFLICT DO NOTHING`

	args := []interface{}{
		nf("oee", "predkosc_obrotnica"),
		nf("oee", "czas_pracy"),
		nf("oee", "czas_postoju"),
		nf("oee", "czas_pomiaru"),
		nf("oee", "czas_przezbrojenia"),
		nb("oee", "status_maszyny"),
		ni("oee", "ilosc_elementow"),
		nf("product", "dlugosc_calc"),
		nf("product", "szerokosc_calc"),
		nf("product", "wysokosc_calc"),
		nf("oee", "dostepnosc"),
		nf("oee", "wydajnosc"),
		nf("oee", "jakosc"),
		nf("product", "cykl"),
		nf("oee", "oee"),
		nf("oee", "czas_przezbrojenia_temp"),
		nb("oee", "status_pracy"),
		nf("oee", "W_na_szt"),
		nf("oee", "M3_na_szt"),
	}

	if _, err := db.Exec(query, args...); err != nil {
		utils.LogMessage(fmt.Sprintf("[DB] Error inserting into oee_temp: %v", err))
	}
}

func SaveShiftSummaryToDB(filename string) {
	defer func() {
		if r := recover(); r != nil {
			utils.LogMessage(fmt.Sprintf("[PANIC] SaveShiftSummaryToDB: %v", r))
		}
	}()

	data := utils.LoadFromJSON(filename)
	if len(data) == 0 {
		if lastShiftOK {
			utils.LogMessage("[SHIFT_SUMMARY] No data in shift_summary.json or error reading from file")
			lastShiftOK = false
		}
		return
	}
	if !lastShiftOK {
		utils.LogMessage("[SHIFT_SUMMARY] Data restored in shift_summary.json")
		lastShiftOK = true
	}

	db, err := getConnection()
	if err != nil {
		utils.LogMessage("[DB] Connection error: " + err.Error())
		return
	}
	defer db.Close()

	// --- helpers ---
	t := func(key string) time.Time {
		str, _ := data[key].(string)
		ts, _ := time.Parse(time.RFC3339Nano, str)
		return ts
	}
	nf := func(keys ...string) float64 {
		var cur any = data
		for _, k := range keys {
			m, ok := cur.(map[string]any)
			if !ok {
				return 0
			}
			cur = m[k]
		}
		return utils.ToFloat(cur)
	}
	an := func(i int, id string) float64 { return nf("analizator", "device_"+strconv.Itoa(i), id) }
	tp := func(i int) float64            { return nf("totaliser", "per_port", strconv.Itoa(i)) }
	ec := func(label string) int {
		if m, ok := data["elements_per_cycle"].(map[string]any); ok {
			return utils.ToInt(m[label])
		}
		return 0
	}

	wNaSzt := nf("energy", "W_na_szt")
	M3naSzt := nf("totaliser", "M3_na_szt")

	query := `
		INSERT INTO shift_summary (
			data_utworzenia, start_zmiany, koniec_zmiany,
			czas_pracy, czas_postoju, czas_przezbrojenia, czas_pomiaru,
			ilosc_elementow, dostepnosc, wydajnosc, jakosc, oee,

			analizator_1_ea_pos, analizator_1_ea_neg, analizator_1_er_pos, analizator_1_er_neg, analizator_1_es, analizator_1_er,
			 analizator_2_ea_pos, analizator_2_ea_neg, analizator_2_er_pos, analizator_2_er_neg, analizator_2_es, analizator_2_er,
			 analizator_3_ea_pos, analizator_3_ea_neg, analizator_3_er_pos, analizator_3_er_neg, analizator_3_es, analizator_3_er,

			totaliser_1, totaliser_2, totaliser_3, totaliser_4, totaliser_5,
			W_na_szt, M3_na_szt,

			cykl0, cykl1, cykl2, cykl3
		) VALUES (
			now(), $1, $2,
			$3, $4, $5, $6,
			$7, $8, $9, $10, $11,

			$12, $13, $14, $15, $16, $17,
			$18, $19, $20, $21, $22, $23,
			$24, $25, $26, $27, $28, $29,

			$30, $31, $32, $33, $34,
			$35, $36,

			$37, $38, $39, $40
		)
		ON CONFLICT DO NOTHING
	`

	args := []interface{}{
		t("start_zmiany"), t("koniec_zmiany"),

		nf("oee", "czas_pracy"),
		nf("oee", "czas_postoju"),
		nf("oee", "czas_przezbrojenia"),
		nf("oee", "czas_pomiaru"),

		int(utils.ToFloat(nf("oee", "ilosc_elementow"))),
		nf("oee", "dostepnosc"),
		nf("oee", "wydajnosc"),
		nf("oee", "jakosc"),
		nf("oee", "oee"),

		an(1, "ea_pos"), an(1, "ea_neg"), an(1, "er_pos"), an(1, "er_neg"), an(1, "es"), an(1, "er"),
		an(2, "ea_pos"), an(2, "ea_neg"), an(2, "er_pos"), an(2, "er_neg"), an(2, "es"), an(2, "er"),
		an(3, "ea_pos"), an(3, "ea_neg"), an(3, "er_pos"), an(3, "er_neg"), an(3, "es"), an(3, "er"),

		tp(1), tp(2), tp(3), tp(4), tp(5),

		wNaSzt, M3naSzt,

		ec("cykl0"), ec("cykl1"), ec("cykl2"), ec("cykl3"),
	}

	if _, err := db.Exec(query, args...); err != nil {
		utils.LogMessage(fmt.Sprintf("[DB] Error inserting into shift_summary: %v", err))
	}
}

func AdjustIdleToChangeover(start, end float64, _ float64) {
	defer func() {
		if r := recover(); r != nil {
			// pełny stacktrace do logów
			utils.LogMessage(fmt.Sprintf("[PANIC] AdjustIdleToChangeover: %v\n%s", r, string(debug.Stack())))
		}
	}()

	// sanity checks
	if math.IsNaN(start) || math.IsNaN(end) {
		utils.LogMessage("[WARN] AdjustIdleToChangeover: NaN in start/end")
		return
	}
	if end < start {
		start, end = end, start
	}

	db, err := getConnection()
	if err != nil {
		utils.LogMessage("[DB] Connection error in AdjustIdleToChangeover: " + err.Error())
		return
	}
	defer db.Close()

	// utils.LogMessage(fmt.Sprintf("[INFO] AdjustIdleToChangeover: range start=%.3f end=%.3f (sec epoch)", start, end))

	// 1) Pobierz pierwsze czas_postoju z zakresu
	var firstPostoj sql.NullFloat64
	const queryFirst = `
		SELECT czas_postoju
		FROM oee_temp
		WHERE EXTRACT(EPOCH FROM timestamp) >= $1
		  AND EXTRACT(EPOCH FROM timestamp) <= $2
		ORDER BY timestamp ASC
		LIMIT 1
	`
	err = db.QueryRow(queryFirst, start, end).Scan(&firstPostoj)
	if err != nil && err != sql.ErrNoRows {
		utils.LogMessage("[DB] QueryRow error in AdjustIdleToChangeover: " + err.Error())
		return
	}

	// 2) Aktualizacja zakresu
	if err == sql.ErrNoRows || !firstPostoj.Valid {
		// brak rekordu → nie nadpisujemy czas_postoju
		const q = `
			UPDATE oee_temp
			SET czas_przezbrojenia = czas_przezbrojenia_temp
			WHERE EXTRACT(EPOCH FROM timestamp) >= $1
			  AND EXTRACT(EPOCH FROM timestamp) <= $2
		`
		res, err := db.Exec(q, start, end)
		if err != nil {
			utils.LogMessage("[DB] Update (no-row) error in AdjustIdleToChangeover: " + err.Error())
			return
		}
		if n, _ := res.RowsAffected(); n == 0 {
			utils.LogMessage("[INFO] AdjustIdleToChangeover: no rows updated (no-row branch)")
		}
		return
	}

	const queryUpdate = `
		UPDATE oee_temp
		SET czas_przezbrojenia = czas_przezbrojenia_temp,
		    czas_postoju = $3
		WHERE EXTRACT(EPOCH FROM timestamp) >= $1
		  AND EXTRACT(EPOCH FROM timestamp) <= $2
	`
	res, err := db.Exec(queryUpdate, start, end, firstPostoj.Float64)
	if err != nil {
		utils.LogMessage("[DB] Update error in AdjustIdleToChangeover: " + err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		utils.LogMessage("[INFO] AdjustIdleToChangeover: no rows updated (normal branch)")
	}
}
