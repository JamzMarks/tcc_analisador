package main

import (
	"encoding/json"
	"fmt"
	"log"
	"tcc_analisador/services"
	"time"
)

func main() {
	cfg := services.LoadConfig()

	fmt.Print("ok")
	// Neo4j
	analyser := services.NewGraphService(cfg.GraphDBURI, cfg.GraphDBUser, cfg.GraphDBPass)
	defer analyser.Close()
	analyser.TestConnection()

	// RabbitMQ
	rabbit, err := services.NewRabbitService(cfg.RabbitURL)
	if err != nil {
		log.Fatalf("Erro ao conectar RabbitMQ: %v", err)
	}
	defer rabbit.Close()
	log.Printf("Escutando fila: %s no RabbitMQ: %s", cfg.QueueName, cfg.RabbitURL)

	// Ticker para análise a cada 45 segundos
	ticker := time.NewTicker(45 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Executa análise de vias
			ways, err := analyser.AnalyzeWays()
			if err != nil {
				log.Printf("Erro na análise de vias: %v", err)
				continue
			}
			log.Printf("Análise concluída para %d vias", len(ways))

			timestamp := time.Now().Unix()
			// marshal para JSON
			msgRabbit, err := json.Marshal(struct {
				Timestamp int64 `json:"timestamp"`
			}{
				Timestamp: timestamp,
			})
			if err != nil {
				fmt.Printf("[ERRO Marshal] %v\n", err)
				return
			}
			// Routing key: signal.update.<deviceID>
			rk := "analysis.completed"
			if err := rabbit.Publish("analysis.events", rk, msgRabbit); err != nil {
				fmt.Printf("[ERRO Rabbit]: %v\n", err)

				// Telemetria opcional
				// telemetry := types.TelemetryError{
				// 	DeviceID:  deviceID,
				// 	Payload:   msgRabbit,
				// 	ErrorMsg:  "Rabbit publish failed",
				// 	Timestamp: time.Now(),
				// 	Module:    "orquestrator",
				// 	Event:     "Publish message to Rabbit",
				// }
				// msgFail, _ := json.Marshal(telemetry)
				// mq.Publish("telemetry.events", "telemetry.signal.fail", msgFail)
			}
		default:
			time.Sleep(60 * time.Second)

		}
	}
}
