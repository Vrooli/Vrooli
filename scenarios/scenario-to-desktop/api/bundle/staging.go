package bundle

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	cmd := exec.Command("go", "build", "-o", shim, "./cmd/vrooli-shim")
	cmd.Dir = runtimeDir
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+goos, "GOARCH="+goarch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build vrooli shim: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return os.Chmod(shim, 0o755)
}
