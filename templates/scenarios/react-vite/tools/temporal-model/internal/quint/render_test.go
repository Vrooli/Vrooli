package quint_test

import (
	"strings"
	"testing"

	"react-vite-temporal-model/internal/contract"
	"react-vite-temporal-model/internal/model"
	"react-vite-temporal-model/internal/quint"
	"react-vite-temporal-model/internal/testkit"
)

func TestRenderKeepsTransitionTableOutOfVerifiedInvariants(t *testing.T) {
	c := contract.Contract{
		SchemaVersion: contract.SchemaVersion,
		FlowID:        "example.flow",
		Domain:        "example",
		Description:   "Example.",
		ContractPath:  "example.flow.json",
		Model: contract.Model{
			Module:     "Example",
			Seed:       "1",
			MaxSteps:   1,
			TraceCount: 1,
			Verify:     contract.Verify{Invariants: []string{"TypeOK"}},
		},
		Outputs: contract.Outputs{
			ModelPath:        "model.qnt",
			ArtifactPath:     "model.formal.generated.json",
			DeclarationsPath: "model.generated.go",
			ReplayTestPath:   "model_formal_replay_test.generated.go",
		},
		States: []contract.State{{ID: "idle", Quint: "Idle", Initial: true}},
		Events: []contract.Event{{ID: "tick", Quint: "Tick"}},
		TransitionDefaults: contract.TransitionDefaults{
			Invalid: &contract.DefaultTransition{To: model.SelfTarget, WantError: true},
		},
		Transitions: []contract.Transition{{From: contract.StringList{"idle"}, Event: contract.StringList{"tick"}, To: model.SelfTarget, WantError: boolPtr(true)}},
		Invariants:  []contract.Invariant{{ID: "type_ok", Quint: "TypeOK", Description: "Type OK."}},
		Traces:      []contract.Trace{{Name: "idle", Initial: "idle", Steps: nil}},
		Runtime: contract.Runtime{
			Go: &contract.GoRuntime{Package: "example", StatusType: "Status", EventType: "Event", ConstantPrefix: "Example"},
		},
		Replay: contract.Replay{
			Kind:     string(model.ReplayKindGoTest),
			TestPath: "model_formal_replay_test.generated.go",
			Transition: contract.ReplayTransition{
				Function:    "TransitionExample",
				StateType:   "State",
				StatusField: "Status",
			},
		},
	}
	flow := testkit.MustCompile(t, c)
	rendered := quint.Render(flow)
	if strings.Contains(rendered, "AllDeclaredTransitionsCovered") {
		t.Fatalf("rendered model still contains fake invariant:\n%s", rendered)
	}
	if !strings.Contains(rendered, "run transitionTable") {
		t.Fatalf("rendered model does not contain transitionTable run:\n%s", rendered)
	}
}

func boolPtr(value bool) *bool {
	return &value
}
