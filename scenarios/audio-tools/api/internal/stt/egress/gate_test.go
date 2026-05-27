package egress_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/stt/egress"
	"audio-tools/internal/stt/egress/mocks"
)

func TestGateEmptyIsIdentity(t *testing.T) {
	g := egress.NewGate()
	out := g.Apply(context.Background(), egress.SegmentDecision{Text: "hi"})
	require.Equal(t, egress.Emit, out.Outcome)
	require.Equal(t, "hi", out.Text)
}

func TestGateNilIsIdentity(t *testing.T) {
	var g *egress.Gate
	out := g.Apply(context.Background(), egress.SegmentDecision{Text: "hi"})
	require.Equal(t, egress.Emit, out.Outcome)
}

func TestGateStopsAtFirstTerminalOutcome(t *testing.T) {
	first := &mocks.FakeStage{NameValue: "first", Decide: func(d egress.SegmentDecision) egress.SegmentDecision {
		d.Outcome = egress.Drop
		d.Reason = "first-drop"
		return d
	}}
	second := &mocks.FakeStage{NameValue: "second"}
	g := egress.NewGate(first, second)

	out := g.Apply(context.Background(), egress.SegmentDecision{Text: "x"})
	require.Equal(t, egress.Drop, out.Outcome)
	require.Equal(t, "first-drop", out.Reason)
	require.Len(t, first.Seen, 1, "first stage ran")
	require.Empty(t, second.Seen, "second stage skipped after terminal outcome")
	require.Equal(t, []string{"first", "second"}, g.Stages())
}

func TestHallucinationStage(t *testing.T) {
	known := map[string]bool{"thank you for watching": true}
	stage := egress.HallucinationStage{IsHallucination: func(s string) bool { return known[s] }}

	drop := stage.Apply(context.Background(), egress.SegmentDecision{Text: "thank you for watching", Outcome: egress.Emit})
	require.Equal(t, egress.Drop, drop.Outcome)
	require.Equal(t, "hallucination", drop.Reason)

	keep := stage.Apply(context.Background(), egress.SegmentDecision{Text: "real speech", Outcome: egress.Emit})
	require.Equal(t, egress.Emit, keep.Outcome)
}

func TestHallucinationStageNilPredicateNoop(t *testing.T) {
	stage := egress.HallucinationStage{}
	out := stage.Apply(context.Background(), egress.SegmentDecision{Text: "anything", Outcome: egress.Emit})
	require.Equal(t, egress.Emit, out.Outcome)
}

func TestConfidenceStage(t *testing.T) {
	stage := egress.ConfidenceStage{NoSpeechThreshold: 0.6, LogProbThreshold: -1.0}

	cases := []struct {
		name string
		conf *sttchain.Confidence
		want egress.Outcome
	}{
		{"nil confidence skips", nil, egress.Emit},
		{"both thresholds crossed drops", &sttchain.Confidence{NoSpeechProb: 0.8, AvgLogProb: -1.5}, egress.Drop},
		{"only no_speech crossed keeps", &sttchain.Confidence{NoSpeechProb: 0.8, AvgLogProb: -0.5}, egress.Emit},
		{"only logprob crossed keeps", &sttchain.Confidence{NoSpeechProb: 0.2, AvgLogProb: -1.5}, egress.Emit},
		{"confident speech keeps", &sttchain.Confidence{NoSpeechProb: 0.1, AvgLogProb: -0.3}, egress.Emit},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := stage.Apply(context.Background(), egress.SegmentDecision{
				Text:       "candidate",
				Confidence: tc.conf,
				Outcome:    egress.Emit,
			})
			require.Equal(t, tc.want, out.Outcome)
		})
	}
}
