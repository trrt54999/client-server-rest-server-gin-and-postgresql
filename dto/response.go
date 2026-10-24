package dto

type ErrorResponse struct {
	Error string `json:"error" example:"Incident not found!"`
}

type TokenResponse struct {
	Token string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}
