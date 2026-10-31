// Package cliresolve resolves installed Vrooli command-line programs without
// depending on a repository checkout or on the installer that produced them.
package cliresolve

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// NotFoundError explains an installed CLI lookup failure without reducing it
// to exec.ExitError (and, consequently, exit code 127). Searched is ordered
// and contains every concrete directory inspected by Resolver.
type NotFoundError struct {
	Name     string
	Searched []string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("installed CLI %q was not found; searched: %s", e.Name, strings.Join(e.Searched, ", "))
}

// IsNotFound reports whether err is a typed installed-CLI lookup failure.
func IsNotFound(err error) bool {
	var target *NotFoundError
	return errors.As(err, &target)
}

// Resolver locates binaries installed below a user's runtime home. It does
// not read the repository contract: an installed node has no repository, and
// the runtime-home binary location is intentionally stable.
type Resolver struct {
	home string
}

// New creates a resolver for home. An empty home uses the process user home.
func New(home string) *Resolver {
	home = filepath.Clean(strings.TrimSpace(home))
	if home == "" {
		if resolved, err := os.UserHomeDir(); err == nil {
			home = resolved
		}
	}
	return &Resolver{home: home}
}

// Home returns the normalized home directory used by the resolver.
func (r *Resolver) Home() string { return r.home }

// SearchDirs returns the ordered directories searched for installed CLIs.
func (r *Resolver) SearchDirs() []string {
	if r == nil || strings.TrimSpace(r.home) == "" {
		return nil
	}
	return []string{filepath.Join(r.home, ".vrooli", "bin")}
}

// Executable returns an absolute, executable path for name.
func (r *Resolver) Executable(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || filepath.Base(name) != name || strings.ContainsAny(name, `/\\`) {
		return "", fmt.Errorf("invalid CLI name %q", name)
	}
	dirs := r.SearchDirs()
	candidates := make([]string, 0, len(dirs)*2)
	for _, dir := range dirs {
		candidates = append(candidates, filepath.Join(dir, name))
		if runtime.GOOS == "windows" && !strings.EqualFold(filepath.Ext(name), ".exe") {
			candidates = append(candidates, filepath.Join(dir, name+".exe"))
		}
	}
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		if runtime.GOOS != "windows" && info.Mode()&0o111 == 0 {
			continue
		}
		absolute, err := filepath.Abs(candidate)
		if err != nil {
			return "", fmt.Errorf("make installed CLI %q absolute: %w", name, err)
		}
		return filepath.Clean(absolute), nil
	}
	return "", &NotFoundError{Name: name, Searched: dirs}
}

// Executable resolves one installed CLI using a home directory alone.
func Executable(home, name string) (string, error) {
	return New(home).Executable(name)
}
