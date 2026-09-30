// Package mocks holds test doubles for the egress gate seams.
package mocks

import (
	"context"

	"audio-tools/internal/stt/egress"
)

// FakeStage is a programmable egress.Stage. Tests set Outcome/Reason (and
// optionally a NameValue) and record the decisions the gate passed in.
type FakeStage struct {
	NameValue string
	// Decide, when set, computes the returned decision from the input.
	// When nil, FakeStage emits the input unchanged.
	Decide func(egress.SegmentDecision) egress.SegmentDecision
	// Seen records every decision Apply observed, in order.
	Seen []egress.SegmentDecision
}

// Name returns NameValue, defaulting to "fake".
func (f *FakeStage) Name() string {
	if f.NameValue == "" {
		return "fake"
	}
	return f.NameValue
}

// Apply records the input and returns Decide(in) (or in unchanged).
func (f *FakeStage) Apply(_ context.Context, in egress.SegmentDecision) egress.SegmentDecision {
	f.Seen = append(f.Seen, in)
	if f.Decide != nil {
		return f.Decide(in)
	}
	return in
}

var _ egress.Stage = (*FakeStage)(nil)
