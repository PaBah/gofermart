package models

import (
	"time"

	"github.com/PaBah/gofermart/internal/utils"
)

type User struct {
	ID       string `json:"id"`
	Login    string `json:"login"`
	Password string `json:"-"`
}

type Order struct {
	Number     string    `json:"number"`
	UserID     string    `json:"-"`
	Status     string    `json:"status"`
	Accrual    float64   `json:"accrual"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type Withdrawal struct {
	OrderNumber string    `json:"order"`
	Sum         float64   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}

func NewUser(login string, originalPassword string) User {
	return User{Login: login, Password: utils.PasswordHash(originalPassword)}
}
