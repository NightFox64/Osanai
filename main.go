package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	bot          *tgbotapi.BotAPI
	chatSettings = make(map[int64]struct {
		Counter    int
		TriggerNum int // Количество сообщений для триггера
	})
)

func main() {
	var err error
	bot, err = tgbotapi.NewBotAPI("7843550853:AAE1Ih5G1WuEnKDPSXRj3DaOLB6y8-mhBF8")
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	log.Printf("Бот запущен: %s", bot.Self.UserName)

	// Устанавливаем вебхук
	webhookURL := "https://Osanai.onrender.com/webhook"
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
		return
	}

	// Инициализация настроек чата, если их еще нет
	if _, exists := chatSettings[update.Message.Chat.ID]; !exists {
		chatSettings[update.Message.Chat.ID] = struct {
			Counter    int
			TriggerNum int
		}{
			Counter:    0,
			TriggerNum: 10, // Значение по умолчанию
		}
	}

	// Обработка команд
	if update.Message.IsCommand() {
		handleCommand(update.Message)
		return
	}

	// Обработка обычных сообщений
	handleRegularMessage(update.Message)
}

func handleCommand(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	command := msg.Command()
	args := msg.CommandArguments()

	switch command {
	case "settrigger":
		// Команда для изменения количества сообщений
		newTrigger, err := strconv.Atoi(args)
		if err != nil || newTrigger <= 0 {
			sendMessage(chatID, "Используйте: /settrigger <число> (например: /settrigger 5)")
			return
		}

		// Обновляем настройки
		settings := chatSettings[chatID]
		settings.TriggerNum = newTrigger
		chatSettings[chatID] = settings

		sendMessage(chatID, fmt.Sprintf("Теперь гифка будет отправляться каждые %d сообщений", newTrigger))
	default:
		sendMessage(chatID, "Доступные команды:\n/settrigger <число> - установить количество сообщений для гифки")
	}
}

func handleRegularMessage(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	settings := chatSettings[chatID]
	settings.Counter++
	chatSettings[chatID] = settings

	// Проверяем, нужно ли отправлять гифку
	if settings.Counter >= settings.TriggerNum {
		sendGif(chatID)
		settings.Counter = 0 // Сбрасываем счетчик
		chatSettings[chatID] = settings
	}
}

func sendGif(chatID int64) {
	msg := tgbotapi.NewAnimation(chatID, tgbotapi.FileID("CgACAgIAAxkBAAMDaD2NxM7H1-jBWRjBHYxlIxNOWIkAAr5zAAKrXvBJaM6laThye-g2BA"))
	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("Ошибка отправки гифки: %v", err)
	}
}

func sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
	}
}
