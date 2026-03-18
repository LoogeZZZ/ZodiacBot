package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func GenerateDailyHoroscope(sign string) (string, error) {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("ключ API пуст! Проверь .env")
	}
	log.Printf("Использую ключ: %s...", apiKey[:5])
	url := "https://api.deepseek.com/v1/chat/completions"

	prompt := fmt.Sprintf("Напиши очень смешной, матерный и жесткий гороскоп на сегодня для знака зодиака %s. Максимум 3 предложения. Используй черный юмор и мат.", ZodiacNames[sign])

	reqBody, _ := json.Marshal(DeepSeekRequest{
		Model: "deepseek-chat",
		Messages: []AIMessage{
			{Role: "system", Content: "Ты — циничный и грубый астролог с черным юмором. Пишешь коротко и хлестко."},
			{Role: "user", Content: prompt},
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
