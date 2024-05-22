package dto

import (
	"fmt"
	"time"
)

type JSONTime time.Time

func (t JSONTime) MarshalJSON() ([]byte, error) {
	//do your serializing here
	stamp := fmt.Sprintf(`"%s"`, time.Time(t).Format(time.RFC3339))
	return []byte(stamp), nil
}

type (
	RegisterUserRequest struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}

	LoginUserRequest struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}

	ActualOrderStateResponse struct {
		Number     string   `json:"number"`
		Status     string   `json:"status"`
		Accrual    int      `json:"accrual,omitempty"`
		UploadedAt JSONTime `json:"uploaded_at"`
	}

	UserBalanceResponse struct {
		Current   float64 `json:"current"`
		Withdrawn float64 `json:"withdrawn"`
	}

	WithdrawalRequest struct {
		Number string  `json:"order"`
		Sum    float64 `json:"sum"`
	}

	WithdrawalsResponse struct {
		OrderNumber string   `json:"order"`
		Sum         float64  `json:"sum"`
		ProcessedAt JSONTime `json:"processed_at"`
	}
)
