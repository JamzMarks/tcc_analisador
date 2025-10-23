package types

type WayFlowResult struct {
	WayID                string  `json:"wayId"`
	RoadType             string  `json:"roadType"`
	AvgFlowByReliability float64 `json:"avgFlowByReliability"`
}
