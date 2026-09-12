package markedrefs

import "testing"

func TestNumMarkerParses(t *testing.T) {
	refs := ParseInlineCode("free tier allows `num[threshold]:100` requests/day", 1)
	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d (%#v)", len(refs), refs)
	}
	ref := refs[0]
	if ref.Marker != MarkerNum {
		t.Fatalf("marker = %q, want %q", ref.Marker, MarkerNum)
	}
	if len(ref.Qualifiers) != 1 || ref.Qualifiers[0] != NumberCategoryThreshold {
		t.Fatalf("qualifiers = %#v, want [threshold]", ref.Qualifiers)
	}
	if ref.Value != "100" {
		t.Fatalf("value = %q, want 100", ref.Value)
	}
	if ref.Raw != "`num[threshold]:100`" {
		t.Fatalf("raw = %q", ref.Raw)
	}
}

func TestNumMarkerIsKnown(t *testing.T) {
	if !IsKnownMarker(MarkerNum) {
		t.Fatalf("expected %q to be a known marker", MarkerNum)
	}
}

func TestNumberCategory(t *testing.T) {
	tests := []struct {
		name    string
		ref     Reference
		wantCat string
		wantOK  bool
	}{
		{"target", Reference{Marker: MarkerNum, Qualifiers: []string{NumberCategoryTarget}}, NumberCategoryTarget, true},
		{"sot", Reference{Marker: MarkerNum, Qualifiers: []string{NumberCategorySoT}}, NumberCategorySoT, true},
		{"no category", Reference{Marker: MarkerNum}, "", false},
		{"unknown category", Reference{Marker: MarkerNum, Qualifiers: []string{"vibes"}}, "", false},
		{"non-num marker", Reference{Marker: MarkerPath, Qualifiers: []string{NumberCategoryTarget}}, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCat, gotOK := NumberCategory(tt.ref)
			if gotCat != tt.wantCat || gotOK != tt.wantOK {
				t.Fatalf("NumberCategory() = (%q, %v), want (%q, %v)", gotCat, gotOK, tt.wantCat, tt.wantOK)
			}
		})
	}
}

func TestKnownNumberCategoriesAreDefensiveCopy(t *testing.T) {
	got := KnownNumberCategories()
	if len(got) == 0 {
		t.Fatal("expected number categories")
	}
	got[0].Name = "mutated"
	if IsKnownNumberCategory("mutated") {
		t.Fatal("KnownNumberCategories returned mutable registry")
	}
	if !IsKnownNumberCategory(NumberCategoryPrice) {
		t.Fatalf("expected %q to be a known number category", NumberCategoryPrice)
	}
}

func TestNumberCategoriesAreNotGlobalQualifiers(t *testing.T) {
	// Number categories are intentionally scoped to the `num` marker and must
	// not leak into the global qualifier registry.
	for _, cat := range []string{
		NumberCategoryTarget, NumberCategoryThreshold, NumberCategoryPrice,
		NumberCategoryVersion, NumberCategoryDecision, NumberCategorySoT,
	} {
		if IsKnownQualifier(cat) {
			t.Fatalf("number category %q must not be a global qualifier", cat)
		}
	}
}
