// Package paramschema compiles and validates the small JSON-Schema subset used
// by capability manifests for operation-specific parameters.
//
// Rationale: operation parameters used to arrive as typed fields on AIParams,
// which meant every new operation's knobs required a proto change, a paramsMap
// branch, and a hand-written validator. The capability manifest now owns each
// operation's parameter contract as JSON Schema, and this package enforces it at
// the submit edge.
//
// Supported keywords (deliberately small):
//
//	type (object|number|integer|boolean|string|array)
//	properties, required, additionalProperties (false)
//	minimum, maximum
//	enum
//	items (for arrays), with the same scalar keywords
//	maxItems (for arrays)
//	minLength, maxLength (for strings, counted in characters)
//	requiresTrue (map field -> [boolean fields that must be true when <field> is present])
//	withinFrame (true: an object with x,y,width,height has positive area inside 0..1)
//
// `requiresTrue` and `withinFrame` are the two non-standard keywords. They exist
// because real operation rules are relational ("inpaint crop controls require
// crop_to_mask"; "the removal rectangle must stay in the frame"), which plain
// JSON Schema cannot express without if/then. They are intentionally narrow and
// validated like every other keyword.
//
// This is intentionally not a general JSON-Schema implementation. If an
// operation needs a keyword this package does not support, add it here with a
// test rather than silently ignoring it — an ignored constraint is worse than a
// rejected one.
package paramschema

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode/utf8"
)

// Schema is a compiled parameter contract for one operation.
type Schema struct {
	raw map[string]any
}

// Compile parses and structurally validates a schema document.
func Compile(raw []byte) (*Schema, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("paramschema: empty schema")
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("paramschema: parse: %w", err)
	}
	if err := checkSchemaShape(doc, "schema"); err != nil {
		return nil, err
	}
	return &Schema{raw: doc}, nil
}

// Validate checks values against the schema. Unknown keys are rejected when the
// schema sets additionalProperties:false (the manifest convention).
func (s *Schema) Validate(values map[string]any) error {
	return validateObject(values, s.raw, "")
}

func checkSchemaShape(sch map[string]any, path string) error {
	if t, ok := sch["type"]; ok {
		ts, ok := t.(string)
		if !ok {
			return fmt.Errorf("paramschema: %s.type must be a string", path)
		}
		switch ts {
		case "object", "number", "integer", "boolean", "string", "array":
		default:
			return fmt.Errorf("paramschema: %s.type %q is not supported", path, ts)
		}
	}
	if props, ok := sch["properties"]; ok {
		pm, ok := props.(map[string]any)
		if !ok {
			return fmt.Errorf("paramschema: %s.properties must be an object", path)
		}
		for name, raw := range pm {
			child, ok := raw.(map[string]any)
			if !ok {
				return fmt.Errorf("paramschema: %s.properties.%s must be an object", path, name)
			}
			if err := checkSchemaShape(child, path+".properties."+name); err != nil {
				return err
			}
		}
	}
	if req, ok := sch["required"]; ok {
		arr, ok := req.([]any)
		if !ok {
			return fmt.Errorf("paramschema: %s.required must be an array", path)
		}
		for _, r := range arr {
			if _, ok := r.(string); !ok {
				return fmt.Errorf("paramschema: %s.required entries must be strings", path)
			}
		}
	}
	if ap, ok := sch["additionalProperties"]; ok {
		if _, ok := ap.(bool); !ok {
			return fmt.Errorf("paramschema: %s.additionalProperties must be a boolean", path)
		}
	}
	if items, ok := sch["items"]; ok {
		child, ok := items.(map[string]any)
		if !ok {
			return fmt.Errorf("paramschema: %s.items must be an object", path)
		}
		if err := checkSchemaShape(child, path+".items"); err != nil {
			return err
		}
	}
	for _, key := range []string{"minLength", "maxLength"} {
		if bound, ok := sch[key]; ok {
			if n, isNum := bound.(float64); !isNum || n < 0 || n != math.Trunc(n) {
				return fmt.Errorf("paramschema: %s.%s must be a non-negative integer", path, key)
			}
		}
	}
	if enum, ok := sch["enum"]; ok {
		if _, ok := enum.([]any); !ok {
			return fmt.Errorf("paramschema: %s.enum must be an array", path)
		}
	}
	if wf, ok := sch["withinFrame"]; ok {
		if b, ok := wf.(bool); !ok || !b {
			return fmt.Errorf("paramschema: %s.withinFrame must be true", path)
		}
	}
	if rt, ok := sch["requiresTrue"]; ok {
		m, ok := rt.(map[string]any)
		if !ok {
			return fmt.Errorf("paramschema: %s.requiresTrue must be an object", path)
		}
		for field, depsRaw := range m {
			deps, ok := depsRaw.([]any)
			if !ok {
				return fmt.Errorf("paramschema: %s.requiresTrue.%s must be an array", path, field)
			}
			for _, d := range deps {
				if _, ok := d.(string); !ok {
					return fmt.Errorf("paramschema: %s.requiresTrue.%s entries must be strings", path, field)
				}
			}
		}
	}
	return nil
}

func validateObject(values map[string]any, sch map[string]any, path string) error {
	props, _ := sch["properties"].(map[string]any)
	if ap, ok := sch["additionalProperties"].(bool); ok && !ap {
		var unknown []string
		for k := range values {
			if _, declared := props[k]; !declared {
				unknown = append(unknown, k)
			}
		}
		if len(unknown) > 0 {
			sort.Strings(unknown)
			return fmt.Errorf("paramschema: unknown parameter(s) %s%s", qualified(path), strings.Join(unknown, ", "))
		}
	}
	if req, ok := sch["required"].([]any); ok {
		for _, r := range req {
			name, _ := r.(string)
			if _, present := values[name]; !present {
				return fmt.Errorf("paramschema: %s%s is required", qualified(path), name)
			}
		}
	}
	if rt, ok := sch["requiresTrue"].(map[string]any); ok {
		fields := make([]string, 0, len(rt))
		for f := range rt {
			fields = append(fields, f)
		}
		sort.Strings(fields)
		for _, field := range fields {
			if _, present := values[field]; !present {
				continue
			}
			for _, d := range rt[field].([]any) {
				dep, _ := d.(string)
				b, ok := values[dep].(bool)
				if !ok || !b {
					return fmt.Errorf("paramschema: %s%s requires %s%s=true", qualified(path), field, qualified(path), dep)
				}
			}
		}
	}
	// Deterministic error order.
	names := make([]string, 0, len(props))
	for name := range props {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		value, present := values[name]
		if !present {
			continue
		}
		child := props[name].(map[string]any)
		if err := validateValue(value, child, joinPath(path, name)); err != nil {
			return err
		}
	}
	return nil
}

// validateWithinFrame enforces that a normalized box has positive area and stays
// inside the unit square. Per-field bounds (each coordinate in 0..1) are plain
// JSON Schema; the sum is the relational part, which is why it lives here.
func validateWithinFrame(values map[string]any, path string) error {
	get := func(name string) (float64, bool) {
		v, ok := values[name].(float64)
		return v, ok
	}
	x, okx := get("x")
	y, oky := get("y")
	w, okw := get("width")
	h, okh := get("height")
	if !okx || !oky || !okw || !okh {
		return fmt.Errorf("paramschema: %s must have numeric x, y, width and height", path)
	}
	if w <= 0 || h <= 0 || x < 0 || y < 0 || x+w > 1 || y+h > 1 {
		return fmt.Errorf("paramschema: %s must have positive area within normalized 0..1 coordinates", path)
	}
	return nil
}

func validateValue(value any, sch map[string]any, path string) error {
	if enum, ok := sch["enum"].([]any); ok && len(enum) > 0 {
		matched := false
		for _, allowed := range enum {
			if valuesEqual(value, allowed) {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("paramschema: %s must be one of %v", path, enum)
		}
	}
	typ, _ := sch["type"].(string)
	switch typ {
	case "object":
		m, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("paramschema: %s must be an object", path)
		}
		if err := validateObject(m, sch, path); err != nil {
			return err
		}
		if wf, _ := sch["withinFrame"].(bool); wf {
			return validateWithinFrame(m, path)
		}
		return nil
	case "array":
		arr, ok := value.([]any)
		if !ok {
			return fmt.Errorf("paramschema: %s must be an array", path)
		}
		if max, ok := sch["maxItems"].(float64); ok && float64(len(arr)) > max {
			return fmt.Errorf("paramschema: %s must have at most %v items", path, max)
		}
		if items, ok := sch["items"].(map[string]any); ok {
			for i, item := range arr {
				if err := validateValue(item, items, fmt.Sprintf("%s[%d]", path, i)); err != nil {
					return err
				}
			}
		}
		return nil
	case "number", "integer":
		n, ok := value.(float64)
		if !ok {
			return fmt.Errorf("paramschema: %s must be a number", path)
		}
		if typ == "integer" && n != math.Trunc(n) {
			return fmt.Errorf("paramschema: %s must be an integer", path)
		}
		if min, ok := sch["minimum"].(float64); ok && n < min {
			return fmt.Errorf("paramschema: %s must be >= %v", path, min)
		}
		if max, ok := sch["maximum"].(float64); ok && n > max {
			return fmt.Errorf("paramschema: %s must be <= %v", path, max)
		}
		return nil
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("paramschema: %s must be a boolean", path)
		}
		return nil
	case "string":
		text, ok := value.(string)
		if !ok {
			return fmt.Errorf("paramschema: %s must be a string", path)
		}
		length := float64(utf8.RuneCountInString(text))
		if min, ok := sch["minLength"].(float64); ok && length < min {
			return fmt.Errorf("paramschema: %s must be at least %v characters", path, min)
		}
		if max, ok := sch["maxLength"].(float64); ok && length > max {
			return fmt.Errorf("paramschema: %s must be at most %v characters", path, max)
		}
		return nil
	default:
		// No type declared: enum/other constraints already applied.
		return nil
	}
}

func valuesEqual(a, b any) bool {
	switch av := a.(type) {
	case float64:
		bv, ok := b.(float64)
		return ok && av == bv
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case bool:
		bv, ok := b.(bool)
		return ok && av == bv
	default:
		return false
	}
}

func joinPath(base, name string) string {
	if base == "" {
		return name
	}
	return base + "." + name
}

func qualified(path string) string {
	if path == "" {
		return ""
	}
	return path + "."
}
