package main

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	chatCounters = make(map[int64]int)
)

func main() {
	bot, err := tgbotapi.NewBotAPI("7843550853:AAE1Ih5G1WuEnKDPSXRj3DaOLB6y8-mhBF8")
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	log.Printf("Бот запущен: %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			log.Printf("FileID: %s", update.Message.Animation.FileID)
			continue
		}

		chatID := update.Message.Chat.ID
		chatCounters[chatID]++

		if chatCounters[chatID]%10 == 0 {
			sendGif(bot, chatID)
		}
	}
}

func sendGif(bot *tgbotapi.BotAPI, chatID int64) {
	msg := tgbotapi.NewAnimation(chatID, tgbotapi.FileID("CgACAgIAAxkBAAMDaD2NxM7H1-jBWRjBHYxlIxNOWIkAAr5zAAKrXvBJaM6laThye-g2BA"))

	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("Ошибка отправки гифки: %v", err)
	}
}
