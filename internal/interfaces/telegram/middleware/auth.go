package middleware

import (
	"github.com/pdkonovalov/gk132_spb_tg2gs/internal/config"

	tele "gopkg.in/telebot.v4"
)

type AuthMiddleware struct {
	chatID int64
}

func NewAuthMiddleware(
	cfg *config.Config,
) (*AuthMiddleware, error) {
	return &AuthMiddleware{
		chatID: cfg.TelegramChatID,
	}, nil
}

func (m *AuthMiddleware) Auth(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		if c.Chat().ID == m.chatID {
			return next(c)
		}
		return nil
	}
}
