package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriterSamplerReportsDeltaAndRate(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "payload")
	if err := os.WriteFile(path, make([]byte, 100), 0o600); err != nil {
		t.Fatal(err)
	}
	sampler := NewWriterSampler([]GovernedRoot{{ID: "test", Root: root, Mount: "/", HotWriterBytesHour: 200}})
	start := time.Unix(100, 0)
	first := sampler.Sample(context.Background(), start)[0]
	if first.Bytes != 100 || first.BytesPerHour != 0 {
		t.Fatalf("first snapshot = %#v", first)
	}
	if first.Root != root || first.RootID != "test" {
		t.Fatalf("root identity = %q/%q, want %q/test", first.Root, first.RootID, root)
	}
	if got := sampler.Sample(context.Background(), start.Add(30*time.Second)); got != nil {
		t.Fatalf("sample inside cadence = %#v, want nil", got)
	}
	if err := os.WriteFile(path, make([]byte, 300), 0o600); err != nil {
		t.Fatal(err)
	}
	second := sampler.Sample(context.Background(), start.Add(time.Hour))[0]
	if second.DeltaBytes != 200 || second.BytesPerHour != 200 || second.Hot {
		t.Fatalf("second snapshot = %#v", second)
	}
}

func TestWriterSamplerExpandsGovernedChildren(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"b", "a"} {
		child := filepath.Join(root, name)
		if err := os.Mkdir(child, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(child, "payload"), make([]byte, 10), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	snapshots := NewWriterSampler([]GovernedRoot{{
		ID: "go-work-dirs", Root: root, Mount: "/", ExpandChildren: true,
	}}).Sample(context.Background(), time.Unix(100, 0))
	if len(snapshots) != 2 {
		t.Fatalf("expanded snapshots = %d, want 2", len(snapshots))
	}
	if snapshots[0].RootID != "go-work-dirs/a" || snapshots[1].RootID != "go-work-dirs/b" {
		t.Fatalf("expanded root IDs = %q, %q; want sorted children", snapshots[0].RootID, snapshots[1].RootID)
	}
}

// The total budget is a ceiling across every root in one sample, not a
// per-root allowance: once it is spent the remaining roots are not walked, and
// they are omitted rather than reported as zero bytes.
func TestWriterSamplerStopsWalkingWhenTotalBudgetIsExhausted(t *testing.T) {
	roots := []GovernedRoot{
		{ID: "first", Root: "/first", Mount: "/", MeasureBudget: 5 * time.Second},
		{ID: "second", Root: "/second", Mount: "/", MeasureBudget: 5 * time.Second},
		{ID: "third", Root: "/third", Mount: "/", MeasureBudget: 5 * time.Second},
	}
	sampler := NewWriterSamplerWithConfig(roots, WriterSamplerConfig{TotalBudget: time.Second})

	wall := time.Unix(1000, 0)
	sampler.clock = func() time.Time { return wall }
	var walked []string
	var budgets []time.Duration
	sampler.measure = func(ctx context.Context, root string) (int64, bool) {
		walked = append(walked, root)
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatalf("walk of %s has no deadline", root)
		}
		// The context deadline is on the real clock; the sampler's own clock
		// only decides how much of the total budget is left.
		budgets = append(budgets, time.Until(deadline))
		// The first walk consumes the whole budget.
		wall = wall.Add(time.Second)
		return 10, false
	}

	snapshots := sampler.Sample(context.Background(), time.Unix(100, 0))
	if len(walked) != 1 || walked[0] != "/first" {
		t.Fatalf("walked %v, want only /first before the budget ran out", walked)
	}
	if budgets[0] > time.Second {
		t.Fatalf("first root was granted %s, more than the %s total budget", budgets[0], time.Second)
	}
	if len(snapshots) != 1 || snapshots[0].RootID != "first" {
		t.Fatalf("snapshots = %#v, want only the measured root", snapshots)
	}
	if _, recorded := sampler.last["second"]; recorded {
		t.Fatal("an unwalked root was recorded as measured")
	}
}

// Expanded children are rotated across samples so a directory with many
// children costs at most ChildrenPerSample walks per tick and every child is
// still measured in turn.
func TestWriterSamplerRotatesExpandedChildrenAcrossSamples(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"a", "b", "c", "d", "e"} {
		if err := os.Mkdir(filepath.Join(root, name), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	sampler := NewWriterSamplerWithConfig(
		[]GovernedRoot{{ID: "go-work-dirs", Root: root, Mount: "/", ExpandChildren: true}},
		WriterSamplerConfig{Interval: time.Minute, ChildrenPerSample: 2},
	)

	ids := func(snapshots []WriterSnapshot) []string {
		out := make([]string, 0, len(snapshots))
		for _, snapshot := range snapshots {
			out = append(out, snapshot.RootID)
		}
		return out
	}
	start := time.Unix(100, 0)
	rounds := [][]string{
		{"go-work-dirs/a", "go-work-dirs/b"},
		{"go-work-dirs/c", "go-work-dirs/d"},
		{"go-work-dirs/e", "go-work-dirs/a"},
	}
	for i, want := range rounds {
		got := ids(sampler.Sample(context.Background(), start.Add(time.Duration(i)*time.Minute)))
		if len(got) != len(want) {
			t.Fatalf("round %d sampled %v, want %v", i, got, want)
		}
		for j := range want {
			if got[j] != want[j] {
				t.Fatalf("round %d sampled %v, want %v", i, got, want)
			}
		}
	}
}
