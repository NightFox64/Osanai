package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ChatData структура для хранения данных о чате
type ChatData struct {
	MessageCount int `json:"message_count"`
	Threshold    int `json:"threshold"`
}

// BotData структура для хранения данных всех чатов
type BotData struct {
	Chats map[int64]ChatData `json:"chats"`
	mu    sync.RWMutex
}

const (
	dataFile    = "bot_data.json"
	maxMessages = 10000
	gifID       = "CgACAgIAAxkBAAMDaD2NxM7H1-jBWRjBHYxlIxNOWIkAAr5zAAKrXvBJaM6laThye-g2BA"
)

var bot *tgbotapi.BotAPI

func main() {
	// Инициализация бота
	token := "7843550853:AAE1Ih5G1WuEnKDPSXRj3DaOLB6y8-mhBF8" // Замените на ваш токен
	var err error

	bot, err = tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Panicf("Ошибка создания бота: %v", err)
	}

	bot.Debug = true
	log.Printf("Авторизован как %s", bot.Self.UserName)

	// Удаляем вебхук если он активен
	_, err = bot.Request(tgbotapi.DeleteWebhookConfig{})
	if err != nil {
		log.Printf("Предупреждение при удалении вебхука: %v", err)
	}

	// Загрузка данных
	botData := NewBotData()
	if err := botData.LoadFromFile(); err != nil {
		log.Printf("Ошибка загрузки данных: %v", err)
	}

	// Настройка обновлений
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)
	log.Println("Бот запущен и ожидает сообщения...")

	// Обработка сообщений
	for update := range updates {
		if update.Message == nil {
			continue
		}

		// Обрабатываем синхронно, а не в горутине
		handleMessage(update.Message, botData)
	}
}

// NewBotData создает новую структуру данных бота
func NewBotData() *BotData {
	return &BotData{
		Chats: make(map[int64]ChatData),
	}
}

// LoadFromFile загружает данные из файла
func (bd *BotData) LoadFromFile() error {
	bd.mu.Lock()
	defer bd.mu.Unlock()

	file, err := os.Open(dataFile)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("Файл данных не найден, создается новый")
			// Создаем пустой файл
			if err := bd.saveToFile(); err != nil {
				return err
			}
			return nil
		}
		return fmt.Errorf("ошибка открытия файла: %v", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&bd.Chats); err != nil {
		return fmt.Errorf("ошибка декодирования JSON: %v", err)
	}

	log.Printf("Загружены данные для %d чатов", len(bd.Chats))
	return nil
}

// saveToFile сохраняет данные в файл (внутренний метод)
func (bd *BotData) saveToFile() error {
	// Создаем временный файл для атомарной записи
	tempFile := dataFile + ".tmp"
	file, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("ошибка создания временного файла: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(bd.Chats); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("ошибка кодирования JSON: %v", err)
	}

	// Закрываем файл перед переименованием
	file.Close()

	// Атомарно заменяем старый файл новым
	if err := os.Rename(tempFile, dataFile); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("ошибка переименования файла: %v", err)
	}

	return nil
}

// SaveToFile сохраняет данные в файл
func (bd *BotData) SaveToFile() error {
	bd.mu.Lock()
	defer bd.mu.Unlock()
	return bd.saveToFile()
}

// GetChatData возвращает данные чата
func (bd *BotData) GetChatData(chatID int64) (ChatData, bool) {
	bd.mu.RLock()
	defer bd.mu.RUnlock()

	data, exists := bd.Chats[chatID]
	return data, exists
}

// SetChatData устанавливает данные чата
func (bd *BotData) SetChatData(chatID int64, data ChatData) error {
	bd.mu.Lock()
	defer bd.mu.Unlock()

	if bd.Chats == nil {
		bd.Chats = make(map[int64]ChatData)
	}

	bd.Chats[chatID] = data
	return bd.saveToFile()
}

// ResetChatCounter сбрасывает счетчик сообщений
func (bd *BotData) ResetChatCounter(chatID int64) error {
	bd.mu.Lock()
	defer bd.mu.Unlock()

	if data, exists := bd.Chats[chatID]; exists {
		data.MessageCount = 0
		bd.Chats[chatID] = data
		return bd.saveToFile()
	}
	return nil
}

// IncrementMessageCount увеличивает счетчик сообщений
func (bd *BotData) IncrementMessageCount(chatID int64) (ChatData, error) {
	bd.mu.Lock()
	defer bd.mu.Unlock()

	if bd.Chats == nil {
		bd.Chats = make(map[int64]ChatData)
	}

	data := bd.Chats[chatID]
	data.MessageCount++
	bd.Chats[chatID] = data

	err := bd.saveToFile()
	return data, err
}

func handleMessage(message *tgbotapi.Message, botData *BotData) {
	chatID := message.Chat.ID
	log.Printf("Получено сообщение от %d: %s", chatID, message.Text)

	// Обработка команд
	if message.IsCommand() {
		switch message.Command() {
		case "start":
			sendWelcomeMessage(chatID)
			return

		case "set":
			handleSetCommand(message, botData)
			return

		case "status":
			handleStatusCommand(message, botData)
			return

		case "reset":
			handleResetCommand(message, botData)
			return

		default:
			sendMessage(chatID, "❌ Неизвестная команда. Используйте /start для справки")
			return
		}
	}

	// Обработка обычных сообщений
	chatData, exists := botData.GetChatData(chatID)
	if !exists || chatData.Threshold == 0 {
		return
	}

	// Увеличиваем счетчик
	newData, err := botData.IncrementMessageCount(chatID)
	if err != nil {
		log.Printf("Ошибка увеличения счетчика для чата %d: %v", chatID, err)
		return
	}

	log.Printf("Чат %d: сообщение %d/%d", chatID, newData.MessageCount, newData.Threshold)

	// Проверяем, достигли ли порога
	if newData.MessageCount >= newData.Threshold {
		// Отправляем гифку
		sendGif(chatID, gifID)

		// Обнуляем счетчик
		if err := botData.ResetChatCounter(chatID); err != nil {
			log.Printf("Ошибка сброса счетчика для чата %d: %v", chatID, err)
		} else {
			log.Printf("Чат %d: достигнут порог %d, отправлена гифка", chatID, newData.Threshold)
		}
	}
}

func sendWelcomeMessage(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "🎉 Привет! Я бот, который отправляет гифку каждые N сообщений.\n\n"+
		"📝 Доступные команды:\n"+
		"/set N - установить порог сообщений (1-10000)\n"+
		"/status - посмотреть текущий статус\n"+
		"/reset - сбросить счетчик сообщений\n\n"+
		"⚡ Пример: /set 50 - буду отправлять гифку каждые 50 сообщений")

	msg.ParseMode = "Markdown"
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки приветственного сообщения: %v", err)
	}
}

func handleSetCommand(message *tgbotapi.Message, botData *BotData) {
	chatID := message.Chat.ID
	args := message.CommandArguments()

	if args == "" {
		sendMessage(chatID, "❌ Укажите количество сообщений. Например: `/set 50`")
		return
	}

	threshold, err := strconv.Atoi(args)
	if err != nil || threshold <= 0 {
		sendMessage(chatID, "❌ Пожалуйста, укажите положительное число")
		return
	}

	if threshold > maxMessages {
		sendMessage(chatID, fmt.Sprintf("❌ Максимальное значение: %d", maxMessages))
		return
	}

	chatData := ChatData{
		Threshold:    threshold,
		MessageCount: 0,
	}

	if err := botData.SetChatData(chatID, chatData); err != nil {
		sendMessage(chatID, "❌ Ошибка сохранения настроек")
		log.Printf("Ошибка установки порога для чата %d: %v", chatID, err)
		return
	}

	sendMessage(chatID, fmt.Sprintf("✅ Установлен порог: *%d сообщений*\nСчетчик сброшен в *0*", threshold))
	log.Printf("Чат %d: установлен порог %d сообщений", chatID, threshold)
}

func handleStatusCommand(message *tgbotapi.Message, botData *BotData) {
	chatID := message.Chat.ID

	chatData, exists := botData.GetChatData(chatID)
	if !exists || chatData.Threshold == 0 {
		sendMessage(chatID, "📊 Порог не установлен. Используйте `/set N`")
		return
	}

	progress := float64(chatData.MessageCount) / float64(chatData.Threshold) * 100
	messageText := fmt.Sprintf("📊 Статус:\n"+
		"• Порог: *%d сообщений*\n"+
		"• Отсчитано: *%d/%d*\n"+
		"• Прогресс: *%.1f%%*",
		chatData.Threshold, chatData.MessageCount, chatData.Threshold, progress)

	sendMessage(chatID, messageText)
}

func handleResetCommand(message *tgbotapi.Message, botData *BotData) {
	chatID := message.Chat.ID

	if err := botData.ResetChatCounter(chatID); err != nil {
		sendMessage(chatID, "❌ Ошибка сброса счетчика")
		log.Printf("Ошибка сброса счетчика для чата %d: %v", chatID, err)
		return
	}

	sendMessage(chatID, "🔄 Счетчик сброшен в *0*")
	log.Printf("Чат %d: счетчик сброшен", chatID)
}

func sendGif(chatID int64, gifID string) {
	gif := tgbotapi.NewAnimation(chatID, tgbotapi.FileID(gifID))
	gif.Caption = "🎉 Достигнут порог сообщений!"

	if _, err := bot.Send(gif); err != nil {
		log.Printf("Ошибка отправки гифки в чат %d: %v", chatID, err)
		sendMessage(chatID, "🎉 Достигнут порог сообщений! (Не удалось отправить гифку)")
	}
}

func sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки сообщения в чат %d: %v", chatID, err)
	}
}
