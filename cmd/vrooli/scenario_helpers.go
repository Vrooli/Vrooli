package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/shell"
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
	runScenarioSubprocessFn = runScenarioSubprocess
	scenarioExecutableFn    = os.Executable
)

func runScenarioSubprocess(spec scenarioSubprocessSpec) error {
	cmd := shell.CommandWithDefaults(shell.Spec{
		Name:   spec.name,
		Args:   spec.args,
		Dir:    spec.dir,
		Env:    spec.env,
		Stdin:  spec.stdin,
		Stdout: spec.stdout,
		Stderr: spec.stderr,
	})
	return cmd.Run()
}

func (app *App) locateTestGenieCLI(root, home string) (string, error) {
	if override := strings.TrimSpace(os.Getenv("VROOLI_TEST_GENIE_CLI")); override != "" {
		if isExecutable(override) {
			return override, nil
		}
	}

	homeCLI := filepath.Join(config.VrooliDir(home), "bin", "test-genie")
	if isExecutable(homeCLI) {
		return homeCLI, nil
	}

	if pathCLI, err := app.lookPath("test-genie"); err == nil && isExecutable(pathCLI) {
		return pathCLI, nil
	}

	repoCLI := filepath.Join(root, "scenarios", "test-genie", "cli", "test-genie")
	if isExecutable(repoCLI) {
		return repoCLI, nil
	}

	return "", fmt.Errorf("test-genie CLI not found (checked VROOLI_TEST_GENIE_CLI, PATH, %s, and %s)", homeCLI, repoCLI)
}

func (app *App) locateScenarioCompletenessCLI(root string) (string, error) {
	if pathCLI, err := app.lookPath("scenario-completeness-scoring"); err == nil && isExecutable(pathCLI) {
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

func (app *App) openScenarioURL(url string) error {
	switch runtime.GOOS {
	case "linux":
		if binary, err := app.lookPath("xdg-open"); err == nil {
			return app.runScenarioSubprocess(scenarioSubprocessSpec{name: binary, args: []string{url}})
		}
		for _, browser := range []string{"firefox", "google-chrome", "chromium"} {
			if binary, err := app.lookPath(browser); err == nil {
				return app.runScenarioSubprocess(scenarioSubprocessSpec{name: binary, args: []string{url}})
			}
		}
		return fmt.Errorf("no browser found for %s", url)
	case "darwin":
		return app.runScenarioSubprocess(scenarioSubprocessSpec{name: "open", args: []string{url}})
	case "windows":
		return app.runScenarioSubprocess(scenarioSubprocessSpec{name: "cmd", args: []string{"/c", "start", "", url}})
	default:
		return fmt.Errorf("unsupported platform for opening URLs: %s", runtime.GOOS)
	}
}

func (app *App) launchDetachedScenario(root string, globals globalOptions, args ...string) error {
	executable, err := app.scenarioExecutable()
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

	cmd := shell.Command(shell.Spec{
		Name: executable,
		Args: commandArgs,
		Dir:  root,
		Env: unsetEnvKeys(app.commandEnv(root, globals),
			"VROOLI_SANDBOX_ID",
			"VROOLI_SANDBOX_MERGED",
			"VROOLI_SANDBOX_SCOPE",
			"SANDBOX_MERGED_DIR",
		),
		Stdin:  devNull,
		Stdout: devNull,
		Stderr: devNull,
	})
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
