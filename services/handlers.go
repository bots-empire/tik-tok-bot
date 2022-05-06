package services

import (
	"fmt"
	"github.com/bots-empire/base-bot/msgs"
	"strings"
	"time"

	"github.com/Stepan1328/tik-tok-bot/db"
	"github.com/Stepan1328/tik-tok-bot/log"
	"github.com/Stepan1328/tik-tok-bot/model"
	"github.com/Stepan1328/tik-tok-bot/services/administrator"
	"github.com/Stepan1328/tik-tok-bot/utils"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	updateCounterHeader = "Today Update's counter: %d"
	updatePrintHeader   = "update number: %d    // miner-bot-update:  %s %s"
	extraneousUpdate    = "extraneous update"
	godUserID           = 872383555

	oneSatoshi = 0.00000001
)

type MessagesHandlers struct {
	Handlers map[string]model.Handler
}

func (h *MessagesHandlers) GetHandler(command string) model.Handler {
	return h.Handlers[command]
}

func (h *MessagesHandlers) Init(userSrv *Users, adminSrv *administrator.Admin) {
	// Start command
	h.OnCommand("/select_language", userSrv.SelectLangCommand)
	h.OnCommand("/start", userSrv.StartCommand)
	h.OnCommand("/admin", adminSrv.AdminLoginCommand)

	// Main command
	h.OnCommand("/main_make_money", userSrv.MakeMoneyCommand)
	h.OnCommand("/main_profile", userSrv.SendProfileCommand)
	h.OnCommand("/main_money_for_a_friend", userSrv.MoneyForAFriendCommand)
	h.OnCommand("/main_more_money", userSrv.MoreMoneyCommand)
	h.OnCommand("/main_statistic", userSrv.MakeStatisticCommand)
	h.OnCommand("/make_money_yourself", userSrv.MakeMoneyYourselfCommand)
	h.OnCommand("/make_money_bot", userSrv.MakeMoneyBotCommand)

	// Spend money command
	h.OnCommand("/main_withdrawal_of_money", userSrv.SpendMoneyWithdrawalCommand)
	h.OnCommand("/paypal_method", userSrv.PaypalReqCommand)
	h.OnCommand("/credit_card_method", userSrv.CreditCardReqCommand)
	h.OnCommand("/withdrawal_method", userSrv.WithdrawalMethodCommand)
	h.OnCommand("/withdrawal_req_amount", userSrv.ReqWithdrawalAmountCommand)
	h.OnCommand("/withdrawal_exit", userSrv.WithdrawalAmountCommand)

	// Log out command
	h.OnCommand("/admin_log_out", userSrv.AdminLogOutCommand)
}

func (h *MessagesHandlers) OnCommand(command string, handler model.Handler) {
	h.Handlers[command] = handler
}

func (u *Users) ActionsWithUpdates(logger log.Logger, sortCentre *utils.Spreader) {
	for update := range u.bot.Chanel {
		localUpdate := update

		go u.checkUpdate(&localUpdate, logger, sortCentre)
	}
}

func (u *Users) checkUpdate(update *tgbotapi.Update, logger log.Logger, sortCentre *utils.Spreader) {
	defer u.panicCather(update)

	if update.Message == nil && update.CallbackQuery == nil {
		return
	}

	if update.Message != nil && update.Message.PinnedMessage != nil {
		return
	}

	u.printNewUpdate(update, logger)
	if update.Message != nil {
		var command string
		user, err := u.auth.CheckingTheUser(update.Message)
		if err == model.ErrNotSelectedLanguage {
			command = "/select_language"
		} else if err != nil {
			u.smthWentWrong(update.Message.Chat.ID, u.bot.BotLang)
			logger.Warn("err with check user: %s", err.Error())
			return
		}

		situation := createSituationFromMsg(u.bot.BotLang, update.Message, user)
		situation.Command = command

		u.checkMessage(situation, logger, sortCentre)
		return
	}

	if update.CallbackQuery != nil {
		if strings.Contains(update.CallbackQuery.Data, "/language") {
			err := u.auth.SetStartLanguage(update.CallbackQuery)
			if err != nil {
				u.smthWentWrong(update.CallbackQuery.Message.Chat.ID, u.bot.BotLang)
				logger.Warn("err with set start language: %s", err.Error())
			}
		}
		situation, err := u.createSituationFromCallback(u.bot.BotLang, update.CallbackQuery)
		if err != nil {
			u.smthWentWrong(update.CallbackQuery.Message.Chat.ID, u.bot.BotLang)
			logger.Warn("err with create situation from callback: %s", err.Error())
			return
		}

		u.checkCallbackQuery(situation, logger, sortCentre)
		return
	}
}

func (u *Users) printNewUpdate(update *tgbotapi.Update, logger log.Logger) {
	model.UpdateStatistic.Mu.Lock()
	defer model.UpdateStatistic.Mu.Unlock()

	if (time.Now().Unix())/86400 > int64(model.UpdateStatistic.Day) {
		u.sendTodayUpdateMsg()
	}

	model.UpdateStatistic.Counter++
	model.SaveUpdateStatistic()

	model.HandleUpdates.WithLabelValues(
		u.bot.BotLink,
		u.bot.BotLang,
	).Inc()

	if update.Message != nil {
		if update.Message.Text != "" {
			logger.Info(updatePrintHeader, model.UpdateStatistic.Counter, u.bot.BotLang, update.Message.Text)
			return
		}
	}

	if update.CallbackQuery != nil {
		if update.CallbackQuery.Data == "/make_money_click" {
			return
		}

		logger.Info(updatePrintHeader, model.UpdateStatistic.Counter, u.bot.BotLang, update.CallbackQuery.Data)
		return
	}

	logger.Info(updatePrintHeader, model.UpdateStatistic.Counter, u.bot.BotLang, extraneousUpdate)
}

func (u *Users) sendTodayUpdateMsg() {
	text := fmt.Sprintf(updateCounterHeader, model.UpdateStatistic.Counter)
	u.Msgs.SendNotificationToDeveloper(text, true)

	model.UpdateStatistic.Counter = 0
	model.UpdateStatistic.Day = int(time.Now().Unix()) / 86400
}

func createSituationFromMsg(botLang string, message *tgbotapi.Message, user *model.User) *model.Situation {
	return &model.Situation{
		Message: message,
		BotLang: botLang,
		User:    user,
		Params: &model.Parameters{
			Level: db.GetLevel(botLang, message.From.ID),
		},
	}
}

func (u *Users) createSituationFromCallback(botLang string, callbackQuery *tgbotapi.CallbackQuery) (*model.Situation, error) {
	user, err := u.auth.GetUser(callbackQuery.From.ID)
	if err != nil {
		return &model.Situation{}, err
	}

	return &model.Situation{
		CallbackQuery: callbackQuery,
		BotLang:       botLang,
		User:          user,
		Command:       strings.Split(callbackQuery.Data, "?")[0],
		Params: &model.Parameters{
			Level: db.GetLevel(botLang, callbackQuery.From.ID),
		},
	}, nil
}

func (u *Users) checkMessage(situation *model.Situation, logger log.Logger, sortCentre *utils.Spreader) {

	if model.Bots[situation.BotLang].MaintenanceMode {
		if situation.User.ID != godUserID {
			msg := tgbotapi.NewMessage(situation.User.ID, "The bot is under maintenance, please try again later")
			_ = u.Msgs.SendMsgToUser(msg)
			return
		}
	}
	if situation.Command == "" {
		situation.Command, situation.Err = u.bot.GetCommandFromText(
			situation.Message, situation.User.Language, situation.User.ID)
	}

	if situation.Err == nil {
		handler := model.Bots[situation.BotLang].MessageHandler.
			GetHandler(situation.Command)

		if handler != nil {
			sortCentre.ServeHandler(handler, situation, func(err error) {
				text := fmt.Sprintf("%s // %s // error with serve user msg command: %s",
					u.bot.BotLang,
					u.bot.BotLink,
					err.Error(),
				)
				u.Msgs.SendNotificationToDeveloper(text, false)

				logger.Warn(text)
				u.smthWentWrong(situation.Message.Chat.ID, situation.User.Language)
			})
			return
		}
	}

	situation.Command = strings.Split(situation.Params.Level, "?")[0]

	handler := model.Bots[situation.BotLang].MessageHandler.
		GetHandler(situation.Command)

	if handler != nil {
		sortCentre.ServeHandler(handler, situation, func(err error) {
			text := fmt.Sprintf("%s // %s // error with serve user level command: %s",
				u.bot.BotLang,
				u.bot.BotLink,
				err.Error(),
			)
			u.Msgs.SendNotificationToDeveloper(text, false)

			logger.Warn(text)
			u.smthWentWrong(situation.Message.Chat.ID, situation.User.Language)
		})
		return
	}

	if err := u.admin.CheckAdminMessage(situation); err == nil {
		return
	}

	u.smthWentWrong(situation.Message.Chat.ID, situation.User.Language)
	if situation.Err != nil {
		logger.Info(situation.Err.Error())
	}
}

func (u *Users) smthWentWrong(chatID int64, lang string) {
	msg := tgbotapi.NewMessage(chatID, u.bot.LangText(lang, "user_level_not_defined"))
	_ = u.Msgs.SendMsgToUser(msg)
}

//func emptyLevel(botLang string, message *tgbotapi.Message, lang string) {
//	msg := tgbotapi.NewMessage(message.Chat.ID, model.LangText(lang, "user_level_not_defined"))
//	_ = msgs.SendMsgToUser(botLang, msg)
//}

func (u *Users) StartCommand(s *model.Situation) error {
	if s.Message != nil {
		if strings.Contains(s.Message.Text, "new_admin") {
			s.Command = s.Message.Text
			return u.admin.CheckNewAdmin(s)
		}
	}

	text := u.bot.LangText(s.User.Language, "main_select_menu")
	db.RdbSetUser(s.BotLang, s.User.ID, "main")

	msg := tgbotapi.NewMessage(s.User.ID, text)
	msg.ReplyMarkup = createMainMenu().Build(u.bot.Language[s.User.Language])

	return u.Msgs.SendMsgToUser(msg)
}

func createMainMenu() msgs.MarkUp {
	return msgs.NewMarkUp(
		msgs.NewRow(msgs.NewDataButton("main_make_money")),
		msgs.NewRow(msgs.NewDataButton("main_money_for_a_friend"),
			msgs.NewDataButton("main_profile")),
		msgs.NewRow(msgs.NewDataButton("main_withdrawal_of_money"),
			msgs.NewDataButton("main_more_money")),
		msgs.NewRow(msgs.NewDataButton("main_statistic")),
	)
}

func (u *Users) MakeMoneyCommand(s *model.Situation) error {
	db.RdbSetUser(s.BotLang, s.User.ID, "main")
	text := u.bot.LangText(s.User.Language, "main_select_menu")

	markup := msgs.NewMarkUp(
		msgs.NewRow(msgs.NewDataButton("make_money_yourself")),
		msgs.NewRow(msgs.NewDataButton("make_money_bot")),
		msgs.NewRow(msgs.NewDataButton("back_to_main_menu_button")),
	).Build(u.bot.Language[s.User.Language])

	return u.Msgs.NewParseMarkUpMessage(s.User.ID, &markup, text)
}

func (u *Users) SendProfileCommand(s *model.Situation) error {
	db.RdbSetUser(s.BotLang, s.User.ID, "main")

	text := u.bot.LangText(s.User.Language, "profile_text",
		s.Message.From.FirstName,
		s.Message.From.UserName,
		s.User.Balance,
		s.User.ReferralCount)

	if len(u.bot.LanguageInBot) > 1 {
		ReplyMarkup := u.createLangMenu(u.bot.LanguageInBot)
		return u.Msgs.NewParseMarkUpMessage(s.User.ID, &ReplyMarkup, text)
	}

	return u.Msgs.NewParseMessage(s.User.ID, text)
}

func (u *Users) MoneyForAFriendCommand(s *model.Situation) error {
	db.RdbSetUser(s.BotLang, s.User.ID, "main")

	link, err := model.EncodeLink(u.bot.GetDataBase(), u.bot.BotLink, &model.ReferralLinkInfo{
		ReferralID: s.User.ID,
		Source:     "bot",
	})

	fmt.Println(link)

	if err != nil {
		return err
	}

	text := u.bot.LangText(s.User.Language, "referral_text",
		link,
		model.AdminSettings.GetParams(s.BotLang).ReferralAmount,
		s.User.ReferralCount)

	return u.Msgs.NewParseMessage(s.User.ID, text)
}

func (u *Users) SelectLangCommand(s *model.Situation) error {
	var text string
	for _, lang := range u.bot.LanguageInBot {
		text += u.bot.LangText(lang, "select_lang_menu") + "\n"
	}
	db.RdbSetUser(s.BotLang, s.User.ID, "main")

	msg := tgbotapi.NewMessage(s.User.ID, text)
	msg.ReplyMarkup = u.createLangMenu(u.bot.LanguageInBot)

	return u.Msgs.SendMsgToUser(msg)
}

func (u *Users) createLangMenu(languages []string) tgbotapi.InlineKeyboardMarkup {
	var markup tgbotapi.InlineKeyboardMarkup

	for _, lang := range languages {
		markup.InlineKeyboard = append(markup.InlineKeyboard, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(u.bot.LangText(lang, "lang_button"), "/language?"+lang),
		})
	}

	return markup
}

func (u *Users) SpendMoneyWithdrawalCommand(s *model.Situation) error {
	db.RdbSetUser(s.BotLang, s.User.ID, "withdrawal")

	text := u.bot.LangText(s.User.Language, "select_payment")
	markUp := msgs.NewMarkUp(
		msgs.NewRow(msgs.NewDataButton("withdrawal_method_1"),
			msgs.NewDataButton("withdrawal_method_2")),
		msgs.NewRow(msgs.NewDataButton("withdrawal_method_3"),
			msgs.NewDataButton("withdrawal_method_4")),
		msgs.NewRow(msgs.NewDataButton("withdrawal_method_5")),
		msgs.NewRow(msgs.NewDataButton("back_to_main_menu_button")),
	).Build(u.bot.Language[s.User.Language])

	return u.Msgs.NewParseMarkUpMessage(s.User.ID, &markUp, text)
}

func (u *Users) PaypalReqCommand(s *model.Situation) error {
	db.RdbSetUser(s.BotLang, s.User.ID, "/withdrawal_req_amount")

	msg := tgbotapi.NewMessage(s.User.ID, u.bot.LangText(s.User.Language, "paypal_method"))
	msg.ReplyMarkup = msgs.NewMarkUp(
		msgs.NewRow(msgs.NewDataButton("withdraw_cancel")),
	).Build(u.bot.Language[s.User.Language])

	return u.Msgs.SendMsgToUser(msg)
}

func (u *Users) CreditCardReqCommand(s *model.Situation) error {
	db.RdbSetUser(s.BotLang, s.User.ID, "/withdrawal_req_amount")

	msg := tgbotapi.NewMessage(s.User.ID, u.bot.LangText(s.User.Language, "credit_card_number"))
	msg.ReplyMarkup = msgs.NewMarkUp(
		msgs.NewRow(msgs.NewDataButton("withdraw_cancel")),
	).Build(u.bot.Language[s.User.Language])

	return u.Msgs.SendMsgToUser(msg)
}

func (u *Users) WithdrawalMethodCommand(s *model.Situation) error {
	db.RdbSetUser(s.BotLang, s.User.ID, "/withdrawal_req_amount")

	msg := tgbotapi.NewMessage(s.User.ID, u.bot.LangText(s.User.Language, "req_withdrawal_amount"))
	msg.ReplyMarkup = msgs.NewMarkUp(
		msgs.NewRow(msgs.NewDataButton("withdraw_cancel")),
	).Build(u.bot.Language[s.User.Language])

	return u.Msgs.SendMsgToUser(msg)
}

func (u *Users) ReqWithdrawalAmountCommand(s *model.Situation) error {
	db.RdbSetUser(s.BotLang, s.User.ID, "/withdrawal_exit")

	msg := tgbotapi.NewMessage(s.User.ID, u.bot.LangText(s.User.Language, "req_withdrawal_amount"))

	return u.Msgs.SendMsgToUser(msg)
}

func (u *Users) WithdrawalAmountCommand(s *model.Situation) error {
	return u.auth.WithdrawMoneyFromBalance(s, s.Message.Text)
}

func (u *Users) AdminLogOutCommand(s *model.Situation) error {
	db.DeleteOldAdminMsg(s.BotLang, s.User.ID)

	text := u.bot.AdminText(model.AdminLang(s.User.ID), "admin_log_out")
	msg := tgbotapi.NewMessage(s.User.ID, text)

	if err := u.Msgs.SendMsgToUser(msg); err != nil {
		return err
	}

	return u.StartCommand(s)
}

func (u *Users) MakeStatisticCommand(s *model.Situation) error {
	currentTime := time.Now()

	users := currentTime.Unix() % 100000000 / 6000

	text := u.bot.LangText(s.User.Language, "statistic_to_user",
		users,
	)

	return u.Msgs.NewParseMessage(s.Message.Chat.ID, text)
}

func (u *Users) MakeMoneyYourselfCommand(s *model.Situation) error {
	link, err := model.EncodeLink(u.bot.GetDataBase(), u.bot.BotLink, &model.ReferralLinkInfo{
		ReferralID: s.User.ID,
		Source:     "tiktok2",
	})
	if err != nil {
		return err
	}

	text := u.bot.LangText(s.BotLang, "make_money_yourself_text",
		model.AdminSettings.GetParams(s.BotLang).TikTokText,
		link,
		model.AdminSettings.GetParams(s.BotLang).ReferralFromSelfVideoAmount,
	)
	return u.Msgs.NewParseMessage(s.Message.Chat.ID, text)
}

func (u *Users) MakeMoneyBotCommand(s *model.Situation) error {
	link, err := model.EncodeLink(u.bot.GetDataBase(), u.bot.BotLink, &model.ReferralLinkInfo{
		ReferralID: s.User.ID,
		Source:     "tiktok1",
	})
	if err != nil {
		return err
	}

	video := tgbotapi.FileID(model.AdminSettings.GlobalParameters[s.BotLang].Parameters.TikTokVideo)

	text := u.bot.LangText(s.BotLang, "make_money_bot_text",
		model.AdminSettings.GetParams(s.BotLang).TikTokText,
		link,
		model.AdminSettings.GetParams(s.BotLang).ReferralFromBotVideoAmount,
	)

	if video == "" {
		return u.Msgs.SendSimpleMsg(s.User.ID, text)
	}

	return u.Msgs.NewParseMarkUpPhotoMessage(s.User.ID, nil, text, video)
}

func (u *Users) MoreMoneyCommand(s *model.Situation) error {
	model.MoreMoneyButtonClick.WithLabelValues(
		u.bot.BotLink,
		s.BotLang,
	).Inc()

	db.RdbSetUser(s.BotLang, s.User.ID, "main")
	text := u.bot.LangText(s.User.Language, "more_money_text",
		model.AdminSettings.GetParams(s.BotLang).BonusAmount,
		model.AdminSettings.GetParams(s.BotLang).BonusAmount)

	markup := msgs.NewIlMarkUp(
		msgs.NewIlRow(msgs.NewIlURLButton("advertising_button", model.AdminSettings.GetAdvertUrl(s.BotLang, model.MainAdvert))),
		msgs.NewIlRow(msgs.NewIlDataButton("get_bonus_button", "/send_bonus_to_user")),
	).Build(u.bot.Language[s.User.Language])

	return u.Msgs.NewParseMarkUpMessage(s.User.ID, &markup, text)
}

func (u *Users) simpleAdminMsg(s *model.Situation, key string) error {
	text := u.bot.AdminText(s.User.Language, key)
	msg := tgbotapi.NewMessage(s.User.ID, text)

	return u.Msgs.SendMsgToUser(msg)
}
