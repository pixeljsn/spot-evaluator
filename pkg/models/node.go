package models

type NodeGroup struct {
	InstanceType string  `json:"instance_type"`
	AZ           string  `json:"az"`
	Region       string  `json:"region"`
	Count        int     `json:"count"`
	IsSpot       bool    `json:"is_spot"`
	OnDemandPrice float64 `json:"on_demand_price"`
	SpotPrice     float64 `json:"spot_price"`
}
