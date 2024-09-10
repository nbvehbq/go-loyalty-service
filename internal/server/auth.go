package server

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
)

type contextKeyType string

const uidKey contextKeyType = "uid"

func Authenticator(s SessionStorage) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session")
			payload := r.Header.Get("Authorization")

			var sid string
			if errors.Is(err, http.ErrNoCookie) {
				sid = payload
			} else {
				sid = cookie.Value
			}

			uid, ok := s.Get(r.Context(), sid)
			if !ok {
				http.Error(w, "session not found", http.StatusUnauthorized)
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, uidKey, uid)

			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
}
