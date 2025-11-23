package services

import (
	"flag"
	"os"
	"strconv"
)

type Config struct {
	DeviceAPI   string
	RabbitURL   string
	PollMs      int
	QueueName   string
	GraphDBURI  string
	GraphDBUser string
	GraphDBPass string
	Port        int
}

func LoadConfig() *Config {
	deviceAPI := flag.String("device-api-url", getenv("DEVICE_API_URL", "http://host.docker.internal:3005/api/v1/camera"), "URL to fetch devices")
	rabbitURL := flag.String("rabbitmq-url", getenv("AMQP_URL", "amqp://user:pass@host.docker.internal:5672/"), "RabbitMQ connection URL")
	pollMs := flag.Int("poll-ms", atoiDefault(getenv("POLL_MS", "30000"), 30000), "Polling interval")
	queue := flag.String("queue", getenv("QUEUE_NAME", "injector_queue"), "RabbitMQ queue name")

	graphURI := flag.String("graph-db-uri", getenv("GRAPH_DB_URI", "neo4j://host.docker.internal:7687"), "Neo4j database URI")
	graphUser := flag.String("graph-db-username", getenv("GRAPH_DB_USERNAME", "neo4j"), "Neo4j username")
	graphPass := flag.String("graph-db-password", getenv("GRAPH_DB_PASSWORD", "password"), "Neo4j password")

	port := flag.Int("port", atoiDefault(getenv("PORT", "7474"), 7474), "HTTP service port")

	flag.Parse()

	return &Config{
		DeviceAPI:   *deviceAPI,
		RabbitURL:   *rabbitURL,
		PollMs:      *pollMs,
		QueueName:   *queue,
		GraphDBURI:  *graphURI,
		GraphDBUser: *graphUser,
		GraphDBPass: *graphPass,
		Port:        *port,
	}
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func atoiDefault(s string, d int) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return d
	}
	return i
}

// func atoi64Default(s string, d int64) int64 {
// 	i, err := strconv.ParseInt(s, 10, 64)
// 	if err != nil {
// 		return d
// 	}
// 	return i
// }

// func atofDefault(s string, d float64) float64 {
// 	f, err := strconv.ParseFloat(s, 64)
// 	if err != nil {
// 		return d
// 	}
// 	return f
// }
