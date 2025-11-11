package paths

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
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
