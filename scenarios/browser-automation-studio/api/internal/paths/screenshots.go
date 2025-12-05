package paths

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// ResolveScreenshotsRoot returns an absolute path for storing screenshot assets.
func ResolveScreenshotsRoot(log *logrus.Logger) string {
	if value := strings.TrimSpace(os.Getenv("BAS_SCREENSHOTS_ROOT")); value != "" {
		if abs, err := filepath.Abs(value); err == nil {
			return abs
		}
		if log != nil {
			log.WithField("path", value).Warn("Using BAS_SCREENSHOTS_ROOT without normalization")
		}
		return value
	}

	cwd, err := os.Getwd()
	if err != nil {
		if log != nil {
			log.WithError(err).Warn("Failed to resolve working directory for screenshots root; using relative default")
		}
		return filepath.Join("scenarios", scenarioRoot, "data", "screenshots")
	}

	absCwd, err := filepath.Abs(cwd)
	if err == nil {
		for dir := absCwd; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
			parent := filepath.Dir(dir)
			if filepath.Base(dir) == scenarioRoot && filepath.Base(parent) == "scenarios" {
				screenshots := filepath.Join(dir, "data", "screenshots")
				if abs, err := filepath.Abs(screenshots); err == nil {
					return abs
				}
				return screenshots
			}
		}
	}

	root := filepath.Join(absCwd, "scenarios", scenarioRoot, "data", "screenshots")
	if abs, err := filepath.Abs(root); err == nil {
		return abs
	}
	return root
}
