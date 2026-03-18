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
