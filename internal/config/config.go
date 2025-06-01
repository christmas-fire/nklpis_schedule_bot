package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config представляет конфигурацию приложения
type Config struct {
	BotToken string
	AdminID  int64
}

// Load загружает конфигурацию из .env файла
func Load() (*Config, error) {
	// Загружаем .env файл
	if err := godotenv.Load(); err != nil {
		log.Printf("Ошибка загрузки .env файла: %v", err)
	}

	// Получаем токен бота
	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		log.Fatal("BOT_TOKEN не найден в .env файле")
	}

	// Получаем ID администратора
	adminStr := os.Getenv("ADMIN")
	if adminStr == "" {
		log.Fatal("ADMIN не найден в .env файле")
	}

	adminID, err := strconv.ParseInt(adminStr, 10, 64)
	if err != nil {
		log.Fatalf("Ошибка парсинга ADMIN ID: %v", err)
	}

	return &Config{
		BotToken: botToken,
		AdminID:  adminID,
	}, nil
}
