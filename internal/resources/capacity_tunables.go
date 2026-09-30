package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/vrooli/vrooli/internal/operatorstate"
)

type effectiveCapacityTunable struct {
	Value           any    `json:"value"`
	Source          string `json:"source"`
	Environment     string `json:"env"`
	ScalesFootprint bool   `json:"scales_footprint"`
}

func (c *Controller) withEffectiveCapacity(status Status) (Status, error) {
	if status.Resource.ManifestPath == "" {
		return status, nil
	}
	resourceManifest, err := c.LoadManifest(status.Resource.ManifestPath)
	if err != nil {
		return status, err
	}
	if resourceManifest.Acceleration == nil {
		return status, nil
	}
	declared := resourceManifest.Acceleration.Capacity.Tunables
	if len(declared) == 0 {
		return status, nil
	}
	doc, err := operatorstate.New(operatorstate.Config{RepoRoot: c.Root}).Load(context.Background())
	if err != nil {
		return status, fmt.Errorf("load operator state for capacity tunables: %w", err)
	}
	overrides := map[string]any(nil)
	if choice, ok := doc.Resources[status.Resource.Name]; ok && choice.Capacity != nil {
		overrides = choice.Capacity.Tunables
	}
	effective := make(map[string]effectiveCapacityTunable, len(declared))
	for _, tunable := range declared {
		value := tunable.Default
		source := "manifest_default"
		if override, ok := overrides[tunable.Name]; ok {
			if err := tunable.ValidateValue(override); err != nil {
				return status, fmt.Errorf("resources.%s.capacity.tunables.%s: %w", status.Resource.Name, tunable.Name, err)
			}
			value = override
			source = "operator_state"
		}
		effective[tunable.Name] = effectiveCapacityTunable{
			Value: value, Source: source, Environment: tunable.Env,
			ScalesFootprint: tunable.EffectiveScalesFootprint(),
		}
	}
	var raw map[string]any
	if len(status.Raw) > 0 {
		_ = json.Unmarshal(status.Raw, &raw)
	}
	if raw == nil {
		raw = make(map[string]any)
	}
	raw["capacity"] = map[string]any{"tunables": effective}
	status.Raw, err = json.Marshal(raw)
	return status, err
}
