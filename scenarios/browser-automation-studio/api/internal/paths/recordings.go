package paths

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/storage"
)

const scenarioRoot = "browser-automation-studio"

// ResolveRecordingsRoot returns an absolute path for storing recording assets.
func ResolveRecordingsRoot(log *logrus.Logger) string {
	if value := strings.TrimSpace(os.Getenv("BAS_RECORDINGS_ROOT")); value != "" {
		if abs, err := filepath.Abs(value); err == nil {
			return abs
		}
		if log != nil {
			log.WithField("path", value).Warn("Using BAS_RECORDINGS_ROOT without normalization")
		}
		return value
	}

	if resolved := resolveScenarioStoragePath("recordings", legacyRecordingsRoot()); resolved != "" {
		return resolved
	}

	cwd, err := os.Getwd()
	if err != nil {
		if log != nil {
			log.WithError(err).Warn("Failed to resolve working directory for recordings root; using relative default")
		}
		return filepath.Join("scenarios", scenarioRoot, "data", "recordings")
	}

	absCwd, err := filepath.Abs(cwd)
	if err == nil {
		for dir := absCwd; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
			parent := filepath.Dir(dir)
			if filepath.Base(dir) == scenarioRoot && filepath.Base(parent) == "scenarios" {
				recordings := filepath.Join(dir, "data", "recordings")
				if abs, err := filepath.Abs(recordings); err == nil {
					return abs
				}
				return recordings
			}
		}
	}

	root := filepath.Join(absCwd, "scenarios", scenarioRoot, "data", "recordings")
	if abs, err := filepath.Abs(root); err == nil {
		return abs
	}
	return root
}

// ResolveSessionProfilesRoot returns an absolute path for storing persisted browser session profiles.
func ResolveSessionProfilesRoot(log *logrus.Logger) string {
	if value := strings.TrimSpace(os.Getenv("BAS_SESSION_PROFILES_ROOT")); value != "" {
		if abs, err := filepath.Abs(value); err == nil {
			return abs
		}
		if log != nil {
			log.WithField("path", value).Warn("Using BAS_SESSION_PROFILES_ROOT without normalization")
		}
		return value
	}

	if resolved := resolveScenarioStoragePath("session-profiles", legacySessionProfilesRoot()); resolved != "" {
		return resolved
	}

	cwd, err := os.Getwd()
	if err != nil {
		if log != nil {
			log.WithError(err).Warn("Failed to resolve working directory for session profiles root; using relative default")
		}
		return filepath.Join("scenarios", scenarioRoot, "data", "session-profiles")
	}

	absCwd, err := filepath.Abs(cwd)
	if err == nil {
		for dir := absCwd; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
			parent := filepath.Dir(dir)
			if filepath.Base(dir) == scenarioRoot && filepath.Base(parent) == "scenarios" {
				root := filepath.Join(dir, "data", "session-profiles")
				if abs, err := filepath.Abs(root); err == nil {
					return abs
				}
				return root
			}
		}
	}

	root := filepath.Join(absCwd, "scenarios", scenarioRoot, "data", "session-profiles")
	if abs, err := filepath.Abs(root); err == nil {
		return abs
	}
	return root
}

func resolveScenarioStoragePath(rel string, legacy string) string {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return ""
	}
	path, err := resolver.Path(storage.Options{ScenarioID: scenarioRoot}, storage.ClassData, rel)
	if err != nil {
		return ""
	}
	_ = migrateLegacyDir(legacy, path)
	return path
}

func legacyRecordingsRoot() string {
	return resolveLegacyScenarioDataPath("recordings")
}

func legacySessionProfilesRoot() string {
	return resolveLegacyScenarioDataPath("session-profiles")
}

func resolveLegacyScenarioDataPath(name string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return filepath.Join("scenarios", scenarioRoot, "data", name)
	}

	absCwd, err := filepath.Abs(cwd)
	if err != nil {
		return filepath.Join(cwd, "scenarios", scenarioRoot, "data", name)
	}

	for dir := absCwd; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		parent := filepath.Dir(dir)
		if filepath.Base(dir) == scenarioRoot && filepath.Base(parent) == "scenarios" {
			return filepath.Join(dir, "data", name)
		}
	}

	return filepath.Join(absCwd, "scenarios", scenarioRoot, "data", name)
}

func migrateLegacyDir(src, dst string) error {
	if src == "" || dst == "" || src == dst {
		return nil
	}
	if _, err := os.Stat(src); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if _, err := os.Stat(dst); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.Rename(src, dst)
}
