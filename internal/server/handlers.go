package server

import (
	"context"
	"database/sql"
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

	userID, err := s.storage.CreateUser(ctx, dto.Login, string(hash))
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrUserExists):
			http.Error(res, err.Error(), http.StatusConflict)
		default:
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	sid, err := s.session.Set(req.Context(), userID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	setCookie(res, sid)
	res.Header().Set("Authorization", sid)

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

	sid, err := s.session.Set(req.Context(), user.ID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	setCookie(res, sid)
	res.Header().Set("Authorization", sid)

	res.WriteHeader(http.StatusOK)
}

func (s *Server) uploadOrderHandler(res http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	if ok, code := validateOrderID(body); !ok {
		http.Error(res, "", code)
		return
	}

	ctx := req.Context()
	uid := UID(ctx)

	if _, err := s.storage.CreateOrder(ctx, uid, string(body)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			res.WriteHeader(http.StatusOK)
			return
		}
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusAccepted)
}

func (s *Server) listOrderHandler(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	uid := UID(ctx)

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

func (s *Server) getBalanceHandler(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	uid := UID(ctx)

	balance, err := s.storage.GetBalance(ctx, uid)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(res).Encode(balance); err != nil {
		logger.Log.Error("error", zap.Error(err))
	}
}

func (s *Server) listWithdrawalsHandler(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	uid := UID(ctx)

	withdrawals, err := s.storage.ListWithdrawals(ctx, uid)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(withdrawals) == 0 {
		res.WriteHeader(http.StatusNoContent)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(res).Encode(withdrawals); err != nil {
		logger.Log.Error("error", zap.Error(err))
	}
}

func (s *Server) withdrawHandler(res http.ResponseWriter, req *http.Request) {
	uid := UID(req.Context())

	var dto model.WithdrawalDTO
	if err := json.NewDecoder(req.Body).Decode(&dto); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	if ok, code := validateOrderID([]byte(dto.Order)); !ok {
		http.Error(res, "", code)
		return
	}

	dto.UserID = uid
	if err := s.storage.CreateWithdrawal(req.Context(), &dto); err != nil {
		if errors.Is(err, storage.ErrBalanceInsufficient) {
			http.Error(res, "", http.StatusPaymentRequired)
			return
		}
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
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

func UID(ctx context.Context) int64 {
	uid := ctx.Value(uidKey).(int64)
	return uid
}

func validateOrderID(b []byte) (bool, int) {
	if !luhn(b) {
		return false, http.StatusUnprocessableEntity
	}

	return true, 0
}
