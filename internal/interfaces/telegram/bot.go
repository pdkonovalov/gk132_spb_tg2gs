package telegram

import (
	"fmt"
	"time"

	"github.com/pdkonovalov/gk132_spb_tg2gs/internal/config"
	"github.com/pdkonovalov/gk132_spb_tg2gs/internal/domain/repository"
	"github.com/pdkonovalov/gk132_spb_tg2gs/internal/interfaces/telegram/handler"
	custom_middleware "github.com/pdkonovalov/gk132_spb_tg2gs/internal/interfaces/telegram/middleware"

	tele "gopkg.in/telebot.v4"
	"gopkg.in/telebot.v4/middleware"
)

func New(
	cfg *config.Config,
	repo repository.Repository,
) (*tele.Bot, error) {
	b, err := tele.NewBot(tele.Settings{
		Token:  cfg.TelegramBotToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		return nil, err
	}

	err = setupMiddleware(cfg, b)
	if err != nil {
		return nil, err
	}

	err = setupRouter(cfg, b, repo)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func setupMiddleware(
	cfg *config.Config,
	b *tele.Bot,
) error {
	b.Use(middleware.Recover())

	if cfg.LogLevel == config.LogLevelDebug {
		b.Use(middleware.Logger())
	}

	auth_middleware, err := custom_middleware.NewAuthMiddleware(cfg)
	if err != nil {
		return fmt.Errorf("Failed create auth middleware: %s", err)
	}
	b.Use(auth_middleware.Auth)

	return nil
}

func setupRouter(
	cfg *config.Config,
	b *tele.Bot,
	repo repository.Repository,
) error {
	handler, err := handler.New(cfg, repo)
	if err != nil {
		return fmt.Errorf("Failed create handler: %s", err)
	}

	b.Handle(tele.OnText, handler.FetchProblem)
	b.Handle(tele.OnChannelPost, handler.FetchProblem)

	return nil
}
