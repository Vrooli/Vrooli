package deliveryramp

import (
	"fmt"
	"strings"
)

type CapabilityDefinition struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Profiles    []string `json:"profiles,omitempty"`
}

type CapabilityRegistry struct {
	definitions map[string]CapabilityDefinition
}

func NewCapabilityRegistry(definitions []CapabilityDefinition) (*CapabilityRegistry, error) {
	registry := &CapabilityRegistry{definitions: make(map[string]CapabilityDefinition, len(definitions))}
	for _, definition := range definitions {
		id := strings.TrimSpace(definition.ID)
		if id == "" || strings.TrimSpace(definition.Description) == "" {
			return nil, fmt.Errorf("capability id and description are required")
		}
		if _, exists := registry.definitions[id]; exists {
			return nil, fmt.Errorf("capability %q is duplicated", id)
		}
		definition.ID = id
		definition.Profiles = append([]string(nil), definition.Profiles...)
		registry.definitions[id] = definition
	}
	return registry, nil
}

func (r *CapabilityRegistry) Resolve(id string) (CapabilityDefinition, bool) {
	if r == nil {
		return CapabilityDefinition{}, false
	}
	definition, ok := r.definitions[strings.TrimSpace(id)]
	definition.Profiles = append([]string(nil), definition.Profiles...)
	return definition, ok
}
