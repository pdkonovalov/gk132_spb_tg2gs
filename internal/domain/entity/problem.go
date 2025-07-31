package entity

import "time"

type Problem struct {
	ProblemID   string
	CameraID    string
	Description string
	StartedAt   time.Time
	IsResolved  bool
	ResolvedAt  *time.Time
}
