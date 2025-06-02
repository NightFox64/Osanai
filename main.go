package main

import (
	"log"
	"net/http"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	bot          *tgbotapi.BotAPI
	chatCounters = make(map[int64]int)
)

func main() {
	var err error
	bot, err = tgbotapi.NewBotAPI(os.Getenv("7843550853:AAE1Ih5G1WuEnKDPSXRj3DaOLB6y8-mhBF8"))
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	log.Printf("Бот запущен: %s", bot.Self.UserName)

	// Устанавливаем вебхук
	webhookURL := "https://dashboard.render.com/web/srv-d0upb4k9c44c73bl5p1g/webhook"
	wh, err := tgbotapi.NewWebhook(webhookURL)
	if err != nil {
		log.Fatalf("Ошибка создания webhook: %v", err)
	}

	_, err = bot.Request(wh)
	if err != nil {
		log.Fatalf("Ошибка установки webhook: %v", err)
	}

	// Получаем канал обновлений
	updates := bot.ListenForWebhook("/webhook")

	// Запускаем сервер
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	go func() {
		log.Fatal(http.ListenAndServe(":"+port, nil))
	}()
	log.Printf("Сервер запущен на порту %s", port)

	// Обрабатываем обновления
	for update := range updates {
		handleUpdate(update)
	}
}

func handleUpdate(update tgbotapi.Update) {
	if update.Message == nil {
		// Проверяем наличие анимации в сообщении
		if update.Message != nil && update.Message.Animation != nil {
			log.Printf("FileID: %s", update.Message.Animation.FileID)
		}
		return
	}

	chatID := update.Message.Chat.ID
	chatCounters[chatID]++

	if chatCounters[chatID]%10 == 0 {
		sendGif(chatID)
	}
}

func sendGif(chatID int64) {
	msg := tgbotapi.NewAnimation(chatID, tgbotapi.FileID("CgACAgIAAxkBAAMDaD2NxM7H1-jBWRjBHYxlIxNOWIkAAr5zAAKrXvBJaM6laThye-g2BA"))
	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("Ошибка отправки гифки: %v", err)
	}
}
