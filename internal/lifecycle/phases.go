package lifecycle

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

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
		if _, err := r.ensureDependencies(item, bootstrapOpts, ready, []string{item.Slug}); err != nil {
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
		envResult, err = r.Ports.BuildEnvironment(item, nil)
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

	if err := r.runWithLifecycleLog(item.Slug, func(logWriter io.Writer) error {
		return r.ensureScenarioDatabase(item, env, logWriter)
	}); err != nil {
		return PhaseResult{}, err
	}

	args := append([]string(nil), opts.Args...)
	var result PhaseResult
	if err := r.runWithLifecycleLog(item.Slug, func(logWriter io.Writer) error {
		var executeErr error
		result, executeErr = r.ExecutePhaseDetailed(item, phaseName, env, args, logWriter)
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
	_, err := r.ExecutePhaseDetailed(item, phaseName, env, args, logWriter)
	return err
}

func (r *Runner) ExecutePhaseDetailed(item scenario.Scenario, phaseName string, env map[string]string, args []string, logWriter io.Writer) (PhaseResult, error) {
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
	lifecycleLogPath := process.ScenarioLifecycleLogPath(r.Home, item.Slug)
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

		if step.Background {
			if err := r.startTrackedProcess(item, phaseName, step, env); err != nil {
				return result, err
			}
			result.ExecutedSteps++
			result.Status = PhaseExecutionCompleted
			continue
		}

		if err := r.runForegroundStep(item, phaseName, finalCmd, env, logWriter); err != nil {
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
			return result, newPhaseStepError(item.Slug, phaseName, step.Name, lifecycleLogPath, err)
		}
		result.ExecutedSteps++
		result.Status = PhaseExecutionCompleted
	}

	return result, nil
}

func (r *Runner) startTrackedProcess(item scenario.Scenario, phase string, step scenario.PhaseStep, env map[string]string) error {
	processID := fmt.Sprintf("vrooli.%s.%s.%s", phase, item.Slug, step.Name)
	logDir := process.ScenarioLogsDir(r.Home, item.Slug)
	if err := os.MkdirAll(logDir, 0o755); err != nil {
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
		Dir:    item.Path,
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
		WorkingDir: item.Path,
		LogFile:    logFile,
		Port:       port,
		StartedAt:  time.Now().UTC(),
		Status:     "running",
	}
	if err := process.WriteScenarioRecord(r.Home, item.Slug, step.Name, record); err != nil {
		_ = cmd.Process.Kill()
		return newPhaseStepError(item.Slug, phase, step.Name, logFile, err)
	}
	if port > 0 {
		if err := r.Ports.WriteLock(port, item.Slug, cmd.Process.Pid); err != nil {
			_ = cmd.Process.Kill()
			_ = process.RemoveScenarioRecord(r.Home, item.Slug, step.Name)
			return newPhaseStepError(item.Slug, phase, step.Name, logFile, err)
		}
	}

	sleepFn(200 * time.Millisecond)
	if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
		record.Status = "failed"
		_ = process.WriteScenarioRecord(r.Home, item.Slug, step.Name, record)
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
	stepEnv := mergeEnv(os.Environ(), env)
	stepEnv = setEnvValue(stepEnv, "LIFECYCLE_PHASE", phase)
	stepEnv = setEnvValue(stepEnv, "VROOLI_LIFECYCLE_MANAGED", "true")

	// Scenario lifecycle steps remain shell-defined by the service.json contract.
	cmd := shell.BashCommand(command, shell.Spec{
		Dir:    item.Path,
		Env:    stepEnv,
		Stdout: logWriter,
		Stderr: logWriter,
	})
	return cmd.Run()
}

func (r *Runner) runWithLifecycleLog(name string, fn func(logWriter io.Writer) error) error {
	path := process.ScenarioLifecycleLogPath(r.Home, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := io.MultiWriter(r.Out, file)
	return fn(writer)
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
	case "version":
		return manifest.Lifecycle.VersionCmd, true
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
