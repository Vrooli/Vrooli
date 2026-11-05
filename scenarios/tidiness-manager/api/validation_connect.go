package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"

	"github.com/vrooli/api-core/metrics"
	"github.com/vrooli/maturity-go/assessment"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tidiness-manager/v1/validation"
	"google.golang.org/protobuf/types/known/structpb"
)

type scenarioValidationHandler struct {
	server *Server
}

func newScenarioValidationHandler(server *Server) *scenarioValidationHandler {
	return &scenarioValidationHandler{server: server}
}

func mountScenarioValidation(router *mux.Router, handler assessment.ValidationServer, describer assessment.Describer) {
	path, connectHandler := scenariovalidationconnect.NewScenarioValidationServiceHandler(assessment.Serve(handler, describer))
	router.PathPrefix(path).Handler(connectHandler)
}

func (h *scenarioValidationHandler) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	if h == nil || h.server == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("tidiness validation handler not wired"))
	}
	scenarioName, scenarioPath, err := h.server.resolveValidationTarget(req.Msg.GetScenario(), req.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return h.validate(ctx, scenarioName, scenarioPath, req.Msg.GetIncludeExecution(), nil)
}

func (h *scenarioValidationHandler) ValidateTarget(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateTargetRequest]) (*connect.Response[scenariovalidationv1.ValidateTargetResponse], error) {
	if h == nil || h.server == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("tidiness validation handler not wired"))
	}
	target := req.Msg.GetTarget()
	if target == nil || target.GetKind() == commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_UNSPECIFIED || strings.TrimSpace(target.GetId()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("validation target kind and id are required"))
	}
	scenarioPath := strings.TrimSpace(req.Msg.GetPath())
	if scenarioPath == "" {
		if strings.TrimSpace(target.GetRoot()) == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("validation target path is required"))
		}
		scenarioPath = target.GetRoot()
		if !filepath.IsAbs(scenarioPath) {
			scenarioPath = filepath.Join(h.server.scenarioLocator.repoRoot, filepath.FromSlash(scenarioPath))
		}
	}
	excludes := normalizeTargetExcludes(target.GetRoot(), req.Msg.GetExclude())
	legacy, err := h.validate(ctx, target.GetId(), scenarioPath, req.Msg.GetIncludeExecution(), excludes)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&scenariovalidationv1.ValidateTargetResponse{
		Target:       target,
		Status:       legacy.Msg.GetStatus(),
		Assessment:   legacy.Msg.GetAssessment(),
		NativeDetail: legacy.Msg.GetNativeDetail(),
		Metrics:      legacy.Msg.GetMetrics(),
	}), nil
}

func (h *scenarioValidationHandler) validate(ctx context.Context, scenarioName, scenarioPath string, includeExecution bool, excludes []string) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	collector := metrics.Start(metrics.WithEnvironment(h.server.environment))
	result, audit, err := buildTidinessScanWithAudit(WithMetrics(ctx, collector), scenarioName, scenarioPath, validationTimeout(includeExecution), excludes...)
	if err != nil {
		collector.Stop()
		h.server.log("tidiness validation failed", map[string]interface{}{
			"error":    err.Error(),
			"scenario": scenarioName,
		})
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("tidiness scan failed: %w", err))
	}
	h.server.storeBudgetAudit(scenarioPath, audit)
	if h.server.store != nil {
		if _, err := h.server.store.ResolveLegacyPercentageDuplicationIssues(ctx, scenarioName); err != nil {
			h.server.log("failed to resolve legacy percentage duplication issues", map[string]interface{}{
				"scenario": scenarioName,
				"error":    err.Error(),
			})
		}
	}
	native, err := tidinessScanToProto(result)
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build tidiness native detail: %w", err))
	}
	execMetrics := collector.Stop()
	resp, err := assessment.BuildValidationResponse(native.GetScenario(), native.GetAssessment(), native, execMetrics)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build shared validation response: %w", err))
	}
	return connect.NewResponse(resp), nil
}

func normalizeTargetExcludes(targetRoot string, excludes []string) []string {
	root := strings.Trim(strings.ReplaceAll(filepath.ToSlash(strings.TrimSpace(targetRoot)), "./", ""), "/")
	if root == "." {
		root = ""
	}
	out := make([]string, 0, len(excludes))
	for _, raw := range excludes {
		pattern := strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(raw)), "./")
		if pattern == "" {
			continue
		}
		if root != "" {
			if pattern == root {
				out = append(out, "**")
				continue
			}
			prefix := root + "/"
			if strings.HasPrefix(pattern, prefix) {
				pattern = strings.TrimPrefix(pattern, prefix)
			}
		}
		out = append(out, pattern)
	}
	return out
}

func (s *Server) resolveValidationTarget(scenario, path string) (string, string, error) {
	scenario = strings.TrimSpace(scenario)
	path = strings.TrimSpace(path)
	if scenario == "" && path == "" {
		return "", "", fmt.Errorf("scenario or path is required")
	}
	if path != "" {
		if scenario == "" {
			scenario = filepath.Base(filepath.Clean(path))
		}
		return scenario, path, nil
	}
	if s.scenarioLocator == nil {
		return "", "", fmt.Errorf("scenario locator not initialized")
	}
	scenarioName, err := s.scenarioLocator.ValidateScenarioName(scenario)
	if err != nil {
		return "", "", err
	}
	scenarioPath, err := s.scenarioLocator.ScenarioPath(scenarioName)
	if err != nil {
		return "", "", err
	}
	return scenarioName, scenarioPath, nil
}

func validationTimeout(includeExecution bool) time.Duration {
	if includeExecution {
		return 10 * time.Minute
	}
	return 120 * time.Second
}

func tidinessScanToProto(result *TidinessScanResponse) (*validationv1.TidinessScanResponse, error) {
	if result == nil {
		return nil, fmt.Errorf("tidiness scan result is nil")
	}
	findings, err := tidinessFindingsToProto(result.Findings)
	if err != nil {
		return nil, err
	}
	violations, err := tidinessFindingsToProto(result.Violations)
	if err != nil {
		return nil, err
	}
	return &validationv1.TidinessScanResponse{
		Scenario:   result.Scenario,
		Status:     result.Status,
		Findings:   findings,
		Violations: violations,
		Summary: &validationv1.TidinessScanSummary{
			TotalFindings:       int32(result.Summary.TotalFindings),
			LongFiles:           int32(result.Summary.LongFiles),
			Complexity:          int32(result.Summary.Complexity),
			Duplication:         int32(result.Summary.Duplication),
			TechDebt:            int32(result.Summary.TechDebt),
			Coupling:            int32(result.Summary.Coupling),
			DuplicationLineDebt: int32(result.Summary.DuplicationLineDebt),
		},
		Assessment: result.Assessment,
		SeamFiles:  result.SeamFiles,
	}, nil
}

func tidinessFindingsToProto(findings []TidinessFinding) ([]*validationv1.TidinessFinding, error) {
	out := make([]*validationv1.TidinessFinding, 0, len(findings))
	for _, finding := range findings {
		evidence, err := tidinessEvidenceToStruct(finding.Evidence)
		if err != nil {
			return nil, fmt.Errorf("convert evidence for %s: %w", finding.RuleID, err)
		}
		out = append(out, &validationv1.TidinessFinding{
			Id:                     finding.ID,
			RuleId:                 finding.RuleID,
			Scenario:               finding.Scenario,
			FilePath:               finding.FilePath,
			Symbol:                 finding.Symbol,
			LineNumber:             int32(finding.LineNumber),
			Category:               finding.Category,
			Severity:               finding.Severity,
			Title:                  finding.Title,
			Description:            finding.Description,
			Evidence:               evidence,
			WhyItMatters:           finding.WhyItMatters,
			RecommendedRemediation: finding.RecommendedRemediation,
			Remediation:            finding.Remediation,
			CampaignGroupHint:      finding.CampaignGroupHint,
		})
	}
	return out, nil
}

func tidinessEvidenceToStruct(evidence map[string]any) (*structpb.Struct, error) {
	if len(evidence) == 0 {
		return &structpb.Struct{}, nil
	}
	payload, err := json.Marshal(evidence)
	if err != nil {
		return nil, err
	}
	var normalized map[string]any
	if err := json.Unmarshal(payload, &normalized); err != nil {
		return nil, err
	}
	return structpb.NewStruct(normalized)
}

// validationDescriber resolves this provider's own identity for the readiness
// contract. It reads only tidiness-manager's descriptor, so a readiness probe
// no longer costs a full scan of the target scenario. A failure yields the zero
// Describer, which reports Unimplemented and makes consumers fall back to the
// legacy probe.
func (s *Server) validationDescriber() assessment.Describer {
	if s == nil || s.scenarioLocator == nil {
		return assessment.Describer{}
	}
	scenarioDir, err := s.scenarioLocator.ScenarioPath("tidiness-manager")
	if err != nil || strings.TrimSpace(scenarioDir) == "" {
		return assessment.Describer{}
	}
	describer, _ := assessment.LoadDescriber(scenarioDir)
	return describer
}
