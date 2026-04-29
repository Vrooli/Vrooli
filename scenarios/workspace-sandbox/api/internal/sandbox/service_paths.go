package sandbox

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/types"
)

// service_paths.go: mount-path bookkeeping + filesystem path validation.
//
// Mount paths split into two flavors:
//   - Persisted (LowerDir/UpperDir/WorkDir/MergedDir): written to the
//     repo at create/start time.
//   - Transient (HomeLowerDir/HomeUpperDir/HomeWorkDir/HomeMergedDir):
//     re-derived on every Get() because the home overlay isn't tracked
//     in the DB.
//
// inferMountPaths is the recovery seam: if a sandbox row was loaded
// without mount paths (legacy data, manual repair, partial migration),
// we attempt to recover them from the driver's BaseDir or from
// existing on-disk dirs before returning to the caller.

var errMissingUpperDir = errors.New("sandbox upper directory missing")

func (s *Service) ensureDiffDirectories(sandbox *types.Sandbox) error {
	if sandbox.UpperDir == "" {
		return errMissingUpperDir
	}
	if _, err := os.Stat(sandbox.UpperDir); err != nil {
		if os.IsNotExist(err) {
			return errMissingUpperDir
		}
		return &types.ValidationError{
			Field:   "upperDir",
			Message: "unable to access sandbox upper directory",
			Hint:    fmt.Sprintf("Check permissions for %s", sandbox.UpperDir),
		}
	}
	if sandbox.LowerDir == "" {
		return &types.ValidationError{
			Field:   "lowerDir",
			Message: "sandbox lower directory not initialized (internal error)",
			Hint:    "This indicates the sandbox was not properly created. Delete and recreate it.",
		}
	}
	if _, err := os.Stat(sandbox.LowerDir); err != nil {
		if os.IsNotExist(err) {
			return &types.ValidationError{
				Field:   "lowerDir",
				Message: "sandbox lower directory missing on disk",
				Hint:    "Project root may have moved or been deleted.",
			}
		}
		return &types.ValidationError{
			Field:   "lowerDir",
			Message: "unable to access sandbox lower directory",
			Hint:    fmt.Sprintf("Check permissions for %s", sandbox.LowerDir),
		}
	}
	return nil
}

func (s *Service) ensureDiffPaths(ctx context.Context, sandbox *types.Sandbox) error {
	updated := s.inferMountPaths(sandbox)
	if updated {
		// Best-effort persist of recovered paths for future operations.
		_ = s.repo.Update(ctx, sandbox)
	}

	if sandbox.UpperDir == "" {
		return &types.ValidationError{
			Field:   "upperDir",
			Message: "sandbox upper directory not initialized (internal error)",
			Hint:    "This indicates the sandbox was not properly created. Delete and recreate it.",
		}
	}

	if sandbox.LowerDir == "" {
		return &types.ValidationError{
			Field:   "lowerDir",
			Message: "sandbox lower directory not initialized (internal error)",
			Hint:    "This indicates the sandbox was not properly created. Delete and recreate it.",
		}
	}

	return nil
}

// inferMountPaths re-derives missing mount paths and re-discovers the
// transient home overlay. Returns true if anything was filled in
// (caller may persist).
//
// DOC: home-overlay seam — recovery path.
func (s *Service) inferMountPaths(sandbox *types.Sandbox) bool {
	updated := false
	if sandbox == nil {
		return false
	}

	if baseDir := s.driverBaseDir(); baseDir != "" {
		updated = applyPathsFromBaseDir(sandbox, baseDir) || updated
	}
	updated = applyPathsFromExistingDirs(sandbox) || updated

	// Home overlay paths are transient (not persisted to DB); recover
	// them from the per-sandbox subdir under HomeOverlayBaseDir if
	// HomeOverlayState says we have one. This is what makes bwrap's
	// home bind work on every Get(), not just immediately after Mount().
	if sandbox.HomeMergedDir == "" && sandbox.HomeOverlayState == types.HomeOverlayPresent {
		if homeBase := s.driverHomeOverlayBaseDir(); homeBase != "" && sandbox.ID != uuid.Nil {
			root := filepath.Join(homeBase, sandbox.ID.String())
			homeMerged := filepath.Join(root, "home-merged")
			if _, err := os.Stat(homeMerged); err == nil {
				sandbox.HomeMergedDir = homeMerged
				sandbox.HomeUpperDir = filepath.Join(root, "home-upper")
				sandbox.HomeWorkDir = filepath.Join(root, "home-work")
				sandbox.HomeLowerDir = os.Getenv("HOME")
				updated = true
			}
		}
	}
	return updated
}

func (s *Service) driverBaseDir() string {
	type baseDirProvider interface {
		BaseDir() string
	}
	if provider, ok := s.driver.(baseDirProvider); ok {
		return provider.BaseDir()
	}
	return ""
}

func (s *Service) driverHomeOverlayBaseDir() string {
	type homeOverlayBaseDirProvider interface {
		HomeOverlayBaseDir() string
	}
	if provider, ok := s.driver.(homeOverlayBaseDirProvider); ok {
		return provider.HomeOverlayBaseDir()
	}
	return ""
}

func applyPathsFromBaseDir(sandbox *types.Sandbox, baseDir string) bool {
	if baseDir == "" || sandbox.ID == uuid.Nil {
		return false
	}
	root := filepath.Join(baseDir, sandbox.ID.String())
	return applyDerivedPaths(sandbox, root)
}

func applyPathsFromExistingDirs(sandbox *types.Sandbox) bool {
	root := ""
	switch {
	case sandbox.MergedDir != "":
		root = filepath.Dir(sandbox.MergedDir)
	case sandbox.WorkDir != "":
		root = filepath.Dir(sandbox.WorkDir)
	case strings.HasSuffix(sandbox.LowerDir, string(filepath.Separator)+"original"):
		root = filepath.Dir(sandbox.LowerDir)
	}
	if root == "" {
		return false
	}
	return applyDerivedPaths(sandbox, root)
}

func applyDerivedPaths(sandbox *types.Sandbox, root string) bool {
	updated := false
	switch sandbox.DriverID {
	case string(driver.DriverCopy):
		if sandbox.LowerDir == "" {
			sandbox.LowerDir = filepath.Join(root, "original")
			updated = true
		}
		if sandbox.UpperDir == "" {
			sandbox.UpperDir = filepath.Join(root, "workspace")
			updated = true
		}
		if sandbox.WorkDir == "" {
			sandbox.WorkDir = filepath.Join(root, "meta")
			updated = true
		}
		if sandbox.MergedDir == "" {
			sandbox.MergedDir = filepath.Join(root, "workspace")
			updated = true
		}
	default:
		if sandbox.UpperDir == "" {
			sandbox.UpperDir = filepath.Join(root, "upper")
			updated = true
		}
		if sandbox.WorkDir == "" {
			sandbox.WorkDir = filepath.Join(root, "work")
			updated = true
		}
		if sandbox.MergedDir == "" {
			sandbox.MergedDir = filepath.Join(root, "merged")
			updated = true
		}
		if sandbox.LowerDir == "" && sandbox.ScopePath != "" {
			sandbox.LowerDir = sandbox.ScopePath
			updated = true
		}
	}
	return updated
}

// ValidatePath checks if a path is valid for use as a sandbox scope.
// Centralized validation: absolute, not a system directory, exists,
// is a directory, and within the project root.
func (s *Service) ValidatePath(ctx context.Context, p, projectRoot string) (*types.PathValidationResult, error) {
	result := &types.PathValidationResult{
		Path:        p,
		ProjectRoot: projectRoot,
	}

	if projectRoot == "" {
		projectRoot = s.config.DefaultProjectRoot
		result.ProjectRoot = projectRoot
	}

	if !filepath.IsAbs(p) {
		result.Valid = false
		result.Error = "Path must be absolute"
		return result, nil
	}

	cleanPath := filepath.Clean(p)
	dangerousPaths := []string{"/", "/bin", "/sbin", "/usr", "/etc", "/var", "/tmp", "/root", "/home"}
	for _, dangerous := range dangerousPaths {
		if cleanPath == dangerous {
			result.Valid = false
			result.Error = "Cannot use system directories"
			return result, nil
		}
	}

	info, err := os.Stat(p)
	if os.IsNotExist(err) {
		result.Valid = false
		result.Exists = false
		result.Error = "Path does not exist"
		return result, nil
	}
	if err != nil {
		result.Valid = false
		result.Error = fmt.Sprintf("Cannot access path: %v", err)
		return result, nil
	}

	result.Exists = true
	result.IsDirectory = info.IsDir()

	if !info.IsDir() {
		result.Valid = false
		result.Error = "Path must be a directory"
		return result, nil
	}

	if projectRoot != "" {
		cleanProjectRoot := filepath.Clean(projectRoot)
		if cleanPath != cleanProjectRoot && !strings.HasPrefix(cleanPath, cleanProjectRoot+string(filepath.Separator)) {
			result.Valid = false
			result.Error = fmt.Sprintf("Path must be within %s", projectRoot)
			result.WithinProjectRoot = false
			return result, nil
		}
		result.WithinProjectRoot = true
	}

	result.Valid = true
	return result, nil
}
