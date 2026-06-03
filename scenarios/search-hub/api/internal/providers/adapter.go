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
// equals filter_value are kept (the multi-leaf-on-one-endpoint case).
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
	items, ok := rawResults.([]any)
	if !ok {
		return nil, fmt.Errorf("provider %q: results_path %q did not resolve to an array", d.GetProviderId(), m.GetResultsPath())
	}

	filterField := strings.TrimSpace(m.GetFilterField())
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
		hits = append(hits, &routingv1.SearchHit{
			ProviderId:    d.GetProviderId(),
			ProviderGroup: d.GetProviderGroup(),
			Type:          d.GetType(),
			Id:            stringField(item, m.GetIdField()),
			Title:         stringField(item, m.GetTitleField()),
			Snippet:       stringField(item, m.GetSnippetField()),
			Path:          stringField(item, m.GetPathField()),
			Score:         normalizeScore(numberField(item, m.GetScoreField()), m.GetScoreScale()),
		})
	}
	return hits, nil
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
	v := lookupPath(item, path)
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

// numberField extracts a numeric score at the given dot path. JSON numbers
// decode to float64; a numeric string is parsed as a fallback. Missing or
// non-numeric fields yield 0.
func numberField(item map[string]any, path string) float64 {
	if strings.TrimSpace(path) == "" {
		return 0
	}
	switch t := lookupPath(item, path).(type) {
	case float64:
		return t
	case string:
		if f, err := strconv.ParseFloat(t, 64); err == nil {
			return f
		}
	}
	return 0
}
