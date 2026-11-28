package ai_test

import (
	"testing"

	"image-tools/internal/ai"
	"image-tools/internal/analysis"
	"image-tools/internal/backends"
	"image-tools/internal/capability"
	"image-tools/internal/looks"
	"image-tools/internal/models"
	"image-tools/internal/ops"

	looksv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/looks"
)

// TestEnginesAgreeWithManifests is the invariant that makes the capability
// manifest set trustworthy as the single declaration of operations. The manifest
// says which engine runs each operation; these assertions prove each engine's
// actual membership is exactly that set — in both directions, so an operation
// cannot gain a runner without a manifest, and a manifest cannot promise a runner
// that does not exist.
//
// It replaces the former vocabulary test, whose subject (internal/operations) was
// a second declaration of the same facts.
func TestEnginesAgreeWithManifests(t *testing.T) {
	// 1. compute: every engine=compute manifest has an in-process operation.
	for _, m := range capability.All() {
		if m.Engine != capability.EngineCompute {
			continue
		}
		if !ops.Has(m.ID) {
			t.Errorf("manifest %q declares engine=compute but internal/ops has no operation", m.ID)
		}
	}
	// 2. compute, reverse: every in-process operation is declared.
	for _, name := range ops.Names() {
		m, ok := capability.Get(name)
		if !ok {
			t.Errorf("internal/ops operation %q has no capability manifest", name)
			continue
		}
		if m.Engine != capability.EngineCompute {
			t.Errorf("internal/ops operation %q has manifest engine %q, want compute", name, m.Engine)
		}
	}

	// 3. ai: catalog membership is exactly engine=ai.
	aiOps := make(map[string]bool)
	for _, op := range ai.List() {
		aiOps[op.Name] = true
		m, ok := capability.Get(op.Name)
		if !ok {
			t.Errorf("AI operation %q has no capability manifest", op.Name)
			continue
		}
		if m.Engine != capability.EngineAI {
			t.Errorf("AI catalog operation %q has manifest engine %q, want ai", op.Name, m.Engine)
		}
	}
	for _, m := range capability.ByEngine(capability.EngineAI) {
		if !aiOps[m.ID] {
			t.Errorf("manifest %q declares engine=ai but the AI catalog does not list it", m.ID)
		}
	}

	// 4. analysis: catalog membership is exactly engine=analysis. Every analysis
	//    operation is declared, including the in-process probe.
	analysisOps := make(map[string]bool)
	for _, op := range analysis.List() {
		analysisOps[op.Name] = true
		m, ok := capability.Get(op.Name)
		if !ok {
			t.Errorf("analysis operation %q has no capability manifest", op.Name)
			continue
		}
		if m.Engine != capability.EngineAnalysis {
			t.Errorf("analysis catalog operation %q has manifest engine %q, want analysis", op.Name, m.Engine)
		}
	}
	for _, m := range capability.ByEngine(capability.EngineAnalysis) {
		if !analysisOps[m.ID] {
			t.Errorf("manifest %q declares engine=analysis but the analysis catalog does not list it", m.ID)
		}
	}

	// 5. unwired is honest: no engine runs it. A forward-declared operation must
	//    not quietly gain execution without its manifest changing.
	for _, m := range capability.ByEngine(capability.EngineUnwired) {
		if ai.Has(m.ID) {
			t.Errorf("unwired operation %q is executable by the AI engine", m.ID)
		}
		if _, hasRunner := aiRunner(m.ID); hasRunner {
			t.Errorf("unwired operation %q has an AI runner", m.ID)
		}
		for _, op := range analysis.List() {
			if op.Name == m.ID {
				t.Errorf("unwired operation %q is executable by the analysis service", m.ID)
			}
		}
		if ops.Has(m.ID) {
			t.Errorf("unwired operation %q is executable in process", m.ID)
		}
	}
}

// aiRunner reports whether the AI engine builds a runner for an operation.
func aiRunner(op string) (struct{}, bool) {
	eng := &ai.Engine{}
	_, ok := eng.BuildRunners()[op]
	return struct{}{}, ok
}

// TestEveryReferenceResolves proves no surface can name an operation the manifest
// set does not declare — models, providers, and built-in Looks included.
func TestEveryReferenceResolves(t *testing.T) {
	check := func(t *testing.T, source, op string) {
		t.Helper()
		if !capability.Has(op) {
			t.Errorf("%s references operation %q which no manifest declares", source, op)
		}
	}

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

	breg := backends.New()
	if err := ai.RegisterProviders(breg, nil, nil, ""); err != nil {
		t.Fatalf("RegisterProviders: %v", err)
	}
	for _, op := range breg.Operations() {
		check(t, "backend provider", op)
	}

	for _, look := range looks.BuiltinLooks() {
		for _, step := range look.GetSteps() {
			check(t, "look "+look.GetId(), step.GetOperation())
			if _, ok := looks.LaneForOperation(step.GetOperation()); !ok {
				t.Errorf("look %q step %q has no engine lane", look.GetId(), step.GetOperation())
			}
			if step.GetKind() == looksv1.StepKind_STEP_KIND_UNSPECIFIED {
				t.Errorf("look %q step %q has no declared kind", look.GetId(), step.GetOperation())
			}
		}
	}
}
