package utils

import (
	"encoding/json"
	"log"
	"strconv"
	"math"
	"fmt"
    "runtime"
    "sync"
    "time"
	"reflect"
)

var warnGate sync.Map

func warnToFloat(msg string) {
	// throttling per-caller: 1 wpis / 30s
	_, file, line, _ := runtime.Caller(2)
	key := fmt.Sprintf("%s:%d", file, line)
	now := time.Now()
	if lastRaw, ok := warnGate.Load(key); ok {
		if now.Sub(lastRaw.(time.Time)) < 30*time.Second {
			return
		}
	}
	warnGate.Store(key, now)
	LogMessage(fmt.Sprintf("[WARNING] ToFloat @ %s:%d: %s", file, line, msg))
}

// --- Konwersja z rozróżnieniem ok/fail ---
func ToFloatOK(v interface{}) (float64, bool) {
	switch t := v.(type) {
	case nil:
		return 0, false
	case float64:
		if math.IsNaN(t) || math.IsInf(t, 0) {
			return 0, false
		}
		return t, true
	case float32:
		f := float64(t)
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return 0, false
		}
		return f, true
	case int, int8, int16, int32, int64:
		return float64(reflect.ValueOf(t).Int()), true
	case uint, uint8, uint16, uint32, uint64:
		return float64(reflect.ValueOf(t).Uint()), true
	case string:
		if t == "" {
			return 0, false
		}
		if f, err := strconv.ParseFloat(t, 64); err == nil {
			if math.IsNaN(f) || math.IsInf(f, 0) {
				return 0, false
			}
			return f, true
		}
		return 0, false
	case json.Number:
		if f, err := t.Float64(); err == nil {
			return f, true
		}
		return 0, false
	case bool:
		if t {
			return 1, true
		}
		return 0, true
	default:
		return 0, false
	}
}

// --- Proste API ---
func ToFloat(v interface{}) float64 {
	if f, ok := ToFloatOK(v); ok {
		return f
	}
	// nil traktujemy cicho jako 0
	if v != nil {
		warnToFloat(fmt.Sprintf("unsupported value (type=%T, value=%v)", v, v))
	}
	return 0
}

func ToInt(v interface{}) int {
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	case float32:
		return int(val)
	case json.Number:
		i, _ := val.Int64()
		return int(i)
	case string:
		i, err := strconv.Atoi(val)
		if err == nil {
			return i
		}
	case bool:
		if val {
			return 1
		}
		return 0
	}
	log.Printf("Unsupported type for ToInt: %T", v)
	return 0
}

func ToBool(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return val
	case float64:
		return val != 0
	case int:
		return val != 0
	case string:
		b, err := strconv.ParseBool(val)
		if err == nil {
			return b
		}
	}
	return false
}

func ToString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case json.Number:
		return val.String()
	case int, int64, float32, float64:
		return strconv.FormatFloat(ToFloat(val), 'f', -1, 64)
	case bool:
		if val {
			return "true"
		}
		return "false"
	}
	return ""
}
