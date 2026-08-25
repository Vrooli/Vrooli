// Package backend models session backends (in-memory "standard" PTY vs
// tmux-backed "persistent") and the registry that tracks them.
package backend

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

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
//
// Optional plug points (KeyMap, PromptDetector, IdleHeuristic) carry
// code-only extension behavior — they're tagged `json:"-"` so the wire
// shape stays unchanged and is JSON-stable for the UI's backend picker.
// All plug points are nil-safe; callers must check for nil before use.
type Descriptor struct {
	ID              ID     `json:"id"`
	DisplayName     string `json:"display_name"`
	Description     string `json:"description"`
	SurvivesRestart bool   `json:"survives_restart"`
	Available       bool   `json:"available"`
	Reason          string `json:"reason,omitempty"`

	KeyMap         KeyMap         `json:"-"`
	PromptDetector PromptDetector `json:"-"`
	IdleHeuristic  IdleHeuristic  `json:"-"`
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

var (
	tmuxProbeMu    sync.Mutex
	tmuxProbeCache = make(map[string]struct {
		available bool
		reason    string
	})
	tmuxProbeCommand = func(path string, args ...string) error {
		return exec.Command(path, args...).Run()
	}
)

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
	if strings.TrimSpace(version) == "" {
		return false, fmt.Sprintf("unexpected tmux version output: %q", version)
	}

	tmuxProbeMu.Lock()
	if cached, ok := tmuxProbeCache[version]; ok {
		tmuxProbeMu.Unlock()
		return cached.available, cached.reason
	}
	tmuxProbeMu.Unlock()

	name := fmt.Sprintf("vrooli-web-console-probe-%d", time.Now().UnixNano())
	defer func() { _ = tmuxProbeCommand(path, "kill-session", "-t", name) }()
	if reason := runTmuxProbeCommand(path, "new-session", "-d", "-s", name, "-e", "K=V"); reason != "" {
		return cacheTmuxProbe(version, false, reason)
	}
	commands := [][]string{
		{"resize-window", "-t", name, "-x", "80", "-y", "24"},
		{"set-buffer", "-b", "vrooli-probe-buffer", "probe"},
		{"paste-buffer", "-p", "-b", "vrooli-probe-buffer", "-t", name},
		{"send-keys", "-t", name, "-X", "cancel"},
		{"display-message", "-t", name, "-p", "#{pane_in_mode}"},
		{"display-message", "-t", name, "-p", "#{pane_dead_status}"},
	}
	for _, command := range commands {
		if reason := runTmuxProbeCommand(path, command...); reason != "" {
			return cacheTmuxProbe(version, false, reason)
		}
	}
	return cacheTmuxProbe(version, true, "")
}

func runTmuxProbeCommand(path string, args ...string) string {
	if err := tmuxProbeCommand(path, args...); err != nil {
		return fmt.Sprintf("tmux probe missing command %q: %v", strings.Join(args, " "), err)
	}
	return ""
}

func cacheTmuxProbe(version string, available bool, reason string) (bool, string) {
	tmuxProbeMu.Lock()
	tmuxProbeCache[version] = struct {
		available bool
		reason    string
	}{available: available, reason: reason}
	tmuxProbeMu.Unlock()
	return available, reason
}
