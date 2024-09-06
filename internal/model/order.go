package model

import (
	"database/sql"
	"encoding/json"
)

type Order struct {
	ID        int64           `db:"id" json:"-"`
	Number    string          `db:"number" json:"number"`
	UserID    int64           `db:"user_id" json:"-"`
	Status    string          `db:"status" json:"status"`
	Accrual   sql.NullFloat64 `db:"accrual" json:"accrual,omitempty"`
	CreatedAt string          `db:"created_at" json:"uploaded_at"`
}

func (v Order) MarshalJSON() ([]byte, error) {
	type OrderAlias Order

	value := v.Accrual.Float64

	aliasValue := struct {
		OrderAlias
		Accrual float64 `json:"accrual,omitempty"`
	}{
		OrderAlias: (OrderAlias)(v),
		Accrual:    value,
	}

	return json.Marshal(aliasValue)
}
