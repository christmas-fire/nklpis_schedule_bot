package main

import (
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/christmas-fire/nklpis_schedule_bot/internal/config"
	"github.com/christmas-fire/nklpis_schedule_bot/internal/database"
	"github.com/christmas-fire/nklpis_schedule_bot/internal/telegram"
)

func main() {
	// Настройка логирования в файл
	f, err := os.OpenFile("app.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Ошибка открытия файла логов: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	log.Println("Бот запущен")

	// Загружаем конфигурацию
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Ошибка загрузки конфигурации:", err)
	}

	// Подключение к SQLite
	db, err := database.NewDatabase("app.db")
	if err != nil {
		log.Fatal("Ошибка подключения к БД:", err)
	}
	defer db.Close()

	// Создаем клиента для работы с Telegram Bot API
	bot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = false
	log.Printf("Authorized on account %s", bot.Self.UserName)

	// Настройка получения обновлений
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	// Создаем обработчик обновлений
	handler := telegram.NewBotHandler(bot, db, cfg.AdminID)

	// Обрабатываем обновления
	handler.ProcessUpdates(updates)
}
