package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vrooli/vrooli/internal/config"
)

type scenarioSubprocessSpec struct {
	name   string
	args   []string
	dir    string
	env    []string
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

var (
	runScenarioSubprocessFn  = runScenarioSubprocess
	scenarioOpenURLFn        = openScenarioURL
	scenarioLaunchDetachedFn = launchDetachedScenario
	scenarioExecutableFn     = os.Executable
)

func runScenarioSubprocess(spec scenarioSubprocessSpec) error {
	cmd := exec.Command(spec.name, spec.args...)
	cmd.Dir = spec.dir
	cmd.Env = spec.env
	cmd.Stdin = spec.stdin
	cmd.Stdout = spec.stdout
	cmd.Stderr = spec.stderr
	if cmd.Stdin == nil {
		cmd.Stdin = os.Stdin
	}
	if cmd.Stdout == nil {
		cmd.Stdout = os.Stdout
	}
	if cmd.Stderr == nil {
		cmd.Stderr = os.Stderr
	}
	return cmd.Run()
}

func locateTestGenieCLI(root, home string) (string, error) {
	if override := strings.TrimSpace(os.Getenv("VROOLI_TEST_GENIE_CLI")); override != "" {
		if isExecutable(override) {
			return override, nil
		}
	}

	homeCLI := filepath.Join(config.VrooliDir(home), "bin", "test-genie")
	if isExecutable(homeCLI) {
		return homeCLI, nil
	}

	if pathCLI, err := lookPathFn("test-genie"); err == nil && isExecutable(pathCLI) {
		return pathCLI, nil
	}

	repoCLI := filepath.Join(root, "scenarios", "test-genie", "cli", "test-genie")
	if isExecutable(repoCLI) {
		return repoCLI, nil
	}

	return "", fmt.Errorf("test-genie CLI not found (checked VROOLI_TEST_GENIE_CLI, PATH, %s, and %s)", homeCLI, repoCLI)
}

func locateScenarioCompletenessCLI(root string) (string, error) {
	if pathCLI, err := lookPathFn("scenario-completeness-scoring"); err == nil && isExecutable(pathCLI) {
		return pathCLI, nil
	}

	repoCLI := filepath.Join(root, "scenarios", "scenario-completeness-scoring", "cli", "scenario-completeness-scoring")
	if isExecutable(repoCLI) {
		return repoCLI, nil
	}

	return "", fmt.Errorf("scenario-completeness-scoring CLI not found (checked PATH and %s)", repoCLI)
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode()&0o111 != 0
}

func writerSupportsStreaming(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func openScenarioURL(url string) error {
	switch runtime.GOOS {
	case "linux":
		if binary, err := lookPathFn("xdg-open"); err == nil {
			return runScenarioSubprocessFn(scenarioSubprocessSpec{name: binary, args: []string{url}})
		}
		for _, browser := range []string{"firefox", "google-chrome", "chromium"} {
			if binary, err := lookPathFn(browser); err == nil {
				return runScenarioSubprocessFn(scenarioSubprocessSpec{name: binary, args: []string{url}})
			}
		}
		return fmt.Errorf("no browser found for %s", url)
	case "darwin":
		return runScenarioSubprocessFn(scenarioSubprocessSpec{name: "open", args: []string{url}})
	case "windows":
		return runScenarioSubprocessFn(scenarioSubprocessSpec{name: "cmd", args: []string{"/c", "start", "", url}})
	default:
		return fmt.Errorf("unsupported platform for opening URLs: %s", runtime.GOOS)
	}
}

func launchDetachedScenario(root string, globals globalOptions, args ...string) error {
	executable, err := scenarioExecutableFn()
	if err != nil {
		return err
	}

	commandArgs := []string{"scenario"}
	commandArgs = append(commandArgs, args...)
	commandArgs = append(commandArgs, passthroughFlags(globals, commandArgs)...)

	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer devNull.Close()

	cmd := exec.Command(executable, commandArgs...)
	cmd.Dir = root
	cmd.Env = unsetEnvKeys(commandEnv(root, globals),
		forceBashEnvVar,
		"VROOLI_SANDBOX_ID",
		"VROOLI_SANDBOX_MERGED",
		"VROOLI_SANDBOX_SCOPE",
		"SANDBOX_MERGED_DIR",
	)
	cmd.Stdin = devNull
	cmd.Stdout = devNull
	cmd.Stderr = devNull
	cmd.SysProcAttr = detachedProcessAttr()
	return cmd.Start()
}

func unsetEnvKeys(env []string, keys ...string) []string {
	if len(keys) == 0 {
		return append([]string(nil), env...)
	}
	remove := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		remove[key] = struct{}{}
	}
	filtered := make([]string, 0, len(env))
	for _, entry := range env {
		key := entry
		if idx := strings.IndexByte(entry, '='); idx >= 0 {
			key = entry[:idx]
		}
		if _, ok := remove[key]; ok {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}
