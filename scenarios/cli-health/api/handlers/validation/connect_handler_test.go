package validation

import (
	"context"
	"errors"
	"log"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"cli-health/internal/services/manifestvalidation"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli-health/v1/validation"
)

type stubValidator struct {
	called bool
}

func (s *stubValidator) ValidateScenario(_ context.Context, scenario string) (manifestvalidation.Report, error) {
	s.called = true
	if scenario == "" {
		return manifestvalidation.Report{}, errors.New("scenario is required")
	}
	return manifestvalidation.Report{Scenario: scenario, Passed: true}, nil
}

func TestValidate_ReservedNameRejected(t *testing.T) {
	v := &stubValidator{}
	h := NewConnectHandler(Deps{
		Logger:        log.New(log.Writer(), "", 0),
		Validator:     v,
		ReservedNames: []string{"vrooli"},
	})
	req := connect.NewRequest(&validationv1.ValidateScenarioRequest{Scenario: "vrooli"})
	_, err := h.ValidateScenario(context.Background(), req)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	var ce *connect.Error
	if !errors.As(err, &ce) {
		t.Fatalf("want connect.Error, got %T: %v", err, err)
	}
	if ce.Code() != connect.CodeInvalidArgument {
		t.Errorf("code = %v, want InvalidArgument", ce.Code())
	}
	if !strings.Contains(ce.Message(), "not a scenario") {
		t.Errorf("message = %q, want substring %q", ce.Message(), "not a scenario")
	}
	if v.called {
		t.Error("validator should not be called for reserved names")
	}
}

func TestValidate_ScenarioPassesThrough(t *testing.T) {
	v := &stubValidator{}
	h := NewConnectHandler(Deps{
		Logger:        log.New(log.Writer(), "", 0),
		Validator:     v,
		ReservedNames: []string{"vrooli"},
	})
	req := connect.NewRequest(&validationv1.ValidateScenarioRequest{Scenario: "cli-health"})
	resp, err := h.ValidateScenario(context.Background(), req)
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if !v.called {
		t.Error("validator was not called")
	}
	if !resp.Msg.GetPassed() {
		t.Errorf("Passed=false; want true")
	}
}
