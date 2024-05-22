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
	Accrual    int       `json:"accrual"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type Withdrawal struct {
	OrderNumber string    `json:"order"`
	Sum         int       `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}

func NewUser(login string, originalPassword string) User {
	return User{Login: login, Password: utils.PasswordHash(originalPassword)}
}
