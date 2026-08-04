package main

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/maturity-go/assessment"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/policy"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
	"google.golang.org/protobuf/proto"
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
	events   store.Store
}

func newCaptureValidationHandler(repoRoot string, policies policy.Store, events store.Store) *captureValidationHandler {
	return &captureValidationHandler{repoRoot: repoRoot, policies: policies, events: events}
}

var literalInstanceIdentifier = regexp.MustCompile(`(?i)(?:^|/)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}(?:/|$)`)

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
			if literalInstanceIdentifier.MatchString(rule.OperationPattern) {
				findings = append(findings, captureValidationFinding{Code: "event_capture.literal_instance_identifier", Title: "Receipt policy names a literal instance", Message: fmt.Sprintf("policy %q operation %q contains a literal instance identifier", rule.PolicyID, rule.OperationPattern), Remediation: "Use a stable operation pattern, never an instance-specific URL."})
			}
			actual, ok := byID[rule.PolicyID]
			if !ok || !sameCaptureRule(actual, rule) {
				findings = append(findings, captureValidationFinding{Code: "event_capture.policy_unreconciled", Title: "Receipt policy is not reconciled", Message: fmt.Sprintf("policy %q is absent or differs from its declaration", rule.PolicyID), Remediation: fmt.Sprintf("Run vrooli-events capture-reconcile --scenario %s.", scenario)})
			} else if !rule.NeverExercised && !receiptEmitted(ctx, h.events, rule) {
				findings = append(findings, captureValidationFinding{Code: "event_capture.policy_unexercised", Title: "Receipt policy has not emitted", Message: fmt.Sprintf("policy %q has no durable receipt emission", rule.PolicyID), Remediation: "Exercise the declared operation or mark a genuinely new declaration neverExercised."})
			}
		}
	}
	return connect.NewResponse(captureValidationResponse(scenario, findings, time.Since(started))), nil
}

func receiptEmitted(ctx context.Context, events store.Store, rule policy.ReceiptProjectionRule) bool {
	if events == nil {
		return false
	}
	items, err := events.Query(ctx, store.QueryFilters{EventType: receiptEventType, Target: rule.TargetScenario, Limit: 1000})
	if err != nil {
		return false
	}
	for _, item := range items {
		var envelope domain.EventEnvelope
		if proto.Unmarshal(item.Payload, &envelope) == nil && envelope.GetTarget().GetScenario() == rule.TargetScenario && envelope.GetTarget().GetOperation() == rule.OperationPattern {
			return true
		}
	}
	return false
}

type captureValidationFinding struct{ Code, Title, Message, Remediation string }

func captureValidationResponse(scenario string, findings []captureValidationFinding, elapsed time.Duration) *scenariovalidationv1.ValidateScenarioResponse {
	clean := len(findings) == 0
	level, next := "L3", ""
	blocking := make([]string, 0, len(findings))
	assessmentFindings := make([]*commonv1.AssessmentFinding, 0, len(findings))
	nativeFindings := make([]any, 0, len(findings))
	for _, finding := range findings {
		blocking = append(blocking, finding.Code)
		maturity := captureFindingMaturity(finding.Code)
		switch currentCaptureLevel(maturity.GetLocalLevel()) {
		case "L0":
			level, next = "L0", "L1"
		case "L1":
			if level == "L0" {
				break
			}
			level, next = "L1", "L2"
		case "L2":
			if level == "L0" || level == "L1" {
				break
			}
			level, next = "L2", "L3"
		}
		assessmentFindings = append(assessmentFindings, &commonv1.AssessmentFinding{Code: finding.Code, Severity: "SEVERITY_ERROR", Title: finding.Title, Message: finding.Message, Remediation: finding.Remediation, FixClass: "manual", Maturity: maturity})
		nativeFindings = append(nativeFindings, map[string]any{"code": finding.Code, "message": finding.Message, "remediation": finding.Remediation})
	}
	sort.Strings(blocking)
	report := &commonv1.MaturityAssessment{Scenario: scenario, Provider: "vrooli-events", Phase: captureConformancePhase, Version: "1.0.0", Local: &commonv1.LocalMaturityAssessment{CurrentLevel: level, NextLevel: next, Clean: clean, BlockingFindingCodes: blocking}, Findings: assessmentFindings, FindingsBySeverity: map[string]int32{"SEVERITY_ERROR": int32(len(findings))}}
	levels := []*commonv1.LocalMaturityLevel{
		{Id: "L0", Name: "Invalid", CapabilitySummary: "Receipt capture is not safely configured.", NextUnlock: "Repair the declaration or reconcile the policies."},
		{Id: "L1", Name: "Validated", CapabilitySummary: "Capture contract is operational.", NextUnlock: "Keep the contract continuously verified."},
		{Id: "L2", Name: "Conformant", CapabilitySummary: "Receipt capture is continuously verified."},
		{Id: "L3", Name: "Proven", CapabilitySummary: "Every declared receipt policy has durable emission evidence."},
	}
	report.Local.Levels = levels
	report.Capabilities = []*commonv1.CapabilityMaturityAssessment{{Id: "declared_capture", Label: "Declared Receipt Capture", CurrentLevel: level, NextLevel: next, Levels: levels, BlockingFindingCodes: blocking, Clean: clean, PriorityRank: 1}}
	report.Presentation = assessment.BuildPhasePresentation(report)
	detail, _ := structpb.NewStruct(map[string]any{"scenario": scenario, "findings": nativeFindings})
	packed, _ := anypb.New(detail)
	status := scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED
	if !clean {
		status = scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED
	}
	return &scenariovalidationv1.ValidateScenarioResponse{Scenario: scenario, Status: status, Assessment: report, NativeDetail: packed, Metrics: &commonv1.ExecutionMetrics{WallClockMs: elapsed.Milliseconds()}}
}

func captureFindingMaturity(code string) *commonv1.FindingMaturity {
	maturity := &commonv1.FindingMaturity{Dimension: "contracts", CleanRequirement: commonv1.CleanRequirement_CLEAN_REQUIREMENT_REQUIRED, CapabilityId: "declared_capture"}
	switch code {
	case "event_capture.policy_unreconciled":
		maturity.LocalLevel = "L2"
		maturity.GlobalImpact = commonv1.GlobalImpact_GLOBAL_IMPACT_EVOLVABILITY_GAP
	case "event_capture.policy_unexercised":
		maturity.LocalLevel = "L3"
		maturity.GlobalImpact = commonv1.GlobalImpact_GLOBAL_IMPACT_EVOLVABILITY_GAP
	default:
		maturity.LocalLevel = "L1"
		maturity.GlobalImpact = commonv1.GlobalImpact_GLOBAL_IMPACT_FOUNDATION_BLOCKER
	}
	return maturity
}

// A finding's maturity impact names the destination rung it blocks. The
// assessment reports the highest achieved rung, which is immediately below it.
func currentCaptureLevel(impact string) string {
	switch impact {
	case "L3":
		return "L2"
	case "L2":
		return "L1"
	default:
		return "L0"
	}
}

func sameCaptureRule(a, b policy.ReceiptProjectionRule) bool {
	return a.PolicyID == b.PolicyID && a.TargetScenario == b.TargetScenario && a.OperationPattern == b.OperationPattern && a.Protocol == b.Protocol && a.EventType == b.EventType && a.ResponseType == b.ResponseType && strings.Join(a.ResponseFields, "\x00") == strings.Join(b.ResponseFields, "\x00") && a.RetentionDays == b.RetentionDays && strings.Join(a.ReadPrincipals, "\x00") == strings.Join(b.ReadPrincipals, "\x00") && a.Enabled == b.Enabled
}

func captureValidationRepoRoot() string {
	root, _ := filepath.Abs(filepath.Join("..", "..", ".."))
	return root
}

var _ scenariovalidationv1connect.ScenarioValidationServiceHandler = (*captureValidationHandler)(nil)
