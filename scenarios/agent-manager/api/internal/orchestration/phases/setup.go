// Workspace setup: sandbox creation for sandboxed runs, working-tree
// resolution for in-place runs.
//
// CreateSandboxWorkspace is the canonical sandbox-creation seam: it
// constructs the workspace-sandbox CreateRequest with a stable
// idempotency key (so retries are safe), persists the resulting
// sandbox UUID on the run, and resolves the host workspace path that the
// runner will chdir into.

package phases

import (
	"context"
	"fmt"
	"path/filepath"

	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// SetupWorkspaceInput is the explicit input to SetupWorkspace.
type SetupWorkspaceInput struct {
	Deps    Deps
	Run     *domain.Run
	Task    *domain.Task
	Profile *domain.AgentProfile
	Sandbox sandbox.Provider

	// ExistingSandboxID is set when WithExistingSandbox was used (resumption,
	// continue-run, etc.). When non-nil, SetupWorkspace skips creation and
	// resolves the existing sandbox's workspace path.
	ExistingSandboxID *uuid.UUID
	ExistingWorkDir   string
}

// SetupWorkspaceOutput carries the workspace state produced by setup.
type SetupWorkspaceOutput struct {
	SandboxID *uuid.UUID
	WorkDir   string
}

// SetupWorkspace dispatches between sandbox creation, sandbox reuse, and
// in-place workspace resolution based on Run.RunMode and ExistingSandboxID.
//
// On error, the returned output is zero-valued; the caller should fail the
// run with the returned domain error.
func SetupWorkspace(ctx context.Context, in SetupWorkspaceInput) (SetupWorkspaceOutput, error) {
	if in.Run.RunMode != domain.RunModeSandboxed {
		workDir, err := UseInPlaceWorkspace(in.Task)
		if err != nil {
			return SetupWorkspaceOutput{}, err
		}
		return SetupWorkspaceOutput{WorkDir: workDir}, nil
	}

	if in.ExistingSandboxID != nil {
		workDir := in.ExistingWorkDir
		if workDir == "" && in.Sandbox != nil {
			if got, err := in.Sandbox.GetWorkspacePath(ctx, *in.ExistingSandboxID); err == nil {
				workDir = got
			}
		}
		if workDir == "" {
			return SetupWorkspaceOutput{}, domain.NewValidationErrorWithHint(
				"sandboxId",
				"sandbox workdir not available",
				"ensure the sandbox is active and has a workdir",
			)
		}
		EmitSystemEvent(ctx, in.Deps, in.Run.ID, "info", "reusing existing sandbox")
		return SetupWorkspaceOutput{
			SandboxID: in.ExistingSandboxID,
			WorkDir:   workDir,
		}, nil
	}

	return CreateSandboxWorkspace(ctx, in)
}

// CreateSandboxWorkspace creates a fresh sandbox via the sandbox provider
// and resolves its host workspace path. Used by SetupWorkspace and
// directly by tests that want to drive the create path in isolation.
func CreateSandboxWorkspace(ctx context.Context, in SetupWorkspaceInput) (SetupWorkspaceOutput, error) {
	if in.Sandbox == nil {
		return SetupWorkspaceOutput{}, domain.NewConfigMissingError("sandbox", "provider not configured", nil)
	}

	idempotencyKey := fmt.Sprintf("sandbox:run:%s", in.Run.ID.String())

	projectRoot := in.Task.ProjectRoot
	if projectRoot != "" && !filepath.IsAbs(projectRoot) {
		if absRoot, err := filepath.Abs(projectRoot); err == nil {
			projectRoot = absRoot
		}
	}

	metadata := map[string]string{
		"agent_manager_run_id": in.Run.ID.String(),
	}
	sbx, err := in.Sandbox.Create(ctx, sandbox.CreateRequest{
		Name:           BuildSandboxName(in.Run, in.Task, in.Profile),
		ScopePath:      in.Task.ScopePath,
		NoLock:         noLockFromSandboxConfig(in.Run.SandboxConfig),
		ProjectRoot:    projectRoot,
		Owner:          in.Run.ID.String(),
		OwnerType:      "run",
		IdempotencyKey: idempotencyKey,
		Behavior:       in.Run.SandboxConfig,
		Metadata:       metadata,
	})
	if err != nil {
		if _, ok := err.(*domain.SandboxError); ok {
			return SetupWorkspaceOutput{}, err
		}
		return SetupWorkspaceOutput{}, &domain.SandboxError{
			Operation:   "create",
			Cause:       err,
			IsTransient: true,
			CanRetry:    true,
		}
	}

	in.Run.SandboxID = &sbx.ID
	if in.Deps.Runs != nil {
		if err := in.Deps.Runs.Update(ctx, in.Run); err != nil {
			return SetupWorkspaceOutput{}, &domain.DatabaseError{
				Operation:   "update",
				EntityType:  "Run",
				EntityID:    in.Run.ID.String(),
				Cause:       err,
				IsTransient: true,
			}
		}
	}

	workDir, err := in.Sandbox.GetWorkspacePath(ctx, sbx.ID)
	if err != nil {
		sbxID := sbx.ID
		return SetupWorkspaceOutput{}, &domain.SandboxError{
			SandboxID:   &sbxID,
			Operation:   "get_workspace_path",
			Cause:       err,
			IsTransient: true,
			CanRetry:    true,
		}
	}

	return SetupWorkspaceOutput{
		SandboxID: &sbx.ID,
		WorkDir:   workDir,
	}, nil
}

// UseInPlaceWorkspace returns the project root for in-place runs.
func UseInPlaceWorkspace(task *domain.Task) (string, error) {
	if task == nil || task.ProjectRoot == "" {
		return "", domain.NewValidationErrorWithHint(
			"projectRoot",
			"project root is required for in-place execution",
			"Specify projectRoot in the task or use sandboxed run mode",
		)
	}
	return task.ProjectRoot, nil
}

// BuildSandboxName constructs a descriptive name for the sandbox.
// Priority: run.Tag > task.Title > scope path.
// Profile name is appended when available for context.
func BuildSandboxName(run *domain.Run, task *domain.Task, profile *domain.AgentProfile) string {
	if run != nil {
		if tag := run.GetTag(); tag != "" {
			return tag
		}
	}

	profileName := ""
	if profile != nil && profile.Name != "" {
		profileName = profile.Name
	}

	if task != nil && task.Title != "" {
		if profileName != "" {
			return fmt.Sprintf("%s (%s)", task.Title, profileName)
		}
		return task.Title
	}

	scope := ""
	if task != nil {
		scope = task.ScopePath
	}
	if scope == "" {
		scope = "/"
	}
	if profileName != "" {
		return fmt.Sprintf("%s (%s)", scope, profileName)
	}
	return scope
}

// noLockFromSandboxConfig returns the NoLock value from a SandboxConfig,
// or nil if the config doesn't explicitly set it. Returning nil lets the
// workspace-sandbox server apply its own DefaultNoLock setting, rather than
// the agent-manager always overriding with false when NoLock isn't specified.
func noLockFromSandboxConfig(cfg *domain.SandboxConfig) *bool {
	if cfg == nil || !cfg.NoLock {
		return nil
	}
	t := true
	return &t
}
