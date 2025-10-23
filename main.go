package main

import (
	"fmt"
	"log"
	"os"
	"tcc_analisador/services"
	"time"
)

func main() {
	amqpURL := os.Getenv("AMQP_URL")
	queueName := os.Getenv("QUEUE_NAME")
	if amqpURL == "" || queueName == "" {
		log.Fatal("Variáveis de ambiente do broker não configuradas")
	}

	uri := os.Getenv("GRAPH_DB_URI")
	username := os.Getenv("GRAPH_DB_USERNAME")
	password := os.Getenv("GRAPH_DB_PASSWORD")
	if uri == "" || username == "" || password == "" {
		log.Fatal("Variáveis de ambiente do banco de grafos não configuradas")
	}

	fmt.Print("ok")
	// Neo4j
	analyser := services.NewGraphService(uri, username, password)
	defer analyser.Close()
	analyser.TestConnection()

	// RabbitMQ
	rabbit, err := services.NewRabbitService(amqpURL, queueName)
	if err != nil {
		log.Fatalf("Erro ao conectar RabbitMQ: %v", err)
	}
	defer rabbit.Close()
	log.Printf("Escutando fila: %s no RabbitMQ: %s", queueName, amqpURL)

	// Ticker para análise a cada 30 segundos
	ticker := time.NewTicker(30 * time.Second)
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

		default:
			log.Printf("algo")
			time.Sleep(30000 * time.Millisecond)
			// // Processa mensagens do RabbitMQ se houver
			// msg, ok := rabbit.GetMessageNonBlocking()
			// if ok {
			// 	// Aqui você processa a mensagem
			// 	err := processMessage(msg, analyser)
			// 	if err != nil {
			// 		log.Printf("Erro ao processar mensagem: %v", err)
			// 	}
			// } else {
			// 	// Pequena pausa para não travar CPU
			// 	time.Sleep(100 * time.Millisecond)
			// }
		}
	}
}
