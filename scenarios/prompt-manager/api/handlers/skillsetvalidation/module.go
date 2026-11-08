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

func NewConnectMount(repoRoot string) (string, http.Handler) {
	h := &handler{repoRoot: repoRoot}
	path, endpoint := scenariovalidationconnect.NewScenarioValidationServiceHandler(h)
	return path, endpoint
}

type handler struct {
	scenariovalidationconnect.UnimplementedScenarioValidationServiceHandler
	repoRoot string
}

func (h *handler) ValidateScenario(_ context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
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
	result := skillset.Validate(root, scenario)
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

// skillSetRemediation names the fix for each validator finding code.
var skillSetRemediation = map[string]string{
	"skill-set.role_missing":      "Declare the owed role in .vrooli/service.json skills with a source SKILL.md, or record skills.waivers.<role> with a reason and declared_at.",
	"skill-set.source_missing":    "Point skills.<role>.source in .vrooli/service.json at an existing SKILL.md inside the scenario.",
	"skill-set.source_unreadable": "Make the declared SKILL.md readable by the prompt-manager API.",
	"skill-set.dialect_invalid":   "Give the declared SKILL.md frontmatter metadata with kind and modes (see docs/agent-system/SKILL_AUTHORING.md).",
	"skill-set.waiver_invalid":    "Give skills.waivers.<role> in .vrooli/service.json both a reason and a declared_at timestamp.",
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
		})
	}
	return findings, map[string]int32{"SEVERITY_ERROR": int32(len(findings))}
}
