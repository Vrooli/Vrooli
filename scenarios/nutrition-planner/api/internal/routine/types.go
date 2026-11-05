package routine

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"nutrition-planner/internal/decimalx"
)

type Template struct {
	ID          string
	Revision    int64
	WorkspaceID string
	SlotName    string
	RecipeID    string
	Quantity    decimalx.Decimal
	Weekdays    []int
	StartDate   string
	EndDate     string
	Mode        string // fixed, flexible, open, social
	Active      bool
	CreatedAt   time.Time
}

type Occurrence struct {
	Date             string
	SlotName         string
	RecipeID         string
	Quantity         decimalx.Decimal
	TemplateID       string
	TemplateRevision int64
	Mode             string
}

type Repository interface {
	Create(context.Context, Template) (Template, error)
	List(context.Context, string) ([]Template, error)
	Get(context.Context, string, string, int64) (Template, error)
	Update(context.Context, UpdateInput) (Template, error)
}

type UpdateInput struct {
	WorkspaceID, ID          string
	ExpectedRevision         int64
	SlotName, RecipeID       string
	Quantity                 decimalx.Decimal
	Weekdays                 []int
	StartDate, EndDate, Mode string
	Active                   bool
}

type ErrNotFound struct {
	ID       string
	Revision int64
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("routine template %q/%d not found", e.ID, e.Revision)
}

type ErrConflict struct {
	ID               string
	Expected, Actual int64
}

func (e ErrConflict) Error() string {
	return fmt.Sprintf("routine template %q revision conflict: expected %d, actual %d", e.ID, e.Expected, e.Actual)
}

type ErrInvalid struct{ Field, Reason string }

func (e ErrInvalid) Error() string { return e.Field + ": " + e.Reason }

func Validate(t Template) error {
	if strings.TrimSpace(t.WorkspaceID) == "" {
		return ErrInvalid{"workspace_id", "must not be empty"}
	}
	if strings.TrimSpace(t.SlotName) == "" {
		return ErrInvalid{"slot_name", "must not be empty"}
	}
	if t.Mode != "fixed" && t.Mode != "flexible" && t.Mode != "open" && t.Mode != "social" {
		return ErrInvalid{"mode", "must be fixed, flexible, open, or social"}
	}
	if t.Mode == "fixed" && strings.TrimSpace(t.RecipeID) == "" {
		return ErrInvalid{"recipe_id", "fixed templates require a recipe"}
	}
	if t.Quantity.IsUnknown() || t.Quantity.IsZero() {
		return ErrInvalid{"quantity", "must be known and non-zero"}
	}
	start, err := time.Parse("2006-01-02", t.StartDate)
	if err != nil {
		return ErrInvalid{"start_date", "must be YYYY-MM-DD"}
	}
	if t.EndDate != "" {
		end, e := time.Parse("2006-01-02", t.EndDate)
		if e != nil || end.Before(start) {
			return ErrInvalid{"end_date", "must be on or after start_date"}
		}
	}
	if len(t.Weekdays) == 0 {
		return ErrInvalid{"weekdays", "at least one weekday is required"}
	}
	seen := map[int]bool{}
	for _, d := range t.Weekdays {
		if d < 0 || d > 6 {
			return ErrInvalid{"weekdays", "values must be 0 through 6"}
		}
		if seen[d] {
			return ErrInvalid{"weekdays", "duplicate weekday"}
		}
		seen[d] = true
	}
	return nil
}

func Generate(templates []Template, from, to string) ([]Occurrence, error) {
	start, err := time.Parse("2006-01-02", from)
	if err != nil {
		return nil, fmt.Errorf("from: invalid date")
	}
	end, err := time.Parse("2006-01-02", to)
	if err != nil {
		return nil, fmt.Errorf("to: invalid date")
	}
	if end.Before(start) {
		return nil, fmt.Errorf("to: must be on or after from")
	}
	if end.Sub(start) > 6*24*time.Hour {
		return nil, fmt.Errorf("horizon: routine generation is bounded to seven days")
	}
	var out []Occurrence
	for _, t := range templates {
		if !t.Active {
			continue
		}
		if err := Validate(t); err != nil {
			return nil, err
		}
		for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
			if !contains(t.Weekdays, int(day.Weekday())) || day.Format("2006-01-02") < t.StartDate || (t.EndDate != "" && day.Format("2006-01-02") > t.EndDate) {
				continue
			}
			out = append(out, Occurrence{Date: day.Format("2006-01-02"), SlotName: t.SlotName, RecipeID: t.RecipeID, Quantity: t.Quantity, TemplateID: t.ID, TemplateRevision: t.Revision, Mode: t.Mode})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Date != out[j].Date {
			return out[i].Date < out[j].Date
		}
		if out[i].SlotName != out[j].SlotName {
			return out[i].SlotName < out[j].SlotName
		}
		return out[i].TemplateID < out[j].TemplateID
	})
	return out, nil
}

func contains(values []int, want int) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
