package cliapp

import (
	"errors"
	"strings"
	"testing"
)

// handle is a trivial started-run handle for the skeleton tests.
type drHandle struct{ id string }

func TestRunDurable_StartsOnceAndDispatchesByMode(t *testing.T) {
	for _, tc := range []struct {
		name string
		mode DurableRunMode
		want string
	}{
		{"human", DurableRunHuman, "human"},
		{"json", DurableRunJSON, "json"},
		{"jsonl", DurableRunJSONL, "jsonl"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			starts := 0
			var rendered string
			spec := DurableRunSpec[drHandle]{
				Start: func() (drHandle, error) { starts++; return drHandle{id: "r1"}, nil },
				Human: func(h drHandle) error { rendered = "human:" + h.id; return nil },
				JSON:  func(h drHandle) error { rendered = "json:" + h.id; return nil },
				JSONL: func(h drHandle) error { rendered = "jsonl:" + h.id; return nil },
			}
			if err := RunDurable(tc.mode, spec); err != nil {
				t.Fatalf("RunDurable: %v", err)
			}
			if starts != 1 {
				t.Fatalf("Start must run exactly once, ran %d", starts)
			}
			if !strings.HasPrefix(rendered, tc.want) {
				t.Fatalf("mode %v rendered %q, want prefix %q", tc.mode, rendered, tc.want)
			}
		})
	}
}

func TestRunDurable_StartErrorRoutedToRenderer(t *testing.T) {
	sentinel := errors.New("start boom")
	var gotMode DurableRunMode
	renderErrCalled := false
	spec := DurableRunSpec[drHandle]{
		Start:            func() (drHandle, error) { return drHandle{}, sentinel },
		RenderStartError: func(mode DurableRunMode, err error) error { gotMode = mode; renderErrCalled = true; return err },
		Human:            func(drHandle) error { t.Fatal("Human must not run after start error"); return nil },
		JSON:             func(drHandle) error { t.Fatal("JSON must not run after start error"); return nil },
		JSONL:            func(drHandle) error { return nil },
	}
	if err := RunDurable(DurableRunJSON, spec); !errors.Is(err, sentinel) {
		t.Fatalf("expected start error, got %v", err)
	}
	if !renderErrCalled || gotMode != DurableRunJSON {
		t.Fatalf("RenderStartError should receive the mode; called=%v mode=%v", renderErrCalled, gotMode)
	}
}

func TestRunDurable_NilRenderStartErrorReturnsRawError(t *testing.T) {
	sentinel := errors.New("raw")
	spec := DurableRunSpec[drHandle]{
		Start: func() (drHandle, error) { return drHandle{}, sentinel },
		Human: func(drHandle) error { return nil },
	}
	if err := RunDurable(DurableRunHuman, spec); !errors.Is(err, sentinel) {
		t.Fatalf("nil RenderStartError should return the raw start error, got %v", err)
	}
}

func TestDurableRunModeFrom(t *testing.T) {
	if DurableRunModeFrom(false, false) != DurableRunHuman {
		t.Fatalf("no flags -> human")
	}
	if DurableRunModeFrom(true, false) != DurableRunJSON {
		t.Fatalf("--json -> json")
	}
	if DurableRunModeFrom(false, true) != DurableRunJSONL {
		t.Fatalf("--jsonl -> jsonl")
	}
	if DurableRunModeFrom(true, true) != DurableRunJSONL {
		t.Fatalf("--json --jsonl -> jsonl (event stream is more specific)")
	}
}

func TestDurableRunLegacy_CarriesEvidence(t *testing.T) {
	ran := false
	lph := DurableRunLegacy(func(args []string) error { ran = true; return nil })
	if lph.Primitive() != PrimitiveDurableRun {
		t.Fatalf("DurableRunLegacy evidence = %q, want durable_run", lph.Primitive())
	}
	cmd := Command{Name: "execute"}.WithLegacyPrimitive(lph)
	if cmd.PrimitiveEvidence() != PrimitiveDurableRun {
		t.Fatalf("WithLegacyPrimitive must set PrimitiveEvidence to durable_run, got %q", cmd.PrimitiveEvidence())
	}
	if cmd.Run == nil || cmd.RunCtx != nil {
		t.Fatalf("WithLegacyPrimitive must set the legacy Run handler (not RunCtx)")
	}
	if err := cmd.Run(nil); err != nil || !ran {
		t.Fatalf("legacy run should be wired, err=%v ran=%v", err, ran)
	}
}

// TestDurableRunEvidence_FlowsIntoArtifact proves a top-level durable_run command
// exports durable_run observed evidence (and reconciles with a declared exception
// without contradiction).
func TestDurableRunEvidence_FlowsIntoArtifact(t *testing.T) {
	cmd := Command{
		Name:         "execute",
		Architecture: CommandArchitecture{Exception: ExceptionDurableRun, ExceptionReason: "server-owned durable run"},
	}.WithLegacyPrimitive(DurableRunLegacy(func([]string) error { return nil }))

	artifact, err := BuildPrimitiveEvidence(EvidenceExportInput{
		Scenario: "test-genie",
		TopLevel: []Command{cmd},
	})
	if err != nil {
		t.Fatalf("BuildPrimitiveEvidence: %v", err)
	}
	if len(artifact.Commands) != 1 {
		t.Fatalf("want 1 command, got %d", len(artifact.Commands))
	}
	got := artifact.Commands[0]
	if got.ObservedPrimitive != string(PrimitiveDurableRun) {
		t.Fatalf("execute observed primitive = %q, want durable_run", got.ObservedPrimitive)
	}
	if got.DeclaredException != string(ExceptionDurableRun) {
		t.Fatalf("execute declared exception = %q, want durable_run", got.DeclaredException)
	}
}
