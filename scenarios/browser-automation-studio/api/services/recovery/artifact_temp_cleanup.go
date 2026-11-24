package recovery

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
)

// removeAbandonedArtifactTempFiles removes only regular files created by
// api-core's WriteFileAtomic naming contract. The caller must hold the
// executionartifactlock and have revalidated a terminal recovery-owned row.
// Published manifests, inventories and referenced storage objects are never
// candidates because they do not use the .tmp- prefix.
func removeAbandonedArtifactTempFiles(root string, executionID uuid.UUID) (int, error) {
	if executionID == uuid.Nil || strings.TrimSpace(root) == "" {
		return 0, fmt.Errorf("artifact temp cleanup requires a root and execution UUID")
	}
	absRoot, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return 0, fmt.Errorf("resolve artifact root: %w", err)
	}
	absRoot, err = filepath.EvalSymlinks(absRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("resolve artifact root symlinks: %w", err)
	}
	rootHandle, err := os.OpenRoot(absRoot)
	if err != nil {
		return 0, fmt.Errorf("open artifact root: %w", err)
	}
	defer rootHandle.Close()
	dirInfo, err := rootHandle.Lstat(executionID.String())
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("inspect execution artifact directory: %w", err)
	}
	if !dirInfo.IsDir() || dirInfo.Mode()&os.ModeSymlink != 0 {
		return 0, fmt.Errorf("execution artifact path is not a regular directory")
	}
	executionRoot, err := rootHandle.OpenRoot(executionID.String())
	if err != nil {
		return 0, fmt.Errorf("open execution artifact directory: %w", err)
	}
	defer executionRoot.Close()
	openedInfo, err := executionRoot.Stat(".")
	if err != nil || !os.SameFile(dirInfo, openedInfo) {
		return 0, fmt.Errorf("execution artifact directory changed during cleanup admission")
	}
	dir, err := executionRoot.Open(".")
	if err != nil {
		return 0, fmt.Errorf("open execution artifact directory listing: %w", err)
	}
	entries, err := dir.ReadDir(-1)
	_ = dir.Close()
	if err != nil {
		return 0, fmt.Errorf("list execution artifact directory: %w", err)
	}
	referenced, err := executionwriter.ReadPurgeInventoryArtifactPaths(executionRoot, executionID)
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), ".tmp-") {
			continue
		}
		if _, exists := referenced[entry.Name()]; exists {
			continue
		}
		fileInfo, err := executionRoot.Lstat(entry.Name())
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return removed, fmt.Errorf("inspect atomic-write temp %q: %w", entry.Name(), err)
		}
		if !fileInfo.Mode().IsRegular() || fileInfo.Mode()&os.ModeSymlink != 0 {
			return removed, fmt.Errorf("atomic-write temp %q is not a regular file", entry.Name())
		}
		currentDirInfo, err := rootHandle.Lstat(executionID.String())
		if err != nil || !os.SameFile(dirInfo, currentDirInfo) {
			return removed, fmt.Errorf("execution artifact path changed during cleanup")
		}
		if err := executionRoot.Remove(entry.Name()); err != nil {
			return removed, fmt.Errorf("remove abandoned atomic-write temp %q: %w", entry.Name(), err)
		}
		removed++
	}
	return removed, nil
}
