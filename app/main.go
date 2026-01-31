package main

import (
	"go_app/communication"
	"go_app/config"
	"go_app/core"
	"go_app/utils"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			utils.LogMessage("[FATAL] PANIC in main thread: " + utils.RecoverToString(r) + "\n" + string(debug.Stack()))
		}
	}()
	utils.LogMessage("[SYSTEM] Program started")
	core.LoadOeeFromJSONFile(config.OeeFilePath)
	communication.RunMQTT()
	communication.RunRestCommunication()
	core.StartShiftScheduler()

	// --- REST + METERS Fetcher ---
	utils.Go("REST+METERS Fetcher", func() {
		for {
			func() {
				defer utils.Catch("REST+METERS Fetcher")()
				restData := communication.GetRestData()
				metersData := communication.GetMetersData()
				utils.SaveToJSON(restData, config.MeasurementFilePath)
				utils.SaveToJSON(metersData, config.MetersFilePath)
			}()
			time.Sleep(500 * time.Millisecond)
		}
	})

	// --- MQTT + OEE ---
	utils.Go("MQTT + OEE", func() {
		for {
			func() {
				defer utils.Catch("MQTT + OEE")()

				if core.IsResetScheduled() {
					core.ResetOeeStateAndFile(config.OeeFilePath)
					core.ClearResetFlag()
				}

				mqttData := communication.GetMQTTData()

				mqttOEE := map[string]map[string]interface{}{
					"master1/port1": mqttData["master1/port1"],
					"master1/port2": mqttData["master1/port2"],
				}
				mqttFlow := map[string]map[string]interface{}{
					"master1/port3": mqttData["master1/port3"],
					"master1/port4": mqttData["master1/port4"],
					"master2/port0": mqttData["master2/port0"],
					"master2/port1": mqttData["master2/port1"],
					"master2/port2": mqttData["master2/port2"],
				}
				utils.SaveToJSON(mqttOEE, config.MqttOeeFilePath)
				utils.SaveToJSON(mqttFlow, config.MqttFlowFilePath)

				// --- wyliczanie OEE ---
				core.CalculateData(mqttData)

				// --- zapis OEE w nowej strukturze ---
				core.SaveOeeFlat(config.OeeFilePath)
			}()
			time.Sleep(config.IntervalMQTTData)
		}
	})

	// --- MEASUREMENTS to DB ---
	utils.Go("MEASUREMENTS to DB", func() {
		for {
			func() {
				defer utils.Catch("MEASUREMENTS to DB")()
				core.SaveMeasurementsToDB(config.MeasurementFilePath)
			}()
			time.Sleep(config.MeasurementUpdateInterval)
		}
	})

	// --- METERS to DB ---
	utils.Go("METERS to DB", func() {
		for {
			func() {
				defer utils.Catch("METERS to DB")()
				core.SaveMetersToDB(config.MetersFilePath)
			}()
			time.Sleep(config.MetersUpdateInterval)
		}
	})

	// --- FLOW to DB ---
	utils.Go("FLOW to DB", func() {
		for {
			func() {
				defer utils.Catch("FLOW to DB")()
				core.SaveFlowDataToDB(config.MqttFlowFilePath)
			}()
			time.Sleep(config.FlowUpdateInterval)
		}
	})

	// --- OEE to DB ---
	utils.Go("OEE to DB", func() {
		for {
			func() {
				defer utils.Catch("OEE to DB")()
				core.SaveOeeTempToDB(config.OeeFilePath)
			}()
			time.Sleep(config.OEEUpdateInterval)
		}
	})

	// --- ALIVE Logger ---
	utils.Go("ALIVE Logger", func() {
		for {
			func() {
				defer utils.Catch("ALIVE Logger")()
				utils.LogMessage("[SYSTEM] App is alive")
			}()
			time.Sleep(30 * time.Minute)
		}
	})

	// --- sygnał stop (graceful w Dockerze) ---
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	utils.LogMessage("[SYSTEM] Stop signal received – shutting down.")
}
