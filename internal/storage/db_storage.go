package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/PaBah/gofermart/db"
	"github.com/PaBah/gofermart/internal/auth"
	"github.com/PaBah/gofermart/internal/models"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type DBStorage struct {
	db *sql.DB
}

func (ds *DBStorage) initialize(ctx context.Context, databaseDSN string) (err error) {

	ds.db, err = sql.Open("pgx", databaseDSN)
	if err != nil {
		return
	}

	driver, err := iofs.New(db.MigrationsFS, "migrations")
	if err != nil {
		return err
	}

	d, err := postgres.WithInstance(ds.db, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", driver, "psql_db", d)
	if err != nil {
		return err
	}

	_ = m.Up()
	return
}

func (ds *DBStorage) CreateUser(ctx context.Context, user models.User) (createdUser models.User, err error) {
	_, DBerr := ds.db.ExecContext(ctx,
		`INSERT INTO users(login, password) VALUES ($1, $2)`, user.Login, user.Password)

	var pgErr *pgconn.PgError
	if errors.As(DBerr, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
		err = ErrAlreadyExists
		return
	}

	return ds.AuthorizeUser(ctx, user.Login)
}

func (ds *DBStorage) AuthorizeUser(ctx context.Context, login string) (user models.User, err error) {
	row := ds.db.QueryRowContext(ctx, `SELECT id, password FROM users WHERE login=$1`, login)
	var id, password string
	err = row.Scan(&id, &password)

	if err != nil {
		return
	}

	user = models.User{ID: id, Login: login, Password: password}
	return
}

func (ds *DBStorage) RegisterOrder(ctx context.Context, orderNumber string) (order models.Order, err error) {
	userID := ctx.Value(auth.ContextUserKey).(string)
	_, DBerr := ds.db.ExecContext(ctx,
		`INSERT INTO orders(number, user_id) VALUES ($1, $2)`, orderNumber, userID)

	var pgErr *pgconn.PgError
	if errors.As(DBerr, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
		err = ErrAlreadyExists
	}

	row := ds.db.QueryRowContext(ctx, `SELECT number, user_id, uploaded_at FROM orders WHERE number=$1`, orderNumber)
	var number, ordersUserID string
	var uploadedAt time.Time

	_ = row.Scan(&number, &ordersUserID, &uploadedAt)
	order = models.Order{Number: number, UserID: ordersUserID, UploadedAt: uploadedAt}
	return
}

func (ds *DBStorage) UpdateOrder(ctx context.Context, order models.Order) (updatedOrder models.Order, err error) {
	_, err = ds.db.ExecContext(ctx,
		`UPDATE orders SET accrual=$1, status=$2 WHERE number=$3`, order.Accrual, order.Status, order.Number)
	if err == nil {
		updatedOrder = order
	}
	return
}

func (ds *DBStorage) GetUsersOrders(ctx context.Context) (orders []models.Order, err error) {
	userID := ctx.Value(auth.ContextUserKey).(string)

	rows, err := ds.db.QueryContext(ctx, `SELECT number, status, accrual, uploaded_at FROM orders where user_id=$1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders = make([]models.Order, 0)
	var accrual sql.NullFloat64
	var uploadedAt time.Time
	var status, number string

	for rows.Next() {
		err = rows.Scan(&number, &status, &accrual, &uploadedAt)
		if err != nil {
			return nil, err
		}
		orders = append(orders, models.Order{
			Number:     number,
			Status:     status,
			Accrual:    accrual.Float64,
			UploadedAt: uploadedAt,
		})
	}
	err = rows.Err()
	return
}

func (ds *DBStorage) GetUsersWithdrawals(ctx context.Context) (withdrawals []models.Withdrawal, err error) {
	userID := ctx.Value(auth.ContextUserKey).(string)

	rows, err := ds.db.QueryContext(ctx, `SELECT number, sum, processed_at FROM withdrawals WHERE user_id=$1 ORDER BY processed_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	withdrawals = make([]models.Withdrawal, 0)
	var sum sql.NullFloat64
	var processedAt time.Time
	var number string

	for rows.Next() {
		err = rows.Scan(&number, &sum, &processedAt)
		if err != nil {
			return nil, err
		}
		withdrawals = append(withdrawals, models.Withdrawal{
			OrderNumber: number,
			Sum:         sum.Float64,
			ProcessedAt: processedAt,
		})
	}
	err = rows.Err()
	return
}

func (ds *DBStorage) CreateWithdrawal(ctx context.Context, withdrawal models.Withdrawal) (createdWithdrawal models.Withdrawal, err error) {
	userID := ctx.Value(auth.ContextUserKey).(string)
	_, err = ds.db.ExecContext(ctx,
		`INSERT INTO withdrawals(number, sum, user_id) VALUES ($1, $2, $3)`, withdrawal.OrderNumber, withdrawal.Sum, userID)
	return
}

func (ds *DBStorage) GetUsersBalance(ctx context.Context) (balance float64, err error) {
	userID := ctx.Value(auth.ContextUserKey).(string)
	row := ds.db.QueryRowContext(ctx, `SELECT SUM(accrual) FROM orders WHERE user_id=$1`, userID)
	var nullBalance sql.NullFloat64

	err = row.Scan(&nullBalance)
	if err != nil {
		return
	}

	balance = nullBalance.Float64
	return
}

func (ds *DBStorage) GetUsersWithdraw(ctx context.Context) (withdraw float64, err error) {
	userID := ctx.Value(auth.ContextUserKey).(string)
	row := ds.db.QueryRowContext(ctx, `SELECT SUM(sum) FROM withdrawals WHERE user_id=$1`, userID)
	var nullWithdraw sql.NullFloat64

	err = row.Scan(&nullWithdraw)
	if err != nil {
		return
	}

	withdraw = nullWithdraw.Float64
	return
}

func (ds *DBStorage) GetAllOrdersIDs(ctx context.Context) (orderIDs []string, err error) {
	rows, err := ds.db.QueryContext(ctx, `SELECT number FROM orders WHERE status in ('NEW', 'PROCESSING')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orderIDs = make([]string, 0)
	var number string

	for rows.Next() {
		err = rows.Scan(&number)
		if err != nil {
			return nil, err
		}
		orderIDs = append(orderIDs, number)
	}
	err = rows.Err()
	return
}

func (ds *DBStorage) Close() error {
	return ds.db.Close()
}

func NewDBStorage(ctx context.Context, databaseDSN string) (DBStorage, error) {
	store := DBStorage{}
	err := store.initialize(ctx, databaseDSN)
	return store, err
}
