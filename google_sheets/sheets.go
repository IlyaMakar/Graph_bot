package google_sheets

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// Инициализация доступа к Google Sheets
func InitSheetService() (*sheets.Service, error) {
	ctx := context.Background()

	// 1. Достаём JSON из переменной .env
	credsJSON := os.Getenv("GOOGLE_CREDENTIALS")
	if credsJSON == "" {
		return nil, fmt.Errorf("GOOGLE_CREDENTIALS не заданы в .env")
	}

	// 2. Подключаемся к API
	config, err := google.JWTConfigFromJSON([]byte(credsJSON), sheets.SpreadsheetsScope)
	if err != nil {
		return nil, fmt.Errorf("ошибка в JSON: %v", err)
	}

	// 3. Создаём сервис
	client := config.Client(ctx)
	srv, err := sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("не удалось создать сервис: %v", err)
	}

	return srv, nil
}
