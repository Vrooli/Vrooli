package capture

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestStrictTokenInteractionFlowSetsMagentaSentinel(t *testing.T) {
	raw, err := strictTokenInteractionFlowJSON()
	if err != nil {
		t.Fatal(err)
	}
	var flow map[string]any
	if err := json.Unmarshal([]byte(raw), &flow); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(raw, "#ff00ff") || !strings.Contains(raw, "cssRules") {
		t.Fatalf("strict flow does not set CSSOM sentinel: %s", raw)
	}
	if len(flow["nodes"].([]any)) != 1 {
		t.Fatalf("strict flow node count = %d, want 1", len(flow["nodes"].([]any)))
	}
}

func TestStrictTokenNonSentinelColorsRecordsUnboundElements(t *testing.T) {
	raw := `{"tagName":"BODY","selector":"body","computed":{"color":"rgb(0, 0, 0)","backgroundColor":"rgba(0, 0, 0, 0)"},"children":[{"selector":"#bound","computed":{"color":"rgb(255, 0, 255)","backgroundColor":"color(srgb 1 0 1)"}}]}`
	sentinelCount, count, samples := strictTokenColorObservation(raw)
	if sentinelCount != 1 || count != 1 || len(samples) != 1 || !strings.HasPrefix(samples[0], "body ") {
		t.Fatalf("strict color report = sentinel %d, non-sentinel %d, %v", sentinelCount, count, samples)
	}
}

func TestMatchesIntentAcceptsProductPhrasing(t *testing.T) {
	tests := []struct {
		name  string
		query string
		label string
		want  bool
	}{
		{name: "article and page words", query: "the design page", label: "Design", want: true},
		{name: "case insensitive exact", query: "Catalog Coverage", label: "catalog coverage", want: true},
		{name: "label contained in query", query: "browse assets", label: "Assets", want: true},
		{name: "unrelated", query: "settings", label: "Design", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchesIntent(tt.query, tt.label); got != tt.want {
				t.Fatalf("matchesIntent(%q, %q) = %v, want %v", tt.query, tt.label, got, tt.want)
			}
		})
	}
}

func TestResolveAcceptsSameOriginPreviewURL(t *testing.T) {
	h := &handlers{}
	resolved, err := h.resolve(context.Background(), "react-component-library", "http://localhost:23906/assets/primitives.text/preview?story=anatomy", "http://localhost:23906/")
	if err != nil {
		t.Fatal(err)
	}
	if resolved == nil || resolved.Rung != 1 || resolved.Route != "/assets/primitives.text/preview?story=anatomy" {
		t.Fatalf("direct preview resolution = %#v, want same-origin rung-1 route", resolved)
	}
}
