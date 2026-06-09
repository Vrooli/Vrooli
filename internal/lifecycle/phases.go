package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

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

	if phaseRequiresBootstrap(phaseName) {
		ready := make(map[string]struct{})
		bootstrapOpts := StartOptions{CustomPath: opts.CustomPath}
		if _, _, err := r.bootstrapScenarioDependencies(item, bootstrapOpts, ready, nil); err != nil {
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
	if err := r.runWithLifecycleLog(lifecycleLogContext{Scenario: item.Slug, Operation: "phase", Phase: phaseName}, func(logWriter, childWriter io.Writer) error {
		var executeErr error
		result, executeErr = r.ExecutePhaseDetailed(item, phaseName, env, args, logWriter, childWriter)
		return executeErr
	}); err != nil {
		r.logError("Scenario phase failed", err, logx.AttrScenario, item.Slug, logx.AttrPhase, phaseName)
		return result, err
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
		if phaseName == "test" && len(args) > 0 {
			quotedArgs := make([]string, 0, len(args))
			for _, arg := range args {
				quotedArgs = append(quotedArgs, shellQuote(arg))
			}
			finalCmd += " " + strings.Join(quotedArgs, " ")
		}
		if phaseName == "test" {
			finalCmd = injectTestGenieAutoStart(finalCmd)
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

func injectTestGenieAutoStart(command string) string {
	fields := strings.Fields(command)
	if len(fields) < 2 {
		return command
	}

	commandIndex := -1
	for index, field := range fields {
		if strings.Contains(field, "=") && !strings.HasPrefix(field, "-") {
			continue
		}
		commandIndex = index
		break
	}
	if commandIndex < 0 || commandIndex+1 >= len(fields) {
		return command
	}
	if filepath.Base(fields[commandIndex]) != "test-genie" || fields[commandIndex+1] != "execute" {
		return command
	}
	for _, field := range fields[commandIndex+2:] {
		if field == "--auto-start" || strings.HasPrefix(field, "--auto-start=") {
			return command
		}
	}

	target := fields[commandIndex] + " execute"
	replacement := fields[commandIndex] + " --auto-start execute"
	return strings.Replace(command, target, replacement, 1)
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

	stepEnv := mergeEnv(os.Environ(), env)
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
		Stdout: file,
		Stderr: file,
	})
	cmd.SysProcAttr = backgroundProcessAttr()

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
	if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
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

	stepEnv := mergeEnv(os.Environ(), env)
	stepEnv = setEnvValue(stepEnv, "LIFECYCLE_PHASE", phase)
	stepEnv = setEnvValue(stepEnv, "VROOLI_LIFECYCLE_MANAGED", "true")

	// Scenario lifecycle steps remain shell-defined by the service.json contract.
	cmd := shell.BashCommand(command, shell.Spec{
		Dir:    sourceDir,
		Env:    stepEnv,
		Stdout: logWriter,
		Stderr: logWriter,
	})
	return cmd.Run()
}

// runWithLifecycleLog opens the scenario lifecycle log file and invokes fn
// with two writers: logWriter is for orchestrator-level output (slog text
// and [INFO]/[WARNING] step headers) — it tees the log file and the
// console, gated by the current verbosity; childWriter is for raw tool
// stdout (vite/pnpm) — it tees the log file and reaches the console only
// at VerbosityVerbose. The log file always receives everything.
func (r *Runner) runWithLifecycleLog(ctx lifecycleLogContext, fn func(logWriter, childWriter io.Writer) error) error {
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
		return err
	}
	if _, err := config.EnsureOwnedDir(filepath.Dir(path)); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	_ = config.ChownToInvokingUser(path)

	logWriter := io.MultiWriter(r.consoleOut(), file)
	childWriter := io.MultiWriter(r.childStdoutConsole(), file)
	startedAt := time.Now().UTC()
	runID := lifecycleRunID(startedAt)
	writeLifecycleRunStart(file, ctx, runID, startedAt)
	err = fn(logWriter, childWriter)
	writeLifecycleRunEnd(file, ctx, runID, startedAt, time.Now().UTC(), err)
	return err
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
