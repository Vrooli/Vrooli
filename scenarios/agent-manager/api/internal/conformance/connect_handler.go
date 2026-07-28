package conformance

import (
	"context"
	"sort"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/metrics"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

type Handler struct {
	// agent-manager serves the validation contract for conformance reporting but
	// is not a readiness-probed phase provider, so it does not implement
	// DescribeProvider. Embedding the generated stub reports Unimplemented, which
	// is exactly the signal that makes consumers fall back to the legacy probe.
	scenariovalidationv1connect.UnimplementedScenarioValidationServiceHandler
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
	local := &commonv1.LocalMaturityAssessment{
		CurrentLevel:         level,
		NextLevel:            next,
		Clean:                clean,
		BlockingFindingCodes: blocking,
	}
	assessment := &commonv1.MaturityAssessment{Scenario: report.Scenario, Provider: "agent-manager", Phase: "agent-conformance", Version: "2.0.0", Local: local, Findings: findings, FindingsBySeverity: bySeverity}
	assessment.Presentation = buildCanonicalPresentation(assessment)
	return assessment
}

// buildCanonicalPresentation is the local, dependency-free projection of this
// provider's deliberately local maturity assessment. It mirrors
// maturity-go/assessment.BuildPhasePresentation for an assessment without
// provider capability levels: the one synthetic "local" capability is part of
// that canonical fallback contract.
func buildCanonicalPresentation(a *commonv1.MaturityAssessment) *commonv1.PhasePresentation {
	if a == nil || a.GetLocal() == nil {
		return nil
	}
	local := a.GetLocal()
	p := &commonv1.PhasePresentation{
		ContractVersion:      "v1",
		Provider:             a.GetProvider(),
		Phase:                a.GetPhase(),
		CurrentLevel:         local.GetCurrentLevel(),
		NextLevel:            local.GetNextLevel(),
		Clean:                local.GetClean(),
		UnknownCount:         local.GetUnknownCount(),
		BlockingFindingCodes: sortedNonBlank(local.GetBlockingFindingCodes()),
		AtMaximum:            local.GetClean() && strings.TrimSpace(local.GetNextLevel()) == "",
	}
	p.Capabilities = []*commonv1.PhaseCapabilityPresentation{{
		Id:                   "local",
		Label:                "Local Maturity",
		CurrentLevel:         local.GetCurrentLevel(),
		NextLevel:            local.GetNextLevel(),
		Clean:                local.GetClean(),
		UnknownCount:         local.GetUnknownCount(),
		BlockingFindingCodes: sortedNonBlank(local.GetBlockingFindingCodes()),
		PriorityRank:         1,
	}}
	if !p.GetAtMaximum() && strings.TrimSpace(p.GetPhase()) != "" {
		p.DocumentationTopics = []string{p.GetPhase() + " maturity next move"}
		for _, code := range p.GetBlockingFindingCodes() {
			p.DocumentationTopics = append(p.DocumentationTopics, p.GetPhase()+" "+code+" canonical fix")
			if len(p.DocumentationTopics) == 3 {
				break
			}
		}
	}
	return p
}

func sortedNonBlank(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	for _, value := range in {
		if value = strings.TrimSpace(value); value != "" {
			seen[value] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for value := range seen {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
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
	case CodeWorkflowInlinePrompt:
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
	case CodeWorkflowInlinePrompt:
		return commonv1.GlobalImpact_GLOBAL_IMPACT_EVOLVABILITY_GAP
	case CodeProfileInvalid, CodeProfileOrphan, CodeProfileOwnership, CodeProfileLegacy, CodeRoleUnresolved:
		return commonv1.GlobalImpact_GLOBAL_IMPACT_EVOLVABILITY_GAP
	default:
		return commonv1.GlobalImpact_GLOBAL_IMPACT_ADVISORY
	}
}
