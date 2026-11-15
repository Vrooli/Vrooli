package capture

import (
	"context"
	"testing"
)

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
