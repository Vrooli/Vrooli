// Package backend models session backends (in-memory "standard" PTY vs
// tmux-backed "persistent") and the registry that tracks them.
package backend

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"web-console/internal/pty"
)

// ID is a typed string for backend identifiers.
// CROSS-LANGUAGE COUPLING: Must match BackendID in ui/src/consts/backend-options.ts.
type ID string

const (
	Standard   ID = "standard"
	Persistent ID = "persistent"
)

// Descriptor describes a session backend's capabilities and availability.
type Descriptor struct {
	ID              ID     `json:"id"`
	DisplayName     string `json:"display_name"`
	Description     string `json:"description"`
	SurvivesRestart bool   `json:"survives_restart"`
	Available       bool   `json:"available"`
	Reason          string `json:"reason,omitempty"`
}

// Registry tracks available session backends and their PTY factories.
type Registry struct {
	mu        sync.RWMutex
	backends  map[ID]Descriptor
	factories map[ID]pty.Factory
	order     []ID // insertion order for deterministic listing
}

// New creates an empty backend registry.
func New() *Registry {
	return &Registry{
		backends:  make(map[ID]Descriptor),
		factories: make(map[ID]pty.Factory),
	}
}

// Register adds a backend descriptor and its factory to the registry.
func (r *Registry) Register(desc Descriptor, factory pty.Factory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.backends[desc.ID]; !exists {
		r.order = append(r.order, desc.ID)
	}
	r.backends[desc.ID] = desc
	r.factories[desc.ID] = factory
}

// Get returns a backend descriptor by ID.
func (r *Registry) Get(id ID) (Descriptor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	desc, ok := r.backends[id]
	return desc, ok
}

// Factory returns the PTY factory registered for the given backend ID.
func (r *Registry) Factory(id ID) (pty.Factory, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, ok := r.factories[id]
	return f, ok
}

// Available returns all backend descriptors in registration order.
func (r *Registry) Available() []Descriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Descriptor, 0, len(r.order))
	for _, id := range r.order {
		result = append(result, r.backends[id])
	}
	return result
}

// IsAvailable reports whether a backend is registered and available.
func (r *Registry) IsAvailable(id ID) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	desc, ok := r.backends[id]
	return ok && desc.Available
}

// ResolveAuto returns Persistent if tmux is available, otherwise Standard.
func (r *Registry) ResolveAuto() ID {
	if r.IsAvailable(Persistent) {
		return Persistent
	}
	return Standard
}

// CheckTmuxAvailable is a function variable for testing.
var CheckTmuxAvailable = defaultCheckTmuxAvailable

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
	parts := strings.Fields(version)
	if len(parts) < 2 {
		return false, fmt.Sprintf("unexpected tmux version output: %q", version)
	}
	verStr := parts[1]
	verStr = strings.TrimPrefix(verStr, "next-")
	verParts := strings.SplitN(verStr, ".", 2)
	if len(verParts) < 2 {
		return true, ""
	}
	major, err := strconv.Atoi(verParts[0])
	if err != nil {
		return false, fmt.Sprintf("cannot parse tmux major version from %q", verStr)
	}
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
