package existence

import (
	"fmt"
	"io"
	"os"
)

// ValidateDirs checks that all required directories exist in the scenario.
func ValidateDirs(scenarioDir string, dirs []string, logWriter io.Writer) Result {
	for _, rel := range dirs {
		abs := resolvePath(scenarioDir, rel)
		if err := ensureDir(abs); err != nil {
			logError(logWriter, "Missing directory: %s", rel)
			return FailMisconfiguration(
				err,
				fmt.Sprintf("Create the '%s' directory. See docs/phases/structure/README.md for required structure and customization options.", rel),
			)
		}
		logStep(logWriter, "  ✓ %s", rel)
	}
	return OKWithCount(len(dirs))
}

// ensureDir verifies that a path exists and is a directory.
func ensureDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("required directory missing: %s: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("expected directory but found file: %s", path)
	}
	return nil
}

// logStep writes a validation step message.
func logStep(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	fmt.Fprintf(w, format+"\n", args...)
}

// logError writes an error message.
func logError(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "[ERROR] ❌ %s\n", msg)
}

// logSuccess writes a success message.
func logSuccess(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "[SUCCESS] ✅ %s\n", msg)
}

// logInfo writes an info message.
func logInfo(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "🔍 %s\n", msg)
}

// logWarn writes a warning message.
func logWarn(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "[WARNING] ⚠️ %s\n", msg)
}
