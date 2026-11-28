package schedule

import (
	"testing"
	"time"
)

func idsOf(groups []*candidate) []string {
	ids := make([]string, len(groups))
	for i, group := range groups {
		ids[i] = group.Jobs[0].ID
	}
	return ids
}

func TestGroupCandidatesSplitsByKindModelAndBatch(t *testing.T) {
	jobs := []Job{
		jobSpec("a", "embed", "owner", func(j *Job) { j.BatchKey = "run-1" }),
		jobSpec("b", "embed", "owner", func(j *Job) { j.BatchKey = "run-1" }),
		jobSpec("c", "embed", "owner", func(j *Job) { j.BatchKey = "run-2" }),
		jobSpec("d", "annotate", "owner", func(j *Job) { j.BatchKey = "run-1" }),
		jobSpec("e", "embed", "owner", func(j *Job) { j.ModelAffinity = "winnow:e4b" }),
	}
	groups := groupCandidates(jobs)
	if len(groups) != 4 {
		t.Fatalf("groups = %d, want 4: %v", len(groups), idsOf(groups))
	}
	bySize := map[int]int{}
	for _, group := range groups {
		bySize[len(group.Jobs)]++
	}
	if bySize[2] != 1 || bySize[1] != 3 {
		t.Fatalf("group sizes = %v, want one pair and three singles", bySize)
	}
	// Kind is part of the key: an embed job and an annotate job sharing a model
	// must not be merged into one batch.
	for _, group := range groups {
		if len(group.Jobs) > 1 && group.Key.Kind != "embed" {
			t.Fatalf("merged group has kind %q", group.Key.Kind)
		}
	}
}

func TestGroupCandidatesAggregatesValueAndSeconds(t *testing.T) {
	jobs := []Job{
		jobSpec("a", "embed", "owner", func(j *Job) {
			j.Value = 10
			j.Estimate.Seconds = 5
			j.NotBefore = testEpoch.Add(2 * time.Minute)
		}),
		jobSpec("b", "embed", "owner", func(j *Job) {
			j.Value = 30
			j.Estimate.Seconds = 15
			j.NotBefore = testEpoch.Add(time.Minute)
		}),
	}
	groups := groupCandidates(jobs)
	if len(groups) != 1 {
		t.Fatalf("groups = %d, want 1", len(groups))
	}
	group := groups[0]
	if group.Value != 40 || group.Seconds != 20 {
		t.Fatalf("value/seconds = %v/%v, want 40/20", group.Value, group.Seconds)
	}
	if group.Density != 2 {
		t.Fatalf("density = %v, want 2", group.Density)
	}
	if !group.Oldest.Equal(testEpoch.Add(time.Minute)) {
		t.Fatalf("oldest = %s, want the earliest due", group.Oldest)
	}
	if group.Jobs[0].ID != "b" {
		t.Fatalf("jobs are not sorted by value density: %v", idsOf(groups))
	}
}

func TestRankCandidatesLadder(t *testing.T) {
	rank := func(jobs []Job, watch func([]*candidate)) []string {
		groups := groupCandidates(jobs)
		if watch != nil {
			watch(groups)
		}
		rankCandidates(groups, budgetSnapshot{ByOwner: map[string]float64{}})
		return idsOf(groups)
	}

	t.Run("priority tier wins over value", func(t *testing.T) {
		order := rank([]Job{
			jobSpec("batch", "embed", "owner", func(j *Job) { j.Value = 1000 }),
			jobSpec("interactive", "embed", "owner", func(j *Job) { j.Priority = PriorityInteractive; j.Value = 1 }),
		}, func(groups []*candidate) {
			for _, group := range groups {
				group.Hot = group.Key.Model == "nomic-embed-text"
			}
		})
		if order[0] != "interactive" {
			t.Fatalf("order = %v, want interactive first", order)
		}
	})

	t.Run("hot model wins at equal density", func(t *testing.T) {
		order := rank([]Job{
			jobSpec("cold", "embed", "owner"),
			jobSpec("hot", "embed", "owner", func(j *Job) { j.ModelAffinity = "winnow:e4b" }),
		}, func(groups []*candidate) {
			for _, group := range groups {
				group.Hot = group.Key.Model == "winnow:e4b"
			}
		})
		if order[0] != "hot" {
			t.Fatalf("order = %v, want the resident model first", order)
		}
	})

	t.Run("value density wins at equal priority and heat", func(t *testing.T) {
		order := rank([]Job{
			jobSpec("cheap", "embed", "owner", func(j *Job) { j.Value = 10; j.Estimate.Seconds = 100 }),
			jobSpec("dense", "embed", "owner", func(j *Job) { j.Value = 10; j.Estimate.Seconds = 10 }),
		}, nil)
		if order[0] != "dense" {
			t.Fatalf("order = %v, want the denser group first", order)
		}
	})

	t.Run("age breaks density ties", func(t *testing.T) {
		order := rank([]Job{
			jobSpec("new", "embed", "owner", func(j *Job) { j.NotBefore = testEpoch.Add(-time.Minute) }),
			jobSpec("old", "embed", "owner", func(j *Job) { j.NotBefore = testEpoch.Add(-time.Hour) }),
		}, nil)
		if order[0] != "old" {
			t.Fatalf("order = %v, want the oldest first", order)
		}
	})

	t.Run("share deficit breaks remaining ties", func(t *testing.T) {
		jobs := []Job{
			jobSpec("heavy", "embed", "a"),
			jobSpec("light", "embed", "b", func(j *Job) { j.ModelAffinity = "winnow:e4b" }),
		}
		groups := groupCandidates(jobs)
		budget := budgetSnapshot{Seconds: 100, ByOwner: map[string]float64{"a": 80, "b": 10}}
		rankCandidates(groups, budget)
		if got := idsOf(groups)[0]; got != "light" {
			t.Fatalf("first = %q, want the owner furthest below its share", got)
		}
	})
}

func TestApplyShareGate(t *testing.T) {
	budget := budgetSnapshot{Seconds: 100, ByOwner: map[string]float64{"a": 90, "b": 10}}
	jobs := []Job{
		jobSpec("a1", "embed", "a"),
		jobSpec("a2", "embed", "a"),
		jobSpec("b1", "embed", "b"),
	}
	policy := Policy{MaxSharePerOwner: 0.5}
	filtered := applyShareGate(jobs, policy, budget)
	if len(filtered) != 1 || filtered[0].Owner != "b" {
		t.Fatalf("filtered = %v, want only owner b", filtered)
	}

	// Every owner over the share: allow everything rather than deadlock.
	allOver := budgetSnapshot{Seconds: 100, ByOwner: map[string]float64{"a": 60, "b": 40}}
	if got := applyShareGate(jobs, Policy{MaxSharePerOwner: 0.3}, allOver); len(got) != 3 {
		t.Fatalf("all-over gate dropped work: %d jobs", len(got))
	}

	// No budget consumed and no configured share both disable the gate.
	if got := applyShareGate(jobs, Policy{MaxSharePerOwner: 0.1}, budgetSnapshot{}); len(got) != 3 {
		t.Fatalf("empty budget dropped work: %d jobs", len(got))
	}
	if got := applyShareGate(jobs, Policy{}, budget); len(got) != 3 {
		t.Fatalf("unset share dropped work: %d jobs", len(got))
	}
}

func TestTrimBatch(t *testing.T) {
	jobs := []Job{
		jobSpec("a", "embed", "owner"),
		jobSpec("b", "embed", "owner"),
		jobSpec("c", "embed", "owner"),
	}
	if got := trimBatch(jobs, 2, 1000); len(got) != 2 {
		t.Fatalf("limit trim = %d jobs, want 2", len(got))
	}
	if got := trimBatch(jobs, 10, 25); len(got) != 2 {
		t.Fatalf("budget trim = %d jobs, want 2 that fit in 25s", len(got))
	}
	if got := trimBatch(jobs, 10, 5); len(got) != 0 {
		t.Fatalf("budget trim = %d jobs, want none when the first job does not fit", len(got))
	}
	if got := trimBatch(jobs, 0, 1000); len(got) != 0 {
		t.Fatalf("zero limit = %d jobs, want none", len(got))
	}
}

func TestAggregateEstimate(t *testing.T) {
	jobs := []Job{
		{Estimate: Estimate{Seconds: 5, RAMBytes: 100, VRAMBytes: 10, Items: 2, Tokens: 100}},
		{Estimate: Estimate{Seconds: 7, RAMBytes: 50, VRAMBytes: 40, Items: 3, Tokens: 200}},
	}
	estimate := aggregateEstimate(jobs)
	want := Estimate{Seconds: 12, RAMBytes: 100, VRAMBytes: 40, Items: 5, Tokens: 300}
	if estimate != want {
		t.Fatalf("aggregate = %+v, want %+v", estimate, want)
	}
}

func TestReadyForModeFiltersPassive(t *testing.T) {
	jobs := []Job{
		jobSpec("batch", "embed", "owner"),
		jobSpec("service", "embed", "owner", func(j *Job) { j.Priority = PriorityService }),
		jobSpec("interactive", "embed", "owner", func(j *Job) { j.Priority = PriorityInteractive }),
	}
	if got := readyForMode(jobs, ModePassive); len(got) != 1 || got[0].ID != "interactive" {
		t.Fatalf("passive filter = %v, want only interactive", describeJobs(got))
	}
	if got := readyForMode(jobs, ModeBalanced); len(got) != 3 {
		t.Fatalf("balanced filter = %d jobs, want 3", len(got))
	}
}

func TestValueDensityFloor(t *testing.T) {
	if got := ValueDensity(10, 0); got != 10 {
		t.Fatalf("zero-second density = %v, want 10", got)
	}
	if got := ValueDensity(-5, 10); got != 0 {
		t.Fatalf("negative value density = %v, want 0", got)
	}
}
