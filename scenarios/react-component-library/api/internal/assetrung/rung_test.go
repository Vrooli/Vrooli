package assetrung

import (
	"errors"
	"testing"
)

func TestOfDeclaredKinds(t *testing.T) {
	tests := []struct {
		kind string
		want Rung
	}{
		{"foundation", RungFoundation},
		{"runtime-hook", RungRuntime},
		{"runtime-service", RungRuntime},
		{"adapter", RungRuntime},
		{"generator", RungRuntime},
		{"primitive", RungPrimitive},
		{"component", RungComponent},
		{"pattern", RungComposition},
		{"navigation", RungComposition},
		{"page-template", RungPageTemplate},
		{"fixture", RungFixture},
	}
	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			got, err := Of(tt.kind)
			if err != nil || got != tt.want {
				t.Fatalf("Of(%q) = %v, %v; want %v", tt.kind, got, err, tt.want)
			}
		})
	}
}

func TestOfUnknownKindFailsClosed(t *testing.T) {
	for _, kind := range []string{"", "typo", "unknown"} {
		got, err := Of(kind)
		if got == RungFoundation || err == nil {
			t.Fatalf("Of(%q) = %v, %v; expected a non-foundation error", kind, got, err)
		}
		var typed UnknownKindError
		if !errors.As(err, &typed) || typed.Kind != kind {
			t.Fatalf("error = %v; want UnknownKindError(%q)", err, kind)
		}
	}
}
