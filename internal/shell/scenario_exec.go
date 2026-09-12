package shell

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/tuning"
)

func IsExecutable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Mode()&tuning.PermExecuteMask != 0
}

func WriterSupportsStreaming(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func OpenURL(lookPath func(string) (string, error), run func(Spec) error, url string) error {
	switch runtime.GOOS {
	case string(hostreqspec.PlatformLinux):
		if binary, err := lookPath("xdg-open"); err == nil {
			return run(Spec{Name: binary, Args: []string{url}})
		}
		for _, browser := range []string{"firefox", "google-chrome", "chromium"} {
			if binary, err := lookPath(browser); err == nil {
				return run(Spec{Name: binary, Args: []string{url}})
			}
		}
		return fmt.Errorf("no browser found for %s", url)
	case string(hostreqspec.PlatformDarwin):
		return run(Spec{Name: "open", Args: []string{url}})
	case string(hostreqspec.PlatformWindows):
		return run(Spec{Name: "cmd", Args: []string{"/c", "start", "", url}})
	default:
		return fmt.Errorf("unsupported platform for opening URLs: %s", runtime.GOOS)
	}
}

func LaunchDetachedScenario(executable, root string, flags, env []string, args ...string) error {
	commandArgs := []string{"scenario"}
	commandArgs = append(commandArgs, args...)
	commandArgs = append(commandArgs, flags...)

	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer devNull.Close()

	cmd := Command(Spec{
		Name:  executable,
		Args:  commandArgs,
		Dir:   root,
		Env:   UnsetEnvKeys(env, "VROOLI_SANDBOX_ID", "VROOLI_SANDBOX_MERGED", "VROOLI_SANDBOX_SCOPE", "SANDBOX_MERGED_DIR"),
		Stdin: devNull, Stdout: devNull, Stderr: devNull,
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
