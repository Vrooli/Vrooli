package langrecover

// Toolchain resolution for the recovery floor.
//
// The floor runs from a scheduler unit whose PATH is a short fixed list, not
// the operator's login shell. On this host that list omitted /usr/local/go/bin,
// so `go mod download` could not run from the unit and every attempt burned a
// breaker slot toward a six-hour suspension while nothing was actually wrong
// with the repository. A tool is therefore resolved through PATH first and
// then through the per-OS table below, which mirrors the default PATH the
// platform package renders into the unit. langrecover cannot import that
// package (it must stay stdlib-only so dependency drift can never break the
// thing that repairs dependency drift), so the table is copied here and the
// equality test lives beside the original in packages/platform-go.

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrToolMissing is returned when a tool is on neither PATH nor the table.
var ErrToolMissing = errors.New("tool not found")

// DefaultPathEntries returns the directories a scheduler unit is expected to
// search for toolchains on the given OS, in order. home is the account's home
// directory; entries under it are skipped when home is empty.
func DefaultPathEntries(goos, home string) []string {
	var entries []string
	switch goos {
	case "windows":
		entries = append(entries,
			filepath.Join(os.Getenv("ProgramFiles"), "Go", "bin"),
			filepath.Join(os.Getenv("SystemRoot"), "System32"),
		)
		if home != "" {
			entries = append(entries,
				filepath.Join(home, "AppData", "Local", "Microsoft", "WinGet", "Links"),
				filepath.Join(home, "go", "bin"),
				filepath.Join(home, ".cargo", "bin"),
				filepath.Join(home, ".vrooli", "bin"),
			)
		}
	case "darwin":
		entries = append(entries, "/usr/local/go/bin", "/opt/homebrew/bin", "/usr/local/bin", "/usr/bin", "/bin")
		if home != "" {
			entries = append(entries,
				filepath.Join(home, "go", "bin"),
				filepath.Join(home, ".cargo", "bin"),
				filepath.Join(home, ".local", "bin"),
				filepath.Join(home, ".vrooli", "bin"),
			)
		}
	default:
		entries = append(entries, "/usr/local/go/bin", "/usr/local/bin", "/usr/bin", "/bin")
		if home != "" {
			entries = append(entries,
				filepath.Join(home, "go", "bin"),
				filepath.Join(home, ".cargo", "bin"),
				filepath.Join(home, ".local", "bin"),
				filepath.Join(home, ".vrooli", "bin"),
			)
		}
	}
	return entries
}

// FindTool resolves name to an executable path: lookPath first, then each
// entry of the table. The returned error lists everything that was searched
// so an operator can see which seam was expected to supply the tool.
func FindTool(name string, lookPath func(string) (string, error), entries []string) (string, error) {
	if path, err := lookPath(name); err == nil && strings.TrimSpace(path) != "" {
		return path, nil
	}
	for _, dir := range entries {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, name)
		if isExecutableFile(candidate) {
			return candidate, nil
		}
		if isExecutableFile(candidate + ".exe") {
			return candidate + ".exe", nil
		}
	}
	return "", &ToolMissingError{Name: name, Searched: append([]string{"PATH"}, entries...)}
}

// ToolMissingError names the tool and the places searched.
type ToolMissingError struct {
	Name     string
	Searched []string
}

func (e *ToolMissingError) Error() string {
	return e.Name + " not found; searched " + strings.Join(e.Searched, ", ")
}

func (e *ToolMissingError) Unwrap() error { return ErrToolMissing }

// ResolveTool resolves name for the current process: PATH, then the table for
// this OS and the current home directory.
func ResolveTool(name, goos string) (string, error) {
	home, _ := os.UserHomeDir()
	return FindTool(name, exec.LookPath, DefaultPathEntries(goos, home))
}

func isExecutableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return filepath.Ext(path) == ".exe" || info.Mode()&0o111 != 0
}
