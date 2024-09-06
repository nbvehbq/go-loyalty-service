package model

import (
	"database/sql"
	"encoding/json"
)

type Balance struct {
	Current  float64         `db:"current" json:"current"`
	Windrawn sql.NullFloat64 `db:"windrawn" json:"windrawn"`
}

func (v Balance) MarshalJSON() ([]byte, error) {
	type Balancelias Balance

	value := v.Windrawn.Float64

	aliasValue := struct {
		Balancelias
		Windrawn float64 `json:"windrawn"`
	}{
		Balancelias: (Balancelias)(v),
		Windrawn:    value,
	}

	return json.Marshal(aliasValue)
}
