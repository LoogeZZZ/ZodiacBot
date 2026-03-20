package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func GenerateDailyHoroscope(db *pgxpool.Pool, sign string) (string, error) {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("ключ API пуст! Проверь .env")
	}

	url := "https://api.deepseek.com/v1/chat/completions"

	basePrompt := GetSetting(db, "ai_prompt")

	if basePrompt == "" {
		basePrompt = "Напиши очень смешной, матерный и жесткий гороскоп на сегодня для знака зодиака %s. Максимум 3 предложения. Используй черный юмор и мат. Будь гендерно нейтральным"
	}

	finalPrompt := fmt.Sprintf(basePrompt, ZodiacNames[sign])

	reqBody, _ := json.Marshal(DeepSeekRequest{
		Model: "deepseek-chat",
		Messages: []AIMessage{
			{Role: "system", Content: "Ты — пьяный астролог-мизантроп, которому всё надоело. Пишешь гороскопы."},
			{Role: "user", Content: finalPrompt},
		},
	})

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка сети: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения тела: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API ошибка %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result DeepSeekResponse
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return "", fmt.Errorf("ошибка парсинга JSON: %v", err)
	}

	log.Printf("ОТВЕТ ОТ DEEPSEEK: %+v\n", result)

	if len(result.Choices) > 0 {
		return result.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("ИИ вернул пустой ответ")

}
