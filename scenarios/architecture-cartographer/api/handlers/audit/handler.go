// Package audit is the Connect-RPC surface for the audit domain.
package audit

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"architecture-cartographer/internal/audit"
	"architecture-cartographer/internal/conflicts"

	"connectrpc.com/connect"
	"github.com/vrooli/maturity-go/assessment"
	repocontract "github.com/vrooli/repo-contract-go"
	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/audit"
	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/audit/audit_v1connect"
	conflictsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/conflicts"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"google.golang.org/protobuf/types/known/durationpb"
)

// Handler implements audit_v1connect.AuditServiceHandler.
type Handler struct {
	audit_v1connect.UnimplementedAuditServiceHandler
	svc audit.Service
}

// NewHandler constructs the Connect handler.
func NewHandler(svc audit.Service) *Handler { return &Handler{svc: svc} }

var _ audit_v1connect.AuditServiceHandler = (*Handler)(nil)
var _ scenariovalidationconnect.ScenarioValidationServiceHandler = (*Handler)(nil)

func (h *Handler) RunAll(ctx context.Context, req *connect.Request[auditv1.AuditRunAllRequest]) (*connect.Response[auditv1.AuditRunAllResponse], error) {
	sweep, err := h.svc.RunAll(ctx, audit.RunAllInput{
		FailOn:                     severityFromProto(req.Msg.GetFailOn()),
		IncludeTypes:               req.Msg.GetIncludeTypes(),
		ExcludeTypes:               req.Msg.GetExcludeTypes(),
		IncludeScenarios:           req.Msg.GetIncludeScenarios(),
		ExcludeScenarios:           req.Msg.GetExcludeScenarios(),
		AllowLowAuthority:          req.Msg.GetAllowLowAuthority(),
		AllowLowAuthorityScenarios: req.Msg.GetAllowLowAuthorityScenarios(),
		Concurrency:                int(req.Msg.GetConcurrency()),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &auditv1.AuditRunAllResponse{
		TotalScenarios:  int32(sweep.TotalScenarios),
		TotalFindings:   int32(sweep.TotalFindings),
		TotalSuppressed: int32(sweep.TotalSuppressed),
		BySeverity:      intMap(sweep.BySeverity),
		ByOutcome:       intMap(sweep.ByOutcome),
		Duration:        durationpb.New(sweep.Duration),
	}
	for _, r := range sweep.Reports {
		out.Reports = append(out.Reports, reportToProto(r))
	}
	return connect.NewResponse(out), nil
}

func (h *Handler) Run(ctx context.Context, req *connect.Request[auditv1.AuditRunRequest]) (*connect.Response[auditv1.AuditRunResponse], error) {
	if req.Msg.GetScenario() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	rep, err := h.svc.Run(ctx, audit.RunInput{
		Scenario:          req.Msg.GetScenario(),
		FailOn:            severityFromProto(req.Msg.GetFailOn()),
		IncludeTypes:      req.Msg.GetIncludeTypes(),
		ExcludeTypes:      req.Msg.GetExcludeTypes(),
		AllowLowAuthority: req.Msg.GetAllowLowAuthority(),
		SkipTS:            req.Msg.GetSkipTs(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(reportToProto(rep)), nil
}

func (h *Handler) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	msg := req.Msg
	if msg == nil {
		msg = &scenariovalidationv1.ValidateScenarioRequest{}
	}
	scenario := strings.TrimSpace(msg.GetScenario())
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	native, err := h.Run(ctx, connect.NewRequest(&auditv1.AuditRunRequest{
		Scenario:          scenario,
		AllowLowAuthority: true,
	}))
	if err != nil {
		return nil, err
	}
	status := scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_UNSPECIFIED
	switch native.Msg.GetOutcome() {
	case auditv1.AuditOutcome_AUDIT_OUTCOME_TOOL_ERROR:
		status = scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_ERROR
	case auditv1.AuditOutcome_AUDIT_OUTCOME_FINDINGS, auditv1.AuditOutcome_AUDIT_OUTCOME_PARTIAL:
		status = scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_DEGRADED
	case auditv1.AuditOutcome_AUDIT_OUTCOME_CLEAN:
		status = scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED
	}
	resp, err := assessment.BuildValidationResponse(native.Msg.GetScenario(), native.Msg.GetAssessment(), native.Msg, assessment.WithValidationStatus(status))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build shared validation response: %w", err))
	}
	return connect.NewResponse(resp), nil
}

func severityFromProto(s conflictsv1.Severity) conflicts.Severity {
	switch s {
	case conflictsv1.Severity_SEVERITY_INFO:
		return conflicts.SeverityInfo
	case conflictsv1.Severity_SEVERITY_WARN:
		return conflicts.SeverityWarn
	case conflictsv1.Severity_SEVERITY_ERROR:
		return conflicts.SeverityError
	case conflictsv1.Severity_SEVERITY_BLOCKER:
		return conflicts.SeverityBlocker
	default:
		return conflicts.SeverityUnspecified
	}
}

func severityToProto(s conflicts.Severity) conflictsv1.Severity {
	switch s {
	case conflicts.SeverityInfo:
		return conflictsv1.Severity_SEVERITY_INFO
	case conflicts.SeverityWarn:
		return conflictsv1.Severity_SEVERITY_WARN
	case conflicts.SeverityError:
		return conflictsv1.Severity_SEVERITY_ERROR
	case conflicts.SeverityBlocker:
		return conflictsv1.Severity_SEVERITY_BLOCKER
	default:
		return conflictsv1.Severity_SEVERITY_UNSPECIFIED
	}
}

func outcomeToProto(o audit.Outcome) auditv1.AuditOutcome {
	switch o {
	case audit.OutcomeClean:
		return auditv1.AuditOutcome_AUDIT_OUTCOME_CLEAN
	case audit.OutcomeFindings:
		return auditv1.AuditOutcome_AUDIT_OUTCOME_FINDINGS
	case audit.OutcomeToolError:
		return auditv1.AuditOutcome_AUDIT_OUTCOME_TOOL_ERROR
	case audit.OutcomePartial:
		return auditv1.AuditOutcome_AUDIT_OUTCOME_PARTIAL
	default:
		return auditv1.AuditOutcome_AUDIT_OUTCOME_UNSPECIFIED
	}
}

func reportToProto(r audit.Report) *auditv1.AuditRunResponse {
	out := &auditv1.AuditRunResponse{
		Scenario:            r.Scenario,
		Outcome:             outcomeToProto(r.Outcome),
		OutcomeReason:       r.OutcomeReason,
		Error:               r.Error,
		TotalFindings:       int32(r.TotalFindings),
		BySeverity:          intMap(r.BySeverity),
		ByType:              intMap(r.ByType),
		ByDomain:            intMap(r.ByDomain),
		SuppressedFindings:  int32(r.SuppressedFindings),
		SnapshotFreshness:   freshnessToProto(r.SnapshotFreshness),
		AuthorityConfidence: authorityConfidenceToProto(r.Domains.Confidence),
		Duration:            durationpb.New(r.Duration),
		Domains: &auditv1.DerivedDomainSummary{
			Authority:   r.Domains.Authority,
			Confidence:  r.Domains.Confidence,
			DomainCount: int32(r.Domains.DomainCount),
		},
		Graph: &auditv1.GraphSummary{
			SnapshotId:      r.Graph.SnapshotID,
			FileCount:       int32(r.Graph.FileCount),
			PackageCount:    int32(r.Graph.PackageCount),
			ImportEdgeCount: int32(r.Graph.ImportEdgeCount),
		},
	}
	for _, c := range r.Findings {
		out.Findings = append(out.Findings, &auditv1.ConflictSummary{
			Id:         c.ID,
			StableId:   c.StableID,
			InstanceId: c.InstanceID,
			Detector:   c.Detector,
			Type:       c.Type,
			Subtype:    c.Subtype,
			Severity:   severityToProto(c.Severity),
			Locations:  append([]string(nil), c.Locations...),
			Domains:    append([]string(nil), c.Domains...),
			Headline:   headlineFor(c),
		})
	}
	if maturity, err := buildMaturityAssessment(r); err == nil {
		out.Assessment = maturity
	}
	return out
}

func buildMaturityAssessment(r audit.Report) (*commonv1.MaturityAssessment, error) {
	spec, err := loadMaturitySpec()
	if err != nil {
		return nil, err
	}
	findings := make([]assessment.Finding, 0, len(r.Findings)+2)
	if r.Outcome == audit.OutcomeToolError {
		findings = append(findings, assessment.Finding{
			Code:        "graph.extract_failed",
			Severity:    architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR.String(),
			Title:       "Architecture audit tool error",
			Message:     r.Error,
			Remediation: "Run `architecture-cartographer audit run " + r.Scenario + "` and inspect graph extraction or detector failures.",
			Source:      architecturev1.FindingSource_FINDING_SOURCE_ARCHITECTURE,
			Phase:       spec.Phase,
		})
	}
	switch strings.TrimSpace(r.Domains.Confidence) {
	case "missing":
		findings = append(findings, assessment.Finding{
			Code:        "domain_authority/missing",
			Severity:    architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING.String(),
			Title:       "Missing architecture domain authority",
			Message:     r.OutcomeReason,
			Remediation: "Write docs/concepts/DOMAINS.md for the target scenario or rerun in advisory mode when intentionally deferred.",
			Source:      architecturev1.FindingSource_FINDING_SOURCE_ARCHITECTURE,
			Phase:       spec.Phase,
		})
	case "low":
		findings = append(findings, assessment.Finding{
			Code:        "domain_authority/low",
			Severity:    architecturev1.FindingSeverity_FINDING_SEVERITY_INFO.String(),
			Title:       "Low-confidence architecture domain authority",
			Message:     r.OutcomeReason,
			Remediation: "Promote inferred domains to docs/concepts/DOMAINS.md.",
			Source:      architecturev1.FindingSource_FINDING_SOURCE_ARCHITECTURE,
			Phase:       spec.Phase,
		})
	}
	for _, c := range r.Findings {
		findings = append(findings, assessment.Finding{
			Code:        architectureFindingCode(c),
			Severity:    severityToProto(c.Severity).String(),
			Title:       headlineFor(c),
			Location:    strings.Join(c.Locations, ", "),
			Remediation: "Use Architecture Cartographer campaign guidance to resolve the drift or add a deliberate suppression.",
			Source:      architecturev1.FindingSource_FINDING_SOURCE_ARCHITECTURE,
			Phase:       spec.Phase,
		})
	}
	return assessment.BuildProtoAssessment(assessment.BuildInput{
		Scenario: r.Scenario,
		Spec:     *spec,
		Findings: findings,
	})
}

func loadMaturitySpec() (*assessment.Spec, error) {
	_, repoRoot, err := repocontract.LoadDefaultFromEnvOrCWD()
	if err != nil {
		return nil, fmt.Errorf("resolve repo root for architecture maturity spec: %w", err)
	}
	path := filepath.Join(repoRoot, "scenarios", "architecture-cartographer", ".vrooli", "maturity.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read architecture maturity spec: %w", err)
	}
	return assessment.ParseSpec(raw)
}

func architectureFindingCode(c conflicts.Conflict) string {
	code := strings.TrimSpace(c.Type)
	if sub := strings.TrimSpace(c.Subtype); sub != "" {
		code += "/" + sub
	}
	if code == "" {
		return "architecture.unknown"
	}
	return code
}

func freshnessToProto(f audit.SnapshotFreshness) auditv1.SnapshotFreshness {
	switch f {
	case audit.SnapshotFreshnessCached:
		return auditv1.SnapshotFreshness_SNAPSHOT_FRESHNESS_CACHED
	case audit.SnapshotFreshnessReExtracted:
		return auditv1.SnapshotFreshness_SNAPSHOT_FRESHNESS_RE_EXTRACTED
	case audit.SnapshotFreshnessFresh:
		return auditv1.SnapshotFreshness_SNAPSHOT_FRESHNESS_FRESH
	default:
		return auditv1.SnapshotFreshness_SNAPSHOT_FRESHNESS_UNSPECIFIED
	}
}

func authorityConfidenceToProto(s string) auditv1.AuthorityConfidence {
	switch s {
	case "low":
		return auditv1.AuthorityConfidence_AUTHORITY_CONFIDENCE_LOW
	case "medium":
		return auditv1.AuthorityConfidence_AUTHORITY_CONFIDENCE_MEDIUM
	case "high":
		return auditv1.AuthorityConfidence_AUTHORITY_CONFIDENCE_HIGH
	case "missing":
		return auditv1.AuthorityConfidence_AUTHORITY_CONFIDENCE_MISSING
	default:
		return auditv1.AuthorityConfidence_AUTHORITY_CONFIDENCE_UNSPECIFIED
	}
}

func intMap(in map[string]int) map[string]int32 {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]int32, len(in))
	for k, v := range in {
		out[k] = int32(v)
	}
	return out
}

func headlineFor(c conflicts.Conflict) string {
	if len(c.Locations) == 0 {
		return c.Type
	}
	return c.Type + " @ " + c.Locations[0]
}
