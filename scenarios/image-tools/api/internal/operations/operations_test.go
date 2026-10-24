package operations

import (
	"sort"
	"testing"
)

// TestAllReturnsDefensiveCopy guards the SSOT against mutation by a consumer:
// All() must hand back a copy, never the backing table.
func TestAllReturnsDefensiveCopy(t *testing.T) {
	a := All()
	if len(a) == 0 {
		t.Fatal("All() returned no operations")
	}
	if len(a) != len(table) {
		t.Fatalf("All() len = %d, want %d", len(a), len(table))
	}
	a[0].Name = "mutated"
	if table[0].Name == "mutated" {
		t.Fatal("All() exposed the backing table — mutating the result changed the SSOT")
	}
}

// TestNamesMatchTableOrder asserts Names() mirrors the declaration order and set.
func TestNamesMatchTableOrder(t *testing.T) {
	names := Names()
	if len(names) != len(table) {
		t.Fatalf("Names() len = %d, want %d", len(names), len(table))
	}
	for i, op := range table {
		if names[i] != op.Name {
			t.Errorf("Names()[%d] = %q, want %q (declaration order)", i, names[i], op.Name)
		}
	}
}

// TestGetAndHas covers the lookup surface for known and unknown ops.
func TestGetAndHas(t *testing.T) {
	op, ok := Get("inpaint")
	if !ok {
		t.Fatal(`Get("inpaint") ok = false, want true`)
	}
	if op.Name != "inpaint" || op.Category != CategoryGeneration {
		t.Errorf("Get(\"inpaint\") = %+v, want name=inpaint category=generation", op)
	}
	if !op.RequiresImage || !op.RequiresMask || !op.PromptDriven {
		t.Errorf("inpaint I/O contract = %+v, want image+mask+prompt", op)
	}

	if _, ok := Get("does_not_exist"); ok {
		t.Error(`Get("does_not_exist") ok = true, want false`)
	}
	if !Has("text_to_image") {
		t.Error(`Has("text_to_image") = false, want true`)
	}
	if Has("") {
		t.Error(`Has("") = true, want false`)
	}
}

// TestByCategory checks the category filter is order-preserving and that an
// empty/unknown category selects nothing.
func TestByCategory(t *testing.T) {
	gen := ByCategory(CategoryGeneration)
	if len(gen) == 0 {
		t.Fatal("ByCategory(generation) returned nothing")
	}
	for _, op := range gen {
		if op.Category != CategoryGeneration {
			t.Errorf("ByCategory(generation) returned %q in category %q", op.Name, op.Category)
		}
	}

	// The AI engine derives its catalog from exactly generation+enhancement; that
	// union must be the sum of the two single-category slices (no overlap, no loss).
	genN := len(ByCategory(CategoryGeneration))
	enhN := len(ByCategory(CategoryEnhancement))
	union := ByCategory(CategoryGeneration, CategoryEnhancement)
	if len(union) != genN+enhN {
		t.Errorf("ByCategory(gen,enh) len = %d, want %d (gen %d + enh %d)", len(union), genN+enhN, genN, enhN)
	}

	if got := ByCategory(); got != nil {
		t.Errorf("ByCategory() with no categories = %v, want nil", got)
	}
	if got := ByCategory(Category("bogus")); got != nil {
		t.Errorf("ByCategory(bogus) = %v, want nil", got)
	}
}

// TestNamesByCategoryIsSorted asserts the set-style accessor is sorted and
// matches ByCategory's membership.
func TestNamesByCategoryIsSorted(t *testing.T) {
	names := NamesByCategory(CategoryAnalysis)
	if len(names) == 0 {
		t.Fatal("NamesByCategory(analysis) returned nothing")
	}
	if !sort.StringsAreSorted(names) {
		t.Errorf("NamesByCategory(analysis) not sorted: %v", names)
	}
	if len(names) != len(ByCategory(CategoryAnalysis)) {
		t.Errorf("NamesByCategory/ByCategory length mismatch for analysis")
	}
}

// TestVocabularyInvariants enforces the structural rules every op row must obey.
// init() already panics on empty/duplicate names + invalid categories; this pins
// the I/O-contract invariants that the rest of the system relies on.
func TestVocabularyInvariants(t *testing.T) {
	seen := make(map[string]struct{}, len(table))
	for _, op := range table {
		if _, dup := seen[op.Name]; dup {
			t.Errorf("duplicate op name %q", op.Name)
		}
		seen[op.Name] = struct{}{}

		if !op.Category.valid() {
			t.Errorf("op %q has invalid category %q", op.Name, op.Category)
		}
		if op.Summary == "" {
			t.Errorf("op %q has an empty summary", op.Name)
		}
		// A mask only makes sense over an input image.
		if op.RequiresMask && !op.RequiresImage {
			t.Errorf("op %q requires a mask but not an image", op.Name)
		}
	}

	// text_to_image is the one prompt-only generator: it must NOT require an input.
	if t2i, _ := Get("text_to_image"); t2i.RequiresImage || t2i.RequiresMask {
		t.Errorf("text_to_image should be prompt-only, got %+v", t2i)
	}
}
