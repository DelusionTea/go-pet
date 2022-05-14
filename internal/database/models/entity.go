package models

import "time"

type ResponseOrder struct {
	Order      string    `json:"number"`
	Status     string    `json:"status"`
	Accrual    float32   `json:"accrual"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type BalanceResponse struct {
	Current   float32 `json:"current"`
	Withdrawn float32 `json:"withdrawn"`
}

type ResponseWithdraws struct {
	Order       string    `json:"order"`
	Sum         float32   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}

type ResponseOrderInfo struct {
	Order   string  `json:"number"`
	Status  string  `json:"status"`
	Accrual float32 `json:"accrual"`
}
