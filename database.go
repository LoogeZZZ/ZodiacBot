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

	ctx := context.Background()
	db.Exec(ctx, userTable)
	db.Exec(ctx, predTable)

	return db
}

// UpsertUser сохраняет или обновляет выбор знака зодиака
func UpsertUser(db *pgxpool.Pool, u User) error {
	query := `
		INSERT INTO users (id, username, zodiac_sign) 
		VALUES ($1, $2, $3) 
		ON CONFLICT (id) 
		DO UPDATE SET zodiac_sign = EXCLUDED.zodiac_sign, username = EXCLUDED.username;`
	_, err := db.Exec(context.Background(), query, u.ID, u.Username, u.ZodiacSign)
	return err
}

// GetUserByID находит юзера по его ID
func GetUserByID(db *pgxpool.Pool, userID int64) (User, error) {
	var u User
	err := db.QueryRow(context.Background(), "SELECT id, username, zodiac_sign FROM users WHERE id=$1", userID).
		Scan(&u.ID, &u.Username, &u.ZodiacSign)
	return u, err
}

// GetRandomPrediction берет один случайный текст из таблицы прогнозов
func GetRandomPrediction(db *pgxpool.Pool) (string, error) {
	var txt string
	query := `SELECT text FROM predictions ORDER BY RANDOM() LIMIT 1`
	err := db.QueryRow(context.Background(), query).Scan(&txt)
	if err != nil {
		return "", err
	}
	return txt, nil
}
