// Package validation hosts the Connect-RPC ScenarioValidationService handler
// that exposes Test Genie's provider-conformance judgment: it validates target
// scenarios that declare themselves as Test Genie phase providers through
// .vrooli/test-genie.json. It is a thin transport wrapper over
// internal/providerconformance.
package validation

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/metrics"
	"github.com/vrooli/maturity-go/assessment"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"google.golang.org/protobuf/types/known/structpb"

	"test-genie/internal/providerconformance"
	"test-genie/internal/selfhealth"
)

// Service implements scenariovalidationconnect.ScenarioValidationServiceHandler
// for the provider-conformance phase.
type Service struct {
	logger    *log.Logger
	validator *providerconformance.Service
	spec      *assessment.Spec
	describer assessment.Describer
}

// NewService builds the provider-conformance validation service. repoRoot is
// the repository root; the maturity spec is Test Genie's own descriptor-embedded
// provider-conformance spec. The live contract probe uses the same Connect
// probe as the fleet conformance scan.
func NewService(logger *log.Logger, repoRoot string, spec *assessment.Spec) *Service {
	if logger == nil {
		logger = log.Default()
	}
	validator := providerconformance.New(repoRoot)
	validator.Probe = selfhealth.DefaultConformanceProbe
	validator.DurableProbe = providerconformance.DefaultDurableConformanceProbe
	// Test Genie answers the same readiness contract it asks of others. A load
	// failure yields the zero Describer, which reports Unimplemented.
	describer, _ := assessment.LoadDescriber(filepath.Join(repoRoot, "scenarios", "test-genie"))
	return &Service{logger: logger, validator: validator, spec: spec, describer: describer}
}

// NewServiceForTest builds a Service with an injected validator seam.
func NewServiceForTest(logger *log.Logger, validator *providerconformance.Service, spec *assessment.Spec) *Service {
	if logger == nil {
		logger = log.Default()
	}
	return &Service{logger: logger, validator: validator, spec: spec}
}

func (s *Service) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	collector := metrics.Start()
	if s.spec == nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("provider-conformance maturity spec is unavailable"))
	}
	report, err := s.validator.ValidateScenario(ctx, req.Msg.GetScenario(), req.Msg.GetPath())
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	maturity, err := providerconformance.BuildMaturityAssessment(report.Scenario, report.Findings, *s.spec)
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build maturity assessment: %w", err))
	}
	native, err := nativeDetail(report)
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build native detail: %w", err))
	}
	resp, err := assessment.BuildValidationResponse(report.Scenario, maturity, native, collector.Stop())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build shared validation response: %w", err))
	}
	return connect.NewResponse(resp), nil
}

func (s *Service) PreviewFix(context.Context, *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("provider-conformance fixes are manual: descriptor authorship requires provider-owner judgment"))
}

func (s *Service) ApplyFix(context.Context, *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("provider-conformance fixes are manual: descriptor authorship requires provider-owner judgment"))
}

func nativeDetail(report providerconformance.Report) (*structpb.Struct, error) {
	detail := map[string]any{
		"scenario": report.Scenario,
		"status":   report.Summary.Status(),
		"probed":   report.Probed,
		"summary": map[string]any{
			"errors":   report.Summary.Errors,
			"warnings": report.Summary.Warnings,
		},
	}
	if report.Phase != "" {
		detail["phase"] = report.Phase
	}
	if report.ProbeSkipReason != "" {
		detail["probeSkipReason"] = report.ProbeSkipReason
	}
	return structpb.NewStruct(detail)
}

// Served composes this service with the shared DescribeProvider implementation.
// Test Genie is itself a provider (the provider-conformance phase), so it must
// answer the same readiness contract it asks every other provider to answer.
func (s *Service) Served() scenariovalidationv1connect.ScenarioValidationServiceHandler {
	return assessment.Serve(s, s.describer)
}
