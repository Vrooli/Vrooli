package langrecover

// Toolchain resolution for the recovery floor.
//
// The floor runs from a scheduler unit whose PATH is a short fixed list, not
// the operator's login shell. On this host that list omitted /usr/local/go/bin,
// so `go mod download` could not run from the unit and every attempt burned a
// breaker slot toward a six-hour suspension while nothing was actually wrong
// with the repository. A tool is therefore resolved through PATH first and
// then through the per-OS table below, which must cover every directory the
// platform package renders into the unit's PATH. langrecover cannot import
// that package (it must stay stdlib-only so dependency drift can never break
// the thing that repairs dependency drift), so the table is a copy; the API
// module, which imports both, holds the test that this table is a superset of
// the platform one.
//
// The Vrooli runtime-home bin directory is the one entry whose location is
// owned by the repository contract rather than by the OS. langrecover cannot
// resolve the contract, so callers resolve that directory and pass it in;
// hand-joining it here would ignore the contract's authority over the
// runtime home layout.

import (
	"context"
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
// directory; entries under it are skipped when home is empty. runtimeBin is
// the Vrooli runtime-home bin directory as resolved by the caller through the
// repository contract; it is searched last and skipped when empty.
func DefaultPathEntries(goos, home, runtimeBin string) []string {
	var entries []string
	if goos == "windows" {
		entries = append(entries,
			filepath.Join(os.Getenv("ProgramFiles"), "Go", "bin"),
			filepath.Join(os.Getenv("SystemRoot"), "System32"),
		)
		if home != "" {
			entries = append(entries,
				filepath.Join(home, "AppData", "Local", "Microsoft", "WinGet", "Links"),
				filepath.Join(home, "go", "bin"),
				filepath.Join(home, ".cargo", "bin"),
			)
		}
	} else {
		entries = append(entries, "/usr/local/go/bin", "/opt/homebrew/bin", "/usr/local/bin", "/usr/bin", "/bin")
		if home != "" {
			entries = append(entries,
				filepath.Join(home, "go", "bin"),
				filepath.Join(home, ".cargo", "bin"),
				filepath.Join(home, ".local", "bin"),
				filepath.Join(home, "bin"),
			)
		}
	}
	if runtimeBin = strings.TrimSpace(runtimeBin); runtimeBin != "" {
		entries = append(entries, runtimeBin)
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

// NewRunner returns the production Runner: each tool resolves through PATH
// and then entries (normally DefaultPathEntries), and the command runs in the
// supplied working directory with combined output.
func NewRunner(entries []string) Runner {
	entries = append([]string(nil), entries...)
	return func(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
		resolved, err := FindTool(name, exec.LookPath, entries)
		if err != nil {
			return nil, err
		}
		cmd := exec.CommandContext(ctx, resolved, args...)
		if dir != "" {
			cmd.Dir = dir
		}
		return cmd.CombinedOutput()
	}
}

func isExecutableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return filepath.Ext(path) == ".exe" || info.Mode()&0o111 != 0
}
