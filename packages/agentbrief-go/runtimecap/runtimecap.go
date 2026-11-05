package runtimecap

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Capability struct {
	Runtime          string
	Event            string
	Supported        bool
	CanInjectContext bool
	InjectionChannel string
	VerifiedBy       string
	VerifiedAt       string
	Reason           string
	Declared         bool
}

type resourceManifest struct {
	Name       string `json:"name"`
	AgentHooks *struct {
		Events []struct {
			Name       string `json:"name"`
			Supported  bool   `json:"supported"`
			CanInject  bool   `json:"can_inject_context"`
			Channel    string `json:"injection_channel"`
			VerifiedBy string `json:"verified_by"`
			VerifiedAt string `json:"verified_at"`
			Reason     string `json:"reason"`
		} `json:"events"`
	} `json:"agent_hooks"`
}

// Lookup reads one runtime resource declaration. Missing declarations are
// deliberately returned as unverified capabilities, never as permission.
func Lookup(repoRoot, runtime, event string) (Capability, error) {
	capability := Capability{Runtime: runtime, Event: event, Reason: "agent_hooks declaration is absent"}
	path := filepath.Join(repoRoot, "resources", runtime, "resource.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return capability, nil
		}
		return Capability{}, fmt.Errorf("read %s: %w", path, err)
	}
	var manifest resourceManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Capability{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if manifest.AgentHooks == nil {
		return capability, nil
	}
	for _, declared := range manifest.AgentHooks.Events {
		if declared.Name != event {
			continue
		}
		return Capability{Runtime: runtime, Event: event, Supported: declared.Supported, CanInjectContext: declared.CanInject, InjectionChannel: declared.Channel, VerifiedBy: declared.VerifiedBy, VerifiedAt: declared.VerifiedAt, Reason: declared.Reason, Declared: true}, nil
	}
	capability.Reason = "event is absent from agent_hooks declaration"
	return capability, nil
}
