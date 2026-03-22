package pipeline

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// BuildProvenance captures the git state at pipeline build time.
// This enables downstream systems (e.g. deployment-manager) to tie builds
// to exact source commits for approval gating and audit trails.
type BuildProvenance struct {
	// GitCommitHash is the full SHA-1 hash of HEAD at build time.
	GitCommitHash string `json:"git_commit_hash"`

	// GitBranch is the branch name (e.g. "main", "feature/foo").
	GitBranch string `json:"git_branch"`

	// GitDirty is true if there were uncommitted changes in the working tree.
	GitDirty bool `json:"git_dirty"`

	// BuiltAt is the timestamp when the build was initiated.
	BuiltAt time.Time `json:"built_at"`

	// Version is the scenario version that was built (from Config or service.json).
	Version string `json:"version"`
}

// CaptureProvenance reads git state from the repository containing dir.
// If dir is not inside a git repository, returns a partial provenance with
// an empty commit hash and dirty=true (conservative default).
func CaptureProvenance(dir, version string) *BuildProvenance {
	p := &BuildProvenance{
		BuiltAt: time.Now(),
		Version: version,
	}

	// Get commit hash
	if hash, err := gitCommand(dir, "rev-parse", "HEAD"); err == nil {
		p.GitCommitHash = hash
	}

	// Get branch name
	if branch, err := gitCommand(dir, "rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		p.GitBranch = branch
	}

	// Check dirty state — any output means uncommitted changes
	if status, err := gitCommand(dir, "status", "--porcelain"); err == nil {
		p.GitDirty = status != ""
	} else {
		// If git status fails, assume dirty (conservative)
		p.GitDirty = true
	}

	return p
}

// gitCommand runs a git command in the given directory and returns trimmed stdout.
func gitCommand(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}
