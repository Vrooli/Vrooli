package recipe

import (
	"bytes"
	"context"
	"testing"

	inteval "audio-tools/internal/eval"
)

func TestApplyAugmentationDeterministicNoise(t *testing.T) {
	base := []inteval.Clip{{ID: "clip", PCM: bytes.Repeat([]byte{0x20, 0x03}, 16000), SampleRate: CanonicalSampleRate, Reference: "hello", Format: "pcm_s16le"}}
	spec := AugmentationSpec{Seed: 7, NoiseTypes: []string{"white", "fan"}, SNRDB: []float64{18, 6}}
	first, firstCond, err := ApplyAugmentation(context.Background(), base, spec)
	if err != nil {
		t.Fatal(err)
	}
	second, secondCond, err := ApplyAugmentation(context.Background(), base, spec)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != len(second) || len(firstCond) != len(secondCond) {
		t.Fatalf("different realization sizes")
	}
	for i := range first {
		if first[i].ID != second[i].ID || !bytes.Equal(first[i].PCM, second[i].PCM) {
			t.Fatalf("condition %d differs between seeded runs", i)
		}
	}
}

func TestApplyAugmentationRecordsCompetingVoiceSkip(t *testing.T) {
	base := []inteval.Clip{{ID: "clip", PCM: bytes.Repeat([]byte{1, 0}, 100), SampleRate: CanonicalSampleRate, Reference: "hello", Format: "pcm_s16le"}}
	_, conditions, err := ApplyAugmentation(context.Background(), base, AugmentationSpec{CompetingVoices: []string{"af_bella"}})
	if err != nil {
		t.Fatal(err)
	}
	var skipped bool
	for _, c := range conditions {
		if c.Kind == "competing_voice" && c.Skipped {
			skipped = true
		}
	}
	if !skipped {
		t.Fatalf("expected skipped competing voice condition")
	}
}

func TestGroupClipsByAugConditionGroupsAcrossBaseClips(t *testing.T) {
	clips := []inteval.Clip{
		{ID: "clip-a", Reference: "clean a"},
		{ID: "clip-b", Reference: "clean b"},
		{ID: "clip-a/noise:fan/6db", Reference: "fan a"},
		{ID: "clip-b/noise:fan/6db", Reference: "fan b"},
		{ID: "clip-a/noise:white/15db", Reference: "white a"},
	}

	groups := GroupClipsByAugCondition(clips)
	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d: %#v", len(groups), groups)
	}
	if groups[0].ID != "clean" || len(groups[0].Clips) != 2 {
		t.Fatalf("clean group = %#v, want two clean clips", groups[0])
	}
	if groups[1].ID != "noise:fan/6db" || len(groups[1].Clips) != 2 {
		t.Fatalf("fan group = %#v, want two fan clips", groups[1])
	}
	if groups[2].ID != "noise:white/15db" || len(groups[2].Clips) != 1 {
		t.Fatalf("white group = %#v, want one white clip", groups[2])
	}
}
