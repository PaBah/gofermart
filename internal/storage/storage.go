package storage

import (
	"context"
	"errors"

	"github.com/PaBah/gofermart/internal/models"
)

var ErrAlreadyExists = errors.New("already exists")

type Repository interface {
	CreateUser(ctx context.Context, user models.User) (models.User, error)
	AuthorizeUser(ctx context.Context, login string) (models.User, error)
	RegisterOrder(ctx context.Context, orderNumber string) (models.Order, error)
	GetUsersOrders(ctx context.Context) ([]models.Order, error)
	GetUsersBalance(ctx context.Context) (int, error)
	GetUsersWithdraw(ctx context.Context) (int, error)
	CreateWithdrawal(ctx context.Context, withdrawal models.Withdrawal) (models.Withdrawal, error)
	GetUsersWithdrawals(ctx context.Context) ([]models.Withdrawal, error)
	GetAllOrdersIDs(ctx context.Context) ([]string, error)
	UpdateOrder(ctx context.Context, order models.Order) (models.Order, error)
}
