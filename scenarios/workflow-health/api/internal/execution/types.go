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
	IncludeExecution   bool
	DryRun             bool
	RunID              string
	Selector           AssetSelector
	AllowFlowExecution bool
	InitialParams      map[string]any
	InitialStore       map[string]any
	Env                map[string]any
	ExtraHeaders       map[string]string
	CollectConsole     bool
	CollectNetwork     bool
	CollectDOM         bool
	RequiresVideo      bool
	RequiresTrace      bool
	RequiresHAR        bool
	Isolation          IsolationCoordinator
	// ElectronTarget attaches selected existing workflow assets to a
	// target-owned Electron renderer. ValidationContext is required with it.
	ElectronTarget    *ElectronTarget
	ValidationContext *ValidationContext
}

type Report struct {
	Scenario  string
	TargetDir string
	Static    validation.Report
	Catalog   *workflows.ScenarioWorkflowCatalog
	Runs      []WorkflowRun
	Findings  []validation.Finding
	Summary   Summary
	Isolation IsolationEvidence
}

// IsolationEvidence is the durable, provider-owned proof collected from the
// target scenario after a workflow suite completes.
type IsolationEvidence struct {
	Installed                       bool
	LeaseID                         string
	InstallError                    string
	HeartbeatError                  string
	ClearError                      string
	TestPoolRequests                int64
	PrimaryDuringTestModeRequests   int64
	TestRootWrites                  int64
	PrimaryRootWritesDuringTestMode int64
}

func (e IsolationEvidence) Leaked() bool {
	return e.PrimaryDuringTestModeRequests > 0 || e.PrimaryRootWritesDuringTestMode > 0
}

// IsolationCoordinator owns one target-scenario lease around a workflow run.
// It deliberately lives at the execution boundary, so static validation and
// direct unit tests remain independent of network routing.
type IsolationCoordinator interface {
	Acquire(context.Context, string, string) (IsolationLease, error)
}

type IsolationLease interface {
	Evidence() IsolationEvidence
	Close(context.Context) IsolationEvidence
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
	Timeline(ctx context.Context, executionID string, headers map[string]string) (*bastimeline.ExecutionTimeline, error)
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
	CollectConsole    bool
	CollectNetwork    bool
	CollectDOM        bool
	RequiresVideo     bool
	RequiresTrace     bool
	RequiresHAR       bool
	ElectronTarget    *ElectronTarget
	ValidationContext *ValidationContext
}

type ElectronTarget struct {
	TargetID       string
	CDPEndpoint    string
	RendererID     string
	RendererURL    string
	RendererTitle  string
	ScenarioName   string
	ArtifactDigest string
	ContextID      string
	CDPTransport   string
}

type ValidationContext struct {
	ContextID        string
	ScenarioName     string
	ArtifactDigest   string
	TargetID         string
	WorkflowID       string
	ProfileID        string
	IsolationLeaseID string
}

type ExecuteResult struct {
	ExecutionID string
	Status      basbase.ExecutionStatus
	Error       string
	Execution   *basexecution.Execution
}
