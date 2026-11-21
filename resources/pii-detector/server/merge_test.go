package main

import "testing"

// nonOverlapping is the merge invariant: after dedupeOverlaps no two kept spans
// may overlap. It is asserted on every table row so a future edit to the greedy
// conflict loop that leaves an overlap is caught even when span counts look
// right.
func nonOverlapping(spans []Span) bool {
	for i := 0; i < len(spans); i++ {
		for j := i + 1; j < len(spans); j++ {
			if overlaps(spans[i], spans[j]) {
				return false
			}
		}
	}
	return true
}

func TestMergeSpans(t *testing.T) {
	cases := []struct {
		name     string
		regex    []Span
		model    []Span
		wantCats []string // categories in final start-sorted order
	}{
		{
			name:     "no overlap keeps all sorted by start",
			regex:    []Span{{Category: CategoryEmail, Start: 0, End: 5, Confidence: 0.99, Source: "regex"}},
			model:    []Span{{Category: "PERSON", Start: 10, End: 15, Confidence: 0.8, Source: "model"}},
			wantCats: []string{CategoryEmail, "PERSON"},
		},
		{
			name:     "overlap prefers higher confidence",
			regex:    []Span{{Category: CategoryCreditCard, Start: 0, End: 16, Confidence: 0.95, Source: "regex"}},
			model:    []Span{{Category: "ACCOUNT", Start: 2, End: 10, Confidence: 0.6, Source: "model"}},
			wantCats: []string{CategoryCreditCard},
		},
		{
			name:     "confidence tie breaks to regex",
			regex:    []Span{{Category: CategoryEmail, Start: 0, End: 10, Confidence: 0.9, Source: "regex"}},
			model:    []Span{{Category: "EMAIL", Start: 0, End: 10, Confidence: 0.9, Source: "model"}},
			wantCats: []string{CategoryEmail},
		},
		{
			name:     "model only when regex empty",
			model:    []Span{{Category: "PERSON", Start: 3, End: 9, Confidence: 0.7, Source: "model"}},
			wantCats: []string{"PERSON"},
		},
		{
			name:     "adjacent touching spans both kept",
			regex:    []Span{{Category: "A", Start: 0, End: 5, Confidence: 0.9, Source: "regex"}},
			model:    []Span{{Category: "B", Start: 5, End: 10, Confidence: 0.9, Source: "model"}},
			wantCats: []string{"A", "B"},
		},
		{
			name: "short high-confidence span contained in longer low-confidence one wins",
			// containment: exercises the preferred() length/confidence path.
			regex:    []Span{{Category: CategorySSN, Start: 4, End: 8, Confidence: 0.95, Source: "regex"}},
			model:    []Span{{Category: "NUMBER", Start: 0, End: 20, Confidence: 0.5, Source: "model"}},
			wantCats: []string{CategorySSN},
		},
		{
			name: "three-way overlap chain resolves to non-overlapping set",
			// B overlaps both A and C; the greedy loop must still leave a
			// non-overlapping result. Mixed confidences force a real resolution.
			regex: []Span{
				{Category: "A", Start: 0, End: 8, Confidence: 0.6, Source: "model"},
				{Category: "C", Start: 6, End: 14, Confidence: 0.6, Source: "model"},
			},
			model: []Span{
				{Category: "B", Start: 4, End: 10, Confidence: 0.9, Source: "model"},
			},
			wantCats: []string{"B"},
		},
		{
			name: "equal confidence equal source different length keeps longer",
			regex: []Span{
				{Category: "SHORT", Start: 0, End: 4, Confidence: 0.8, Source: "regex"},
				{Category: "LONG", Start: 0, End: 10, Confidence: 0.8, Source: "regex"},
			},
			wantCats: []string{"LONG"},
		},
		{
			name: "utf8 byte-offset spans merge over multibyte text",
			// Offsets are byte offsets; a contained lower-confidence span is
			// dropped and the surviving offsets stay valid bytes.
			regex:    []Span{{Category: CategoryEmail, Start: 7, End: 22, Confidence: 0.99, Source: "regex"}},
			model:    []Span{{Category: "TOKEN", Start: 10, End: 15, Confidence: 0.5, Source: "model"}},
			wantCats: []string{CategoryEmail},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := MergeSpans(c.regex, c.model)
			if !nonOverlapping(got) {
				t.Fatalf("merge invariant violated, overlapping spans: %+v", got)
			}
			if len(got) != len(c.wantCats) {
				t.Fatalf("got %d spans %+v, want cats %v", len(got), got, c.wantCats)
			}
			for i, want := range c.wantCats {
				if got[i].Category != want {
					t.Errorf("span[%d].Category = %q, want %q (all: %+v)", i, got[i].Category, want, got)
				}
			}
		})
	}
}

func TestMergeEmpty(t *testing.T) {
	if got := MergeSpans(nil, nil); got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}
