package opscatalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"swarm-manager/internal/agentops"
)

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

// seedContractDoc returns one seeded contract as a schema-valid document.
func seedContractDoc(t *testing.T, id agentops.OperationID) agentops.OperationContract {
	t.Helper()
	for _, oc := range agentops.SeedOperationContracts() {
		if oc.ID == id {
			return oc
		}
	}
	t.Fatalf("no seed for %q", id)
	return agentops.OperationContract{}
}

func TestLoadSyntheticCatalogEndToEnd(t *testing.T) {
	dir := t.TempDir()
	review := seedContractDoc(t, agentops.OpReviewRound)
	writeJSON(t, filepath.Join(dir, DirOperationContracts, "review-round.json"), review)

	// A system-default binding for review-round pinned to a synthetic mode.
	writeJSON(t, filepath.Join(dir, DirBindings, "review-default.json"), agentops.OperationBinding{
		Kind: "agentops-operation-binding", Operation: agentops.OpReviewRound,
		Layer: agentops.LayerSystemDefault, Mode: "synthetic-loop", ModeRevision: "sha256:" + repeat64('a'),
	})
	// A backlog-item transition policy.
	writeJSON(t, filepath.Join(dir, DirPolicy, "backlog-default.json"), agentops.TransitionPolicy{
		Kind: "agentops-transition-policy", ID: "backlog-default", Version: "1.0.0", DomainKind: "backlog-item",
		Transitions: []agentops.PolicyTransition{
			{FromState: "running", OnOutcome: "accepted", Action: agentops.ActionCompleteItem, ToState: "terminal-complete"},
		},
	})

	c, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	lc, ok := c.Contract(agentops.OpReviewRound, "")
	if !ok || lc.Contract.ID != agentops.OpReviewRound {
		t.Fatalf("review-round not indexed")
	}
	if !agentops.IsWellFormedDigest(lc.Revision) {
		t.Fatalf("contract revision %q not a canonical digest", lc.Revision)
	}
	if b, ok := c.SystemBindingFor(agentops.OpReviewRound, "1.0.0"); !ok || b.Binding.Mode != "synthetic-loop" {
		t.Fatalf("system binding not resolved: %+v ok=%v", b, ok)
	}
	if p, ok := c.PolicyForDomain("backlog-item"); !ok || p.Policy.ID != "backlog-default" {
		t.Fatalf("policy for backlog-item not resolved")
	}
	if _, ok := c.TargetCapability(agentops.TargetBacklogItem); !ok {
		t.Fatalf("target capability registry missing backlog-item")
	}
}

func TestLoadFailsClosedOnEmptyContracts(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, DirOperationContracts), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil {
		t.Fatalf("empty catalog must fail closed")
	}
}

func TestLoadRejectsInvalidContract(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, DirOperationContracts), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, DirOperationContracts, "bad.json"), []byte(`{"kind":"agentops-operation-contract","id":"not-registered"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil {
		t.Fatalf("invalid contract must fail the whole load")
	}
}

func TestLoadRejectsVersionConflict(t *testing.T) {
	dir := t.TempDir()
	review := seedContractDoc(t, agentops.OpReviewRound)
	writeJSON(t, filepath.Join(dir, DirOperationContracts, "a.json"), review)
	writeJSON(t, filepath.Join(dir, DirOperationContracts, "b.json"), review) // same id@version
	if _, err := Load(dir); err == nil {
		t.Fatalf("duplicate id@version must be rejected")
	}
}

func TestLoadRejectsOverrideBindingInCatalog(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, DirOperationContracts, "review-round.json"), seedContractDoc(t, agentops.OpReviewRound))
	writeJSON(t, filepath.Join(dir, DirBindings, "override.json"), agentops.OperationBinding{
		Kind: "agentops-operation-binding", Operation: agentops.OpReviewRound,
		Layer: agentops.LayerInitiativeOverride, Owner: &agentops.BindingOwner{Kind: "initiative", ID: "x"},
		Mode: "m", ModeRevision: "r",
	})
	if _, err := Load(dir); err == nil {
		t.Fatalf("override binding in the shipped catalog must be rejected")
	}
}

func TestLoadRejectsBindingForUnknownOperation(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, DirOperationContracts, "review-round.json"), seedContractDoc(t, agentops.OpReviewRound))
	writeJSON(t, filepath.Join(dir, DirBindings, "orphan.json"), agentops.OperationBinding{
		Kind: "agentops-operation-binding", Operation: agentops.OpExecutionRun,
		Layer: agentops.LayerSystemDefault, Mode: "m", ModeRevision: "r",
	})
	if _, err := Load(dir); err == nil {
		t.Fatalf("binding referencing an undeclared operation must be rejected")
	}
}

func TestPolicyForDomainAmbiguityFailsClosed(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, DirOperationContracts, "review-round.json"), seedContractDoc(t, agentops.OpReviewRound))
	for _, id := range []string{"p1", "p2"} {
		writeJSON(t, filepath.Join(dir, DirPolicy, id+".json"), agentops.TransitionPolicy{
			Kind: "agentops-transition-policy", ID: id, Version: "1.0.0", DomainKind: "backlog-item",
			Transitions: []agentops.PolicyTransition{{FromState: "running", OnOutcome: "accepted", Action: agentops.ActionCompleteItem, ToState: "terminal-complete"}},
		})
	}
	c, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if _, ok := c.PolicyForDomain("backlog-item"); ok {
		t.Fatalf("two policies for one domain must resolve ambiguously (ok=false)")
	}
}

// TestShippedCatalogLoads proves the catalog the scenario ships on disk is valid
// and non-empty, so the running server's fail-closed startup load succeeds.
func TestShippedCatalogLoads(t *testing.T) {
	root := shippedCatalogRoot(t)
	if _, err := os.Stat(filepath.Join(root, DirOperationContracts)); err != nil {
		t.Skipf("shipped catalog not present at %s", root)
	}
	c, err := Load(root)
	if err != nil {
		t.Fatalf("shipped catalog failed to load: %v", err)
	}
	// Every seeded operation identity must be materialized on disk.
	for _, want := range agentops.AllOperationIDs {
		if _, ok := c.Contract(want, ""); !ok {
			t.Errorf("shipped catalog missing operation %q", want)
		}
	}
}

// TestShippedCatalogBindsEveryOperation proves the Phase-4 acceptance contract at
// the data layer: every declared operation identity has exactly one shipped
// system-default binding, so every ledger target-bound behavior resolves to a
// mode without reading Go. The binding must name a non-empty mode + revision.
func TestShippedCatalogBindsEveryOperation(t *testing.T) {
	root := shippedCatalogRoot(t)
	if _, err := os.Stat(filepath.Join(root, DirOperationContracts)); err != nil {
		t.Skipf("shipped catalog not present at %s", root)
	}
	c, err := Load(root)
	if err != nil {
		t.Fatalf("shipped catalog failed to load: %v", err)
	}
	for _, op := range agentops.AllOperationIDs {
		b, ok := c.SystemBindingFor(op, "1.0.0")
		if !ok {
			t.Errorf("operation %q has no shipped system-default binding", op)
			continue
		}
		if b.Binding.Layer != agentops.LayerSystemDefault {
			t.Errorf("operation %q binding is layer %q, want system-default", op, b.Binding.Layer)
		}
		if b.Binding.Mode == "" || b.Binding.ModeRevision == "" {
			t.Errorf("operation %q binding names empty mode/revision", op)
		}
	}
}

// shippedCatalogRoot returns scenarios/swarm-manager (five dirs up from this
// test package: opscatalog -> internal -> api -> swarm-manager).
func shippedCatalogRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// .../scenarios/swarm-manager/api/internal/opscatalog
	return filepath.Clean(filepath.Join(wd, "..", "..", ".."))
}

func repeat64(b byte) string {
	out := make([]byte, 64)
	for i := range out {
		out[i] = b
	}
	return string(out)
}
