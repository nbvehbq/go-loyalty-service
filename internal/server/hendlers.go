package server

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/nbvehbq/go-loyalty-service/internal/logger"
	"github.com/nbvehbq/go-loyalty-service/internal/model"
	"github.com/nbvehbq/go-loyalty-service/internal/storage"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

func setCookie(w http.ResponseWriter, payload string) {
	cookie := &http.Cookie{
		Name:     "session",
		Value:    payload,
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	http.SetCookie(w, cookie)
}

func (s *Server) registerHandler(res http.ResponseWriter, req *http.Request) {
	var err error
	defer func() {
		if err != nil {
			logger.Log.Error("error", zap.Error(err))
		}
	}()

	ctx := req.Context()

	var dto model.RegisterDTO
	if err = json.NewDecoder(req.Body).Decode(&dto); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(dto.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(res, "hash password", http.StatusInternalServerError)
		return
	}

	userId, err := s.storage.CreateUser(ctx, dto.Login, string(hash))
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrUserExists):
			http.Error(res, err.Error(), http.StatusConflict)
		default:
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	sid, err := s.session.Set(userId)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	setCookie(res, sid)

	res.WriteHeader(http.StatusOK)
}

func (s *Server) loginHandler(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	var dto model.RegisterDTO
	if err := json.NewDecoder(req.Body).Decode(&dto); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := s.storage.GetUserByLogin(ctx, dto.Login)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrUserNotFound):
			http.Error(res, err.Error(), http.StatusUnauthorized)
		default:
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(dto.Password)); err != nil {
		http.Error(res, "", http.StatusUnauthorized)
		return
	}

	sid, err := s.session.Set(user.Id)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	setCookie(res, sid)

	res.WriteHeader(http.StatusOK)
}

func (s *Server) uploadOrderHandler(res http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, b := range body {
		if b < 0x30 || b > 0x39 {
			http.Error(res, "", http.StatusBadRequest)
			return
		}
	}

	if !luhn(body) {
		http.Error(res, "", http.StatusUnprocessableEntity)
		return
	}

	ctx := req.Context()
	uid := ctx.Value("uid").(int64)

	order, err := s.storage.GetOrderByNumber(ctx, string(body))
	if err != nil && !errors.Is(err, storage.ErrOrderNotFound) {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	if order != nil {
		if order.UserId == uid {
			res.WriteHeader(http.StatusOK)
			return
		}

		http.Error(res, "", http.StatusConflict)
		return
	}

	if err := s.storage.CreateOrder(ctx, uid, string(body)); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusCreated)
}

func (s *Server) listOrderHandler(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	uid := ctx.Value("uid").(int64)

	orders, err := s.storage.ListOrders(ctx, uid)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(orders) == 0 {
		res.WriteHeader(http.StatusNoContent)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(res).Encode(orders); err != nil {
		logger.Log.Error("error", zap.Error(err))
	}
}

func luhn(s []byte) bool {
	var sum int
	for i := 0; i < len(s); i++ {
		v := int(s[i] - '0')
		if i&1 == len(s)&1 {
			v *= 2
			if v > 9 {
				v -= 9
			}
		}
		sum += v
	}

	return sum%10 == 0
}
