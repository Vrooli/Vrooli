// Package logx is the canonical seam for structured logging.
//
// Production wires logx.Std (wrapping *log.Logger) once in main.go and
// threads it into every domain that emits log lines. Tests substitute
// mocks.FakeLogger to assert what was logged without polluting stderr.
//
// The interface starts minimal — Printf(format, args...) only — because
// the existing audio-tools code base emits log via log.Printf. Adding
// leveled methods (Infof, Errorf) is non-breaking as long as Std and
// FakeLogger grow them together. Don't add levels until a second
// consumer needs them.
package logx

import "log"

// seam: Logger is the structured-logging seam (SEAMS.md row "logx.Logger").
// Production wires logx.Std; tests wire mocks.FakeLogger.
//
// Logger is the structured-logging seam every domain depends on.
type Logger interface {
	Printf(format string, args ...any)
}

// Std is the production Logger; wraps *log.Logger.
type Std struct {
	L *log.Logger
}

// Printf delegates to the underlying *log.Logger. If L is nil, falls
// back to log.Default so a forgotten wire-up doesn't panic at runtime.
func (s Std) Printf(format string, args ...any) {
	if s.L == nil {
		log.Default().Printf(format, args...)
		return
	}
	s.L.Printf(format, args...)
}

// Compile-time guarantee that Std satisfies Logger.
var _ Logger = Std{}
