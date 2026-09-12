package backend

// Plug-point interfaces let a BackendDescriptor carry optional, code-only
// extension behavior. None of these change the JSON shape of Descriptor —
// they're tagged `json:"-"` on the struct. All fields are nil-safe; callers
// must use the accessor helpers below (or check for nil) before invoking.
//
// Why interfaces here and not in session/: keeping the contracts in this
// package avoids an import cycle between session/ and internal/backend.
// Session backends (standard, persistent, remote) carry KeyMap and
// IdleHeuristic; prompt detection belongs to the agent harness running in
// the terminal and is selected by agent type (api/backends.PromptDetectorFor).

// KeyMap converts symbolic key names (e.g. "Enter", "Ctrl+C", "ArrowUp")
// into the byte sequence the backend's foreground process expects. A nil
// KeyMap means the session's default map is used.
type KeyMap interface {
	// Encode returns the byte sequence for the named key, or false if the
	// name is not recognized by this map. Names not recognized here fall
	// back to the session default.
	Encode(name string) ([]byte, bool)
}

// PromptOption is one choice in a prompt an agent is showing.
type PromptOption struct {
	// Key is how the option is chosen: its number for numbered prompts, or
	// its 1-based position for arrow-selected prompts.
	Key      string
	Label    string
	Selected bool
}

// PendingPrompt is what an agent is asking, when a detector can read it.
type PendingPrompt struct {
	// Kind is "question", "permission", or "unknown" (detected, not parsed).
	Kind    string
	Text    string
	Options []PromptOption
	// FreeTextHint names the option that takes a typed answer, when one exists.
	FreeTextHint string
	// Cancellable is true when the harness lets Escape dismiss the prompt.
	Cancellable bool
}

// PromptAnalysis is a detector's reading of one screen.
type PromptAnalysis struct {
	// AwaitingInput is true when the agent is at its input prompt (idle).
	AwaitingInput bool
	// Prompt is set when the agent is showing a question or permission box.
	Prompt *PendingPrompt
	// Working is true when the screen shows the agent's in-progress marker.
	Working bool
	// Confidence is 0..1 for whichever of the above the detector reported.
	Confidence float32
}

// PromptDetector reads an agent harness's screen: whether it is working, at
// its input prompt, or asking the user something. A nil PromptDetector means
// the harness has no screen reading; consumers fall back to the output clock.
type PromptDetector interface {
	// Analyze inspects a screen view. The view is caller-owned and must not
	// be retained.
	Analyze(view ScreenView) PromptAnalysis
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
