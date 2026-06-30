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
