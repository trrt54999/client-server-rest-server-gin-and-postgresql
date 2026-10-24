package dto

type UpdateIncidentStatusDto struct {
	Status string `json:"status" binding:"required,oneof=open in_progress resolved closed"`
}
