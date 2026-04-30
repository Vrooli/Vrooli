// Runner execution + model-level fallback chain walk.
//
// ExecuteAgent runs the agent through runner.Execute, then validates the
// outcome and reports the result + execErr. ExecuteWithModelFallback wraps
// it with the preset-chain walk: when the runner rejects the current model
// with a classified "unavailable" error, advance to the next chain entry
// and retry inside the same Run. The loop is capped at the chain length
// to guarantee termination even if the classifier is overly permissive.
//
// Mutates Run.ActualModel, Run.Status (to Running), Run.SessionID, and the
// runtime callbacks (transcript cursor, runner pid/pgid, session id).

package phases

import (
	"context"
	"fmt"
	"sync"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/modelregistry"
	"agent-manager/internal/runstate"

	"github.com/google/uuid"
)

// ExecuteAgentInput is the explicit input to ExecuteAgent.
type ExecuteAgentInput struct {
	Deps         Deps
	Run          *domain.Run
	Task         *domain.Task
	Profile      *domain.AgentProfile
	Runner       runner.Runner
	WorkingDir   string
	SandboxID    *uuid.UUID
	Prompt       string
	SystemPrompt string
	Attachments  []runner.Attachment
	EnvVars      map[string]string
	EventSink    runner.EventSink
	RunState     *runstate.State
	Mu           *sync.Mutex
	ModelHealth  ModelHealthReporter
	ModelChains  ModelChainResolver

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
	in.Run.UpdatedAt = time.Now()
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

	transcriptCfg, runState, err := PrepareTranscriptConfig(ctx, PrepareTranscriptInput{
		Deps:       in.Deps,
		Run:        in.Run,
		WorkingDir: in.WorkingDir,
		Mu:         in.Mu,
		Existing:   in.RunState,
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
		SystemPrompt:   in.SystemPrompt,
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

// ExecuteWithModelFallbackInput is the explicit input.
type ExecuteWithModelFallbackInput struct {
	ExecuteAgentInput
}

// ExecuteWithModelFallback walks the preset chain on model-unavailable
// errors. On any non-model failure (or on success) it returns immediately.
// The first outcome that is not a model-unavailable error determines
// Run.ActualModel.
func ExecuteWithModelFallback(ctx context.Context, in ExecuteWithModelFallbackInput) ExecuteAgentOutput {
	chain := resolveModelFallbackChain(in.ModelChains, in.Run)
	if len(chain) == 0 {
		out := ExecuteAgent(ctx, in.ExecuteAgentInput)
		recordActualModel(in.Run, currentModel(in.Run))
		return out
	}

	var out ExecuteAgentOutput
	for attempt := 0; attempt < len(chain); attempt++ {
		model := chain[attempt]
		applyModelForAttempt(ctx, in.Deps, in.Run, model, attempt, chain)
		out = ExecuteAgent(ctx, in.ExecuteAgentInput)

		kind := classifyExecutionOutcome(in.Run, out.Result, out.ExecErr)
		reportHealth(in.ModelHealth, in.Run, out.Result, model, kind)
		if kind != runner.ModelErrorUnavailable {
			recordActualModel(in.Run, model)
			return out
		}

		if attempt == len(chain)-1 {
			recordActualModel(in.Run, model)
			EmitSystemEvent(ctx, in.Deps, in.Run.ID, "warn",
				"model fallback exhausted — all entries in preset chain were rejected")
			return out
		}

		next := chain[attempt+1]
		EmitSystemEvent(ctx, in.Deps, in.Run.ID, "warn", fmt.Sprintf(
			"model fallback: %s -> %s (runner rejected model)",
			describeModel(model), describeModel(next),
		))
	}
	return out
}

func reportHealth(reporter ModelHealthReporter, run *domain.Run, result *runner.ExecuteResult, modelID string, kind runner.ModelErrorKind) {
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
	switch kind {
	case runner.ModelErrorUnavailable:
		message := "runtime classification: model unavailable"
		if result != nil && result.ErrorMessage != "" {
			message = result.ErrorMessage
		}
		reporter.MarkModelUnavailable(runnerType, modelID, message)
	case runner.ModelErrorNone:
		if result != nil && result.Success {
			reporter.MarkModelHealthy(runnerType, modelID)
		}
	}
}

func resolveModelFallbackChain(resolver ModelChainResolver, run *domain.Run) modelregistry.PresetChain {
	if resolver == nil || run == nil || run.ResolvedConfig == nil {
		return nil
	}
	cfg := run.ResolvedConfig
	if cfg.ModelPreset == domain.ModelPresetUnspecified {
		return nil
	}
	chain, ok := resolver.ResolvePreset(string(cfg.RunnerType), string(cfg.ModelPreset))
	if !ok || len(chain) == 0 {
		return nil
	}
	return chain
}

func applyModelForAttempt(ctx context.Context, deps Deps, run *domain.Run, model string, attempt int, chain modelregistry.PresetChain) {
	if run == nil || run.ResolvedConfig == nil {
		return
	}
	if run.ResolvedConfig.Model == model {
		return
	}
	run.ResolvedConfig.Model = model
	if attempt > 0 {
		EmitSystemEvent(ctx, deps, run.ID, "info", fmt.Sprintf(
			"model attempt %d/%d: %s",
			attempt+1, len(chain), describeModel(model),
		))
	}
}

func classifyExecutionOutcome(run *domain.Run, result *runner.ExecuteResult, execErr error) runner.ModelErrorKind {
	if result != nil && result.Success {
		return runner.ModelErrorNone
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
	runnerType := domain.RunnerTypeClaudeCode
	if run != nil && run.ResolvedConfig != nil {
		runnerType = run.ResolvedConfig.RunnerType
	}
	return runner.ClassifyModelError(runnerType, stderr, exitCode)
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

func describeModel(model string) string {
	if model == "" {
		return "<runner default>"
	}
	return model
}

// PrepareTranscriptInput is the explicit input to PrepareTranscriptConfig.
type PrepareTranscriptInput struct {
	Deps       Deps
	Run        *domain.Run
	WorkingDir string
	Mu         *sync.Mutex
	Existing   *runstate.State
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
		startedAt := time.Now().UTC()
		if in.Run.StartedAt != nil {
			startedAt = in.Run.StartedAt.UTC()
		}
		s, err := runstate.Open(in.Run.ID, runstate.OpenOptions{
			RunnerType: in.Run.ResolvedConfig.RunnerType,
			WorkingDir: in.WorkingDir,
			StartedAt:  startedAt,
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
				return in.Deps.Runs.Update(context.Background(), in.Run)
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
				return in.Deps.Runs.Update(context.Background(), in.Run)
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
				return in.Deps.Runs.Update(context.Background(), in.Run)
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
