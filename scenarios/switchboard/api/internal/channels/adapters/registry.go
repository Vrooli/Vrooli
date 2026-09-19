package adapters

import (
	"sync"

	"switchboard/internal/channels"
)

// Factory constructs one production adapter. Adapter packages register their
// factory from an init function in this package's built-in registry file. A
// new adapter can register itself by adding one file to this package.
type Factory func() channels.Adapter

var (
	factoriesMu sync.RWMutex
	factories   []Factory
)

// Register adds an adapter factory to the process-wide registry.
func Register(factory Factory) {
	if factory == nil {
		return
	}
	factoriesMu.Lock()
	defer factoriesMu.Unlock()
	factories = append(factories, factory)
}

// NewAll constructs the registered adapters in registration order. The
// caller owns the returned slice and each adapter instance.
func NewAll() []channels.Adapter {
	factoriesMu.RLock()
	registered := append([]Factory(nil), factories...)
	factoriesMu.RUnlock()

	result := make([]channels.Adapter, 0, len(registered))
	for _, factory := range registered {
		if adapter := factory(); adapter != nil {
			result = append(result, adapter)
		}
	}
	return result
}
