package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ========== Инициализация БД ==========

func InitDB(connStr string) *pgxpool.Pool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}

	// Проверяем соединение
	if err := db.Ping(ctx); err != nil {
		log.Fatalf("БД не отвечает: %v", err)
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id BIGINT PRIMARY KEY,
			username TEXT,
			zodiac_sign TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS predictions (
			id SERIAL PRIMARY KEY,
			text TEXT NOT NULL
		);`,
		// Исправлена синтаксическая ошибка: убрана лишняя скобка и добавлено закрытие
		`CREATE TABLE IF NOT EXISTS daily_horoscope (
			id SERIAL PRIMARY KEY,
			zodiac_sign TEXT NOT NULL,
			prediction_text TEXT NOT NULL,
			target_date DATE DEFAULT CURRENT_DATE,
			UNIQUE(zodiac_sign, target_date)
		);`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT
		);`,
	}

	for _, q := range queries {
		if _, err := db.Exec(ctx, q); err != nil {
			log.Fatalf("Ошибка создания таблицы: %v\nЗапрос: %s", err, q)
		}
	}

	log.Println("Таблицы БД проверены/созданы")
	return db
}

// ========== Работа с пользователями ==========

func UpsertUser(db *pgxpool.Pool, u User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO users (id, username, zodiac_sign) 
		VALUES ($1, $2, $3) 
		ON CONFLICT (id) 
		DO UPDATE SET zodiac_sign = EXCLUDED.zodiac_sign, username = EXCLUDED.username`
	_, err := db.Exec(ctx, query, u.ID, u.Username, u.ZodiacSign)
	return err
}

func GetUserByID(db *pgxpool.Pool, userID int64) (User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var u User
	query := `SELECT id, username, zodiac_sign FROM users WHERE id=$1`
	err := db.QueryRow(ctx, query, userID).Scan(&u.ID, &u.Username, &u.ZodiacSign)
	return u, err
}

func GetAllUsers(db *pgxpool.Pool) ([]User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := db.Query(ctx, "SELECT id, username, zodiac_sign FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.ZodiacSign); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// ========== Работа с настройками + кэш ==========

var (
	promptCache     string
	promptCacheMu   sync.RWMutex
	promptCacheTime time.Time
	cacheTTL        = 5 * time.Minute
)

func GetSetting(db *pgxpool.Pool, key string) string {
	// Пытаемся взять из кэша
	promptCacheMu.RLock()
	if time.Since(promptCacheTime) < cacheTTL && promptCache != "" {
		defer promptCacheMu.RUnlock()
		return promptCache
	}
	promptCacheMu.RUnlock()

	// Иначе идём в БД
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var val string
	query := `SELECT value FROM settings WHERE key = $1`
	err := db.QueryRow(ctx, query, key).Scan(&val)
	if err != nil {
		return ""
	}

	// Обновляем кэш
	promptCacheMu.Lock()
	promptCache = val
	promptCacheTime = time.Now()
	promptCacheMu.Unlock()

	return val
}

func UpdateSetting(db *pgxpool.Pool, key, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.Exec(ctx,
		"INSERT INTO settings (key, value) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET value = $2",
		key, value)

	// Если обновили промпт — сбрасываем кэш
	if err == nil && key == "ai_prompt" {
		promptCacheMu.Lock()
		promptCache = ""
		promptCacheTime = time.Time{}
		promptCacheMu.Unlock()
	}
	return err
}

// ========== Гороскопы (ежедневные) ==========

func SaveDailyPrediction(db *pgxpool.Pool, sign string, text string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO daily_horoscope (zodiac_sign, prediction_text, target_date) 
		VALUES ($1, $2, CURRENT_DATE)
		ON CONFLICT (zodiac_sign, target_date) 
		DO UPDATE SET prediction_text = EXCLUDED.prediction_text`
	_, err := db.Exec(ctx, query, sign, text)
	return err
}

func GetDailyPrediction(db *pgxpool.Pool, sign string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var txt string
	query := `SELECT prediction_text FROM daily_horoscope WHERE zodiac_sign = $1 AND target_date = CURRENT_DATE`
	err := db.QueryRow(ctx, query, sign).Scan(&txt)
	return txt, err
}

func GetCurrentHoroscopes(db *pgxpool.Pool) map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := db.Query(ctx, "SELECT zodiac_sign, prediction_text FROM daily_horoscope WHERE target_date = CURRENT_DATE")
	if err != nil {
		log.Printf("Ошибка получения гороскопов: %v", err)
		return make(map[string]string)
	}
	defer rows.Close()

	res := make(map[string]string)
	for rows.Next() {
		var sign, text string
		if err := rows.Scan(&sign, &text); err != nil {
			log.Printf("Ошибка сканирования: %v", err)
			continue
		}
		res[sign] = text
	}
	return res
}

// ========== Вспомогательные функции (совместимость) ==========

// GetRandomPredictionBySign — если понадобится, но в текущей логике не используется
func GetRandomPredictionBySign(db *pgxpool.Pool, sign string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var txt string
	query := `SELECT text FROM predictions WHERE zodiac_sign = $1 ORDER BY RANDOM() LIMIT 1`
	err := db.QueryRow(ctx, query, sign).Scan(&txt)
	return txt, err
}
