package cfg

const (
	UserTable = "id bigint, " +
		"balance int, " +
		"advert_channel int, " +
		"referral_count int, " +
		"take_bonus boolean, " +
		"lang text, " +
		"register_time bigint, " +
		"min_withdrawal int, " +
		"first_withdrawal boolean"

	Links = "hash text, " +
		"referral_id bigint, " +
		"source text"

	Subs = "id int"
)

type DBConfig struct {
	User     string
	Password string
	Names    map[string]string
}

////DBCfg Local config
//var DBCfg = DBConfig{
//	User:     "root",
//	Password: "",
//	Names:    map[string]string{"it": "italy"},
//}

//DBCfg Server config
var DBCfg = DBConfig{
	User:     "root",
	Password: ":!BlackR1",
	Names:    map[string]string{"es": "espany", "it": "italy_tok", "br": "portugaly", "mx": "mexico", "de": "germany"},
}

//DBCfg Local Server Test config
//var DBCfg = DBConfig{
//	User:     "root",
//	Password: "",
//	Names:    map[string]string{"es": "espany", "it": "italy", "br": "portugaly", "mx": "mexico"},
//}

////DBCfg My Server config
//var DBCfg = DBConfig{
//	User:     "root",
//	Password: ":root",
//	Names:    map[string]string{"it": "italy"},
//}
