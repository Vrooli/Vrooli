package guidance

import (
	"context"
	"errors"
	"io"
	"log"
	"testing"

	"connectrpc.com/connect"
	guidancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/guidance"
	guidancesvc "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/guidance"
)

// fakeRunner drives the guidance service without the template engine.
type fakeRunner struct {
	result      guidancesvc.NextGateResult
	err         error
	gotScenario string
}

func (f *fakeRunner) NextGate(_ context.Context, scenario string) (guidancesvc.NextGateResult, error) {
	f.gotScenario = scenario
	return f.result, f.err
}

func quietLogger() *log.Logger { return log.New(io.Discard, "", 0) }

func newHandler(runner guidancesvc.Runner) *connectHandler {
	return NewConnectHandler(Deps{Service: guidancesvc.NewService(runner), Logger: quietLogger()})
}

func TestNextGateMapsGateAndChecks(t *testing.T) {
	runner := &fakeRunner{result: guidancesvc.NextGateResult{
		Scenario:         "acme",
		Completed:        1,
		Required:         3,
		FinalizeRequired: false,
		Gate: &guidancesvc.Gate{
			ID:          "example-domain-removed",
			Title:       "Remove the example domain",
			Description: "Detemplate the scenario",
			Required:    true,
			Docs:        []string{"docs/ORIENTATION.md"},
			Remediation: []string{"run detemplate"},
			Checks: []guidancesvc.Check{{
				Kind:    "text_absent_tree",
				Label:   "notes",
				Passed:  false,
				Message: "found residue",
			}},
		},
	}}
	handler := newHandler(runner)

	resp, err := handler.NextGate(context.Background(), connect.NewRequest(&guidancev1.NextGateRequest{Scenario: "acme"}))
	if err != nil {
		t.Fatalf("NextGate: %v", err)
	}
	if runner.gotScenario != "acme" {
		t.Fatalf("scenario not forwarded: %q", runner.gotScenario)
	}
	msg := resp.Msg
	if msg.Scenario != "acme" || msg.Completed != 1 || msg.Required != 3 {
		t.Fatalf("progress mapped wrong: %#v", msg)
	}
	if msg.Gate == nil || msg.Gate.Id != "example-domain-removed" || !msg.Gate.Required {
		t.Fatalf("gate mapped wrong: %#v", msg.Gate)
	}
	if len(msg.Gate.Checks) != 1 || msg.Gate.Checks[0].Kind != "text_absent_tree" || msg.Gate.Checks[0].Passed {
		t.Fatalf("checks mapped wrong: %#v", msg.Gate.Checks)
	}
}

func TestNextGateCompleteHasNoGate(t *testing.T) {
	runner := &fakeRunner{result: guidancesvc.NextGateResult{
		Scenario:  "acme",
		Complete:  true,
		Finalized: true,
		Completed: 3,
		Required:  3,
		Message:   "all gates complete",
	}}
	handler := newHandler(runner)

	resp, err := handler.NextGate(context.Background(), connect.NewRequest(&guidancev1.NextGateRequest{Scenario: "acme"}))
	if err != nil {
		t.Fatalf("NextGate: %v", err)
	}
	if !resp.Msg.Complete || !resp.Msg.Finalized {
		t.Fatalf("expected complete+finalized: %#v", resp.Msg)
	}
	if resp.Msg.Gate != nil {
		t.Fatalf("expected no gate when complete, got %#v", resp.Msg.Gate)
	}
}

func TestNextGateRunnerErrorMapsToInternal(t *testing.T) {
	handler := newHandler(&fakeRunner{err: errors.New("orient failed")})

	_, err := handler.NextGate(context.Background(), connect.NewRequest(&guidancev1.NextGateRequest{Scenario: "acme"}))
	if err == nil {
		t.Fatal("expected error")
	}
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("code = %v, want Internal", connect.CodeOf(err))
	}
}

func TestNextGateEmptyScenarioIsRejected(t *testing.T) {
	runner := &fakeRunner{}
	handler := newHandler(runner)

	_, err := handler.NextGate(context.Background(), connect.NewRequest(&guidancev1.NextGateRequest{Scenario: "   "}))
	if err == nil {
		t.Fatal("expected error for empty scenario")
	}
	if runner.gotScenario != "" {
		t.Fatalf("runner should not be invoked for empty scenario, got %q", runner.gotScenario)
	}
}
