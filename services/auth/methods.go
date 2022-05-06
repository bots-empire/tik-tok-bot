package auth

import (
	"database/sql"
	"github.com/Stepan1328/tik-tok-bot/model"
	"github.com/bots-empire/base-bot/msgs"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"strconv"
	"strings"
)

func (a *Auth) WithdrawMoneyFromBalance(s *model.Situation, amount string) error {
	amount = strings.Replace(amount, " ", "", -1)
	amountInt, err := strconv.Atoi(amount)
	if err != nil {
		msg := tgbotapi.NewMessage(s.User.ID, a.bot.LangText(s.User.Language, "incorrect_amount"))
		return a.msgs.SendMsgToUser(msg)
	}

	if amountInt < model.AdminSettings.GetParams(s.BotLang).MinWithdrawalAmount {
		return a.minAmountNotReached(s.User, s.BotLang)
	}

	if s.User.Balance < amountInt {
		msg := tgbotapi.NewMessage(s.User.ID, a.bot.LangText(s.User.Language, "lack_of_funds"))
		return a.msgs.SendMsgToUser(msg)
	}

	return a.sendInvitationToSubs(s, amount)
}

func (a *Auth) minAmountNotReached(u *model.User, botLang string) error {
	text := a.bot.LangText(u.Language, "minimum_amount_not_reached",
		model.AdminSettings.GetParams(botLang).MinWithdrawalAmount)

	return a.msgs.NewParseMessage(u.ID, text)
}

func (a *Auth) sendInvitationToSubs(s *model.Situation, amount string) error {
	text := a.bot.LangText(s.User.Language, "withdrawal_not_subs_text")

	msg := tgbotapi.NewMessage(s.User.ID, text)
	msg.ReplyMarkup = msgs.NewIlMarkUp(
		msgs.NewIlRow(msgs.NewIlURLButton("advertising_button", model.AdminSettings.GetAdvertUrl(s.BotLang, model.MainAdvert))),
		msgs.NewIlRow(msgs.NewIlDataButton("im_subscribe_button", "/withdrawal_money?"+amount)),
	).Build(a.bot.AdminLibrary[s.BotLang])

	return a.msgs.SendMsgToUser(msg)
}

func (a *Auth) CheckSubscribeToWithdrawal(s *model.Situation, amount int) bool {
	if s.User.Balance < amount {
		return false
	}

	if !a.CheckSubscribe(s, "withdrawal") {
		_ = a.sendInvitationToSubs(s, strconv.Itoa(amount))
		return false
	}

	s.User.Balance -= amount
	dataBase := a.bot.GetDataBase()
	rows, err := dataBase.Query(`
UPDATE users 
	SET balance = ?
WHERE id = ?;`,
		s.User.Balance,
		s.User.ID)
	if err != nil {
		return false
	}
	_ = rows.Close()

	msg := tgbotapi.NewMessage(s.User.ID, a.bot.LangText(s.User.Language, "successfully_withdrawn"))
	_ = a.msgs.SendMsgToUser(msg)
	return true
}

func (a *Auth) GetABonus(s *model.Situation) error {
	if !a.CheckSubscribe(s, "get_bonus") {
		text := a.bot.LangText(s.User.Language, "user_dont_subscribe")
		return a.msgs.SendSimpleMsg(s.User.ID, text)
	}

	if s.User.TakeBonus {
		text := a.bot.LangText(s.User.Language, "bonus_already_have")
		return a.msgs.SendSimpleMsg(s.User.ID, text)
	}

	s.User.Balance += model.AdminSettings.GetParams(s.BotLang).BonusAmount
	dataBase := a.bot.GetDataBase()
	rows, err := dataBase.Query(`
UPDATE users 
	SET balance = ?, 
	    take_bonus = ? 
WHERE id = ?;`,
		s.User.Balance,
		true,
		s.User.ID)
	if err != nil {
		return err
	}
	_ = rows.Close()

	text := a.bot.LangText(s.User.Language, "bonus_have_received")
	return a.msgs.SendSimpleMsg(s.User.ID, text)
}

func (a *Auth) CheckSubscribe(s *model.Situation, source string) bool {
	model.CheckSubscribe.WithLabelValues(
		a.bot.BotLink,
		s.BotLang,
		model.AdminSettings.GetAdvertUrl(s.BotLang, model.MainAdvert),
		source,
	).Inc()

	member, err := model.Bots[s.BotLang].Bot.GetChatMember(tgbotapi.GetChatMemberConfig{
		ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
			ChatID: model.AdminSettings.GetAdvertChannelID(s.BotLang, model.MainAdvert),
			UserID: s.User.ID,
		},
	})

	if err == nil {
		if err := a.addMemberToSubsBase(s); err != nil {
			return false
		}
		return checkMemberStatus(member)
	}
	return false
}

func checkMemberStatus(member tgbotapi.ChatMember) bool {
	if member.IsAdministrator() {
		return true
	}
	if member.IsCreator() {
		return true
	}
	if member.Status == "member" {
		return true
	}
	return false
}

func (a *Auth) addMemberToSubsBase(s *model.Situation) error {
	dataBase := a.bot.GetDataBase()
	rows, err := dataBase.Query(`
SELECT * FROM subs 
	WHERE id = ?;`,
		s.User.ID)
	if err != nil {
		return err
	}

	user, err := a.readUser(rows)
	if err != nil {
		return err
	}

	if user.ID != 0 {
		return nil
	}
	rows, err = dataBase.Query(`
INSERT INTO subs VALUES(?);`,
		s.User.ID)
	if err != nil {
		return err
	}
	_ = rows.Close()
	return nil
}

func (a *Auth) readUser(rows *sql.Rows) (*model.User, error) {
	defer rows.Close()

	var users []*model.User

	for rows.Next() {
		var id int64

		if err := rows.Scan(&id); err != nil {
			return nil, model.ErrScanSqlRow
		}

		users = append(users, &model.User{
			ID: id,
		})
	}
	if len(users) == 0 {
		users = append(users, &model.User{
			ID: 0,
		})
	}
	return users[0], nil
}
