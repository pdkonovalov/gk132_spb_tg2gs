package handler

import (
	"fmt"
	"time"

	"github.com/pdkonovalov/gk132_spb_tg2gs/internal/config"
	"github.com/pdkonovalov/gk132_spb_tg2gs/internal/domain/repository"
	"github.com/pdkonovalov/gk132_spb_tg2gs/internal/parser"

	tele "gopkg.in/telebot.v4"
)

type Handler struct {
	location   *time.Location
	repository repository.Repository
}

func New(
	cfg *config.Config,
	repo repository.Repository,
) (*Handler, error) {
	location, err := time.LoadLocation(cfg.TelegramTimezone)
	if err != nil {
		return nil, err
	}
	return &Handler{
		location:   location,
		repository: repo,
	}, nil
}

func (h *Handler) FetchProblem(c tele.Context) error {
	message := c.Message().Text

	problem, ok := parser.ParseProblemMessage(message, h.location)
	if !ok {
		return fmt.Errorf("Message '%s' is not valid problem message", message)
	}

	if problem == nil {
		return fmt.Errorf("Message '%s' is valid problem message, but parsed problem is nil", message)
	}

	if !problem.IsResolved {
		err := h.repository.Create(problem)
		if err != nil {
			return err
		}
	} else {
		err := h.repository.Update(problem)
		if err != nil {
			return err
		}
	}

	return nil
}
