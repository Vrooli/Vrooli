package validation

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"connectrpc.com/connect"

	internalaudit "quality-health/internal/audit"
	"quality-health/internal/surfaces"

	"github.com/vrooli/maturity-go/assessment"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

// stubAuditor satisfies the Auditor interface for handler tests without spinning
// up CodeFacts or real filesystem discovery.
type stubAuditor struct{}

func (s *stubAuditor) Audit(_ context.Context, req internalaudit.Request) (internalaudit.Response, error) {
	return internalaudit.Response{
		RunID:  "qh-stub",
		Status: "passed",
		Inventory: surfaces.Inventory{
			Scenario:   req.Scenario,
			TargetKind: "scenario",
		},
		Maturity: internalaudit.Maturity{Rung: 3, Label: "L3"},
	}, nil
}

func TestValidateScenarioAttachesMetrics(t *testing.T) {
	h := NewConnectHandler(Deps{
		Auditor:      &stubAuditor{},
		MaturitySpec: testMaturitySpec(t),
	})
	resp, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "quality-health"}))
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	m := resp.Msg.GetMetrics()
	if m == nil {
		t.Fatal("metrics must be attached to the response")
	}
	if m.GetWallClockMs() < 0 {
		t.Fatalf("wall clock must be non-negative, got %d", m.GetWallClockMs())
	}
	env := m.GetEnvironment()
	if env == nil {
		t.Fatal("metrics environment must be populated with the stdlib baseline")
	}
	if env.GetOs() != runtime.GOOS {
		t.Fatalf("env os = %q, want %q", env.GetOs(), runtime.GOOS)
	}
	if env.GetArch() != runtime.GOARCH {
		t.Fatalf("env arch = %q, want %q", env.GetArch(), runtime.GOARCH)
	}
	if env.GetNumCpu() != int32(runtime.NumCPU()) {
		t.Fatalf("env num_cpu = %d, want %d", env.GetNumCpu(), runtime.NumCPU())
	}
}

func testMaturitySpec(t *testing.T) *assessment.Spec {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", ".vrooli", "maturity.json"))
	if err != nil {
		t.Fatalf("read maturity.json: %v", err)
	}
	spec, err := assessment.ParseSpec(raw)
	if err != nil {
		t.Fatalf("parse maturity spec: %v", err)
	}
	return spec
}
