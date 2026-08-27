package resources

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/tuning"

	platform "github.com/vrooli/platform-go"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/process"
)

// startCompanions launches every declared host-side companion (idempotent,
// best-effort). A companion that fails to start is a logged warning, never fatal:
// the container is already up and the resource's health check surfaces a dead
// companion (e.g. whisper's edge owns the canonical port, so a down edge fails
// the health probe). When no companions are declared this is a no-op, keeping the
// driver byte-identical for every non-adopting compose-service resource.
func startCompanions(resourceName string, companions []ResourceCompanion, recoveryAttempts int, stderr io.Writer) {
	for _, c := range companions {
		if err := startCompanion(resourceName, c, recoveryAttempts); err != nil {
			fmt.Fprintf(stderr, "warning: companion %q for %s did not start: %v\n", c.Name, resourceName, err)
		}
	}
}

// stopCompanions signals every declared companion to exit (idempotent,
// best-effort). No-op when none are declared.
func stopCompanions(resourceName string, companions []ResourceCompanion, stderr io.Writer) {
	for _, c := range companions {
		if err := stopCompanion(resourceName, c); err != nil {
			fmt.Fprintf(stderr, "warning: companion %q for %s did not stop cleanly: %v\n", c.Name, resourceName, err)
		}
	}
}

// resolveCompanionDir is the dir resolver seam (tests point it at a temp dir so
// the supervisor never touches the real runtime home).
var resolveCompanionDir = companionDir

type CompanionStatus struct {
	Name    string `json:"name"`
	Port    int    `json:"port,omitempty"`
	PID     int    `json:"pid,omitempty"`
	Alive   bool   `json:"alive"`
	Failed  bool   `json:"failed,omitempty"`
	Failure string `json:"failure,omitempty"`
}

const companionCrashWindow = tuning.LongOperationBudget

// companionDir resolves <home>/.vrooli/processes/resources/<resource> (the
// runtime-home processes authority), creating it owned by the operator.
func companionDir(resourceName string) (string, error) {
	home, err := config.HomeDir()
	if err != nil {
		return "", err
	}
	root, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyProcesses)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, "resources", resourceName)
	if _, err := config.EnsureOwnedDir(dir); err != nil {
		return "", err
	}
	return dir, nil
}

func startCompanion(resourceName string, c ResourceCompanion, recoveryAttempts int) error {
	if strings.TrimSpace(c.Name) == "" || strings.TrimSpace(c.Command) == "" {
		return fmt.Errorf("companion declaration is missing name/command")
	}
	dir, err := resolveCompanionDir(resourceName)
	if err != nil {
		return err
	}
	pidPath := filepath.Join(dir, c.Name+".pid")
	// Idempotent: a live companion is left running so a restart does not gap the
	// data path (the reverse proxy is stateless and dials the recreated container).
	if pid, ok := readCompanionPID(pidPath); ok {
		if process.IsPIDRunning(pid) {
			clearCompanionCrashState(dir, c.Name)
			return nil
		}
		if err := recordCompanionCrashAttempt(dir, c.Name, recoveryAttempts); err != nil {
			return err
		}
	}
	bin, err := exec.LookPath(c.Command)
	if err != nil {
		return fmt.Errorf("resolve %q on PATH: %w", c.Command, err)
	}
	logPath := filepath.Join(dir, c.Name+".log")
	logf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, tuning.PermFile)
	if err != nil {
		return err
	}
	defer logf.Close()

	cmd := exec.Command(bin, c.Args...)
	cmd.Stdout = logf
	cmd.Stderr = logf
	if err := platform.ConfigureCommand(cmd, platform.ProcessOptions{Detached: true}); err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	pid := cmd.Process.Pid
	// Detach: the companion outlives this short-lived control process.
	_ = cmd.Process.Release()
	clearCompanionFailure(dir, c.Name)
	return os.WriteFile(pidPath, []byte(strconv.Itoa(pid)), tuning.PermFile)
}

func terminateCompanion(pid int) error {
	return platform.KillProcess(pid, false)
}

func stopCompanion(resourceName string, c ResourceCompanion) error {
	dir, err := resolveCompanionDir(resourceName)
	if err != nil {
		return err
	}
	pidPath := filepath.Join(dir, c.Name+".pid")
	pid, ok := readCompanionPID(pidPath)
	if !ok {
		clearCompanionCrashState(dir, c.Name)
		return nil // nothing tracked — already stopped
	}
	if process.IsPIDRunning(pid) {
		if err := terminateCompanion(pid); err != nil {
			return err
		}
	}
	clearCompanionCrashState(dir, c.Name)
	return os.Remove(pidPath)
}

func companionStatuses(resourceName string, companions []ResourceCompanion) ([]CompanionStatus, error) {
	if len(companions) == 0 {
		return nil, nil
	}
	dir, err := resolveCompanionDir(resourceName)
	if err != nil {
		return nil, err
	}
	statuses := make([]CompanionStatus, 0, len(companions))
	for _, c := range companions {
		pid, _ := readCompanionPID(filepath.Join(dir, c.Name+".pid"))
		failed, failure := readCompanionFailure(dir, c.Name)
		statuses = append(statuses, CompanionStatus{
			Name:    c.Name,
			Port:    c.Port,
			PID:     pid,
			Alive:   pid > 0 && process.IsPIDRunning(pid),
			Failed:  failed,
			Failure: failure,
		})
	}
	return statuses, nil
}

func downCompanions(statuses []CompanionStatus) []CompanionStatus {
	down := make([]CompanionStatus, 0)
	for _, status := range statuses {
		if !status.Alive {
			down = append(down, status)
		}
	}
	return down
}

func companionDownMessage(resourceName string, down []CompanionStatus) string {
	if len(down) == 0 {
		return scenarioruntime.HealthStatusHealthy
	}
	parts := make([]string, 0, len(down))
	for _, status := range down {
		label := strings.TrimSpace(status.Name)
		if label == "" {
			label = "unnamed"
		}
		if status.Port > 0 {
			label = fmt.Sprintf("%s companion down (port %d)", label, status.Port)
		} else {
			label = fmt.Sprintf("%s companion down", label)
		}
		if status.Failed {
			if strings.TrimSpace(status.Failure) != "" {
				label += ": " + status.Failure
			} else {
				label += ": crash-loop cap reached"
			}
		}
		parts = append(parts, label)
	}
	message := "running; " + strings.Join(parts, "; ")
	if resourceName == "whisper" {
		message += " - STT unavailable, capacity reporting blind"
	}
	return message
}

func readCompanionPID(pidPath string) (int, bool) {
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 0, false
	}
	return pid, true
}

func companionAttemptsPath(dir, name string) string {
	return filepath.Join(dir, name+".attempts")
}

func companionFailedPath(dir, name string) string {
	return filepath.Join(dir, name+".failed")
}

func recordCompanionCrashAttempt(dir, name string, recoveryAttempts int) error {
	if recoveryAttempts <= 0 {
		return nil
	}
	now := time.Now()
	attempts := recentCompanionAttempts(companionAttemptsPath(dir, name), now)
	if len(attempts) >= recoveryAttempts {
		message := fmt.Sprintf("crash-loop cap reached: %d respawn attempts within %s", recoveryAttempts, companionCrashWindow)
		if err := os.WriteFile(companionFailedPath(dir, name), []byte(message), tuning.PermFile); err != nil {
			return err
		}
		appendCompanionLog(dir, name, "companion supervisor: "+message+"\n")
		return fmt.Errorf("%s", message)
	}
	attempts = append(attempts, now)
	return writeCompanionAttempts(companionAttemptsPath(dir, name), attempts)
}

func recentCompanionAttempts(path string, now time.Time) []time.Time {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	cutoff := now.Add(-companionCrashWindow)
	lines := strings.Split(string(data), "\n")
	attempts := make([]time.Time, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		unixNano, err := strconv.ParseInt(strings.TrimSpace(line), 10, 64)
		if err != nil {
			continue
		}
		at := time.Unix(0, unixNano)
		if at.After(cutoff) {
			attempts = append(attempts, at)
		}
	}
	return attempts
}

func writeCompanionAttempts(path string, attempts []time.Time) error {
	lines := make([]string, 0, len(attempts))
	for _, at := range attempts {
		lines = append(lines, strconv.FormatInt(at.UnixNano(), 10))
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), tuning.PermFile)
}

func readCompanionFailure(dir, name string) (bool, string) {
	data, err := os.ReadFile(companionFailedPath(dir, name))
	if err != nil {
		return false, ""
	}
	return true, strings.TrimSpace(string(data))
}

func clearCompanionCrashState(dir, name string) {
	clearCompanionFailure(dir, name)
	_ = os.Remove(companionAttemptsPath(dir, name))
}

func clearCompanionFailure(dir, name string) {
	_ = os.Remove(companionFailedPath(dir, name))
}

func appendCompanionLog(dir, name, line string) {
	logPath := filepath.Join(dir, name+".log")
	logf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, tuning.PermFile)
	if err != nil {
		return
	}
	defer logf.Close()
	_, _ = logf.WriteString(line)
}
