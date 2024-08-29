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
