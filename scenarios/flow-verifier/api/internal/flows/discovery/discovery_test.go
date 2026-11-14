package discovery

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	_ "flow-verifier/internal/flows/kinds/temporal" // register the temporal kind
	"flow-verifier/internal/flows/kinds/temporal/contract"
	"flow-verifier/internal/flows/kinds/temporal/model"
	"flow-verifier/internal/testkit"
)

func TestFindContractsIgnoresGeneratedAndDependencyDirs(t *testing.T) {
	root := t.TempDir()
	writeFlow(t, root, "api/visible/flow/flow.json", "example.visible.api")
	writeFlow(t, root, ".git/flow/flow.json", "example.git.api")
	writeFlow(t, root, "node_modules/flow/flow.json", "example.node.api")
	writeFlow(t, root, "dist/flow/flow.json", "example.dist.api")
	writeFlow(t, root, "build/flow/flow.json", "example.build.api")
	writeFlow(t, root, "coverage/flow/flow.json", "example.coverage.api")
	writeFlow(t, root, "_apalache-out/flow/flow.json", "example.apalache.api")

	contracts, err := FindContracts(root)
	if err != nil {
		t.Fatalf("FindContracts() error = %v", err)
	}
	if got, want := len(contracts), 1; got != want {
		t.Fatalf("contracts = %d, want %d", got, want)
	}
	if contracts[0].FlowID() != "example.visible.api" {
		t.Fatalf("flow id = %s", contracts[0].FlowID())
	}
}

func TestFindContractUsesNotFoundSentinel(t *testing.T) {
	_, err := FindContract(t.TempDir(), "missing.flow")
	if !errors.Is(err, ErrFlowNotFound) {
		t.Fatalf("FindContract() error = %v, want ErrFlowNotFound", err)
	}
}

func TestFindContractIgnoresInvalidUnrelatedContracts(t *testing.T) {
	root := t.TempDir()
	writeFlow(t, root, "api/visible/flow/flow.json", "example.visible.api")
	badPath := filepath.Join(root, "api/unrelated/flow/flow.json")
	if err := os.MkdirAll(filepath.Dir(badPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(badPath, []byte(`{"kind":"temporal","flowId":"example.unrelated.api"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	spec, err := FindContract(root, "example.visible.api")
	if err != nil {
		t.Fatalf("FindContract() error = %v", err)
	}
	if got, want := spec.FlowID(), "example.visible.api"; got != want {
		t.Fatalf("flow id = %s, want %s", got, want)
	}
	if _, err := FindContracts(root); err == nil {
		t.Fatal("global inventory should still report invalid unrelated contracts")
	}
}

func writeFlow(t *testing.T, root string, rel string, flowID string) {
	t.Helper()
	raw := testkit.ValidRawContract()
	raw.FlowID = flowID
	raw.States = raw.States[:1]
	raw.Events = raw.Events[:1]
	raw.TransitionDefaults.Terminal = nil
	raw.Transitions = raw.Transitions[:1]
	raw.Transitions[0].To = model.SelfTarget
	wantError := true
	raw.Transitions[0].WantError = &wantError
	raw.Traces = []contract.Trace{{Name: "idle", Initial: "idle", Steps: []contract.TraceStep{}}}
	testkit.WriteFlowJSON(t, root, rel, raw)
}
