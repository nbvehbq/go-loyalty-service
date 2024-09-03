package postgres

import (
	"context"
	"database/sql"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/nbvehbq/go-loyalty-service/internal/model"
	"github.com/nbvehbq/go-loyalty-service/internal/storage"
	"github.com/pkg/errors"
)

const (
	StatusNew         = "NEW"
	StatusRegistered  = "REGISTERED"
	StatusInvalid     = "INVALID"
	StatusProccessing = "PROCESSING"
	StatusProcessed   = "PROCESSED"
)

type Storage struct {
	db *sqlx.DB
}

func NewStorage(ctx context.Context, DSN string) (*Storage, error) {
	db, err := sqlx.ConnectContext(ctx, "pgx", DSN)
	if err != nil {
		return nil, errors.Wrap(err, "connect to db")
	}

	if err := initDatabaseStructure(ctx, db); err != nil {
		return nil, errors.Wrap(err, "init db")
	}

	return &Storage{db: db}, nil
}

func initDatabaseStructure(ctx context.Context, db *sqlx.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS "user" (
	  id SERIAL NOT NULL,
	  login TEXT NOT NULL,
	  password_hash TEXT NOT NULL,

		CONSTRAINT "user_id_pkey" PRIMARY KEY ("id")
	);
	
	CREATE UNIQUE INDEX IF NOT EXISTS "user_login_key" ON "user"("login");

	CREATE TABLE IF NOT EXISTS "order" (
	  id SERIAL NOT NULL,
		number TEXT NOT NULL,
	  user_id INTEGER NOT NULL,
		status TEXT NOT NULL,
		accrual INT,
	  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

		CONSTRAINT "order_id_pkey" PRIMARY KEY ("id"),
		CONSTRAINT "order_number_key" UNIQUE ("number")
	);

	CREATE INDEX IF NOT EXISTS "order_createdAt_idx" ON "order"(created_at DESC);

	ALTER TABLE "order" DROP CONSTRAINT IF EXISTS "order_user_fkey";
	ALTER TABLE "order" ADD CONSTRAINT "order_user_fkey" FOREIGN KEY ("user_id") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;
	
	CREATE TABLE IF NOT EXISTS "withdrawal" (
	  id SERIAL NOT NULL,
		user_id INTEGER NOT NULL,
		"order" TEXT NOT NULL,
		sum INT NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

		CONSTRAINT "withdrawal_id_pkey" PRIMARY KEY ("id")
	);

	ALTER TABLE "withdrawal" DROP CONSTRAINT IF EXISTS "withdrawal_user_fkey";
	ALTER TABLE "withdrawal" ADD CONSTRAINT "withdrawal_user_fkey" FOREIGN KEY ("user_id") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;
	`
	_, err := db.ExecContext(ctx, query)
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) CreateUser(ctx context.Context, login, pass string) (int64, error) {
	var id int64
	query := `INSERT INTO "user" (login, password_hash) VALUES ($1, $2) returning id;`

	if err := s.db.QueryRowContext(ctx, query, login, pass).
		Scan(&id); err != nil {
		var pqErr *pgconn.PgError
		if errors.As(err, &pqErr) && pgerrcode.UniqueViolation == pqErr.Code {
			return id, storage.ErrUserExists
		}

		return 0, errors.Wrap(err, "create user")
	}

	return id, nil
}

func (s *Storage) GetUserByLogin(ctx context.Context, login string) (*model.User, error) {
	var user model.User
	query := `SELECT id, login, password_hash FROM "user" WHERE login = $1;`

	if err := s.db.GetContext(ctx, &user, query, login); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrUserNotFound
		}
		return nil, errors.Wrap(err, "get user")
	}

	return &user, nil
}

func (s *Storage) CreateOrder(ctx context.Context, uid int64, order string) error {
	query := `INSERT INTO "order" (number, user_id, status) VALUES ($1, $2, $3);`

	if _, err := s.db.ExecContext(ctx, query, order, uid, StatusNew); err != nil {
		return errors.Wrap(err, "create order")
	}

	return nil
}

func (s *Storage) GetOrderByNumber(ctx context.Context, number string) (*model.Order, error) {
	var order model.Order
	query := `SELECT id, number, user_id, status, accrual, created_at FROM "order" WHERE number = $1;`

	if err := s.db.GetContext(ctx, &order, query, number); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrOrderNotFound
		}
		return nil, errors.Wrap(err, "get order")
	}

	return &order, nil
}

func (s *Storage) ListOrders(ctx context.Context, uid int64) ([]*model.Order, error) {
	var orders []*model.Order
	query := `SELECT id, number, user_id, status, accrual, created_at FROM "order" 
	WHERE user_id = $1 ORDER BY created_at DESC;`

	if err := s.db.SelectContext(ctx, &orders, query, uid); err != nil {
		return nil, errors.Wrap(err, "list orders")
	}

	return orders, nil
}

func (s *Storage) GetBalance(ctx context.Context, uid int64) (*model.Balance, error) {
	var balance model.Balance
	query := `
	SELECT SUM(u.accrual) - SUM(u.windrawn) "current", SUM(u.windrawn) windrawn from (
		SELECT SUM(accrual) accrual, 0 windrawn FROM "order" WHERE user_id = $1
		union ALL 
		SELECT 0, SUM(sum) FROM "withdrawal" WHERE user_id = $1
	) u;`

	if err := s.db.GetContext(ctx, &balance, query, uid); err != nil {
		return nil, errors.Wrap(err, "get balance")
	}

	return &balance, nil
}

func (s *Storage) ListWithdrawals(ctx context.Context, uid int64) ([]*model.Withdrawal, error) {
	var withdrawals []*model.Withdrawal
	query := `SELECT id, user_id, "order", sum, created_at FROM "withdrawal" WHERE user_id = $1 ORDER BY created_at DESC;`

	if err := s.db.SelectContext(ctx, &withdrawals, query, uid); err != nil {
		return nil, errors.Wrap(err, "list withdrawals")
	}

	return withdrawals, nil
}

func (s *Storage) CreateWithdrawal(ctx context.Context, dto *model.WithdrawalDTO) error {
	query := `INSERT INTO "withdrawal" (user_id, "order", sum) VALUES ($1, $2, $3);`

	if _, err := s.db.ExecContext(ctx, query, dto.UserId, dto.Order, dto.Sum); err != nil {
		return errors.Wrap(err, "create withdrawal")
	}

	return nil
}
