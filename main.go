package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"employee-bot/google_sheets"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"google.golang.org/api/sheets/v4"
)

var (
	userRoles     = make(map[int64]string)
	userLocations = make(map[int64]string)
	userStates    = make(map[int64]string)
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Ошибка загрузки .env файла")
	}

	sheetsService, err := google_sheets.InitSheetService()
	if err != nil {
		log.Fatalf("Ошибка Google Sheets: %v", err)
	}
	fmt.Println("✅ Успешно подключено к Google Sheets!")

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_TOKEN"))
	if err != nil {
		log.Fatal("Ошибка бота: ", err)
	}
	bot.Debug = true
	log.Printf("Бот запущен: @%s", bot.Self.UserName)

	// Инициализация таблицы
	initSheetHeaders(sheetsService)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID
		username := strings.TrimSpace(update.Message.From.FirstName + " " + update.Message.From.LastName)

		log.Printf("Получено сообщение от %s (%d): %s", username, chatID, update.Message.Text)

		// Обработка команд верхнего уровня
		switch update.Message.Text {
		case "🏠 Главное меню":
			delete(userStates, chatID)
			showMainMenu(bot, chatID)
			continue
		case "🔄 Сменить роль":
			resetUserData(chatID)
			sendRoleSelection(bot, chatID)
			continue
		case "📍 Сменить адрес":
			askForLocation(bot, chatID)
			continue
		case "ℹ️ Мой статус":
			showUserStatus(bot, chatID)
			continue
		case "/start":
			resetUserData(chatID)
			sendRoleSelection(bot, chatID)
			continue
		}

		// Обработка состояний
		if state, exists := userStates[chatID]; exists {
			switch state {
			case "awaiting_role":
				processRoleSelection(bot, chatID, update.Message.Text)
				continue
			case "awaiting_location":
				processLocationSelection(bot, chatID, update.Message.Text)
				continue
			case "awaiting_status":
				processStatusSelection(bot, sheetsService, chatID, username, update.Message.Text)
				continue
			}
		}

		// Основные действия
		switch update.Message.Text {
		case "✅ На работе", "🏠 Ушёл":
			writeStatus(bot, sheetsService, chatID, username, getStatusFromText(update.Message.Text))
		default:
			if _, hasRole := userRoles[chatID]; !hasRole {
				sendRoleSelection(bot, chatID)
			} else {
				showMainMenu(bot, chatID)
			}
		}
	}
}

func initSheetHeaders(service *sheets.Service) {
	_, err := service.Spreadsheets.Values.Update(
		os.Getenv("GOOGLE_SHEET_ID"),
		"Users!A1:G1",
		&sheets.ValueRange{
			Values: [][]interface{}{
				{"Дата", "Время", "Имя сотрудника", "Роль", "Адрес", "Действие", "Комментарий"},
			},
		},
	).ValueInputOption("RAW").Do()

	if err != nil {
		log.Printf("Ошибка инициализации таблицы: %v", err)
	}
}

func resetUserData(chatID int64) {
	delete(userRoles, chatID)
	delete(userLocations, chatID)
	delete(userStates, chatID)
}

func processRoleSelection(bot *tgbotapi.BotAPI, chatID int64, text string) {
	if text == "Барбер" || text == "Администратор" {
		userRoles[chatID] = text
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("✅ Роль «%s» сохранена!", text))
		bot.Send(msg)

		if text == "Барбер" {
			askForLocation(bot, chatID)
		} else {
			askForStatus(bot, chatID)
		}
	} else {
		msg := tgbotapi.NewMessage(chatID, "Пожалуйста, выберите роль из предложенных вариантов")
		bot.Send(msg)
		sendRoleSelection(bot, chatID)
	}
}

func processLocationSelection(bot *tgbotapi.BotAPI, chatID int64, text string) {
	if text == "🏠 Главное меню" {
		delete(userStates, chatID)
		showMainMenu(bot, chatID)
		return
	}

	if contains([]string{"Центр", "Север", "Юг", "Восток"}, text) {
		userLocations[chatID] = text
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("📍 Вы выбрали адрес: %s", text))
		bot.Send(msg)
		askForStatus(bot, chatID)
	} else {
		msg := tgbotapi.NewMessage(chatID, "Пожалуйста, выберите адрес из предложенных вариантов")
		bot.Send(msg)
		askForLocation(bot, chatID)
	}
}

func processStatusSelection(bot *tgbotapi.BotAPI, service *sheets.Service, chatID int64, username, text string) {
	if text == "✅ На работе" || text == "🏠 Ушёл" {
		writeStatus(bot, service, chatID, username, getStatusFromText(text))
		delete(userStates, chatID)
		showMainMenu(bot, chatID)
	} else {
		msg := tgbotapi.NewMessage(chatID, "Пожалуйста, выберите статус из предложенных вариантов")
		bot.Send(msg)
		askForStatus(bot, chatID)
	}
}

func showUserStatus(bot *tgbotapi.BotAPI, chatID int64) {
	role := userRoles[chatID]
	location, hasLocation := userLocations[chatID]

	msgText := fmt.Sprintf("📊 Ваш текущий статус:\n\n👤 Роль: %s\n", role)
	if hasLocation {
		msgText += fmt.Sprintf("📍 Локация: %s\n", location)
	}

	msg := tgbotapi.NewMessage(chatID, msgText)
	msg.ReplyMarkup = createMainMenuKeyboard()
	bot.Send(msg)
}

func sendRoleSelection(bot *tgbotapi.BotAPI, chatID int64) {
	userStates[chatID] = "awaiting_role"
	msg := tgbotapi.NewMessage(chatID, "👥 Выберите вашу роль:")
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Барбер"),
			tgbotapi.NewKeyboardButton("Администратор"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🏠 Главное меню"),
		),
	)
	bot.Send(msg)
}

func askForLocation(bot *tgbotapi.BotAPI, chatID int64) {
	userStates[chatID] = "awaiting_location"
	msg := tgbotapi.NewMessage(chatID, "🏢 Выберите адрес, где вы сейчас работаете:")
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Центр"),
			tgbotapi.NewKeyboardButton("Север"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Юг"),
			tgbotapi.NewKeyboardButton("Восток"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🏠 Главное меню"),
		),
	)
	bot.Send(msg)
}

func askForStatus(bot *tgbotapi.BotAPI, chatID int64) {
	userStates[chatID] = "awaiting_status"
	msg := tgbotapi.NewMessage(chatID, "Отметьте ваш текущий статус:")
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("✅ На работе"),
			tgbotapi.NewKeyboardButton("🏠 Ушёл"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🏠 Главное меню"),
		),
	)
	bot.Send(msg)
}

func showMainMenu(bot *tgbotapi.BotAPI, chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "📱 Главное меню:")
	msg.ReplyMarkup = createMainMenuKeyboard()
	bot.Send(msg)
}

func createMainMenuKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("✅ На работе"),
			tgbotapi.NewKeyboardButton("🏠 Ушёл"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📍 Сменить адрес"),
			tgbotapi.NewKeyboardButton("ℹ️ Мой статус"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🔄 Сменить роль"),
		),
	)
}

func writeStatus(bot *tgbotapi.BotAPI, service *sheets.Service, chatID int64, fullName, status string) {
	now := time.Now()
	record := []interface{}{
		now.Format("2006-01-02"),
		now.Format("15:04:05"),
		fullName,
		userRoles[chatID],
		userLocations[chatID],
		status,
		"",
	}

	_, err := service.Spreadsheets.Values.Append(
		os.Getenv("GOOGLE_SHEET_ID"),
		"Users!A:G",
		&sheets.ValueRange{
			Values: [][]interface{}{record},
		},
	).ValueInputOption("RAW").Do()

	var msgText string
	if err != nil {
		log.Printf("Ошибка записи: %v", err)
		msgText = "❌ Ошибка отметки! Попробуйте позже."
	} else {
		if status == "Пришел" {
			msgText = "✅ Вы зарегистрированы на работе. Хорошего дня!"
		} else {
			msgText = "🏠 До свидания! Хорошего отдыха!"
		}
	}

	msg := tgbotapi.NewMessage(chatID, msgText)
	msg.ReplyMarkup = createMainMenuKeyboard()
	bot.Send(msg)
}

func getStatusFromText(text string) string {
	if text == "✅ На работе" {
		return "Пришел"
	}
	return "Ушёл"
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
