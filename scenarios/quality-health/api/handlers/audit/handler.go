package audit

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"connectrpc.com/connect"

	internalaudit "quality-health/internal/audit"
	"quality-health/internal/autofix"
	"quality-health/internal/commands"
	"quality-health/internal/contracts"
	"quality-health/internal/surfaces"

	"github.com/vrooli/maturity-go/assessment"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/quality-health/v1/audit"
	auditconnect "github.com/vrooli/vrooli/packages/proto/gen/go/quality-health/v1/audit/audit_v1connect"
)

type Deps struct {
	Service      *internalaudit.Service
	Logger       *log.Logger
	MaturitySpec *assessment.Spec
}

type Handler struct {
	auditconnect.UnimplementedAuditServiceHandler
	svc    *internalaudit.Service
	logger *log.Logger
	spec   *assessment.Spec
}

func NewHandler(svc *internalaudit.Service, logger *log.Logger) *Handler {
	return NewHandlerWithDeps(Deps{Service: svc, Logger: logger})
}

func NewHandlerWithDeps(deps Deps) *Handler {
	if deps.Logger == nil {
		deps.Logger = log.Default()
	}
	if deps.Service == nil {
		deps.Service = internalaudit.New(nil)
	}
	return &Handler{svc: deps.Service, logger: deps.Logger, spec: deps.MaturitySpec}
}

var _ auditconnect.AuditServiceHandler = (*Handler)(nil)

func (h *Handler) AuditQuality(ctx context.Context, req *connect.Request[auditv1.AuditQualityRequest]) (*connect.Response[auditv1.AuditQualityResponse], error) {
	if req.Msg.GetScenario() == "" && req.Msg.GetPath() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario or path is required"))
	}
	report, err := h.svc.Audit(ctx, internalaudit.Request{
		Scenario:                req.Msg.GetScenario(),
		Path:                    req.Msg.GetPath(),
		RuleIDs:                 req.Msg.GetRuleIds(),
		Surfaces:                req.Msg.GetSurfaces(),
		IncludeCommandExecution: req.Msg.GetIncludeCommandExecution(),
		IncludeAutofixPreview:   req.Msg.GetIncludeAutofixPreview(),
		UseCache:                req.Msg.GetUseCache(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	resp, err := ResponseToProto(report, h.spec)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build maturity assessment: %w", err))
	}
	return connect.NewResponse(resp), nil
}

func (h *Handler) ListContracts(_ context.Context, req *connect.Request[auditv1.ListContractsRequest]) (*connect.Response[auditv1.ListContractsResponse], error) {
	items := contracts.List(req.Msg.GetLanguage(), req.Msg.GetFramework(), req.Msg.GetSurfaceKind(), req.Msg.GetRuleIds())
	out := &auditv1.ListContractsResponse{}
	for _, item := range items {
		out.Contracts = append(out.Contracts, contractToProto(item))
	}
	return connect.NewResponse(out), nil
}

func (h *Handler) ExplainFinding(_ context.Context, req *connect.Request[auditv1.ExplainFindingRequest]) (*connect.Response[auditv1.ExplainFindingResponse], error) {
	ruleID := strings.TrimSpace(req.Msg.GetRuleId())
	if ruleID == "" {
		ruleID = strings.TrimSpace(req.Msg.GetFindingId())
	}
	if ruleID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("finding_id or rule_id is required"))
	}
	contract, ok := contracts.ByRule(ruleID)
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("unknown Quality Health rule %q", ruleID))
	}
	return connect.NewResponse(&auditv1.ExplainFindingResponse{
		Contract:     contractToProto(contract),
		WhyItMatters: contract.WhyItMatters,
		Remediation:  contract.Remediation,
		NextSteps: []string{
			fmt.Sprintf("Run `quality-health audit run %s --rule %s --json` to refresh current evidence.", req.Msg.GetScenario(), ruleID),
			fmt.Sprintf("Run `quality-health fix-config run %s --rule %s --dry-run` if the rule is autofixable.", req.Msg.GetScenario(), ruleID),
		},
	}), nil
}

func (h *Handler) PreviewFixConfig(ctx context.Context, req *connect.Request[auditv1.FixConfigRequest]) (*connect.Response[auditv1.FixConfigResponse], error) {
	inv, candidates, err := h.svc.PreviewFix(ctx, req.Msg.GetScenario(), req.Msg.GetPath(), req.Msg.GetRuleIds())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(fixResponseToProto(inv, false, candidates)), nil
}

func (h *Handler) ApplyFixConfig(ctx context.Context, req *connect.Request[auditv1.FixConfigRequest]) (*connect.Response[auditv1.FixConfigResponse], error) {
	inv, candidates, err := h.svc.ApplyFix(ctx, req.Msg.GetScenario(), req.Msg.GetPath(), req.Msg.GetRuleIds())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(fixResponseToProto(inv, true, candidates)), nil
}

func ResponseToProto(in internalaudit.Response, spec *assessment.Spec) (*auditv1.AuditQualityResponse, error) {
	errors, warnings, infos := findingCounts(in.Findings)
	assessment, err := buildMaturityAssessment(in, spec)
	if err != nil {
		return nil, err
	}
	out := &auditv1.AuditQualityResponse{
		RunId:          in.RunID,
		Status:         in.Status,
		Summary:        in.Summary,
		Scenario:       in.Inventory.Scenario,
		TargetKind:     in.Inventory.TargetKind,
		TargetPath:     in.Inventory.RootPath,
		DegradedReason: in.Inventory.DegradedReason,
		Maturity: &auditv1.MaturitySummary{
			Rung:      int32(in.Maturity.Rung),
			Label:     in.Maturity.Label,
			Rationale: in.Maturity.Rationale,
		},
		Counts: &auditv1.AuditSummary{
			Errors:           int32(errors),
			Warnings:         int32(warnings),
			Infos:            int32(infos),
			Surfaces:         int32(len(in.Inventory.Surfaces)),
			Contracts:        int32(len(in.Contracts)),
			AutofixableCount: int32(autofixableCount(in.Findings)),
		},
		NextSteps:  in.NextSteps,
		Assessment: assessment,
	}
	for _, s := range in.Inventory.Surfaces {
		out.Surfaces = append(out.Surfaces, surfaceToProto(s))
	}
	for _, c := range in.Contracts {
		out.Contracts = append(out.Contracts, &auditv1.ContractEvaluation{ContractId: c.ContractID, SurfaceId: c.SurfaceID, Status: c.Status, RuleIds: c.RuleIDs})
	}
	for _, f := range in.Findings {
		out.Findings = append(out.Findings, findingToProto(f))
	}
	for _, r := range in.CommandResults {
		out.CommandResults = append(out.CommandResults, commandToProto(r))
	}
	for _, c := range in.AutofixCandidates {
		out.AutofixCandidates = append(out.AutofixCandidates, candidateToProto(c))
	}
	return out, nil
}

func buildMaturityAssessment(in internalaudit.Response, spec *assessment.Spec) (*commonv1.MaturityAssessment, error) {
	if spec == nil {
		return nil, fmt.Errorf("maturity spec is required")
	}
	findings := make([]assessment.Finding, 0, len(in.Findings))
	for _, f := range in.Findings {
		findings = append(findings, assessment.Finding{
			Code:        f.RuleID,
			Severity:    severityToAssessment(f.Severity),
			Title:       f.RuleID,
			Message:     f.Message,
			Location:    f.FilePath,
			Remediation: f.Remediation,
			Source:      architecturev1.FindingSource_FINDING_SOURCE_STANDARDS,
			Phase:       spec.Phase,
		})
	}
	return assessment.BuildProtoAssessment(assessment.BuildInput{
		Scenario: in.Inventory.Scenario,
		Spec:     *spec,
		Findings: findings,
	})
}

func severityToAssessment(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "error":
		return architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR.String()
	case "warning", "warn":
		return architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING.String()
	case "info":
		return architecturev1.FindingSeverity_FINDING_SEVERITY_INFO.String()
	default:
		return severity
	}
}

func surfaceToProto(in surfaces.Surface) *auditv1.QualitySurface {
	return &auditv1.QualitySurface{Id: in.ID, Kind: in.Kind, Language: in.Language, Framework: in.Framework, RootPath: in.RootPath, PackageManager: in.PackageManager, Status: in.Status, Confidence: in.Confidence}
}

func contractToProto(in contracts.Contract) *auditv1.QualityContract {
	return &auditv1.QualityContract{Id: in.ID, Title: in.Title, Category: in.Category, Severity: in.Severity, Language: in.Language, Framework: in.Framework, SurfaceKind: in.SurfaceKind, RuleIds: in.RuleIDs, Description: in.Description, WhyItMatters: in.WhyItMatters, Remediation: in.Remediation, AutofixAvailable: in.AutofixAvailable, FixClass: in.FixClass}
}

func findingToProto(in internalaudit.Finding) *auditv1.QualityFinding {
	return &auditv1.QualityFinding{Id: in.ID, Scenario: in.Scenario, TargetKind: in.TargetKind, SurfaceId: in.SurfaceID, SurfaceKind: in.SurfaceKind, Language: in.Language, Framework: in.Framework, RuleId: in.RuleID, Category: in.Category, Severity: in.Severity, FilePath: in.FilePath, Symbol: in.Symbol, Message: in.Message, Evidence: in.Evidence, Expected: in.Expected, Observed: in.Observed, WhyItMatters: in.WhyItMatters, Remediation: in.Remediation, AutofixAvailable: in.AutofixAvailable, AutofixCommand: in.AutofixCommand, SourceCommand: in.SourceCommand, CreatedAt: in.CreatedAt, FixClass: in.FixClass}
}

func commandToProto(in commands.Result) *auditv1.CommandResult {
	return &auditv1.CommandResult{Name: in.Name, Command: in.Command, WorkingDirectory: in.WorkingDirectory, Status: in.Status, ExitCode: int32(in.ExitCode), StdoutExcerpt: in.StdoutExcerpt, StderrExcerpt: in.StderrExcerpt, TimeoutSeconds: int32(in.TimeoutSeconds), FailureReason: in.FailureReason}
}

func candidateToProto(in autofix.Candidate) *auditv1.AutofixCandidate {
	return &auditv1.AutofixCandidate{RuleId: in.RuleID, FilePath: in.FilePath, Description: in.Description, Before: in.Before, After: in.After, Applied: in.Applied}
}

func fixResponseToProto(inv surfaces.Inventory, applied bool, candidates []autofix.Candidate) *auditv1.FixConfigResponse {
	out := &auditv1.FixConfigResponse{Scenario: inv.Scenario, Applied: applied}
	for _, candidate := range candidates {
		out.Candidates = append(out.Candidates, candidateToProto(candidate))
	}
	if len(candidates) == 0 {
		out.Messages = append(out.Messages, "No supported config fixes are available.")
	}
	return out
}

func findingCounts(findings []internalaudit.Finding) (errors, warnings, infos int) {
	for _, f := range findings {
		switch f.Severity {
		case "error":
			errors++
		case "warning":
			warnings++
		default:
			infos++
		}
	}
	return errors, warnings, infos
}

func autofixableCount(findings []internalaudit.Finding) int {
	count := 0
	for _, f := range findings {
		if f.AutofixAvailable {
			count++
		}
	}
	return count
}
