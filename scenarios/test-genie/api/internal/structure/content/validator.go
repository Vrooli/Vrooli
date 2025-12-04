package content

import (
	"fmt"
	"io"
)

// ManifestValidator validates the service.json manifest content.
type ManifestValidator interface {
	// Validate checks the service.json content and structure.
	Validate() Result
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
