package models

type NodeGroup struct {
    InstanceType string  `json:"instance_type"`
    AZ           string  `json:"az"`
    Region       string  `json:"region"`
    Count        int     `json:"count"`
    IsSpot       bool    `json:"is_spot"`
}
