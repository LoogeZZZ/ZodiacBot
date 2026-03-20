package main

type DeepSeekRequest struct {
	Model    string      `json:"model"`
	Messages []AIMessage `json:"messages"`
}

type AIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type DeepSeekResponse struct {
	Choices []struct {
		Message AIMessage `json:"message"`
	} `json:"choices"`
}

type User struct {
	ID         int64  // Telegram User ID
	Username   string // Никнейм пользователя
	ZodiacSign string // Технический код знака (aries, leo...)
}

var ZodiacNames = map[string]string{
	"aries":       "Овен ♈",
	"taurus":      "Телец ♉",
	"gemini":      "Близнецы ♊",
	"cancer":      "Рак ♋",
	"leo":         "Лев ♌",
	"virgo":       "Дева ♍",
	"libra":       "Весы ♎",
	"scorpio":     "Скорпион ♏",
	"sagittarius": "Стрелец ♐",
	"capricorn":   "Козерог ♑",
	"aquarius":    "Водолей ♒",
	"pisces":      "Рыбы ♓",
}

type DashboardData struct {
	UsersCount      int
	NextUpdate      string
	CurrentPrompt   string
	DailyHoroscopes map[string]string
	User            []User
}
