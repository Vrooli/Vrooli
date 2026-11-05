package deliveryramp

import (
	"fmt"
	"sort"
	"strings"
)

// RampOperation names an operation that a delivery ramp may expose. A ramp
// can refuse an operation explicitly while still advertising the capabilities
// it does support.
type RampOperation string

const (
	OperationProbe      RampOperation = "probe"
	OperationBuild      RampOperation = "build"
	OperationExecute    RampOperation = "execute"
	OperationDistribute RampOperation = "distribute"
	OperationRecover    RampOperation = "recover"
)

type OperationCapability struct {
	Operation  RampOperation `json:"operation"`
	Supported  bool          `json:"supported"`
	Reason     string        `json:"reason,omitempty"`
	NextAction string        `json:"next_action,omitempty"`
}

func (c OperationCapability) Validate() error {
	if strings.TrimSpace(string(c.Operation)) == "" {
		return fmt.Errorf("operation is required")
	}
	if !c.Supported && (strings.TrimSpace(c.Reason) == "" || strings.TrimSpace(c.NextAction) == "") {
		return fmt.Errorf("unsupported operation %q requires reason and next_action", c.Operation)
	}
	return nil
}

// RampDefinition is the bounded capability view consumed by orchestration and
// operator surfaces. It contains no owner implementation or secret material.
type RampDefinition struct {
	ID           string                 `json:"id"`
	Description  string                 `json:"description"`
	Targets      []string               `json:"targets"`
	Formats      []DeliveryFormat       `json:"formats"`
	Capabilities []CapabilityDefinition `json:"capabilities"`
	Operations   []OperationCapability  `json:"operations"`
}

func (d RampDefinition) Validate() error {
	if strings.TrimSpace(d.ID) == "" || strings.TrimSpace(d.Description) == "" {
		return fmt.Errorf("ramp id and description are required")
	}
	if len(d.Targets) == 0 {
		return fmt.Errorf("ramp %q must declare at least one target", d.ID)
	}
	seen := make(map[string]struct{}, len(d.Targets))
	for _, target := range d.Targets {
		target = strings.TrimSpace(target)
		if target == "" {
			return fmt.Errorf("ramp %q contains an empty target", d.ID)
		}
		if _, ok := seen[target]; ok {
			return fmt.Errorf("ramp %q declares duplicate target %q", d.ID, target)
		}
		seen[target] = struct{}{}
	}
	seen = make(map[string]struct{}, len(d.Formats))
	for _, format := range d.Formats {
		value := strings.TrimSpace(string(format))
		if value == "" {
			return fmt.Errorf("ramp %q contains an empty format", d.ID)
		}
		if _, ok := seen[value]; ok {
			return fmt.Errorf("ramp %q declares duplicate format %q", d.ID, value)
		}
		seen[value] = struct{}{}
	}
	if _, err := NewCapabilityRegistry(d.Capabilities); err != nil {
		return fmt.Errorf("ramp %q capabilities: %w", d.ID, err)
	}
	operationSeen := make(map[RampOperation]struct{}, len(d.Operations))
	for _, operation := range d.Operations {
		if err := operation.Validate(); err != nil {
			return fmt.Errorf("ramp %q: %w", d.ID, err)
		}
		if _, ok := operationSeen[operation.Operation]; ok {
			return fmt.Errorf("ramp %q declares duplicate operation %q", d.ID, operation.Operation)
		}
		operationSeen[operation.Operation] = struct{}{}
	}
	return nil
}

// UnsupportedOperationError preserves the owner-provided repair path when a
// caller asks a registered ramp for an operation it cannot perform.
type UnsupportedOperationError struct {
	Ramp       string
	Operation  RampOperation
	Reason     string
	NextAction string
}

func (e *UnsupportedOperationError) Error() string {
	return fmt.Sprintf("ramp %q does not support %q: %s; next action: %s", e.Ramp, e.Operation, e.Reason, e.NextAction)
}

type RampRegistry struct {
	definitions map[string]RampDefinition
}

func NewRampRegistry(definitions []RampDefinition) (*RampRegistry, error) {
	registry := &RampRegistry{definitions: make(map[string]RampDefinition, len(definitions))}
	for _, definition := range definitions {
		if err := definition.Validate(); err != nil {
			return nil, err
		}
		id := strings.TrimSpace(definition.ID)
		if _, exists := registry.definitions[id]; exists {
			return nil, fmt.Errorf("ramp %q is duplicated", id)
		}
		definition.ID = id
		registry.definitions[id] = cloneRampDefinition(definition)
	}
	return registry, nil
}

func (r *RampRegistry) Resolve(id string) (RampDefinition, bool) {
	if r == nil {
		return RampDefinition{}, false
	}
	definition, ok := r.definitions[strings.TrimSpace(id)]
	if !ok {
		return RampDefinition{}, false
	}
	return cloneRampDefinition(definition), true
}

// List returns a stable capability view for UI and CLI consumers.
func (r *RampRegistry) List() []RampDefinition {
	if r == nil {
		return nil
	}
	result := make([]RampDefinition, 0, len(r.definitions))
	for _, definition := range r.definitions {
		result = append(result, cloneRampDefinition(definition))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func (r *RampRegistry) RequireOperation(rampID string, operation RampOperation) error {
	definition, ok := r.Resolve(rampID)
	if !ok {
		return fmt.Errorf("ramp %q is not registered", strings.TrimSpace(rampID))
	}
	for _, capability := range definition.Operations {
		if capability.Operation != operation {
			continue
		}
		if capability.Supported {
			return nil
		}
		return &UnsupportedOperationError{Ramp: definition.ID, Operation: operation, Reason: capability.Reason, NextAction: capability.NextAction}
	}
	return &UnsupportedOperationError{Ramp: definition.ID, Operation: operation, Reason: "operation is not declared by the ramp", NextAction: "register the operation or select a ramp that supports it"}
}

func cloneRampDefinition(definition RampDefinition) RampDefinition {
	definition.Targets = append([]string(nil), definition.Targets...)
	definition.Formats = append([]DeliveryFormat(nil), definition.Formats...)
	definition.Capabilities = append([]CapabilityDefinition(nil), definition.Capabilities...)
	for index := range definition.Capabilities {
		definition.Capabilities[index].Profiles = append([]string(nil), definition.Capabilities[index].Profiles...)
	}
	definition.Operations = append([]OperationCapability(nil), definition.Operations...)
	return definition
}
