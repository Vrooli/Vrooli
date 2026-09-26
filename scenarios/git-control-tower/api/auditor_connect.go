package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"connectrpc.com/connect"

	"git-control-tower/internal/policygate"
	"github.com/vrooli/cli-core/cliutil"
	auditorv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/auditor"
	auditorconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/auditor/auditor_v1connect"
)

const auditorFixOperation = "repo.auditor.fix"

var _ auditorconnect.AuditorServiceHandler = (*auditorConnectServer)(nil)

type auditorConnectServer struct{ server *Server }

func (s auditorConnectServer) StartCheck(ctx context.Context, req *connect.Request[auditorv1.StartCheckRequest]) (*connect.Response[auditorv1.StartCheckResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("start check request is required"))
	}
	scenarioName := strings.TrimSpace(req.Msg.GetScenarioName())
	if scenarioName == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario_name is required"))
	}
	if !s.server.capabilities.IsAvailable(ctx, "scenario-auditor") {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("scenario-auditor is not available"))
	}
	if err := s.server.validateAuditorRepository(ctx, req.Msg.GetRepositoryId()); err != nil {
		return nil, err
	}
	checkType := strings.TrimSpace(req.Msg.GetCheckType())
	if checkType == "" {
		checkType = "full"
	}
	result, err := s.server.auditorClient.StartCheck(ctx, scenarioName, checkType)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&auditorv1.StartCheckResponse{JobId: result.JobID, Status: auditorJobStatusToProto(&result.Status)}), nil
}

func (s auditorConnectServer) GetJobStatus(ctx context.Context, req *connect.Request[auditorv1.GetJobStatusRequest]) (*connect.Response[auditorv1.JobStatus], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("job status request is required"))
	}
	jobID := strings.TrimSpace(req.Msg.GetJobId())
	if jobID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("job_id is required"))
	}
	if !s.server.capabilities.IsAvailable(ctx, "scenario-auditor") {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("scenario-auditor is not available"))
	}
	if err := s.server.validateAuditorRepository(ctx, req.Msg.GetRepositoryId()); err != nil {
		return nil, err
	}
	result, err := s.server.auditorClient.GetJobStatus(ctx, jobID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(auditorJobStatusToProto(result)), nil
}

func (s auditorConnectServer) ListRules(ctx context.Context, req *connect.Request[auditorv1.ListRulesRequest]) (*connect.Response[auditorv1.RulesResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("list rules request is required"))
	}
	if !s.server.capabilities.IsAvailable(ctx, "scenario-auditor") {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("scenario-auditor is not available"))
	}
	if err := s.server.validateAuditorRepository(ctx, req.Msg.GetRepositoryId()); err != nil {
		return nil, err
	}
	result, err := s.server.auditorClient.ListRules(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(auditorRulesResponseToProto(result)), nil
}

func (s auditorConnectServer) ListViolations(ctx context.Context, req *connect.Request[auditorv1.ListViolationsRequest]) (*connect.Response[auditorv1.ViolationsResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("list violations request is required"))
	}
	scenarioName := strings.TrimSpace(req.Msg.GetScenarioName())
	if scenarioName == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario_name is required"))
	}
	if !s.server.capabilities.IsAvailable(ctx, "scenario-auditor") {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("scenario-auditor is not available"))
	}
	if err := s.server.validateAuditorRepository(ctx, req.Msg.GetRepositoryId()); err != nil {
		return nil, err
	}
	result, err := s.server.auditorClient.GetViolations(ctx, scenarioName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(auditorViolationsResponseToProto(result)), nil
}

func (s auditorConnectServer) PreviewFix(ctx context.Context, req *connect.Request[auditorv1.FixRequest]) (*connect.Response[auditorv1.FixResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("fix request is required"))
	}
	fixReq, err := auditorFixRequestFromProto(req.Msg.GetScenarioNames(), req.Msg.GetRuleIds())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if !s.server.capabilities.IsAvailable(ctx, "scenario-auditor") {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("scenario-auditor is not available"))
	}
	result, err := s.server.auditorClient.ApplyFix(ctx, AuditorFixRequest{ScenarioNames: fixReq.ScenarioNames, RuleIDs: fixReq.RuleIDs, DryRun: true})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(auditorFixResponseToProto(result)), nil
}

func (s *Server) validateAuditorRepository(ctx context.Context, repositoryID string) error {
	if _, err := s.resolveRepoForConnect(ctx, strings.TrimSpace(repositoryID), ""); err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	return nil
}

func (s auditorConnectServer) ApplyFix(ctx context.Context, req *connect.Request[auditorv1.ApplyFixRequest]) (*connect.Response[auditorv1.FixResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("fix request is required"))
	}
	fixReq, err := auditorFixRequestFromProto(req.Msg.GetScenarioNames(), req.Msg.GetRuleIds())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if !s.server.capabilities.IsAvailable(ctx, "scenario-auditor") {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("scenario-auditor is not available"))
	}
	principal, ok := policygate.PrincipalFromContext(ctx)
	if !ok || principal.Kind != cliutil.CallerKindHuman {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("verified human authority is required for auditor fixes"))
	}
	resolved, err := s.server.resolveRepoForConnect(ctx, req.Msg.GetRepositoryId(), "")
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	subjectContext, err := auditorFixSubjectContext(fixReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	preview, err := s.server.prepareMutationWithContext(ctx, repositoryIDFor(&RepoRecord{ID: resolved.ID, Path: resolved.Path}), auditorFixOperation, subjectContext)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if preview.ExpectedRevision != strings.TrimSpace(req.Msg.GetExpectedRevision()) || preview.SubjectDigest != strings.TrimSpace(req.Msg.GetSubjectDigest()) {
		return nil, connect.NewError(connect.CodeAborted, errors.New("repository or selected auditor fix changed; review the exact fix preview again"))
	}
	consumed, err := s.server.intentService.Consume(ctx, principal, req.Msg.GetIntentId(), preview.RepositoryID, auditorFixOperation, preview.ExpectedRevision, preview.SubjectDigest)
	if err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	writeCtx := policygate.WithIntent(ctx, consumed.AsHumanIntent())
	if err := requireHumanMutation(writeCtx, "apply auditor fix"); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	result, err := s.server.auditorClient.ApplyFix(writeCtx, AuditorFixRequest{ScenarioNames: fixReq.ScenarioNames, RuleIDs: fixReq.RuleIDs})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(auditorFixResponseToProto(result)), nil
}

func auditorFixRequestFromProto(scenarioNames, ruleIDs []string) (AuditorFixRequest, error) {
	cleanScenarios := cleanAuditorValues(scenarioNames)
	cleanRules := cleanAuditorValues(ruleIDs)
	if len(cleanScenarios) == 0 {
		return AuditorFixRequest{}, errors.New("scenario_names is required")
	}
	if len(cleanRules) == 0 {
		return AuditorFixRequest{}, errors.New("rule_ids is required")
	}
	return AuditorFixRequest{ScenarioNames: cleanScenarios, RuleIDs: cleanRules}, nil
}

func cleanAuditorValues(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			if _, exists := seen[trimmed]; exists {
				continue
			}
			seen[trimmed] = struct{}{}
			result = append(result, trimmed)
		}
	}
	sort.Strings(result)
	return result
}

func auditorFixSubjectContext(req AuditorFixRequest) (string, error) {
	data, err := json.Marshal(struct {
		ScenarioNames []string `json:"scenario_names"`
		RuleIDs       []string `json:"rule_ids"`
	}{ScenarioNames: cleanAuditorValues(req.ScenarioNames), RuleIDs: cleanAuditorValues(req.RuleIDs)})
	if err != nil {
		return "", fmt.Errorf("encode auditor fix subject: %w", err)
	}
	return string(data), nil
}

func auditorFixResponseToProto(response *AuditorFixResponse) *auditorv1.FixResponse {
	if response == nil {
		return &auditorv1.FixResponse{}
	}
	result := &auditorv1.FixResponse{Count: int32(response.Count), UnfixableRules: append([]string(nil), response.UnfixableRules...), Errors: append([]string(nil), response.Errors...)}
	for _, item := range response.Results {
		mapped := &auditorv1.FixResult{ScenarioName: item.ScenarioName, RuleId: item.RuleID, Fixed: item.Fixed, FilePath: item.FilePath, Error: item.Error}
		for _, change := range item.Changes {
			mapped.Changes = append(mapped.Changes, &auditorv1.FixChange{Type: change.Type, Detail: change.Detail})
		}
		result.Results = append(result.Results, mapped)
	}
	return result
}

func auditorJobStatusToProto(status *AuditorJobStatus) *auditorv1.JobStatus {
	if status == nil {
		return &auditorv1.JobStatus{}
	}
	result := &auditorv1.JobStatus{
		Id: status.ID, Scenario: status.Scenario, ScanType: status.ScanType, Status: status.Status,
		StartedAt: status.StartedAt, ElapsedSeconds: status.ElapsedSeconds,
		TotalScenarios: int32(status.TotalScenarios), ProcessedScenarios: int32(status.ProcessedScenarios),
		ProcessedFiles: int32(status.ProcessedFiles), TotalFiles: int32(status.TotalFiles),
		CurrentScenario: status.CurrentScenario, CurrentFile: status.CurrentFile,
		Message: status.Message, Error: status.Error,
	}
	if status.CompletedAt != nil {
		result.CompletedAt = *status.CompletedAt
	}
	if status.Result != nil {
		result.Result = auditorCheckResultToProto(status.Result)
	}
	return result
}

func auditorCheckResultToProto(result *AuditorCheckResult) *auditorv1.CheckResult {
	if result == nil {
		return nil
	}
	mapped := &auditorv1.CheckResult{
		CheckId: result.CheckID, Status: result.Status, ScanType: result.ScanType,
		StartedAt: result.StartedAt, CompletedAt: result.CompletedAt, DurationSeconds: result.Duration,
		FilesScanned: int32(result.FilesScanned), Message: result.Message, ScenarioName: result.ScenarioName,
	}
	for _, violation := range result.Violations {
		mapped.Violations = append(mapped.Violations, auditorViolationToProto(violation))
	}
	mapped.Statistics = intCountsToProto(result.Statistics)
	if result.Summary != nil {
		mapped.Summary = auditorViolationSummaryToProto(result.Summary)
	}
	return mapped
}

func auditorViolationToProto(violation AuditorViolation) *auditorv1.Violation {
	return &auditorv1.Violation{
		Id: violation.ID, ScenarioName: violation.ScenarioName, Type: violation.Type,
		Severity: violation.Severity, Title: violation.Title, Description: violation.Description,
		FilePath: violation.FilePath, LineNumber: int32(violation.LineNumber), CodeSnippet: violation.CodeSnippet,
		Recommendation: violation.Recommendation, Standard: violation.Standard,
		DiscoveredAt: violation.DiscoveredAt, Source: violation.Source,
		Metadata: metadataToProto(violation.Metadata),
	}
}

func auditorViolationSummaryToProto(summary *AuditorViolationSummary) *auditorv1.ViolationSummary {
	if summary == nil {
		return nil
	}
	mapped := &auditorv1.ViolationSummary{
		Total: int32(summary.Total), HighestSeverity: summary.HighestSeverity,
		RecommendedSteps: append([]string(nil), summary.RecommendedSteps...), GeneratedAt: summary.GeneratedAt,
	}
	mapped.BySeverity = intCountsToProto(summary.BySeverity)
	for _, count := range summary.ByRule {
		mapped.ByRule = append(mapped.ByRule, &auditorv1.RuleCount{RuleId: count.RuleID, Count: int32(count.Count)})
	}
	for _, excerpt := range summary.TopViolations {
		mapped.TopViolations = append(mapped.TopViolations, &auditorv1.ViolationExcerpt{
			Id: excerpt.ID, Severity: excerpt.Severity, RuleId: excerpt.RuleID,
			Title: excerpt.Title, FilePath: excerpt.FilePath,
		})
	}
	return mapped
}

func auditorRulesResponseToProto(response *AuditorRulesListResponse) *auditorv1.RulesResponse {
	if response == nil {
		return &auditorv1.RulesResponse{}
	}
	result := &auditorv1.RulesResponse{Count: int32(response.Count), Total: int32(response.Total), Categories: metadataToProto(response.Categories)}
	ids := make([]string, 0, len(response.Rules))
	for id := range response.Rules {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		rule := response.Rules[id]
		result.Rules = append(result.Rules, &auditorv1.Rule{
			Id: rule.ID, Name: rule.Name, Description: rule.Description, Category: rule.Category,
			Severity: rule.Severity, Enabled: rule.Enabled, Standard: rule.Standard,
			Targets: append([]string(nil), rule.Targets...),
		})
	}
	return result
}

func auditorViolationsResponseToProto(response *AuditorViolationsResponse) *auditorv1.ViolationsResponse {
	if response == nil {
		return &auditorv1.ViolationsResponse{}
	}
	result := &auditorv1.ViolationsResponse{}
	for _, violation := range response.Violations {
		result.Violations = append(result.Violations, auditorViolationToProto(violation))
	}
	return result
}

func intCountsToProto(values map[string]int) []*auditorv1.IntCount {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]*auditorv1.IntCount, 0, len(keys))
	for _, key := range keys {
		result = append(result, &auditorv1.IntCount{Key: key, Count: int32(values[key])})
	}
	return result
}

func metadataToProto(values map[string]any) []*auditorv1.KeyValue {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]*auditorv1.KeyValue, 0, len(keys))
	for _, key := range keys {
		encoded, err := json.Marshal(values[key])
		if err != nil {
			encoded = []byte(fmt.Sprint(values[key]))
		}
		result = append(result, &auditorv1.KeyValue{Key: key, Value: string(encoded)})
	}
	return result
}
