package catalog

import (
	"context"
	"fmt"
	"strings"
	"time"

	"nutrition-planner/internal/decimalx"
)

type EvidenceKind string

const (
	EvidenceLabel    EvidenceKind = "label"
	EvidenceSource   EvidenceKind = "source"
	EvidenceUser     EvidenceKind = "user_assertion"
	EvidenceEstimate EvidenceKind = "estimate"
)

type NutrientValue struct {
	NutrientID string
	Amount     decimalx.Decimal
	Unit       string
	Basis      decimalx.Decimal
	BasisUnit  string
	Evidence   EvidenceKind
	SourceRef  string
}

type Revision struct {
	ID               string
	Revision         int64
	WorkspaceID      string
	ConceptID        string
	Name             string
	ProductName      string
	Preparation      string
	ServingQuantity  decimalx.Decimal
	ServingUnit      string
	Nutrients        []NutrientValue
	AllergenEvidence map[string]string
	SourceType       string
	SourceRef        string
	CreatedAt        time.Time
}

type Repository interface {
	Create(context.Context, Revision) (Revision, error)
	List(context.Context, string) ([]Revision, error)
	Get(context.Context, string, string, int64) (Revision, error)
}

type ErrNotFound struct {
	ID       string
	Revision int64
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("catalog revision %q/%d not found", e.ID, e.Revision)
}

type ErrInvalid struct{ Field, Reason string }

func (e ErrInvalid) Error() string { return e.Field + ": " + e.Reason }

func Validate(v Revision) error {
	if strings.TrimSpace(v.WorkspaceID) == "" {
		return ErrInvalid{"workspace_id", "must not be empty"}
	}
	if strings.TrimSpace(v.Name) == "" {
		return ErrInvalid{"name", "must not be empty"}
	}
	if strings.TrimSpace(v.ConceptID) == "" {
		return ErrInvalid{"concept_id", "must not be empty"}
	}
	if v.ServingQuantity.IsUnknown() || v.ServingQuantity.IsZero() {
		return ErrInvalid{"serving_quantity", "must be known and non-zero"}
	}
	if strings.TrimSpace(v.ServingUnit) == "" {
		return ErrInvalid{"serving_unit", "must not be empty"}
	}
	for i, n := range v.Nutrients {
		if strings.TrimSpace(n.NutrientID) == "" {
			return ErrInvalid{fmt.Sprintf("nutrients[%d].nutrient_id", i), "must not be empty"}
		}
		if n.Amount.IsUnknown() != n.Basis.IsUnknown() {
			return ErrInvalid{fmt.Sprintf("nutrients[%d]", i), "amount and basis must be known together"}
		}
		if !n.Amount.IsUnknown() && (n.Basis.IsZero() || strings.TrimSpace(n.Unit) == "" || strings.TrimSpace(n.BasisUnit) == "") {
			return ErrInvalid{fmt.Sprintf("nutrients[%d]", i), "known values require non-zero basis and units"}
		}
	}
	return nil
}
