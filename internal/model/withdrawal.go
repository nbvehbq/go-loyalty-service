package model

type Withdrawal struct {
	Id        int64  `db:"id" json:"-"`
	UserId    int64  `db:"user_id" json:"-"`
	Order     string `db:"order" json:"order"`
	Sum       int64  `db:"sum" json:"sum"`
	CreatedAt string `db:"created_at" json:"processed_at"`
}

type WithdrawalDTO struct {
	UserId int64  `json:"-"`
	Order  string `json:"order"`
	Sum    int64  `json:"sum"`
}
