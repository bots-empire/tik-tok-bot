package administrator

import (
	"database/sql"
	"log"
	"strconv"

	"github.com/Stepan1328/tik-tok-bot/model"
	"github.com/pkg/errors"
)

func (a *Admin) countUsers() int {
	dataBase := a.bot.GetDataBase()
	rows, err := dataBase.Query(`
SELECT COUNT(*) FROM users;`)
	if err != nil {
		log.Println(err.Error())
	}
	return a.readRows(rows)
}

func (a *Admin) readRows(rows *sql.Rows) int {
	defer rows.Close()

	var count int

	for rows.Next() {
		if err := rows.Scan(&count); err != nil {
			a.msgs.SendNotificationToDeveloper(errors.Wrap(err, "failed to scan row").Error(), false)
		}
	}

	return count
}

func (a *Admin) countAllUsers() int {
	var sum int
	for _, handler := range model.Bots {
		rows, err := handler.DataBase.Query(`
SELECT COUNT(*) FROM users;`)
		if err != nil {
			log.Println(err.Error())
			continue
		}
		sum += a.readRows(rows)
	}
	return sum
}

func (a *Admin) countReferrals(botLang string, amountUsers int) string {
	var refText string
	rows, err := model.Bots[botLang].DataBase.Query("SELECT SUM(referral_count) FROM users;")
	if err != nil {
		log.Println(err.Error())
	}

	count := a.readRows(rows)
	refText = strconv.Itoa(count) + " (" + strconv.Itoa(int(float32(count)*100.0/float32(amountUsers))) + "%)"
	return refText
}

func countBlockedUsers(botLang string) int {
	//var count int
	//for _, value := range assets.AdminSettings.BlockedUsers {
	//	count += value
	//}
	//return count
	return model.AdminSettings.GlobalParameters[botLang].BlockedUsers
}

func (a *Admin) countSubscribers(botLang string) int {
	rows, err := model.Bots[botLang].DataBase.Query(`
SELECT COUNT(DISTINCT id) FROM subs;`)
	if err != nil {
		log.Println(err.Error())
	}

	return a.readRows(rows)
}
