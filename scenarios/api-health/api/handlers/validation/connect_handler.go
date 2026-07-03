package validation

import (
	"context"
	"errors"
	"fmt"
	"log"

	"connectrpc.com/connect"

	internalvalidation "api-health/internal/validation"

	"github.com/vrooli/api-core/metrics"
	"github.com/vrooli/maturity-go/assessment"
	"github.com/vrooli/maturity-go/autofix"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

type Validator interface {
	ValidateScenario(ctx context.Context, scenario, explicitPath string, includeExecution bool) (internalvalidation.Report, error)
}

type Fixer interface {
	Preview(root string, ruleIDs []string) ([]autofix.Candidate, error)
	Apply(root string, ruleIDs []string) ([]autofix.Candidate, error)
}

type Deps struct {
	Logger       *log.Logger
	Validator    Validator
	Fixers       Fixer
	MaturitySpec *assessment.Spec
	Environment  *commonv1.CaptureEnvironment
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	if h.deps.Validator == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("validation.ValidateScenario: validator not wired"))
	}
	collector := metrics.Start(metrics.WithEnvironment(h.deps.Environment))
	report, err := h.deps.Validator.ValidateScenario(internalvalidation.WithMetrics(ctx, collector), req.Msg.GetScenario(), req.Msg.GetPath(), req.Msg.GetIncludeExecution())
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	maturityAssessment, err := buildMaturityAssessment(report, h.deps.MaturitySpec)
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build maturity assessment: %w", err))
	}
	nativeDetail, err := buildNativeDetail(report)
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build native detail: %w", err))
	}
	resp, err := assessment.BuildValidationResponse(report.Scenario, maturityAssessment, nativeDetail, collector.Stop())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build shared validation response: %w", err))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) PreviewFix(ctx context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return h.fix(ctx, req.Msg, false)
}

func (h *connectHandler) ApplyFix(ctx context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return h.fix(ctx, req.Msg, true)
}

func (h *connectHandler) fix(ctx context.Context, req *scenariovalidationv1.FixRequest, apply bool) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	if h.deps.Validator == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("validation fix: validator not wired"))
	}
	if h.deps.Fixers == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("validation fix: fixer registry not wired"))
	}
	report, err := h.deps.Validator.ValidateScenario(ctx, req.GetScenario(), req.GetPath(), false)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	root := report.Target.RootPath
	if root == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("target scenario could not be resolved"))
	}
	var candidates []autofix.Candidate
	if apply {
		candidates, err = h.deps.Fixers.Apply(root, req.GetRuleIds())
	} else {
		candidates, err = h.deps.Fixers.Preview(root, req.GetRuleIds())
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	resp := buildFixResponse(report.Scenario, apply, candidates)
	if len(resp.Candidates) == 0 {
		resp.Messages = append(resp.Messages, "no deterministic API Health fixes available")
	}
	return connect.NewResponse(resp), nil
}

func buildFixResponse(scenario string, applied bool, candidates []autofix.Candidate) *scenariovalidationv1.FixResponse {
	resp := &scenariovalidationv1.FixResponse{Scenario: scenario, Applied: applied}
	for _, c := range candidates {
		resp.Candidates = append(resp.Candidates, &scenariovalidationv1.FixCandidate{
			RuleId:      c.RuleID,
			FilePath:    c.FilePath,
			Description: c.Description,
			Before:      c.Before,
			After:       c.After,
			Applied:     applied,
		})
	}
	return resp
}

func buildMaturityAssessment(rep internalvalidation.Report, spec *assessment.Spec) (*commonv1.MaturityAssessment, error) {
	if spec == nil {
		return nil, errors.New("maturity spec is required")
	}
	findings := make([]assessment.Finding, 0, len(rep.Findings))
	for _, f := range rep.Findings {
		findings = append(findings, assessment.Finding{
			Code:        f.Code,
			Severity:    severityToken(f.Severity),
			Title:       f.Title,
			Message:     f.Message,
			Location:    f.Location,
			Remediation: f.Remediation,
			Phase:       spec.Phase,
		})
	}
	return assessment.BuildProtoAssessment(assessment.BuildInput{
		Scenario: rep.Scenario,
		Spec:     *spec,
		Findings: findings,
	})
}

func severityToken(s internalvalidation.Severity) string {
	switch s {
	case internalvalidation.SeverityError:
		return "SEVERITY_ERROR"
	case internalvalidation.SeverityWarn:
		return "SEVERITY_WARNING"
	case internalvalidation.SeverityInfo:
		return "SEVERITY_INFO"
	default:
		return "SEVERITY_UNSPECIFIED"
	}
}

func buildNativeDetail(rep internalvalidation.Report) (*structpb.Struct, error) {
	return structpb.NewStruct(map[string]any{
		"scenario": rep.Scenario,
		"target": map[string]any{
			"scenario":                  rep.Target.Scenario,
			"root_path":                 rep.Target.RootPath,
			"resolution":                string(rep.Target.Resolution),
			"api_kind":                  string(rep.Target.APIKind),
			"api_dir":                   rep.Target.APIDir,
			"has_api_dir":               rep.Target.HasAPIDir,
			"service_manifest_path":     rep.Target.ServiceManifestPath,
			"service_manifest_readable": rep.Target.ServiceManifestReadable,
			"service_ports_api":         rep.Target.Service.PortsAPI,
			"service_health_api_path":   rep.Target.Service.HealthAPIPath,
			"service_health_api_check":  rep.Target.Service.HealthAPICheck,
			"service_health_check_url":  rep.Target.Service.HealthAPICheckURL,
			"lifecycle": map[string]any{
				"main_path":               rep.Target.Lifecycle.MainPath,
				"main_readable":           rep.Target.Lifecycle.MainReadable,
				"manifest_healthy":        rep.Target.Lifecycle.ManifestHealthy,
				"preflight_healthy":       rep.Target.Lifecycle.PreflightHealthy,
				"server_runner_healthy":   rep.Target.Lifecycle.ServerRunnerHealthy,
				"direct_listen_and_serve": rep.Target.Lifecycle.DirectListenAndServe,
				"diagnostics":             stringSlice(rep.Target.Lifecycle.Diagnostics),
			},
			"health_probe": map[string]any{
				"requested":         rep.Target.Health.Requested,
				"url":               rep.Target.Health.URL,
				"status_code":       rep.Target.Health.StatusCode,
				"content_type":      rep.Target.Health.ContentType,
				"elapsed_millis":    rep.Target.Health.ElapsedMillis,
				"failure_class":     rep.Target.Health.FailureClass,
				"error":             rep.Target.Health.Error,
				"schema_valid":      rep.Target.Health.SchemaValid,
				"schema_violations": stringSlice(rep.Target.Health.SchemaViolations),
				"payload":           healthPayloadDetail(rep.Target.Health.Payload),
			},
			"http_semantics": map[string]any{
				"inspected_files":   stringSlice(rep.Target.HTTP.InspectedFiles),
				"routes":            routesDetail(rep.Target.HTTP.Routes),
				"response_patterns": responsePatternsDetail(rep.Target.HTTP.ResponsePatterns),
				"diagnostics":       stringSlice(rep.Target.HTTP.Diagnostics),
			},
			"runtime_hygiene": map[string]any{
				"inspected_files": stringSlice(rep.Target.Runtime.InspectedFiles),
				"signals":         runtimeSignalsDetail(rep.Target.Runtime.Signals),
				"diagnostics":     stringSlice(rep.Target.Runtime.Diagnostics),
			},
			"diagnostics": stringSlice(rep.Target.Diagnostics),
		},
		"summary": map[string]any{
			"errors":   rep.Summary.Errors,
			"warnings": rep.Summary.Warnings,
			"infos":    rep.Summary.Infos,
			"passed":   rep.Passed,
		},
	})
}

func runtimeSignalsDetail(signals []internalvalidation.RuntimeSignal) []any {
	out := make([]any, 0, len(signals))
	for _, signal := range signals {
		out = append(out, map[string]any{
			"kind":   signal.Kind,
			"source": signal.Source,
			"detail": signal.Detail,
		})
	}
	return out
}

func routesDetail(routes []internalvalidation.HTTPRoute) []any {
	out := make([]any, 0, len(routes))
	for _, route := range routes {
		out = append(out, map[string]any{
			"path":      route.Path,
			"method":    route.Method,
			"class":     route.Class,
			"source":    route.Source,
			"versioned": route.Versioned,
			"exempt":    route.Exempt,
		})
	}
	return out
}

func responsePatternsDetail(patterns []internalvalidation.HTTPResponsePattern) []any {
	out := make([]any, 0, len(patterns))
	for _, pattern := range patterns {
		out = append(out, map[string]any{
			"kind":           pattern.Kind,
			"content_type":   pattern.ContentType,
			"source":         pattern.Source,
			"header_present": pattern.HeaderPresent,
		})
	}
	return out
}

func healthPayloadDetail(payload *internalvalidation.HealthPayload) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	return map[string]any{
		"status":           payload.Status,
		"service":          payload.Service,
		"timestamp":        payload.Timestamp,
		"readiness":        payload.Readiness,
		"version":          payload.Version,
		"dependency_count": payload.DependencyCount,
	}
}

func stringSlice(in []string) []any {
	out := make([]any, 0, len(in))
	for _, s := range in {
		out = append(out, s)
	}
	return out
}
