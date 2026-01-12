package bundle

import (
	"errors"
	"os"
	"path/filepath"
)

// defaultRuntimeResolver is the default implementation of RuntimeResolver.
type defaultRuntimeResolver struct{}

// Resolve locates and returns the absolute path to the runtime source directory.
func (r *defaultRuntimeResolver) Resolve() (string, error) {
	candidates := []string{}

	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, "..", "runtime"))
	}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "..", "runtime"),
			filepath.Join(exeDir, "..", "..", "runtime"),
		)
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return filepath.Abs(candidate)
		}
	}

	return "", errors.New("runtime source directory not found")
}
