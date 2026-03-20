package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lpernett/godotenv"
	"github.com/robfig/cron/v3"
)

var dbpool *pgxpool.Pool

func main() {
	godotenv.Load()

	dbURL := os.Getenv("DB_URL")
	botToken := os.Getenv("BOT_TOKEN")

	if dbURL == "" || botToken == "" {
		log.Fatal("Ошибка: переменные окружения DB_URL или BOT_TOKEN не установлены")
	}

	var err error
	dbpool = InitDB(dbURL)
	defer dbpool.Close()

	log.Println("Запуск проверки генерации ИИ...")
	runDailyUpdate(dbpool)

	startCron(dbpool)

	go StartAdminPanel(dbpool)

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Авторизован на аккаунте %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {

		if update.CallbackQuery != nil {
			handleCallback(bot, dbpool, update.CallbackQuery)
			continue
		}

		if update.Message != nil {
			handleMessage(bot, dbpool, update.Message)
		}
	}

}

func handleMessage(bot *tgbotapi.BotAPI, dbpool *pgxpool.Pool, msg *tgbotapi.Message) {
	userID := msg.From.ID
	userName := msg.From.FirstName
	botTag := "@" + bot.Self.UserName

	if msg.Chat.Type == "private" {
		switch msg.Text {
		case "/start":
			reply := tgbotapi.NewMessage(msg.Chat.ID, "Привет, "+userName+"! Я твой личный астро-бот.")
			reply.ReplyMarkup = getPrivateMenu()
			bot.Send(reply)

		case "👤 Мой профиль":
			user, err := GetUserByID(dbpool, userID)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Ты еще не выбрал знак. Нажми 'Изменить знак'."))
			} else {
				text := fmt.Sprintf("📋 Твой профиль:\nИмя: %s\nЗнак: %s", userName, ZodiacNames[user.ZodiacSign])
				bot.Send(tgbotapi.NewMessage(msg.Chat.ID, text))
			}

		case "🔄 Изменить знак", "/change":
			reply := tgbotapi.NewMessage(msg.Chat.ID, "Выбери свой знак зодиака:")
			reply.ReplyMarkup = getZodiacKeyboard()
			bot.Send(reply)

		case "🔮 Получить прогноз":
			user, err := GetUserByID(dbpool, userID)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Сначала выбери знак! 👆"))
				return
			}

			prediction, err := GetDailyPrediction(dbpool, user.ZodiacSign)
			if err != nil {
				// Если в базе нет на сегодня, генерируем на лету
				prediction, err = GenerateDailyHoroscope(dbpool, user.ZodiacSign)
				SaveDailyPrediction(dbpool, user.ZodiacSign, prediction)
			}

			if prediction == "" {
				prediction = "Звезды сегодня в ахуе и молчат."
			}

			text := fmt.Sprintf("🔮 Прогноз для тебя (%s):\n\n%s", ZodiacNames[user.ZodiacSign], prediction)
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, text))
		}
		return
	}

	if strings.Contains(msg.Text, botTag) || strings.HasPrefix(msg.Text, "/change") {

		bot.Send(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))

		if strings.HasPrefix(msg.Text, "/change") {
			reply := tgbotapi.NewMessage(msg.Chat.ID, userName+", выбери знак:")
			reply.ReplyMarkup = getZodiacKeyboard()
			sent, _ := bot.Send(reply)
			deleteDelayed(bot, msg.Chat.ID, sent.MessageID, 30*time.Second)
			return
		}

		user, err := GetUserByID(dbpool, userID)
		if err != nil {
			res, _ := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, userName+", нажми /change, чтобы я тебя запомнил!"))
			deleteDelayed(bot, msg.Chat.ID, res.MessageID, 10*time.Second)
			return
		}

		prediction, err := GetDailyPrediction(dbpool, user.ZodiacSign)
		if err != nil {
			prediction, err = GenerateDailyHoroscope(dbpool, user.ZodiacSign)
			SaveDailyPrediction(dbpool, user.ZodiacSign, prediction)
		}

		if prediction == "" {
			prediction = "Звезды сегодня в ахуе и молчат."
		}

		text := fmt.Sprintf("🔮 %s (%s):\n\n%s", userName, ZodiacNames[user.ZodiacSign], prediction)
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, text))
	}
}

func handleCallback(bot *tgbotapi.BotAPI, dbpool *pgxpool.Pool, cb *tgbotapi.CallbackQuery) {
	user := User{
		ID:         cb.From.ID,
		Username:   cb.From.UserName,
		ZodiacSign: cb.Data,
	}

	UpsertUser(dbpool, user)

	bot.Send(tgbotapi.NewDeleteMessage(cb.Message.Chat.ID, cb.Message.MessageID))

	text := "✅ Запомнил, что ты " + ZodiacNames[cb.Data]
	sent, _ := bot.Send(tgbotapi.NewMessage(cb.Message.Chat.ID, text))

	deleteDelayed(bot, cb.Message.Chat.ID, sent.MessageID, 10*time.Second)

	bot.Send(tgbotapi.NewCallback(cb.ID, ""))
}

func deleteDelayed(bot *tgbotapi.BotAPI, cID int64, mID int, d time.Duration) {
	go func() {
		time.Sleep(d)
		bot.Send(tgbotapi.NewDeleteMessage(cID, mID))
	}()
}

func getZodiacKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Овен ♈", "aries"),
			tgbotapi.NewInlineKeyboardButtonData("Телец ♉", "taurus"),
			tgbotapi.NewInlineKeyboardButtonData("Близнецы ♊", "gemini"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Рак ♋", "cancer"),
			tgbotapi.NewInlineKeyboardButtonData("Лев ♌", "leo"),
			tgbotapi.NewInlineKeyboardButtonData("Дева ♍", "virgo"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Весы ♎", "libra"),
			tgbotapi.NewInlineKeyboardButtonData("Скорпион ♏", "scorpio"),
			tgbotapi.NewInlineKeyboardButtonData("Стрелец ♐", "sagittarius"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Козерог ♑", "capricorn"),
			tgbotapi.NewInlineKeyboardButtonData("Водолей ♒", "aquarius"),
			tgbotapi.NewInlineKeyboardButtonData("Рыбы ♓", "pisces"),
		),
	)
}
func getPrivateMenu() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🔮 Получить прогноз"),
			tgbotapi.NewKeyboardButton("👤 Мой профиль"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🔄 Изменить знак"),
		),
	)
}

func startCron(db *pgxpool.Pool) {
	c := cron.New()

	c.AddFunc("0 6 * * *", func() {
		log.Printf("Запуск ежедневной генерации прогнозов...")
		signs := []string{"aries", "taurus", "gemini", "cancer", "leo", "virgo", "libra", "scorpio", "sagittarius", "capricorn", "aquarius", "pisces"}

		for _, sign := range signs {
			text, _ := GenerateDailyHoroscope(db, sign)
			SaveDailyPrediction(db, sign, text)
		}
	})

	c.Start()
}

func runDailyUpdate(db *pgxpool.Pool) {
	signs := []string{"aries", "taurus", "gemini", "cancer", "leo", "virgo", "libra", "scorpio", "sagittarius", "capricorn", "aquarius", "pisces"}

	log.Println("Начинаю массовую генерацию гороскопов через DeepSeek...")

	for _, sign := range signs {
		text, err := GenerateDailyHoroscope(dbpool, sign)
		if err != nil {
			log.Printf("Ошибка генерации для %s: %v", sign, err)
			continue
		}

		err = SaveDailyPrediction(db, sign, text)
		if err != nil {
			log.Printf("Ошибка сохранения в базу для %s: %v", sign, err)
		} else {
			log.Printf("Гороскоп для %s успешно обновлен!", sign)
		}
	}
	log.Println("Массовая генерация завершена успешно.")
}
