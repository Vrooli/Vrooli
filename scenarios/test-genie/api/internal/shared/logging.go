package shared

import (
	"fmt"
	"io"
)

// DefaultLogWriter returns a safe io.Writer, defaulting to io.Discard if w is nil.
// This eliminates the repetitive nil-check pattern found in runner constructors.
func DefaultLogWriter(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	return w
}

// LogStep writes a structured step message to the log.
// Uses consistent formatting that can be parsed for observation streaming.
func LogStep(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	fmt.Fprintf(w, format+"\n", args...)
}

// LogSuccess writes a success message with a consistent marker.
func LogSuccess(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "[SUCCESS] ✅ %s\n", msg)
}

// LogInfo writes an info/progress message with a consistent marker.
func LogInfo(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "🔍 %s\n", msg)
}

// LogWarn writes a warning message with a structured marker.
func LogWarn(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "[WARNING] ⚠️ %s\n", msg)
}

// LogError writes an error message with a structured marker.
func LogError(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "[ERROR] ❌ %s\n", msg)
}
