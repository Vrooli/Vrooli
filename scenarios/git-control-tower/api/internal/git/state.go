// Package git provides read-only extraction of a repository's git state for
// baseline capture. It never mutates the repository (feedback_no_git_mutations):
// every invocation is a read-only porcelain query.
//
// The single entry point is Capture, which returns a populated State in one
// call. A Runner seam (CaptureWith) lets tests drive the parser without a real
// git binary.
package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// State is a snapshot of a repository's git state at baseline-capture time.
// It is embedded in a baseline manifest and serialized to JSON, so field tags
// are stable wire contract.
type State struct {
	Sha           string    `json:"sha"`
	Branch        string    `json:"branch"`                  // "" when detached
	Detached      bool      `json:"detached,omitempty"`      // true on detached HEAD
	Dirty         bool      `json:"dirty"`                   // working tree has changes
	DirtySummary  string    `json:"dirty_summary,omitempty"` // e.g. "12 modified, 3 untracked"
	CommitMessage string    `json:"commit_message,omitempty"`
	CommitAuthor  string    `json:"commit_author,omitempty"`
	CommitDate    time.Time `json:"commit_date,omitempty"`
	Sandboxed     bool      `json:"sandboxed,omitempty"` // captured inside a VROOLI sandbox overlay
}

// Runner executes a read-only git command in repoDir and returns stdout. The
// production runner shells out to git; tests inject a fake.
type Runner func(ctx context.Context, repoDir string, args ...string) ([]byte, error)

// ErrNotARepository is returned when repoDir is not inside a git work tree.
var ErrNotARepository = errors.New("not a git repository")

// execRunner is the production Runner. It prepends --no-optional-locks so a
// read never contends for .git/index.lock with a concurrent writer.
func execRunner(ctx context.Context, repoDir string, args ...string) ([]byte, error) {
	full := append([]string{"--no-optional-locks", "-C", repoDir}, args...)
	cmd := exec.CommandContext(ctx, "git", full...)
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return out, fmt.Errorf("git %s: %w (%s)", strings.Join(args, " "), err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return out, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return out, nil
}

// Capture reads the git state of repoDir using the real git binary.
func Capture(ctx context.Context, repoDir string) (State, error) {
	return CaptureWith(ctx, repoDir, execRunner)
}

// CaptureWith reads the git state of repoDir using the provided Runner. It is
// the testable form of Capture.
//
// Robustness: a repository with no commits yet (HEAD unresolved) is not an
// error — sha/branch/commit fields are left empty and only the dirty summary
// is populated. A non-repository directory returns ErrNotARepository.
func CaptureWith(ctx context.Context, repoDir string, run Runner) (State, error) {
	if _, err := run(ctx, repoDir, "rev-parse", "--is-inside-work-tree"); err != nil {
		// Keep the stable sentinel for callers that need to classify a
		// non-repository, but retain the probe failure. The old implementation
		// erased stderr and made permissions, malformed paths, and canceled
		// commands indistinguishable from an actually missing work tree.
		return State{}, fmt.Errorf("%w: %w", ErrNotARepository, err)
	}

	st := State{Sandboxed: os.Getenv("VROOLI_SANDBOX_MERGED") != ""}

	// HEAD sha — empty when the repo has no commits yet.
	if out, err := run(ctx, repoDir, "rev-parse", "HEAD"); err == nil {
		st.Sha = strings.TrimSpace(string(out))
	}

	// Branch — "HEAD" indicates detached.
	if out, err := run(ctx, repoDir, "rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		branch := strings.TrimSpace(string(out))
		if branch == "HEAD" {
			st.Detached = true
		} else {
			st.Branch = branch
		}
	}

	// Working-tree dirtiness + summary.
	if out, err := run(ctx, repoDir, "status", "--porcelain", "--untracked-files=all"); err == nil {
		st.Dirty, st.DirtySummary = summarizeStatus(string(out))
	}

	// HEAD commit metadata (skipped on an empty repo).
	if st.Sha != "" {
		if out, err := run(ctx, repoDir, "log", "-1", "--pretty=format:%s%x00%an%x00%cI"); err == nil {
			parts := strings.SplitN(strings.TrimRight(string(out), "\n"), "\x00", 3)
			if len(parts) == 3 {
				st.CommitMessage = parts[0]
				st.CommitAuthor = parts[1]
				if t, perr := time.Parse(time.RFC3339, strings.TrimSpace(parts[2])); perr == nil {
					st.CommitDate = t
				}
			}
		}
	}

	return st, nil
}

// summarizeStatus parses `git status --porcelain` output into a dirty flag and
// a human summary like "12 modified, 3 untracked". Untracked lines start with
// "??"; everything else is a tracked change.
func summarizeStatus(porcelain string) (bool, string) {
	var modified, untracked int
	for _, line := range strings.Split(porcelain, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.HasPrefix(line, "??") {
			untracked++
		} else {
			modified++
		}
	}
	if modified == 0 && untracked == 0 {
		return false, ""
	}
	var parts []string
	if modified > 0 {
		parts = append(parts, fmt.Sprintf("%d modified", modified))
	}
	if untracked > 0 {
		parts = append(parts, fmt.Sprintf("%d untracked", untracked))
	}
	return true, strings.Join(parts, ", ")
}
