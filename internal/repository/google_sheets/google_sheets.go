package google_sheets

import (
	"context"
	"fmt"

	"github.com/pdkonovalov/gk132_spb_tg2gs/internal/config"
	"github.com/pdkonovalov/gk132_spb_tg2gs/internal/domain/entity"

	freedb "github.com/FreeLeh/GoFreeDB"
	"github.com/FreeLeh/GoFreeDB/google/auth"
)

type problemGS struct {
	ProblemID   string `db:"ID проблемы (автоматически)"`
	CameraID    string `db:"ID камеры (автоматически)"`
	Description string `db:"Описание проблемы (автоматически)"`
	StartedAt   string `db:"Время возникновения проблемы (автоматически)"`
	IsResolved  string `db:"Статус проблемы (автоматически)"`
	ResolvedAt  string `db:"Время устранения проблемы (автоматически)"`
}

func convertProblemToStruct(problem *entity.Problem) *problemGS {
	if problem == nil {
		return nil
	}

	var is_resolved string
	if problem.IsResolved {
		is_resolved = "устранена"
	} else {
		is_resolved = "актуальна"
	}

	var resolved_at string
	if problem.ResolvedAt != nil {
		resolved_at = problem.ResolvedAt.Format("02.01.2006 15:04:05")
	}

	return &problemGS{
		ProblemID:   problem.ProblemID,
		CameraID:    problem.CameraID,
		Description: problem.Description,
		StartedAt:   problem.StartedAt.Format("02.01.2006 15:04:05"),
		IsResolved:  is_resolved,
		ResolvedAt:  resolved_at,
	}
}

func convertProblemToMap(problem *entity.Problem) map[string]interface{} {
	if problem == nil {
		return nil
	}

	var is_resolved string
	if problem.IsResolved {
		is_resolved = "устранена"
	} else {
		is_resolved = "актуальна"
	}

	var resolved_at string
	if problem.ResolvedAt != nil {
		resolved_at = problem.ResolvedAt.Format("02.01.2006 15:04:05")
	}

	return map[string]interface{}{
		"ID проблемы (автоматически)":                  problem.ProblemID,
		"ID камеры (автоматически)":                    problem.CameraID,
		"Описание проблемы (автоматически)":            problem.Description,
		"Время возникновения проблемы (автоматически)": problem.StartedAt.Format("02.01.2006 15:04:05"),
		"Статус проблемы (автоматически)":              is_resolved,
		"Время устранения проблемы (автоматически)":    resolved_at,
	}
}

type google_sheets struct {
	row_store freedb.GoogleSheetRowStore
}

func New(cfg *config.Config) (*google_sheets, error) {
	gs := google_sheets{}

	auth, err := auth.NewServiceFromFile(
		cfg.GoogleSheetsServiceAccountCredentialsFile,
		freedb.GoogleAuthScopes,
		auth.ServiceConfig{},
	)

	if err != nil {
		return nil, fmt.Errorf("Failed create new google sheets service: %s", err)
	}

	row_store := freedb.NewGoogleSheetRowStore(
		auth,
		cfg.GoogleSheetsSpreadsheetID,
		cfg.GoogleSheetsSheet,
		freedb.GoogleSheetRowStoreConfig{Columns: []string{"ID проблемы (автоматически)", "ID камеры (автоматически)", "Описание проблемы (автоматически)", "Время возникновения проблемы (автоматически)", "Статус проблемы (автоматически)", "Время устранения проблемы (автоматически)"}},
	)

	gs.row_store = *row_store

	return &gs, nil
}

func (gs *google_sheets) Close(ctx context.Context) error {
	return gs.row_store.Close(ctx)
}

func (gs *google_sheets) Create(problem *entity.Problem) error {
	if problem == nil {
		return fmt.Errorf("Failed create problem, problem is nil")
	}

	count, err := gs.row_store.
		Count().
		Where("ID проблемы (автоматически) = ?", problem.ProblemID).
		Exec(context.Background())

	if err != nil {
		return fmt.Errorf("Failed check problem is exists before create: %s", err)
	}

	if count > 0 {
		return fmt.Errorf("Failed create problem, problem with '%s' id alredy exists", problem.ProblemID)
	}

	err = gs.row_store.Insert(convertProblemToStruct(problem)).Exec(context.Background())
	if err != nil {
		return fmt.Errorf("Failed create problem '%v', error: %s", *problem, err)
	}
	return nil
}

func (gs *google_sheets) Update(problem *entity.Problem) error {
	if problem == nil {
		return fmt.Errorf("Failed update problem, problem is nil")
	}

	count, err := gs.row_store.
		Count().
		Where("ID проблемы (автоматически) = ?", problem.ProblemID).
		Exec(context.Background())

	if err != nil {
		return fmt.Errorf("Failed check problem is exists before update: %s", err)
	}

	if count == 0 {
		return gs.Create(problem)
	}

	if count != 1 {
		return fmt.Errorf("Failed update problem, more than one problem with '%s' id is exists", problem.ProblemID)
	}

	err = gs.row_store.
		Update(convertProblemToMap(problem)).
		Where("ID проблемы (автоматически) = ?", problem.ProblemID).
		Exec(context.Background())
	if err != nil {
		return fmt.Errorf("Failed update problem '%v', error: %s", *problem, err)
	}
	return nil
}
