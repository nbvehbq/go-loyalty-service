package model

type Withdrawal struct {
	ID        int64   `db:"id" json:"-"`
	UserID    int64   `db:"user_id" json:"-"`
	Order     string  `db:"order" json:"order"`
	Sum       float64 `db:"sum" json:"sum"`
	CreatedAt string  `db:"created_at" json:"processed_at"`
}

type WithdrawalDTO struct {
	UserID int64   `json:"-"`
	Order  string  `json:"order"`
	Sum    float64 `json:"sum"`
}
