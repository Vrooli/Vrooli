package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/policy"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

const captureConformancePhase = "event-capture-conformance"

// captureValidationHandler is the Events-owned proof that an opt-in receipt
// declaration is valid and has reached the global policy snapshot.
type captureValidationHandler struct {
	scenariovalidationv1connect.UnimplementedScenarioValidationServiceHandler
	repoRoot string
	policies policy.Store
}

func newCaptureValidationHandler(repoRoot string, policies policy.Store) *captureValidationHandler {
	return &captureValidationHandler{repoRoot: repoRoot, policies: policies}
}

func (h *captureValidationHandler) DescribeProvider(_ context.Context, _ *connect.Request[scenariovalidationv1.DescribeProviderRequest]) (*connect.Response[scenariovalidationv1.DescribeProviderResponse], error) {
	return connect.NewResponse(&scenariovalidationv1.DescribeProviderResponse{
		Provider: "vrooli-events", Phase: captureConformancePhase, SpecVersion: "1.0.0", Contract: "scenario-validation/v1",
		Capabilities: &scenariovalidationv1.ProviderCapabilities{DeliveryMode: "inline"},
	}), nil
}

func (h *captureValidationHandler) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	started := time.Now()
	scenario := strings.TrimSpace(req.Msg.GetScenario())
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("scenario is required"))
	}
	rules, err := loadCaptureDeclarationRulesAtRoot(h.repoRoot, scenario)
	findings := []captureValidationFinding{}
	if err != nil {
		findings = append(findings, captureValidationFinding{Code: "event_capture.declaration_invalid", Title: "Receipt declaration is invalid", Message: err.Error(), Remediation: "Repair the declaration under .vrooli/vrooli-events and run capture-preview."})
	} else {
		existing, listErr := h.policies.ListReceiptProjections(ctx, policy.ReceiptProjectionFilters{})
		if listErr != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list receipt policies: %w", listErr))
		}
		byID := make(map[string]policy.ReceiptProjectionRule, len(existing))
		for _, rule := range existing {
			byID[rule.PolicyID] = rule
		}
		for _, rule := range rules {
			actual, ok := byID[rule.PolicyID]
			if !ok || !sameCaptureRule(actual, rule) {
				findings = append(findings, captureValidationFinding{Code: "event_capture.policy_unreconciled", Title: "Receipt policy is not reconciled", Message: fmt.Sprintf("policy %q is absent or differs from its declaration", rule.PolicyID), Remediation: fmt.Sprintf("Run vrooli-events capture-reconcile --scenario %s.", scenario)})
			}
		}
	}
	return connect.NewResponse(captureValidationResponse(scenario, findings, time.Since(started))), nil
}

type captureValidationFinding struct{ Code, Title, Message, Remediation string }

func captureValidationResponse(scenario string, findings []captureValidationFinding, elapsed time.Duration) *scenariovalidationv1.ValidateScenarioResponse {
	clean := len(findings) == 0
	level, next := "L2", ""
	blocking := make([]string, 0, len(findings))
	assessmentFindings := make([]*commonv1.AssessmentFinding, 0, len(findings))
	nativeFindings := make([]any, 0, len(findings))
	for _, finding := range findings {
		blocking = append(blocking, finding.Code)
		maturity := captureFindingMaturity(finding.Code)
		if maturity.GetLocalLevel() == "L0" {
			level, next = "L0", "L1"
		} else if level != "L0" {
			level, next = "L1", "L2"
		}
		assessmentFindings = append(assessmentFindings, &commonv1.AssessmentFinding{Code: finding.Code, Severity: "SEVERITY_ERROR", Title: finding.Title, Message: finding.Message, Remediation: finding.Remediation, FixClass: "manual", Maturity: maturity})
		nativeFindings = append(nativeFindings, map[string]any{"code": finding.Code, "message": finding.Message, "remediation": finding.Remediation})
	}
	sort.Strings(blocking)
	assessment := &commonv1.MaturityAssessment{Scenario: scenario, Provider: "vrooli-events", Phase: captureConformancePhase, Version: "1.0.0", Local: &commonv1.LocalMaturityAssessment{CurrentLevel: level, NextLevel: next, Clean: clean, BlockingFindingCodes: blocking}, Findings: assessmentFindings, FindingsBySeverity: map[string]int32{"SEVERITY_ERROR": int32(len(findings))}}
	levels := []*commonv1.LocalMaturityLevel{
		{Id: "L0", Name: "Invalid", CapabilitySummary: "Receipt capture is not safely configured.", NextUnlock: "Repair the declaration or reconcile the policies."},
		{Id: "L1", Name: "Validated", CapabilitySummary: "Capture contract is operational.", NextUnlock: "Keep the contract continuously verified."},
		{Id: "L2", Name: "Conformant", CapabilitySummary: "Receipt capture is continuously verified."},
	}
	assessment.Local.Levels = levels
	assessment.Capabilities = []*commonv1.CapabilityMaturityAssessment{{Id: "declared_capture", Label: "Declared Receipt Capture", CurrentLevel: level, NextLevel: next, Levels: levels, BlockingFindingCodes: blocking, Clean: clean, PriorityRank: 1}}
	assessment.Presentation = capturePhasePresentation(assessment)
	detail, _ := structpb.NewStruct(map[string]any{"scenario": scenario, "findings": nativeFindings})
	packed, _ := anypb.New(detail)
	status := scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED
	if !clean {
		status = scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED
	}
	return &scenariovalidationv1.ValidateScenarioResponse{Scenario: scenario, Status: status, Assessment: assessment, NativeDetail: packed, Metrics: &commonv1.ExecutionMetrics{WallClockMs: elapsed.Milliseconds()}}
}

func levelIndex(level string) int {
	if level == "L0" {
		return 0
	}
	if level == "L1" {
		return 1
	}
	return 2
}

func captureFindingMaturity(code string) *commonv1.FindingMaturity {
	maturity := &commonv1.FindingMaturity{Dimension: "contracts", CleanRequirement: commonv1.CleanRequirement_CLEAN_REQUIREMENT_REQUIRED, CapabilityId: "declared_capture"}
	switch code {
	case "event_capture.policy_unreconciled":
		maturity.LocalLevel = "L1"
		maturity.GlobalImpact = commonv1.GlobalImpact_GLOBAL_IMPACT_EVOLVABILITY_GAP
	default:
		maturity.LocalLevel = "L0"
		maturity.GlobalImpact = commonv1.GlobalImpact_GLOBAL_IMPACT_FOUNDATION_BLOCKER
	}
	return maturity
}

// capturePhasePresentation mirrors the portable maturity v1 projection. This
// provider deliberately avoids importing maturity-go because that library is
// not part of the Events scenario's approved dependency surface.
func capturePhasePresentation(assessment *commonv1.MaturityAssessment) *commonv1.PhasePresentation {
	local := assessment.GetLocal()
	levels := local.GetLevels()
	current := levels[levelIndex(local.GetCurrentLevel())]
	blocking := append([]string(nil), local.GetBlockingFindingCodes()...)
	sort.Strings(blocking)
	presentation := &commonv1.PhasePresentation{
		ContractVersion: "v1", Provider: assessment.GetProvider(), Phase: assessment.GetPhase(),
		CurrentLevel: local.GetCurrentLevel(), NextLevel: local.GetNextLevel(), Clean: local.GetClean(),
		UnknownCount: local.GetUnknownCount(), BlockingFindingCodes: blocking,
		AtMaximum: local.GetClean() && local.GetNextLevel() == "", CurrentLevelLabel: current.GetName(),
		CeilingLevel: levels[len(levels)-1].GetId(), NorthStar: levels[len(levels)-1].GetCapabilitySummary(),
		NextAction: current.GetNextUnlock(),
	}
	capability := assessment.GetCapabilities()[0]
	capabilityPresentation := &commonv1.PhaseCapabilityPresentation{
		Id: capability.GetId(), Label: capability.GetLabel(), CurrentLevel: capability.GetCurrentLevel(), NextLevel: capability.GetNextLevel(),
		Clean: capability.GetClean(), UnknownCount: capability.GetUnknownCount(), BlockingFindingCodes: append([]string(nil), blocking...),
		PriorityRank: capability.GetPriorityRank(), PriorityReason: capability.GetPriorityReason(), CurrentLevelLabel: current.GetName(),
	}
	findingsByCode := make(map[string]*commonv1.PhasePresentationFinding)
	for _, finding := range assessment.GetFindings() {
		if finding.GetMaturity().GetCapabilityId() != capability.GetId() {
			continue
		}
		entry := findingsByCode[finding.GetCode()]
		if entry == nil {
			entry = &commonv1.PhasePresentationFinding{
				Code: finding.GetCode(), Severity: finding.GetSeverity(), Title: finding.GetTitle(), Message: finding.GetMessage(), Remediation: finding.GetRemediation(),
				FixAffordance: commonv1.FixAffordance_FIX_AFFORDANCE_MANUAL,
			}
			findingsByCode[finding.GetCode()] = entry
			capabilityPresentation.Findings = append(capabilityPresentation.Findings, entry)
		}
		entry.Count++
	}
	sort.Slice(capabilityPresentation.Findings, func(i, j int) bool {
		return capabilityPresentation.Findings[i].GetCode() < capabilityPresentation.Findings[j].GetCode()
	})
	presentation.Capabilities = []*commonv1.PhaseCapabilityPresentation{capabilityPresentation}
	if !presentation.GetAtMaximum() {
		presentation.DocumentationTopics = []string{assessment.GetPhase() + " maturity next move"}
		for _, code := range blocking {
			presentation.DocumentationTopics = append(presentation.DocumentationTopics, assessment.GetPhase()+" "+code+" canonical fix")
			if len(presentation.DocumentationTopics) == 3 {
				break
			}
		}
	}
	return presentation
}

func sameCaptureRule(a, b policy.ReceiptProjectionRule) bool {
	return a.PolicyID == b.PolicyID && a.TargetScenario == b.TargetScenario && a.OperationPattern == b.OperationPattern && a.Protocol == b.Protocol && a.EventType == b.EventType && a.ResponseType == b.ResponseType && strings.Join(a.ResponseFields, "\x00") == strings.Join(b.ResponseFields, "\x00") && a.RetentionDays == b.RetentionDays && strings.Join(a.ReadPrincipals, "\x00") == strings.Join(b.ReadPrincipals, "\x00") && a.Enabled == b.Enabled
}

func captureValidationRepoRoot() string {
	if root := os.Getenv("PROJECT_ROOT"); root != "" {
		return root
	}
	root, _ := filepath.Abs(filepath.Join("..", "..", ".."))
	return root
}

var _ scenariovalidationv1connect.ScenarioValidationServiceHandler = (*captureValidationHandler)(nil)
