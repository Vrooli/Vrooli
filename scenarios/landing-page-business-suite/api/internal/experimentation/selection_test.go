package experimentation

import (
	"fmt"
	"strings"
	"testing"
)

func TestValidateSelectionEnforcesExperimentAssignmentPolicy(t *testing.T) {
	space := Space{
		Axes: map[string]Axis{
			"persona": {Variants: []string{"founder", "operator"}},
			"cta":     {Variants: []string{"demo", "trial"}},
		},
		DisallowedCombinations: []map[string]string{{"persona": "founder", "cta": "trial"}},
	}

	for name, tc := range map[string]struct {
		selection map[string]string
		wantError string
	}{
		"accepts complete allowed assignment": {selection: map[string]string{"persona": "operator", "cta": "trial"}},
		"rejects unknown axis":                {selection: map[string]string{"persona": "operator", "cta": "trial", "region": "us"}, wantError: "unknown axis region"},
		"rejects missing axis":                {selection: map[string]string{"persona": "operator"}, wantError: "axis cta is required"},
		"rejects invalid value":               {selection: map[string]string{"persona": "operator", "cta": "purchase"}, wantError: "invalid value 'purchase' for axis cta"},
		"rejects disallowed combination":      {selection: map[string]string{"persona": "founder", "cta": "trial"}, wantError: "is disallowed"},
	} {
		t.Run(name, func(t *testing.T) {
			err := ValidateSelection(space, tc.selection)
			if tc.wantError == "" {
				if err != nil {
					t.Fatalf("ValidateSelection() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("ValidateSelection() error = %v, want %q", err, tc.wantError)
			}
		})
	}
}

func TestSelectWeightedRandomVariant(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		if got := SelectWeightedRandomVariant(nil); got != nil {
			t.Fatalf("SelectWeightedRandomVariant(nil) = %#v, want nil", got)
		}
	})
	t.Run("zero weights preserve first fallback", func(t *testing.T) {
		variants := []*VariantSnapshot{{Variant: VariantSnapshotMeta{Slug: "first"}}, {Variant: VariantSnapshotMeta{Slug: "second"}}}
		if got := SelectWeightedRandomVariant(variants); got == nil || got.Variant.Slug != "first" {
			t.Fatalf("fallback = %#v, want first", got)
		}
	})
	t.Run("disabled variants are never selected", func(t *testing.T) {
		variants := []*VariantSnapshot{{Variant: VariantSnapshotMeta{Slug: "archived", Status: "archived", Weight: 1000}}, {Variant: VariantSnapshotMeta{Slug: "active", Weight: 1}}}
		for range 100 {
			if got := SelectWeightedRandomVariant(variants); got == nil || got.Variant.Slug != "active" {
				t.Fatalf("selected %#v, want active", got)
			}
		}
	})
}

func TestSelectVariantForVisitorIsSticky(t *testing.T) {
	variants := []*VariantSnapshot{
		{Variant: VariantSnapshotMeta{Slug: "a", Weight: 50, Status: "active"}},
		{Variant: VariantSnapshotMeta{Slug: "b", Weight: 50, Status: "active"}},
	}
	first := SelectVariantForVisitor(variants, "visitor-123")
	for i := 0; i < 100; i++ {
		if got := SelectVariantForVisitor(variants, "visitor-123"); got.Variant.Slug != first.Variant.Slug {
			t.Fatalf("selection changed: first=%s got=%s", first.Variant.Slug, got.Variant.Slug)
		}
	}
}

func TestSelectVariantForVisitorDistributesAcrossVisitors(t *testing.T) {
	variants := []*VariantSnapshot{{Variant: VariantSnapshotMeta{Slug: "a", Weight: 70, Status: "active"}}, {Variant: VariantSnapshotMeta{Slug: "b", Weight: 30, Status: "active"}}}
	counts := map[string]int{}
	for i := 0; i < 1000; i++ {
		counts[SelectVariantForVisitor(variants, fmt.Sprintf("visitor-%d", i)).Variant.Slug]++
	}
	if counts["a"] < 620 || counts["a"] > 780 {
		t.Fatalf("distribution outside tolerance: %#v", counts)
	}
	t.Logf("observed distribution across 1000 visitors: a=%d (%.3f), b=%d (%.3f)", counts["a"], float64(counts["a"])/1000, counts["b"], float64(counts["b"])/1000)
}

func TestVariantWeight(t *testing.T) {
	for name, test := range map[string]struct {
		snapshot *VariantSnapshot
		want     int
	}{
		"nil":      {nil, 0},
		"positive": {&VariantSnapshot{Variant: VariantSnapshotMeta{Weight: 75}}, 75},
		"zero":     {&VariantSnapshot{Variant: VariantSnapshotMeta{Weight: 0}}, 0},
		"negative": {&VariantSnapshot{Variant: VariantSnapshotMeta{Weight: -5}}, 0},
		"archived": {&VariantSnapshot{Variant: VariantSnapshotMeta{Status: "archived", Weight: 75}}, 0},
	} {
		t.Run(name, func(t *testing.T) {
			if got := VariantWeight(test.snapshot); got != test.want {
				t.Fatalf("VariantWeight() = %d, want %d", got, test.want)
			}
		})
	}
}
