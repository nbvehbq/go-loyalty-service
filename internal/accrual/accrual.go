package accrual

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/nbvehbq/go-loyalty-service/internal/logger"
	"github.com/nbvehbq/go-loyalty-service/internal/model"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var (
	ErrToManyRequests     = errors.New("too many requests")
	ErrOrderNotRegistered = errors.New("order not registered")
)

type Storage interface {
	ListUnaccruedOrders(context.Context) ([]model.Order, error)
	SetAccrual(context.Context, model.Accrual) (*model.Order, error)
}

type Accrual struct {
	client     *resty.Client
	address    string
	retryAfter atomic.Value
	storage    Storage
}

func NewAccrual(addr string, storage Storage) *Accrual {
	c := resty.New()

	a := &Accrual{
		address: addr,
		client:  c,
		storage: storage,
	}

	a.retryAfter.Store(0)

	return a
}

func (a *Accrual) Run(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second * 1):
				if err := a.do(ctx); err != nil {
					logger.Log.Error("do", zap.Error(err))
				}
			}
		}
	}()
}

func (a *Accrual) do(ctx context.Context) error {
	orders, err := a.storage.ListUnaccruedOrders(ctx)
	if err != nil {
		return errors.Wrap(err, "list unaccrued orders")
	}

	input := a.generator(ctx, orders)

	result := a.setAccrual(ctx, a.getAccrual(ctx, input))
	for res := range result {
		logger.Log.Info("got result", zap.Any("order", res))
	}

	return nil
}

func (a *Accrual) generator(ctx context.Context, input []model.Order) chan model.Order {
	output := make(chan model.Order)

	go func() {
		defer close(output)

		for _, order := range input {
			select {
			case <-ctx.Done():
				return
			case output <- order:
			}
		}
	}()

	return output
}

func (a *Accrual) setAccrual(ctx context.Context, input chan model.Accrual) chan model.Order {
	result := make(chan model.Order)

	go func() {
		defer close(result)

		for accrual := range input {

			order, err := a.storage.SetAccrual(ctx, accrual)
			if err != nil {
				logger.Log.Error("set accrual", zap.Error(err))
				continue
			}

			select {
			case <-ctx.Done():
				return
			case result <- *order:
			}
		}

	}()

	return result
}

func (a *Accrual) getAccrual(ctx context.Context, input chan model.Order) chan model.Accrual {
	result := make(chan model.Accrual)

	go func() {
		defer close(result)

		for order := range input {
			if a.retryAfter.Load().(int) > 0 {
				logger.Log.Info("too many requests. Wait... ")
				continue
			}

			var accrual model.Accrual
			res, err := a.client.R().
				SetResult(&accrual).
				Get(fmt.Sprintf("%s/api/orders/%s", a.address, order.Number))

			if err != nil {
				logger.Log.Error("resty post", zap.Error(err))
				continue
			}

			if res.StatusCode() == http.StatusTooManyRequests {
				val := res.Header().Get("Retry-After")
				retryAfter, err := strconv.Atoi(val)
				if err != nil {
					logger.Log.Error("parse retry-after", zap.Error(err))
					continue
				}
				if a.retryAfter.Load().(int) == 0 {
					logger.Log.Info("too many requests", zap.Int("retry-after", retryAfter))
					a.retryAfter.Store(retryAfter)
					go a.wait(ctx, retryAfter)
				}

				continue
			}

			if res.StatusCode() == http.StatusNoContent {
				// logger.Log.Error("status", zap.Error(ErrOrderNotRegistered))
				continue
			}

			if res.StatusCode() != http.StatusOK {
				logger.Log.Error("status", zap.Int("code", res.StatusCode()))
				continue
			}

			accrual.UserID = order.UserID

			select {
			case <-ctx.Done():
				return
			case result <- accrual:
			}
		}
	}()

	return result
}

func (a *Accrual) wait(ctx context.Context, seconds int) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(time.Duration(seconds) * time.Second):
		a.retryAfter.Store(0)
	}
}
