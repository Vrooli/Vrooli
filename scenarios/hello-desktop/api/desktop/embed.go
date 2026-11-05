// Package desktop exposes hello-desktop's HTTP surface and lifecycle for
// in-process desktop shells. The command entry point remains a thin listener
// wrapper around this constructor.
package desktop

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// Lifecycle owns API resources that must be started and stopped with the
// desktop shell. Implementations must stop background work before returning.
type Lifecycle interface {
	Start(context.Context) error
	Stop(context.Context) error
}

type lifecycle struct {
	mu      sync.Mutex
	started bool
	stopped bool
}

func (l *lifecycle) Start(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.started || l.stopped {
		return fmt.Errorf("hello-desktop lifecycle cannot start more than once")
	}
	l.started = true
	return nil
}

func (l *lifecycle) Stop(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.started {
		return fmt.Errorf("hello-desktop lifecycle must start before stop")
	}
	l.stopped = true
	return nil
}

type greetingState struct {
	mu      sync.RWMutex
	name    string
	message string
}

// New returns the API handler and its lifecycle hooks for Wails or another
// in-process host. It does not create a TCP listener or start background work.
func New() (http.Handler, Lifecycle, error) {
	state := &greetingState{}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})
	mux.HandleFunc("/api/greet", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "" {
			name = "Desktop User"
		}
		message := "Hello, " + name + "!"
		state.mu.Lock()
		state.name, state.message = name, message
		state.mu.Unlock()
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"message": message})
	})
	// This test bridge is used only by the canonical fixture journey. A
	// run-specific name prevents stale API state from satisfying the assertion.
	mux.HandleFunc("/api/test/last-greeting", func(w http.ResponseWriter, r *http.Request) {
		expected := r.URL.Query().Get("name")
		state.mu.RLock()
		name, message := state.name, state.message
		state.mu.RUnlock()
		if expected != "" && expected != name {
			http.Error(w, "greeting not observed", http.StatusNotFound)
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"name": name, "message": message})
	})
	return mux, &lifecycle{}, nil
}
