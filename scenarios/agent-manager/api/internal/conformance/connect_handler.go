package conformance

import (
	"context"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/metrics"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

type Handler struct {
	service Service
}

func NewHandler(repoRoot string, posture ...PermissionPostureReader) *Handler {
	var reader PermissionPostureReader
	if len(posture) > 0 {
		reader = posture[0]
	}
	return &Handler{service: Service{RepoRoot: repoRoot, PermissionPosture: reader}}
}

func (h *Handler) ValidateScenario(_ context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	collector := metrics.Start()
	report, err := h.service.Validate(req.Msg.GetScenario(), req.Msg.GetPath())
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	detail, err := nativeDetail(report)
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	status := scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED
	if len(report.Findings) > 0 {
		status = scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED
	}
	packedDetail, err := anypb.New(detail)
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &scenariovalidationv1.ValidateScenarioResponse{
		Scenario:     report.Scenario,
		Status:       status,
		Assessment:   buildAssessment(report),
		NativeDetail: packedDetail,
		Metrics:      collector.Stop(),
	}
	return connect.NewResponse(response), nil
}

func (h *Handler) PreviewFix(_ context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return connect.NewResponse(&scenariovalidationv1.FixResponse{Scenario: req.Msg.GetScenario(), Messages: []string{"Agent conformance is read-only; follow the reported profile/dependency remediation."}}), nil
}

func (h *Handler) ApplyFix(_ context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return connect.NewResponse(&scenariovalidationv1.FixResponse{Scenario: req.Msg.GetScenario(), Messages: []string{"Agent conformance has no deterministic source rewriter and intentionally applies no changes."}}), nil
}

func nativeDetail(report Report) (*structpb.Struct, error) {
	items := make([]any, 0, len(report.Findings))
	for _, finding := range report.Findings {
		items = append(items, map[string]any{"code": finding.Code, "severity": finding.Severity, "location": finding.Location, "message": finding.Message, "remediation": finding.Remediation})
	}
	return structpb.NewStruct(map[string]any{"provider": "agent-manager", "phase": "agent-conformance", "scenario": report.Scenario, "findings": items})
}

func buildAssessment(report Report) *commonv1.MaturityAssessment {
	findings := make([]*commonv1.AssessmentFinding, 0, len(report.Findings))
	blocking := make([]string, 0, len(report.Findings))
	bySeverity := map[string]int32{}
	level := "L4"
	next := ""
	for _, finding := range report.Findings {
		findingLevel, required := maturityFor(finding)
		findings = append(findings, &commonv1.AssessmentFinding{Code: finding.Code, Severity: finding.Severity, Title: finding.Title, Message: finding.Message, Location: finding.Location, Remediation: finding.Remediation, FixClass: "detection_only", Maturity: &commonv1.FindingMaturity{CapabilityId: "portable_profiles", LocalLevel: findingLevel, GlobalImpact: globalImpactFor(finding), Dimension: "contracts", CleanRequirement: required}})
		bySeverity[finding.Severity]++
		if required == commonv1.CleanRequirement_CLEAN_REQUIREMENT_REQUIRED {
			blocking = append(blocking, finding.Code)
			if findingLevel < level {
				level = findingLevel
			}
		}
	}
	clean := len(findings) == 0
	if !clean && level == "L4" {
		level = "L3"
	}
	if level < "L4" {
		next = "L" + string(level[1]+1)
	}
	return &commonv1.MaturityAssessment{Scenario: report.Scenario, Provider: "agent-manager", Phase: "agent-conformance", Version: "2.0.0", Local: &commonv1.LocalMaturityAssessment{CurrentLevel: level, NextLevel: next, Clean: clean, BlockingFindingCodes: blocking}, Findings: findings, FindingsBySeverity: bySeverity}
}

func maturityFor(finding Finding) (string, commonv1.CleanRequirement) {
	switch finding.Code {
	case CodeDependencyMissing, CodeDependencyDisabled:
		return "L0", commonv1.CleanRequirement_CLEAN_REQUIREMENT_REQUIRED
	case CodeProfileInvalid, CodeProfileOrphan, CodeProfileOwnership, CodeProfileLegacy:
		return "L1", commonv1.CleanRequirement_CLEAN_REQUIREMENT_REQUIRED
	case CodeRoleUnresolved:
		return "L2", commonv1.CleanRequirement_CLEAN_REQUIREMENT_REQUIRED
	case CodeDirectSpawnBypass:
		return "L3", commonv1.CleanRequirement_CLEAN_REQUIREMENT_REQUIRED
	case CodePermissionPosture:
		return "L4", commonv1.CleanRequirement_CLEAN_REQUIREMENT_REQUIRED
	default:
		return "L3", commonv1.CleanRequirement_CLEAN_REQUIREMENT_ADVISORY
	}
}

func globalImpactFor(finding Finding) commonv1.GlobalImpact {
	switch finding.Code {
	case CodeDependencyMissing, CodeDependencyDisabled, CodeDirectSpawnBypass:
		return commonv1.GlobalImpact_GLOBAL_IMPACT_CAPABILITY_GAP
	case CodePermissionPosture:
		return commonv1.GlobalImpact_GLOBAL_IMPACT_SAFETY_BLOCKER
	case CodeProfileInvalid, CodeProfileOrphan, CodeProfileOwnership, CodeProfileLegacy, CodeRoleUnresolved:
		return commonv1.GlobalImpact_GLOBAL_IMPACT_EVOLVABILITY_GAP
	default:
		return commonv1.GlobalImpact_GLOBAL_IMPACT_ADVISORY
	}
}
