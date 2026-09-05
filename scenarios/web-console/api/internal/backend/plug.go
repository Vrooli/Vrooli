package backend

// Plug-point interfaces let a BackendDescriptor carry optional, code-only
// extension behavior. None of these change the JSON shape of Descriptor —
// they're tagged `json:"-"` on the struct. All fields are nil-safe; callers
// must use the accessor helpers below (or check for nil) before invoking.
//
// Why interfaces here and not in session/: keeping the contracts in this
// package avoids an import cycle between session/ and internal/backend, and
// makes the descriptor the single registration point for per-backend
// behavior — KeyMap, prompt detection, idle heuristics all hang off the
// same struct that already declares the backend.

// KeyMap converts symbolic key names (e.g. "Enter", "Ctrl+C", "ArrowUp")
// into the byte sequence the backend's foreground process expects. A nil
// KeyMap means the session's default map is used.
type KeyMap interface {
	// Encode returns the byte sequence for the named key, or false if the
	// name is not recognized by this map. Names not recognized here fall
	// back to the session default.
	Encode(name string) ([]byte, bool)
}

// PromptDetector classifies the current screen as "agent is waiting for
// user input" or not. Used by future driver code that wants to know when
// it can safely send the next instruction. A nil PromptDetector means no
// per-backend hint is available; consumers should fall back to idle
// heuristics.
type PromptDetector interface {
	// IsAwaitingInput inspects a screen view and reports whether the
	// foreground process appears to be at an input prompt. The view is
	// caller-owned and must not be retained.
	IsAwaitingInput(view ScreenView) bool
}

// IdleHeuristic decides when the session is "quiet enough" that no more
// output is imminent. A nil IdleHeuristic means the session's default
// quiet-window logic is used.
type IdleHeuristic interface {
	// QuietWindowExceeded reports whether the time since the last frame
	// implies the foreground process is idle for this backend. lastFrame
	// is monotonic-relative; sinceLast is the wall-clock delta.
	QuietWindowExceeded(sinceLastMillis int64) bool
}

// ScreenView is the narrow read surface PromptDetector consumes. Defined
// here (not in session/) so this package has no upward dependency. The
// session package adapts its richer screen type to this interface.
type ScreenView interface {
	Cols() int
	Rows() int
	CursorRow() int
	CursorCol() int
	PlainText() string
}
