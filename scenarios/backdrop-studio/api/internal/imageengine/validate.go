package imageengine

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	opsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ops"
	"google.golang.org/protobuf/encoding/protojson"
)

// ValidateParams reports whether a style's parameters for one operation are
// something image-tools will actually accept.
//
// This lives here rather than in the catalog because this package owns the
// image-tools boundary: it is the only place that knows the wire format, and
// the catalog should be able to ask "will the engine take this?" without
// learning protobuf.
//
// It exists because of a defect shipped on 2026-08-12. Styles were authored
// requesting parameters that had no proto field, and protojson rejects unknown
// fields — so the styles stored cleanly, passed every unit suite, and then
// failed their first real render with a 400. Catching it here moves the failure
// to the moment someone writes the style, which is the only moment they can
// still fix it cheaply.
//
// Brand slots are left unresolved on purpose. They only ever appear in string
// fields (ink colours), so they round-trip fine, and validating the shape
// before a brand is bound is exactly what a catalog write needs.
func ValidateParams(operation, raw string) error {
	operation = strings.TrimSpace(operation)
	if operation == "" {
		return fmt.Errorf("treatment parameters: operation is required")
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil // no override is legitimate: the op runs on its defaults
	}

	// A bare value or array would be accepted by the merge step and then
	// silently discarded, shipping the default under a name that promised
	// something else.
	var probe map[string]any
	if err := json.Unmarshal([]byte(raw), &probe); err != nil {
		return fmt.Errorf("treatment %q parameters must be a JSON object: %w", operation, err)
	}

	// The operation name is also the proto oneof field name, so the parameters
	// can be checked in place without a separate mapping table to drift.
	wrapped := fmt.Sprintf("{%q:%s}", operation, raw)
	pb := &opsv1.OpParams{}
	if err := protojson.Unmarshal([]byte(wrapped), pb); err != nil {
		return fmt.Errorf("treatment %q parameters are not accepted by image-tools: %w (sent %s)", operation, err, raw)
	}
	return nil
}

// ValidateChain checks a whole style's parameter map against the operations it
// declares. A parameter block for an operation the style never runs is dead
// configuration — almost always a typo in the operation name — and reporting it
// is more useful than ignoring it.
func ValidateChain(treatments []string, params map[string]string) error {
	if len(params) == 0 {
		return nil
	}
	declared := make(map[string]bool, len(treatments))
	for _, op := range treatments {
		declared[strings.TrimSpace(op)] = true
	}

	// Sorted so the error a caller sees is stable across runs rather than
	// depending on map iteration order.
	ops := make([]string, 0, len(params))
	for op := range params {
		ops = append(ops, op)
	}
	sort.Strings(ops)

	for _, op := range ops {
		if !declared[op] {
			return fmt.Errorf("treatment parameters name operation %q, which the style does not run (declared: %s)", op, strings.Join(treatments, ", "))
		}
		if err := ValidateParams(op, params[op]); err != nil {
			return err
		}
	}
	return nil
}
