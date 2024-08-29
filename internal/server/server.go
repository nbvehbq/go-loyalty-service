package server

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/nbvehbq/go-loyalty-service/internal/logger"
	"github.com/nbvehbq/go-loyalty-service/internal/model"
)

type Repository interface {
	CreateUser(ctx context.Context, login, pass string) (int64, error)
	GetUserByLogin(ctx context.Context, login string) (*model.User, error)
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
	mux := chi.NewRouter()

	s := &Server{
		srv:     &http.Server{Addr: cfg.ServerAddress, Handler: mux},
		storage: storage,
		session: session,
		DSN:     cfg.DSN,
	}

	mux.Post(`/api/user/register`, logger.WithLogging(s.registerHandler))
	mux.Post(`/api/user/login`, logger.WithLogging(s.loginHandler))

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
