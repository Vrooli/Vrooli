package recipe

import (
	"bytes"
	"testing"
)

func TestBuildDeterministicFromSeedAndCorpus(t *testing.T) {
	clips := []Clip{
		{ID: "b", PCM: bytes.Repeat([]byte{2, 0}, 1600), SampleRate: CanonicalSampleRate, Reference: "bravo", Format: "pcm_s16le"},
		{ID: "a", PCM: bytes.Repeat([]byte{1, 0}, 1600), SampleRate: CanonicalSampleRate, Reference: "alpha", Format: "pcm_s16le"},
	}

	first, firstPlan, err := Build(Spec{Seed: 42, TargetDurationSeconds: 1, GapMs: 250}, clips)
	if err != nil {
		t.Fatal(err)
	}
	second, secondPlan, err := Build(Spec{Seed: 42, TargetDurationSeconds: 1, GapMs: 250}, []Clip{clips[1], clips[0]})
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(first.PCM, second.PCM) {
		t.Fatalf("same seed/corpus produced different audio bytes")
	}
	if first.Reference != second.Reference {
		t.Fatalf("same seed/corpus produced different reference: %q vs %q", first.Reference, second.Reference)
	}
	if !equalStrings(firstPlan.ClipIDs, secondPlan.ClipIDs) {
		t.Fatalf("same seed/corpus produced different order: %v vs %v", firstPlan.ClipIDs, secondPlan.ClipIDs)
	}
}

func TestBuildInsertsZeroGapAndReferenceSpaces(t *testing.T) {
	clipA := []byte{1, 0, 1, 0}
	clipB := []byte{2, 0, 2, 0}
	got, plan, err := Build(Spec{Seed: 1, GapMs: 100}, []Clip{
		{ID: "a", PCM: clipA, SampleRate: CanonicalSampleRate, Reference: "alpha"},
		{ID: "b", PCM: clipB, SampleRate: CanonicalSampleRate, Reference: "bravo"},
	})
	if err != nil {
		t.Fatal(err)
	}
	gapBytes := CanonicalSampleRate * 2 * 100 / 1000
	if len(got.PCM) != len(clipA)+len(clipB)+gapBytes {
		t.Fatalf("unexpected pcm length: got %d want %d", len(got.PCM), len(clipA)+len(clipB)+gapBytes)
	}
	if plan.Reference != got.Reference || got.Reference == "" {
		t.Fatalf("reference not recorded: clip=%q plan=%q", got.Reference, plan.Reference)
	}
	if bytes.Contains(got.PCM[len(clipA):len(clipA)+gapBytes], []byte{1}) || bytes.Contains(got.PCM[len(clipA):len(clipA)+gapBytes], []byte{2}) {
		t.Fatalf("gap contains non-silence bytes")
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
