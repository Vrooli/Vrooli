package lifecycle

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vrooli/vrooli/internal/tuning"

	platform "github.com/vrooli/platform-go"
)

func stageArtifact(target string, directory bool) (stageRoot, staged string, cleanup func(), err error) {
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

func swapArtifact(staged, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), tuning.PermDir); err != nil {
		return err
	}
	if err := platform.AtomicReplace(staged, target); err != nil {
		return fmt.Errorf("atomic artifact replace %s: %w", target, err)
	}
	return nil
}
