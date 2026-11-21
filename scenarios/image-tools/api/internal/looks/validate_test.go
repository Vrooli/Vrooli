package looks

import (
	"errors"
	"strings"
	"testing"

	looksv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/looks"
)

func look(name string, steps ...*looksv1.LookStep) *looksv1.Look {
	return &looksv1.Look{Id: "test", Name: name, Steps: steps}
}

func step(op string, kind looksv1.StepKind) *looksv1.LookStep {
	return &looksv1.LookStep{Operation: op, Kind: kind}
}

// TestBuiltinLooksValidate keeps the seeded library honest: every shipped Look
// must satisfy the same chain rules a custom Look does.
func TestBuiltinLooksValidate(t *testing.T) {
	for _, l := range BuiltinLooks() {
		if err := Validate(l); err != nil {
			t.Errorf("built-in Look %q is invalid: %v", l.GetId(), err)
		}
	}
}

func TestValidateAcceptsLinearImageChain(t *testing.T) {
	l := look("chain",
		step("resize", looksv1.StepKind_STEP_KIND_DETERMINISTIC),
		step("upscale", looksv1.StepKind_STEP_KIND_AI),
	)
	if err := Validate(l); err != nil {
		t.Fatalf("valid chain rejected: %v", err)
	}
}

func TestValidateRejectsUnknownOperation(t *testing.T) {
	err := Validate(look("x", step("not_an_op", looksv1.StepKind_STEP_KIND_DETERMINISTIC)))
	if err == nil || !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

// ocr exists in the operation vocabulary but has no durable AI/deterministic
// runner, so it cannot be a Look step.
func TestValidateRejectsNonExecutableOperation(t *testing.T) {
	err := Validate(look("x", step("ocr", looksv1.StepKind_STEP_KIND_UNSPECIFIED)))
	if err == nil || !strings.Contains(err.Error(), "not available as a workflow step") {
		t.Fatalf("expected non-executable rejection, got %v", err)
	}
}

func TestValidateRejectsWrongLane(t *testing.T) {
	err := Validate(look("x", step("upscale", looksv1.StepKind_STEP_KIND_DETERMINISTIC)))
	if err == nil || !strings.Contains(err.Error(), "does not match the operation's lane") {
		t.Fatalf("expected lane mismatch, got %v", err)
	}
}

// The bug this validation exists for: a text output fed into an image input.
func TestValidateRejectsImageAfterNonImage(t *testing.T) {
	l := look("bad",
		step("caption", looksv1.StepKind_STEP_KIND_AI),
		step("upscale", looksv1.StepKind_STEP_KIND_AI),
	)
	err := Validate(l)
	if err == nil || !strings.Contains(err.Error(), "needs an image input") {
		t.Fatalf("expected image-after-text rejection, got %v", err)
	}
}

func TestValidateAllowsExternalMask(t *testing.T) {
	// inpaint needs image + mask at step 0; the mask is supplied externally.
	l := look("masked", step("inpaint", looksv1.StepKind_STEP_KIND_AI))
	if err := Validate(l); err != nil {
		t.Fatalf("masked first step rejected: %v", err)
	}
}

func TestValidateRejectsUnknownInputPort(t *testing.T) {
	st := step("resize", looksv1.StepKind_STEP_KIND_DETERMINISTIC)
	st.Inputs = []*looksv1.StepInput{{Name: "bogus"}}
	err := Validate(look("x", st))
	if err == nil || !strings.Contains(err.Error(), "which the operation does not have") {
		t.Fatalf("expected unknown-port rejection, got %v", err)
	}
}

func TestValidateRejectsOutputKindMismatch(t *testing.T) {
	st := step("upscale", looksv1.StepKind_STEP_KIND_AI)
	st.OutputKind = "text"
	err := Validate(look("x", st))
	if err == nil || !strings.Contains(err.Error(), "declared output kind") {
		t.Fatalf("expected output-kind mismatch, got %v", err)
	}
}

func TestValidateRejectsCrossScenarioStep(t *testing.T) {
	st := step("tts", looksv1.StepKind_STEP_KIND_AI)
	st.Scenario = "audio-tools"
	err := Validate(look("x", st))
	if err == nil || !strings.Contains(err.Error(), "cross-scenario step") {
		t.Fatalf("expected cross-scenario rejection, got %v", err)
	}
}

func TestValidateAllowsSourceWiringAtLaterStep(t *testing.T) {
	// A later step may explicitly take the original image instead of the previous
	// output (e.g. blend semantics).
	l := look("wired",
		step("filter", looksv1.StepKind_STEP_KIND_DETERMINISTIC),
		&looksv1.LookStep{
			Operation: "adjust",
			Kind:      looksv1.StepKind_STEP_KIND_DETERMINISTIC,
			Inputs:    []*looksv1.StepInput{{Name: "image", From: "source"}},
		},
	)
	if err := Validate(l); err != nil {
		t.Fatalf("source wiring rejected: %v", err)
	}
}

// Compile derives the lane from the catalogs, so an omitted kind is resolved
// correctly even for compute-mechanism ops that run on the AI job lane.
func TestCompileDerivesLaneAndPorts(t *testing.T) {
	l := look("derive",
		&looksv1.LookStep{Operation: "naturalize"},
		&looksv1.LookStep{Operation: "resize"},
	)
	out := Compile(l, "", "", true)
	if len(out.GetSteps()) != 2 {
		t.Fatalf("steps = %d", len(out.GetSteps()))
	}
	if got := out.GetSteps()[0].GetKind(); got != looksv1.StepKind_STEP_KIND_AI {
		t.Errorf("naturalize lane = %v, want AI", got)
	}
	if got := out.GetSteps()[1].GetKind(); got != looksv1.StepKind_STEP_KIND_DETERMINISTIC {
		t.Errorf("resize lane = %v, want deterministic", got)
	}
	if got := out.GetSteps()[0].GetOutputKind(); got != "image" {
		t.Errorf("naturalize output kind = %q, want image", got)
	}
	if got := out.GetSteps()[0].GetScenario(); got != ImageToolsScenario {
		t.Errorf("scenario = %q, want %q", got, ImageToolsScenario)
	}
}
