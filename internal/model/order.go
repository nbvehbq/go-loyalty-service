package model

import (
	"database/sql"
	"encoding/json"
)

type Order struct {
	Id        int64         `db:"id" json:"-"`
	Number    string        `db:"number" json:"number"`
	UserId    int64         `db:"user_id" json:"-"`
	Status    string        `db:"status" json:"status"`
	Accrual   sql.NullInt64 `db:"accrual" json:"accrual,omitempty"`
	CreatedAt string        `db:"created_at" json:"uploaded_at"`
}

func (v Order) MarshalJSON() (b []byte, err error) {
	type OrderAlias Order

	value := v.Accrual.Int64

	aliasValue := struct {
		OrderAlias
		Accrual int64 `json:"accrual,omitempty"`
	}{
		OrderAlias: (OrderAlias)(v),
		Accrual:    value,
	}

	return json.Marshal(aliasValue)
}
