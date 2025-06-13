🧔 BarberShop Bot
Telegram-бот для управления барбершопом на Go

https://i.imgur.com/JK7w3Qp.png

🛠 Технические характеристики
go
import (
    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
    "github.com/joho/godotenv"
)
Язык: Go 1.16+

Библиотеки:

go-telegram-bot-api/v5 - работа с Telegram API

godotenv - управление конфигурацией

Архитектура: Finite State Machine

Хранение данных: In-memory maps

🔑 Аутентификация
env
TELEGRAM_TOKEN=your_bot_token
ADMIN_LOGIN=admin
ADMIN_PASSWORD=secret123
🚀 Запуск
bash
go mod tidy
go run main.go
📌 Основные функции
Функция	Метод	Описание
Управление сменами	handleShiftChange()	Отметка ✅ на смене/❌ ушел
Обмен сообщениями	handleMessageFlow()	Админ ↔ Барбер коммуникация
Управление доступом	verifyAdminAuth()	RBAC через env-переменные
