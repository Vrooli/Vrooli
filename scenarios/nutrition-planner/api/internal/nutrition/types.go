// Package nutrition contains the deterministic, evidence-aware nutrition core.
package nutrition

import (
	"context"
	"fmt"
	"strings"
	"time"

	"nutrition-planner/internal/decimalx"
)

type Contribution struct {
	NutrientID string
	Amount     decimalx.Decimal
	Basis      decimalx.Decimal
	Quantity   decimalx.Decimal
	Unit       string
	BasisUnit  string
	Source     string
}

type Result struct {
	NutrientID      string
	Known           decimalx.Decimal
	Complete        bool
	Unresolved      []string
	EvidenceSources []string
}

type EvaluationStatus string

const (
	Pass          EvaluationStatus = "pass"
	Fail          EvaluationStatus = "fail"
	Unknown       EvaluationStatus = "unknown"
	NotApplicable EvaluationStatus = "not_applicable"
)

type Target struct {
	ID            string
	Revision      int64
	WorkspaceID   string
	NutrientID    string
	Lower         decimalx.Decimal
	Upper         decimalx.Decimal
	Period        string
	Scope         string
	Enforcement   string
	Provenance    string
	EffectiveFrom time.Time
	EffectiveTo   *time.Time
	Active        bool
}

type TargetRepository interface {
	Create(context.Context, Target) (Target, error)
	List(context.Context, string) ([]Target, error)
	Get(context.Context, string, string, int64) (Target, error)
}

type TargetNotFound struct {
	ID       string
	Revision int64
}

func (e TargetNotFound) Error() string {
	return fmt.Sprintf("target %q/%d not found", e.ID, e.Revision)
}

func ValidateTarget(t Target) error {
	if strings.TrimSpace(t.WorkspaceID) == "" {
		return fmt.Errorf("workspace_id: must not be empty")
	}
	if strings.TrimSpace(t.NutrientID) == "" {
		return fmt.Errorf("nutrient_id: must not be empty")
	}
	if t.Lower.IsUnknown() && t.Upper.IsUnknown() {
		return fmt.Errorf("target: at least one bound is required")
	}
	if !t.Lower.IsUnknown() && !t.Upper.IsUnknown() {
		if c, _ := decimalx.Compare(t.Lower, t.Upper); c > 0 {
			return fmt.Errorf("target: lower bound must not exceed upper bound")
		}
	}
	if t.Period != "local_day" && t.Period != "rolling_days" && t.Period != "calendar_week" {
		return fmt.Errorf("period: unsupported evaluation period")
	}
	if t.Scope != "selected_meal" && t.Scope != "planned_day" && t.Scope != "recorded_so_far" && t.Scope != "expected_day" && t.Scope != "selected_week" {
		return fmt.Errorf("scope: unsupported contribution scope")
	}
	if t.Enforcement != "required" && t.Enforcement != "preferred" && t.Enforcement != "informational" {
		return fmt.Errorf("enforcement: unsupported level")
	}
	if t.EffectiveFrom.IsZero() {
		return fmt.Errorf("effective_from: must be set")
	}
	return nil
}

type Evaluation struct {
	Status EvaluationStatus
	Reason string
}

type Intake struct {
	Date       string
	NutrientID string
	Planned    decimalx.Decimal
	Actual     decimalx.Decimal
	Recorded   bool
	Past       bool
}

type IntakeEvent struct {
	ID, WorkspaceID, Date, RecipeID, NutrientID, Unit, Reason, CorrectionOf string
	RecipeRevision                                                          int64
	Amount                                                                  decimalx.Decimal
	timeRecorded                                                            time.Time
}

func (e IntakeEvent) RecordedAt() time.Time { return e.timeRecorded }

type IntakeRepository interface {
	Append(context.Context, string, IntakeEvent) error
	List(context.Context, string) ([]IntakeEvent, error)
}

// AggregateScope selects one contribution for each occurrence. In expected-day
// views an explicit intake replaces its plan; an unrecorded past occurrence is
// unresolved rather than silently counted as planned food.
func AggregateScope(nutrientID, scope string, intakes []Intake) (Result, error) {
	result := Result{NutrientID: nutrientID, Known: decimalx.KnownInt(0), Complete: true}
	for _, intake := range intakes {
		if intake.NutrientID != nutrientID {
			continue
		}
		var value decimalx.Decimal
		switch scope {
		case "selected_meal", "planned_day", "selected_week":
			value = intake.Planned
		case "recorded_so_far":
			if !intake.Recorded {
				continue
			}
			value = intake.Actual
		case "expected_day":
			if intake.Recorded {
				value = intake.Actual
			} else if intake.Past {
				value = decimalx.Unknown
			} else {
				value = intake.Planned
			}
		default:
			return Result{}, fmt.Errorf("scope: unsupported contribution scope %q", scope)
		}
		if value.IsUnknown() {
			result.Complete = false
			result.Unresolved = append(result.Unresolved, intake.Date)
			continue
		}
		result.Known, _ = decimalx.Add(result.Known, value)
	}
	return result, nil
}

func Scale(c Contribution) (decimalx.Decimal, error) {
	if c.Amount.IsUnknown() || c.Basis.IsUnknown() || c.Quantity.IsUnknown() {
		return decimalx.Unknown, nil
	}
	if c.Basis.IsZero() {
		return decimalx.Unknown, fmt.Errorf("%s basis must not be zero", c.NutrientID)
	}
	if c.Unit == "" || c.BasisUnit == "" || c.Unit != c.BasisUnit {
		return decimalx.Unknown, fmt.Errorf("%s quantity and basis units must be compatible", c.NutrientID)
	}
	ratio, err := decimalx.Div(c.Quantity, c.Basis)
	if err != nil {
		return decimalx.Unknown, err
	}
	return decimalx.Mul(c.Amount, ratio)
}

func Aggregate(nutrientID string, contributions []Contribution) (Result, error) {
	total := decimalx.KnownInt(0)
	result := Result{NutrientID: nutrientID, Complete: true}
	for _, c := range contributions {
		if c.NutrientID != nutrientID {
			continue
		}
		v, err := Scale(c)
		if err != nil {
			return Result{}, err
		}
		if v.IsUnknown() {
			result.Complete = false
			result.Unresolved = append(result.Unresolved, c.Source)
			continue
		}
		total, err = decimalx.Add(total, v)
		if err != nil {
			return Result{}, err
		}
		if c.Source != "" {
			result.EvidenceSources = append(result.EvidenceSources, c.Source)
		}
	}
	result.Known = total
	return result, nil
}

func Evaluate(result Result, target Target) Evaluation {
	if target.NutrientID == "" || target.NutrientID != result.NutrientID {
		return Evaluation{Status: NotApplicable, Reason: "target does not apply to this nutrient"}
	}
	if !target.Lower.IsUnknown() {
		below, _ := decimalx.Compare(result.Known, target.Lower)
		if below < 0 {
			if result.Complete {
				return Evaluation{Status: Fail, Reason: "complete total is below the lower bound"}
			}
			return Evaluation{Status: Unknown, Reason: "known subtotal is below the lower bound and unresolved contributions remain"}
		}
	}
	if !target.Upper.IsUnknown() {
		over, _ := decimalx.Compare(result.Known, target.Upper)
		if over > 0 {
			return Evaluation{Status: Fail, Reason: "known subtotal exceeds the upper bound"}
		}
		if !result.Complete {
			return Evaluation{Status: Unknown, Reason: "unresolved contributions could exceed the upper bound"}
		}
	}
	if !result.Complete {
		return Evaluation{Status: Unknown, Reason: "target is only partially evaluated"}
	}
	return Evaluation{Status: Pass, Reason: "complete total is within the configured bounds"}
}
