// Package providers is the descriptor→unified-result adapter runtime. It holds
// the ONE generic mapping path that turns any provider's heterogeneous JSON
// response into the router's unified SearchHit shape, driven entirely by the
// declarative ResultMapping on the provider descriptor.
//
// This package is what makes the router's no-conditional-monolith invariant
// true: there is zero provider-specific code here. Adding a provider is a
// registry row carrying its ResultMapping; this code never changes. Phase 4
// wires the live fan-out (call endpoint → MapResults); Phase 3 ships and tests
// the mapping path itself against captured fixtures.
package providers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

// MapResults applies the descriptor's ResultMapping to a provider's raw JSON
// response body and returns unified SearchHits tagged with the leaf's
// provenance (provider_id, provider_group, type). Scores are normalized to the
// [0,1] band per the mapping's score_scale so they are comparable before
// rerank. When filter_field/filter_value are set, only items whose filter_field
// equals filter_value are kept (the multi-leaf-on-one-endpoint case). When
// presence_field is set, only items where that field is populated are kept (the
// presence-discriminated case, e.g. ui-health surfaces that carry a `widget`).
//
// It returns an error only on malformed JSON or a results_path that does not
// resolve to an array; individual items with missing fields degrade gracefully
// (empty string / zero score) rather than failing the whole response, so one
// odd row never sinks a query.
func MapResults(d *registryv1.ProviderDescriptor, body []byte) ([]*routingv1.SearchHit, error) {
	m := d.GetResultMapping()
	if m == nil {
		return nil, fmt.Errorf("provider %q: nil result_mapping", d.GetProviderId())
	}

	var root any
	if err := json.Unmarshal(body, &root); err != nil {
		return nil, fmt.Errorf("provider %q: decode response: %w", d.GetProviderId(), err)
	}

	rawResults := lookupPath(root, m.GetResultsPath())
	// An absent or JSON-null results array is an honest "no matches", not a
	// mapping failure: many providers (e.g. ui-health) omit the `results` key
	// entirely on a zero-result query. Map that to an empty hit set so the leaf
	// reports an honest empty group rather than degrading. Only a results_path
	// that resolves to a *present, non-array* value is a real mapping error.
	if rawResults == nil {
		return []*routingv1.SearchHit{}, nil
	}
	items, ok := rawResults.([]any)
	if !ok {
		return nil, fmt.Errorf("provider %q: results_path %q did not resolve to an array", d.GetProviderId(), m.GetResultsPath())
	}

	filterField := strings.TrimSpace(m.GetFilterField())
	presenceField := strings.TrimSpace(m.GetPresenceField())
	measureField := strings.TrimSpace(m.GetMeasureField())
	hits := make([]*routingv1.SearchHit, 0, len(items))
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			continue // skip non-object entries rather than failing the batch
		}
		if filterField != "" {
			if stringField(item, filterField) != m.GetFilterValue() {
				continue
			}
		}
		if presenceField != "" && !isPresent(lookupPath(item, presenceField)) {
			continue // keep only items where the presence_field is populated
		}
		hits = append(hits, &routingv1.SearchHit{
			ProviderId:    d.GetProviderId(),
			ProviderGroup: d.GetProviderGroup(),
			Type:          d.GetType(),
			Id:            stringField(item, m.GetIdField()),
			Title:         stringField(item, m.GetTitleField()),
			Snippet:       stringField(item, m.GetSnippetField()),
			Path:          stringField(item, m.GetPathField()),
			Score:         normalizeScore(numberField(item, m.GetScoreField()), m.GetScoreScale()),
			// Measure carrier: nil for every retrieval provider (measure_field
			// unset); populated only for the measures provider, generically.
			Measure: decodeMeasureHit(item, measureField),
		})
	}
	return hits, nil
}

// decodeMeasureHit decodes the per-item measure object at `path` into a
// SearchHit.MeasureHit, following the fixed contract keys the measures provider
// emits (measure_id, scenario, params, answer, needs, effect, executed_query,
// confidence). It returns nil when `path` is empty (every non-measure provider)
// or resolves to no object — so the field stays unset for retrieval hits. This
// is the generic carrier: there is still zero provider-specific code here, the
// descriptor's measure_field is the only switch.
func decodeMeasureHit(item map[string]any, path string) *routingv1.MeasureHit {
	if path == "" {
		return nil
	}
	obj, ok := lookupPath(item, path).(map[string]any)
	if !ok || len(obj) == 0 {
		return nil
	}
	mh := &routingv1.MeasureHit{
		MeasureId:     coerceString(obj["measure_id"]),
		Scenario:      coerceString(obj["scenario"]),
		Answer:        coerceString(obj["answer"]),
		Effect:        coerceString(obj["effect"]),
		ExecutedQuery: coerceString(obj["executed_query"]),
		Confidence:    coerceNumber(obj["confidence"]),
	}
	if pm, ok := obj["params"].(map[string]any); ok && len(pm) > 0 {
		mh.Params = make(map[string]string, len(pm))
		for k, v := range pm {
			mh.Params[k] = coerceString(v)
		}
	}
	if needs, ok := obj["needs"].([]any); ok {
		for _, n := range needs {
			if s := coerceString(n); s != "" {
				mh.Needs = append(mh.Needs, s)
			}
		}
	}
	return mh
}

// normalizeScore maps a provider's raw score onto the comparable [0,1] band per
// its declared scale. COSINE values are clamped; PERCENT values are divided by
// 100 then clamped; RAW/UNSPECIFIED values pass through unchanged (the router
// cannot normalize an unbounded scale without provider-side min/max — rerank,
// not this step, makes raw scores comparable).
func normalizeScore(v float64, scale registryv1.ScoreScale) float64 {
	switch scale {
	case registryv1.ScoreScale_SCORE_SCALE_COSINE_0_1:
		return clamp01(v)
	case registryv1.ScoreScale_SCORE_SCALE_PERCENT_0_100:
		return clamp01(v / 100.0)
	default:
		return v
	}
}

func clamp01(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v > 1:
		return 1
	default:
		return v
	}
}

// lookupPath descends a decoded JSON value along a dot-separated path
// (e.g. "results" or "payload.title"). An empty path returns the node itself.
// A missing or non-object intermediate yields nil.
func lookupPath(node any, path string) any {
	path = strings.TrimSpace(path)
	if path == "" {
		return node
	}
	cur := node
	for _, seg := range strings.Split(path, ".") {
		obj, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = obj[seg]
	}
	return cur
}

// stringField extracts a string at the given dot path within one result item.
// Non-string scalars are coerced to their natural string form so a numeric id
// still renders; missing fields and empty paths yield "".
func stringField(item map[string]any, path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	return coerceString(lookupPath(item, path))
}

// coerceString renders a decoded JSON value as a string (string verbatim,
// number/bool in natural form, nil as ""). Shared by the path-based field
// extractor and the measure-object decoder.
func coerceString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", t)
	}
}

// coerceNumber renders a decoded JSON value as a float (number verbatim, numeric
// string parsed; otherwise 0). Shared by numberField and the measure decoder.
func coerceNumber(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case string:
		if f, err := strconv.ParseFloat(t, 64); err == nil {
			return f
		}
	}
	return 0
}

// isPresent reports whether a JSON value (from lookupPath) counts as "present"
// for a presence_field filter: a non-nil value that isn't an empty
// string/object/array. A populated nested object (e.g. ui-health's `widget`
// WidgetDeclaration) is present; an omitted field (nil) or an empty container
// is not. Scalar non-strings (numbers, bools) are always present.
func isPresent(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(t) != ""
	case map[string]any:
		return len(t) > 0
	case []any:
		return len(t) > 0
	default:
		return true
	}
}

// numberField extracts a numeric score at the given dot path. JSON numbers
// decode to float64; a numeric string is parsed as a fallback. Missing or
// non-numeric fields yield 0.
func numberField(item map[string]any, path string) float64 {
	if strings.TrimSpace(path) == "" {
		return 0
	}
	return coerceNumber(lookupPath(item, path))
}
