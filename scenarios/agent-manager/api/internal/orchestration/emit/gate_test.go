package emit_test

import (
	"errors"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/emit"
)

type recordingSink struct {
	emitted []*domain.RunEvent
	emitErr error
	closed  int
}

func (r *recordingSink) Emit(e *domain.RunEvent) error {
	r.emitted = append(r.emitted, e)
	return r.emitErr
}

func (r *recordingSink) Close() error {
	r.closed++
	return nil
}

type sequencedSink struct {
	*recordingSink
	seq int64
}

func (s *sequencedSink) LastSequence() int64 { return s.seq }

func TestGate_NilGateIsNoOp(t *testing.T) {
	var g *emit.Gate
	if err := g.Emit(&domain.RunEvent{}); err != nil {
		t.Errorf("nil gate Emit returned error: %v", err)
	}
	if err := g.Close(); err != nil {
		t.Errorf("nil gate Close returned error: %v", err)
	}
	if g.LastSequence() != 0 {
		t.Errorf("nil gate LastSequence != 0")
	}
}

func TestGate_NilSinkIsNoOp(t *testing.T) {
	g := emit.NewGate(nil)
	if err := g.Emit(&domain.RunEvent{}); err != nil {
		t.Errorf("nil sink gate Emit returned error: %v", err)
	}
	if err := g.Close(); err != nil {
		t.Errorf("nil sink gate Close returned error: %v", err)
	}
}

func TestGate_ForwardsEmit(t *testing.T) {
	r := &recordingSink{}
	g := emit.NewGate(r)
	evt := &domain.RunEvent{Sequence: 7}
	if err := g.Emit(evt); err != nil {
		t.Fatalf("Emit returned error: %v", err)
	}
	if len(r.emitted) != 1 || r.emitted[0] != evt {
		t.Fatalf("event not forwarded; got %v", r.emitted)
	}
}

func TestGate_PropagatesEmitError(t *testing.T) {
	want := errors.New("persist failed")
	r := &recordingSink{emitErr: want}
	g := emit.NewGate(r)
	if err := g.Emit(&domain.RunEvent{}); !errors.Is(err, want) {
		t.Fatalf("expected wrapped sink error, got %v", err)
	}
}

func TestGate_ForwardsClose(t *testing.T) {
	r := &recordingSink{}
	g := emit.NewGate(r)
	if err := g.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if r.closed != 1 {
		t.Fatalf("Close not forwarded once; got %d", r.closed)
	}
}

func TestGate_LastSequenceForwardsForSequencedSink(t *testing.T) {
	s := &sequencedSink{recordingSink: &recordingSink{}, seq: 42}
	g := emit.NewGate(s)
	if got := g.LastSequence(); got != 42 {
		t.Fatalf("LastSequence not forwarded; got %d", got)
	}
}

func TestGate_LastSequenceZeroForNonSequencedSink(t *testing.T) {
	g := emit.NewGate(&recordingSink{})
	if got := g.LastSequence(); got != 0 {
		t.Fatalf("LastSequence on non-sequenced sink should be 0; got %d", got)
	}
}

func TestGate_SatisfiesEventSink(t *testing.T) {
	var _ runner.EventSink = (*emit.Gate)(nil)
}
