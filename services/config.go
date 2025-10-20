package services

import (
	"flag"
	"os"
	"strconv"
)

type Config struct {
	DeviceAPI  string
	RabbitURL  string
	PollMs     int
	QueueName  string
	DeltaRange float64
	Seed       int64
}

func LoadConfig() *Config {
	deviceAPI := flag.String("device-api-url", getenv("DEVICE_API_URL", "http://host.docker.internal:3005/api/v1/camera"), "URL to fetch devices")
	rabbitURL := flag.String("rabbitmq-url", getenv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"), "RabbitMQ connection URL")
	pollMs := flag.Int("poll-ms", atoiDefault(getenv("POLL_MS", "30000"), 30000), "Polling interval")
	queue := flag.String("queue", getenv("QUEUE_NAME", "injector_queue"), "RabbitMQ queue name")
	delta := flag.Float64("delta-range", atofDefault(getenv("DELTA_RANGE", "0.09"), 0.09), "Max delta per tick")
	seed := flag.Int64("seed", atoi64Default(getenv("RANDOM_SEED", "0"), 0), "Random seed (0 = time-based)")
	flag.Parse()

	return &Config{
		DeviceAPI:  *deviceAPI,
		RabbitURL:  *rabbitURL,
		PollMs:     *pollMs,
		QueueName:  *queue,
		DeltaRange: *delta,
		Seed:       *seed,
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

func atoi64Default(s string, d int64) int64 {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return d
	}
	return i
}

func atofDefault(s string, d float64) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return d
	}
	return f
}
