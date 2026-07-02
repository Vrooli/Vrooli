package execution

import (
	"context"
	"time"

	"workflow-health/internal/artifacts"
	"workflow-health/internal/validation"
	"workflow-health/internal/workflows"

	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
)

type AssetSelector struct {
	CasePaths []string
	FlowPaths []string
}

type Options struct {
	IncludeExecution      bool
	DryRun                bool
	RunID                 string
	Selector              AssetSelector
	AllowFlowExecution    bool
	ConfirmMutating       bool
	RoutedIsolationProven bool
	InitialParams         map[string]any
	InitialStore          map[string]any
	Env                   map[string]any
	ExtraHeaders          map[string]string
	CollectConsole        bool
	CollectNetwork        bool
	CollectDOM            bool
	RequiresVideo         bool
	RequiresTrace         bool
	RequiresHAR           bool
}

type Report struct {
	Scenario  string
	TargetDir string
	Static    validation.Report
	Catalog   *workflows.ScenarioWorkflowCatalog
	Runs      []WorkflowRun
	Findings  []validation.Finding
	Summary   Summary
}

type Summary struct {
	Selected int
	Executed int
	Refused  int
	Skipped  int
	Failed   int
	Passed   int
}

type WorkflowRun struct {
	Asset       workflows.WorkflowAsset
	ExecutionID string
	Status      string
	Success     bool
	Refused     bool
	Skipped     bool
	DryRun      bool
	Error       string
	StartedAt   time.Time
	CompletedAt time.Time
	Artifact    artifacts.RunArtifact
	Timeline    *bastimeline.ExecutionTimeline
}

type BASClient interface {
	ValidateResolved(ctx context.Context, definition map[string]any) (*ValidationResult, error)
	ExecuteAdhoc(ctx context.Context, req ExecuteRequest) (*ExecuteResult, error)
	Timeline(ctx context.Context, executionID string) (*bastimeline.ExecutionTimeline, error)
}

type ValidationResult struct {
	Valid    bool
	Errors   []ValidationIssue
	Warnings []ValidationIssue
}

type ValidationIssue struct {
	Code    string
	Message string
}

type ExecuteRequest struct {
	Definition  map[string]any
	Name        string
	Description string
	Parameters  Parameters
	Options     ExecuteOptions
}

type Parameters struct {
	ProjectRoot   string
	InitialParams map[string]any
	InitialStore  map[string]any
	Env           map[string]any
	ExtraHeaders  map[string]string
}

type ExecuteOptions struct {
	CollectConsole bool
	CollectNetwork bool
	CollectDOM     bool
	RequiresVideo  bool
	RequiresTrace  bool
	RequiresHAR    bool
}

type ExecuteResult struct {
	ExecutionID string
	Status      basbase.ExecutionStatus
	Error       string
	Execution   *basexecution.Execution
}
