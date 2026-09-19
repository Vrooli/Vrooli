package retention

import (
	"fmt"
	"sort"
	"sync"
)

// Registry maps a budget name to the Pruner a component registered for it.
// Registration is by the manifest's budget key, so the name is declared once and
// used for the lookup, the log line, and the finding.
type Registry struct {
	mu      sync.RWMutex
	pruners map[string]Pruner
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{pruners: make(map[string]Pruner)}
}

// Register binds a Pruner to a budget name. Registering the same name twice is
// an error: the second registration would silently win, and which of two
// selection rules is enforcing a budget must never be a matter of init order.
func (r *Registry) Register(name string, p Pruner) error {
	if name == "" {
		return fmt.Errorf("register retention pruner: name is empty")
	}
	if p == nil {
		return fmt.Errorf("register retention pruner %q: pruner is nil", name)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.pruners[name]; exists {
		return fmt.Errorf("register retention pruner %q: already registered", name)
	}
	r.pruners[name] = p
	return nil
}

// Lookup returns the Pruner registered for name.
func (r *Registry) Lookup(name string) (Pruner, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.pruners[name]
	return p, ok
}

// Names returns the registered budget names, sorted.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.pruners))
	for name := range r.pruners {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
