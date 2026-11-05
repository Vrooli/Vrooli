package cleanup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vrooli/api-core/cleanupplan"
)

// FrozenPlan is the small control-plane projection of the node's uninstall
// plan. Raw artifact entries are intentional: the node owns platform-specific
// artifact fields, while Bridge must preserve the exact resolved lists it
// authorized and display them without re-discovering the host.
type FrozenPlan struct {
	PlanHash        string
	Remove          []json.RawMessage
	Keep            []json.RawMessage
	CannotAttribute []json.RawMessage
}

func ParseFrozenPlan(raw []byte) (FrozenPlan, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return FrozenPlan{}, fmt.Errorf("decode cleanup plan: %w", err)
	}
	remove, err := section(object, "remove")
	if err != nil {
		return FrozenPlan{}, err
	}
	keep, err := section(object, "keep")
	if err != nil {
		return FrozenPlan{}, err
	}
	cannot, err := section(object, "cannot_attribute")
	if err != nil {
		return FrozenPlan{}, err
	}
	computed := cleanupplan.HashResolvedArtifacts(remove, keep, cannot)
	claimed := stringField(object, "plan_hash")
	if claimed != "" && claimed != computed {
		return FrozenPlan{}, fmt.Errorf("plan_hash: claimed %q does not match resolved artifact lists %q", claimed, computed)
	}
	return FrozenPlan{PlanHash: computed, Remove: remove, Keep: keep, CannotAttribute: cannot}, nil
}

// ComputePlanHash is exported for table tests and adapters that already hold
// the three resolved sections separately. It uses the same shared primitive
// as the local uninstall planner.
func ComputePlanHash(remove, keep, cannotAttribute []json.RawMessage) string {
	return cleanupplan.HashResolvedArtifacts(remove, keep, cannotAttribute)
}

type Receipt struct {
	PlanID          string
	PlanHash        string
	Target          string
	Scope           string
	AuthorizingUser string
	Removed         []json.RawMessage
	Preserved       []json.RawMessage
	CannotAttribute []json.RawMessage
	Attempts        []json.RawMessage
}

// ValidateReceipt checks the durable receipt contract before it is stored on
// the operation. A receipt is evidence, not a success string: it must name the
// exact frozen plan, target, operator, all three artifact outcomes, and every
// apply attempt.
func ValidateReceipt(raw []byte, op Operation) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return fmt.Errorf("decode cleanup receipt: %w", err)
	}
	for _, field := range []string{"plan_id", "plan_hash", "target", "scope", "removed", "preserved", "cannot_attribute", "attempts"} {
		if _, ok := object[field]; !ok {
			return fmt.Errorf("receipt: required field %q is missing", field)
		}
	}
	if value := stringField(object, "plan_id"); value != op.ID {
		return fmt.Errorf("receipt: plan_id %q does not match operation %q", value, op.ID)
	}
	if value := stringField(object, "plan_hash"); value != op.PlanHash {
		return fmt.Errorf("receipt: plan_hash %q does not match frozen plan %q", value, op.PlanHash)
	}
	if value := stringField(object, "target"); value != op.Target {
		return fmt.Errorf("receipt: target %q does not match operation target %q", value, op.Target)
	}
	if value := stringField(object, "scope"); value != op.Scope {
		return fmt.Errorf("receipt: scope %q does not match operation scope %q", value, op.Scope)
	}
	if value := strings.TrimSpace(stringField(object, "authorizing_user")); value == "" || value != op.OperatorID {
		return fmt.Errorf("receipt: authorizing_user does not match operation operator")
	}
	for _, field := range []string{"removed", "preserved", "cannot_attribute", "attempts"} {
		var entries []json.RawMessage
		if err := json.Unmarshal(object[field], &entries); err != nil {
			return fmt.Errorf("receipt: %s must be an array: %w", field, err)
		}
	}
	return nil
}

func section(object map[string]json.RawMessage, name string) ([]json.RawMessage, error) {
	raw, ok := object[name]
	if !ok {
		return nil, nil
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(compact(raw), &entries); err != nil {
		return nil, fmt.Errorf("plan: %s must be an array: %w", name, err)
	}
	return entries, nil
}

func stringField(object map[string]json.RawMessage, name string) string {
	var value string
	_ = json.Unmarshal(object[name], &value)
	return strings.TrimSpace(value)
}

func compact(raw []byte) []byte {
	var out bytes.Buffer
	if err := json.Compact(&out, raw); err != nil {
		return raw
	}
	return out.Bytes()
}
