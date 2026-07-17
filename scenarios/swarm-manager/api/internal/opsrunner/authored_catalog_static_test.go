package opsrunner

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/operatingmode"
	"swarm-manager/internal/opscatalog"
)

// scenarioRootDir is the swarm-manager scenario root (operation-contracts/,
// bindings/, policy/, modes/) relative to this package (internal/opsrunner).
const scenarioRootDir = "../../.."

func loadShippedCatalogAndModes(t *testing.T) (*opscatalog.Catalog, map[string]operatingmode.Definition) {
	t.Helper()
	cat, err := opscatalog.Load(scenarioRootDir)
	if err != nil {
		t.Fatalf("load shipped catalog: %v", err)
	}
	defs, err := operatingmode.LoadModesFromDir(filepath.Join(scenarioRootDir, "modes"))
	if err != nil {
		t.Fatalf("load shipped modes: %v", err)
	}
	byID := make(map[string]operatingmode.Definition, len(defs))
	for mode, def := range defs {
		byID[string(mode)] = def
	}
	return cat, byID
}

// TestNoUnboundOperationContract is the static architecture gate for binding
// coverage: EVERY operation contract in the Go SSOT
// (agentops.SeedOperationContracts) must (a) be materialized in the shipped
// on-disk catalog at its exact version (genopscatalog drift check) and (b)
// have a system-default binding at that version — an operation without a
// default binding is invocable in no scope and would only fail at runtime
// with ErrNoBinding. Red-proof: TestUnboundContractDetectionFiresOnViolation
// builds a catalog with a contract and no binding and proves the same lookup
// reports the gap.
// [REQ:REQ-P0-011-OPERATION-CONTRACTS]
func TestNoUnboundOperationContract(t *testing.T) {
	cat, _ := loadShippedCatalogAndModes(t)
	for _, oc := range agentops.SeedOperationContracts() {
		if _, ok := cat.Contract(oc.ID, oc.Version); !ok {
			t.Errorf("operation contract %s@%s is in the Go SSOT but not in the shipped catalog — re-run `go run ./api/cmd/genopscatalog <scenario-root>`", oc.ID, oc.Version)
			continue
		}
		if _, ok := cat.SystemBindingFor(oc.ID, oc.Version); !ok {
			t.Errorf("operation %s@%s has NO system-default binding — every shipped contract must be bound to an implementing mode (bindings/%s.json)", oc.ID, oc.Version, oc.ID)
		}
	}
}

// TestNoIncompatibleDefaultBinding is the static architecture gate for binding
// validity: every shipped system-default binding must name a REGISTERED mode
// whose revision the production ModeChecker accepts and whose declared target
// kind both (a) provides every capability the operation's contract requires
// and (b) passes the EXACT resolver path (agentops.ResolveBinding with the
// LivePreparer as checker — the same object production wires as both
// ModePreparer and ModeChecker), so this test can never drift from what the
// runner enforces at invocation time.
//
// On target-kind interpretation: a contract declares required CAPABILITIES,
// not target kinds; a mode declares exactly ONE target kind. The binding is
// valid when the mode's kind is among the operation's compatible kinds — an
// operation compatible with several kinds (e.g. review-round with backlog-item
// and initiative) is bound per kind via distinct operations or override
// layers, and invoking it on a kind the bound mode does not serve fails
// closed at runtime with ErrIncompatibleMode (covered by the red-proof below).
// [REQ:REQ-P0-011-REVISION-PINNING]
func TestNoIncompatibleDefaultBinding(t *testing.T) {
	cat, defs := loadShippedCatalogAndModes(t)
	preparer := NewLivePreparer(cat, defs).WithDelegated(defs)
	for _, lb := range cat.SystemBindings() {
		b := lb.Binding
		lc, ok := cat.Contract(b.Operation, b.OperationVersion)
		if !ok {
			t.Errorf("binding %s names operation %s@%s the catalog does not declare", lb.Source, b.Operation, b.OperationVersion)
			continue
		}
		def, ok := defs[b.Mode]
		if !ok {
			t.Errorf("binding %s names mode %q which is not a registered mode", lb.Source, b.Mode)
			continue
		}
		kind := agentops.TargetKind(def.Target.Kind)
		if err := agentops.CheckOperationTargetCompatibility(b.Operation, lc.Contract.TargetRequirements.Capabilities, kind); err != nil {
			t.Errorf("binding %s: bound mode %q targets %q, which cannot provide the operation's required capabilities: %v", lb.Source, b.Mode, kind, err)
			continue
		}
		// The resolver's own path: RevisionExists + ModeCompatible via the
		// production checker. A failure here is exactly the failure a live
		// invocation would hit.
		if _, err := agentops.ResolveBinding(b.Operation, []agentops.OperationBinding{b}, agentops.ResolutionScope{Target: kind}, preparer); err != nil {
			t.Errorf("binding %s does not resolve through the production checker for target %q: %v", lb.Source, kind, err)
		}
	}
}

// TestUnboundContractDetectionFiresOnViolation red-proofs
// TestNoUnboundOperationContract: a synthetic catalog holding one contract and
// an EMPTY bindings/ directory must report no system-default binding for it.
func TestUnboundContractDetectionFiresOnViolation(t *testing.T) {
	root := t.TempDir()
	contractsDir := filepath.Join(root, opscatalog.DirOperationContracts)
	if err := os.MkdirAll(contractsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	oc := agentops.SeedOperationContracts()[0]
	raw, err := json.MarshalIndent(oc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contractsDir, string(oc.ID)+".json"), append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	cat, err := opscatalog.Load(root)
	if err != nil {
		t.Fatalf("load synthetic catalog: %v", err)
	}
	if _, ok := cat.SystemBindingFor(oc.ID, oc.Version); ok {
		t.Fatalf("synthetic catalog has no bindings, yet SystemBindingFor(%s@%s) reported one — the unbound-contract gate would never fire", oc.ID, oc.Version)
	}
}

// TestIncompatibleBindingDetectionFiresOnViolation red-proofs
// TestNoIncompatibleDefaultBinding through the resolver itself: a binding to a
// registered mode whose target kind cannot serve the invoked target must fail
// with ErrIncompatibleMode, and a binding to an unregistered mode must fail
// with ErrDeletedRevision. Uses the same production LivePreparer checker.
func TestIncompatibleBindingDetectionFiresOnViolation(t *testing.T) {
	cat, defs := loadShippedCatalogAndModes(t)
	preparer := NewLivePreparer(cat, defs).WithDelegated(defs)
	sys, ok := cat.SystemBindingFor(agentops.OpExecutionRun, "1.0.0")
	if !ok {
		t.Fatal("shipped catalog must bind execution-run")
	}
	// The bound mode targets plan-execution; invoking against a scenario target
	// must fail closed as incompatible.
	if _, err := agentops.ResolveBinding(sys.Binding.Operation, []agentops.OperationBinding{sys.Binding}, agentops.ResolutionScope{Target: agentops.TargetScenario}, preparer); !errors.Is(err, agentops.ErrIncompatibleMode) {
		t.Fatalf("resolver accepted a binding whose mode cannot serve the target (err=%v) — the incompatible-binding gate would never fire", err)
	}
	ghost := sys.Binding
	ghost.Mode = "no-such-mode"
	if _, err := agentops.ResolveBinding(ghost.Operation, []agentops.OperationBinding{ghost}, agentops.ResolutionScope{Target: agentops.TargetPlanExecution}, preparer); !errors.Is(err, agentops.ErrDeletedRevision) {
		t.Fatalf("resolver accepted a binding to an unregistered mode (err=%v)", err)
	}
}
