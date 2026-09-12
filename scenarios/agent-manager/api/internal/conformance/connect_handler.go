package conformance

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"agent-manager/internal/runsignal"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/metrics"
	assessmentpkg "github.com/vrooli/maturity-go/assessment"
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
	service  Service
	repoRoot string
}

func NewHandler(repoRoot string, posture ...PermissionPostureReader) *Handler {
	var reader PermissionPostureReader
	if len(posture) > 0 {
		reader = posture[0]
	}
	return &Handler{service: Service{RepoRoot: repoRoot, PermissionPosture: reader}, repoRoot: repoRoot}
}

func (h *Handler) ValidateScenario(_ context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	collector := metrics.Start()
	report, err := h.service.Validate(req.Msg.GetScenario(), req.Msg.GetPath())
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	var accuracy []runsignal.DetectorAccuracy
	if report.Scenario == agentManagerSelfScenario {
		accuracy, err = runsignal.ClassificationAccuracy(h.repoRoot)
		if err != nil {
			code, title := "classifier_accuracy.unmeasured", "Classifier accuracy could not be measured"
			if errors.Is(err, runsignal.ErrIncompleteLabelCoverage) {
				code, title = "classifier_accuracy.coverage_missing", "Classifier labelled coverage is incomplete"
			}
			report.add(code, title, "internal/runsignal/testdata/classification", "Repair the committed labelled corpus and rerun the classifier accuracy gate: "+err.Error())
		} else {
			for _, result := range accuracy {
				if result.Precision < result.Threshold || result.Recall < result.Threshold {
					report.add("classifier_accuracy.below_threshold", fmt.Sprintf("Detector %s is below its accuracy threshold", result.ID), result.ID, "Improve the detector or labelled corpus until both precision and recall meet the committed threshold.")
				}
			}
		}
	}
	detail, err := nativeDetail(report, accuracy)
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
		Assessment:   buildAssessment(report, accuracy),
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

func nativeDetail(report Report, accuracy []runsignal.DetectorAccuracy) (*structpb.Struct, error) {
	items := make([]any, 0, len(report.Findings))
	for _, finding := range report.Findings {
		items = append(items, map[string]any{"code": finding.Code, "severity": finding.Severity, "location": finding.Location, "message": finding.Message, "remediation": finding.Remediation})
	}
	results := make([]any, 0, len(accuracy))
	for _, result := range accuracy {
		results = append(results, map[string]any{"id": result.ID, "precision": result.Precision, "recall": result.Recall, "threshold": result.Threshold})
	}
	return structpb.NewStruct(map[string]any{"provider": "agent-manager", "phase": "agent-conformance", "scenario": report.Scenario, "findings": items, "classifier_accuracy": results})
}

func buildAssessment(report Report, publishedAccuracy ...[]runsignal.DetectorAccuracy) *commonv1.MaturityAssessment {
	findings := make([]*commonv1.AssessmentFinding, 0, len(report.Findings))
	blocking := make([]string, 0, len(report.Findings))
	bySeverity := map[string]int32{}
	level := "L4"
	next := ""
	for _, finding := range report.Findings {
		findingLevel, required := maturityFor(finding)
		findings = append(findings, &commonv1.AssessmentFinding{Code: finding.Code, Severity: finding.Severity, Title: finding.Title, Message: finding.Message, Location: finding.Location, Remediation: finding.Remediation, FixClass: "detection_only", Maturity: &commonv1.FindingMaturity{CapabilityId: capabilityFor(finding), LocalLevel: findingLevel, GlobalImpact: globalImpactFor(finding), Dimension: dimensionFor(finding), CleanRequirement: required}})
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
	if report.Scenario == agentManagerSelfScenario {
		var accuracy []runsignal.DetectorAccuracy
		if len(publishedAccuracy) > 0 {
			accuracy = publishedAccuracy[0]
		}
		assessment.Capabilities = []*commonv1.CapabilityMaturityAssessment{classifierAccuracyCapability(report, accuracy)}
	}
	assessment.Presentation = assessmentpkg.BuildPhasePresentation(assessment)
	return assessment
}

func classifierAccuracyCapability(report Report, accuracy []runsignal.DetectorAccuracy) *commonv1.CapabilityMaturityAssessment {
	level, next, clean := "L2", "", true
	for _, finding := range report.Findings {
		switch finding.Code {
		case "classifier_accuracy.unmeasured":
			level, next, clean = "L0", "L1", false
		case "classifier_accuracy.coverage_missing":
			if level != "L0" {
				level, next, clean = "L1", "L2", false
			}
		case "classifier_accuracy.below_threshold":
			if clean {
				level, next, clean = "L1", "L2", false
			}
		}
	}
	return &commonv1.CapabilityMaturityAssessment{
		Id:             "classifier_accuracy",
		Label:          "Classifier Accuracy",
		CurrentLevel:   level,
		NextLevel:      next,
		CurrentSummary: accuracySummary(accuracy),
		NextUnlock:     "Meet every committed detector precision and recall threshold.",
		Clean:          clean,
		PriorityRank:   1,
		Levels: []*commonv1.LocalMaturityLevel{
			{Id: "L0", Name: "Unmeasured", Description: "The labelled corpus or scorer cannot produce a complete result."},
			{Id: "L1", Name: "Measured", Description: "All shipped detectors have published precision and recall."},
			{Id: "L2", Name: "Threshold met", Description: "Every detector meets its committed precision and recall threshold."},
		},
	}
}

func accuracySummary(accuracy []runsignal.DetectorAccuracy) string {
	if len(accuracy) == 0 {
		return "No classifier accuracy rows were published."
	}
	items := make([]string, 0, len(accuracy))
	for _, result := range accuracy {
		items = append(items, fmt.Sprintf("%s precision=%.2f recall=%.2f threshold=%.2f", result.ID, result.Precision, result.Recall, result.Threshold))
	}
	return strings.Join(items, "; ")
}

func maturityFor(finding Finding) (string, commonv1.CleanRequirement) {
	switch finding.Code {
	case "classifier_accuracy.unmeasured":
		return "L0", commonv1.CleanRequirement_CLEAN_REQUIREMENT_REQUIRED
	case "classifier_accuracy.coverage_missing":
		return "L1", commonv1.CleanRequirement_CLEAN_REQUIREMENT_REQUIRED
	case "classifier_accuracy.below_threshold":
		return "L2", commonv1.CleanRequirement_CLEAN_REQUIREMENT_REQUIRED
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

func capabilityFor(finding Finding) string {
	if strings.HasPrefix(finding.Code, "classifier_accuracy.") {
		return "classifier_accuracy"
	}
	return "portable_profiles"
}

func dimensionFor(finding Finding) string {
	if strings.HasPrefix(finding.Code, "classifier_accuracy.") {
		return "tests"
	}
	return "contracts"
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
