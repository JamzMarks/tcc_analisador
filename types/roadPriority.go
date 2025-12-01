package types

var RoadPriorityMap = map[string]float64{
	"motorway":       1.00,
	"motorway_link":  0.90,
	"trunk":          0.90,
	"primary":        0.88,
	"secondary":      0.82,
	"tertiary":       0.80,
	"primary_link":   0.88,
	"secondary_link": 0.82,
	"tertiary_link":  0.80,
	"unclassified":   0.75,
	"residential":    0.75,
	"service":        0.30,
	"living_street":  0.25,
	"pedestrian":     0.20,
}

func GetRoadPriority(roadType string) float64 {
	if val, ok := RoadPriorityMap[roadType]; ok {
		return val
	}
	return 0.50
}
