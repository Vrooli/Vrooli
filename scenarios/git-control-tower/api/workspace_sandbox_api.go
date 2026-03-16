package main

import "context"

// WorkspaceSandboxAPI abstracts all communication with workspace-sandbox.
// This is the primary seam for isolating workspace-sandbox side effects.
//
// Production code uses WorkspaceSandboxClient which makes HTTP requests.
// Test code can use FakeWorkspaceSandboxAPI to exercise domain logic
// without real network calls.
//
// SEAM BOUNDARY: All workspace-sandbox operations must flow through this interface.
type WorkspaceSandboxAPI interface {
	// GetCommitPreview fetches all pending approved changes for a project.
	GetCommitPreview(ctx context.Context, projectRoot string) (*workspaceSandboxCommitPreview, error)

	// GetCommitPreviewForPaths fetches approved changes filtered to specific paths.
	GetCommitPreviewForPaths(ctx context.Context, projectRoot string, paths []string) (*workspaceSandboxCommitPreview, error)

	// MarkCommitted notifies workspace-sandbox that files have been committed
	// externally (e.g., via git-control-tower's commit flow).
	MarkCommitted(ctx context.Context, projectRoot string, filePaths []string, commitHash, commitMessage string) error

	// GetProvenanceByRun returns pending applied changes grouped by agent-manager run ID.
	GetProvenanceByRun(ctx context.Context, projectRoot string) (*workspaceSandboxProvenanceResponse, error)
}

// Verify WorkspaceSandboxClient implements WorkspaceSandboxAPI at compile time.
var _ WorkspaceSandboxAPI = (*WorkspaceSandboxClient)(nil)
