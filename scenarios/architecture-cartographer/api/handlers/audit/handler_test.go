package audit

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"architecture-cartographer/internal/audit"

	"connectrpc.com/connect"
	"github.com/vrooli/maturity-go/assessment"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

// stubAuditService returns a clean, zero-finding audit report.
type stubAuditService struct{}

func (s *stubAuditService) Run(_ context.Context, in audit.RunInput) (audit.Report, error) {
	return audit.Report{
		Scenario: in.Scenario,
		Outcome:  audit.OutcomeClean,
	}, nil
}

func (s *stubAuditService) RunAll(_ context.Context, _ audit.RunAllInput) (audit.SweepReport, error) {
	return audit.SweepReport{}, nil
}

func testAuditMaturitySpec() *assessment.Spec {
	spec := &assessment.Spec{
		Provider: "architecture-cartographer",
		Phase:    "architecture",
		Version:  "test",
		Levels: []assessment.Level{
			{ID: "L0", Name: "No map"},
			{ID: "L1", Name: "Domains mapped"},
			{ID: "L2", Name: "No drift"},
		},
		Findings: map[string]assessment.FindingMapping{},
		Fallback: assessment.FallbackPolicy{
			LocalLevelImpact: "L2",
			GlobalImpact:     assessment.ImpactEvolvabilityGap,
			Dimension:        "structure",
			SeverityDefault:  "SEVERITY_ERROR",
		},
	}
	if err := assessment.ValidateSpec(*spec); err != nil {
		panic(err)
	}
	return spec
}

func TestValidateScenarioAttachesMetrics(t *testing.T) {
	h := NewHandler(HandlerDeps{
		Svc:          &stubAuditService{},
		MaturitySpec: testAuditMaturitySpec(),
	})
	resp, err := h.ValidateScenario(
		context.Background(),
		connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "architecture-cartographer"}),
	)
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

func TestValidateScenarioNativeDetailPreserved(t *testing.T) {
	h := NewHandler(HandlerDeps{
		Svc:          &stubAuditService{},
		MaturitySpec: testAuditMaturitySpec(),
	})
	resp, err := h.ValidateScenario(
		context.Background(),
		connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "architecture-cartographer"}),
	)
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	// native_detail must be packed so test-genie's gate can inspect it.
	if resp.Msg.GetNativeDetail() == nil {
		t.Fatal("native_detail must be populated in the response")
	}
}

func TestMaturitySpecCoversArchitectureFindings(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", ".vrooli", "maturity.json"))
	if err != nil {
		t.Fatal(err)
	}
	spec, err := assessment.ParseSpec(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := assessment.ValidateSpec(*spec); err != nil {
		t.Fatalf("ValidateSpec: %v", err)
	}
	if len(spec.Capabilities) != 5 {
		t.Fatalf("capabilities = %d, want 5", len(spec.Capabilities))
	}
	for _, code := range emittedArchitectureFindingCodes() {
		mapping, ok := spec.Findings[code]
		if !ok {
			t.Fatalf("maturity spec does not map emitted finding code %q", code)
		}
		if mapping.CapabilityID == "" {
			t.Fatalf("maturity spec finding %q must declare capability_id", code)
		}
		if mapping.CleanRequirement == "" {
			t.Fatalf("maturity spec finding %q must declare clean_requirement", code)
		}
	}
	if spec.Fallback.CapabilityID == "" {
		t.Fatal("maturity spec fallback must declare capability_id")
	}
}

func emittedArchitectureFindingCodes() []string {
	return []string{
		"graph.extract_failed",
		"domain_authority/missing",
		"domain_authority/low",
		"domains_doc_parse_warning",
		"cycle",
		"cycle/cross-domain",
		"layering",
		"cross_scenario",
		"coupling_smell",
		"mislocated_file",
		"naming",
		"glossary_drift",
		"convergence_drift",
		"surface_coherence",
		"intent.req_unowned_domain",
		"intent.req_transport_owned",
		"intent.domain_unrequired",
		"intent.ot_no_domain",
		"intent.vocab_drift",
	}
}
