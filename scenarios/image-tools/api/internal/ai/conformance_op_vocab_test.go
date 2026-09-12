package ai_test

import (
	"testing"

	"image-tools/internal/ai"
	"image-tools/internal/analysis"
	"image-tools/internal/backends"
	"image-tools/internal/looks"
	"image-tools/internal/models"
	"image-tools/internal/operations"
	"image-tools/internal/ops"

	looksv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/looks"
)

// TestOpVocabularyConformance is the W1 enforcement (Phase 1): every operation
// referenced anywhere in the codebase — by a registry model, a backend provider,
// a built-in Look step, or the analysis catalog — must resolve to the single
// operation vocabulary SSOT (internal/operations). A reference to an op absent
// from the SSOT fails the build, making the three-hand-synced-copies drift this
// refactor removed structurally impossible to reintroduce.
func TestOpVocabularyConformance(t *testing.T) {
	check := func(t *testing.T, source, op string) {
		t.Helper()
		if !operations.Has(op) {
			t.Errorf("%s references operation %q which is not in the vocabulary SSOT", source, op)
		}
	}

	// 1. Registry seed: every model op + default_for op.
	reg, err := models.Load()
	if err != nil {
		t.Fatalf("models.Load: %v", err)
	}
	for _, m := range reg.Models() {
		for _, op := range m.Operations {
			check(t, "model "+m.ID, op)
		}
		for _, op := range m.DefaultFor {
			check(t, "model "+m.ID+" default_for", op)
		}
	}

	// 2. AI discovery catalog: every op the engine builds a runner for, and each
	//    must be a generation/enhancement op (the categories ai owns).
	for _, op := range ai.List() {
		check(t, "ai catalog", op.Name)
		switch op.Category {
		case operations.CategoryGeneration, operations.CategoryEnhancement:
		default:
			t.Errorf("ai catalog op %q has category %q; ai owns only generation/enhancement", op.Name, op.Category)
		}
	}

	// 3. Backend providers: every op any registered standalone/builtin provider
	//    can execute.
	breg := backends.New()
	if err := ai.RegisterProviders(breg, nil, nil, ""); err != nil {
		t.Fatalf("RegisterProviders: %v", err)
	}
	for _, op := range breg.Operations() {
		check(t, "backend provider", op)
	}

	// 4. Built-in Looks: every step's operation, resolved against the vocabulary
	//    its kind belongs to (AI steps → the AI/model vocabulary SSOT; deterministic
	//    steps → the deterministic ops vocabulary) — mirroring looks.Validate.
	for _, look := range looks.BuiltinLooks() {
		for _, step := range look.GetSteps() {
			op := step.GetOperation()
			switch step.GetKind() {
			case looksv1.StepKind_STEP_KIND_AI:
				check(t, "look "+look.GetId()+" AI step", op)
			case looksv1.StepKind_STEP_KIND_DETERMINISTIC:
				if !ops.Has(op) {
					t.Errorf("look %s deterministic step references %q which is not a known deterministic op", look.GetId(), op)
				}
			default:
				t.Errorf("look %s step %q has an unspecified kind", look.GetId(), op)
			}
		}
	}

	// 5. Analysis catalog: every model-backed analysis op (probe is a pure-Go
	//    introspection op outside the model vocabulary and is exempt).
	for _, info := range analysis.List() {
		if info.Name == analysis.OpProbe {
			continue
		}
		check(t, "analysis catalog", info.Name)
	}
}
