package model

type Balance struct {
	Current  int64 `db:"current" json:"current"`
	Windrawn int64 `db:"windrawn" json:"windrawn"`
}
