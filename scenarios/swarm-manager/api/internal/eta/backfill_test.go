package eta

import "testing"

func TestNormalizeEffort(t *testing.T) {
	cases := map[string]string{
		"m": "M", "  L ": "L", "XS": "XS", "xl": "XL", "s": "S",
		"": "", "medium": "", "XXL": "",
	}
	for in, want := range cases {
		if got := NormalizeEffort(in); got != want {
			t.Errorf("NormalizeEffort(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLeadTimeHours(t *testing.T) {
	if h, ok := LeadTimeHours("2026-06-01T00:00:00Z", "2026-06-02T00:00:00Z"); !ok || h != 24 {
		t.Errorf("expected 24h span, got %v ok=%v", h, ok)
	}
	// Same instant -> no signal.
	if _, ok := LeadTimeHours("2026-06-01T00:00:00Z", "2026-06-01T00:00:00Z"); ok {
		t.Error("same-instant span should yield no signal")
	}
	// Reversed -> no signal.
	if _, ok := LeadTimeHours("2026-06-02T00:00:00Z", "2026-06-01T00:00:00Z"); ok {
		t.Error("negative span should yield no signal")
	}
	// Unparseable -> no signal.
	if _, ok := LeadTimeHours("not-a-time", "2026-06-01T00:00:00Z"); ok {
		t.Error("unparseable created should yield no signal")
	}
}

func TestBuildBackfillSamples(t *testing.T) {
	items := []CompletedItem{
		{Ref: "execute/a", Kind: "execute", Effort: "m", Initiative: "init-x", Created: "2026-06-01T00:00:00Z", Completed: "2026-06-03T00:00:00Z"},
		{Ref: "execute/b", Kind: "execute", Effort: "", Created: "2026-06-01T00:00:00Z", Completed: "2026-06-01T12:00:00Z"},
		{Ref: "fix/c", Kind: "fix", Effort: "L", Created: "2026-06-01T00:00:00Z", Completed: "2026-06-01T00:00:00Z"}, // no span
		{Ref: "execute/d", Kind: "execute", Effort: "S", Created: "bad", Completed: "2026-06-02T00:00:00Z"},          // unparseable
		{Ref: "execute/seen", Kind: "execute", Effort: "M", Created: "2026-06-01T00:00:00Z", Completed: "2026-06-02T00:00:00Z"},
	}
	already := map[string]struct{}{"execute/seen": {}}

	samples, rep := BuildBackfillSamples(items, already)

	if rep.Produced != 2 {
		t.Errorf("Produced = %d, want 2", rep.Produced)
	}
	if rep.SkippedNoTime != 2 {
		t.Errorf("SkippedNoTime = %d, want 2 (no-span + unparseable)", rep.SkippedNoTime)
	}
	if rep.SkippedAlready != 1 {
		t.Errorf("SkippedAlready = %d, want 1", rep.SkippedAlready)
	}
	if len(samples) != 2 {
		t.Fatalf("len(samples) = %d, want 2", len(samples))
	}
	// First sample: 2-day span = 48h, effort normalized to M.
	if samples[0].Ref != "execute/a" || samples[0].EffortClass != "M" || samples[0].DurationHours != 48 {
		t.Errorf("sample[0] = %+v, want execute/a M 48h", samples[0])
	}
	if samples[0].Initiative != "init-x" {
		t.Errorf("sample[0].Initiative = %q, want init-x", samples[0].Initiative)
	}
	// Second sample: unsized item keeps empty effort class (folds into global).
	if samples[1].Ref != "execute/b" || samples[1].EffortClass != "" || samples[1].DurationHours != 12 {
		t.Errorf("sample[1] = %+v, want execute/b (unsized) 12h", samples[1])
	}
}
