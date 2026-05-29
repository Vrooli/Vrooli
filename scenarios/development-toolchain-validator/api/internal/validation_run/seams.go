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
	// SkillPrompt is the steer skill's full instruction text, fetched
	// from prompt-manager. It becomes the agent's prompt so the sandboxed
	// run actually executes the skill against the golden. Empty when the
	// content could not be resolved; the adapter falls back to a generic
	// description in that case.
	SkillPrompt string
}

// SkillContentSource resolves a steer skill's full instruction text by
// id. The worker uses it to build the agent prompt for a skill run so
// the sandboxed agent has the actual skill to execute, not just its name.
// Production wires the prompt-manager REST adapter; tests use fakes.
//
// seam: SkillContentSource
type SkillContentSource interface {
	SkillContent(ctx context.Context, skillID string) (string, error)
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
// (test-genie, scenario-completeness-scoring) against a golden. Each
// invocation is one "tool validation" run. Both the golden slug (how the
// tool targets the scenario) and the absolute golden path (what
// test-genie reads via --scenario-path) are supplied because different
// tools need different targeting.
//
// seam: ToolRunner
type ToolRunner interface {
	Invoke(ctx context.Context, toolName, goldenSlug, goldenPath string) (ToolResult, error)
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
