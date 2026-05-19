package validation_run

import (
	"context"
	"time"

	manifest "development-toolchain-validator/internal/manifest"
)

// SandboxedRunSpec is the input to AgentManagerClient.StartSandboxedRun.
type SandboxedRunSpec struct {
	SkillID    string
	GoldenSlug string
	GoldenPath string
}

// AgentManagerClient is the outbound seam for invoking agent-manager
// sandboxed runs. The validation_run worker consumes this; production
// wires the api-core/discovery-backed Connect adapter; tests use fakes.
//
// seam: AgentManagerClient
type AgentManagerClient interface {
	StartSandboxedRun(ctx context.Context, spec SandboxedRunSpec) (runID string, err error)
	WaitForTerminal(ctx context.Context, runID string, timeout time.Duration) (RunSummary, error)
}

// ToolRunner is the outbound seam for invoking development-tool CLIs
// (scenario-auditor, test-genie, scenario-completeness-scoring) against
// a golden. Each invocation is one "tool validation" run.
//
// seam: ToolRunner
type ToolRunner interface {
	Invoke(ctx context.Context, toolName, goldenPath string) (ToolResult, error)
}

// WorkspaceSandboxClient is the optional outbound seam for fetching
// per-path file content from a sandbox workspace. Used only when the
// manifest's content rules require body inspection.
//
// seam: WorkspaceSandboxClient
type WorkspaceSandboxClient interface {
	FetchPathContent(ctx context.Context, sandboxID, path string) (string, error)
}

// GoldenSource is the manifest-lookup seam over the local golden
// registry. The worker uses this to resolve the on-disk path the
// AgentManager/ToolRunner needs.
//
// seam: GoldenSource
type GoldenSource interface {
	GoldenPath(ctx context.Context, goldenSlug string) (string, error)
}

// ManifestSource is the manifest-lookup seam over the local manifest
// store. The worker resolves (skill_id, golden_slug) before invoking
// the evaluator.
//
// seam: ManifestSource
type ManifestSource interface {
	GetManifest(ctx context.Context, skillID, goldenSlug string) (manifest.Manifest, error)
}
