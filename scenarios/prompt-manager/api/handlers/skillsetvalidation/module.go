package skillsetvalidation

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	maturityassessment "github.com/vrooli/maturity-go/assessment"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"prompt-manager/internal/skillset"
)

// NewConnectMount mounts the skill-set validator. recall is this scenario's own
// skill search, used for the advisory intent-recall check; nil skips the
// measurement (the missing-intent warning still applies).
func NewConnectMount(repoRoot string, recall skillset.Recaller) (string, http.Handler) {
	h := &handler{repoRoot: repoRoot, recall: recall}
	path, endpoint := scenariovalidationconnect.NewScenarioValidationServiceHandler(h)
	return path, endpoint
}

type handler struct {
	scenariovalidationconnect.UnimplementedScenarioValidationServiceHandler
	repoRoot string
	recall   skillset.Recaller
}

func (h *handler) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	scenario := req.Msg.GetScenario()
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("scenario is required"))
	}
	// Honour a caller-resolved scenario path (isolated workspaces, template deep
	// validation) before the conventional repository location.
	root := req.Msg.GetPath()
	if root == "" {
		root = skillset.ScenarioRoot(h.repoRoot, scenario)
	}
	result := skillset.ValidateWithIntents(ctx, root, scenario, h.recall)
	clean := result.Status == "ok"
	level := "L1"
	if clean {
		level = "L2"
	}
	if result.Status == "unavailable" {
		level = "L0"
	}
	now := time.Now()
	levels := []*commonv1.LocalMaturityLevel{{Id: "L0", Name: "Skill set unavailable", StatusLabel: "Unavailable", NextUnlock: "Readable service metadata."}, {Id: "L1", Name: "Skill set inspectable", StatusLabel: "Foundation", NextUnlock: "Clear skill-set findings."}, {Id: "L2", Name: "Skill set clean", StatusLabel: "Complete", CapabilitySummary: "Every owed skill role is declared and reachable."}}
	local := &commonv1.LocalMaturityAssessment{CurrentLevel: level, NextLevel: func() string {
		if clean {
			return ""
		}
		if level == "L0" {
			return "L1"
		}
		return "L2"
	}(), Clean: clean, BlockingFindingCodes: result.Findings, Levels: levels}
	findings, bySeverity := skillSetFindings(result)
	findings, bySeverity = appendIntentWarnings(findings, bySeverity, result)
	assessment := &commonv1.MaturityAssessment{Scenario: scenario, Provider: "prompt-manager", Phase: "skill-set", Version: "2.0.0", Local: local, Findings: findings, FindingsBySeverity: bySeverity}
	assessment.Presentation = maturityassessment.BuildPhasePresentation(assessment)
	detail, _ := structpb.NewStruct(map[string]any{"roles": result.Roles, "findings": result.Findings})
	native, _ := anypb.New(detail)
	status := scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED
	if !clean {
		status = scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED
	}
	return connect.NewResponse(&scenariovalidationv1.ValidateScenarioResponse{Scenario: scenario, Status: status, Assessment: assessment, NativeDetail: native, Metrics: &commonv1.ExecutionMetrics{StartedAt: timestamppb.New(now), CompletedAt: timestamppb.Now()}}), nil
}

// findingMaturity attaches the maturity metadata every assessment finding must
// carry. It mirrors the ladders declared in .vrooli/test-genie.json: advisory
// intent findings sit on L2 and never block, structural ones block L0/L1.
func findingMaturity(code string) *commonv1.FindingMaturity {
	m := &commonv1.FindingMaturity{
		CapabilityId: "declaration_integrity", LocalLevel: "L1", Dimension: "structure",
		GlobalImpact:     commonv1.GlobalImpact_GLOBAL_IMPACT_ADVISORY,
		CleanRequirement: commonv1.CleanRequirement_CLEAN_REQUIREMENT_REQUIRED,
	}
	switch code {
	case "skill-set.service_missing", "skill-set.service_malformed":
		m.LocalLevel = "L0"
	case "skill-set.role_missing":
		m.CapabilityId = "role_coverage"
	case "skill-set.dialect_invalid":
		m.CapabilityId, m.Dimension = "dialect_conformance", "standards"
	case "skill-set.intent_missing", "skill-set.intent_recall":
		m.CapabilityId, m.LocalLevel = "role_coverage", "L2"
		m.CleanRequirement = commonv1.CleanRequirement_CLEAN_REQUIREMENT_ADVISORY
	}
	return m
}

// skillSetRemediation names the fix for each validator finding code.
var skillSetRemediation = map[string]string{
	"skill-set.role_missing":      "Declare the owed role in .vrooli/service.json skills with a source SKILL.md, or record skills.waivers.<role> with a reason and declared_at.",
	"skill-set.source_missing":    "Point skills.<role>.source in .vrooli/service.json at an existing SKILL.md inside the scenario.",
	"skill-set.source_unreadable": "Make the declared SKILL.md readable by the prompt-manager API.",
	"skill-set.dialect_invalid":   "Give the declared SKILL.md frontmatter metadata with kind and modes (see docs/agent-system/SKILL_AUTHORING.md).",
	"skill-set.waiver_invalid":    "Give skills.waivers.<role> in .vrooli/service.json both a reason and a declared_at timestamp.",
	"skill-set.intent_missing":    "Add .vrooli/intents.json (or a scenarios/search-hub/intents/<scenario>.json corpus) with the plain requests this scenario's skill should answer.",
	"skill-set.intent_recall":     "Rewrite the skill description and tags in the words of the missed requests (see `search-hub intents run --scenario <name>`); the skill search returns nothing below its similarity floor.",
	"skill-set.service_missing":   "Create .vrooli/service.json for the scenario.",
	"skill-set.service_malformed": "Repair .vrooli/service.json so it parses as JSON.",
}

// skillSetFindings turns the validator's finding codes into typed assessment
// findings. A failed validation must carry the ERROR findings that fail it, so
// a consumer counting severities never sees "0 errors" on a red phase.
func skillSetFindings(result skillset.Result) ([]*commonv1.AssessmentFinding, map[string]int32) {
	if len(result.Findings) == 0 {
		return nil, nil
	}
	reasons := make(map[string]skillset.Role, len(result.Roles))
	for _, role := range result.Roles {
		reasons[role.Role] = role
	}
	findings := make([]*commonv1.AssessmentFinding, 0, len(result.Findings))
	for _, raw := range result.Findings {
		code, role, _ := strings.Cut(raw, ":")
		message := code
		location := ".vrooli/service.json"
		if role != "" {
			message = role + " skill: " + code
			location += " skills." + role
			if declared, ok := reasons[role]; ok {
				if declared.Reason != "" {
					message = role + " skill: " + declared.Reason
				}
				if declared.Source != "" {
					location = declared.Source
				}
			}
		}
		remediation := skillSetRemediation[code]
		if role != "" {
			remediation = strings.ReplaceAll(remediation, "<role>", role)
		}
		findings = append(findings, &commonv1.AssessmentFinding{
			Code:        code,
			Severity:    "SEVERITY_ERROR",
			Title:       code,
			Message:     message,
			Location:    location,
			Remediation: remediation,
			Maturity:    findingMaturity(code),
		})
	}
	return findings, map[string]int32{"SEVERITY_ERROR": int32(len(findings))}
}

// appendIntentWarnings adds the advisory intent findings. They carry
// SEVERITY_WARNING and never affect pass/fail.
func appendIntentWarnings(findings []*commonv1.AssessmentFinding, by map[string]int32, result skillset.Result) ([]*commonv1.AssessmentFinding, map[string]int32) {
	if len(result.Warnings) == 0 {
		return findings, by
	}
	if by == nil {
		by = map[string]int32{}
	}
	for _, raw := range result.Warnings {
		code, _, _ := strings.Cut(raw, ":")
		message := "the scenario ships a skill but declares no intent cases, so nothing measures whether the skill can be found"
		if code == "skill-set.intent_recall" && result.Intents != nil {
			message = fmt.Sprintf("the scenario's own skill surfaced for %d of %d declared intents; missed: %s",
				result.Intents.Passed, result.Intents.Cases, strings.Join(result.Intents.Missed, "; "))
		}
		findings = append(findings, &commonv1.AssessmentFinding{
			Code:        code,
			Severity:    "SEVERITY_WARNING",
			Title:       code,
			Message:     message,
			Location:    "skills/",
			Remediation: skillSetRemediation[code],
			Maturity:    findingMaturity(code),
		})
		by["SEVERITY_WARNING"]++
	}
	return findings, by
}
