package trials

import (
	"context"
	"encoding/json"
	"strings"
)

// RunResult is the outcome of a single dispatched trial: the metrics the runner
// observed. The service stamps ID/At/TaskID/Suite/GuideTaskID around it.
type RunResult struct {
	Verdict        Verdict
	Tokens         int64
	DurationMs     int64
	SandboxDiffRef string
	Model          string
	Detail         string // human note (e.g. why a run errored); not persisted to the wire
}

// Runner dispatches one trial task through agent-manager (runner=opencode + a
// local model) inside workspace-sandbox and reports the result. It NEVER returns
// an error — a failed dispatch is an honest VerdictError, so a broken sandbox
// degrades one run rather than failing the whole suite.
type Runner interface {
	RunTask(ctx context.Context, task TrialTask, model string) RunResult
}

// defaultModel is the local-model id used when the caller passes none. The
// concrete model is configured operator-side; this is a sane fallback label.
const defaultModel = "ollama/qwen2.5-coder"

// execTrialRunner is the production Runner. It dispatches via agent-manager and
// parses the run's metrics. The exact agent-manager dispatch contract is the
// integration seam — wired operator-side for live runs; CI never dispatches a
// live model (the fake Runner is used in tests). Any failure degrades to
// VerdictError with a detail, never a fabricated pass.
type execTrialRunner struct {
	run CommandRunner
}

// NewRunner returns the production Runner.
func NewRunner() Runner { return &execTrialRunner{run: execRunner} }

// NewRunnerWithCommand returns a Runner using the given CommandRunner (tests).
func NewRunnerWithCommand(run CommandRunner) Runner { return &execTrialRunner{run: run} }

var _ Runner = (*execTrialRunner)(nil)

func (r *execTrialRunner) RunTask(ctx context.Context, task TrialTask, model string) RunResult {
	if model == "" {
		model = defaultModel
	}
	res := RunResult{Verdict: VerdictError, Model: model}
	if r.run == nil {
		res.Detail = "no dispatch runner configured"
		return res
	}
	// Dispatch through agent-manager with the local-model runner, sandboxed. The
	// flags are the documented integration contract; agent-manager attributes the
	// changed files to the returned sandbox diff ref.
	out, err := r.run(ctx, "agent-manager", "trials", "dispatch",
		"--task", task.ID,
		"--suite", task.Suite,
		"--runner", "opencode",
		"--model", model,
		"--sandboxed",
		"--json",
	)
	if err != nil {
		res.Detail = "agent-manager dispatch failed: " + err.Error()
		return res
	}
	return parseDispatch(out, model)
}

// dispatchPayload is the subset of agent-manager's dispatch JSON the trials
// domain consumes. Tolerant of extra fields.
type dispatchPayload struct {
	Verdict        string `json:"verdict"`
	Tokens         int64  `json:"tokens"`
	DurationMs     int64  `json:"duration_ms"`
	SandboxDiffRef string `json:"sandbox_diff_ref"`
	Model          string `json:"model"`
}

// parseDispatch maps agent-manager's dispatch JSON to a RunResult. An
// unparseable payload is an honest VerdictError.
func parseDispatch(out []byte, model string) RunResult {
	var p dispatchPayload
	if err := json.Unmarshal(out, &p); err != nil {
		return RunResult{Verdict: VerdictError, Model: model, Detail: "unparseable dispatch result"}
	}
	res := RunResult{
		Verdict:        verdictFromString(p.Verdict),
		Tokens:         p.Tokens,
		DurationMs:     p.DurationMs,
		SandboxDiffRef: p.SandboxDiffRef,
		Model:          model,
	}
	if p.Model != "" {
		res.Model = p.Model
	}
	return res
}

func verdictFromString(s string) Verdict {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "pass", "passed", "success", "ok":
		return VerdictPass
	case "fail", "failed", "wrong", "insufficient":
		return VerdictFail
	default:
		return VerdictError
	}
}
