package model

type RegisterDTO struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type User struct {
	ID           int64  `db:"id" json:"id"`
	Login        string `db:"login" json:"login"`
	PasswordHash string `db:"password_hash" json:"-"`
}
