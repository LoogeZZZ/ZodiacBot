package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4/middleware"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

func StartAdminPanel(dbpool *pgxpool.Pool) {
	e := echo.New()

	e.Use(middleware.BasicAuth(func(username, password string, c echo.Context) (bool, error) {

		if username == os.Getenv("ADMIN_USER") && password == os.Getenv("ADMIN_PASS") {
			return true, nil
		}
		return false, nil
	}))

	e.GET("/", func(c echo.Context) error {
		users, _ := GetAllUsers(dbpool)
		prompt := GetSetting(dbpool, "ai_prompt")
		horoscopes := GetCurrentHoroscopes(dbpool)

		currentPrompt := GetSetting(dbpool, "ai_prompt")
		if currentPrompt == "" {
			currentPrompt = "Напиши гороскоп для %s" // Заглушка, если в базе пусто
		}

		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), 6, 0, 0, 0, now.Location())
		if now.After(next) {
			next = next.Add(24 * time.Hour)
		}
		timeLeft := time.Until(next).Round(time.Second).String()

		hRows := ""
		signs := []string{"aries", "taurus", "gemini", "cancer", "leo", "virgo", "libra", "scorpio", "sagittarius", "capricorn", "aquarius", "pisces"}
		for _, s := range signs {
			txt := horoscopes[s]
			if txt == "" {
				txt = "<mark>Не сгенерировано</mark>"
			}
			hRows += fmt.Sprintf("<tr><td>%s</td><td>%s</td></tr>", ZodiacNames[s], txt)
		}

		uRows := ""
		for _, u := range users {
			uRows += fmt.Sprintf("<tr><td>%d</td><td>@%s</td><td>%s</td></tr>", u.ID, u.Username, ZodiacNames[u.ZodiacSign])
		}

		html := fmt.Sprintf(`
		<!DOCTYPE html>
		<html lang="ru">
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1">
			<title>AstroBot Admin</title>
			<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/pico.min.css">
		</head>
		<body class="container">
			<header>
				<nav>
				  <ul><li><strong>🪐 AstroBot AI Admin</strong></li></ul>
				  <ul><li><kbd>Обновление через: %s</kbd></li></ul>
				</nav>
			</header>

			<main>
				<section>
					<div class="grid">
						<article>👥 Пользователей: <strong>%d</strong></article>
						<article>🔋 Статус ИИ: <strong>Online</strong></article>
					</div>
				</section>

				<article>
					<header>📝 Настройка промпта</header>
					 <form action="/save-prompt" method="POST">
            <textarea name="prompt" rows="5" placeholder="Введите системный промпт...">%s</textarea>
            <footer>
                <button type="submit" class="outline">💾 Сохранить настройки</button>
            </footer>
        </form>
				</article>

				<article>
					<header>🚀 Гороскопы на сегодня</header>
					<figure>
						<table role="grid">
							<thead><tr><th>Знак</th><th>Текст прогноза</th></tr></thead>
							<tbody>%s</tbody>
						</table>
					</figure>
					<form action="/force-update" method="POST">
						<button type="submit" class="secondary">🔥 Force Update (Все знаки)</button>
					</form>
				</article>

				<article>
					<header>👥 Активные пользователи</header>
					<figure>
						<table role="grid">
							<thead><tr><th>ID</th><th>Username</th><th>Знак</th></tr></thead>
							<tbody>%s</tbody>
						</table>
					</figure>
				</article>
			</main>
		</body>
		</html>`, timeLeft, len(users), prompt, hRows, uRows)

		return c.HTML(http.StatusOK, html)
	})

	e.POST("/save-prompt", func(c echo.Context) error {
		UpdateSetting(dbpool, "ai_prompt", c.FormValue("prompt"))
		return c.Redirect(http.StatusSeeOther, "/")
	})

	e.POST("/force-update", func(c echo.Context) error {
		go runDailyUpdate(dbpool)
		return c.Redirect(http.StatusSeeOther, "/")
	})

	e.Logger.Fatal(e.Start(":8080"))
}
