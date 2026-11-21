package proposals

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestProposalTransitionOwnerRejectsInvalidAndTerminalReversal(t *testing.T) {
	for _, tc := range []struct {
		name string
		from State
		to   State
	}{
		{name: "unknown source", from: State("mystery"), to: StateOpen},
		{name: "unknown target", from: StateOpen, to: State("mystery")},
		{name: "committed reversal", from: StateCommitted, to: StateOpen},
		{name: "withdrawn reversal", from: StateWithdrawn, to: StateSuperseded},
		{name: "superseded reversal", from: StateSuperseded, to: StateCommitted},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := Proposal{State: tc.from}
			if err := transitionState(&p, tc.to); !errors.Is(err, ErrInvalid) {
				t.Fatalf("transition error = %v, want ErrInvalid", err)
			}
			if p.State != tc.from {
				t.Fatalf("rejected transition changed state to %q", p.State)
			}
		})
	}
}

func TestProposalTransitionOwnerAllowsOnlyOpenLifecycleEdges(t *testing.T) {
	for _, target := range []State{StateCommitted, StateWithdrawn, StateSuperseded} {
		p := Proposal{State: StateOpen, Revision: 7}
		if err := transitionState(&p, target); err != nil || p.State != target || p.Revision != 7 {
			t.Fatalf("open -> %s = %#v, err=%v", target, p, err)
		}
	}
}

func TestProposalStoreRejectsInvalidStateAndStaleUpdate(t *testing.T) {
	svc := testService(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	p := Proposal{ID: "p1", RepositoryID: "1", State: StateOpen, Revision: 1, CreatedAt: now, UpdatedAt: now}
	if err := svc.store.Insert(ctx, p); err != nil {
		t.Fatal(err)
	}
	invalid := p
	invalid.State = State("unknown")
	if err := svc.store.Update(ctx, invalid, 1, StateOpen); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid-state update error = %v", err)
	}
	first := p
	first.Revision = 2
	first.Message.Subject = "newer review"
	if err := svc.store.Update(ctx, first, 1, StateOpen); err != nil {
		t.Fatal(err)
	}
	stale := p
	stale.Revision = 2
	stale.Message.Subject = "stale review"
	if err := svc.store.Update(ctx, stale, 1, StateOpen); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale update error = %v, want ErrConflict", err)
	}
	got, err := svc.store.Get(ctx, p.ID)
	if err != nil || got.Revision != 2 || got.Message.Subject != "newer review" {
		t.Fatalf("stored proposal = %#v, err=%v", got, err)
	}
}

func TestPublishFailureRecoversAsOpenProposal(t *testing.T) {
	svc := testService(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	p := Proposal{ID: "recover", RepositoryID: "1", State: StateOpen, Revision: 3, CreatedAt: now, UpdatedAt: now}
	if err := svc.store.Insert(ctx, p); err != nil {
		t.Fatal(err)
	}
	_, returned, err := svc.store.Publish(ctx, p, humanActor, func(context.Context) (string, error) {
		return "", errors.New("candidate publication failed")
	}, now.Add(time.Minute))
	if err == nil || returned.State != StateOpen {
		t.Fatalf("failed publish returned %#v, err=%v", returned, err)
	}
	got, err := svc.store.Get(ctx, p.ID)
	if err != nil || got.State != StateOpen || got.Revision != p.Revision {
		t.Fatalf("failed publication recovery = %#v, err=%v", got, err)
	}
}
