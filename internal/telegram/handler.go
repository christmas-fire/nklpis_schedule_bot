package telegram

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/net/html"

	"github.com/christmas-fire/nklpis_schedule_bot/internal/database"
)

// BotHandler содержит экземпляры Telegram Bot API и БД
type BotHandler struct {
	bot           *tgbotapi.BotAPI
	db            *database.Database
	adminID       int64 // ID админа
	userKeyboard  tgbotapi.ReplyKeyboardMarkup
	adminKeyboard tgbotapi.ReplyKeyboardMarkup
}

// NewBotHandler создает новый экземпляр BotHandler
func NewBotHandler(bot *tgbotapi.BotAPI, db *database.Database, adminID int64) *BotHandler {
	userKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📆 Расписание 2 корпуса"),
		),
	)

	adminKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📄 Посмотреть логи"),
			tgbotapi.NewKeyboardButton("🗄️ Посмотреть БД"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📆 Расписание 2 корпуса"),
		),
	)

	return &BotHandler{
		bot:           bot,
		db:            db,
		adminID:       adminID,
		userKeyboard:  userKeyboard,
		adminKeyboard: adminKeyboard,
	}
}

// ProcessUpdates обрабатывает входящие обновления
func (h *BotHandler) ProcessUpdates(updates tgbotapi.UpdatesChannel) {
	for update := range updates {
		if update.Message == nil {
			continue
		}

		// Добавляем пользователя в БД, если его там нет
		user := update.Message.From
		err := h.db.AddUserIfNotExists(int64(user.ID), user.UserName, user.FirstName)
		if err != nil {
			log.Printf("Ошибка при добавлении пользователя в БД: %v", err)
		}

		log.Printf("[%s] %s", user.UserName, update.Message.Text)

		switch update.Message.Text {
		case "/start":
			msgText := ""
			var replyMarkup interface{}

			if user.ID == h.adminID {
				msgText = "🛠 Время поработать!"
				replyMarkup = h.adminKeyboard
			} else {
				msgText = "Привет! 👋\nВоспользуйся кнопкой ниже, чтобы получить расписание для 2-го корпуса."
				replyMarkup = h.userKeyboard
			}

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
			msg.ReplyMarkup = replyMarkup
			h.bot.Send(msg)

		case "/schedule", "📆 Расписание 2 корпуса":
			// Удаляем сообщение пользователя с клавиатурой (если возможно)
			deleteMsg := tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID)
			if _, err := h.bot.Request(deleteMsg); err != nil {
				log.Printf("Не удалось удалить сообщение: %v", err)
			}
			h.sendScheduleImages(update.Message.Chat.ID)

		case "📄 Посмотреть логи":
			if user.ID == h.adminID {
				h.sendLogs(update.Message.Chat.ID)
			} else {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "У вас нет доступа к этой команде ✋")
				h.bot.Send(msg)
			}

		case "🗄️ Посмотреть БД":
			if user.ID == h.adminID {
				dbUsers, err := h.db.GetAllUsers()
				if err != nil {
					log.Printf("Ошибка запроса к БД: %v", err)
					errMsg := tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка запроса к БД ❌")
					h.bot.Send(errMsg)
					continue
				}

				output := "🗄️ Список пользователей:\n\n"
				if len(dbUsers) == 0 {
					output += "База данных пользователей пуста."
				} else {
					// Объединяем пользователей, следя за лимитом сообщений Telegram (4096 символов)
					currentChunk := output
					for _, dbUser := range dbUsers {
						userString := fmt.Sprintf("ID: <code>%d</code>\nUsername: @%s\nName: %s\nCreated: %s",
							dbUser.ID,
							dbUser.Username,
							dbUser.FirstName,
							dbUser.CreatedAt.Format("2006-01-02 15:04:05"))

						if len(currentChunk)+len(userString)+2 > 4000 { // Оставляем запас
							msg := tgbotapi.NewMessage(update.Message.Chat.ID, currentChunk)
							msg.ParseMode = tgbotapi.ModeHTML
							h.bot.Send(msg)
							currentChunk = userString + "\n---\n\n"
						} else {
							currentChunk += userString + "\n---\n\n"
						}
					}
					// Отправляем последний кусок
					if currentChunk != output || len(dbUsers) == 1 { // Если добавили хоть одного пользователя или всего один пользователь
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, currentChunk)
						msg.ParseMode = tgbotapi.ModeHTML
						h.bot.Send(msg)
					}
				}

			} else {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "У вас нет доступа к этой команде ✋")
				h.bot.Send(msg)
			}

		default:
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Неизвестная команда.\nИспользуй кнопку ниже или /schedule ✉️")
			msg.ReplyMarkup = h.userKeyboard // Обычным пользователям всегда показываем userKeyboard
			h.bot.Send(msg)
		}
	}
}

// sendScheduleImages парсит страницу и отправляет изображения по фильтру
func (h *BotHandler) sendScheduleImages(chatID int64) {
	const url = "https://nklpis.ru/student/obrazovanie/raspisanije2/"

	resp, err := http.Get(url)
	if err != nil {
		log.Println("Ошибка при загрузке страницы:", err)
		msg := tgbotapi.NewMessage(chatID, "Не удалось загрузить расписание. Попробуйте позже.")
		h.bot.Send(msg)
		return
	}
	defer resp.Body.Close()

	log.Printf("Статус ответа: %s", resp.Status)

	doc := html.NewTokenizer(resp.Body)

	var imgURLs []string
	var inWhiteBox bool

	for {
		tt := doc.Next()
		switch tt {
		case html.ErrorToken:
			if len(imgURLs) == 0 {
				log.Println("Изображения не найдены в HTML")
			} else {
				log.Printf("Найдено изображений: %d", len(imgURLs))
				for _, url := range imgURLs {
					log.Printf("URL изображения: %s", url)
				}
			}
			h.sendImages(chatID, imgURLs)
			return
		case html.StartTagToken:
			tn, hasAttr := doc.TagName()
			tagName := string(tn)

			if tagName == "div" && hasAttr {
				for {
					key, val, more := doc.TagAttr()
					if string(key) == "class" && strings.Contains(string(val), "white-box padding-box") {
						inWhiteBox = true
					}
					if !more {
						break
					}
				}
			}

			if tagName == "img" && inWhiteBox {
				var src string
				for {
					key, val, more := doc.TagAttr()
					if string(key) == "src" {
						src = string(val)
					}
					if !more {
						break
					}
				}
				if src != "" {
					if src == "/upload/images/index--img(391).png" ||
						src == "/upload/images/index--img(389).png" ||
						src == "/upload/images/index--img(398).png" ||
						src == "/upload/images/index--img(399).png" {
						imgURLs = append(imgURLs, src)
						log.Printf("Добавлено изображение: %s", src)
					}
				}
			}
		case html.EndTagToken:
			tn, _ := doc.TagName()
			tagName := string(tn)
			if tagName == "div" && inWhiteBox {
				inWhiteBox = false
			}
		}
	}
}

// sendLogs отправляет файл логов админу
func (h *BotHandler) sendLogs(chatID int64) {
	logFile, err := os.Open("app.log")
	if err != nil {
		log.Printf("Ошибка чтения файла логов: %v", err)
		errMsg := tgbotapi.NewMessage(chatID, "Ошибка чтения файла логов ❌")
		h.bot.Send(errMsg)
		return
	}
	defer logFile.Close()

	docMsg := tgbotapi.NewDocument(chatID, tgbotapi.FileReader{Name: "app.log", Reader: logFile})
	docMsg.Caption = "📄 Файл с логами приложения"
	_, err = h.bot.Send(docMsg)
	if err != nil {
		log.Printf("Ошибка отправки файла логов: %v", err)
		errMsg := tgbotapi.NewMessage(chatID, "Ошибка отправки файла логов ❌")
		h.bot.Send(errMsg)
	}
}

// sendImages отправляет найденные изображения в чат
func (h *BotHandler) sendImages(chatID int64, urls []string) {
	if len(urls) == 0 {
		msg := tgbotapi.NewMessage(chatID, "Изображения для 2-го корпуса не найдены.")
		h.bot.Send(msg)
		return
	}

	var mediaGroup []interface{}
	for i, u := range urls {
		fullURL := resolveRelative(u)
		media := tgbotapi.NewInputMediaPhoto(tgbotapi.FileURL(fullURL))
		if i == 0 {
			media.Caption = "📅 Вот расписание для 2-го корпуса"
		}
		mediaGroup = append(mediaGroup, media)
	}

	// Telegram ограничивает до 10 медиа в одной группе
	for i := 0; i < len(mediaGroup); i += 10 {
		end := i + 10
		if end > len(mediaGroup) {
			end = len(mediaGroup)
		}
		_, err := h.bot.SendMediaGroup(tgbotapi.MediaGroupConfig{
			ChatID: chatID,
			Media:  mediaGroup[i:end],
		})
		if err != nil {
			log.Printf("Ошибка при отправке медиа-группы: %v", err)
		}
	}
}

// resolveRelative добавляет домен, если ссылка относительная
func resolveRelative(href string) string {
	const baseURL = "https://nklpis.ru"
	if strings.HasPrefix(href, "http") {
		return href
	}
	if strings.HasPrefix(href, "/") {
		return baseURL + href
	}
	return baseURL + "/" + href
}
