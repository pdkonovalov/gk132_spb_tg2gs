package repository

import "github.com/pdkonovalov/gk132_spb_tg2gs/internal/domain/entity"

type Repository interface {
	Create(*entity.Problem) error
	Update(*entity.Problem) error
}
