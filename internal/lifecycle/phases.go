package lifecycle

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/vrooli/vrooli/internal/ports"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
)

func (r *Runner) RunPhase(name, phaseName string, opts PhaseOptions) error {
	item, err := r.loadScenario(name, opts.CustomPath)
	if err != nil {
		return err
	}

	if phaseRequiresBootstrap(phaseName) {
		ready := make(map[string]struct{})
		bootstrapOpts := StartOptions{CustomPath: opts.CustomPath}
		if _, err := r.ensureDependencies(item, bootstrapOpts, ready, []string{item.Slug}); err != nil {
			return err
		}
	}

	var envResult ports.Environment
	if opts.ProjectMode {
		envResult, err = r.Ports.BuildProjectEnvironment(item)
		if err != nil {
			return err
		}
	} else {
		envResult, err = r.Ports.BuildEnvironment(item, nil)
		if err != nil {
			return err
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
		return err
	}

	args := append([]string(nil), opts.Args...)
	return r.runWithLifecycleLog(item.Slug, func(logWriter io.Writer) error {
		return r.ExecutePhase(item, phaseName, env, args, logWriter)
	})
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
	phase, ok := lookupPhase(item.Manifest, phaseName)
	if !ok {
		return nil
	}
	if len(phase.Steps) == 0 && phase.Description == "" {
		return nil
	}

	for index, step := range phase.Steps {
		if strings.TrimSpace(step.Run) == "" {
			continue
		}
		ok, reason, err := stepConditionsMet(item, step.Condition, env)
		if err != nil {
			return err
		}
		if !ok {
			r.infof(logWriter, "[%d/%d] Skipping %s - %s", index+1, len(phase.Steps), step.Name, reason)
			continue
		}

		r.infof(logWriter, "[%d/%d] %s", index+1, len(phase.Steps), step.Name)

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
				return err
			}
			continue
		}

		if err := r.runForegroundStep(item, phaseName, finalCmd, env, logWriter); err != nil {
			if phaseName == "stop" {
				r.warnf(logWriter, "Stop step completed with non-zero exit: %s", step.Name)
				continue
			}
			return err
		}
	}

	return nil
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
		return err
	}
	defer file.Close()

	stepEnv := mergeEnv(os.Environ(), env)
	stepEnv = setEnvValue(stepEnv, "VROOLI_PROCESS_ID", processID)
	stepEnv = setEnvValue(stepEnv, "VROOLI_PHASE", phase)
	stepEnv = setEnvValue(stepEnv, "VROOLI_SCENARIO", item.Slug)
	stepEnv = setEnvValue(stepEnv, "VROOLI_STEP", step.Name)
	stepEnv = setEnvValue(stepEnv, "VROOLI_LIFECYCLE_MANAGED", "true")

	cmd := exec.Command("bash", "-lc", step.Run)
	cmd.Dir = item.Path
	cmd.Env = stepEnv
	cmd.Stdout = file
	cmd.Stderr = file
	cmd.SysProcAttr = backgroundProcessAttr()

	if err := cmd.Start(); err != nil {
		return err
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
		return err
	}
	if port > 0 {
		_ = r.Ports.WriteLock(port, item.Slug, cmd.Process.Pid)
	}

	time.Sleep(200 * time.Millisecond)
	if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
		record.Status = "failed"
		_ = process.WriteScenarioRecord(r.Home, item.Slug, step.Name, record)
		return fmt.Errorf("background step %s exited immediately", step.Name)
	}
	return nil
}

func (r *Runner) runForegroundStep(item scenario.Scenario, phase, command string, env map[string]string, logWriter io.Writer) error {
	stepEnv := mergeEnv(os.Environ(), env)
	stepEnv = setEnvValue(stepEnv, "LIFECYCLE_PHASE", phase)
	stepEnv = setEnvValue(stepEnv, "VROOLI_LIFECYCLE_MANAGED", "true")

	cmd := exec.Command("bash", "-lc", command)
	cmd.Dir = item.Path
	cmd.Env = stepEnv
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter
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
	case "test":
		return manifest.Lifecycle.Test, true
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
