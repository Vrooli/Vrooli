package installgateway

import (
	"fmt"
	"strings"
)

// Adapter describes an ecosystem edge without making callers know manager
// switches. Mutation support is explicit: C/C++ discovery can be supported
// without pretending that a safe universal mutation command exists.
type Adapter interface {
	ID() string
	MutationSupported() bool
	FrozenReproduction() ([]string, error)
}

type adapter struct {
	id        string
	supported bool
}

func (a adapter) ID() string                            { return a.id }
func (a adapter) MutationSupported() bool               { return a.supported }
func (a adapter) FrozenReproduction() ([]string, error) { return FrozenReproductionArgs(a.id) }

// DefaultAdapters is the typed capability registry shared by status and
// install callers. Adding an ecosystem extends this table, not every caller.
func DefaultAdapters() map[string]Adapter {
	return map[string]Adapter{
		"npm":    adapter{id: "npm", supported: true},
		"pnpm":   adapter{id: "pnpm", supported: true},
		"yarn":   adapter{id: "yarn", supported: true},
		"bun":    adapter{id: "bun", supported: true},
		"go":     adapter{id: "go", supported: true},
		"python": adapter{id: "pip", supported: true},
		"pip":    adapter{id: "pip", supported: true},
		"rust":   adapter{id: "cargo", supported: true},
		"cargo":  adapter{id: "cargo", supported: true},
		"c":      adapter{id: "c", supported: false},
		"cpp":    adapter{id: "cpp", supported: false},
	}
}

func AdapterFor(ecosystem string) (Adapter, error) {
	id := strings.ToLower(strings.TrimSpace(ecosystem))
	adapter, ok := DefaultAdapters()[id]
	if !ok {
		return nil, fmt.Errorf("unsupported ecosystem %q", ecosystem)
	}
	return adapter, nil
}
