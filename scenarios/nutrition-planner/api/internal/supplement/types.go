package supplement

import (
	"context"
	"fmt"
	"strings"
	"time"

	"nutrition-planner/internal/decimalx"
)

type Schedule struct {
	ID                string
	Revision          int64
	WorkspaceID       string
	ProductRevisionID string
	Dose              decimalx.Decimal
	DoseUnit          string
	Weekdays          []int
	StartDate         string
	EndDate           string
	Paused            bool
	Confirmed         bool
	CreatedAt         time.Time
}

type CreateInput struct {
	WorkspaceID, ProductRevisionID string
	Dose                           decimalx.Decimal
	DoseUnit, StartDate, EndDate   string
	Weekdays                       []int
	Confirmed                      bool
}

type UpdateInput struct {
	WorkspaceID, ID              string
	ExpectedRevision             int64
	Dose                         decimalx.Decimal
	DoseUnit, StartDate, EndDate string
	Weekdays                     []int
	Paused, Confirmed            bool
}

type Repository interface {
	Create(context.Context, Schedule) (Schedule, error)
	List(context.Context, string) ([]Schedule, error)
	Get(context.Context, string, string, int64) (Schedule, error)
	Update(context.Context, UpdateInput) (Schedule, error)
}

type ErrNotFound struct {
	ID       string
	Revision int64
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("supplement schedule %q/%d not found", e.ID, e.Revision)
}

type ErrConflict struct {
	ID               string
	Expected, Actual int64
}

func (e ErrConflict) Error() string {
	return fmt.Sprintf("supplement schedule %q revision conflict: expected %d, actual %d", e.ID, e.Expected, e.Actual)
}

type ErrInvalid struct{ Field, Reason string }

func (e ErrInvalid) Error() string { return e.Field + ": " + e.Reason }

func Validate(s Schedule) error {
	if strings.TrimSpace(s.WorkspaceID) == "" {
		return ErrInvalid{"workspace_id", "must not be empty"}
	}
	if strings.TrimSpace(s.ProductRevisionID) == "" {
		return ErrInvalid{"product_revision_id", "must not be empty"}
	}
	if s.Dose.IsUnknown() || s.Dose.IsZero() {
		return ErrInvalid{"dose", "must be known and non-zero"}
	}
	if strings.TrimSpace(s.DoseUnit) == "" {
		return ErrInvalid{"dose_unit", "must not be empty"}
	}
	start, err := time.Parse("2006-01-02", s.StartDate)
	if err != nil {
		return ErrInvalid{"start_date", "must be YYYY-MM-DD"}
	}
	if s.EndDate != "" {
		end, e := time.Parse("2006-01-02", s.EndDate)
		if e != nil || end.Before(start) {
			return ErrInvalid{"end_date", "must be a date on or after start_date"}
		}
	}
	if len(s.Weekdays) == 0 {
		return ErrInvalid{"weekdays", "at least one weekday is required"}
	}
	seen := map[int]bool{}
	for _, day := range s.Weekdays {
		if day < 0 || day > 6 {
			return ErrInvalid{"weekdays", "values must be 0 through 6"}
		}
		if seen[day] {
			return ErrInvalid{"weekdays", "duplicate weekday"}
		}
		seen[day] = true
	}
	return nil
}

func AppliesOn(s Schedule, date time.Time) bool {
	if s.Paused || !s.Confirmed || Validate(s) != nil {
		return false
	}
	day := date.UTC().Truncate(24 * time.Hour)
	start, _ := time.ParseInLocation("2006-01-02", s.StartDate, time.UTC)
	if day.Before(start) {
		return false
	}
	if s.EndDate != "" {
		end, _ := time.ParseInLocation("2006-01-02", s.EndDate, time.UTC)
		if day.After(end) {
			return false
		}
	}
	for _, weekday := range s.Weekdays {
		if int(day.Weekday()) == weekday {
			return true
		}
	}
	return false
}
