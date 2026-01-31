package communication

import (
	"encoding/json"
	"fmt"
	"go_app/config"
	"go_app/utils"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var mqttData = struct {
	sync.RWMutex
	Master1Port1 map[string]interface{}
	Master1Port2 map[string]interface{}
	Master1Port3 map[string]interface{}
	Master1Port4 map[string]interface{}
	Master2Port0 map[string]interface{}
	Master2Port1 map[string]interface{}
	Master2Port2 map[string]interface{}
}{
	Master1Port1: make(map[string]interface{}),
	Master1Port2: make(map[string]interface{}),
	Master1Port3: make(map[string]interface{}),
	Master1Port4: make(map[string]interface{}),
	Master2Port0: make(map[string]interface{}),
	Master2Port1: make(map[string]interface{}),
	Master2Port2: make(map[string]interface{}),
}

var mqttFieldMapping = map[string]string{
	"Switch State X01 - Pin 2": "maszyna_on/off",
	"Switch State X01 - Pin 4": "Elementy",
	"Switch State X02 - Pin 2": "Predkosc_sygnal",
	"Analog value port 0":      "Dlugosc",
	"Analog value port 1":      "Wysokosc",
	"Analog value port 2":      "Szerokosc",
}

var mqttFlowmeterMapping = map[string]string{
	"Flow":          "flow",
	"Pressure":      "pressure",
	"Temperature":   "temperature",
	"Totaliser":     "totaliser",
	"Device status": "device_status",
	"ts":            "timestamp",
	"valid":         "is_valid",
}

var mqttEventMapping = map[string]string{
	"vendorId": "vendor_id",
	"deviceId": "device_id",
	"event":    "event_type",
}

// --- state flag (jak w REST/meters) ---
var mqttState = struct {
	sync.Mutex
	offline bool
}{}

// mqttUpdateState loguje przejścia ONLINE/OFFLINE tylko raz
func mqttUpdateState(online bool, reason string) {
	mqttState.Lock()
	defer mqttState.Unlock()

	if online {
		if mqttState.offline {
			utils.LogMessage("[MQTT] is ONLINE again")
		}
		mqttState.offline = false
	} else {
		if !mqttState.offline {
			utils.LogMessage(fmt.Sprintf("[MQTT] went OFFLINE (%s)", reason))
		}
		mqttState.offline = true
	}
}

func RunMQTT() {
	utils.Go("MQTT listener", startMQTTListener)
}

func startMQTTListener() {
	backoff := 10 * time.Second
	firstLog := true

	for {
		func() {
			defer func() {
				if r := recover(); r != nil {
					utils.LogMessage(fmt.Sprintf("[ERROR] PANIC in startMQTTListener loop: %v", r))
				}
			}()

			opts := mqtt.NewClientOptions().
				AddBroker(fmt.Sprintf("tcp://%s:%s", config.MqttBroker, config.MqttPort)).
				SetClientID("go_mqtt_client_" + time.Now().Format("150405")).
				SetAutoReconnect(false)
			opts.CleanSession = true
			opts.SetKeepAlive(30 * time.Second)
			opts.SetPingTimeout(10 * time.Second)
			opts.SetWriteTimeout(10 * time.Second)
			opts.SetConnectTimeout(10 * time.Second)
			opts.SetOrderMatters(false)

			opts.OnConnect = func(client mqtt.Client) {
				utils.LogMessage("[MQTT] Connected to broker")
				mqttUpdateState(true, "")
				backoff = 10 * time.Second // reset backoff

				subs := getSubscriptions()
				if len(subs) == 0 {
					utils.LogMessage("[MQTT] No topics to subscribe")
					return
				}
				tok := client.SubscribeMultiple(subs, onMessage)
				if tok.Wait() && tok.Error() != nil {
					utils.LogMessage(fmt.Sprintf("[MQTT] SubscribeMultiple error: %v", tok.Error()))
					return
				}
				for topic := range subs {
					utils.LogMessage("[MQTT] Subscribed: " + topic)
				}
			}
			opts.OnConnectionLost = func(_ mqtt.Client, err error) {
				mqttUpdateState(false, err.Error())
			}

			if firstLog {
				utils.LogMessage("[MQTT] First connect attempt")
				firstLog = false
			}

			client := mqtt.NewClient(opts)
			if token := client.Connect(); token.Wait() && token.Error() != nil {
				mqttUpdateState(false, token.Error().Error())
				time.Sleep(backoff)
				return // zamiast continue
			}

			utils.LogMessage("[MQTT] Client running, waiting for messages")
			for client.IsConnected() {
				time.Sleep(1 * time.Second)
			}
			mqttUpdateState(false, "disconnected")
			time.Sleep(backoff)
		}()
	}
}

func onMessage(_ mqtt.Client, msg mqtt.Message) {
	defer func() {
		if r := recover(); r != nil {
			utils.LogMessage(fmt.Sprintf("[ERROR] PANIC in onMessage for topic %s: %v", msg.Topic(), r))
		}
	}()

	topic := msg.Topic()
	payload := msg.Payload()

	if len(payload) == 0 {
		utils.LogMessage(fmt.Sprintf("[MQTT] Empty payload for topic %s", topic))
		return
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(payload, &raw); err != nil {
		utils.LogMessage(fmt.Sprintf("[MQTT] JSON decode error from %s: %v", topic, err))
		return
	}
	if raw == nil {
		utils.LogMessage(fmt.Sprintf("[MQTT] Nil JSON from %s", topic))
		return
	}

	translated := make(map[string]interface{})

	if data, ok := raw["data"].(map[string]interface{}); ok {
		if v, ok := data["isValid"]; ok {
			translated["is_valid"] = v
		}
		if items, ok := data["items"].(map[string]interface{}); ok {
			for k, v := range items {
				if t, ok := mqttFieldMapping[k]; ok {
					translated[t] = v
					continue
				}
				if t, ok := mqttFlowmeterMapping[k]; ok {
					translated[t] = v
					continue
				}
			}
		}
		if ts, ok := raw["timestamp"]; ok {
			translated["timestamp"] = normalizeTimestamp(ts)
		} else {
			translated["timestamp"] = time.Now().UTC().Format(time.RFC3339Nano)
		}
	} else {
		for k, v := range raw {
			if t, ok := mqttFieldMapping[k]; ok {
				translated[t] = v
			} else if t, ok := mqttFlowmeterMapping[k]; ok {
				translated[t] = v
			} else if t, ok := mqttEventMapping[k]; ok {
				translated[t] = v
			}
		}
		translated["timestamp"] = time.Now().UTC().Format(time.RFC3339Nano)
	}

	if msg.Retained() {
		translated["_retained"] = true
	}

	assignByTopic(topic, translated)
}

func assignByTopic(topic string, translated map[string]interface{}) {
	mqttData.Lock()
	defer mqttData.Unlock()

	if len(config.MqttTopics) > 0 && topic == config.MqttTopics[0] {
		mqttData.Master1Port1 = translated
		return
	}
	if len(config.MqttTopics) > 1 && topic == config.MqttTopics[1] {
		mqttData.Master1Port2 = translated
		return
	}
	if len(config.MqttTopics) > 2 && topic == config.MqttTopics[2] {
		mqttData.Master1Port3 = translated
		return
	}
	if len(config.MqttTopics) > 3 && topic == config.MqttTopics[3] {
		mqttData.Master1Port4 = translated
		return
	}
	if len(config.MqttTopics) > 4 && topic == config.MqttTopics[4] {
		mqttData.Master2Port0 = translated
		return
	}
	if len(config.MqttTopics) > 5 && topic == config.MqttTopics[5] {
		mqttData.Master2Port1 = translated
		return
	}
	if len(config.MqttTopics) > 6 && topic == config.MqttTopics[6] {
		mqttData.Master2Port2 = translated
		return
	}

	utils.LogMessage(fmt.Sprintf("[MQTT] Message from unknown topic: %s", topic))
}

func normalizeTimestamp(v interface{}) string {
	switch ts := v.(type) {
	case string:
		if t, err := time.Parse(time.RFC3339, ts); err == nil {
			return t.UTC().Format(time.RFC3339Nano)
		}
		return time.Now().UTC().Format(time.RFC3339Nano)
	case float64:
		return time.Unix(int64(ts), 0).UTC().Format(time.RFC3339Nano)
	case int64:
		return time.Unix(ts, 0).UTC().Format(time.RFC3339Nano)
	case int:
		return time.Unix(int64(ts), 0).UTC().Format(time.RFC3339Nano)
	default:
		return time.Now().UTC().Format(time.RFC3339Nano)
	}
}

func getSubscriptions() map[string]byte {
	topics := map[string]byte{}
	for _, topic := range config.MqttTopics {
		if topic == "" {
			continue
		}
		topics[topic] = 0 // QoS 0
	}
	return topics
}

func GetMQTTData() map[string]map[string]interface{} {
	mqttData.RLock()
	defer mqttData.RUnlock()

	return map[string]map[string]interface{}{
		"master1/port1": copyMap(mqttData.Master1Port1),
		"master1/port2": copyMap(mqttData.Master1Port2),
		"master1/port3": copyMap(mqttData.Master1Port3),
		"master1/port4": copyMap(mqttData.Master1Port4),
		"master2/port0": copyMap(mqttData.Master2Port0),
		"master2/port1": copyMap(mqttData.Master2Port1),
		"master2/port2": copyMap(mqttData.Master2Port2),
	}
}

func copyMap(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
