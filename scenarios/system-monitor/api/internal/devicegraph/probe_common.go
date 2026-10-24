package devicegraph

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Shared helpers for the command-driven macOS and Windows backends. Both
// platforms answer in JSON and both grade an absent probe the same way, so
// these shapes live here rather than being duplicated per platform.

// flattenProfilerItems walks the profiler's recursive "_items" nesting and
// returns every node, each annotated with the name of its parent so the graph
// can rebuild the containment edges the nesting expressed.
func flattenProfilerItems(nodes []map[string]any) []map[string]any {
	flattened := make([]map[string]any, 0, len(nodes))
	var walk func(items []map[string]any, parent string)
	walk = func(items []map[string]any, parent string) {
		for _, item := range items {
			copied := make(map[string]any, len(item)+1)
			for key, value := range item {
				if key == "_items" {
					continue
				}
				copied[key] = value
			}
			if parent != "" {
				copied["_parent_name"] = parent
			}
			flattened = append(flattened, copied)
			children, ok := item["_items"].([]any)
			if !ok {
				continue
			}
			nested := make([]map[string]any, 0, len(children))
			for _, child := range children {
				if mapped, ok := child.(map[string]any); ok {
					nested = append(nested, mapped)
				}
			}
			walk(nested, jsonString(item, "_name"))
		}
	}
	walk(nodes, "")
	return flattened
}

func jsonString(item map[string]any, key string) string {
	value, present := item[key]
	if !present {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return ""
	}
}

func jsonNumber(item map[string]any, key string) (float64, bool) {
	value, present := item[key]
	if !present {
		return 0, false
	}
	switch typed := value.(type) {
	case float64:
		return typed, true
	case string:
		parsed, err := strconv.ParseFloat(firstField(typed), 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func enumerationRungs(b *builder, device *Device, mechanism string) {
	device.Rungs[RungIdentity] = b.grader.measured(RungIdentity, mechanism)
	telemetry := b.grader.notApplicable(RungTelemetry, "bus enumeration reports configuration, not a live measurement")
	device.Rungs[RungTelemetry] = telemetry
	device.Rungs[RungEvidence] = b.grader.evidenceFor(telemetry)
	if device.Driver == "" {
		device.Rungs[RungControl] = b.grader.unmeasurable(RungControl,
			"macOS did not report a driver bound to this device", mechanism)
	} else {
		device.Rungs[RungControl] = b.grader.measured(RungControl, mechanism)
	}
	device.Rungs[RungAnticipation] = b.grader.notApplicable(RungAnticipation,
		"bus enumeration carries no forward-looking signal")
}

func unavailableSubsystem(b *builder, name, reason, mechanism string) Subsystem {
	return Subsystem{
		Name: name,
		Rungs: rungSet(
			b.grader.unavailable(RungIdentity, reason, mechanism),
			b.grader.unavailable(RungTelemetry, reason, mechanism),
			b.grader.unavailable(RungEvidence, "nothing to retain: "+reason, evidenceMechanism),
			b.grader.unavailable(RungControl, reason, mechanism),
			b.grader.unavailable(RungAnticipation, "no forward-looking signal: "+reason, trendMechanism),
		),
	}
}

func sanitizeIdentity(value string) string {
	replacer := strings.NewReplacer(" ", "-", "\t", "-", "/", "-")
	return strings.ToLower(strings.Trim(replacer.Replace(strings.TrimSpace(value)), "-"))
}

// firstField returns the first whitespace-separated token, which is how macOS
// reports ids and sizes ("0x1234 (Vendor)", "500107862016 bytes").
func firstField(value string) string {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// decodeJSONObjects accepts both shapes a shell probe can emit: a list of
// records, or a bare record when the pipeline yielded exactly one.
func decodeJSONObjects(data []byte) ([]map[string]any, error) {
	var list []map[string]any
	if err := json.Unmarshal(data, &list); err == nil {
		return list, nil
	}
	var single map[string]any
	if err := json.Unmarshal(data, &single); err != nil {
		return nil, fmt.Errorf("probe output was not JSON: %w", err)
	}
	return []map[string]any{single}, nil
}
