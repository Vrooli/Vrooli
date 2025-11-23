package engine

import (
	"context"
	"fmt"
	"strings"
)

// StaticFactory resolves pre-constructed engines by name.
type StaticFactory struct {
	engines map[string]AutomationEngine
}

// NewStaticFactory constructs a factory from a set of engines keyed by name.
func NewStaticFactory(engines ...AutomationEngine) *StaticFactory {
	m := make(map[string]AutomationEngine, len(engines))
	for _, eng := range engines {
		if eng == nil {
			continue
		}
		m[strings.ToLower(eng.Name())] = eng
	}
	return &StaticFactory{engines: m}
}

// Resolve returns the engine by name (case-insensitive).
func (f *StaticFactory) Resolve(_ context.Context, name string) (AutomationEngine, error) {
	if f == nil {
		return nil, fmt.Errorf("engine factory not configured")
	}
	if name == "" {
		name = "browserless"
	}
	eng, ok := f.engines[strings.ToLower(name)]
	if !ok {
		return nil, fmt.Errorf("engine %q not registered", name)
	}
	return eng, nil
}
