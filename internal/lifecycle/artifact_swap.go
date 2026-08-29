package lifecycle

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/tuning"

	platform "github.com/vrooli/platform-go"
)

func stageArtifact(target string, directory bool) (stageRoot, staged string, cleanup func(), err error) {
	if err := reapArtifactStages(filepath.Dir(target)); err != nil {
		return "", "", func() {}, err
	}
	stageRoot, err = os.MkdirTemp(filepath.Dir(target), ".vrooli-artifact-stage-")
	if err != nil {
		return "", "", func() {}, err
	}
	cleanup = func() { _ = os.RemoveAll(stageRoot) }
	if directory {
		return stageRoot, stageRoot, cleanup, nil
	}
	return stageRoot, filepath.Join(stageRoot, filepath.Base(target)), cleanup, nil
}

// reapArtifactStages removes abandoned staging directories left by a process
// that was interrupted before its deferred cleanup ran. Only the exact
// lifecycle-owned prefix is eligible; ordinary sibling directories are never
// touched.
func reapArtifactStages(parent string) error {
	entries, err := os.ReadDir(parent)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), ".vrooli-artifact-stage-") {
			continue
		}
		if err := os.RemoveAll(filepath.Join(parent, entry.Name())); err != nil {
			return fmt.Errorf("reap abandoned artifact stage %s: %w", entry.Name(), err)
		}
	}
	return nil
}

func swapArtifact(staged, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), tuning.PermDir); err != nil {
		return err
	}
	if err := platform.AtomicReplace(staged, target); err != nil {
		return fmt.Errorf("atomic artifact replace %s: %w", target, err)
	}
	return nil
}
