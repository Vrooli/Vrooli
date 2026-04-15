package projectstate

import (
	"os"
	"path/filepath"
)

const setupStateDirRel = ".vrooli/state/setup"

func SetupStateDir(root string) string {
	return filepath.Join(root, filepath.FromSlash(setupStateDirRel))
}

func SetupCompletePath(root string) string {
	return filepath.Join(SetupStateDir(root), ".setup-complete")
}

func ResourcesPopulatedPath(root string) string {
	return filepath.Join(SetupStateDir(root), ".resources-populated")
}

func ResourcePopulatedPath(root, resource string) string {
	return filepath.Join(SetupStateDir(root), "."+resource+"-populated")
}

func HasSetupComplete(root string) bool {
	return fileExists(SetupCompletePath(root))
}

func HasResourcesPopulated(root string) bool {
	return fileExists(ResourcesPopulatedPath(root))
}

func HasResourcePopulated(root, resource string) bool {
	return fileExists(ResourcePopulatedPath(root, resource))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
