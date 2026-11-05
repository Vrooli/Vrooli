package kind

import (
	"sort"
	"sync"
)

var (
	registryMu sync.RWMutex
	registry   = map[string]Kind{}
)

// Register installs a Kind in the global registry. It panics on duplicate
// names — registration is an init-time programming error, not a runtime
// condition callers can recover from.
func Register(k Kind) {
	registryMu.Lock()
	defer registryMu.Unlock()
	name := k.Name()
	if name == "" {
		panic("flows/kind: Kind.Name() must be non-empty")
	}
	if _, exists := registry[name]; exists {
		panic("flows/kind: duplicate registration for " + name)
	}
	registry[name] = k
}

// Get returns the Kind registered under name, or false if none is.
func Get(name string) (Kind, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	k, ok := registry[name]
	return k, ok
}

// All returns every registered Kind in deterministic (by-name) order.
func All() []Kind {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]Kind, 0, len(registry))
	for _, k := range registry {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}

// Names returns the names of every registered Kind in sorted order.
func Names() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]string, 0, len(registry))
	for name := range registry {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
