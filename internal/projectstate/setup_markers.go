package projectstate

import (
	"os"
	"path/filepath"
	"strings"
)

const setupStateDirRel = ".vrooli/state/setup"

func SetupStateDir(root string) string {
	return filepath.Join(root, filepath.FromSlash(setupStateDirRel))
}

func legacySetupCompletePath(root string) string {
	return filepath.Join(root, "data", ".setup-complete")
}

func legacyResourcesPopulatedPath(root string) string {
	return filepath.Join(root, "data", ".resources-populated")
}

func legacyResourcePopulatedPath(root, resource string) string {
	return filepath.Join(root, "data", "."+strings.TrimSpace(resource)+"-populated")
}

func SetupCompletePath(root string) string {
	return filepath.Join(SetupStateDir(root), ".setup-complete")
}

func ResourcesPopulatedPath(root string) string {
	return filepath.Join(SetupStateDir(root), ".resources-populated")
}

func ResourcePopulatedPath(root, resource string) string {
	return filepath.Join(SetupStateDir(root), "."+strings.TrimSpace(resource)+"-populated")
}

func HasSetupComplete(root string) bool {
	return markerExists(SetupCompletePath(root), legacySetupCompletePath(root))
}

func HasResourcesPopulated(root string) bool {
	return markerExists(ResourcesPopulatedPath(root), legacyResourcesPopulatedPath(root))
}

func HasResourcePopulated(root, resource string) bool {
	return markerExists(ResourcePopulatedPath(root, resource), legacyResourcePopulatedPath(root, resource))
}

func markerExists(currentPath, legacyPath string) bool {
	if fileExists(currentPath) {
		return true
	}
	if strings.TrimSpace(legacyPath) == "" || !fileExists(legacyPath) {
		return false
	}
	_ = promoteLegacyMarker(currentPath, legacyPath)
	return true
}

func promoteLegacyMarker(currentPath, legacyPath string) error {
	data, err := os.ReadFile(legacyPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(currentPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(currentPath, data, 0o644); err != nil {
		return err
	}
	if err := os.Remove(legacyPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
