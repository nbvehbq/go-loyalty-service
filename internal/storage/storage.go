package storage

import "errors"

var (
	ErrUserExists          = errors.New("user exists")
	ErrUserNotFound        = errors.New("user not found")
	ErrOrderNotFound       = errors.New("order not found")
	ErrBalanceInsufficient = errors.New("balance insufficient")
)
