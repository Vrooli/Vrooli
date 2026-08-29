package scenarioexec

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/api-core/scenariocli"
	"github.com/vrooli/platform-go"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/shell"
)

type SubprocessSpec struct {
	Name   string
	Args   []string
	Dir    string
	Env    []string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func RunSubprocess(spec SubprocessSpec) error {
	cmd := shell.CommandWithDefaults(shell.Spec{
		Name:   spec.Name,
		Args:   spec.Args,
		Dir:    spec.Dir,
		Env:    spec.Env,
		Stdin:  spec.Stdin,
		Stdout: spec.Stdout,
		Stderr: spec.Stderr,
	})
	return cmd.Run()
}

func IsExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode()&tuning.PermExecuteMask != 0
}

func WriterSupportsStreaming(w io.Writer) bool {
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

func LocateTestGenieCLI(lookPath func(string) (string, error), root, home string) (string, error) {
	if override := strings.TrimSpace(os.Getenv("VROOLI_TEST_GENIE_CLI")); override != "" {
		if IsExecutable(override) {
			return override, nil
		}
	}
	_ = lookPath
	path, err := scenariocli.ResolveExecutable(root, home, "test-genie")
	if err != nil {
		homeCLI, _ := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
		homeCLI = filepath.Join(homeCLI, "test-genie")
		return "", fmt.Errorf("test-genie CLI not found via manifest-driven resolution (checked VROOLI_TEST_GENIE_CLI and attempted %s): %w", homeCLI, err)
	}
	return path, nil
}

func LocateScenarioCompletenessCLI(lookPath func(string) (string, error), root string) (string, error) {
	_ = lookPath
	home, _ := config.HomeDir()
	path, err := scenariocli.ResolveExecutable(root, home, "scenario-completeness-scoring")
	if err != nil {
		homeCLI, _ := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
		homeCLI = filepath.Join(homeCLI, "scenario-completeness-scoring")
		return "", fmt.Errorf("scenario-completeness-scoring CLI not found via manifest-driven resolution (attempted %s): %w", homeCLI, err)
	}
	return path, nil
}

func OpenURL(lookPath func(string) (string, error), run func(SubprocessSpec) error, url string) error {
	switch runtime.GOOS {
	case string(hostreqspec.PlatformLinux):
		if binary, err := lookPath("xdg-open"); err == nil {
			return run(SubprocessSpec{Name: binary, Args: []string{url}})
		}
		for _, browser := range []string{"firefox", "google-chrome", "chromium"} {
			if binary, err := lookPath(browser); err == nil {
				return run(SubprocessSpec{Name: binary, Args: []string{url}})
			}
		}
		return fmt.Errorf("no browser found for %s", url)
	case string(hostreqspec.PlatformDarwin):
		return run(SubprocessSpec{Name: "open", Args: []string{url}})
	case string(hostreqspec.PlatformWindows):
		return run(SubprocessSpec{Name: "cmd", Args: []string{"/c", "start", "", url}})
	default:
		return fmt.Errorf("unsupported platform for opening URLs: %s", runtime.GOOS)
	}
}

func LaunchDetachedScenario(executable, root string, globals rootcli.GlobalOptions, env []string, args ...string) error {
	commandArgs := []string{"scenario"}
	commandArgs = append(commandArgs, args...)
	commandArgs = append(commandArgs, rootcli.PassthroughFlags(globals, commandArgs)...)

	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer devNull.Close()

	cmd := shell.Command(shell.Spec{
		Name: executable,
		Args: commandArgs,
		Dir:  root,
		Env: UnsetEnvKeys(env,
			"VROOLI_SANDBOX_ID",
			"VROOLI_SANDBOX_MERGED",
			"VROOLI_SANDBOX_SCOPE",
			"SANDBOX_MERGED_DIR",
		),
		Stdin:  devNull,
		Stdout: devNull,
		Stderr: devNull,
	})
	if err := platform.ConfigureCommand(cmd, platform.ProcessOptions{Detached: true}); err != nil {
		return err
	}
	return cmd.Start()
}

func UnsetEnvKeys(env []string, keys ...string) []string {
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
