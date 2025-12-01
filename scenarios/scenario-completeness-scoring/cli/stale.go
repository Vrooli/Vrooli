package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/buildinfo"
	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) warnIfBinaryStale() {
	if buildFingerprint == "" || buildFingerprint == "unknown" {
		return
	}
	srcRoot := cliutil.ResolveSourceRoot(buildSourceRoot, genericSourceRootEnvVar, legacySourceRootEnvVar)
	if srcRoot == "" {
		return
	}
	var skipNames []string
	if executable, err := os.Executable(); err == nil {
		skipNames = append(skipNames, filepath.Base(executable))
	}
	fingerprint, err := buildinfo.ComputeFingerprint(srcRoot, skipNames...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: unable to verify CLI freshness: %v\n", err)
		return
	}

	if fingerprint != buildFingerprint {
		if a.maybeAutoRebuild(srcRoot, fingerprint) {
			return
		}
		fmt.Fprintf(os.Stderr, "Warning: scenario-completeness-scoring CLI binary built at %s (fingerprint %s) does not match the current sources (fingerprint %s).\nRun scenarios/scenario-completeness-scoring/cli/install.sh to rebuild the CLI before rerunning this command.\n\n", buildTimestamp, buildFingerprint, fingerprint)
	}
}

func (a *App) maybeAutoRebuild(srcRoot, currentFingerprint string) bool {
	if !a.canAutoRebuild(srcRoot) {
		return false
	}

	repoRoot, ok := findRepositoryRoot(srcRoot)
	if !ok {
		return false
	}

	executable, err := os.Executable()
	if err != nil {
		return false
	}

	installerArgs := []string{
		"go", "run", "./cmd/cli-installer",
		"--module", srcRoot,
		"--output", executable,
		"--force", "true",
	}
	cmd := exec.Command(installerArgs[0], installerArgs[1:]...)
	cmd.Dir = filepath.Join(repoRoot, "packages", "cli-core")
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: CLI auto-rebuild failed: %v\n", err)
		return false
	}

	fmt.Fprintf(os.Stderr, "CLI rebuilt from current sources (fingerprint %s); restarting command...\n", currentFingerprint)
	a.restartWithExecutable(executable)
	return true
}

func (a *App) canAutoRebuild(srcRoot string) bool {
	if _, err := exec.LookPath("go"); err != nil {
		return false
	}
	if !strings.Contains(srcRoot, string(os.PathSeparator)+"scenarios"+string(os.PathSeparator)) {
		return false
	}
	return true
}

func findRepositoryRoot(start string) (string, bool) {
	dir := filepath.Clean(start)
	for {
		candidate := filepath.Join(dir, "packages", "cli-core")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}

func (a *App) restartWithExecutable(executable string) {
	cmd := exec.Command(executable, os.Args[1:]...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Error restarting CLI: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
