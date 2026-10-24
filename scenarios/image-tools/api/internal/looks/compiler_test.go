package looks

import (
	"strings"
	"testing"

	looksv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/looks"
)

func builtinByID(t *testing.T, id string) *looksv1.Look {
	t.Helper()
	for _, l := range BuiltinLooks() {
		if l.GetId() == id {
			return l
		}
	}
	t.Fatalf("no built-in Look %q", id)
	return nil
}

// TestCompileStyleLookFillsTemplate proves a STYLE Look resolves its prompt
// template with the subject, sets the primary prompt, flags requires_image, and
// warns about the model-backed step.
func TestCompileStyleLookFillsTemplate(t *testing.T) {
	out := Compile(builtinByID(t, "anime"), "a golden retriever", "soft lighting", true)
	if len(out.GetSteps()) != 1 {
		t.Fatalf("want 1 compiled step, got %d", len(out.GetSteps()))
	}
	step := out.GetSteps()[0]
	if step.GetOperation() != "edit_instruct" || step.GetKind() != looksv1.StepKind_STEP_KIND_AI {
		t.Fatalf("unexpected step %v", step)
	}
	prompt := step.GetParams()["prompt"]
	if !strings.Contains(prompt, "a golden retriever") {
		t.Errorf("prompt should contain the subject, got %q", prompt)
	}
	if !strings.Contains(prompt, "soft lighting") {
		t.Errorf("prompt should contain the free-form addition, got %q", prompt)
	}
	if out.GetPrimaryPrompt() != prompt {
		t.Errorf("primary prompt %q != step prompt %q", out.GetPrimaryPrompt(), prompt)
	}
	// Look default param (strength) merged in.
	if step.GetParams()["strength"] != "0.6" {
		t.Errorf("expected merged default strength=0.6, got %q", step.GetParams()["strength"])
	}
	if !out.GetRequiresImage() {
		t.Error("edit_instruct requires an input image")
	}
	if !hasWarningContaining(out.GetWarnings(), "model-backed") {
		t.Errorf("expected a model-backed warning, got %v", out.GetWarnings())
	}
}

// TestCompileDeterministicLookNoAIWarning proves a pure film Look compiles to
// deterministic steps, requires an image, and carries no model warning.
func TestCompileDeterministicLookNoAIWarning(t *testing.T) {
	out := Compile(builtinByID(t, "noir"), "", "", true)
	if !out.GetRequiresImage() {
		t.Error("a deterministic edit requires an input image")
	}
	if hasWarningContaining(out.GetWarnings(), "model-backed") {
		t.Errorf("a pure-deterministic Look must not warn about models, got %v", out.GetWarnings())
	}
	for _, s := range out.GetSteps() {
		if s.GetKind() != looksv1.StepKind_STEP_KIND_DETERMINISTIC {
			t.Errorf("noir step %q should be deterministic", s.GetOperation())
		}
	}
}

// TestCompileWarnsWhenInputMissing proves the compiler flags a Look that needs
// an input image when the caller has none.
func TestCompileWarnsWhenInputMissing(t *testing.T) {
	out := Compile(builtinByID(t, "polaroid-600"), "", "", false)
	if !hasWarningContaining(out.GetWarnings(), "input image") {
		t.Errorf("expected a missing-input warning, got %v", out.GetWarnings())
	}
}

// TestCompileSubjectFallback proves an empty subject falls back to "the image"
// so the resolved instruction still reads naturally.
func TestCompileSubjectFallback(t *testing.T) {
	out := Compile(builtinByID(t, "photoreal"), "", "", true)
	if !strings.Contains(out.GetPrimaryPrompt(), "the image") {
		t.Errorf("empty subject should fall back to 'the image', got %q", out.GetPrimaryPrompt())
	}
}

func hasWarningContaining(warnings []string, sub string) bool {
	for _, w := range warnings {
		if strings.Contains(w, sub) {
			return true
		}
	}
	return false
}
