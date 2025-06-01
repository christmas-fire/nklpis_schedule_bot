package database

import (
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// Database представляет соединение с базой данных
type Database struct {
	DB *sqlx.DB
}

// NewDatabase создает новое подключение к БД
func NewDatabase(dbPath string) (*Database, error) {
	db, err := sqlx.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// Проверка соединения
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	// Создаем таблицу, если она не существует
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS bot_users (
			id INTEGER PRIMARY KEY,
			username TEXT,
			first_name TEXT,
			created_at TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		return nil, err
	}

	log.Println("Успешное подключение к БД")

	return &Database{DB: db}, nil
}

// Close закрывает соединение с БД
func (d *Database) Close() error {
	log.Println("Закрытие соединения с БД")
	return d.DB.Close()
}

// AddUserIfNotExists добавляет пользователя, если его нет в БД
func (d *Database) AddUserIfNotExists(id int64, username, firstName string) error {
	_, err := d.DB.Exec(`
		INSERT INTO bot_users (id, username, first_name, created_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (id) DO NOTHING
	`, id, username, firstName, time.Now())
	return err
}

// GetAllUsers извлекает всех пользователей из БД
func (d *Database) GetAllUsers() ([]User, error) {
	var users []User
	err := d.DB.Select(&users, "SELECT id, username, first_name, created_at FROM bot_users")
	if err != nil {
		return nil, err
	}

	// Обработка NULL значений
	for i := range users {
		if users[i].Username == "" {
			users[i].Username = "нет"
		}
		if users[i].FirstName == "" {
			users[i].FirstName = "нет"
		}
	}

	return users, nil
}

// User представляет структуру пользователя из БД
type User struct {
	ID        int64     `db:"id"`
	Username  string    `db:"username"`
	FirstName string    `db:"first_name"`
	CreatedAt time.Time `db:"created_at"`
}
