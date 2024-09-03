package server

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/nbvehbq/go-loyalty-service/internal/logger"
	"github.com/nbvehbq/go-loyalty-service/internal/model"
)

type Repository interface {
	CreateUser(ctx context.Context, login, pass string) (int64, error)
	GetUserByLogin(ctx context.Context, login string) (*model.User, error)
	CreateOrder(ctx context.Context, uid int64, order string) error
	GetOrderByNumber(ctx context.Context, number string) (*model.Order, error)
	ListOrders(ctx context.Context, uid int64) ([]*model.Order, error)
}

type SessionStorage interface {
	Set(int64) (string, error)
	Get(string) (int64, bool)
}

type Server struct {
	srv     *http.Server
	storage Repository
	session SessionStorage
	DSN     string
}

func NewServer(storage Repository, session SessionStorage, cfg *Config) (*Server, error) {
	r := chi.NewRouter()

	s := &Server{
		srv:     &http.Server{Addr: cfg.ServerAddress, Handler: r},
		storage: storage,
		session: session,
		DSN:     cfg.DSN,
	}

	r.Use(logger.Middleware)
	r.Use(middleware.Recoverer)

	// Public routes
	r.Group(func(r chi.Router) {
		r.Post(`/api/user/register`, s.registerHandler)
		r.Post(`/api/user/login`, s.loginHandler)
	})

	// Private routes
	r.Group(func(r chi.Router) {
		r.Use(Authenticator(s.session))

		r.Post(`/api/user/orders`, s.uploadOrderHandler)
		r.Get(`/api/user/orders`, s.listOrderHandler)
	})

	return s, nil
}

func (s *Server) Run(ctx context.Context) error {
	logger.Log.Info("Server started.")

	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	logger.Log.Info("Server stoped.")

	return s.srv.Shutdown(ctx)
}
