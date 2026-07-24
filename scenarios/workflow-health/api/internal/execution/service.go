package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"workflow-health/internal/artifacts"
	"workflow-health/internal/validation"
	"workflow-health/internal/workflows"

	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
)

type Service struct {
	Validator *validation.Engine
	Client    BASClient
	Now       func() time.Time
}

func NewService(client BASClient) *Service {
	return &Service{
		Validator: validation.NewEngine(),
		Client:    client,
		Now:       func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) RunScenario(ctx context.Context, scenario, path string, opts Options) (Report, error) {
	if s == nil {
		return Report{}, fmt.Errorf("execution service is nil")
	}
	validator := s.Validator
	if validator == nil {
		validator = validation.NewEngine()
	}
	static, err := validator.ValidateScenario(ctx, scenario, path)
	if err != nil {
		return Report{}, err
	}
	report := Report{
		Scenario:  static.Scenario,
		TargetDir: static.TargetPath,
		Static:    static,
		Catalog:   static.Catalog,
		Findings:  append([]validation.Finding(nil), static.Findings...),
	}
	selected := selectAssets(static.Catalog, opts)
	report.Summary.Selected = len(selected)
	if !opts.IncludeExecution {
		report.Summary.Skipped = len(selected)
		return report, nil
	}
	if s.Client == nil && !opts.DryRun {
		return report, fmt.Errorf("BAS client is required when execution is enabled")
	}
	isolationInstalled := false
	if opts.Isolation != nil && !opts.DryRun {
		lease, err := opts.Isolation.Acquire(ctx, scenario, firstNonEmpty(opts.RunID, fmt.Sprintf("workflow-health-%d", s.now().UnixNano())))
		if err != nil {
			report.Isolation.InstallError = err.Error()
			report.Findings = append(report.Findings, isolationFinding("routed test isolation was not installed: "+err.Error()))
		} else {
			report.Isolation = lease.Evidence()
			report.Isolation.Installed = true
			isolationInstalled = true
			if opts.ExtraHeaders == nil {
				opts.ExtraHeaders = map[string]string{}
			}
			opts.ExtraHeaders["X-Vrooli-Test-Mode"] = "1"
			defer func() {
				report.Isolation = lease.Close(context.Background())
				if report.Isolation.ClearError != "" {
					report.Findings = append(report.Findings, isolationFinding("routed test isolation cleanup failed: "+report.Isolation.ClearError))
				}
				if report.Isolation.Leaked() {
					report.Findings = append(report.Findings, isolationFinding("routed test isolation leaked primary storage requests or writes"))
				}
				validation.SortFindings(report.Findings)
			}()
		}
	}
	// Resolve the target to an absolute path before it feeds workflow reads or
	// the BAS ProjectRoot; BAS resolves ProjectRoot against its own CWD, so a
	// relative value there loads the wrong selector manifest.
	targetDir, err := filepath.Abs(static.TargetPath)
	if err != nil {
		return Report{}, fmt.Errorf("resolve target path %q: %w", static.TargetPath, err)
	}
	writer := artifacts.NewWriter(targetDir, firstNonEmpty(opts.RunID, fmt.Sprintf("workflow-health-%d", s.now().Unix())))
	for _, asset := range selected {
		run := s.runAsset(ctx, writer, asset, targetDir, opts, isolationInstalled)
		report.Runs = append(report.Runs, run)
		switch {
		case run.Refused:
			report.Summary.Refused++
			report.Findings = append(report.Findings, executionFinding(validation.CodeExecutionRefused, run))
		case run.Skipped:
			report.Summary.Skipped++
		case run.Success:
			report.Summary.Executed++
			report.Summary.Passed++
		default:
			report.Summary.Executed++
			report.Summary.Failed++
			report.Findings = append(report.Findings, executionFinding(validation.CodeExecutionFailed, run))
		}
	}
	validation.SortFindings(report.Findings)
	return report, nil
}

func isolationFinding(description string) validation.Finding {
	return validation.Finding{
		Code:        validation.CodeExecutionRefused,
		Severity:    validation.SeverityError,
		Title:       "Routed test isolation unavailable",
		Description: description,
		Remediation: "Wire database.RoutedDB, test-mode middleware, and devrouting file roots on the target scenario.",
	}
}

func (s *Service) runAsset(ctx context.Context, writer *artifacts.Writer, asset workflows.WorkflowAsset, targetDir string, opts Options, isolationInstalled bool) (run WorkflowRun) {
	started := s.now()
	run = WorkflowRun{Asset: asset, StartedAt: started}
	defer func() {
		if run.CompletedAt.IsZero() {
			run.CompletedAt = s.now()
		}
		// Persist a per-workflow diagnostic even when BAS rejects a request or
		// its execution-status lookup fails. Previously those early returns
		// discarded the only actionable error and left the durable provider with
		// a generic aggregate finding.
		if writer == nil || run.Artifact.Latest != "" {
			return
		}
		latest := artifacts.WorkflowLatest{
			RunID:       opts.RunID,
			AssetID:     asset.ID,
			AssetPath:   asset.Path,
			ExecutionID: run.ExecutionID,
			Status:      run.Status,
			Success:     run.Success,
			Error:       run.Error,
			StartedAt:   run.StartedAt,
			CompletedAt: run.CompletedAt,
			DurationMs:  run.CompletedAt.Sub(run.StartedAt).Milliseconds(),
			Summary:     artifacts.Counts(run.Timeline),
		}
		if artifact, err := writer.WriteWorkflow(asset.ID, asset.Path, run.Timeline, latest); err == nil {
			run.Artifact = artifact
		}
	}()
	if reason := refusalReason(asset, isolationInstalled); reason != "" {
		run.Refused = true
		run.Status = "refused"
		run.Error = reason
		return run
	}
	definition, err := readWorkflowDefinition(filepath.Join(targetDir, asset.Path))
	if err != nil {
		run.Skipped = true
		run.Status = "skipped"
		run.Error = err.Error()
		return run
	}
	if opts.DryRun {
		run.DryRun = true
		run.Success = true
		run.Status = "dry_run"
		return run
	}
	if validation, err := s.Client.ValidateResolved(ctx, definition); err != nil {
		run.Status = "failed"
		run.Error = fmt.Sprintf("BAS validation request failed: %v", err)
		return run
	} else if validation != nil && !validation.Valid {
		run.Status = "failed"
		run.Error = summarizeValidationIssues(validation.Errors)
		return run
	}
	result, err := s.Client.ExecuteAdhoc(ctx, ExecuteRequest{
		Definition:  definition,
		Name:        asset.Name,
		Description: asset.Description,
		Parameters: Parameters{
			ProjectRoot:   filepath.Join(targetDir, "bas"),
			InitialParams: opts.InitialParams,
			InitialStore:  opts.InitialStore,
			Env:           opts.Env,
			ExtraHeaders:  opts.ExtraHeaders,
		},
		Options: ExecuteOptions{
			CollectConsole: opts.CollectConsole,
			CollectNetwork: opts.CollectNetwork,
			CollectDOM:     opts.CollectDOM,
			RequiresVideo:  opts.RequiresVideo,
			RequiresTrace:  opts.RequiresTrace,
			RequiresHAR:    opts.RequiresHAR,
		},
	})
	if err != nil {
		run.Status = "failed"
		run.Error = err.Error()
		return run
	}
	run.ExecutionID = result.ExecutionID
	run.Status = strings.ToLower(strings.TrimPrefix(result.Status.String(), "EXECUTION_STATUS_"))
	run.Success = result.Status == basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED && strings.TrimSpace(result.Error) == ""
	run.Error = strings.TrimSpace(result.Error)
	if run.Status == "" || run.Status == "unspecified" {
		if run.Success {
			run.Status = "completed"
		} else {
			run.Status = "failed"
		}
	}
	if result.ExecutionID != "" {
		if timeline, err := s.Client.Timeline(ctx, result.ExecutionID); err == nil {
			run.Timeline = timeline
		}
	}
	return run
}

func selectAssets(catalog *workflows.ScenarioWorkflowCatalog, opts Options) []workflows.WorkflowAsset {
	if catalog == nil {
		return nil
	}
	caseSet := stringSet(opts.Selector.CasePaths)
	flowSet := stringSet(opts.Selector.FlowPaths)
	var out []workflows.WorkflowAsset
	for _, c := range catalog.Cases {
		if len(caseSet) > 0 {
			if _, ok := caseSet[c.Path]; !ok {
				continue
			}
		}
		out = append(out, c.WorkflowAsset)
	}
	if opts.AllowFlowExecution || len(flowSet) > 0 {
		for _, f := range catalog.Flows {
			if len(flowSet) > 0 {
				if _, ok := flowSet[f.Path]; !ok {
					continue
				}
			} else if !opts.AllowFlowExecution {
				continue
			}
			out = append(out, f.WorkflowAsset)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func refusalReason(asset workflows.WorkflowAsset, isolationInstalled bool) string {
	if asset.ParseError != "" {
		return "workflow JSON did not parse; static validation must pass before execution"
	}
	if !asset.Safety.Mutating {
		return ""
	}
	if !asset.Safety.RequiresConfirmation {
		return "mutating or destructive workflow execution requires metadata.labels.requires_confirmation=true"
	}
	if !asset.Safety.RequiresIsolation {
		return "mutating or destructive workflow execution requires metadata.labels.routed_isolation=true"
	}
	if !isolationInstalled {
		return "mutating workflow execution requires proven routed test isolation"
	}
	return ""
}

func readWorkflowDefinition(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read workflow %s: %w", filepath.ToSlash(path), err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse workflow %s: %w", filepath.ToSlash(path), err)
	}
	resolved, err := workflows.ResolveBASDocument(doc)
	if err != nil {
		return nil, fmt.Errorf("resolve BAS workflow %s: %w", filepath.ToSlash(path), err)
	}
	return resolved.Definition, nil
}

func summarizeValidationIssues(issues []ValidationIssue) string {
	if len(issues) == 0 {
		return "BAS validation failed"
	}
	parts := make([]string, 0, len(issues))
	for _, issue := range issues {
		parts = append(parts, strings.TrimSpace(firstNonEmpty(issue.Code, issue.Message)))
	}
	return "BAS validation failed: " + strings.Join(parts, "; ")
}

func stringSet(values []string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(values))
	for _, v := range values {
		if v = strings.TrimSpace(filepath.ToSlash(v)); v != "" {
			out[v] = struct{}{}
		}
	}
	return out
}

func executionFinding(code string, run WorkflowRun) validation.Finding {
	description := strings.TrimSpace(run.Error)
	if description == "" {
		description = "Workflow execution did not complete successfully."
	}
	return validation.Finding{
		Code:        code,
		Severity:    validation.SeverityError,
		Title:       executionTitle(code),
		Description: description,
		FilePath:    run.Asset.Path,
		AssetID:     run.Asset.ID,
		Remediation: executionRemediation(code),
	}
}

func executionTitle(code string) string {
	switch code {
	case validation.CodeExecutionRefused:
		return "Workflow execution refused"
	case validation.CodeExecutionFailed:
		return "Workflow execution failed"
	default:
		return code
	}
}

func executionRemediation(code string) string {
	switch code {
	case validation.CodeExecutionRefused:
		return "Confirm mutating execution only after routed isolation proof is present."
	case validation.CodeExecutionFailed:
		return "Inspect BAS validation, execution output, and workflow artifacts."
	default:
		return "Inspect and repair the workflow asset."
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func (s *Service) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now()
	}
	return time.Now().UTC()
}
