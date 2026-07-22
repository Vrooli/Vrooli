// Package pty defines the PTY abstraction used by web-console session
// backends. It holds the types — interface, launch spec, factory, input
// kinds — but no concrete implementation. Backends (standard creack/pty,
// tmux, future remote PTYs) implement PTY in their own packages.
//
// Keeping these types in their own package breaks the import cycle that
// would otherwise form between the backend registry and the realPTY
// implementation in package main.
package pty

import "context"

// InputKind discriminates keystroke input (one or more bytes originating
// from the xterm keyboard / toolbar) from pasted input (a clipboard
// payload, which may be large and may include control bytes that a
// mode-sensitive multiplexer must not interpret as navigation commands).
//
// For the standard PTY the distinction is cosmetic: both paths end at
// the same `ptmx.Write`. For the tmux-backed PTY the distinction is
// load-bearing: keystrokes route through `tmux send-keys -l --` so tmux
// delivers them literally to the pane regardless of whether the client
// is in copy-mode / command-prompt / menu / prefix-pending, and pastes
// route through `tmux load-buffer` + `paste-buffer -d` which auto-exits
// copy-mode and atomically delivers the payload without byte-by-byte
// mode interpretation.
type InputKind uint8

const (
	// KindKeystroke is the default for ordinary terminal input.
	KindKeystroke InputKind = iota
	// KindPaste is used for clipboard paste payloads originating
	// from the context menu, native paste capture, or touch long-press
	// paste. Tmux routes these through paste-buffer.
	KindPaste
)

// DOC: docs/internal/SEAMS.md#pty-factory-seam-api
// PTY represents a pseudo-terminal process with read/write, resize, and
// lifecycle control. The default implementation wraps creack/pty; tests can
// substitute a pipe-based fake via Factory.
type PTY interface {
	Read(p []byte) (int, error)
	// WriteInput delivers client-origin bytes to the underlying process.
	// The kind parameter selects the delivery mechanism: see InputKind.
	// Returns a typed error on failure (backend-specific); callers map
	// the error to a stdin_ack.reason.
	WriteInput(data []byte, kind InputKind) error
	SetSize(cols, rows uint16) error
	Close() error
	Kill() error
	// ExitCode waits for the process to finish and returns its exit code.
	// Call only after Read returns an error (process exited). Returns -1 if
	// the exit code cannot be determined.
	ExitCode() int
	// HasChildProcess reports whether the shell process has any child
	// processes running (e.g. a script, interactive program, etc.).
	HasChildProcess() bool
	// ProbeReady blocks until the PTY pipeline is confirmed to be accepting
	// writes that will reach the underlying process. For synchronous
	// backends this is a no-op; for async backends (tmux) this waits for
	// the attach-session handshake to complete.
	ProbeReady(ctx context.Context) error
	// CurrentDir returns the PTY process's current working directory when it
	// can be determined. Backends may fall back to their launch directory
	// when live cwd discovery is unavailable.
	CurrentDir(ctx context.Context) (string, error)
}

// LaunchSpec contains the environment and execution parameters for a
// newly created terminal session.
type LaunchSpec struct {
	SessionID string
	Shell     string
	Cols      uint16
	Rows      uint16
	// WorkingDir overrides the configured default for this terminal. Recovery
	// uses the orphan's recorded cwd so reattached work resumes in place.
	WorkingDir string
	Env        map[string]string
}

// Factory creates a PTY-backed process for the given launch spec. Inject
// a custom factory into SessionManager for testing without real processes.
type Factory func(spec LaunchSpec) (PTY, error)
