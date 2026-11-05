package bundle

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vrooli/envkit-go"
)

// defaultCLIStager is the default implementation of CLIStager.
type defaultCLIStager struct {
	fileOps         FileOperations
	runtimeResolver RuntimeResolver
}

// Stage copies CLI binaries into bundle/bin and writes a thin vrooli shim.
func (s *defaultCLIStager) Stage(bundleRoot, platform string) error {
	binDir := filepath.Join(bundleRoot, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return err
	}

	cliDir := filepath.Join(bundleRoot, "cli")
	if info, err := os.Stat(cliDir); err == nil && info.IsDir() {
		if err := filepath.WalkDir(cliDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			dst := filepath.Join(binDir, filepath.Base(path))
			if err := s.fileOps.CopyPath(path, dst); err != nil {
				return err
			}
			return os.Chmod(dst, 0o755)
		}); err != nil {
			return err
		}
	}

	resolver := s.runtimeResolver
	if resolver == nil {
		resolver = &defaultRuntimeResolver{}
	}
	runtimeDir, err := resolver.Resolve()
	if err != nil {
		return fmt.Errorf("resolve runtime source for vrooli shim: %w", err)
	}
	goos, goarch, err := (&defaultPlatformResolver{}).ParseKey(platform)
	if err != nil {
		return err
	}
	shimName := "vrooli"
	if goos == "windows" {
		shimName += ".exe"
	}
	shim := filepath.Join(binDir, shimName)
	// Build away from the final path: the copied CLI may already occupy
	// bin/vrooli, and `go build -o` rejects an existing non-object file.
	tmp, err := os.CreateTemp(binDir, ".vrooli-shim-*")
	if err != nil {
		return fmt.Errorf("create temporary vrooli shim output: %w", err)
	}
	tmpPath := tmp.Name()
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close temporary vrooli shim output: %w", err)
	}
	defer os.Remove(tmpPath)
	cmd := exec.Command("go", "build", "-o", tmpPath, "./cmd/vrooli-shim")
	cmd.Dir = runtimeDir
	cmd.Env = envkit.Toolchain(envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, envkit.Env{"CGO_ENABLED=0", "GOOS=" + goos, "GOARCH=" + goarch}), envkit.ToolchainOptions{})
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build vrooli shim: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if err := os.Rename(tmpPath, shim); err != nil {
		return fmt.Errorf("publish vrooli shim: %w", err)
	}
	return os.Chmod(shim, 0o755)
}
