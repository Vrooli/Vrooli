package platform

import (
	"fmt"
	"os"
	"path/filepath"
)

// AtomicReplace replaces target with staged using platform-specific artifact
// semantics. Callers do not import OS-specific rename behavior.
func AtomicReplace(staged, target string) error {
	return atomicReplace(staged, target)
}

func atomicRenameReplace(staged, target string) error {
	displaced := filepath.Join(filepath.Dir(target), ".vrooli-displaced-"+filepath.Base(target))
	if err := os.RemoveAll(displaced); err != nil {
		return err
	}
	targetMoved := false
	if err := os.Rename(target, displaced); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("move current artifact aside: %w", err)
		}
	} else {
		targetMoved = true
	}
	if err := os.Rename(staged, target); err != nil {
		if targetMoved {
			_ = os.Rename(displaced, target)
		}
		return fmt.Errorf("install staged artifact: %w", err)
	}
	if targetMoved {
		_ = os.RemoveAll(displaced)
	}
	return nil
}
