package quality_test

import (
	"context"
	"testing"

	"audio-tools/internal/ai/sttchain"
	sttpkg "audio-tools/internal/stt"
	"audio-tools/internal/stt/quality"
)

func TestPolicyDropsKnownHallucination(t *testing.T) {
	policy := quality.New(sttpkg.Defaults(), nil, nil)
	decision := policy.Apply(context.Background(), "Thanks for watching!", "en", nil, nil)
	if !decision.Filtered {
		t.Fatal("decision.Filtered = false, want true")
	}
	if decision.Text != "" {
		t.Fatalf("decision.Text = %q, want empty", decision.Text)
	}
	if decision.FilterReason != "hallucination" {
		t.Fatalf("decision.FilterReason = %q, want hallucination", decision.FilterReason)
	}
}

func TestPolicyDropsLowConfidenceSilence(t *testing.T) {
	policy := quality.New(sttpkg.Defaults(), nil, nil)
	decision := policy.Apply(context.Background(), "background", "en", &sttchain.Confidence{
		NoSpeechProb: 0.99,
		AvgLogProb:   -2.5,
	}, nil)
	if !decision.Filtered {
		t.Fatal("decision.Filtered = false, want true")
	}
	if decision.FilterReason != "low_confidence" {
		t.Fatalf("decision.FilterReason = %q, want low_confidence", decision.FilterReason)
	}
}

func TestPolicyPreservesLegitimateSpeechWithoutConfidence(t *testing.T) {
	policy := quality.New(sttpkg.Defaults(), nil, nil)
	decision := policy.Apply(context.Background(), "hello world", "en", nil, nil)
	if decision.Filtered {
		t.Fatalf("decision.Filtered = true, reason %q", decision.FilterReason)
	}
	if decision.Text != "hello world" {
		t.Fatalf("decision.Text = %q, want hello world", decision.Text)
	}
}
