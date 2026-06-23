package validation

import (
	"context"
	"log"
	"testing"

	"connectrpc.com/connect"

	"ui-health/internal/codefacts"
	"ui-health/internal/services/manifestvalidation"
	"ui-health/internal/uiruntime"

	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

// spyDescriber records whether Code Facts was consulted and returns a canned answer.
type spyDescriber struct {
	calls int
	facts codefacts.Facts
}

func (s *spyDescriber) Describe(_ context.Context, _, _ string) codefacts.Facts {
	s.calls++
	return s.facts
}

// spyRuntime records whether the runtime/render group ran.
type spyRuntime struct {
	calls    int
	findings []manifestvalidation.Finding
}

func (s *spyRuntime) Check(_ context.Context, _ uiruntime.Input) []manifestvalidation.Finding {
	s.calls++
	return s.findings
}

func newGatingHandler(facts codefacts.Facts, rt uiruntime.Checker) (*connectHandler, *spyDescriber) {
	desc := &spyDescriber{facts: facts}
	h := NewConnectHandler(Deps{
		Logger:       log.New(log.Writer(), "", 0),
		Validator:    &stubValidator{},
		MaturitySpec: testMaturitySpec(),
		CodeFacts:    desc,
		Runtime:      rt,
	})
	return h, desc
}

// TestStaticOnlySkipsCodeFactsAndRuntime is the Phase 4 DoD: a static-only
// validation (include_execution=false) makes zero Code Facts calls and never
// runs the runtime/render group.
func TestStaticOnlySkipsCodeFactsAndRuntime(t *testing.T) {
	rt := &spyRuntime{}
	h, desc := newGatingHandler(codefacts.Facts{HasUI: true, Framework: "react-vite"}, rt)

	_, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario:         "demo",
		IncludeExecution: false,
	}))
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if desc.calls != 0 {
		t.Fatalf("static-only must not consult Code Facts, got %d call(s)", desc.calls)
	}
	if rt.calls != 0 {
		t.Fatalf("static-only must not run the runtime group, got %d call(s)", rt.calls)
	}
}

// TestExecutionRunsRuntimeWhenUIPresent: include_execution=true on a UI scenario
// consults Code Facts once and runs the runtime group once, folding its findings
// into the single report.
func TestExecutionRunsRuntimeWhenUIPresent(t *testing.T) {
	rt := &spyRuntime{findings: []manifestvalidation.Finding{{
		Severity: manifestvalidation.SeverityInfo,
		Code:     "runtime_render_ok",
		Message:  "render passed",
	}}}
	h, desc := newGatingHandler(codefacts.Facts{HasUI: true, Framework: "react-vite"}, rt)

	resp, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario:         "demo",
		IncludeExecution: true,
	}))
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if desc.calls != 1 {
		t.Fatalf("execution path must consult Code Facts once, got %d", desc.calls)
	}
	if rt.calls != 1 {
		t.Fatalf("execution path must run the runtime group once, got %d", rt.calls)
	}
	var found bool
	for _, f := range resp.Msg.GetAssessment().GetFindings() {
		if f.GetCode() == "runtime_render_ok" {
			found = true
		}
	}
	if !found {
		t.Fatal("runtime finding must be folded into the single report")
	}
}

// TestExecutionSkipsRuntimeWhenNoUI: include_execution=true on a non-UI scenario
// consults Code Facts (to learn there is no UI) but does not run the runtime
// group — there is nothing to render.
func TestExecutionSkipsRuntimeWhenNoUI(t *testing.T) {
	rt := &spyRuntime{}
	h, desc := newGatingHandler(codefacts.Facts{HasUI: false}, rt)

	_, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario:         "demo",
		IncludeExecution: true,
	}))
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if desc.calls != 1 {
		t.Fatalf("execution path must consult Code Facts once, got %d", desc.calls)
	}
	if rt.calls != 0 {
		t.Fatalf("non-UI scenario must not run the runtime group, got %d call(s)", rt.calls)
	}
}

// TestExecutionWithoutRuntimeCheckerIsStatic: when no RuntimeChecker is wired
// (Phase 4 default), an execution request stays static — Code Facts is not even
// consulted, since there is nothing to gate.
func TestExecutionWithoutRuntimeCheckerIsStatic(t *testing.T) {
	h, desc := newGatingHandler(codefacts.Facts{HasUI: true}, nil)

	_, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario:         "demo",
		IncludeExecution: true,
	}))
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if desc.calls != 0 {
		t.Fatalf("with no runtime checker wired, Code Facts must not be consulted, got %d", desc.calls)
	}
}
