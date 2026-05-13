package operatingmode

import (
	"context"
	"strings"
	"testing"
)

// TestPhaseKindEveryRegisteredPhaseHasKind enforces the P1 contract: every
// initiative-scoped phase in the production registry must declare a
// non-empty Kind. Item-level mode is intentionally skipped — it has no
// phase graph.
func TestPhaseKindEveryRegisteredPhaseHasKind(t *testing.T) {
	for _, mode := range Modes() {
		def, err := DefinitionFor(mode)
		if err != nil {
			t.Fatalf("DefinitionFor(%q): %v", mode, err)
		}
		if mode == ModeItemLevel {
			continue
		}
		for phaseName, phase := range def.PhaseGraph.Phases {
			if !IsValidPhaseKind(phase.Kind) {
				t.Errorf("mode %q phase %q kind = %q, want one of investigate|execute|review|reconcile",
					mode, phaseName, phase.Kind)
			}
		}
	}
}

// TestPhaseKindHolisticLoopBackfillKinds pins the kind assignment on every
// holistic-loop phase so the catalog wire surface remains stable for UI
// renderers that key off PhaseKind for column placement.
func TestPhaseKindHolisticLoopBackfillKinds(t *testing.T) {
	def, err := DefinitionFor(ModeHolisticLoop)
	if err != nil {
		t.Fatalf("DefinitionFor: %v", err)
	}
	want := map[Phase]PhaseKind{
		"investigate": PhaseKindInvestigate,
		"plan":        PhaseKindInvestigate,
		"execute":     PhaseKindExecute,
		"review":      PhaseKindReview,
	}
	for phase, expected := range want {
		got := def.PhaseGraph.Phases[phase].Kind
		if got != expected {
			t.Errorf("holistic-loop %q kind = %q, want %q", phase, got, expected)
		}
	}
}

// TestPhaseKindPhasedPlanDrainBackfillKinds pins kind assignment on every
// phased-plan-drain phase. Pinning the assignments here protects the
// Operations Center column placement contract from silent drift when
// phase definitions are refactored.
func TestPhaseKindPhasedPlanDrainBackfillKinds(t *testing.T) {
	def, err := DefinitionFor(ModePhasedPlanDrain)
	if err != nil {
		t.Fatalf("DefinitionFor: %v", err)
	}
	want := map[Phase]PhaseKind{
		"prepare_plan":      PhaseKindInvestigate,
		"execute_next":      PhaseKindExecute,
		"classify_progress": PhaseKindReview,
		"review":            PhaseKindReview,
	}
	for phase, expected := range want {
		got := def.PhaseGraph.Phases[phase].Kind
		if got != expected {
			t.Errorf("phased-plan-drain %q kind = %q, want %q", phase, got, expected)
		}
	}
}

// TestPhaseKindValidatorRejectsEmptyKind exercises the registry validator's
// guarantee that an initiative-scoped phase cannot ship without a kind.
// Reaching the validator with an empty Kind is the canonical failure
// mode for new phase definitions; the assertion message pins the wording
// so phase authors get an actionable error.
func TestPhaseKindValidatorRejectsEmptyKind(t *testing.T) {
	defs := cloneRegistryForTest()
	def := defs[ModeHolisticLoop]
	phase := def.PhaseGraph.Phases["investigate"]
	phase.Kind = ""
	def.PhaseGraph.Phases["investigate"] = phase
	defs[ModeHolisticLoop] = def

	err := validateDefinitions(defs)
	if err == nil {
		t.Fatal("validateDefinitions accepted empty Kind, want rejection")
	}
	if !strings.Contains(err.Error(), "kind must be one of investigate|execute|review|reconcile") {
		t.Fatalf("validateDefinitions error = %v, want kind-validation message", err)
	}
}

// TestPhaseKindValidatorRejectsUnknownKind ensures the validator catches
// typos like `PhaseKind("investigation")`. IsValidPhaseKind is the gate;
// the validator surfaces the failure with the expected message.
func TestPhaseKindValidatorRejectsUnknownKind(t *testing.T) {
	defs := cloneRegistryForTest()
	def := defs[ModeHolisticLoop]
	phase := def.PhaseGraph.Phases["investigate"]
	phase.Kind = PhaseKind("investigation") // typo: trailing 'ion'
	def.PhaseGraph.Phases["investigate"] = phase
	defs[ModeHolisticLoop] = def

	err := validateDefinitions(defs)
	if err == nil {
		t.Fatal("validateDefinitions accepted unknown Kind value")
	}
	if !strings.Contains(err.Error(), "kind must be one of") {
		t.Fatalf("validateDefinitions error = %v, want kind-validation message", err)
	}
}

func TestAutoStartAfterAcceptsValidSinglePredecessor(t *testing.T) {
	defs := cloneRegistryForTest()
	def := defs[ModeHolisticLoop]
	// "review" already exists; mark it as auto-starting after "execute".
	phase := def.PhaseGraph.Phases["review"]
	phase.AutoStartAfter = []Phase{"execute"}
	def.PhaseGraph.Phases["review"] = phase
	defs[ModeHolisticLoop] = def

	if err := validateDefinitions(defs); err != nil {
		t.Fatalf("validateDefinitions rejected valid single-predecessor auto-start: %v", err)
	}
}

func TestAutoStartAfterRejectsLengthGreaterThanOne(t *testing.T) {
	defs := cloneRegistryForTest()
	def := defs[ModeHolisticLoop]
	phase := def.PhaseGraph.Phases["review"]
	phase.AutoStartAfter = []Phase{"execute", "plan"}
	def.PhaseGraph.Phases["review"] = phase
	defs[ModeHolisticLoop] = def

	err := validateDefinitions(defs)
	if err == nil {
		t.Fatal("validateDefinitions accepted multi-predecessor auto-start")
	}
	if !strings.Contains(err.Error(), "supports at most one predecessor") {
		t.Fatalf("validateDefinitions error = %v, want length-validation message", err)
	}
}

func TestAutoStartAfterRejectsUnknownTarget(t *testing.T) {
	defs := cloneRegistryForTest()
	def := defs[ModeHolisticLoop]
	phase := def.PhaseGraph.Phases["review"]
	phase.AutoStartAfter = []Phase{"does-not-exist"}
	def.PhaseGraph.Phases["review"] = phase
	defs[ModeHolisticLoop] = def

	err := validateDefinitions(defs)
	if err == nil {
		t.Fatal("validateDefinitions accepted auto-start referencing unknown phase")
	}
	if !strings.Contains(err.Error(), "auto_start_after references unregistered phase") {
		t.Fatalf("validateDefinitions error = %v, want unregistered-phase message", err)
	}
}

func TestAutoStartAfterRejectsSelfReference(t *testing.T) {
	defs := cloneRegistryForTest()
	def := defs[ModeHolisticLoop]
	phase := def.PhaseGraph.Phases["review"]
	phase.AutoStartAfter = []Phase{"review"}
	def.PhaseGraph.Phases["review"] = phase
	defs[ModeHolisticLoop] = def

	err := validateDefinitions(defs)
	if err == nil {
		t.Fatal("validateDefinitions accepted self-referential auto-start")
	}
	if !strings.Contains(err.Error(), "cannot reference itself") {
		t.Fatalf("validateDefinitions error = %v, want self-reference message", err)
	}
}

// TestPhaseKindFlowsToCatalog verifies that the `phase_kind` field reaches
// the catalog wire shape. The Operations Center page consumes this field
// directly to lay out lane columns, so a regression here would silently
// break UI rendering even though tsc and the registry validator both pass.
func TestPhaseKindFlowsToCatalog(t *testing.T) {
	svc := newTestService(t, t.TempDir(), &fakeAgent{}, &fakePrompts{})
	catalog, err := svc.Catalog()
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}
	for _, entry := range catalog.Modes {
		if !entry.SupportsPhases {
			continue
		}
		for _, phase := range entry.Phases {
			if phase.PhaseKind == "" {
				t.Errorf("mode %q phase %q catalog payload phase_kind is empty",
					entry.Mode, phase.Phase)
			}
		}
	}
}

// TestPhaseKindFlowsToWorkspace verifies the workspace endpoint also
// surfaces `phase_kind`. The workspace and catalog channels are populated
// independently from the same underlying PhaseDefinition, so both need
// explicit coverage to catch field-omission regressions on either side.
func TestPhaseKindFlowsToWorkspace(t *testing.T) {
	svc := newTestService(t, t.TempDir(), &fakeAgent{}, &fakePrompts{})
	for _, mode := range []Mode{ModeHolisticLoop, ModePhasedPlanDrain} {
		def := MustDefinition(mode)
		// Use buildCatalogEntry-equivalent assertion via Catalog rather
		// than constructing a Workspace (which needs an initiative
		// fixture); the WorkspacePhase wire shape is exercised in the
		// service-level workspace tests.
		entry := mustEntry(t, svc, mode)
		if len(entry.Phases) != len(def.PhaseGraph.Phases) {
			t.Fatalf("catalog phase count = %d, registry = %d for mode %q",
				len(entry.Phases), len(def.PhaseGraph.Phases), mode)
		}
		for _, phase := range entry.Phases {
			registryKind := def.PhaseGraph.Phases[Phase(phase.Phase)].Kind
			if phase.PhaseKind != string(registryKind) {
				t.Errorf("mode %q phase %q catalog kind = %q, registry = %q",
					mode, phase.Phase, phase.PhaseKind, registryKind)
			}
		}
	}
}

// TestPhaseKindFlowsToWorkspaceEndpoint verifies the workspace endpoint
// surfaces `phase_kind` directly on each WorkspacePhase. Catalog and
// workspace channels are populated independently from the same registry
// definition, so this test is the canonical guard for the workspace path.
func TestPhaseKindFlowsToWorkspaceEndpoint(t *testing.T) {
	root := t.TempDir()
	svc := newTestService(t, root, &fakeAgent{}, &fakePrompts{})

	workspace, err := svc.Workspace(context.Background(), "init-a")
	if err != nil {
		t.Fatalf("Workspace returned error: %v", err)
	}
	def := MustDefinition(ModeHolisticLoop)
	for _, phase := range workspace.Definition.Phases {
		registryKind := def.PhaseGraph.Phases[Phase(phase.Phase)].Kind
		if phase.PhaseKind != string(registryKind) {
			t.Errorf("workspace phase %q kind = %q, want %q",
				phase.Phase, phase.PhaseKind, registryKind)
		}
	}
}

func mustEntry(t *testing.T, svc *Service, mode Mode) ModeCatalogEntry {
	t.Helper()
	catalog, err := svc.Catalog()
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}
	for _, entry := range catalog.Modes {
		if entry.Mode == string(mode) {
			return entry
		}
	}
	t.Fatalf("mode %q not in catalog", mode)
	return ModeCatalogEntry{}
}
