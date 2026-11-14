package cliapp

import (
	"bytes"
	"strings"
	"testing"
)

func TestPassthrough_CarriesEvidenceAndStreamsOutput(t *testing.T) {
	handler := Passthrough(func(ctx OperationContext) (PassthroughSpec, error) {
		return PassthroughSpec{
			Command: "sh",
			Args:    []string{"-c", "printf passthrough:%s \"$1\"", "sh", ctx.Positional("value")},
		}, nil
	})
	if handler.Primitive() != PrimitivePassthrough {
		t.Fatalf("Passthrough evidence = %q, want passthrough", handler.Primitive())
	}

	var out bytes.Buffer
	ctx := NewTestRunContext(TestRunContextOptions{
		Schema:      ArgSchema{Positionals: []Positional{{Name: "value", Required: true}}},
		Positionals: map[string]string{"value": "ok"},
		Stdout:      &out,
	})

	if err := handler.Run(ctx); err != nil {
		t.Fatalf("Passthrough.Run: %v", err)
	}
	if got := out.String(); got != "passthrough:ok" {
		t.Fatalf("stdout = %q", got)
	}
}

func TestRunPassthroughRejectsEmptyCommand(t *testing.T) {
	if err := RunPassthrough(PassthroughSpec{}, nil, nil); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("expected empty command error, got %v", err)
	}
}

func TestPassthroughLegacy_CarriesEvidence(t *testing.T) {
	ran := false
	lph := PassthroughLegacy(func(args []string) error {
		ran = len(args) == 1 && args[0] == "x"
		return nil
	})
	if lph.Primitive() != PrimitivePassthrough {
		t.Fatalf("PassthroughLegacy evidence = %q, want passthrough", lph.Primitive())
	}
	cmd := Command{Name: "delegate"}.WithLegacyPrimitive(lph)
	if cmd.PrimitiveEvidence() != PrimitivePassthrough {
		t.Fatalf("WithLegacyPrimitive evidence = %q, want passthrough", cmd.PrimitiveEvidence())
	}
	if err := cmd.Run([]string{"x"}); err != nil || !ran {
		t.Fatalf("legacy passthrough run not wired: err=%v ran=%v", err, ran)
	}
}
