package server

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/nbvehbq/go-loyalty-service/internal/logger"
)

type Repository interface{}

type Server struct {
	srv     *http.Server
	storage Repository
	DSN     string
}

func NewServer(storage Repository, cfg *Config) (*Server, error) {
	mux := chi.NewRouter()

	s := &Server{
		srv:     &http.Server{Addr: cfg.ServerAddress, Handler: mux},
		storage: storage,
		DSN:     cfg.DSN,
	}

	mux.Get(`/`, logger.WithLogging(s.helloHandler))

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
