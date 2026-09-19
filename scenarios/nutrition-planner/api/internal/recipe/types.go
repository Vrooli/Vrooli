package recipe

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Recipe struct {
	ID, WorkspaceID                                          string
	Revision                                                 int64
	Name, Notes, SourceURL, SourceType, OriginalText, Status string
	CanonicalYield, ServingUnit                              string
	Ingredients                                              []Ingredient
	CreatedAt, UpdatedAt                                     time.Time
	IdempotencyKey, RequestHash                              string
	Methods                                                  []Method
	Groups, RequiredAppliances                               []string
	AllergenEvidence                                         map[string]string
}
type (
	Ingredient struct {
		ID, Name, Amount, Unit, Preparation string
		Discrete                            bool
	}
	CreateInput struct {
		WorkspaceID, Name, Notes, SourceURL, SourceType, OriginalText, IdempotencyKey, CanonicalYield, ServingUnit string
		Methods                                                                                                    []Method
		Groups, RequiredAppliances                                                                                 []string
		AllergenEvidence                                                                                           map[string]string
		Ingredients                                                                                                []Ingredient
	}
	UpdateInput struct {
		WorkspaceID, ID                                                                               string
		ExpectedRevision                                                                              int64
		Name, Notes, SourceURL, SourceType, OriginalText, IdempotencyKey, CanonicalYield, ServingUnit string
		Methods                                                                                       []Method
		Groups, RequiredAppliances                                                                    []string
		AllergenEvidence                                                                              map[string]string
		Ingredients                                                                                   []Ingredient
	}
)

type Repository interface {
	Create(context.Context, Recipe) (Recipe, error)
	List(context.Context, string) ([]Recipe, error)
	Get(context.Context, string, string) (Recipe, error)
	Update(context.Context, UpdateInput) (Recipe, error)
}
type ErrNotFound struct{ ID string }

func (e ErrNotFound) Error() string { return fmt.Sprintf("recipe %q not found", e.ID) }

type ErrForbidden struct{ ID string }

func (e ErrForbidden) Error() string { return fmt.Sprintf("recipe %q is outside the workspace", e.ID) }

type ErrConflict struct {
	ID               string
	Expected, Actual int64
}

func (e ErrConflict) Error() string {
	return fmt.Sprintf("recipe %q revision conflict: expected %d, actual %d", e.ID, e.Expected, e.Actual)
}

type ErrInvalid struct{ Field, Reason string }

func (e ErrInvalid) Error() string { return e.Field + ": " + e.Reason }

type ErrIdempotencyConflict struct{ Key string }

func (e ErrIdempotencyConflict) Error() string {
	return fmt.Sprintf("idempotency key %q was used with a different payload", e.Key)
}

func validateName(n string) error {
	if strings.TrimSpace(n) == "" {
		return ErrInvalid{"name", "must not be empty"}
	}
	return nil
}
