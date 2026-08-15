package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/logx"
	"github.com/vrooli/vrooli/internal/ports"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/shell"
)

type PhaseExecutionStatus string

const (
	PhaseExecutionCompleted PhaseExecutionStatus = "completed"
	PhaseExecutionSkipped   PhaseExecutionStatus = "skipped"
	PhaseExecutionUndefined PhaseExecutionStatus = "undefined"
)

type PhaseResult struct {
	Scenario      string               `json:"scenario"`
	Phase         string               `json:"phase"`
	Defined       bool                 `json:"defined"`
	Status        PhaseExecutionStatus `json:"status"`
	ExecutedSteps int                  `json:"executed_steps"`
	SkippedSteps  int                  `json:"skipped_steps"`
	// Run-lifecycle bookkeeping, populated by RunPhaseDetailed so callers can
	// build a typed run result / persist a run record. RunID is the same id
	// written to the lifecycle log markers. ExitCode/Failed reflect the phase
	// step outcome (0 on success). LogFile is the scenario lifecycle log path.
	RunID     string    `json:"run_id,omitempty"`
	StartedAt time.Time `json:"started_at,omitempty"`
	EndedAt   time.Time `json:"ended_at,omitempty"`
	ExitCode  int       `json:"exit_code"`
	LogFile   string    `json:"log_file,omitempty"`
}

type PhaseStepError struct {
	Scenario string
	Phase    string
	Step     string
	LogPath  string
	Exit     int
	Err      error
}

func (e *PhaseStepError) Error() string {
	if e == nil {
		return ""
	}

	context := fmt.Sprintf("scenario %q phase %q", e.Scenario, e.Phase)
	if strings.TrimSpace(e.Step) != "" {
		context += fmt.Sprintf(" step %q", e.Step)
	}
	if e.Exit > 0 {
		context += fmt.Sprintf(" failed with exit code %d", e.Exit)
	} else {
		context += " failed"
	}
	if strings.TrimSpace(e.LogPath) != "" {
		context += fmt.Sprintf(" (log: %s)", e.LogPath)
	}
	if e.Err != nil {
		return context + ": " + e.Err.Error()
	}
	return context
}

func (e *PhaseStepError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (e *PhaseStepError) ExitCode() int {
	if e == nil {
		return 1
	}
	if e.Exit > 0 {
		return e.Exit
	}
	return extractExitCode(e.Err)
}

func (r *Runner) RunPhase(name, phaseName string, opts PhaseOptions) error {
	_, err := r.RunPhaseDetailed(name, phaseName, opts)
	return err
}

func (r *Runner) RunPhaseDetailed(name, phaseName string, opts PhaseOptions) (PhaseResult, error) {
	r.logInfo("Scenario phase requested", logx.AttrScenario, name, logx.AttrPhase, phaseName, logx.AttrProjectMode, opts.ProjectMode)
	item, err := r.loadScenario(name, opts.CustomPath)
	if err != nil {
		r.logError("Failed to load scenario for phase", err, logx.AttrScenario, name, logx.AttrPhase, phaseName)
		return PhaseResult{}, err
	}
	release, err := r.acquireScenarioLock(item.Slug)
	if err != nil {
		wrapped := fmt.Errorf("scenario %q phase %q blocked by concurrent lifecycle operation: %w", item.Slug, phaseName, err)
		r.logError("Scenario phase blocked by concurrent lifecycle operation", wrapped,
			logx.AttrScenario, item.Slug,
			logx.AttrPhase, phaseName,
			logx.AttrOperation, "phase",
		)
		return PhaseResult{}, wrapped
	}
	defer release()

	if phaseRequiresBootstrap(phaseName) {
		ready := make(map[string]struct{})
		setupCache := make(setupCheckCache)
		bootstrapOpts := StartOptions{CustomPath: opts.CustomPath}
		if _, _, err := r.bootstrapScenarioDependencies(item, bootstrapOpts, ready, setupCache, nil); err != nil {
			return PhaseResult{}, err
		}
	}

	var envResult ports.Environment
	if opts.ProjectMode {
		envResult, err = r.Ports.BuildProjectEnvironment(item)
		if err != nil {
			return PhaseResult{}, err
		}
	} else {
		envResult, err = r.prepareScenarioEnvironment(item, disabledRuntimeRegistrySession())
		if err != nil {
			return PhaseResult{}, err
		}
	}

	env := make(map[string]string, len(envResult.EnvVars)+3)
	for key, value := range envResult.EnvVars {
		env[key] = value
	}
	if opts.ManageRuntime {
		env["TEST_MANAGE_RUNTIME"] = "true"
	} else if opts.AllowSkipMissingRuntime {
		env["TEST_ALLOW_SKIP_MISSING_RUNTIME"] = "true"
	}
	if strings.TrimSpace(os.Getenv("GOWORK")) == "" {
		mode := strings.ToLower(strings.TrimSpace(os.Getenv("VROOLI_SCENARIO_GOWORK")))
		if mode != "on" && mode != "auto" {
			env["GOWORK"] = "off"
		}
	}

	args := append([]string(nil), opts.Args...)
	var result PhaseResult
	logPath, _ := process.ScenarioLifecycleLogPath(r.Home, item.Slug)
	meta, runErr := r.runWithLifecycleLog(lifecycleLogContext{Scenario: item.Slug, Operation: "phase", Phase: phaseName, RunID: strings.TrimSpace(opts.RunID)}, func(logWriter, childWriter io.Writer) error {
		var executeErr error
		result, executeErr = r.ExecutePhaseDetailed(item, phaseName, env, args, logWriter, childWriter)
		return executeErr
	})
	result.RunID = meta.RunID
	result.StartedAt = meta.StartedAt
	result.EndedAt = meta.EndedAt
	result.LogFile = logPath
	result.ExitCode = extractExitCode(runErr)
	if runErr != nil {
		r.logError("Scenario phase failed", runErr, logx.AttrScenario, item.Slug, logx.AttrPhase, phaseName)
		return result, runErr
	}
	r.logInfo("Scenario phase completed",
		logx.AttrScenario, item.Slug,
		logx.AttrPhase, phaseName,
		logx.AttrStatus, result.Status,
		"executed_steps", result.ExecutedSteps,
		"skipped_steps", result.SkippedSteps,
	)
	return result, nil
}

func phaseRequiresBootstrap(phaseName string) bool {
	switch strings.TrimSpace(phaseName) {
	case "develop", "test":
		return true
	default:
		return false
	}
}

func (r *Runner) ExecutePhase(item scenario.Scenario, phaseName string, env map[string]string, args []string, logWriter io.Writer) error {
	_, err := r.ExecutePhaseDetailed(item, phaseName, env, args, logWriter, logWriter)
	return err
}

// ExecutePhaseDetailed runs a phase's steps. logWriter receives orchestrator
// messages (infof/warnf headers, slog text if routed there); childWriter
// receives raw child-process stdout for foreground steps. Callers that do
// not need to split the two can pass the same writer for both
// (tests/io.Discard). Production callers should provide a logWriter that
// tees to the scenario lifecycle log file plus the console at the current
// verbosity, and a childWriter that tees to the log file plus the console
// only at verbose — see runWithLifecycleLog.
func (r *Runner) ExecutePhaseDetailed(item scenario.Scenario, phaseName string, env map[string]string, args []string, logWriter io.Writer, childWriter io.Writer) (PhaseResult, error) {
	if childWriter == nil {
		childWriter = logWriter
	}
	phase, ok := lookupPhase(item.Manifest, phaseName)
	if !ok {
		r.logDebug("Scenario phase not defined", logx.AttrScenario, item.Slug, logx.AttrPhase, phaseName)
		return undefinedPhaseResult(item.Slug, phaseName), nil
	}
	if !phaseDefined(phase) {
		r.logDebug("Scenario phase empty; treating as undefined", logx.AttrScenario, item.Slug, logx.AttrPhase, phaseName)
		return undefinedPhaseResult(item.Slug, phaseName), nil
	}
	if phaseName == "setup" {
		if err := r.provisionSharedPackages(item, env, logWriter, childWriter); err != nil {
			return PhaseResult{}, err
		}
	}

	result := PhaseResult{
		Scenario: item.Slug,
		Phase:    phaseName,
		Defined:  true,
		Status:   PhaseExecutionSkipped,
	}
	lifecycleLogPath, err := process.ScenarioLifecycleLogPath(r.Home, item.Slug)
	if err != nil {
		return result, err
	}
	for index, step := range phase.Steps {
		if strings.TrimSpace(step.Run) == "" {
			continue
		}
		ok, reason, err := stepConditionsMet(item, step.Condition, env)
		if err != nil {
			return result, err
		}
		if !ok {
			r.infof(logWriter, "[%d/%d] Skipping %s - %s", index+1, len(phase.Steps), step.Name, reason)
			r.logDebug("Skipping lifecycle step",
				logx.AttrScenario, item.Slug,
				logx.AttrPhase, phaseName,
				logx.AttrStep, step.Name,
				"reason", reason,
			)
			result.SkippedSteps++
			continue
		}

		r.infof(logWriter, "[%d/%d] %s", index+1, len(phase.Steps), step.Name)
		r.logDebug("Executing lifecycle step",
			logx.AttrScenario, item.Slug,
			logx.AttrPhase, phaseName,
			logx.AttrStep, step.Name,
			"background", step.Background,
		)

		finalCmd := step.Run
		if phaseName == "test" {
			if len(args) > 0 && isTestGenieExecuteCommand(finalCmd) {
				quotedArgs := make([]string, 0, len(args))
				for _, arg := range args {
					quotedArgs = append(quotedArgs, shellQuote(arg))
				}
				finalCmd += " " + strings.Join(quotedArgs, " ")
			}
			finalCmd = injectTestGenieTestFlags(finalCmd)
		}

		if step.Background {
			if err := r.startTrackedProcess(item, phaseName, step, env); err != nil {
				return result, err
			}
			result.ExecutedSteps++
			result.Status = PhaseExecutionCompleted
			continue
		}

		sink := newStepSink(childWriter)
		stepErr := r.runForegroundStep(item, phaseName, finalCmd, env, sink)
		sink.Flush()
		if stepErr != nil {
			if phaseName == "stop" {
				r.warnf(logWriter, "Stop step completed with non-zero exit: %s", step.Name)
				r.logWarn("Stop lifecycle step returned non-zero exit but execution will continue",
					logx.AttrScenario, item.Slug,
					logx.AttrPhase, phaseName,
					logx.AttrStep, step.Name,
				)
				result.ExecutedSteps++
				result.Status = PhaseExecutionCompleted
				continue
			}
			// Replay the tail of the failing step to stderr so users never
			// have to rerun with --verbose just to see the compiler/linker
			// error. Skip replay on context.Canceled (the user interrupted
			// — the log file already has everything).
			if r.Verbosity != VerbosityVerbose && !errors.Is(stepErr, context.Canceled) && r.Err != nil {
				sink.ReplayTo(r.Err, item.Slug+" "+phaseName+" "+step.Name, lifecycleLogPath)
			}
			return result, newPhaseStepError(item.Slug, phaseName, step.Name, lifecycleLogPath, stepErr)
		}
		result.ExecutedSteps++
		result.Status = PhaseExecutionCompleted
	}

	return result, nil
}

// injectTestGenieTestFlags rewrites a lifecycle `test-genie execute` test step to
// carry the one flag the lifecycle path requires, idempotently:
//
//   - `--auto-start` (a global test-genie flag, before the subcommand): the
//     lifecycle owns the target scenario's runtime, so the suite may auto-start
//     surfaces it needs.
//
// The server owns the run and `execute` owns foreground/background policy. A
// caller that wants inline blocking can pass `--wait` through `vrooli scenario
// test`; the wrapper does not force it.
func injectTestGenieTestFlags(command string) string {
	fields := strings.Fields(command)

	binIdx, execIdx := testGenieExecuteIndexes(fields)
	if binIdx < 0 || execIdx < 0 {
		return command
	}

	hasAutoStart := false
	for i := binIdx + 1; i < execIdx; i++ {
		if fields[i] == "--auto-start" || strings.HasPrefix(fields[i], "--auto-start=") {
			hasAutoStart = true
		}
	}
	if hasAutoStart {
		return command
	}

	// Splice --auto-start (a test-genie GLOBAL flag) into the binary..execute
	// span, preserving the possibly-quoted tail byte-for-byte. The span replace
	// touches only the binary + any existing global flags + the `execute` token,
	// leaving the scenario positional and the rest of the command untouched.
	span := strings.Join(fields[binIdx:execIdx+1], " ")
	rebuilt := fields[binIdx]
	for i := binIdx + 1; i < execIdx; i++ {
		rebuilt += " " + fields[i]
	}
	if !hasAutoStart {
		rebuilt += " --auto-start"
	}
	rebuilt += " execute"
	return strings.Replace(command, span, rebuilt, 1)
}

func isTestGenieExecuteCommand(command string) bool {
	fields := strings.Fields(command)
	_, execIdx := testGenieExecuteIndexes(fields)
	return execIdx >= 0
}

func testGenieExecuteIndexes(fields []string) (int, int) {
	// Locate the test-genie binary (the first token that is not an env
	// assignment).
	binIdx := -1
	for i, f := range fields {
		if strings.Contains(f, "=") && !strings.HasPrefix(f, "-") {
			continue
		}
		binIdx = i
		break
	}
	if binIdx < 0 || filepath.Base(fields[binIdx]) != "test-genie" {
		return -1, -1
	}

	// Locate the `execute` subcommand, allowing global flags (e.g. an existing
	// --auto-start) between the binary and the subcommand.
	execIdx := -1
	for i := binIdx + 1; i < len(fields); i++ {
		if fields[i] == "execute" {
			execIdx = i
			break
		}
		if !strings.HasPrefix(fields[i], "-") {
			break // a positional before `execute` — not a test-genie execute command
		}
	}
	if execIdx < 0 {
		return -1, -1
	}
	return binIdx, execIdx
}

func (r *Runner) startTrackedProcess(item scenario.Scenario, phase string, step scenario.PhaseStep, env map[string]string) error {
	// Record/log directories are keyed by the instance record slug so two
	// variants running the same step (e.g. "develop") never overwrite each
	// other's record/PID/log files. Live ⇒ bare slug (unchanged). See §1a / P1.
	slug := recordSlug(item)
	processID := fmt.Sprintf("vrooli.%s.%s.%s", phase, slug, step.Name)
	logDir, err := process.ScenarioLogsDir(r.Home, slug)
	if err != nil {
		return err
	}
	if _, err := config.EnsureOwnedDir(logDir); err != nil {
		return err
	}
	logFile := filepath.Join(logDir, processID+".log")
	if _, err := os.Stat(logFile); err == nil {
		_ = os.Rename(logFile, logFile+".bak")
	}

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return newPhaseStepError(item.Slug, phase, step.Name, logFile, err)
	}
	defer file.Close()

	sourceDir, err := r.effectiveSourceDir(item)
	if err != nil {
		return newPhaseStepError(item.Slug, phase, step.Name, logFile, err)
	}

	stepEnv := lifecycleStepEnv(phase, env)
	stepEnv = setEnvValue(stepEnv, "VROOLI_PROCESS_ID", processID)
	stepEnv = setEnvValue(stepEnv, "VROOLI_PHASE", phase)
	stepEnv = setEnvValue(stepEnv, "VROOLI_SCENARIO", item.Slug)
	stepEnv = setEnvValue(stepEnv, "VROOLI_STEP", step.Name)
	stepEnv = setEnvValue(stepEnv, "VROOLI_LIFECYCLE_MANAGED", "true")

	// Scenario lifecycle steps remain shell-defined by the service.json contract.
	// Week 6 removes project-level Bash orchestration, but scenario steps
	// intentionally continue to run as user-authored shell commands.
	cmd := shell.BashCommand(step.Run, shell.Spec{
		Dir:    sourceDir,
		Env:    stepEnv,
		Stdin:  strings.NewReader(""),
		Stdout: file,
		Stderr: file,
	})
	if err := platform.ConfigureCommand(cmd, platform.ProcessOptions{Detached: true}); err != nil {
		return newPhaseStepError(item.Slug, phase, step.Name, logFile, err)
	}

	if err := cmd.Start(); err != nil {
		return newPhaseStepError(item.Slug, phase, step.Name, logFile, err)
	}

	port := inferStepPort(item.Manifest, step.Name, env)
	record := process.Record{
		PID:        cmd.Process.Pid,
		PGID:       cmd.Process.Pid,
		ProcessID:  processID,
		Phase:      phase,
		Scenario:   item.Slug,
		Step:       step.Name,
		Command:    step.Run,
		WorkingDir: sourceDir,
		LogFile:    logFile,
		Port:       port,
		StartedAt:  time.Now().UTC(),
		Status:     "running",
	}
	if err := process.WriteScenarioRecord(r.Home, slug, step.Name, record); err != nil {
		_ = cmd.Process.Kill()
		return newPhaseStepError(item.Slug, phase, step.Name, logFile, err)
	}
	if err := recordRuntimeProcessRef(context.Background(), r.runtimeDeps(), r.Home, env, record); err != nil {
		_ = cmd.Process.Kill()
		_ = process.RemoveScenarioRecord(r.Home, slug, step.Name)
		return newPhaseStepError(item.Slug, phase, step.Name, logFile, err)
	}

	r.runtimeDeps().sleep(200 * time.Millisecond)
	if !platform.IsPIDRunning(cmd.Process.Pid) {
		record.Status = "failed"
		_ = process.WriteScenarioRecord(r.Home, slug, step.Name, record)
		return newPhaseStepError(
			item.Slug,
			phase,
			step.Name,
			logFile,
			fmt.Errorf("background step exited immediately: %w", err),
		)
	}
	return nil
}

func (r *Runner) runForegroundStep(item scenario.Scenario, phase, command string, env map[string]string, logWriter io.Writer) error {
	sourceDir, err := r.effectiveSourceDir(item)
	if err != nil {
		return err
	}

	stepEnv := lifecycleStepEnv(phase, env)

	// Scenario lifecycle steps remain shell-defined by the service.json contract.
	cmd := shell.BashCommand(command, shell.Spec{
		Dir:    sourceDir,
		Env:    stepEnv,
		Stdin:  strings.NewReader(""),
		Stdout: logWriter,
		Stderr: logWriter,
	})
	return cmd.Run()
}

// lifecycleStepEnv makes setup and test phases safe for server-owned
// execution. Those phases may invoke package managers and other tools that
// prompt when they detect stale state; a lifecycle request has no interactive
// stdin to answer such prompts, so fail deterministically instead.
func lifecycleStepEnv(phase string, overrides map[string]string) []string {
	stepEnv := mergeEnv(os.Environ(), overrides)
	stepEnv = setEnvValue(stepEnv, "LIFECYCLE_PHASE", phase)
	stepEnv = setEnvValue(stepEnv, "VROOLI_LIFECYCLE_MANAGED", "true")
	if phase == "setup" || phase == "test" {
		stepEnv = setEnvValue(stepEnv, "CI", "true")
		stepEnv = setEnvValue(stepEnv, "VROOLI_LIFECYCLE_NONINTERACTIVE", "true")
	}
	return stepEnv
}

// runWithLifecycleLog opens the scenario lifecycle log file and invokes fn
// with two writers: logWriter is for orchestrator-level output (slog text
// and [INFO]/[WARNING] step headers) — it tees the log file and the
// console, gated by the current verbosity; childWriter is for raw tool
// stdout (vite/pnpm) — it tees the log file and reaches the console only
// at VerbosityVerbose. The log file always receives everything.
func (r *Runner) runWithLifecycleLog(ctx lifecycleLogContext, fn func(logWriter, childWriter io.Writer) error) (RunMeta, error) {
	if strings.TrimSpace(ctx.Scenario) == "" {
		ctx.Scenario = "unknown"
	}
	if strings.TrimSpace(ctx.Operation) == "" {
		ctx.Operation = "start"
	}
	if strings.TrimSpace(ctx.Phase) == "" {
		ctx.Phase = "unknown"
	}
	path, err := process.ScenarioLifecycleLogPath(r.Home, ctx.Scenario)
	if err != nil {
		return RunMeta{}, err
	}
	if _, err := config.EnsureOwnedDir(filepath.Dir(path)); err != nil {
		return RunMeta{}, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return RunMeta{}, err
	}
	defer file.Close()
	_ = config.ChownToInvokingUser(path)

	logWriter := io.MultiWriter(r.consoleOut(), file)
	childWriter := io.MultiWriter(r.childStdoutConsole(), file)
	startedAt := time.Now().UTC()
	runID := strings.TrimSpace(ctx.RunID)
	if runID == "" {
		runID = lifecycleRunID(startedAt)
	}
	writeLifecycleRunStart(file, ctx, runID, startedAt)
	err = fn(logWriter, childWriter)
	endedAt := time.Now().UTC()
	writeLifecycleRunEnd(file, ctx, runID, startedAt, endedAt, err)
	return RunMeta{RunID: runID, StartedAt: startedAt, EndedAt: endedAt}, err
}

func startLifecycleLogContext(scenarioName, operation, phase string) lifecycleLogContext {
	if strings.TrimSpace(operation) == "" {
		operation = "start"
	}
	return lifecycleLogContext{Scenario: scenarioName, Operation: operation, Phase: phase}
}

func lifecycleRunID(t time.Time) string {
	return fmt.Sprintf("%s-%d", t.Format("20060102-150405.000000000"), os.Getpid())
}

func writeLifecycleRunStart(w io.Writer, ctx lifecycleLogContext, runID string, startedAt time.Time) {
	cwd, _ := os.Getwd()
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "=== VROOLI LIFECYCLE RUN START ===")
	_, _ = fmt.Fprintf(w, "run_id: %s\n", runID)
	_, _ = fmt.Fprintf(w, "scenario: %s\n", ctx.Scenario)
	_, _ = fmt.Fprintf(w, "operation: %s\n", ctx.Operation)
	_, _ = fmt.Fprintf(w, "phase: %s\n", ctx.Phase)
	_, _ = fmt.Fprintf(w, "started_at: %s\n", startedAt.Format(time.RFC3339Nano))
	_, _ = fmt.Fprintf(w, "pid: %d\n", os.Getpid())
	if cwd != "" {
		_, _ = fmt.Fprintf(w, "cwd: %s\n", cwd)
	}
	_, _ = fmt.Fprintln(w, "==================================")
}

func writeLifecycleRunEnd(w io.Writer, ctx lifecycleLogContext, runID string, startedAt, endedAt time.Time, err error) {
	status := "completed"
	if err != nil {
		status = "failed"
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "=== VROOLI LIFECYCLE RUN END ===")
	_, _ = fmt.Fprintf(w, "run_id: %s\n", runID)
	_, _ = fmt.Fprintf(w, "scenario: %s\n", ctx.Scenario)
	_, _ = fmt.Fprintf(w, "operation: %s\n", ctx.Operation)
	_, _ = fmt.Fprintf(w, "phase: %s\n", ctx.Phase)
	_, _ = fmt.Fprintf(w, "status: %s\n", status)
	if err != nil {
		var phaseErr *PhaseStepError
		if errors.As(err, &phaseErr) {
			if strings.TrimSpace(phaseErr.Step) != "" {
				_, _ = fmt.Fprintf(w, "step: %s\n", phaseErr.Step)
			}
			if exit := phaseErr.ExitCode(); exit > 0 {
				_, _ = fmt.Fprintf(w, "exit_code: %d\n", exit)
			}
		}
		_, _ = fmt.Fprintf(w, "error: %s\n", firstErrorLine(err))
	}
	_, _ = fmt.Fprintf(w, "ended_at: %s\n", endedAt.Format(time.RFC3339Nano))
	_, _ = fmt.Fprintf(w, "duration: %s\n", endedAt.Sub(startedAt).Round(time.Millisecond))
	_, _ = fmt.Fprintln(w, "================================")
}

func firstErrorLine(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	if idx := strings.IndexByte(msg, '\n'); idx >= 0 {
		msg = strings.TrimSpace(msg[:idx])
	}
	return msg
}

func lookupPhase(manifest scenario.ServiceManifest, phaseName string) (scenario.Phase, bool) {
	switch phaseName {
	case "setup":
		return manifest.Lifecycle.Setup, true
	case "develop":
		return manifest.Lifecycle.Develop, true
	case "build":
		return manifest.Lifecycle.Build, true
	case "deploy":
		return manifest.Lifecycle.Deploy, true
	case "clean":
		return manifest.Lifecycle.Clean, true
	case "test":
		return manifest.Lifecycle.Test, true
	case "backup":
		return manifest.Lifecycle.Backup, true
	case "restore":
		return manifest.Lifecycle.Restore, true
	case "production":
		return manifest.Lifecycle.Production, true
	case "stop":
		return manifest.Lifecycle.Stop, true
	default:
		return scenario.Phase{}, false
	}
}

func (r *Runner) infof(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, "[INFO]    "+format+"\n", args...)
}

func (r *Runner) warnf(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, "[WARNING] "+format+"\n", args...)
}

func undefinedPhaseResult(scenarioName, phaseName string) PhaseResult {
	return PhaseResult{
		Scenario: scenarioName,
		Phase:    phaseName,
		Status:   PhaseExecutionUndefined,
	}
}

func phaseDefined(phase scenario.Phase) bool {
	return strings.TrimSpace(phase.Description) != "" || phase.Condition != nil || len(phase.Steps) > 0
}

func newPhaseStepError(scenarioName, phaseName, stepName, logPath string, err error) error {
	return &PhaseStepError{
		Scenario: scenarioName,
		Phase:    phaseName,
		Step:     stepName,
		LogPath:  logPath,
		Exit:     extractExitCode(err),
		Err:      err,
	}
}

func extractExitCode(err error) int {
	if err == nil {
		return 0
	}
	var withCode interface{ ExitCode() int }
	if errors.As(err, &withCode) {
		return withCode.ExitCode()
	}
	return 0
}
