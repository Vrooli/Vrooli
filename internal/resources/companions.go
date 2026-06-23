package resources

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

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
func startCompanions(resourceName string, companions []ResourceCompanion, stderr io.Writer) {
	for _, c := range companions {
		if err := startCompanion(resourceName, c); err != nil {
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

func startCompanion(resourceName string, c ResourceCompanion) error {
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
	if pid, ok := readCompanionPID(pidPath); ok && process.IsPIDRunning(pid) {
		return nil
	}
	bin, err := exec.LookPath(c.Command)
	if err != nil {
		return fmt.Errorf("resolve %q on PATH: %w", c.Command, err)
	}
	logPath := filepath.Join(dir, c.Name+".log")
	logf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer logf.Close()

	cmd := exec.Command(bin, c.Args...)
	cmd.Stdout = logf
	cmd.Stderr = logf
	cmd.SysProcAttr = detachSysProcAttr()
	if err := cmd.Start(); err != nil {
		return err
	}
	pid := cmd.Process.Pid
	// Detach: the companion outlives this short-lived control process.
	_ = cmd.Process.Release()
	return os.WriteFile(pidPath, []byte(strconv.Itoa(pid)), 0o644)
}

func stopCompanion(resourceName string, c ResourceCompanion) error {
	dir, err := resolveCompanionDir(resourceName)
	if err != nil {
		return err
	}
	pidPath := filepath.Join(dir, c.Name+".pid")
	pid, ok := readCompanionPID(pidPath)
	if !ok {
		return nil // nothing tracked — already stopped
	}
	if process.IsPIDRunning(pid) {
		if err := terminateCompanion(pid); err != nil {
			return err
		}
	}
	return os.Remove(pidPath)
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
