package types

var RoadPriorityMap = map[string]float64{
	"motorway":       1.00,
	"motorway_link":  0.70,
	"trunk":          0.90,
	"primary":        0.80,
	"secondary":      0.70,
	"tertiary":       0.60,
	"primary_link":   0.60,
	"secondary_link": 0.60,
	"tertiary_link":  0.60,
	"unclassified":   0.50,
	"residential":    0.40,
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
