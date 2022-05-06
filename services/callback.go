package services

import (
	"fmt"
	"github.com/bots-empire/base-bot/msgs"
	"strconv"
	"strings"

	"github.com/Stepan1328/tik-tok-bot/db"
	"github.com/Stepan1328/tik-tok-bot/log"
	"github.com/Stepan1328/tik-tok-bot/model"
	"github.com/Stepan1328/tik-tok-bot/utils"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type CallBackHandlers struct {
	Handlers map[string]model.Handler
}

func (h *CallBackHandlers) GetHandler(command string) model.Handler {
	return h.Handlers[command]
}

func (h *CallBackHandlers) Init(userSrv *Users) {
	// Start commands
	h.OnCommand("/language", userSrv.LanguageCommand)

	// Money commands
	h.OnCommand("/send_bonus_to_user", userSrv.GetBonusCommand)
	h.OnCommand("/withdrawal_money", userSrv.RecheckSubscribeCommand)
	h.OnCommand("/promotion_case", userSrv.PromotionCaseCommand)
}

func (h *CallBackHandlers) OnCommand(command string, handler model.Handler) {
	h.Handlers[command] = handler
}

func (u *Users) checkCallbackQuery(s *model.Situation, logger log.Logger, sortCentre *utils.Spreader) {
	if strings.Contains(s.Params.Level, "admin") {
		if err := u.admin.CheckAdminCallback(s); err != nil {
			text := fmt.Sprintf("%s // %s // error with serve admin callback command: %s",
				u.bot.BotLang,
				u.bot.BotLink,
				err.Error(),
			)
			u.Msgs.SendNotificationToDeveloper(text, false)

			logger.Warn(text)
		}
		return
	}

	handler := model.Bots[s.BotLang].CallbackHandler.
		GetHandler(s.Command)

	if handler != nil {
		sortCentre.ServeHandler(handler, s, func(err error) {
			text := fmt.Sprintf("%s // %s // error with serve user callback command: %s",
				u.bot.BotLang,
				u.bot.BotLink,
				err.Error(),
			)
			u.Msgs.SendNotificationToDeveloper(text, false)

			logger.Warn(text)
			u.smthWentWrong(s.CallbackQuery.Message.Chat.ID, s.User.Language)
		})

		return
	}

	text := fmt.Sprintf("%s // %s // get callback data='%s', but they didn't react in any way",
		u.bot.BotLang,
		u.bot.BotLink,
		s.CallbackQuery.Data,
	)
	u.Msgs.SendNotificationToDeveloper(text, false)

	logger.Warn(text)
}

func (u *Users) LanguageCommand(s *model.Situation) error {
	lang := strings.Split(s.CallbackQuery.Data, "?")[1]

	level := db.GetLevel(s.BotLang, s.User.ID)
	if strings.Contains(level, "admin") {
		return nil
	}

	s.User.Language = lang

	return u.StartCommand(s)
}

func (u *Users) GetBonusCommand(s *model.Situation) error {
	return u.auth.GetABonus(s)
}

func (u *Users) RecheckSubscribeCommand(s *model.Situation) error {
	amount := strings.Split(s.CallbackQuery.Data, "?")[1]
	s.Message = &tgbotapi.Message{
		Text: amount,
	}
	if err := u.Msgs.SendAnswerCallback(s.CallbackQuery, u.bot.LangText(s.User.Language, "invitation_to_subscribe")); err != nil {
		return err
	}
	amountInt, _ := strconv.Atoi(amount)

	if u.auth.CheckSubscribeToWithdrawal(s, amountInt) {
		db.RdbSetUser(s.BotLang, s.User.ID, "main")

		return u.StartCommand(s)
	}
	return nil
}

func (u *Users) PromotionCaseCommand(s *model.Situation) error {
	cost, err := strconv.Atoi(strings.Split(s.CallbackQuery.Data, "?")[1])
	if err != nil {
		return err
	}

	if s.User.Balance < cost {
		lowBalanceText := u.bot.LangText(s.User.Language, "not_enough_money")
		return u.Msgs.SendAnswerCallback(s.CallbackQuery, lowBalanceText)
	}

	db.RdbSetUser(s.BotLang, s.User.ID, s.CallbackQuery.Data)
	msg := tgbotapi.NewMessage(s.User.ID, u.bot.LangText(s.User.Language, "invitation_to_send_link_text"))
	msg.ReplyMarkup = msgs.NewMarkUp(
		msgs.NewRow(msgs.NewDataButton("withdraw_cancel")),
	).Build(u.bot.Language[s.User.Language])

	callBackText := u.bot.LangText(s.User.Language, "invitation_to_send_link")
	if err := u.Msgs.SendAnswerCallback(s.CallbackQuery, callBackText); err != nil {
		return err
	}

	return u.Msgs.SendMsgToUser(msg)
}
