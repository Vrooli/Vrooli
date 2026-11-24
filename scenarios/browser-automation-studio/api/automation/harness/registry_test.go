package harness

import (
	"context"
	"strings"
	"testing"
)

type borrowedStub struct{}

func (borrowedStub) Observe(_ context.Context, req ObserveRequest) (Observation, error) {
	return Observation{Mode: req.Mode}, nil
}
func (borrowedStub) Act(context.Context, ActRequest) (ActionResult, error) {
	return ActionResult{Accepted: true}, nil
}

type accessibilityBorrowedStub struct{ modes []string }

func (s *accessibilityBorrowedStub) Observe(_ context.Context, req ObserveRequest) (Observation, error) {
	s.modes = append(s.modes, req.Mode)
	return Observation{Mode: req.Mode}, nil
}
func (*accessibilityBorrowedStub) Act(context.Context, ActRequest) (ActionResult, error) {
	return ActionResult{Accepted: true}, nil
}

type noTouchBorrowed struct{}

func (noTouchBorrowed) Observe(context.Context, ObserveRequest) (Observation, error) {
	panic("unsupported upstream adapter must not call the borrowed session")
}
func (noTouchBorrowed) Act(context.Context, ActRequest) (ActionResult, error) {
	panic("unsupported upstream adapter must not call the borrowed session")
}

func descriptor(id string) Descriptor {
	return Descriptor{ID: id, Version: "1", ReplayFidelity: "semantic", ObservationModes: []string{"dom"}, ActionVocabulary: []string{"click"}, EffectClasses: []string{"read"}, InferenceRoles: []string{"browser-observe"}}
}

func TestRegistryQualificationUsesBorrowedSessionAndCatalogIsSorted(t *testing.T) {
	r, err := NewRegistry(
		Entry{Descriptor: descriptor("z-adapter")},
		Entry{Descriptor: descriptor("a-adapter"), Probe: func(ctx context.Context, session BorrowedSession) Qualification {
			if _, err := session.Observe(ctx, ObserveRequest{Mode: "navigation"}); err != nil {
				return Qualification{Outcome: OutcomeFailed, Error: err.Error()}
			}
			return Qualification{Outcome: OutcomeQualified, Evidence: []string{"borrowed session observation succeeded"}, Limits: []string{"action path not exercised"}}
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	got := r.List()
	if len(got) != 2 || got[0].ID != "a-adapter" || got[1].ID != "z-adapter" {
		t.Fatalf("registry list = %#v", got)
	}
	qualified := r.Qualify(context.Background(), "a-adapter", borrowedStub{})
	if qualified.Outcome != OutcomeQualified {
		t.Fatalf("qualification = %#v", qualified)
	}
	unsupported := r.Qualify(context.Background(), "z-adapter", borrowedStub{})
	if unsupported.Outcome != OutcomeUnsupported {
		t.Fatalf("catalog entry without probe must be unsupported, got %#v", unsupported)
	}
	noSession := r.Qualify(context.Background(), "a-adapter", nil)
	if noSession.Outcome != OutcomeUnsupported {
		t.Fatalf("nil session qualification = %#v", noSession)
	}
}

func TestRegistryRejectsDuplicateAndUnpinnedUpstream(t *testing.T) {
	r, err := NewRegistry(Entry{Descriptor: descriptor("one")})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Register(Entry{Descriptor: descriptor("one")}); err == nil {
		t.Fatal("expected duplicate id rejection")
	}
	d := descriptor("external")
	d.Upstream = "browser-use"
	if err := r.Register(Entry{Descriptor: d}); err == nil {
		t.Fatal("expected unpinned upstream rejection")
	}
	d.PinnedVersion = "1.0.0"
	if err := r.Register(Entry{Descriptor: d}); err != nil {
		t.Fatalf("pinned adapter registration: %v", err)
	}
}

func TestRegistryReturnsDefensiveDescriptorCopies(t *testing.T) {
	r, err := NewRegistry(Entry{Descriptor: descriptor("one")})
	if err != nil {
		t.Fatal(err)
	}
	listed := r.List()
	listed[0].ObservationModes[0] = "mutated"
	if got := r.List()[0].ObservationModes[0]; got != "dom" {
		t.Fatalf("registry descriptor mutated through List: %q", got)
	}
}

func TestDefaultRegistryQualifiesOnlyAReadOnTheBorrowedSession(t *testing.T) {
	r, err := NewDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	result := r.Qualify(context.Background(), "bas-typed-flow", borrowedStub{})
	if result.Outcome != OutcomeQualified {
		t.Fatalf("default qualification = %#v", result)
	}
	if len(result.Evidence) == 0 || len(result.Limits) == 0 {
		t.Fatalf("qualification must retain evidence and measured limits: %#v", result)
	}
	var nativeModes []string
	catalogued := make(map[string]Descriptor)
	for _, descriptor := range r.List() {
		if descriptor.ID == "bas-typed-flow" {
			nativeModes = descriptor.ObservationModes
		}
		catalogued[descriptor.ID] = descriptor
	}
	if len(nativeModes) != 2 || nativeModes[0] != "navigation" || nativeModes[1] != "screenshot" {
		t.Fatalf("default observation modes = %#v", nativeModes)
	}
	for id, version := range map[string]string{
		"browser-use": "1.0.3", "playwright-mcp": "0.0.85", "playwright-cli": "0.1.25", "stagehand": "4.3.0",
	} {
		d, ok := catalogued[id]
		if !ok || d.PinnedVersion != version {
			t.Fatalf("adapter %q pin = %#v, want %q", id, d, version)
		}
		got := r.Qualify(context.Background(), id, borrowedStub{})
		if got.Outcome != OutcomeUnsupported {
			t.Fatalf("unimplemented adapter %q must remain unsupported, got %#v", id, got)
		}
		if (id == "browser-use" || id == "playwright-mcp" || id == "playwright-cli") && len(got.Limits) == 0 {
			t.Fatalf("adapter %q must explain its current broker-invariant failure, got %#v", id, got)
		}
	}
	for id, wants := range map[string][]string{
		"browser-use":    {"optional exact-pin Python adapter", "removes Python and finish_from_repl", "no service-Agent Browser start/stop or REPL construction"},
		"playwright-mcp": {"createConnection entrypoint", "Exact-entrypoint fake-session tests", "No real BAS task session"},
		"playwright-cli": {"controls a Playwright browser directly"},
	} {
		got := r.Qualify(context.Background(), id, noTouchBorrowed{})
		if got.Outcome != OutcomeUnsupported || len(got.Limits) != 1 {
			t.Errorf("adapter %q qualification=%#v, expected explicit unsupported reason", id, got)
			continue
		}
		for _, want := range wants {
			if !strings.Contains(got.Limits[0], want) {
				t.Errorf("adapter %q qualification=%#v, expected reason containing %q", id, got, want)
			}
		}
	}
	for _, id := range []string{"bas-vision-loop", "raw-cdp"} {
		if _, ok := catalogued[id]; !ok {
			t.Fatalf("candidate adapter %q is not in the matrix", id)
		}
		if got := r.Qualify(context.Background(), id, borrowedStub{}); got.Outcome != OutcomeUnsupported {
			t.Fatalf("unimplemented adapter %q must remain unsupported, got %#v", id, got)
		}
	}
}

func TestAccessibilityFirstAdapterUsesOnlyBrokerBorrowedTreeRead(t *testing.T) {
	r, err := NewDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	session := &accessibilityBorrowedStub{}
	result := r.Qualify(context.Background(), "accessibility-first", session)
	if result.Outcome != OutcomeQualified {
		t.Fatalf("accessibility qualification = %#v", result)
	}
	if len(session.modes) != 1 || session.modes[0] != "accessibility-tree" {
		t.Fatalf("adapter observation modes = %#v", session.modes)
	}
	if len(result.Limits) == 0 || !strings.Contains(result.Limits[0], "read-only") {
		t.Fatalf("read-only limits missing: %#v", result)
	}
	var descriptor Descriptor
	for _, item := range r.List() {
		if item.ID == "accessibility-first" {
			descriptor = item
		}
	}
	if len(descriptor.ActionVocabulary) != 0 || descriptor.ReplayFidelity != "observation-only" {
		t.Fatalf("descriptor overstates adapter support: %#v", descriptor)
	}
}
