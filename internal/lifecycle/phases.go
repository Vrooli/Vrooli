package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/envkit-go"
	"github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/logx"
	"github.com/vrooli/vrooli/internal/network"
	"github.com/vrooli/vrooli/internal/ports"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
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
	ctx := opts.Context
	if ctx == nil {
		ctx = context.Background()
	}
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
		bootstrapOpts := StartOptions{CustomPath: opts.CustomPath}
		if _, _, err := r.bootstrapScenarioDependencies(item, bootstrapOpts, newStartSession(ctx)); err != nil {
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
		envResult, err = r.prepareScenarioEnvironment(ctx, item, disabledRuntimeRegistrySession())
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

	var result PhaseResult
	logPath, _ := process.ScenarioLifecycleLogPath(r.Home, item.Slug)
	meta, runErr := r.runWithLifecycleLog(lifecycleLogContext{Scenario: item.Slug, Operation: "phase", Phase: phaseName, RunID: strings.TrimSpace(opts.RunID)}, func(logWriter, childWriter io.Writer) error {
		var executeErr error
		result, executeErr = r.executePhaseDetailed(ctx, item, phaseName, env, logWriter, childWriter)
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

// RunPhaseDetailedContext is the explicit cancellation-aware phase entry
// point. The legacy method remains available to callers that do not need a
// cancellation context.
func (r *Runner) RunPhaseDetailedContext(ctx context.Context, name, phaseName string, opts PhaseOptions) (PhaseResult, error) {
	opts.Context = ctx
	return r.RunPhaseDetailed(name, phaseName, opts)
}

func phaseRequiresBootstrap(phaseName string) bool {
	switch strings.TrimSpace(phaseName) {
	case "develop":
		return true
	default:
		return false
	}
}

// ExecutePhaseDetailed runs a phase's steps. logWriter receives orchestrator
// messages (infof/warnf headers, slog text if routed there); childWriter
// receives raw child-process stdout for foreground steps. Callers that do
// not need to split the two can pass the same writer for both
// (tests/io.Discard). Production callers should provide a logWriter that
// tees to the scenario lifecycle log file plus the console at the current
// verbosity, and a childWriter that tees to the log file plus the console
// only at verbose — see runWithLifecycleLog.
func (r *Runner) ExecutePhaseDetailed(item scenario.Scenario, phaseName string, env map[string]string, logWriter io.Writer, childWriter io.Writer) (PhaseResult, error) {
	return r.executePhaseDetailed(context.Background(), item, phaseName, env, logWriter, childWriter)
}

func (r *Runner) executePhaseDetailed(ctx context.Context, item scenario.Scenario, phaseName string, env map[string]string, logWriter io.Writer, childWriter io.Writer) (PhaseResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if childWriter == nil {
		childWriter = logWriter
	}
	phase, ok := lookupPhase(item.Manifest, phaseName)
	derivedComponentPhase := len(item.Manifest.Components) > 0 && (phaseName == "setup" || phaseName == "develop")
	if !ok && !derivedComponentPhase {
		r.logDebug("Scenario phase not defined", logx.AttrScenario, item.Slug, logx.AttrPhase, phaseName)
		return undefinedPhaseResult(item.Slug, phaseName), nil
	}
	if !phaseDefined(phase) && !derivedComponentPhase {
		r.logDebug("Scenario phase empty; treating as undefined", logx.AttrScenario, item.Slug, logx.AttrPhase, phaseName)
		return undefinedPhaseResult(item.Slug, phaseName), nil
	}
	result := PhaseResult{
		Scenario: item.Slug,
		Phase:    phaseName,
		Defined:  true,
		Status:   PhaseExecutionSkipped,
	}
	if phaseName == "setup" {
		if err := r.provisionSharedPackages(ctx, item, env, logWriter, childWriter); err != nil {
			return PhaseResult{}, err
		}
		built, err := r.buildDeclaredComponents(ctx, item, env, childWriter)
		if err != nil {
			return result, err
		}
		result.ExecutedSteps += built
		if built > 0 {
			result.Status = PhaseExecutionCompleted
		}
	}
	steps := declaredPhaseSteps(item.Manifest, phaseName, phase.Steps)
	lifecycleLogPath, err := process.ScenarioLifecycleLogPath(r.Home, item.Slug)
	if err != nil {
		return result, err
	}
	for index, step := range steps {
		if len(step.Exec) == 0 {
			continue
		}
		ok, reason, err := stepConditionsMet(item, step.Condition, env)
		if err != nil {
			return result, err
		}
		if !ok {
			r.infof(logWriter, "[%d/%d] Skipping %s - %s", index+1, len(steps), step.Name, reason)
			r.logDebug("Skipping lifecycle step",
				logx.AttrScenario, item.Slug,
				logx.AttrPhase, phaseName,
				logx.AttrStep, step.Name,
				"reason", reason,
			)
			result.SkippedSteps++
			continue
		}
		if _, component, isComponent := componentForStep(item.Manifest, step.Name); isComponent {
			ok, reason, err = stepConditionsMet(item, component.Run.Condition, env)
			if err != nil {
				return result, err
			}
			if !ok {
				r.infof(logWriter, "[%d/%d] Skipping %s - %s", index+1, len(steps), step.Name, reason)
				result.SkippedSteps++
				continue
			}
		}

		r.infof(logWriter, "[%d/%d] %s", index+1, len(steps), step.Name)
		r.logDebug("Executing lifecycle step",
			logx.AttrScenario, item.Slug,
			logx.AttrPhase, phaseName,
			logx.AttrStep, step.Name,
			"background", step.Background,
		)

		if step.Background {
			if err := r.startTrackedProcessContext(ctx, item, phaseName, step, env); err != nil {
				return result, err
			}
			result.ExecutedSteps++
			result.Status = PhaseExecutionCompleted
			continue
		}

		sink := newStepSink(childWriter)
		stepErr := r.runForegroundStep(ctx, item, phaseName, step, env, sink)
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

// declaredPhaseSteps makes components the sole authority for process launch.
// Setup shell mirrors are ignored because buildDeclaredComponents derives the
// same work from build.kind. Develop shell mirrors are replaced by one typed
// background launch per component. Explicit argv provisioning steps remain.
func declaredPhaseSteps(manifest scenario.ServiceManifest, phaseName string, authored []scenario.PhaseStep) []scenario.PhaseStep {
	steps := make([]scenario.PhaseStep, 0, len(authored)+len(manifest.Components))
	for _, step := range authored {
		if phaseName != "setup" && phaseName != "develop" || len(step.Exec) > 0 {
			steps = append(steps, step)
		}
	}
	if phaseName != "develop" {
		return steps
	}
	for _, name := range orderedComponentNames(manifest.Components) {
		component := manifest.Components[name]
		steps = append(steps, scenario.PhaseStep{
			Name:       "start-" + name,
			Exec:       append([]string(nil), component.Run.Argv...),
			Background: true,
			Condition:  component.Run.Condition,
		})
	}
	return steps
}

func orderedComponentNames(components map[string]scenario.Component) []string {
	indegree := make(map[string]int, len(components))
	dependents := make(map[string][]string, len(components))
	for name := range components {
		indegree[name] = 0
	}
	for name, component := range components {
		dependencies := make([]string, 0, len(component.Run.DependsOn)+1)
		for _, dependency := range component.Run.DependsOn {
			dependencies = append(dependencies, dependency.Component)
		}
		if component.Run.SupervisedBy != "" {
			dependencies = append(dependencies, component.Run.SupervisedBy)
		}
		for _, dependency := range uniqueStrings(dependencies) {
			if _, ok := components[dependency]; !ok {
				continue
			}
			indegree[name]++
			dependents[dependency] = append(dependents[dependency], name)
		}
	}
	ready := make([]string, 0, len(components))
	for name, degree := range indegree {
		if degree == 0 {
			ready = append(ready, name)
		}
	}
	sortComponentNames(ready, components)
	ordered := make([]string, 0, len(components))
	for len(ready) > 0 {
		name := ready[0]
		ready = ready[1:]
		ordered = append(ordered, name)
		for _, dependent := range dependents[name] {
			indegree[dependent]--
			if indegree[dependent] == 0 {
				ready = append(ready, dependent)
				sortComponentNames(ready, components)
			}
		}
	}
	return ordered
}

func sortComponentNames(names []string, components map[string]scenario.Component) {
	roleOrder := map[string]int{"api": 0, "worker": 1, "sidecar": 1, "ui": 2}
	sort.Slice(names, func(i, j int) bool {
		left, leftOK := roleOrder[components[names[i]].Role]
		right, rightOK := roleOrder[components[names[j]].Role]
		if !leftOK {
			left = 3
		}
		if !rightOK {
			right = 3
		}
		if left == right {
			return names[i] < names[j]
		}
		return left < right
	})
}

func (r *Runner) buildDeclaredComponents(ctx context.Context, item scenario.Scenario, env map[string]string, writer io.Writer) (int, error) {
	executed := 0
	registry := BuilderRegistry()
	buildMode := BuildModeForEnv(env, os.Getenv)
	for _, name := range orderedComponentNames(item.Manifest.Components) {
		component := item.Manifest.Components[name]
		if component.Build.Reuse != "" {
			continue
		}
		ok, _, err := stepConditionsMet(item, component.Run.Condition, env)
		if err != nil {
			return executed, err
		}
		if !ok {
			continue
		}
		spec, exists := registry[component.Build.Kind]
		if !exists || spec.Reserved {
			return executed, fmt.Errorf("component %s has no executable builder %q", name, component.Build.Kind)
		}
		targets, targetErr := componentBuildTargets(name, item.Path, item.Slug, component, spec, runtimeGOOS())
		if targetErr != nil {
			return executed, targetErr
		}
		install, _, installErr := installNeeded(item.Path, component, spec)
		if installErr != nil {
			return executed, fmt.Errorf("component %s install inputs: %w", name, installErr)
		}
		commandIndex := 0
		if install {
			replacer := strings.NewReplacer("{dir}", component.Build.Dir, "{scenario}", item.Slug, "{component}", name)
			argv := make([]string, len(spec.Install))
			for index, value := range spec.Install {
				argv[index] = replacer.Replace(value)
			}
			stepEnv := cloneStringMap(spec.Environment)
			step := scenario.PhaseStep{
				Name: fmt.Sprintf("build-%s-%d", name, commandIndex+1),
				Exec: argv,
				CWD:  filepath.ToSlash(component.Build.Dir),
				Env:  stepEnv,
			}
			if err := r.runForegroundStep(ctx, item, "setup", step, env, writer); err != nil {
				return executed, fmt.Errorf("build component %s: %w", name, err)
			}
			if err := recordInstallDigest(item.Path, component, spec); err != nil {
				return executed, fmt.Errorf("component %s install digest: %w", name, err)
			}
			executed++
			commandIndex++
		}
		for _, buildTarget := range targets {
			buildArgv := spec.BuildArgv(buildMode)
			target := buildTarget.Output
			publishTarget := componentPublishTarget(target, spec.StageDirectory)
			var stagedOutput string
			var cleanupStage func()
			if spec.StagedOutput {
				_, stagedOutput, cleanupStage, err = stageArtifact(publishTarget, spec.StageDirectory)
				if err != nil {
					return executed, fmt.Errorf("stage component %s output: %w", name, err)
				}
			}
			stageDir := stagedOutput
			if !spec.StageDirectory && stagedOutput != "" {
				stageDir = filepath.Dir(stagedOutput)
			}
			buildOutput := target
			if spec.StagedOutput {
				buildOutput = stagedOutput
			}
			replacer := strings.NewReplacer(
				"{dir}", component.Build.Dir,
				"{scenario}", item.Slug,
				"{component}", name,
				"{entry}", buildTarget.Entry,
				"{output}", buildOutput,
				"{stage_output_dir}", stageDir,
			)
			if spec.StagedOutput {
				buildArgv = append(buildArgv, spec.StageArgs...)
			}
			template := buildArgv
			if len(template) == 0 {
				if cleanupStage != nil {
					cleanupStage()
				}
				continue
			}
			argv := make([]string, len(template))
			for index, value := range template {
				argv[index] = replacer.Replace(value)
			}
			stepEnv := cloneStringMap(spec.Environment)
			step := scenario.PhaseStep{
				Name: fmt.Sprintf("build-%s-%d", name, commandIndex+1),
				Exec: argv,
				CWD:  filepath.ToSlash(component.Build.Dir),
				Env:  stepEnv,
			}
			if err := r.runForegroundStep(ctx, item, "setup", step, env, writer); err != nil {
				if cleanupStage != nil {
					cleanupStage()
				}
				return executed, fmt.Errorf("build component %s: %w", name, err)
			}
			if spec.StagedOutput {
				if err := swapArtifact(stagedOutput, publishTarget); err != nil {
					if cleanupStage != nil {
						cleanupStage()
					}
					return executed, fmt.Errorf("publish component %s output: %w", name, err)
				}
			}
			if cleanupStage != nil {
				cleanupStage()
			}
			executed++
			commandIndex++
		}
	}
	return executed, nil
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func componentPublishTarget(target string, directory bool) string {
	if directory {
		return filepath.Dir(target)
	}
	return target
}

func slicesContainTemplate(values []string, token string) bool {
	for _, value := range values {
		if strings.Contains(value, token) {
			return true
		}
	}
	return false
}

func runtimeGOOS() string {
	return runtime.GOOS
}

func (r *Runner) startTrackedProcess(item scenario.Scenario, phase string, step scenario.PhaseStep, env map[string]string) error {
	return r.startTrackedProcessContext(context.Background(), item, phase, step, env)
}

func (r *Runner) startTrackedProcessContext(ctx context.Context, item scenario.Scenario, phase string, step scenario.PhaseStep, env map[string]string) error {
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

	file, err := config.OpenOwnedFile(logFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return newPhaseStepError(item.Slug, phase, step.Name, logFile, err)
	}
	defer file.Close()

	stepEnv := lifecycleStepEnv(phase, env)
	stepEnv = setEnvValue(stepEnv, "VROOLI_PROCESS_ID", processID)
	stepEnv = setEnvValue(stepEnv, "VROOLI_PHASE", phase)
	stepEnv = setEnvValue(stepEnv, "VROOLI_SCENARIO", item.Slug)
	stepEnv = setEnvValue(stepEnv, "VROOLI_STEP", step.Name)
	stepEnv = setEnvValue(stepEnv, "VROOLI_LIFECYCLE_MANAGED", "true")

	declared, _, err := declaredCommandForStep(item, step, stepEnv)
	if err != nil {
		return newPhaseStepError(item.Slug, phase, step.Name, logFile, err)
	}
	if _, component, ok := componentForStep(item.Manifest, step.Name); ok {
		if err := r.waitForComponentDependencies(item, component, env); err != nil {
			return newPhaseStepError(item.Slug, phase, step.Name, logFile, err)
		}
		if err := createComponentRuntimeDirectories(item.Path, component); err != nil {
			return newPhaseStepError(item.Slug, phase, step.Name, logFile, err)
		}
	}
	port := declared.Port
	if port > 0 {
		if inspection := r.runtimeDeps().inspectPort(port); len(inspection.Listeners) > 0 {
			return newPhaseStepError(item.Slug, phase, step.Name, logFile, portConflictError(port, inspection))
		}
	}

	// Background steps are intentionally detached daemons. Their process group
	// must outlive this orchestration context so the runtime supervisor can own
	// and tear them down after the phase returns.
	cmd := exec.CommandContext(context.Background(), declared.Argv[0], declared.Argv[1:]...)
	cmd.Dir = declared.Dir
	cmd.Env = declared.Env
	cmd.Stdin = strings.NewReader("")
	cmd.Stdout = file
	cmd.Stderr = file
	workingDir := declared.Dir
	commandText := strings.Join(declared.Argv, " ")
	if err := platform.ConfigureCommand(cmd, platform.ProcessOptions{Detached: true}); err != nil {
		return newPhaseStepError(item.Slug, phase, step.Name, logFile, err)
	}

	if err := cmd.Start(); err != nil {
		return newPhaseStepError(item.Slug, phase, step.Name, logFile, err)
	}
	containmentRelease, err := platform.AssignProcessContainment(cmd.Process)
	if err != nil {
		_ = cmd.Process.Kill()
		return newPhaseStepError(item.Slug, phase, step.Name, logFile, fmt.Errorf("assign process containment: %w", err))
	}
	r.registerContainment(cmd.Process.Pid, containmentRelease)

	record := process.Record{
		PID:        cmd.Process.Pid,
		PGID:       cmd.Process.Pid,
		ProcessID:  processID,
		Phase:      phase,
		Scenario:   item.Slug,
		Step:       step.Name,
		Command:    commandText,
		WorkingDir: workingDir,
		LogFile:    logFile,
		Port:       port,
		PortKey:    declared.PortKey,
		StartedAt:  time.Now().UTC(),
		Status:     "running",
	}
	if err := process.WriteScenarioRecord(r.Home, slug, step.Name, record); err != nil {
		_ = cmd.Process.Kill()
		r.releaseContainment(cmd.Process.Pid)
		return newPhaseStepError(item.Slug, phase, step.Name, logFile, err)
	}
	if err := recordRuntimeProcessRef(ctx, r.runtimeDeps(), r.Home, env, record); err != nil {
		_ = cmd.Process.Kill()
		r.releaseContainment(cmd.Process.Pid)
		_ = process.RemoveScenarioRecord(r.Home, slug, step.Name)
		return newPhaseStepError(item.Slug, phase, step.Name, logFile, err)
	}

	if err := AwaitContext(ctx, r.awaitClock(), backgroundLaunchPolicy, func() (bool, error) {
		return r.runtimeDeps().isPIDRunning(cmd.Process.Pid), nil
	}); err != nil {
		record.Status = "failed"
		_ = process.WriteScenarioRecord(r.Home, slug, step.Name, record)
		r.releaseContainment(cmd.Process.Pid)
		exitErr := backgroundProcessExitError(cmd, logFile)
		return newPhaseStepError(
			item.Slug,
			phase,
			step.Name,
			logFile,
			exitErr,
		)
	}
	return nil
}

func (r *Runner) waitForComponentDependencies(item scenario.Scenario, component scenario.Component, env map[string]string) error {
	for _, dependency := range component.Run.DependsOn {
		target, exists := item.Manifest.Components[dependency.Component]
		if !exists {
			return fmt.Errorf("component dependency %q is not declared", dependency.Component)
		}
		records, err := process.ReadScenarioRecords(r.Home, recordSlug(item))
		if err != nil {
			return err
		}
		started := false
		for _, record := range process.LiveRecords(records) {
			if record.Step == "start-"+dependency.Component {
				started = true
				break
			}
		}
		if !started {
			return fmt.Errorf("component dependency %q has not started", dependency.Component)
		}
		if dependency.Wait == "ready" {
			if err := r.awaitComponentReadinessNamed(recordSlug(item), item.Manifest, dependency.Component, target, env); err != nil {
				return fmt.Errorf("component dependency %q readiness: %w", dependency.Component, err)
			}
		}
	}
	return nil
}

func (r *Runner) awaitComponentReadinessNamed(scenarioSlug string, manifest scenario.ServiceManifest, name string, component scenario.Component, env map[string]string) error {
	readiness := derivedReadiness(component)
	timeout := time.Duration(readiness.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	var lastErr error
	err := Await(r.awaitClock(), AwaitPolicy{Timeout: timeout, Interval: 250 * time.Millisecond}, func() (bool, error) {
		lastErr = r.checkComponentReadinessNamed(scenarioSlug, manifest, name, component, env)
		return lastErr == nil, nil
	})
	if err != nil {
		return fmt.Errorf("timed out after %s: %w", timeout, lastErr)
	}
	return nil
}

func checkComponentReadiness(manifest scenario.ServiceManifest, component scenario.Component, env map[string]string) error {
	return checkComponentReadinessNamed(manifest, "component", component, env)
}

func checkComponentReadinessNamed(manifest scenario.ServiceManifest, name string, component scenario.Component, env map[string]string) error {
	readiness := derivedReadiness(component)
	if readiness == nil {
		return nil
	}
	if readiness.Type == "process_alive" {
		return fmt.Errorf("component %q process liveness requires a runner", name)
	}
	portValue, exists := envPortValue(manifest.PortEnvVar(component.Run.Port), env)
	if !exists {
		return fmt.Errorf("run.port %q has no allocated value", component.Run.Port)
	}
	switch readiness.Type {
	case "port_open":
		conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(portValue)), 500*time.Millisecond)
		if err != nil {
			return err
		}
		return conn.Close()
	case "http":
		path, err := ports.ExpandTemplate(readiness.Path, env)
		if err != nil {
			return err
		}
		if path == "" {
			path = "/"
		}
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		client := http.Client{Timeout: 500 * time.Millisecond}
		response, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d%s", portValue, path))
		if err != nil {
			return err
		}
		defer response.Body.Close()
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			return fmt.Errorf("HTTP %d", response.StatusCode)
		}
		return nil
	default:
		return fmt.Errorf("unsupported readiness type %q", readiness.Type)
	}
}

func (r *Runner) checkComponentReadinessNamed(scenarioSlug string, manifest scenario.ServiceManifest, name string, component scenario.Component, env map[string]string) error {
	readiness := derivedReadiness(component)
	if readiness.Type == "process_alive" {
		if strings.TrimSpace(scenarioSlug) == "" {
			return fmt.Errorf("component %q process liveness requires a scenario slug", name)
		}
		records, err := process.ReadScenarioRecords(r.Home, scenarioSlug)
		if err != nil {
			return err
		}
		for _, record := range process.LiveRecords(records) {
			if record.Step == "start-"+name {
				return nil
			}
		}
		return fmt.Errorf("component process is not alive")
	}
	return checkComponentReadinessNamed(manifest, name, component, env)
}

func derivedReadiness(component scenario.Component) *scenario.ComponentReadiness {
	if component.Run.Readiness != nil {
		return component.Run.Readiness
	}
	if strings.TrimSpace(component.Run.Port) != "" {
		return &scenario.ComponentReadiness{Type: "port_open", TimeoutMS: 30000}
	}
	return &scenario.ComponentReadiness{Type: "process_alive", TimeoutMS: 30000}
}

func envPortValue(envVar string, env map[string]string) (int, bool) {
	value, exists := env[envVar]
	if !exists {
		return 0, false
	}
	port, err := strconv.Atoi(value)
	return port, err == nil && port > 0
}

func createComponentRuntimeDirectories(scenarioPath string, component scenario.Component) error {
	paths := append([]string(nil), component.Run.DataDirs...)
	if strings.TrimSpace(component.Run.LogDir) != "" {
		paths = append(paths, component.Run.LogDir)
	}
	for _, relative := range paths {
		dir, err := componentWorkingDir(scenarioPath, relative)
		if err != nil {
			return fmt.Errorf("runtime directory %q: %w", relative, err)
		}
		if _, err := config.EnsureOwnedDir(dir); err != nil {
			return fmt.Errorf("create runtime directory %q: %w", relative, err)
		}
	}
	return nil
}

func portConflictError(port int, inspection network.PortInspection) error {
	holders := make([]string, 0, len(inspection.Listeners))
	for _, listener := range inspection.Listeners {
		holder := "listener with unknown process"
		if listener.PID > 0 {
			holder = fmt.Sprintf("pid %d", listener.PID)
		}
		if command := strings.TrimSpace(listener.Command); command != "" {
			holder += fmt.Sprintf(" (%s)", command)
		}
		holders = append(holders, holder)
	}
	return fmt.Errorf("port %d is already in use by %s", port, strings.Join(holders, ", "))
}

func backgroundProcessExitError(cmd *exec.Cmd, logFile string) error {
	message := "background step exited immediately"
	if cmd != nil {
		if waitErr := cmd.Wait(); waitErr != nil {
			message += ": " + waitErr.Error()
		}
	}
	if line := lastLogLine(logFile); line != "" {
		message += "; last log line: " + line
	} else {
		message += "; no log output"
	}
	return errors.New(message)
}

func lastLogLine(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimRight(string(data), "\r\n"), "\n")
	for index := len(lines) - 1; index >= 0; index-- {
		if line := strings.TrimSpace(lines[index]); line != "" {
			return line
		}
	}
	return ""
}

func (r *Runner) runForegroundStep(ctx context.Context, item scenario.Scenario, phase string, step scenario.PhaseStep, env map[string]string, logWriter io.Writer) error {
	stepEnv := lifecycleStepEnv(phase, env)

	declared, _, err := declaredCommandForStep(item, step, stepEnv)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, declared.Argv[0], declared.Argv[1:]...)
	cmd.Dir = declared.Dir
	cmd.Env = declared.Env
	cmd.Stdin = strings.NewReader("")
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter
	if err := platform.ConfigureCommand(cmd, platform.ProcessOptions{}); err != nil {
		return err
	}
	return cmd.Run()
}

type declaredStepCommand struct {
	Argv    []string
	Dir     string
	Env     []string
	Port    int
	PortKey string
}

func componentForStep(manifest scenario.ServiceManifest, stepName string) (string, scenario.Component, bool) {
	name := strings.TrimPrefix(strings.TrimSpace(stepName), "start-")
	component, ok := manifest.Components[name]
	return name, component, ok
}

func declaredCommandForStep(item scenario.Scenario, step scenario.PhaseStep, baseEnv []string) (declaredStepCommand, bool, error) {
	_, component, isComponent := componentForStep(item.Manifest, step.Name)
	if isComponent {
		argv, err := ResolveComponentArgv(component.Run.Argv, item.Path, item.Slug, item.Manifest.Components)
		if err != nil {
			return declaredStepCommand{}, false, err
		}
		environment := envSliceMap(baseEnv)
		argv, err = expandDeclaredValues(argv, environment)
		if err != nil {
			return declaredStepCommand{}, false, fmt.Errorf("component %s run.argv: %w", step.Name, err)
		}
		dir, err := componentWorkingDir(item.Path, component.Run.CWD)
		if err != nil {
			return declaredStepCommand{}, false, err
		}
		commandEnv := append([]string(nil), baseEnv...)
		keys := make([]string, 0, len(component.Run.Env))
		for key := range component.Run.Env {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			value, err := ports.ExpandTemplate(component.Run.Env[key], environment)
			if err != nil {
				return declaredStepCommand{}, false, fmt.Errorf("component %s run.env.%s: %w", step.Name, key, err)
			}
			commandEnv = setEnvValue(commandEnv, key, value)
			environment[key] = value
		}
		portKey := ""
		port := 0
		if strings.TrimSpace(component.Run.Port) != "" {
			portKey = item.Manifest.PortEnvVar(component.Run.Port)
			if portKey == "" {
				return declaredStepCommand{}, false, fmt.Errorf("component %s run.port %q is not declared", step.Name, component.Run.Port)
			}
			port, _ = strconv.Atoi(environment[portKey])
		}
		return declaredStepCommand{Argv: argv, Dir: dir, Env: commandEnv, Port: port, PortKey: portKey}, true, nil
	}
	if len(step.Exec) == 0 {
		return declaredStepCommand{}, false, nil
	}
	environment := envSliceMap(baseEnv)
	argv, err := expandDeclaredValues(step.Exec, environment)
	if err != nil {
		return declaredStepCommand{}, false, fmt.Errorf("step %s exec: %w", step.Name, err)
	}
	dir, err := componentWorkingDir(item.Path, step.CWD)
	if err != nil {
		return declaredStepCommand{}, false, err
	}
	commandEnv := append([]string(nil), baseEnv...)
	keys := make([]string, 0, len(step.Env))
	for key := range step.Env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value, err := ports.ExpandTemplate(step.Env[key], environment)
		if err != nil {
			return declaredStepCommand{}, false, fmt.Errorf("step %s env.%s: %w", step.Name, key, err)
		}
		commandEnv = setEnvValue(commandEnv, key, value)
		environment[key] = value
	}
	return declaredStepCommand{Argv: argv, Dir: dir, Env: commandEnv}, true, nil
}

func expandDeclaredValues(values []string, environment map[string]string) ([]string, error) {
	out := make([]string, len(values))
	for index, value := range values {
		expanded, err := ports.ExpandTemplate(value, environment)
		if err != nil {
			return nil, fmt.Errorf("argv[%d]: %w", index, err)
		}
		out[index] = expanded
	}
	if len(out) == 0 || strings.TrimSpace(out[0]) == "" {
		return nil, errors.New("declared argv requires an executable")
	}
	return out, nil
}

func envSliceMap(values []string) map[string]string {
	out := make(map[string]string, len(values))
	for _, value := range values {
		key, resolved, ok := strings.Cut(value, "=")
		if ok {
			out[key] = resolved
		}
	}
	return out
}

// lifecycleStepEnv makes setup safe for server-owned execution. Setup may
// invoke package managers and other tools that
// prompt when they detect stale state; a lifecycle request has no interactive
// stdin to answer such prompts, so fail deterministically instead.
func lifecycleStepEnv(phase string, overrides map[string]string) []string {
	overlay := make(envkit.Env, 0, len(overrides))
	for key, value := range overrides {
		overlay = append(overlay, key+"="+value)
	}
	stepEnv := envkit.WithOverlay(envkit.Env(os.Environ()), envkit.ForeignScenario, overlay)
	stepEnv = setEnvValue(stepEnv, "PATH", hostreqkit.AugmentUserToolPath(
		envValue(stepEnv, "HOME"),
		envValue(stepEnv, "PATH"),
		envValue(stepEnv, "LOCALAPPDATA"),
	))
	stepEnv = setEnvValue(stepEnv, "LIFECYCLE_PHASE", phase)
	stepEnv = setEnvValue(stepEnv, "VROOLI_LIFECYCLE_MANAGED", "true")
	if phase == "setup" {
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
	file, err := config.OpenOwnedFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
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
	return strings.TrimSpace(phase.Description) != "" || len(phase.Steps) > 0
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
