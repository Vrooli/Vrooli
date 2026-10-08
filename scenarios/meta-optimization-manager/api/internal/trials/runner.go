package trials

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// RunResult is the EVIDENCE a single dispatched trial produced — never a
// verdict. agent-manager is only the sandboxed-agent spawner; deciding PASS/FAIL
// is the Evaluator's job (Service calls it next). The one verdict the Runner may
// set is VerdictError, when the spawn/collection itself failed — an honest
// degradation that fails one run, never a fabricated pass and never the suite.
type RunResult struct {
	Verdict        Verdict // VerdictError on a dispatch/collection failure; else VerdictUnspecified
	Tokens         int64   // summary.tokens_used
	DurationMs     int64   // ended_at - started_at
	ChangedFiles   int     // run.changed_files — the abstention signal for negative cases
	SandboxDiffRef string  // sandbox id / diff path — an attribution pointer
	Diff           string  // the unified diff the agent produced (empty when it abstained)
	RunID          string  // agent-manager run id (evidence pointer)
	Detail         string  // human note (e.g. why a run errored); not persisted to the wire
}

// Runner dispatches one trial task through agent-manager's real sandboxed
// primitive and reports the EVIDENCE. It NEVER
// returns an error — a failed dispatch is an honest VerdictError, so a broken
// sandbox degrades one run rather than failing the whole suite.
type Runner interface {
	RunTask(ctx context.Context, task TrialTask, fixture Fixture) RunResult
}

const (
	trialScenario   = "meta-optimization-manager"
	trialProfileKey = "meta-optimization-manager/trials"
)

// defaultPollInterval bounds how often the Runner polls a sandboxed run for
// terminal status. Trials are long and operator-invoked; a few seconds keeps the
// poll cheap without hammering agent-manager.
const defaultPollInterval = 5 * time.Second

// agent-manager terminal run statuses (proto enum names; the API marshals enums
// by name with UseProtoNames). COMPLETE/NEEDS_REVIEW carry a usable diff (trials
// read the diff but never apply it, so a pending review does not block evidence
// collection); FAILED/CANCELLED mean the run itself errored.
const (
	statusComplete    = "RUN_STATUS_COMPLETE"
	statusNeedsReview = "RUN_STATUS_NEEDS_REVIEW"
	statusFailed      = "RUN_STATUS_FAILED"
	statusCancelled   = "RUN_STATUS_CANCELLED"
)

// execTrialRunner is the production Runner. It drives agent-manager's
// profile reconcile-scenario → task create → run create --run-mode sandboxed → poll run get
// → run diff sequence through the CommandRunner seam and parses each response
// JSON (tolerant of extra/unknown fields, pinned to agent-manager's snake_case
// API shape). Any step failure degrades to VerdictError with a detail — never a
// fabricated pass.
type execTrialRunner struct {
	run          CommandRunner
	pollInterval time.Duration
}

// NewRunner returns the production Runner.
func NewRunner() Runner { return &execTrialRunner{run: execRunner, pollInterval: defaultPollInterval} }

// NewRunnerWithCommand returns a Runner using the given CommandRunner (tests).
func NewRunnerWithCommand(run CommandRunner) Runner {
	return &execTrialRunner{run: run, pollInterval: defaultPollInterval}
}

// newRunnerForTest returns a Runner with a custom poll interval so a
// non-terminal→terminal poll sequence can be exercised without a real sleep.
func newRunnerForTest(run CommandRunner, pollInterval time.Duration) *execTrialRunner {
	return &execTrialRunner{run: run, pollInterval: pollInterval}
}

var _ Runner = (*execTrialRunner)(nil)

func isTerminalStatus(s string) bool {
	switch s {
	case statusComplete, statusNeedsReview, statusFailed, statusCancelled:
		return true
	default:
		return false
	}
}

func (r *execTrialRunner) RunTask(ctx context.Context, task TrialTask, fixture Fixture) RunResult {
	res := RunResult{Verdict: VerdictError}
	if r.run == nil {
		res.Detail = "no dispatch runner configured"
		return res
	}

	// Bound the whole spawn+poll sequence; each individual command additionally
	// carries its own LookPath guard + timeout inside execRunner.
	ctx, cancel := context.WithTimeout(ctx, dispatchTimeout)
	defer cancel()

	// 1. Reconcile the manifest-declared profile; runtime selection belongs to
	// Agent Manager's role resolver, never to this consumer.
	profileID, err := r.reconcileProfile(ctx)
	if err != nil {
		res.Detail = "profile reconciliation failed: " + err.Error()
		return res
	}

	// 2. task create — the agent works on the fixture's target/ (its scope).
	taskID, err := r.createTask(ctx, task, fixture)
	if err != nil {
		res.Detail = "task create failed: " + err.Error()
		return res
	}

	// 3. run create — sandboxed, auto-clean on terminal so trial sandboxes don't
	//    accumulate.
	runID, err := r.createRun(ctx, taskID, profileID)
	if err != nil {
		res.Detail = "run create failed: " + err.Error()
		return res
	}
	res.RunID = runID

	// 4. poll run get → terminal.
	run, err := r.pollRun(ctx, runID)
	if err != nil {
		res.Detail = "run get failed: " + err.Error()
		return res
	}
	res.SandboxDiffRef = run.sandboxRef()
	res.Tokens = run.tokens()
	res.DurationMs = run.durationMs()
	res.ChangedFiles = run.ChangedFiles

	if run.Status == statusFailed || run.Status == statusCancelled {
		res.Detail = "sandboxed run did not complete: status=" + enumShort(run.Status) + " " + run.ErrorMsg
		return res // VerdictError — the run itself errored
	}

	// 5. run diff — capture the agent's produced changes (only if it changed
	//    anything; an empty change set is the abstention signal for negatives).
	if res.ChangedFiles > 0 {
		diff, derr := r.runDiff(ctx, runID)
		if derr != nil {
			res.Detail = "run diff failed: " + derr.Error()
			return res // VerdictError — we can't evaluate without the evidence
		}
		res.Diff = diff
	}

	// Evidence collected — the verdict is the Evaluator's call.
	res.Verdict = VerdictUnspecified
	res.Detail = ""
	return res
}

// =============================================================================
// agent-manager response shapes (the narrow subset trials consumes). Pinned to
// agent-manager's snake_case API marshaler (protoconv UseProtoNames=true);
// json.Decode ignores unknown fields, so extra response fields never break us.
// =============================================================================

type amTask struct {
	ID string `json:"id"`
}

type amRunSummary struct {
	TokensUsed int64 `json:"tokens_used"`
}

type amRun struct {
	ID           string        `json:"id"`
	Status       string        `json:"status"`
	SandboxID    string        `json:"sandbox_id"`
	DiffPath     string        `json:"diff_path"`
	ChangedFiles int           `json:"changed_files"`
	StartedAt    string        `json:"started_at"`
	EndedAt      string        `json:"ended_at"`
	ErrorMsg     string        `json:"error_msg"`
	Summary      *amRunSummary `json:"summary"`
}

func (run amRun) tokens() int64 {
	if run.Summary == nil {
		return 0
	}
	return run.Summary.TokensUsed
}

func (run amRun) sandboxRef() string {
	if id := strings.TrimSpace(run.SandboxID); id != "" {
		return id
	}
	if p := strings.TrimSpace(run.DiffPath); p != "" {
		return p
	}
	return run.ID
}

func (run amRun) durationMs() int64 {
	start, serr := time.Parse(time.RFC3339, run.StartedAt)
	end, eerr := time.Parse(time.RFC3339, run.EndedAt)
	if serr != nil || eerr != nil {
		return 0
	}
	d := end.Sub(start)
	if d < 0 {
		return 0
	}
	return d.Milliseconds()
}

// reconcileProfile applies MoM's manifest-declared, role-only profile and
// returns its stable Agent Manager ID. Reconciliation is idempotent; it cannot
// introduce a caller-selected runner, model, or policy.
func (r *execTrialRunner) reconcileProfile(ctx context.Context) (string, error) {
	out, err := r.run(ctx, "agent-manager", "profile", "reconcile-scenario",
		"--scenario", trialScenario, "--json")
	if err != nil {
		return "", err
	}
	var env struct {
		Results []struct {
			ProfileKey string `json:"profile_key"`
			ProfileID  string `json:"profile_id"`
		} `json:"results"`
	}
	if err := json.Unmarshal(out, &env); err != nil {
		return "", fmt.Errorf("decode profile reconciliation: %w", err)
	}
	for _, item := range env.Results {
		if item.ProfileKey == trialProfileKey && item.ProfileID != "" {
			return item.ProfileID, nil
		}
	}
	return "", fmt.Errorf("profile reconciliation returned no id for %q", trialProfileKey)
}

// createTask creates the agent task scoped to the fixture's target codebase.
func (r *execTrialRunner) createTask(ctx context.Context, task TrialTask, fixture Fixture) (string, error) {
	title := fmt.Sprintf("trial %s: %s", fixture.Family, task.ID)
	out, err := r.run(ctx, "agent-manager", "task", "create",
		"--title", title,
		"--description", fixture.Prompt,
		"--scope-path", fixture.TargetDir,
		"--json",
	)
	if err != nil {
		return "", err
	}
	var env struct {
		Task *amTask `json:"task"`
	}
	if err := json.Unmarshal(out, &env); err != nil {
		return "", fmt.Errorf("decode task create: %w", err)
	}
	if env.Task == nil || env.Task.ID == "" {
		return "", errors.New("task create returned no task id")
	}
	return env.Task.ID, nil
}

// createRun starts a sandboxed run that auto-cleans on terminal.
func (r *execTrialRunner) createRun(ctx context.Context, taskID, profileID string) (string, error) {
	out, err := r.run(ctx, "agent-manager", "run", "create",
		"--task-id", taskID,
		"--profile-id", profileID,
		"--run-mode", "sandboxed",
		"--sandbox-retention-mode", "delete_on_terminal",
		"--json",
	)
	if err != nil {
		return "", err
	}
	run, err := decodeRunEnvelope(out)
	if err != nil {
		return "", fmt.Errorf("decode run create: %w", err)
	}
	if run.ID == "" {
		return "", errors.New("run create returned no run id")
	}
	return run.ID, nil
}

// pollRun polls run get until the run reaches a terminal status or the context
// deadline elapses (treated by the caller as VerdictError).
func (r *execTrialRunner) pollRun(ctx context.Context, runID string) (*amRun, error) {
	for {
		out, err := r.run(ctx, "agent-manager", "run", "get", runID, "--json")
		if err != nil {
			return nil, err
		}
		run, derr := decodeRunEnvelope(out)
		if derr != nil {
			return nil, fmt.Errorf("decode run get: %w", derr)
		}
		if isTerminalStatus(run.Status) {
			return run, nil
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("run %s did not reach a terminal status before timeout (last=%s)",
				runID, enumShort(run.Status))
		case <-time.After(r.pollInterval):
		}
	}
}

// runDiff captures the agent's unified diff as raw text (run diff prints the
// diff, not JSON).
func (r *execTrialRunner) runDiff(ctx context.Context, runID string) (string, error) {
	out, err := r.run(ctx, "agent-manager", "run", "diff", runID)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// decodeRunEnvelope unwraps {"run": {...}} and decodes the run. The wrapper key
// is a single word, identical under either agent-manager field-name convention.
func decodeRunEnvelope(out []byte) (*amRun, error) {
	var env struct {
		Run *amRun `json:"run"`
	}
	if err := json.Unmarshal(out, &env); err != nil {
		return nil, err
	}
	if env.Run == nil {
		return nil, errors.New("response has no run object")
	}
	return env.Run, nil
}

// enumShort trims the RUN_STATUS_ prefix for human detail strings.
func enumShort(s string) string {
	return strings.ToLower(strings.TrimPrefix(s, "RUN_STATUS_"))
}
