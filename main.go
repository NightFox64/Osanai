package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/mattn/go-sqlite3"
)

var (
	bot *tgbotapi.BotAPI
	db  *sql.DB
)

// Инициализация базы данных
func initDB() error {
	var err error
	db, err = sql.Open("sqlite3", "./data/settings.db")
	if err != nil {
		return err
	}

	// Создаем таблицу если ее нет
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS chat_settings (
		chat_id INTEGER PRIMARY KEY,
		counter INTEGER DEFAULT 0,
		trigger_num INTEGER DEFAULT 10
	)`)
	return err
}

// Получаем настройки чата
func getChatSettings(chatID int64) (counter, triggerNum int, err error) {
	row := db.QueryRow("SELECT counter, trigger_num FROM chat_settings WHERE chat_id = ?", chatID)
	err = row.Scan(&counter, &triggerNum)
	if err == sql.ErrNoRows {
		// Если чата нет в БД - создаем запись с настройками по умолчанию
		_, err = db.Exec("INSERT INTO chat_settings (chat_id, counter, trigger_num) VALUES (?, 0, 10)", chatID)
		return 0, 10, nil
	}
	return
}

// Обновляем счетчик
func updateCounter(chatID int64, counter int) error {
	_, err := db.Exec("UPDATE chat_settings SET counter = ? WHERE chat_id = ?", counter, chatID)
	return err
}

// Устанавливаем новый триггер
func setTrigger(chatID int64, triggerNum int) error {
	_, err := db.Exec("UPDATE chat_settings SET trigger_num = ? WHERE chat_id = ?", triggerNum, chatID)
	return err
}

func main() {
	// 1. Инициализация БД
	if err := initDB(); err != nil {
		log.Fatalf("Ошибка инициализации БД: %v", err)
	}
	defer db.Close()

	// 2. Инициализация бота
	var err error
	bot, err = tgbotapi.NewBotAPI("7843550853:AAE1Ih5G1WuEnKDPSXRj3DaOLB6y8-mhBF8")
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	log.Printf("Бот запущен: %s", bot.Self.UserName)

	// 3. Настройка вебхука
	webhookURL := "https://Osanai.onrender.com/webhook"
	wh, _ := tgbotapi.NewWebhook(webhookURL)
	if _, err = bot.Request(wh); err != nil {
		log.Fatalf("Ошибка установки webhook: %v", err)
	}

	// 4. Запуск сервера
	updates := bot.ListenForWebhook("/webhook")
	go func() {
		port := "8080"
		log.Fatal(http.ListenAndServe(":"+port, nil))
	}()

	// 5. Обработка сообщений
	for update := range updates {
		handleUpdate(update)
	}
}

func handleUpdate(update tgbotapi.Update) {
	if update.Message == nil {
		return
	}

	//chatID := update.Message.Chat.ID

	if update.Message.IsCommand() {
		handleCommand(update.Message)
		return
	}

	handleRegularMessage(update.Message)
}

func handleCommand(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	command := msg.Command()
	args := msg.CommandArguments()

	switch command {
	case "settrigger":
		newTrigger, err := strconv.Atoi(args)
		if err != nil || newTrigger <= 0 {
			sendMessage(chatID, "Используйте: /settrigger <число> (например: /settrigger 5)")
			return
		}

		if err := setTrigger(chatID, newTrigger); err != nil {
			sendMessage(chatID, "Ошибка сохранения настроек")
			return
		}
		sendMessage(chatID, fmt.Sprintf("Установлено: гифка каждые %d сообщений", newTrigger))

	case "currenttrigger":
		_, triggerNum, err := getChatSettings(chatID)
		if err != nil {
			sendMessage(chatID, "Ошибка получения настроек")
			return
		}
		sendMessage(chatID, fmt.Sprintf("Текущее значение: %d сообщений", triggerNum))

	default:
		sendMessage(chatID, "Доступные команды:\n/settrigger <число> - установить триггер\n/currenttrigger - показать текущее значение")
	}
}

func handleRegularMessage(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	counter, triggerNum, err := getChatSettings(chatID)
	if err != nil {
		log.Printf("Ошибка получения настроек: %v", err)
		return
	}

	counter++
	if err := updateCounter(chatID, counter); err != nil {
		log.Printf("Ошибка обновления счетчика: %v", err)
		return
	}

	if counter >= triggerNum {
		sendGif(chatID)
		if err := updateCounter(chatID, 0); err != nil {
			log.Printf("Ошибка сброса счетчика: %v", err)
		}
	}
}

func sendGif(chatID int64) {
	msg := tgbotapi.NewAnimation(chatID, tgbotapi.FileID("CgACAgIAAxkBAAMDaD2NxM7H1-jBWRjBHYxlIxNOWIkAAr5zAAKrXvBJaM6laThye-g2BA"))
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки гифки: %v", err)
	}
}

func sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
	}
}
