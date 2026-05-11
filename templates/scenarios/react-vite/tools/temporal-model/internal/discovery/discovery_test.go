package discovery

import (
	"testing"

	"react-vite-temporal-model/internal/contract"
	"react-vite-temporal-model/internal/model"
	"react-vite-temporal-model/internal/testkit"
)

func TestFindContractsIgnoresGeneratedAndDependencyDirs(t *testing.T) {
	root := t.TempDir()
	writeFlow(t, root, "api/example.flow.json", "example.visible.api")
	writeFlow(t, root, ".git/hidden.flow.json", "example.git.api")
	writeFlow(t, root, "node_modules/hidden.flow.json", "example.node.api")
	writeFlow(t, root, "dist/hidden.flow.json", "example.dist.api")
	writeFlow(t, root, "build/hidden.flow.json", "example.build.api")
	writeFlow(t, root, "coverage/hidden.flow.json", "example.coverage.api")
	writeFlow(t, root, "_apalache-out/hidden.flow.json", "example.apalache.api")

	contracts, err := FindContracts(root)
	if err != nil {
		t.Fatalf("FindContracts() error = %v", err)
	}
	if got, want := len(contracts), 1; got != want {
		t.Fatalf("contracts = %d, want %d", got, want)
	}
	if contracts[0].FlowID != "example.visible.api" {
		t.Fatalf("flow id = %s", contracts[0].FlowID)
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
