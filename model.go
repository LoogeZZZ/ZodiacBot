package main

// User описывает данные пользователя в системе.
type User struct {
	ID         int64  // Telegram User ID
	Username   string // Никнейм пользователя
	ZodiacSign string // Технический код знака (aries, leo...)
}

// ZodiacNames сопоставляет технический код с красивым названием.
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
