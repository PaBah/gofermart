package storage

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/PaBah/gofermart/internal/auth"
	"github.com/PaBah/gofermart/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestDBStorage_Close(t *testing.T) {
	db, mock, _ := sqlmock.New()
	dbStorage := DBStorage{db: db}
	mock.ExpectClose().WillReturnError(nil)

	err := dbStorage.Close()
	assert.NoError(t, err)
}

func TestDBStorage_CreateAuthUser(t *testing.T) {
	db, mock, _ := sqlmock.New()
	ds := &DBStorage{
		db: db,
	}
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO users(login, password) VALUES ($1, $2)")).
		WithArgs("test", "test").WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, password FROM users WHERE login=$1")).
		WithArgs("test").
		WillReturnRows(sqlmock.NewRows([]string{"id", "password"}).
			AddRow("test", "test"))

	user := models.User{Login: "test", Password: "test"}
	createdUser, err := ds.CreateUser(context.Background(), user)
	assert.NoError(t, err, "User created without error")
	assert.Equal(t, "test", createdUser.Login, "User store correctly")
}

func TestDBStorage_RegisterOrder(t *testing.T) {
	db, mock, _ := sqlmock.New()
	ds := &DBStorage{
		db: db,
	}
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO orders(number, user_id) VALUES ($1, $2)")).
		WithArgs("test", "test").WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT number, user_id, uploaded_at FROM orders WHERE number=$1")).
		WithArgs("test").
		WillReturnRows(sqlmock.NewRows([]string{"number", "user_id", "uploaded_at"}).
			AddRow("test", "test", time.Now()))

	createdOrder, err := ds.RegisterOrder(context.WithValue(context.Background(), auth.ContextUserKey, "test"), "test")
	assert.NoError(t, err, "User created without error")
	assert.Equal(t, "test", createdOrder.Number, "Order store correctly")
	assert.Equal(t, "test", createdOrder.UserID, "Order owner store correctly")
}

func TestDBStorage_UpdateOrder(t *testing.T) {
	db, mock, _ := sqlmock.New()
	ds := &DBStorage{
		db: db,
	}
	mock.ExpectExec(regexp.QuoteMeta("UPDATE orders SET accrual=$1, status=$2 WHERE number=$3")).
		WithArgs(123.4, "PROCESSED", "test").WillReturnResult(sqlmock.NewResult(1, 1))
	order := models.Order{Number: "test", Accrual: 123.4, Status: "PROCESSED"}
	updatedOrder, err := ds.UpdateOrder(context.WithValue(context.Background(), auth.ContextUserKey, "test"), order)
	assert.NoError(t, err, "User created without error")
	assert.Equal(t, "test", updatedOrder.Number, "Order store correctly")
	assert.Equal(t, "PROCESSED", updatedOrder.Status, "Order status store correctly")
	assert.Equal(t, 123.4, updatedOrder.Accrual, "Order accrual store correctly")
}

func TestDBStorage_GetUsersOrders(t *testing.T) {
	db, mock, _ := sqlmock.New()
	ds := &DBStorage{
		db: db,
	}
	timestamp := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT number, status, accrual, uploaded_at FROM orders where user_id=$1")).
		WithArgs("test").
		WillReturnRows(sqlmock.NewRows([]string{"number", "status", "accrual", "uploaded_at"}).
			AddRow("test", "NEW", 0, timestamp))

	orders, err := ds.GetUsersOrders(context.WithValue(context.Background(), auth.ContextUserKey, "test"))
	assert.NoError(t, err, "NO error on orders list")
	assert.Equal(t, orders, []models.Order{models.Order{Number: "test", Status: "NEW", Accrual: 0, UploadedAt: timestamp}}, "Order lists equal")
}

func TestDBStorage_GetUsersWithdrawals(t *testing.T) {
	db, mock, _ := sqlmock.New()
	ds := &DBStorage{
		db: db,
	}
	timestamp := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT number, sum, processed_at FROM withdrawals WHERE user_id=$1 ORDER BY processed_at DESC")).
		WithArgs("test").
		WillReturnRows(sqlmock.NewRows([]string{"number", "sum", "processed_at"}).
			AddRow("test", 0, timestamp))

	withdrawals, err := ds.GetUsersWithdrawals(context.WithValue(context.Background(), auth.ContextUserKey, "test"))
	assert.NoError(t, err, "NO error on withdrawals list")
	assert.Equal(t, withdrawals, []models.Withdrawal{models.Withdrawal{OrderNumber: "test", Sum: 0, ProcessedAt: timestamp}}, "Withdrawal lists equal")
}

func TestDBStorage_CreateWithdrawal(t *testing.T) {
	db, mock, _ := sqlmock.New()
	ds := &DBStorage{
		db: db,
	}
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO withdrawals(number, sum, user_id) VALUES ($1, $2, $3)")).
		WithArgs("test", 123.4, "test").WillReturnResult(sqlmock.NewResult(1, 1))

	withdrawal := models.Withdrawal{OrderNumber: "test", Sum: 123.4}
	_, err := ds.CreateWithdrawal(context.WithValue(context.Background(), auth.ContextUserKey, "test"), withdrawal)
	assert.NoError(t, err, "User created without error")
}

func TestDBStorage_GetUsersBalance(t *testing.T) {
	db, mock, _ := sqlmock.New()
	ds := &DBStorage{
		db: db,
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT SUM(accrual) FROM orders WHERE user_id=$1")).
		WithArgs("test").
		WillReturnRows(sqlmock.NewRows([]string{"sum_accrual"}).
			AddRow(123.7))

	balance, err := ds.GetUsersBalance(context.WithValue(context.Background(), auth.ContextUserKey, "test"))
	assert.NoError(t, err, "NO error on balance")
	assert.Equal(t, balance, 123.7, "Balances equal")
}

func TestDBStorage_GetUsersWithdraw(t *testing.T) {
	db, mock, _ := sqlmock.New()
	ds := &DBStorage{
		db: db,
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT SUM(sum) FROM withdrawals WHERE user_id=$1")).
		WithArgs("test").
		WillReturnRows(sqlmock.NewRows([]string{"sum_sum"}).
			AddRow(12.3))

	withdraw, err := ds.GetUsersWithdraw(context.WithValue(context.Background(), auth.ContextUserKey, "test"))
	assert.NoError(t, err, "NO error on withdraw")
	assert.Equal(t, withdraw, 12.3, "Withdraws equal")
}

func TestDBStorage_GetAllOrdersIDs(t *testing.T) {
	db, mock, _ := sqlmock.New()
	ds := &DBStorage{
		db: db,
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT number FROM orders WHERE status in ('NEW', 'PROCESSING')")).
		WillReturnRows(sqlmock.NewRows([]string{"number"}).
			AddRow("test1").AddRow("test2").AddRow("test3"))

	orderIDs, err := ds.GetAllOrdersIDs(context.WithValue(context.Background(), auth.ContextUserKey, "test"))
	assert.NoError(t, err, "NO error on orders IDs list")
	assert.Equal(t, orderIDs, []string{"test1", "test2", "test3"}, "Order IDs lists equal")
}
