package discovery

import (
	"testing"

	"react-vite-temporal-model/internal/contract"
	"react-vite-temporal-model/internal/model"
	"react-vite-temporal-model/internal/testkit"
)

func TestFindContractsIgnoresGeneratedAndDependencyDirs(t *testing.T) {
	root := t.TempDir()
	writeFlow(t, root, "api/example.flow.json", "example.visible")
	writeFlow(t, root, ".git/hidden.flow.json", "example.git")
	writeFlow(t, root, "node_modules/hidden.flow.json", "example.node")
	writeFlow(t, root, "dist/hidden.flow.json", "example.dist")
	writeFlow(t, root, "build/hidden.flow.json", "example.build")
	writeFlow(t, root, "coverage/hidden.flow.json", "example.coverage")
	writeFlow(t, root, "_apalache-out/hidden.flow.json", "example.apalache")

	contracts, err := FindContracts(root)
	if err != nil {
		t.Fatalf("FindContracts() error = %v", err)
	}
	if got, want := len(contracts), 1; got != want {
		t.Fatalf("contracts = %d, want %d", got, want)
	}
	if contracts[0].FlowID != "example.visible" {
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
	raw.Outputs.DeclarationsPath = "api/example.generated.go"
	raw.Outputs.ReplayTestPath = "api/example_formal_replay_test.generated.go"
	raw.Replay.TestPath = raw.Outputs.ReplayTestPath
	testkit.WriteFlowJSON(t, root, rel, raw)
}
