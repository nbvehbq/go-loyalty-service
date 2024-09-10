package model

import (
	"database/sql"
	"encoding/json"
)

type Balance struct {
	Current   float64         `db:"current" json:"current"`
	Withdrawn sql.NullFloat64 `db:"withdrawn" json:"withdrawn"`
}

func (v Balance) MarshalJSON() ([]byte, error) {
	type Balancelias Balance

	value := v.Withdrawn.Float64

	aliasValue := struct {
		Balancelias
		Withdrawn float64 `json:"withdrawn"`
	}{
		Balancelias: (Balancelias)(v),
		Withdrawn:   value,
	}

	return json.Marshal(aliasValue)
}
