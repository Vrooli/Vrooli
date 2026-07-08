package operatingmode

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Guard is the generic field-predicate branching model. A leaf guard compares a
// dotted field-path in a round's structured output to a value (eq/ne/gt/gte/lt/
// lte), tests membership (in/not_in), or presence (exists/not_exists); the
// composites all/any/not combine leaves into any boolean expression. `always`
// matches unconditionally (the default/fallback edge). This single vocabulary
// expresses any phase DAG with no mode-specific branch kinds — it replaces the
// closed always/payload_bool/progress_decision TransitionCondition set.
//
// The struct maps directly onto the schema's Guard oneOf: a JSON guard document
// unmarshals into exactly one shape (leaf op, membership, or a composite key).
type Guard struct {
	Op     string  `json:"op,omitempty"`
	Field  string  `json:"field,omitempty"`
	Value  any     `json:"value,omitempty"`
	Values []any   `json:"values,omitempty"`
	All    []Guard `json:"all,omitempty"`
	Any    []Guard `json:"any,omitempty"`
	Not    *Guard  `json:"not,omitempty"`
}

// Guard operator constants for the leaf forms. Composites are identified by the
// all/any/not fields rather than an op.
const (
	GuardOpAlways    = "always"
	GuardOpEq        = "eq"
	GuardOpNe        = "ne"
	GuardOpGt        = "gt"
	GuardOpGte       = "gte"
	GuardOpLt        = "lt"
	GuardOpLte       = "lte"
	GuardOpIn        = "in"
	GuardOpNotIn     = "not_in"
	GuardOpExists    = "exists"
	GuardOpNotExists = "not_exists"
)

// FieldLookup resolves a dotted field-path (e.g. `replan_needed`,
// `progress.decision`) against a round's structured output. present is false
// when any path segment is missing.
type FieldLookup interface {
	Lookup(path string) (value any, present bool)
}

// MapFieldLookup wraps a decoded JSON object (the structured result envelope)
// and resolves dotted paths by descending into nested objects. Non-map values
// encountered mid-path (Go structs such as ProgressState) are coerced through a
// JSON round-trip so struct-shaped payloads resolve identically to map-shaped
// ones.
type MapFieldLookup struct {
	root map[string]any
}

// NewMapFieldLookup builds a FieldLookup over the given structured-output map.
func NewMapFieldLookup(root map[string]any) MapFieldLookup {
	return MapFieldLookup{root: root}
}

// Lookup resolves a dotted path. Each segment must index into an object; the
// final value (which may be nil for an explicit JSON null) is returned with
// present=true.
func (m MapFieldLookup) Lookup(path string) (any, bool) {
	segments := strings.Split(path, ".")
	var cur any = m.root
	for i, seg := range segments {
		asMap, ok := coerceToMap(cur)
		if !ok {
			return nil, false
		}
		next, ok := asMap[seg]
		if !ok {
			return nil, false
		}
		if i == len(segments)-1 {
			return next, true
		}
		cur = next
	}
	return cur, true
}

// coerceToMap returns v as a map[string]any, JSON round-tripping structs so
// that struct-valued payload fields (e.g. a ProgressState stored under
// `progress`) resolve the same as decoded JSON objects.
func coerceToMap(v any) (map[string]any, bool) {
	switch t := v.(type) {
	case nil:
		return nil, false
	case map[string]any:
		return t, true
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, false
		}
		var m map[string]any
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, false
		}
		return m, true
	}
}

// Eval reports whether the guard matches the given structured output.
// Composites are recognised before leaf ops; an unset composite with no op
// (the zero Guard) never matches.
func (g Guard) Eval(lookup FieldLookup) bool {
	switch {
	case len(g.All) > 0:
		for _, sub := range g.All {
			if !sub.Eval(lookup) {
				return false
			}
		}
		return true
	case len(g.Any) > 0:
		for _, sub := range g.Any {
			if sub.Eval(lookup) {
				return true
			}
		}
		return false
	case g.Not != nil:
		return !g.Not.Eval(lookup)
	}

	switch g.Op {
	case GuardOpAlways:
		return true
	case GuardOpExists:
		return existsPositive(lookup, g.Field)
	case GuardOpNotExists:
		return !existsPositive(lookup, g.Field)
	case GuardOpEq:
		return eqPositive(lookup, g.Field, g.Value)
	case GuardOpNe:
		return !eqPositive(lookup, g.Field, g.Value)
	case GuardOpIn:
		return inPositive(lookup, g.Field, g.Values)
	case GuardOpNotIn:
		return !inPositive(lookup, g.Field, g.Values)
	case GuardOpGt, GuardOpGte, GuardOpLt, GuardOpLte:
		return compareNumeric(lookup, g.Op, g.Field, g.Value)
	default:
		return false
	}
}

// existsPositive is the base presence predicate: the field is present and its
// value is non-null. not_exists is its negation.
func existsPositive(lookup FieldLookup, field string) bool {
	value, present := lookup.Lookup(field)
	return present && value != nil
}

// eqPositive is the base equality predicate: the field is present and its
// (scalar) value equals want. ne is its negation, so an absent field is ne.
func eqPositive(lookup FieldLookup, field string, want any) bool {
	value, present := lookup.Lookup(field)
	if !present {
		return false
	}
	return scalarEqual(value, want)
}

// inPositive is the base membership predicate: the field is present and its
// value equals one of the members. not_in is its negation.
func inPositive(lookup FieldLookup, field string, members []any) bool {
	value, present := lookup.Lookup(field)
	if !present {
		return false
	}
	for _, member := range members {
		if scalarEqual(value, member) {
			return true
		}
	}
	return false
}

// compareNumeric evaluates gt/gte/lt/lte. Both operands must be numeric; a
// missing or non-numeric field does not match.
func compareNumeric(lookup FieldLookup, op, field string, want any) bool {
	value, present := lookup.Lookup(field)
	if !present {
		return false
	}
	lhs, lok := toFloat(value)
	rhs, rok := toFloat(want)
	if !lok || !rok {
		return false
	}
	switch op {
	case GuardOpGt:
		return lhs > rhs
	case GuardOpGte:
		return lhs >= rhs
	case GuardOpLt:
		return lhs < rhs
	case GuardOpLte:
		return lhs <= rhs
	default:
		return false
	}
}

// scalarEqual compares two JSON scalars. Numbers compare by float value (so an
// int seed equals a float declared value); bool and string compare by kind.
func scalarEqual(a, b any) bool {
	if af, aok := toFloat(a); aok {
		if bf, bok := toFloat(b); bok {
			return af == bf
		}
		return false
	}
	switch av := a.(type) {
	case bool:
		bv, ok := b.(bool)
		return ok && av == bv
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	default:
		return false
	}
}

// toFloat normalises the numeric types JSON decoding and Go literals produce.
// Bools and strings are not numeric.
func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

// validateGuard checks a guard's structural validity, returning an actionable
// error. It enforces the schema's shape rules in Go so the semantic validator
// can reject malformed guards with a typed message even when the raw document
// bypassed JSON-schema validation.
func validateGuard(g Guard) error {
	composites := 0
	if len(g.All) > 0 {
		composites++
	}
	if len(g.Any) > 0 {
		composites++
	}
	if g.Not != nil {
		composites++
	}
	isComposite := composites > 0
	if isComposite {
		if composites > 1 {
			return fmt.Errorf("guard combines multiple composites (all/any/not); use exactly one")
		}
		if strings.TrimSpace(g.Op) != "" || strings.TrimSpace(g.Field) != "" || g.Value != nil || len(g.Values) > 0 {
			return fmt.Errorf("composite guard must not also declare op/field/value/values")
		}
		for i, sub := range g.All {
			if err := validateGuard(sub); err != nil {
				return fmt.Errorf("all[%d]: %w", i, err)
			}
		}
		for i, sub := range g.Any {
			if err := validateGuard(sub); err != nil {
				return fmt.Errorf("any[%d]: %w", i, err)
			}
		}
		if g.Not != nil {
			if err := validateGuard(*g.Not); err != nil {
				return fmt.Errorf("not: %w", err)
			}
		}
		return nil
	}

	switch g.Op {
	case GuardOpAlways:
		if strings.TrimSpace(g.Field) != "" || g.Value != nil || len(g.Values) > 0 {
			return fmt.Errorf("always guard must not declare field/value/values")
		}
		return nil
	case GuardOpExists, GuardOpNotExists:
		if err := requireGuardField(g.Field); err != nil {
			return err
		}
		if g.Value != nil || len(g.Values) > 0 {
			return fmt.Errorf("%s guard must not declare value/values", g.Op)
		}
		return nil
	case GuardOpEq, GuardOpNe, GuardOpGt, GuardOpGte, GuardOpLt, GuardOpLte:
		if err := requireGuardField(g.Field); err != nil {
			return err
		}
		if g.Value == nil {
			return fmt.Errorf("%s guard requires a value", g.Op)
		}
		if len(g.Values) > 0 {
			return fmt.Errorf("%s guard must not declare values (use value)", g.Op)
		}
		return nil
	case GuardOpIn, GuardOpNotIn:
		if err := requireGuardField(g.Field); err != nil {
			return err
		}
		if len(g.Values) == 0 {
			return fmt.Errorf("%s guard requires a non-empty values set", g.Op)
		}
		if g.Value != nil {
			return fmt.Errorf("%s guard must not declare value (use values)", g.Op)
		}
		return nil
	case "":
		return fmt.Errorf("guard requires an op or a composite (all/any/not)")
	default:
		return fmt.Errorf("unknown guard op %q", g.Op)
	}
}

func requireGuardField(field string) error {
	if !isValidFieldPath(field) {
		return fmt.Errorf("guard field %q must be a dotted lowercase field-path", field)
	}
	return nil
}

// GuardKind returns a short display kind for a guard used by the UI transition
// rendering: the composite key (all/any/not) for composites, or the leaf op
// (always/eq/ne/in/…) for leaves. It is generic — no mode-specific vocabulary.
func GuardKind(g Guard) string {
	switch {
	case len(g.All) > 0:
		return "all"
	case len(g.Any) > 0:
		return "any"
	case g.Not != nil:
		return "not"
	}
	return g.Op
}

// GuardLabel renders a human-readable, operator-facing description of a guard
// generically from its structure, so CLI and UI emit identical strings without
// any bespoke per-mode branch vocabulary: "always", "on replan_needed = true",
// "on progress.decision in [continue, complete]", "not(…)", etc.
func GuardLabel(g Guard) string {
	switch {
	case len(g.All) > 0:
		return "all(" + joinGuardLabels(g.All) + ")"
	case len(g.Any) > 0:
		return "any(" + joinGuardLabels(g.Any) + ")"
	case g.Not != nil:
		return "not(" + GuardLabel(*g.Not) + ")"
	}
	switch g.Op {
	case GuardOpAlways:
		return "always"
	case GuardOpExists:
		return "when " + g.Field + " is set"
	case GuardOpNotExists:
		return "when " + g.Field + " is unset"
	case GuardOpEq:
		return "on " + g.Field + " = " + renderGuardValue(g.Value)
	case GuardOpNe:
		return "on " + g.Field + " ≠ " + renderGuardValue(g.Value)
	case GuardOpGt:
		return "on " + g.Field + " > " + renderGuardValue(g.Value)
	case GuardOpGte:
		return "on " + g.Field + " ≥ " + renderGuardValue(g.Value)
	case GuardOpLt:
		return "on " + g.Field + " < " + renderGuardValue(g.Value)
	case GuardOpLte:
		return "on " + g.Field + " ≤ " + renderGuardValue(g.Value)
	case GuardOpIn:
		return "on " + g.Field + " in [" + renderGuardValues(g.Values) + "]"
	case GuardOpNotIn:
		return "on " + g.Field + " not in [" + renderGuardValues(g.Values) + "]"
	default:
		return g.Op
	}
}

func joinGuardLabels(guards []Guard) string {
	parts := make([]string, 0, len(guards))
	for _, sub := range guards {
		parts = append(parts, GuardLabel(sub))
	}
	return strings.Join(parts, ", ")
}

func renderGuardValues(values []any) string {
	parts := make([]string, 0, len(values))
	for _, v := range values {
		parts = append(parts, renderGuardValue(v))
	}
	return strings.Join(parts, ", ")
}

// renderGuardValue renders a JSON scalar for display. Numbers drop a trailing
// ".0" so an integer-valued float reads as "3", not "3.0".
func renderGuardValue(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		if f, ok := toFloat(v); ok {
			return strconv.FormatFloat(f, 'f', -1, 64)
		}
		return fmt.Sprintf("%v", v)
	}
}

// isValidFieldPath enforces the schema's FieldPath pattern
// ^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)*$ in Go.
func isValidFieldPath(field string) bool {
	if field == "" {
		return false
	}
	for _, segment := range strings.Split(field, ".") {
		if segment == "" {
			return false
		}
		for i, r := range segment {
			switch {
			case r >= 'a' && r <= 'z':
			case i > 0 && r >= '0' && r <= '9':
			case i > 0 && r == '_':
			default:
				return false
			}
		}
	}
	return true
}
