package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go_app/config"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"
	"time"
)

// --- Goroutine helpers ---

// Go uruchamia fn w gorutinie z automatycznym recoverem + stacktrace.
// Go uruchamia fn w gorutinie z automatycznym recoverem + stacktrace.
func Go(context string, fn func()) {
	if fn == nil {
		LogMessage(fmt.Sprintf("[WARN] utils.Go called with nil fn (%s) — skipping", context))
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				LogMessage(fmt.Sprintf("[ERROR] PANIC in goroutine %s: %v\n%s", context, r, string(debug.Stack())))
			}
		}()
		fn()
	}()
}

// Catch zwraca funkcję do defer: loguje panic + stacktrace (używaj wewnątrz pętli).
func Catch(context string) func() {
	return func() {
		if r := recover(); r != nil {
			LogMessage(fmt.Sprintf("[ERROR] PANIC in [%s]: %v\n%s", context, r, string(debug.Stack())))
		}
	}
}

// RecoverToString zamienia panic value na string.
func RecoverToString(r interface{}) string {
	switch v := r.(type) {
	case string:
		return v
	case error:
		return v.Error()
	default:
		return fmt.Sprintf("%v", v) // zamiast "unknown panic"
	}
}


// --- File lock management ---

var fileLocks = make(map[string]*sync.Mutex)
var locksMutex sync.Mutex

func getLockForFile(filename string) *sync.Mutex {
	locksMutex.Lock()
	defer locksMutex.Unlock()

	if fileLocks[filename] == nil {
		fileLocks[filename] = &sync.Mutex{}
	}
	return fileLocks[filename]
}

// --- Logging ---

func LogMessage(message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fullMessage := fmt.Sprintf("[%s] %s\n", timestamp, message)

	_ = os.MkdirAll(filepath.Dir(config.SystemLogPath), os.ModePerm)
	f, err := os.OpenFile(config.SystemLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("[ERROR] Cannot open log file:", err)
		return
	}
	defer f.Close()

	_, _ = f.WriteString(fullMessage)
}

// --- Save JSON to file (atomowo, z opcjonalną kopią .bak i fsync) ---

func SaveToJSON(data interface{}, filename string) {
	if data == nil {
		LogMessage(fmt.Sprintf("[WARNING] SaveToJSON: nil passed – skipping save to %s", filename))
		return
	}
	if filename == "" {
		filename = config.DefaultJsonFile
	}

	lock := getLockForFile(filename)
	lock.Lock()
	defer lock.Unlock()

	dir := filepath.Dir(filename)
	base := filepath.Base(filename)
	tmp := filepath.Join(dir, "."+base+".tmp")
	bak := filename + ".bak"

	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		LogMessage(fmt.Sprintf("[ERROR] SaveToJSON: mkdir failed for %s: %v", dir, err))
		return
	}

	// 1) Kopia poprzedniego pliku (tylko dla krytycznych JSON)
	if config.JsonWithBackup[filename] {
		if _, err := os.Stat(filename); err == nil {
			if err := copyFile(filename, bak); err != nil {
				LogMessage(fmt.Sprintf("[WARNING] SaveToJSON: failed to create backup %s: %v", bak, err))
			}
		}
	}

	// 2) Zapis do pliku tymczasowego
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		LogMessage(fmt.Sprintf("[ERROR] SaveToJSON: open temp %s failed: %v", tmp, err))
		return
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "    ")
	if err := enc.Encode(data); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		LogMessage(fmt.Sprintf("[ERROR] SaveToJSON: encode failed for %s: %v", tmp, err))
		return
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		LogMessage(fmt.Sprintf("[ERROR] SaveToJSON: fsync failed for %s: %v", tmp, err))
		return
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		LogMessage(fmt.Sprintf("[ERROR] SaveToJSON: close failed for %s: %v", tmp, err))
		return
	}
	// fsync katalogu (żeby utrwalić metadane przed/po rename)
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}

	// 3) Atomowy rename
	if err := os.Rename(tmp, filename); err != nil {
		_ = os.Remove(tmp)
		LogMessage(fmt.Sprintf("[ERROR] SaveToJSON: rename %s -> %s failed: %v", tmp, filename, err))
		return
	}
	// 4) fsync katalogu po rename
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
}

// pomocnicza kopia pliku (best-effort)
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// --- Load JSON as map (z fallbackiem na .bak i prostą naprawą ucięcia) ---

func LoadFromJSON(filename string) map[string]interface{} {
	if filename == "" {
		filename = config.DefaultJsonFile
	}

	lock := getLockForFile(filename)
	lock.Lock()
	defer lock.Unlock()

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		LogMessage(fmt.Sprintf("[WARNING] File %s does not exist, returning empty map", filename))
		return map[string]interface{}{}
	}

	// 1) normalny odczyt
	if _, m := readAndDecodeJSONMap(filename); m != nil && !hasDecodeErrorMarker(m) {
		return m
	}

	// 2) próba .bak (tylko jeśli plik jest krytyczny)
	if config.JsonWithBackup[filename] {
		bak := filename + ".bak"
		if _, err := os.Stat(bak); err == nil {
			if _, mb := readAndDecodeJSONMap(bak); mb != nil && !hasDecodeErrorMarker(mb) {
				LogMessage(fmt.Sprintf("[WARNING] Using backup file %s", bak))
				return mb
			}
		}
	}

	// 3) próba „naprawy” JSON (domknięcie } / ])
	if raw, err := os.ReadFile(filename); err == nil {
		if fixed := tryFixTruncatedJSON(raw); fixed != nil {
			var m map[string]interface{}
			if json.Unmarshal(fixed, &m) == nil {
				LogMessage("[WARNING] Recovered truncated JSON in memory")
				return m
			}
		}
	}

	return map[string]interface{}{}
}

func readAndDecodeJSONMap(path string) ([]byte, map[string]interface{}) {
	raw, err := os.ReadFile(path)
	if err != nil {
		LogMessage(fmt.Sprintf("[ERROR] Failed to read file %s: %v", path, err))
		return nil, map[string]interface{}{"__decode_error__": true}
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		LogMessage(fmt.Sprintf("[ERROR] Failed to decode JSON from %s: %v", path, err))
		return raw, map[string]interface{}{"__decode_error__": true}
	}
	return raw, out
}

func hasDecodeErrorMarker(m map[string]interface{}) bool {
	_, bad := m["__decode_error__"]
	return bad
}

func tryFixTruncatedJSON(raw []byte) []byte {
	trim := bytes.TrimSpace(raw)
	if len(trim) == 0 {
		return nil
	}
	if trim[0] == '{' && trim[len(trim)-1] != '}' {
		return append(trim, '}')
	}
	if trim[0] == '[' && trim[len(trim)-1] != ']' {
		return append(trim, ']')
	}
	return nil
}

// --- Dalsze helpery bez zmian ---

func LoadFromJSONArray(filename string) []map[string]interface{} {
	if filename == "" {
		filename = config.DefaultJsonFile
	}

	lock := getLockForFile(filename)
	lock.Lock()
	defer lock.Unlock()

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		LogMessage(fmt.Sprintf("[WARNING] File %s does not exist, returning empty array", filename))
		return []map[string]interface{}{}
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		LogMessage(fmt.Sprintf("[ERROR] Failed to read file %s: %v", filename, err))
		return []map[string]interface{}{}
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		LogMessage(fmt.Sprintf("[ERROR] Failed to decode JSON array from %s: %v", filename, err))
		return []map[string]interface{}{}
	}
	LogMessage(fmt.Sprintf("[INFO] Loaded JSON array from %s", filename))
	return result
}

func LoadFromJSONMapArray(path string) map[string][]map[string]interface{} {
	lock := getLockForFile(path)
	lock.Lock()
	defer lock.Unlock()

	raw, err := os.ReadFile(path)
	if err != nil {
		LogMessage("[ERROR] Cannot read file: " + err.Error())
		return nil
	}

	var data map[string][]map[string]interface{}
	err = json.Unmarshal(raw, &data)
	if err != nil {
		LogMessage("[ERROR] Failed to decode JSON map array: " + err.Error())
		return nil
	}
	return data
}

func LoadFromJSONMap(filename string) map[string]map[string]interface{} {
	lock := getLockForFile(filename)
	lock.Lock()
	defer lock.Unlock()

	data, err := os.ReadFile(filename)
	if err != nil {
		LogMessage("[ERROR] Cannot read file: " + err.Error())
		return nil
	}

	var result map[string]map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		LogMessage("[ERROR] Failed to decode JSON map: " + err.Error())
		return nil
	}
	return result
}

func GroupByDevice(entries []map[string]interface{}) map[string][]map[string]interface{} {
	grouped := make(map[string][]map[string]interface{})
	for _, entry := range entries {
		deviceID := fmt.Sprintf("device_%v", entry["device_id"])
		grouped[deviceID] = append(grouped[deviceID], entry)
	}
	return grouped
}

func CopyNestedMap(src map[string]map[string]interface{}) map[string]map[string]interface{} {
	if src == nil {
		return nil
	}
	dst := make(map[string]map[string]interface{})
	for key, inner := range src {
		newInner := make(map[string]interface{})
		for k, v := range inner {
			newInner[k] = v
		}
		dst[key] = newInner
	}
	return dst
}
