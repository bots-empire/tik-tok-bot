package administrator

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Stepan1328/tik-tok-bot/db"
	"github.com/Stepan1328/tik-tok-bot/model"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type AdminMessagesHandlers struct {
	Handlers map[string]model.Handler
}

func (h *AdminMessagesHandlers) GetHandler(command string) model.Handler {
	return h.Handlers[command]
}

func (h *AdminMessagesHandlers) Init(adminSrv *Admin) {
	h.OnCommand("/make_money", adminSrv.UpdateParameterCommand)
	h.OnCommand("/change_text_url", adminSrv.SetNewTextUrlCommand)
	h.OnCommand("/advertisement_setting", adminSrv.AdvertisementSettingCommand)
	h.OnCommand("/get_new_source", adminSrv.GetNewSourceCommand)
	h.OnCommand("/set_video_tik_tok", adminSrv.SetVideoTikTokCommand)
	h.OnCommand("/set_text_tik_tok", adminSrv.SetTextTikTokCommand)
	h.OnCommand("/change_bot_referral_amount", adminSrv.SetBotReferralAmount)
	h.OnCommand("/change_self_referral_amount", adminSrv.SetSelfReferralAmount)
}

func (h *AdminMessagesHandlers) OnCommand(command string, handler model.Handler) {
	h.Handlers[command] = handler
}

func (a *Admin) UpdateParameterCommand(s *model.Situation) error {
	if strings.Contains(s.Params.Level, "make_money?") && s.Message.Text == "← Назад к ⚙️ Заработок" {
		if err := a.setAdminBackButton(s.User.ID, "operation_canceled"); err != nil {
			return err
		}
		db.DeleteOldAdminMsg(s.BotLang, s.User.ID)
		s.Command = "admin/make_money_setting"

		return a.MakeMoneySettingCommand(s)
	}

	partitions := strings.Split(s.Params.Level, "?")
	if len(partitions) < 2 {
		return fmt.Errorf("smth went wrong")
	}

	partition := partitions[1]

	if partition == currencyType {
		model.AdminSettings.GetParams(s.BotLang).Currency = s.Message.Text
	} else {
		err := a.setNewIntParameter(s, partition)
		if err != nil {
			return err
		}
	}

	model.SaveAdminSettings()
	err := a.setAdminBackButton(s.User.ID, "operation_completed")
	if err != nil {
		return nil
	}
	db.DeleteOldAdminMsg(s.BotLang, s.User.ID)
	s.Command = "admin/make_money_setting"

	return a.MakeMoneySettingCommand(s)
}

func (a *Admin) setNewIntParameter(s *model.Situation, partition string) error {
	lang := model.AdminLang(s.User.ID)

	newAmount, err := strconv.Atoi(s.Message.Text)
	if err != nil || newAmount <= 0 {
		text := a.bot.AdminText(lang, "incorrect_make_money_change_input")
		return a.msgs.NewParseMessage(s.User.ID, text)
	}

	switch partition {
	case bonusAmount:
		model.AdminSettings.GetParams(s.BotLang).BonusAmount = newAmount
	case minWithdrawalAmount:
		model.AdminSettings.GetParams(s.BotLang).MinWithdrawalAmount = newAmount
	case referralAmount:
		model.AdminSettings.GetParams(s.BotLang).ReferralAmount = newAmount
	}

	return nil
}

func (a *Admin) SetNewTextUrlCommand(s *model.Situation) error {
	capitation := strings.Split(s.Params.Level, "?")[1]
	channel, _ := strconv.Atoi(strings.Split(s.Params.Level, "?")[2])
	lang := model.AdminLang(s.User.ID)
	status := "operation_canceled"

	switch capitation {
	case "change_url":
		url, chatID := getUrlAndChatID(s.Message)
		if chatID == 0 {
			text := a.bot.AdminText(lang, "chat_id_not_update")
			return a.msgs.NewParseMessage(s.User.ID, text)
		}
		model.AdminSettings.UpdateAdvertChannelID(s.BotLang, chatID, channel)
		model.AdminSettings.UpdateAdvertUrl(s.BotLang, channel, url)
		//assets.AdminSettings.UpdateAdvertChan(s.BotLang, advertChan)
	case "change_text":
		model.AdminSettings.UpdateAdvertText(s.BotLang, s.Message.Text, channel)
	case "change_photo":
		if len(s.Message.Photo) == 0 {
			text := a.bot.AdminText(lang, "send_only_photo")
			return a.msgs.NewParseMessage(s.User.ID, text)
		}
		model.AdminSettings.UpdateAdvertPhoto(s.BotLang, channel, s.Message.Photo[0].FileID)
	case "change_video":
		if s.Message.Video == nil {
			text := a.bot.AdminText(lang, "send_only_video")
			return a.msgs.NewParseMessage(s.User.ID, text)
		}
		model.AdminSettings.UpdateAdvertVideo(s.BotLang, channel, s.Message.Video.FileID)
	}
	model.SaveAdminSettings()
	status = "operation_completed"

	if err := a.setAdminBackButton(s.User.ID, status); err != nil {
		return err
	}
	db.RdbSetUser(s.BotLang, s.User.ID, "admin")
	db.DeleteOldAdminMsg(s.BotLang, s.User.ID)

	callback := &tgbotapi.CallbackQuery{
		Data: "admin/change_advert_chan?" + strconv.Itoa(channel),
	}
	s.CallbackQuery = callback
	return a.AdvertisementChanMenuCommand(s)
}

func (a *Admin) AdvertisementSettingCommand(s *model.Situation) error {
	s.CallbackQuery = &tgbotapi.CallbackQuery{
		Data: "admin/change_text_url?",
	}
	s.Command = "admin/advertisement"
	return a.AdvertisementMenuCommand(s)
}

func getUrlAndChatID(message *tgbotapi.Message) (string, int64) {
	data := strings.Split(message.Text, "\n")
	if len(data) != 2 {
		return "", 0
	}

	chatId, err := strconv.Atoi(data[0])
	if err != nil {
		return "", 0
	}

	//advert := &assets.AdvertChannel{
	//	Url:       map[int]string{channel: data[1]},
	//	ChannelID: int64(chatId),
	//}

	//advert.Url[channel] = data[1]
	//advert.ChannelID = int64(chatId)

	return data[1], int64(chatId)
}

func (a *Admin) CheckAdminMessage(s *model.Situation) error {
	if !ContainsInAdmin(s.User.ID) {
		return a.notAdmin(s.User)
	}

	s.Command, s.Err = a.bot.GetCommandFromText(s.Message, s.User.Language, s.User.ID)
	if s.Err == nil {
		Handler := model.Bots[s.BotLang].AdminMessageHandler.
			GetHandler(s.Command)

		if Handler != nil {
			return Handler(s)
		}
	}

	s.Command = strings.TrimLeft(strings.Split(s.Params.Level, "?")[0], "admin")

	Handler := model.Bots[s.BotLang].AdminMessageHandler.
		GetHandler(s.Command)

	if Handler != nil {
		return Handler(s)
	}

	return model.ErrCommandNotConverted
}

//func (a *Admin) StartTestMailing1Command(s *model.Situation) error {
//	go db.StartTestMailing1(s.BotLang, s.User)
//	return msgs.NewParseMessage(s.BotLang, s.User.ID, "тестовая рассылка запущена")
//}

func (a *Admin) SetVideoTikTokCommand(s *model.Situation) error {
	lang := model.AdminLang(s.User.ID)

	if s.Message.Video == nil {
		text := a.bot.AdminText(lang, "send_only_video")
		return a.msgs.NewParseMessage(s.User.ID, text)
	}

	model.AdminSettings.UpdateTikTokVideo(s.BotLang, s.Message.Video.FileID)

	model.SaveAdminSettings()
	status := "operation_completed"

	if err := a.setAdminBackButton(s.User.ID, status); err != nil {
		return err
	}
	db.RdbSetUser(s.BotLang, s.User.ID, "admin")
	db.DeleteOldAdminMsg(s.BotLang, s.User.ID)

	s.Command = "admin/make_money_setting"
	s.Params.Level = "admin/set_video_tik_tok"
	return a.MakeMoneySettingCommand(s)
}

func (a *Admin) SetTextTikTokCommand(s *model.Situation) error {
	model.AdminSettings.UpdateTikTokText(s.BotLang, s.Message.Text)

	model.SaveAdminSettings()
	status := "operation_completed"

	if err := a.setAdminBackButton(s.User.ID, status); err != nil {
		return err
	}
	db.RdbSetUser(s.BotLang, s.User.ID, "admin")
	db.DeleteOldAdminMsg(s.BotLang, s.User.ID)

	s.Command = "admin/make_money_setting"
	s.Params.Level = "admin/set_text_tik_tok"
	return a.MakeMoneySettingCommand(s)
}

func (a *Admin) SetBotReferralAmount(s *model.Situation) error {
	lang := model.AdminLang(s.User.ID)

	newAmount, err := strconv.Atoi(s.Message.Text)
	if err != nil || newAmount <= 0 {
		text := a.bot.AdminText(lang, "incorrect_make_money_change_input")
		return a.msgs.NewParseMessage(s.User.ID, text)
	}

	model.AdminSettings.GlobalParameters[s.BotLang].Parameters.ReferralFromBotVideoAmount = newAmount

	model.SaveAdminSettings()
	err = a.setAdminBackButton(s.User.ID, "operation_completed")
	if err != nil {
		return nil
	}
	db.DeleteOldAdminMsg(s.BotLang, s.User.ID)
	s.Command = "admin/change_referral_amount_tik_tok"

	return a.ChangeReferralAmountTikTokCommand(s)
}

func (a *Admin) SetSelfReferralAmount(s *model.Situation) error {
	lang := model.AdminLang(s.User.ID)

	newAmount, err := strconv.Atoi(s.Message.Text)
	if err != nil || newAmount <= 0 {
		text := a.bot.AdminText(lang, "incorrect_make_money_change_input")
		return a.msgs.NewParseMessage(s.User.ID, text)
	}

	model.AdminSettings.GlobalParameters[s.BotLang].Parameters.ReferralFromSelfVideoAmount = newAmount

	model.SaveAdminSettings()
	err = a.setAdminBackButton(s.User.ID, "operation_completed")
	if err != nil {
		return nil
	}
	db.DeleteOldAdminMsg(s.BotLang, s.User.ID)
	s.Command = "admin/change_referral_amount_tik_tok"

	return a.ChangeReferralAmountTikTokCommand(s)
}
