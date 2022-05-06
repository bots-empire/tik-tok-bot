package model

type User struct {
	ID              int64  `json:"id"`
	Balance         int    `json:"balance"`
	AdvertChannel   int    `json:"advert_channel"`
	ReferralCount   int    `json:"referral_count"`
	TakeBonus       bool   `json:"take_bonus"`
	Language        string `json:"lang"`
	RegisterTime    int64  `json:"register_time"`
	MinWithdrawal   int    `json:"min_withdrawal"`
	FirstWithdrawal bool   `json:"first_withdrawal"`
}
