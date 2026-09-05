// Runner execution + immutable policy-candidate fallback.
//
// ExecuteAgent runs the agent through runner.Execute, then validates the
// outcome and reports the result + execErr. ExecuteWithModelFallback wraps it
// with the run-owned candidate sequence; execution never rereads catalog state.
//
// Mutates Run.ActualModel, Run.Status (to Running), Run.SessionID, and the
// runtime callbacks (transcript cursor, runner pid/pgid, session id).

package phases

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/eventlog"
	"agent-manager/internal/fallback"
	"agent-manager/internal/runstate"

	"github.com/google/uuid"
)

// ExecuteAgentInput is the explicit input to ExecuteAgent.
type ExecuteAgentInput struct {
	Deps          Deps
	Run           *domain.Run
	Task          *domain.Task
	Profile       *domain.AgentProfile
	Runner        runner.Runner
	WorkingDir    string
	RunStateRoot  string
	RunStateWrite func()
	SandboxID     *uuid.UUID
	Prompt        string
	SystemPrompt  string
	Attachments   []runner.Attachment
	EnvVars       map[string]string
	EventSink     runner.EventSink
	RunState      *runstate.State
	Mu            *sync.Mutex
	ModelHealth   ModelHealthReporter
	Runners       runner.Registry

	// OnRunning fires when the run flips to RunStatusRunning. Used by
	// the spawn dispatcher to release the startup slot. Nil is safe.
	OnRunning func()
}

// ExecuteAgentOutput carries the result + transient state of one Execute call.
type ExecuteAgentOutput struct {
	Result   *runner.ExecuteResult
	ExecErr  error
	RunState *runstate.State
}

// ExecuteAgent runs the runner once with the supplied request and applies
// validateRunOutcome. Promotes typed terminal errors from the runner result
// into ExecErr so the failure-event emitter takes the typed-error branch.
func ExecuteAgent(ctx context.Context, in ExecuteAgentInput) ExecuteAgentOutput {
	out := ExecuteAgentOutput{RunState: in.RunState}

	in.Run.Status = domain.RunStatusRunning
	in.Run.UpdatedAt = in.Deps.Now()
	if in.Deps.Runs != nil {
		if err := in.Deps.Runs.Update(ctx, in.Run); err != nil {
			EmitSystemEvent(ctx, in.Deps, in.Run.ID, "warn",
				"failed to persist run start: "+err.Error())
		}
	}
	if in.Deps.Broadcaster != nil {
		in.Deps.Broadcaster.BroadcastRunStatus(in.Run)
	}
	// Signal the spawn dispatcher that the startup slot can be
	// released. Done immediately after status flip so the next queued
	// run can begin its bootstrap as soon as we're past the codex
	// SQLite/rollout-file race window.
	if in.OnRunning != nil {
		in.OnRunning()
	}
	if in.Run.ResolvedConfig != nil && in.Run.ResolvedConfig.Effort != "" {
		caps := in.Runner.Capabilities()
		if !caps.SupportsEffort || (!caps.EffortModelSpecific && caps.EffortMappings[string(in.Run.ResolvedConfig.Effort)] == "") {
			EmitSystemEvent(ctx, in.Deps, in.Run.ID, "warn", fmt.Sprintf(
				"reasoning effort %q ignored: runner %q has no declared native mapping",
				in.Run.ResolvedConfig.Effort,
				in.Runner.Type(),
			))
		}
	}

	transcriptCfg, runState, err := PrepareTranscriptConfig(ctx, PrepareTranscriptInput{
		Deps:          in.Deps,
		Run:           in.Run,
		RunStateRoot:  in.RunStateRoot,
		RunStateWrite: in.RunStateWrite,
		WorkingDir:    in.WorkingDir,
		Mu:            in.Mu,
		Existing:      in.RunState,
	})
	if err != nil {
		out.ExecErr = err
		return out
	}
	out.RunState = runState

	req := runner.ExecuteRequest{
		RunID:          in.Run.ID,
		Tag:            in.Run.GetTag(),
		Profile:        in.Profile,
		ResolvedConfig: in.Run.ResolvedConfig,
		Task:           in.Task,
		WorkingDir:     in.WorkingDir,
		SandboxID:      in.SandboxID,
		Prompt:         in.Prompt,
		SystemPrompt:   groundWorkingDir(in.SystemPrompt, in.WorkingDir),
		EventSink:      in.EventSink,
		Attachments:    in.Attachments,
		Environment:    in.EnvVars,
		Transcript:     transcriptCfg,
	}

	result, execErr := in.Runner.Execute(ctx, req)
	out.Result = result
	out.ExecErr = execErr

	// Promote typed terminal errors from the runner result into execErr so
	// the failure-event emitter takes the typed-error branch and the
	// classifier surfaces the right ErrorCode (SANDBOX_NO_EXIT_INFO etc.).
	if out.ExecErr == nil && out.Result != nil && out.Result.TerminalError != nil {
		out.ExecErr = out.Result.TerminalError
	}

	// Categorize silent launch failures.
	validated := ValidateRunOutcome(ctx, ValidateOutcomeInput{
		Deps:    in.Deps,
		Run:     in.Run,
		Result:  out.Result,
		ExecErr: out.ExecErr,
	})
	out.Result = validated.Result
	out.ExecErr = validated.ExecErr

	return out
}

// groundWorkingDir strengthens path grounding by appending an explicit,
// unambiguous working-directory directive to the system prompt.
//
// Every runner already receives the working directory — but only as a single
// passive line buried in a large system prompt (e.g. opencode's "Working
// directory: <dir>"). Capable cloud models connect that fact to file paths
// reliably; weaker models (notably local Ollama coders) intermittently ignore
// it and emit a write to a hallucinated absolute path, producing a run that
// "succeeds" while writing nothing useful. Stating the directory plainly with
// a directive to use it eliminated that failure in testing.
//
// Applied to all runners via the shared SystemPrompt field (Claude delivers it
// through --append-system-prompt; Codex/OpenCode prepend it to the prompt).
// The directive is model-agnostic and harmless to capable models. Empty
// workingDir (no resolved directory) leaves the prompt untouched.
func groundWorkingDir(systemPrompt, workingDir string) string {
	dir := strings.TrimSpace(workingDir)
	if dir == "" {
		return systemPrompt
	}
	directive := "<working-directory>\n" +
		"Your working directory for this task is the absolute path: " + dir + "\n" +
		"All relative paths resolve against it. When you create, edit, or read files, use " +
		"paths under this directory — either an absolute path beginning with " + dir + "/ or a " +
		"path relative to it. Do not invent or target any other directory.\n" +
		"</working-directory>"
	if strings.TrimSpace(systemPrompt) == "" {
		return directive
	}
	return systemPrompt + "\n\n" + directive
}

// ExecuteWithModelFallbackInput is the explicit input.
type ExecuteWithModelFallbackInput struct {
	ExecuteAgentInput
}

// ExecuteWithModelFallback walks the immutable candidate sequence persisted on
// the run. A historical run without a snapshot executes once; it never falls
// back through mutable policy state.
func ExecuteWithModelFallback(ctx context.Context, in ExecuteWithModelFallbackInput) ExecuteAgentOutput {
	if policySnapshot(in.Run) != nil {
		return executePolicySnapshot(ctx, in)
	}
	out := ExecuteAgent(ctx, in.ExecuteAgentInput)
	recordActualModel(in.Run, currentModel(in.Run))
	return out
}

// executePolicySnapshot walks only the candidate values copied into the run at
// creation. It never consults the active catalog or the legacy preset resolver,
// so reloads can affect new runs without changing an in-flight run.
func executePolicySnapshot(ctx context.Context, in ExecuteWithModelFallbackInput) ExecuteAgentOutput {
	snapshot := policySnapshot(in.Run)
	if snapshot == nil {
		return ExecuteAgent(ctx, in.ExecuteAgentInput)
	}
	if len(snapshot.Candidates) == 0 {
		return ExecuteAgentOutput{ExecErr: domain.NewValidationError("policySnapshot.candidates", "persisted candidate sequence is empty")}
	}
	start := persistedPolicyCandidateIndex(in.Run, snapshot)
	if start < 0 || start >= len(snapshot.Candidates) {
		return ExecuteAgentOutput{ExecErr: domain.NewValidationError("policySnapshot.selectedIndex", "selected candidate index is out of range")}
	}

	var out ExecuteAgentOutput
	lastIndex := start
	lastCandidate := snapshot.Candidates[start]
	lastReason := "no persisted policy candidate was runnable"
	for index := start; index < len(snapshot.Candidates); index++ {
		candidate := snapshot.Candidates[index]
		lastIndex = index
		lastCandidate = candidate
		if reason := invalidPolicyCandidateReason(candidate); reason != "" {
			lastReason = reason
			emitPolicyCandidate(ctx, in.Deps, in.Run, snapshot, index, candidate, eventlog.PolicyCandidateOutcomeSkipped, reason, "candidate_invalid")
			continue
		}
		candidateRunner, reason := resolveCandidateRunner(ctx, in.Runner, in.Runners, candidate.RunnerType)
		if candidateRunner == nil {
			lastReason = reason
			emitPolicyCandidate(ctx, in.Deps, in.Run, snapshot, index, candidate, eventlog.PolicyCandidateOutcomeSkipped, reason, "runner_unavailable")
			continue
		}
		if reason := toolRestrictionCandidateReason(in.Run.ResolvedConfig, candidateRunner); reason != "" {
			lastReason = reason
			emitPolicyCandidate(ctx, in.Deps, in.Run, snapshot, index, candidate, eventlog.PolicyCandidateOutcomeSkipped, reason, "tool_restriction_unsupported")
			continue
		}

		applyPolicyCandidate(ctx, in.Deps, in.Run, candidate)
		emitPolicyCandidate(ctx, in.Deps, in.Run, snapshot, index, candidate, eventlog.PolicyCandidateOutcomeAttempted, "", "")
		attemptInput := in.ExecuteAgentInput
		attemptInput.Runner = candidateRunner
		out = ExecuteAgent(ctx, attemptInput)

		ce := classifyExecutionOutcome(candidateRunner, out.Result, out.ExecErr)
		reportHealth(in.ModelHealth, in.Run, out.Result, candidate.Model, ce)
		if ce == nil {
			recordActualModel(in.Run, candidate.Model)
			emitPolicyCandidate(ctx, in.Deps, in.Run, snapshot, index, candidate, eventlog.PolicyCandidateOutcomeSelected, "", "")
			return out
		}

		reason = modelFallbackReason(ce)
		lastReason = reason
		failureClass := "execution_failure"
		if ce.IsModelUnavailable() {
			failureClass = "model_unavailable"
		}
		emitPolicyCandidate(ctx, in.Deps, in.Run, snapshot, index, candidate, eventlog.PolicyCandidateOutcomeFailed, reason, failureClass)
		if !ce.IsModelUnavailable() {
			recordActualModel(in.Run, candidate.Model)
			return out
		}
	}

	recordActualModel(in.Run, lastCandidate.Model)
	emitPolicyCandidate(ctx, in.Deps, in.Run, snapshot, lastIndex, lastCandidate, eventlog.PolicyCandidateOutcomeExhausted, lastReason, "snapshot_exhausted")
	out.ExecErr = &domain.RunnerError{
		RunnerType: lastCandidate.RunnerType,
		Operation:  "policy_candidates",
		Cause: fmt.Errorf(
			"persisted policy candidates exhausted at index %d for catalog %s: %s",
			lastIndex,
			snapshot.CatalogDigest,
			lastReason,
		),
		IsTransient: false,
	}
	return out
}

// toolRestrictionCandidateReason re-checks the selected runner after policy
// fallback. A run may have been valid for its initial runner but become unsafe
// after a cross-runner candidate switch.
func toolRestrictionCandidateReason(cfg *domain.RunConfig, candidate runner.Runner) string {
	if cfg == nil || (len(cfg.AllowedTools) == 0 && len(cfg.DeniedTools) == 0) || candidate == nil {
		return ""
	}
	if cfg.ToolRestrictionPolicy.Effective() == domain.ToolRestrictionPolicyAdvisory || candidate.Capabilities().SupportsToolRestriction {
		return ""
	}
	declared := "allowedTools"
	if len(cfg.AllowedTools) == 0 {
		declared = "deniedTools"
	}
	return fmt.Sprintf("runner %q cannot enforce %s", candidate.Type(), declared)
}

// persistedPolicyCandidateIndex resumes at the candidate already copied into
// ResolvedConfig. applyPolicyCandidate persists that runner/model pair before
// launch, so a process restart retries the interrupted candidate (at-least-once)
// without replaying earlier candidates or consulting the active catalog.
func persistedPolicyCandidateIndex(run *domain.Run, snapshot *domain.ExecutionPolicySnapshot) int {
	if snapshot == nil {
		return -1
	}
	start := snapshot.SelectedIndex
	if run == nil || run.ResolvedConfig == nil || start < 0 || start >= len(snapshot.Candidates) {
		return start
	}
	for index := start; index < len(snapshot.Candidates); index++ {
		candidate := snapshot.Candidates[index]
		if candidate.RunnerType != run.ResolvedConfig.RunnerType {
			continue
		}
		switch candidate.SelectionType {
		case domain.ModelSelectionTypeModel:
			if candidate.Model == run.ResolvedConfig.Model {
				return index
			}
		case domain.ModelSelectionTypeRunnerDefault:
			if run.ResolvedConfig.Model == "" {
				return index
			}
		}
	}
	return start
}

func invalidPolicyCandidateReason(candidate domain.ExecutionCandidate) string {
	if !candidate.RunnerType.IsValid() {
		return "candidate runner is invalid"
	}
	switch candidate.SelectionType {
	case domain.ModelSelectionTypeModel:
		if strings.TrimSpace(candidate.Model) == "" {
			return "model candidate has no model identifier"
		}
	case domain.ModelSelectionTypeRunnerDefault:
		if strings.TrimSpace(candidate.Model) != "" {
			return "runner_default candidate unexpectedly contains a model identifier"
		}
	default:
		return "candidate selection type is invalid"
	}
	return ""
}

func policySnapshot(run *domain.Run) *domain.ExecutionPolicySnapshot {
	if run == nil || run.ResolvedConfig == nil {
		return nil
	}
	return run.ResolvedConfig.PolicySnapshot
}

func resolveCandidateRunner(ctx context.Context, current runner.Runner, registry runner.Registry, runnerType domain.RunnerType) (runner.Runner, string) {
	resolved := current
	if resolved == nil || resolved.Type() != runnerType {
		if registry == nil {
			return nil, "runner registry is not configured"
		}
		var err error
		resolved, err = registry.Get(runnerType)
		if err != nil || resolved == nil {
			if err != nil {
				return nil, err.Error()
			}
			return nil, "runner is not registered"
		}
	}
	available, message := resolved.IsAvailable(ctx)
	if !available {
		if strings.TrimSpace(message) == "" {
			message = "runner is unavailable"
		}
		return nil, message
	}
	return resolved, ""
}

func applyPolicyCandidate(ctx context.Context, deps Deps, run *domain.Run, candidate domain.ExecutionCandidate) {
	if run == nil || run.ResolvedConfig == nil {
		return
	}
	run.ResolvedConfig.RunnerType = candidate.RunnerType
	if candidate.SelectionType == domain.ModelSelectionTypeRunnerDefault {
		run.ResolvedConfig.Model = ""
	} else {
		run.ResolvedConfig.Model = candidate.Model
	}
	run.UpdatedAt = deps.Now()
	if deps.Runs != nil {
		if err := deps.Runs.Update(ctx, run); err != nil {
			EmitSystemEvent(ctx, deps, run.ID, "warn", "failed to persist policy candidate selection: "+err.Error())
		}
	}
}

func emitPolicyCandidate(ctx context.Context, deps Deps, run *domain.Run, snapshot *domain.ExecutionPolicySnapshot, index int, candidate domain.ExecutionCandidate, outcome eventlog.PolicyCandidateOutcome, reason, failureClass string) {
	if run == nil || snapshot == nil {
		return
	}
	EmitPolicyCandidateAttempt(ctx, deps, run.ID, eventlog.PolicyCandidateAttemptPayload{
		CatalogDigest:   snapshot.CatalogDigest,
		SnapshotIndex:   index,
		Runner:          string(candidate.RunnerType),
		SelectionType:   string(candidate.SelectionType),
		Model:           candidate.Model,
		ChallengerModel: candidate.ChallengerModel,
		CanaryArm:       candidate.CanaryArm,
		Outcome:         outcome,
		Reason:          reason,
		FailureClass:    failureClass,
	})
}

// modelFallbackReason renders the typed reason recorded on the
// classified error as the wire-form Reason on the fallback event. The
// ClassifiedError's stable Reason string (rate_limit, model_unknown,
// …) is the source of truth; we pass it through unmodified so stats /
// dashboards can group on it. Returns "" when there is no classified
// error (treated downstream as "no signal").
func modelFallbackReason(ce *fallback.ClassifiedError) string {
	if ce == nil {
		return ""
	}
	return string(ce.Reason)
}

// reportHealth maps the classified outcome to ModelHealthReporter
// updates. ce==nil means "no failure observed" — the run succeeded
// (or there was no signal at all, in which case we leave health
// unchanged rather than risk false-positive transitions).
func reportHealth(reporter ModelHealthReporter, run *domain.Run, result *runner.ExecuteResult, modelID string, ce *fallback.ClassifiedError) {
	if reporter == nil || modelID == "" {
		return
	}
	runnerType := ""
	if run != nil && run.ResolvedConfig != nil {
		runnerType = string(run.ResolvedConfig.RunnerType)
	}
	if runnerType == "" {
		return
	}
	switch {
	case ce == nil:
		if result != nil && result.Success {
			reporter.MarkModelHealthy(runnerType, modelID)
		}
	case ce.IsModelUnavailable():
		message := ce.Message
		if message == "" {
			message = "runtime classification: model unavailable"
		}
		reporter.MarkModelUnavailable(runnerType, modelID, message)
	}
}

// classifyExecutionOutcome converts an Execute outcome into a typed
// *fallback.ClassifiedError. Returns nil only on success; otherwise
// delegates to the runner's codec-aware classifier.
func classifyExecutionOutcome(r runner.Runner, result *runner.ExecuteResult, execErr error) *fallback.ClassifiedError {
	if result != nil && result.Success {
		return nil
	}
	stderr := ""
	exitCode := 0
	if result != nil {
		stderr = result.ErrorMessage
		exitCode = result.ExitCode
	}
	if stderr == "" && execErr != nil {
		stderr = execErr.Error()
	}
	if r == nil {
		return fallback.NewTextClassifier().Classify(fallback.ClassifyInput{
			Stderr:   stderr,
			ExitCode: exitCode,
			Cause:    execErr,
		})
	}
	return r.Classify(stderr, exitCode)
}

func recordActualModel(run *domain.Run, model string) {
	if run == nil {
		return
	}
	run.ActualModel = model
}

func currentModel(run *domain.Run) string {
	if run == nil || run.ResolvedConfig == nil {
		return ""
	}
	return run.ResolvedConfig.Model
}

// PrepareTranscriptInput is the explicit input to PrepareTranscriptConfig.
type PrepareTranscriptInput struct {
	Deps          Deps
	Run           *domain.Run
	RunStateRoot  string
	RunStateWrite func()
	WorkingDir    string
	Mu            *sync.Mutex
	Existing      *runstate.State
}

// PrepareTranscriptConfig opens (or reuses) the runstate for this run and
// returns a runner.TranscriptConfig wired with the persisting callbacks
// the runner uses to checkpoint progress.
func PrepareTranscriptConfig(ctx context.Context, in PrepareTranscriptInput) (*runner.TranscriptConfig, *runstate.State, error) {
	if in.Run == nil || in.Run.ResolvedConfig == nil {
		return nil, in.Existing, nil
	}
	state := in.Existing
	if state == nil {
		startedAt := in.Deps.Now().UTC()
		if in.Run.StartedAt != nil {
			startedAt = in.Run.StartedAt.UTC()
		}
		s, err := runstate.Open(in.Run.ID, runstate.OpenOptions{
			RootDir:    in.RunStateRoot,
			RunnerType: in.Run.ResolvedConfig.RunnerType,
			WorkingDir: in.WorkingDir,
			StartedAt:  startedAt,
			OnWrite:    in.RunStateWrite,
		})
		if err != nil {
			return nil, nil, err
		}
		state = s
		snap := state.Snapshot()
		lockedMutate(in.Mu, func() {
			in.Run.TranscriptPath = snap.TranscriptPath
			in.Run.TranscriptCursor = snap.Cursor.TranscriptCursor
			in.Run.TranscriptLastSeq = snap.Cursor.TranscriptLastSeq
		})
		if in.Deps.Runs != nil {
			if err := in.Deps.Runs.Update(ctx, in.Run); err != nil {
				return nil, nil, err
			}
		}
	}

	cfg := &runner.TranscriptConfig{
		TranscriptPath: in.Run.TranscriptPath,
		StderrPath:     state.Snapshot().StderrPath,
		StdoutFile:     state.TranscriptWriter(),
		StderrFile:     state.StderrWriter(),
		OnProcessStart: func(pid, pgid int) error {
			lockedMutate(in.Mu, func() {
				in.Run.RunnerPID = pid
				in.Run.RunnerPGID = pgid
			})
			if err := state.PersistProcess(pid, pgid); err != nil {
				return err
			}
			if in.Deps.Runs != nil {
				_, err := in.Deps.Runs.UpdateRunnerStreamState(context.Background(), in.Run)
				return err
			}
			return nil
		},
		OnAdvance: func(cursor, lastSeq int64) error {
			lockedMutate(in.Mu, func() {
				if cursor > in.Run.TranscriptCursor {
					in.Run.TranscriptCursor = cursor
				}
				if lastSeq > in.Run.TranscriptLastSeq {
					in.Run.TranscriptLastSeq = lastSeq
				}
			})
			if err := state.PersistCursor(in.Run.TranscriptCursor, in.Run.TranscriptLastSeq); err != nil {
				return err
			}
			if in.Deps.Runs != nil {
				_, err := in.Deps.Runs.UpdateRunnerStreamState(context.Background(), in.Run)
				return err
			}
			return nil
		},
		OnSessionID: func(sessionID string) error {
			same := false
			lockedMutate(in.Mu, func() {
				if in.Run.SessionID == sessionID {
					same = true
					return
				}
				in.Run.SessionID = sessionID
			})
			if same {
				return nil
			}
			if err := state.PersistSessionID(sessionID); err != nil {
				return err
			}
			if in.Deps.Runs != nil {
				_, err := in.Deps.Runs.UpdateRunnerStreamState(context.Background(), in.Run)
				return err
			}
			return nil
		},
	}
	return cfg, state, nil
}

func lockedMutate(mu *sync.Mutex, fn func()) {
	if mu != nil {
		mu.Lock()
		defer mu.Unlock()
	}
	fn()
}
