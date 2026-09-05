package templateengine

import (
	"io"

	"github.com/vrooli/vrooli/internal/scenarioexec"
)

// HandlerDeps carries the runtime seams the engine needs to reach the host:
// where to write progress, the repository root, and how to run subprocesses.
// It is generic over the caller's context type so the engine stays decoupled
// from any single transport.
type HandlerDeps[C any] struct {
	Stdout             func(C) io.Writer
	Stderr             func(C) io.Writer
	Root               func(C) string
	RunSubprocess      func(C, scenarioexec.SubprocessSpec) error
	LocateTestGenieCLI func(C) (string, error)
	CommandEnv         func(C) []string
}
