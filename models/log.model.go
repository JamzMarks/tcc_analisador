package models

import "time"

type MetricEvent struct {
	EdgeID    string    // id da aresta CONNECT_TO
	DeviceID  string    // id do device que enviou
	Flow      float64   // [0..1]
	Timestamp time.Time // momento da medição
}
