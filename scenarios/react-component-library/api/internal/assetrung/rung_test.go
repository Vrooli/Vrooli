package assetrung

import (
	"errors"
	"fmt"
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
		if want := `unknown catalog asset kind "` + kind + `"`; err.Error() != want {
			t.Fatalf("error = %q; want %q", err.Error(), want)
		}
	}
}

func TestRungNamesAndStringRepresentEverySentinel(t *testing.T) {
	tests := []struct {
		rung Rung
		name string
	}{
		{RungUnknown, "unknown"},
		{RungFixture, "fixture"},
		{RungFoundation, "foundation"},
		{RungRuntime, "runtime"},
		{RungPrimitive, "primitive"},
		{RungComponent, "component"},
		{RungComposition, "composition"},
		{RungPageTemplate, "page-template"},
		{Rung(99), "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rung.Name(); got != tt.name {
				t.Fatalf("Rung(%d).Name() = %q; want %q", tt.rung, got, tt.name)
			}
			wantString := fmt.Sprintf("%d (%s)", int(tt.rung), tt.name)
			if got := tt.rung.String(); got != wantString {
				t.Fatalf("Rung(%d).String() = %q; want %q", tt.rung, got, wantString)
			}
		})
	}
}

func TestAllIsTheOrderedDependencyRungSetAndReturnsIndependentSlices(t *testing.T) {
	want := []Rung{RungFoundation, RungRuntime, RungPrimitive, RungComponent, RungComposition, RungPageTemplate}
	got := All()
	if len(got) != len(want) {
		t.Fatalf("All() = %v; want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("All() = %v; want ordered dependency rungs %v", got, want)
		}
		if i > 0 && got[i-1] >= got[i] {
			t.Fatalf("dependency rungs are not strictly increasing: %v", got)
		}
	}
	for _, excluded := range []Rung{RungUnknown, RungFixture} {
		for _, rung := range got {
			if rung == excluded {
				t.Fatalf("All() must exclude non-dependency rung %s", excluded)
			}
		}
	}

	got[0] = RungUnknown
	if again := All(); again[0] != RungFoundation {
		t.Fatalf("mutating one All() result changed subsequent calls: %v", again)
	}
}
