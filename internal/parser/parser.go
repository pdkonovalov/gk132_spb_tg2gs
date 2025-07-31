package parser

import (
	"strconv"
	"strings"
	"time"

	"github.com/pdkonovalov/gk132_spb_tg2gs/internal/domain/entity"
)

func ParseProblemMessage(message string, location *time.Location) (*entity.Problem, bool) {
	problem, ok := tryParseProblemStarted(message, location)
	if ok {
		return problem, ok
	}

	problem, ok = tryParseProblemResolved(message, location)
	if ok {
		return problem, ok
	}

	return nil, false
}

func tryParseProblemStarted(message string, location *time.Location) (*entity.Problem, bool) {
	rows := strings.Split(message, "\n")
	if len(rows) != 3 {
		return nil, false
	}

	problem := entity.Problem{}

	{
		// try parse first row
		// wait "Problem: С камеры <CameraID> <Description>"
		// or "Problem: <Description>"

		parts := strings.Split(rows[0], ": ")
		if len(parts) != 2 || parts[0] != "Problem" {
			return nil, false
		}

		small_parts := strings.Split(parts[1], " ")

		if len(small_parts) >= 3 &&
			small_parts[0] == "C" &&
			small_parts[1] == "камеры" {
			problem.CameraID = small_parts[2]

			if len(small_parts) > 3 {
				problem.Description = strings.Join(small_parts[3:], " ")
			}
		} else {
			problem.Description = parts[1]
		}
	}

	{
		// try parse second row
		// wait "Problem started at 15:04:05 on 2006.01.02"

		layout := "Problem started at 15:04:05 on 2006.01.02"

		started_at, err := time.ParseInLocation(layout, rows[1], location)
		if err != nil {
			return nil, false
		}

		problem.StartedAt = started_at
	}

	{
		// try parse third row
		// wait "Original problem ID: <ProblemID>"

		parts := strings.Split(rows[2], ": ")

		if len(parts) != 2 ||
			parts[0] != "Original problem ID" {
			return nil, false
		}

		problem.ProblemID = parts[1]
	}

	return &problem, true
}

func tryParseProblemResolved(message string, location *time.Location) (*entity.Problem, bool) {
	rows := strings.Split(message, "\n")
	if len(rows) != 3 {
		return nil, false
	}

	problem := entity.Problem{}

	{
		// try parse first row
		// wait "Resolved in 366d 0h 0m 0s: С камеры <CameraID> <Description>"
		// or "Resolved in 366d 0h 0m 0s: <Description>"

		parts := strings.Split(rows[0], ": ")
		if len(parts) != 2 || len(parts[0]) < len("Resolved in") || parts[0][:len("Resolved in")] != "Resolved in" {
			return nil, false
		}

		small_parts := strings.Split(parts[1], " ")

		if len(small_parts) >= 3 &&
			small_parts[0] == "C" &&
			small_parts[1] == "камеры" {
			problem.CameraID = small_parts[2]

			if len(small_parts) > 3 {
				problem.Description = strings.Join(small_parts[3:], " ")
			}
		} else {
			problem.Description = parts[1]
		}
	}

	{
		// try parse second row
		// wait "Problem has been resolved in 366d 0h 0m 0s at 15:04:05 on 2006.01.02"

		parts := strings.Split(rows[1], " at ")

		if len(parts) != 2 {
			return nil, false
		}

		var resolved_in time.Duration

		{
			small_parts := strings.Split(parts[0], " in ")

			if len(small_parts) != 2 {
				return nil, false
			}

			small_small_parts := strings.Split(small_parts[1], "d ")

			if len(small_small_parts) == 1 {
				resolved_in_try, err := time.ParseDuration(strings.ReplaceAll(small_parts[1], " ", ""))
				if err != nil {
					return nil, false
				}
				resolved_in = resolved_in_try
			} else if len(small_small_parts) == 2 {
				resolved_in_try, err := time.ParseDuration(strings.ReplaceAll(small_small_parts[1], " ", ""))
				if err != nil {
					return nil, false
				}
				resolved_in = resolved_in_try

				day_count_try, err := strconv.Atoi(small_small_parts[0])
				if err != nil {
					return nil, false
				}
				day_count := day_count_try

				resolved_in = resolved_in + 24*time.Hour*time.Duration(day_count)
			} else {
				return nil, false
			}
		}

		layout := "15:04:05 on 2006.01.02"

		problem.IsResolved = true

		resolved_at, err := time.ParseInLocation(layout, parts[1], location)
		if err != nil {
			return nil, false
		}

		problem.ResolvedAt = &resolved_at

		problem.StartedAt = problem.ResolvedAt.Add(-resolved_in)
	}

	{
		// try parse third row
		// wait "Original problem ID: <ProblemID>"

		parts := strings.Split(rows[2], ": ")

		if len(parts) != 2 ||
			parts[0] != "Original problem ID" {
			return nil, false
		}

		problem.ProblemID = parts[1]
	}

	return &problem, true
}
