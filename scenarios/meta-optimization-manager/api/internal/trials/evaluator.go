package trials

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Evaluator decides the verdict for a dispatched trial from the EVIDENCE the
// Runner collected. This is the half that must live in MoM: agent-manager
// returns a sandbox diff + metrics; whether the SWE task was actually solved is
// MoM's call. Evaluation is deterministic-first (run the fixture's oracle
// against the agent's diff), with an agent-judge fallback only for families that
// have no deterministic oracle — and those results are labelled lower-confidence.
//
// Judge NEVER fabricates a pass: any uncertainty or evaluation failure is
// VerdictError, and the run is still recorded.
type Evaluator interface {
	Judge(ctx context.Context, task TrialTask, fixture Fixture, res RunResult) Verdict
}

// oracleChecker applies the agent's diff onto a fresh copy of the fixture target
// and runs the fixture oracle there, returning the oracle's process exit code
// (0 = solved). A non-nil error means the check could not be PERFORMED
// (copy/apply/exec failure) → VerdictError, which is distinct from a clean
// non-zero exit (→ VerdictFail). Production wires a filesystem+exec impl; tests
// fake it so CI never touches a real sandbox.
type oracleChecker func(ctx context.Context, fixture Fixture, diff string) (exitCode int, output string, err error)

// agentJudge is the lower-confidence fallback for families with NO deterministic
// oracle: a single bounded rubric call. Returns a Verdict (Pass/Fail) or an
// error if the judge is unavailable. A nil agentJudge means "not configured" —
// an oracle-less family then yields VerdictError (honest, never a guessed pass).
type agentJudge func(ctx context.Context, task TrialTask, fixture Fixture, res RunResult) (Verdict, error)

// oracleTimeout bounds a single deterministic-oracle check.
const oracleTimeout = 5 * time.Minute

type prodEvaluator struct {
	check  oracleChecker
	judge  agentJudge
	logger *log.Logger
}

// NewEvaluator returns the production Evaluator: a real filesystem+exec oracle
// checker and no agent-judge (oracle-less families degrade to VerdictError until
// a judge is wired). All five shipped fixtures carry a deterministic oracle, so
// the fallback is not on the normal path.
func NewEvaluator(logger *log.Logger) Evaluator {
	return &prodEvaluator{check: realOracleCheck, judge: nil, logger: logger}
}

// NewEvaluatorWithDeps constructs an Evaluator with explicit seams (tests /
// future agent-judge wiring).
func NewEvaluatorWithDeps(check oracleChecker, judge agentJudge, logger *log.Logger) Evaluator {
	if check == nil {
		check = realOracleCheck
	}
	return &prodEvaluator{check: check, judge: judge, logger: logger}
}

var _ Evaluator = (*prodEvaluator)(nil)

func (e *prodEvaluator) logf(format string, args ...any) {
	if e.logger != nil {
		e.logger.Printf(format, args...)
	}
}

func (e *prodEvaluator) Judge(ctx context.Context, task TrialTask, fixture Fixture, res RunResult) Verdict {
	// Defensive: the Service does not call Judge on a runner error, but never
	// upgrade an error to a pass if it does.
	if res.Verdict == VerdictError {
		return VerdictError
	}

	// Negative / honesty case: pass = correct abstention. The strongest
	// deterministic signal available from the evidence is "the agent made no
	// substantive change". A fabricated diff is a fail.
	if fixture.Negative || task.Negative {
		if res.ChangedFiles == 0 && strings.TrimSpace(res.Diff) == "" {
			return VerdictPass
		}
		e.logf("trials: negative task %s fabricated %d changed file(s) — fail", task.ID, res.ChangedFiles)
		return VerdictFail
	}

	// Deterministic-first: run the fixture oracle against the produced diff.
	if len(fixture.Oracle) > 0 {
		exitCode, output, err := e.check(ctx, fixture, res.Diff)
		if err != nil {
			e.logf("trials: oracle check errored for task %s: %v", task.ID, err)
			return VerdictError
		}
		if exitCode == 0 {
			return VerdictPass
		}
		e.logf("trials: oracle failed for task %s (exit=%d): %s", task.ID, exitCode, truncateOutput(output))
		return VerdictFail
	}

	// Agent-judge fallback — only for open-ended families with no oracle.
	// Lower-confidence by construction; labelled in the log.
	if e.judge == nil {
		e.logf("trials: task %s has no deterministic oracle and no agent-judge configured — error", task.ID)
		return VerdictError
	}
	verdict, err := e.judge(ctx, task, fixture, res)
	if err != nil {
		e.logf("trials: agent-judge unavailable for task %s: %v", task.ID, err)
		return VerdictError
	}
	e.logf("trials: task %s judged %s by agent-judge (LOWER CONFIDENCE — no deterministic oracle)", task.ID, verdictName(verdict))
	return verdict
}

// realOracleCheck copies the fixture target into a throwaway dir, applies the
// agent's diff, and runs the oracle there. Exit 0 = solved. Only ever exercised
// in the operator live e2e — CI fakes this seam.
func realOracleCheck(ctx context.Context, fixture Fixture, diff string) (int, string, error) {
	ctx, cancel := context.WithTimeout(ctx, oracleTimeout)
	defer cancel()

	work, err := os.MkdirTemp("", "mom-trial-oracle-*")
	if err != nil {
		return 0, "", fmt.Errorf("oracle workdir: %w", err)
	}
	defer os.RemoveAll(work)

	if err := copyTree(fixture.TargetDir, work); err != nil {
		return 0, "", fmt.Errorf("copy fixture target: %w", err)
	}
	if strings.TrimSpace(diff) != "" {
		if err := applyDiff(ctx, work, diff); err != nil {
			return 0, "", fmt.Errorf("apply diff: %w", err)
		}
	}
	if len(fixture.Oracle) == 0 {
		return 0, "", errors.New("fixture has no oracle command")
	}

	// The oracle script lives in the fixture dir (OUTSIDE the agent's target/
	// scope, so it can't be gamed), but runs with cwd = the diff-applied copy so
	// it inspects the agent's result with plain relative paths. Resolve any
	// oracle argument that names a file in the fixture dir to its absolute path.
	args := append([]string(nil), fixture.Oracle...)
	for i := 1; i < len(args); i++ {
		if cand := filepath.Join(fixture.Dir, args[i]); fileExists(cand) {
			args[i] = cand
		}
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = work
	out, runErr := cmd.CombinedOutput()
	if runErr == nil {
		return 0, string(out), nil
	}
	var ee *exec.ExitError
	if errors.As(runErr, &ee) {
		// The oracle ran and reported failure — a clean FAIL, not an error.
		return ee.ExitCode(), string(out), nil
	}
	// Could not run the oracle at all → can't evaluate.
	return 0, string(out), fmt.Errorf("run oracle %v: %w", fixture.Oracle, runErr)
}

// applyDiff applies a unified diff onto a working tree, preferring git apply
// (handles creates/deletes/renames) and falling back to patch(1).
func applyDiff(ctx context.Context, work, diff string) error {
	patchPath := filepath.Join(work, ".mom-trial.patch")
	if err := os.WriteFile(patchPath, []byte(diff), 0o600); err != nil {
		return err
	}
	defer os.Remove(patchPath)

	git := exec.CommandContext(ctx, "git", "apply", "--whitespace=nowarn", "-p1", patchPath)
	git.Dir = work
	if out, err := git.CombinedOutput(); err == nil {
		return nil
	} else {
		gitErr := fmt.Errorf("git apply: %w (%s)", err, strings.TrimSpace(string(out)))
		// Fall back to patch(1).
		f, oerr := os.Open(patchPath)
		if oerr != nil {
			return gitErr
		}
		defer f.Close()
		p := exec.CommandContext(ctx, "patch", "-p1", "--no-backup-if-mismatch", "-r", "-")
		p.Dir = work
		p.Stdin = f
		if pout, perr := p.CombinedOutput(); perr != nil {
			return fmt.Errorf("%v; patch fallback: %w (%s)", gitErr, perr, strings.TrimSpace(string(pout)))
		}
		return nil
	}
}

// copyTree recursively copies src into dst (which must already exist),
// preserving file modes. Symlinks are skipped (fixtures are plain files).
func copyTree(src, dst string) error {
	return filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, rerr := filepath.Rel(src, p)
		if rerr != nil {
			return rerr
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm()|0o700)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		return copyFile(p, target, info.Mode().Perm())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

func truncateOutput(s string) string {
	s = strings.TrimSpace(s)
	const max = 240
	if len(s) > max {
		return s[:max] + "…"
	}
	return s
}

func verdictName(v Verdict) string {
	switch v {
	case VerdictPass:
		return "pass"
	case VerdictFail:
		return "fail"
	case VerdictError:
		return "error"
	default:
		return "unspecified"
	}
}
