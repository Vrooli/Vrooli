package composition

import (
	"context"
	"errors"
	"math"
	"testing"
)

func TestPlanFramesUsesLatestSourcePTSAtOrBeforeExactTarget(t *testing.T) {
	plan, err := PlanFrames(context.Background(), FrameClock{
		SourcePTS: []int64{0, 2, 4}, SourceTimeBase: Rational{Num: 1, Den: 10}, SourceEndPTS: 6,
		TrimStartPTS: 0, TrimEndPTS: 6, PlaybackRate: Rational{Num: 1, Den: 1}, OutputRate: Rational{Num: 10, Den: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []int64{0, 0, 2, 2, 4, 4}
	wantIndex := []int{0, 0, 1, 1, 2, 2}
	if len(plan) != len(want) {
		t.Fatalf("frame count = %d, want %d", len(plan), len(want))
	}
	for i, sample := range plan {
		if sample.OutputIndex != int64(i) || sample.SourcePTS != want[i] || sample.SourceIndex != wantIndex[i] {
			t.Errorf("frame %d = %+v, want source index %d PTS %d", i, sample, wantIndex[i], want[i])
		}
		if sample.OutputTimeSeconds.Num*10 != int64(i)*sample.OutputTimeSeconds.Den {
			t.Errorf("frame %d output time = %+v", i, sample.OutputTimeSeconds)
		}
		if sample.TargetSourcePTS.Num != int64(i) || sample.TargetSourcePTS.Den != 1 {
			t.Errorf("frame %d target PTS = %+v", i, sample.TargetSourcePTS)
		}
	}
}

func TestPlanFramesSupportsVFRRepeatedPTSAndRationalPlayback(t *testing.T) {
	plan, err := PlanFrames(context.Background(), FrameClock{
		SourcePTS: []int64{0, 0, 3, 7, 10}, SourceTimeBase: Rational{Num: 1, Den: 10}, SourceEndPTS: 12,
		TrimStartPTS: 0, TrimEndPTS: 10, PlaybackRate: Rational{Num: 2, Den: 1}, OutputRate: Rational{Num: 5, Den: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []int64{0, 3, 7}
	wantIndex := []int{1, 2, 3} // At duplicate PTS zero, the final equal-PTS sample wins deterministically.
	if len(plan) != len(want) {
		t.Fatalf("frame count = %d, want %d", len(plan), len(want))
	}
	for i, sample := range plan {
		if sample.SourcePTS != want[i] || sample.SourceIndex != wantIndex[i] {
			t.Errorf("frame %d selected index/PTS %d/%d, want %d/%d", i, sample.SourceIndex, sample.SourcePTS, wantIndex[i], want[i])
		}
	}
}

func TestPlanFramesRejectsTrimmedEOFAcrossEndUnlessHoldRequested(t *testing.T) {
	clock := FrameClock{SourcePTS: []int64{0, 4}, SourceTimeBase: Rational{Num: 1, Den: 10}, SourceEndPTS: 6,
		TrimStartPTS: 4, TrimEndPTS: 9, PlaybackRate: Rational{Num: 1, Den: 1}, OutputRate: Rational{Num: 10, Den: 1}}
	if _, err := PlanFrames(context.Background(), clock); err == nil {
		t.Fatal("expected explicit EOF policy to be required")
	}
	clock.EOF = RangeHoldLast
	plan, err := PlanFrames(context.Background(), clock)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan) != 5 || plan[0].SourcePTS != 4 || plan[4].SourcePTS != 4 || plan[4].SourceIndex != 1 {
		t.Fatalf("hold-last plan = %+v", plan)
	}
}

func TestPlanFramesRequiresExplicitUnderflowPolicy(t *testing.T) {
	clock := FrameClock{SourcePTS: []int64{2, 4}, SourceTimeBase: Rational{Num: 1, Den: 10}, SourceEndPTS: 6,
		TrimStartPTS: 0, TrimEndPTS: 2, PlaybackRate: Rational{Num: 1, Den: 1}, OutputRate: Rational{Num: 10, Den: 1}}
	if _, err := PlanFrames(context.Background(), clock); err == nil {
		t.Fatal("expected underflow refusal")
	}
	clock.Underflow = RangeHoldFirst
	plan, err := PlanFrames(context.Background(), clock)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan) != 2 || plan[0].SourcePTS != 2 || plan[1].SourcePTS != 2 || plan[0].SourceIndex != 0 {
		t.Fatalf("hold-first plan = %+v", plan)
	}
}

func TestPlanFramesRejectsInvalidAndUnboundedInputs(t *testing.T) {
	base := FrameClock{SourcePTS: []int64{0, 4}, SourceTimeBase: Rational{Num: 1, Den: 10}, SourceEndPTS: 6,
		TrimStartPTS: 0, TrimEndPTS: 4, PlaybackRate: Rational{Num: 1, Den: 1}, OutputRate: Rational{Num: 10, Den: 1}}
	cases := []struct {
		name   string
		mutate func(*FrameClock)
	}{
		{"unordered", func(c *FrameClock) { c.SourcePTS = []int64{0, 4, 3} }},
		{"bad-time-base", func(c *FrameClock) { c.SourceTimeBase = Rational{Num: 0, Den: 1} }},
		{"bad-trim", func(c *FrameClock) { c.TrimEndPTS = c.TrimStartPTS }},
		{"unknown-policy", func(c *FrameClock) { c.EOF = RangePolicy(99) }},
		{"wrong-underflow-policy", func(c *FrameClock) { c.Underflow = RangeHoldLast }},
		{"wrong-eof-policy", func(c *FrameClock) { c.EOF = RangeHoldFirst }},
		{"too-many-frames", func(c *FrameClock) {
			c.OutputRate = Rational{Num: (maxPlannedFrames + 1) * 3, Den: 1}
			c.TrimEndPTS = 4
		}},
		{"large-integer-overflow", func(c *FrameClock) { c.OutputRate = Rational{Num: math.MaxInt64, Den: 1}; c.TrimEndPTS = math.MaxInt64 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := base
			c.SourcePTS = append([]int64(nil), base.SourcePTS...)
			tc.mutate(&c)
			if _, err := PlanFrames(context.Background(), c); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestPlanFramesHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	clock := FrameClock{SourcePTS: []int64{0}, SourceTimeBase: Rational{Num: 1, Den: 24}, SourceEndPTS: 24,
		TrimStartPTS: 0, TrimEndPTS: 24, PlaybackRate: Rational{Num: 1, Den: 1}, OutputRate: Rational{Num: 24, Den: 1}}
	if _, err := PlanFrames(ctx, clock); !errors.Is(err, context.Canceled) {
		t.Fatalf("PlanFrames error = %v, want context.Canceled", err)
	}
}
