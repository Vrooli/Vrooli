// Package categories owns the operator-defined category set and append-only
// classification evidence. It deliberately has no triage dependency.
package categories

import (
	"context"
	"fmt"
	"time"
)

const UncategorizedName = "uncategorized"

type Category struct {
	ID          string
	Name        string
	Description string
	Reserved    bool
	CreatedAt   time.Time
	RetiredAt   *time.Time
}

func (c Category) Active() bool { return c.RetiredAt == nil }

type ClassificationState string

const (
	StateProposed      ClassificationState = "proposed"
	StateConfirmed     ClassificationState = "confirmed"
	StateOverridden    ClassificationState = "overridden"
	StateUncategorized ClassificationState = "uncategorized"
)

type Classification struct {
	ID                  string
	SignalID            string
	ProposedCategoryID  string
	ProposedConfidence  float64
	Model               string
	ConfirmedCategoryID string
	State               ClassificationState
	Reason              string
	CreatedAt           time.Time
}

type Repository interface {
	EnsureUncategorized(context.Context, time.Time) (Category, error)
	Create(context.Context, Category) (Category, error)
	List(context.Context, bool) ([]Category, error)
	Get(context.Context, string) (Category, error)
	Rename(context.Context, string, string, string) (Category, error)
	Retire(context.Context, string, time.Time) (Category, error)
	AppendClassification(context.Context, Classification) (Classification, error)
	LatestClassification(context.Context, string) (Classification, bool, error)
	LatestConfirmedByCategory(context.Context, string) ([]Classification, error)
	EnqueueReclassification(context.Context, string, string, time.Time) error
}

type ErrInvalidCategory struct{ Reason string }

func (e ErrInvalidCategory) Error() string { return fmt.Sprintf("invalid category: %s", e.Reason) }

type ErrReservedCategory struct{ ID string }

func (e ErrReservedCategory) Error() string { return fmt.Sprintf("category %q is reserved", e.ID) }

type ErrCategoryNotFound struct{ ID string }

func (e ErrCategoryNotFound) Error() string { return fmt.Sprintf("category %q not found", e.ID) }
