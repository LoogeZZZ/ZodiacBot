package main

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

func InitDB(connStr string) *pgxpool.Pool {
	db, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}

	userTable := `
	CREATE TABLE IF NOT EXISTS users (
		id BIGINT PRIMARY KEY,
		username TEXT,
		zodiac_sign TEXT NOT NULL
	);`

	predTable := `
	CREATE TABLE IF NOT EXISTS predictions (
		id SERIAL PRIMARY KEY,
		text TEXT NOT NULL
	);`

	dailyHoroscope := `CREATE TABLE IF NOT EXISTS daily_horoscope (
		id SERIAL PRIMARY KEY,
		zodiac_sign TEXT NOT NULL,
		prediction_text TEXT NOT NULL,
		target_date DATE DEFAULT CURRENT_DATE,
		UNIQUE(zodiac_sign, target_date);`

	ctx := context.Background()
	db.Exec(ctx, dailyHoroscope)
	db.Exec(ctx, userTable)
	db.Exec(ctx, predTable)

	return db
}

func UpsertUser(db *pgxpool.Pool, u User) error {
	query := `
		INSERT INTO users (id, username, zodiac_sign) 
		VALUES ($1, $2, $3) 
		ON CONFLICT (id) 
		DO UPDATE SET zodiac_sign = EXCLUDED.zodiac_sign, username = EXCLUDED.username;`
	_, err := db.Exec(context.Background(), query, u.ID, u.Username, u.ZodiacSign)
	return err
}

func GetUserByID(db *pgxpool.Pool, userID int64) (User, error) {
	var u User
	err := db.QueryRow(context.Background(), "SELECT id, username, zodiac_sign FROM users WHERE id=$1", userID).
		Scan(&u.ID, &u.Username, &u.ZodiacSign)
	return u, err
}

func GetRandomPredictionBySign(db *pgxpool.Pool, sign string) (string, error) {
	var txt string

	query := `SELECT text FROM predictions WHERE zodiac_sign = $1 ORDER BY RANDOM() LIMIT 1`

	err := db.QueryRow(context.Background(), query, sign).Scan(&txt)
	if err != nil {
		return "", err
	}
	return txt, nil
}

func SaveDailyPrediction(db *pgxpool.Pool, sign string, text string) error {
	query := `
		INSERT INTO daily_horoscope (zodiac_sign, prediction_text, target_date) 
		VALUES ($1, $2, CURRENT_DATE)
		ON CONFLICT (zodiac_sign, target_date) 
		DO UPDATE SET prediction_text = EXCLUDED.prediction_text;`

	_, err := db.Exec(context.Background(), query, sign, text)
	return err
}

func GetDailyPrediction(db *pgxpool.Pool, sign string) (string, error) {
	var txt string
	query := `SELECT prediction_text FROM daily_horoscope WHERE zodiac_sign = $1 AND target_date = CURRENT_DATE`

	err := db.QueryRow(context.Background(), query, sign).Scan(&txt)
	return txt, err
}

func GetSetting(db *pgxpool.Pool, key string) string {
	var val string
	query := `SELECT value FROM settings WHERE key = $1`
	err := db.QueryRow(context.Background(), query, key).Scan(&val)
	if err != nil {
		return ""
	}
	return val
}

func UpdateSetting(db *pgxpool.Pool, key, value string) error {
	_, err := db.Exec(context.Background(),
		"INSERT INTO settings (key, value) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET value = $2",
		key, value)
	return err
}

func GetCurrentHoroscopes(db *pgxpool.Pool) map[string]string {
	rows, _ := db.Query(context.Background(), "SELECT zodiac_sign, prediction_text FROM daily_horoscope WHERE target_date = CURRENT_DATE")
	defer rows.Close()
	res := make(map[string]string)
	for rows.Next() {
		var sign, text string
		rows.Scan(&sign, &text)
		res[sign] = text
	}
	return res
}

func GetAllUsers(db *pgxpool.Pool) ([]User, error) {
	rows, err := db.Query(context.Background(), "SELECT id, username, zodiac_sign FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		var u User
		rows.Scan(&u.ID, &u.Username, &u.ZodiacSign)
		users = append(users, u)
	}
	return users, nil
}
