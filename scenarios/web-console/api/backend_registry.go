package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

// BackendID is a typed string for backend identifiers.
// CROSS-LANGUAGE COUPLING: Must match BackendID in ui/src/consts/backend-options.ts.
type BackendID string

const (
	BackendStandard   BackendID = "standard"
	BackendPersistent BackendID = "persistent"
)

// BackendDescriptor describes a session backend's capabilities and availability.
type BackendDescriptor struct {
	ID              BackendID `json:"id"`
	DisplayName     string    `json:"display_name"`
	Description     string    `json:"description"`
	SurvivesRestart bool      `json:"survives_restart"`
	Available       bool      `json:"available"`
	Reason          string    `json:"reason,omitempty"`
}

// BackendRegistry tracks available session backends and their PTY factories.
type BackendRegistry struct {
	mu        sync.RWMutex
	backends  map[BackendID]BackendDescriptor
	factories map[BackendID]PTYFactory
	order     []BackendID // insertion order for deterministic listing
}

// NewBackendRegistry creates an empty backend registry.
func NewBackendRegistry() *BackendRegistry {
	return &BackendRegistry{
		backends:  make(map[BackendID]BackendDescriptor),
		factories: make(map[BackendID]PTYFactory),
	}
}

// Register adds a backend descriptor and its PTY factory to the registry.
func (r *BackendRegistry) Register(desc BackendDescriptor, factory PTYFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.backends[desc.ID]; !exists {
		r.order = append(r.order, desc.ID)
	}
	r.backends[desc.ID] = desc
	r.factories[desc.ID] = factory
}

// Get returns a backend descriptor by ID.
func (r *BackendRegistry) Get(id BackendID) (BackendDescriptor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	desc, ok := r.backends[id]
	return desc, ok
}

// Factory returns the PTY factory for a backend.
func (r *BackendRegistry) Factory(id BackendID) (PTYFactory, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, ok := r.factories[id]
	return f, ok
}

// Available returns all backend descriptors in registration order.
func (r *BackendRegistry) Available() []BackendDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]BackendDescriptor, 0, len(r.order))
	for _, id := range r.order {
		result = append(result, r.backends[id])
	}
	return result
}

// IsAvailable reports whether a backend is registered and available.
func (r *BackendRegistry) IsAvailable(id BackendID) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	desc, ok := r.backends[id]
	return ok && desc.Available
}

// checkTmuxAvailable is a function variable for testing.
var checkTmuxAvailable = defaultCheckTmuxAvailable

// defaultCheckTmuxAvailable probes for tmux and returns availability + reason.
func defaultCheckTmuxAvailable() (bool, string) {
	path, err := exec.LookPath("tmux")
	if err != nil {
		return false, "tmux is not installed"
	}
	out, err := exec.Command(path, "-V").Output()
	if err != nil {
		return false, fmt.Sprintf("tmux version check failed: %v", err)
	}
	version := strings.TrimSpace(string(out))
	// Parse "tmux X.Y" format
	parts := strings.Fields(version)
	if len(parts) < 2 {
		return false, fmt.Sprintf("unexpected tmux version output: %q", version)
	}
	verStr := parts[1]
	// Handle versions like "3.4" or "next-3.5"
	verStr = strings.TrimPrefix(verStr, "next-")
	// Split on '.' and check major.minor >= 2.6
	verParts := strings.SplitN(verStr, ".", 2)
	if len(verParts) < 2 {
		// Single number version (e.g. "4") — assume OK
		return true, ""
	}
	major, err := strconv.Atoi(verParts[0])
	if err != nil {
		return false, fmt.Sprintf("cannot parse tmux major version from %q", verStr)
	}
	// Extract just the numeric part of minor (e.g., "4a" -> 4)
	minorStr := verParts[1]
	for i, c := range minorStr {
		if c < '0' || c > '9' {
			minorStr = minorStr[:i]
			break
		}
	}
	minor, err := strconv.Atoi(minorStr)
	if err != nil {
		return false, fmt.Sprintf("cannot parse tmux minor version from %q", verStr)
	}
	if major < 2 || (major == 2 && minor < 6) {
		return false, fmt.Sprintf("tmux %s is too old (minimum 2.6)", verStr)
	}
	return true, ""
}

// InitDefaultRegistry creates a backend registry with the standard and
// persistent (tmux) backends registered.
func InitDefaultRegistry() *BackendRegistry {
	reg := NewBackendRegistry()

	// Standard (raw PTY) backend — always available
	reg.Register(BackendDescriptor{
		ID:              BackendStandard,
		DisplayName:     "Standard",
		Description:     "In-memory session. Fast and lightweight, but lost on restart.",
		SurvivesRestart: false,
		Available:       true,
	}, defaultPTYFactory)

	// Persistent (tmux) backend — available only if tmux is installed
	tmuxAvail, tmuxReason := checkTmuxAvailable()
	reg.Register(BackendDescriptor{
		ID:              BackendPersistent,
		DisplayName:     "Persistent",
		Description:     "Backed by tmux. Survives web console restarts.",
		SurvivesRestart: true,
		Available:       tmuxAvail,
		Reason:          tmuxReason,
	}, tmuxPTYFactory)

	return reg
}
