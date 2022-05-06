package model

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
)

const (
	adminPath      = "assets/admin"
	jsonFormatName = ".json"
	GlobalMailing  = 4
	MainAdvert     = 5
	oneSatoshi     = 0.00000001
)

type Admin struct {
	AdminID          map[int64]*AdminUser         `json:"admin_id"`
	GlobalParameters map[string]*GlobalParameters `json:"global_parameters"`
}

type GlobalParameters struct {
	Parameters        *Params        `json:"parameters"`
	AdvertisingChan   *AdvertChannel `json:"advertising_chan"`
	BlockedUsers      int            `json:"blocked_users"`
	AdvertisingText   map[int]string `json:"advertising_text"`
	AdvertisingPhoto  map[int]string
	AdvertisingVideo  map[int]string
	AdvertisingChoice map[int]string
}

type AdminUser struct {
	Language           string `json:"language"`
	FirstName          string `json:"first_name"`
	SpecialPossibility bool   `json:"special_possibility"`
}

type Params struct {
	BonusAmount                 int `json:"bonus_amount"`
	MinWithdrawalAmount         int `json:"min_withdrawal_amount"`
	ReferralAmount              int `json:"referral_amount"`
	ReferralFromBotVideoAmount  int
	ReferralFromSelfVideoAmount int

	TikTokVideo string
	TikTokText  string

	ButtonUnderAdvert bool

	Currency string `json:"currency"`
}

type AdvertChannel struct {
	Url       map[int]string `json:"url"`
	ChannelID map[int]int64  `json:"channel_id"`
}

var AdminSettings *Admin

func UploadAdminSettings() {
	var settings *Admin
	data, err := os.ReadFile(adminPath + jsonFormatName)
	if err != nil {
		fmt.Println(err)
	}

	err = json.Unmarshal(data, &settings)
	if err != nil {
		fmt.Println(err)
	}

	for lang, globalBot := range Bots {
		validateSettings(settings, lang)
		for _, lang = range globalBot.LanguageInBot {
			validateSettings(settings, lang)
		}
	}

	AdminSettings = settings
	SaveAdminSettings()
}

func validateSettings(settings *Admin, lang string) {
	if settings.GlobalParameters == nil {
		settings.GlobalParameters = make(map[string]*GlobalParameters)
	}

	if settings.GlobalParameters[lang] == nil {
		settings.GlobalParameters[lang] = &GlobalParameters{}
	}

	if settings.GlobalParameters[lang].Parameters == nil {
		settings.GlobalParameters[lang].Parameters = &Params{}
	}

	if settings.GlobalParameters[lang].AdvertisingChan == nil {
		settings.GlobalParameters[lang].AdvertisingChan = &AdvertChannel{
			Url: map[int]string{
				0: "https://google.com",
				1: "https://google.com",
				2: "https://google.com",
				5: "https://google.com"},
			ChannelID: make(map[int]int64),
		}
	}

	if settings.GlobalParameters[lang].AdvertisingChoice == nil {
		settings.GlobalParameters[lang].AdvertisingChoice = make(map[int]string)
	}

	if settings.GlobalParameters[lang].AdvertisingText == nil {
		settings.GlobalParameters[lang].AdvertisingText = make(map[int]string)
	}

	if settings.GlobalParameters[lang].AdvertisingPhoto == nil {
		settings.GlobalParameters[lang].AdvertisingPhoto = make(map[int]string)
	}
	if settings.GlobalParameters[lang].AdvertisingVideo == nil {
		settings.GlobalParameters[lang].AdvertisingVideo = make(map[int]string)
	}
}

func SaveAdminSettings() {
	data, err := json.MarshalIndent(AdminSettings, "", "  ")
	if err != nil {
		panic(err)
	}

	if err = os.WriteFile(adminPath+jsonFormatName, data, 0600); err != nil {
		panic(err)
	}
}

func (a *Admin) GetCurrency(lang string) string {
	return a.GlobalParameters[lang].Parameters.Currency
}

func (a *Admin) GetAdvertText(lang string, channel int) string {
	return a.GlobalParameters[lang].AdvertisingText[channel]
}

func (a *Admin) UpdateAdvertUrl(lang string, channel int, value string) {
	a.GlobalParameters[lang].AdvertisingChan.Url[channel] = value
}

func (a *Admin) UpdateAdvertChannelID(lang string, value int64, channel int) {
	a.GlobalParameters[lang].AdvertisingChan.ChannelID[channel] = value
}

func (a *Admin) UpdateAdvertText(lang string, value string, channel int) {
	a.GlobalParameters[lang].AdvertisingText[channel] = value
}

func (a *Admin) UpdateAdvertPhoto(lang string, channel int, value string) {
	a.GlobalParameters[lang].AdvertisingPhoto[channel] = value
}

func (a *Admin) UpdateAdvertVideo(lang string, channel int, value string) {
	a.GlobalParameters[lang].AdvertisingVideo[channel] = value
}

func (a *Admin) UpdateAdvertChoice(lang string, channel int, value string) {
	a.GlobalParameters[lang].AdvertisingChoice[channel] = value
}

func (a *Admin) UpdateTikTokText(lang string, value string) {
	a.GlobalParameters[lang].Parameters.TikTokText = value
}

func (a *Admin) UpdateTikTokVideo(lang string, value string) {
	a.GlobalParameters[lang].Parameters.TikTokVideo = value
}

func (a *Admin) GetAdvertUrl(lang string, channel int) string {
	return a.GlobalParameters[lang].AdvertisingChan.Url[channel]
}

func (a *Admin) GetAdvertChannelID(lang string, channel int) int64 {
	return a.GlobalParameters[lang].AdvertisingChan.ChannelID[channel]
}

func (a *Admin) UpdateAdvertChan(lang string, newChan *AdvertChannel) {
	a.GlobalParameters[lang].AdvertisingChan = newChan
}

func (a *Admin) UpdateBlockedUsers(lang string, value int) {
	a.GlobalParameters[lang].BlockedUsers = value
}

func (a *Admin) GetParams(lang string) *Params {
	return a.GlobalParameters[lang].Parameters
}

// ----------------------------------------------------
//
// Update Statistic
//
// ----------------------------------------------------

type UpdateInfo struct {
	Mu      *sync.Mutex
	Counter int
	Day     int
}

var UpdateStatistic *UpdateInfo

func UploadUpdateStatistic() {
	info := &UpdateInfo{}
	info.Mu = new(sync.Mutex)
	strStatistic, err := Bots["it"].Rdb.Get("update_statistic").Result()
	if err != nil {
		UpdateStatistic = info
		return
	}

	data := strings.Split(strStatistic, "?")
	if len(data) != 2 {
		UpdateStatistic = info
		return
	}
	info.Counter, _ = strconv.Atoi(data[0])
	info.Day, _ = strconv.Atoi(data[1])
	UpdateStatistic = info
}

func SaveUpdateStatistic() {
	strStatistic := strconv.Itoa(UpdateStatistic.Counter) + "?" + strconv.Itoa(UpdateStatistic.Day)
	_, err := Bots["it"].Rdb.Set("update_statistic", strStatistic, 0).Result()
	if err != nil {
		log.Println(err)
	}
}
