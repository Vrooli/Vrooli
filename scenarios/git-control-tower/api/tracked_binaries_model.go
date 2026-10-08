package main

// TrackedBinariesResponse lists compiled executables that git is tracking.
type TrackedBinariesResponse struct {
	Binaries []TrackedBinary `json:"binaries"`
	// TotalBytes is the combined working-tree size of every tracked binary.
	TotalBytes int64 `json:"total_bytes"`
	// HistoryWarning states plainly that untracking does not reclaim history,
	// so the UI never implies the repository shrinks. Always populated when
	// Binaries is non-empty.
	HistoryWarning string `json:"history_warning,omitempty"`
}

// TrackedBinary is one committed build artifact.
type TrackedBinary struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
	// Format is the detected executable format: "elf", "mach-o", or "pe".
	Format string `json:"format"`
	// OwnerDir is the scenario/resource directory that should ignore this path
	// (e.g. "scenarios/tidiness-manager"), empty when the binary sits at the
	// repo root and the root .gitignore is the only owner.
	OwnerDir string `json:"owner_dir"`
	// IgnorePattern is the line to append to OwnerDir's .gitignore, relative to
	// that directory (e.g. "/cli/cli").
	IgnorePattern string `json:"ignore_pattern"`
	// AlreadyIgnored is true when the pattern is already present, meaning the
	// remediation only needs the index removal.
	AlreadyIgnored bool `json:"already_ignored"`
}

// UntrackBinaryRequest asks to remove one binary from the index and ignore it.
type UntrackBinaryRequest struct {
	Path          string `json:"path"`
	OwnerDir      string `json:"owner_dir"`
	IgnorePattern string `json:"ignore_pattern"`
}

// UntrackBinaryResponse reports the result of untracking one binary.
type UntrackBinaryResponse struct {
	Success bool `json:"success"`
	// RemovedFromIndex is true when git rm --cached succeeded.
	RemovedFromIndex bool `json:"removed_from_index"`
	// IgnoreAddedTo is the .gitignore that gained the pattern, empty when the
	// pattern was already present.
	IgnoreAddedTo string `json:"ignore_added_to,omitempty"`
	Error         string `json:"error,omitempty"`
}
