package agents

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// CapabilityGrant is the reference to scopes and program-runtime bindings
// declared by the agent profile. Switchboard does not define a second policy
// vocabulary or copy the profile into its own store.
type CapabilityGrant struct {
	Scopes          []string `json:"scopes"`
	ProgramBindings []string `json:"program_bindings,omitempty"`
}

type profileDocument struct {
	Grant *CapabilityGrant `json:"grant"`
}

func ParseGrant(raw []byte) (CapabilityGrant, error) {
	var profile profileDocument
	if err := json.Unmarshal(raw, &profile); err != nil {
		return CapabilityGrant{}, fmt.Errorf("parse agent descriptor: %w", err)
	}
	if profile.Grant == nil {
		return CapabilityGrant{}, fmt.Errorf("agent descriptor has no capability grant")
	}
	grant := CapabilityGrant{Scopes: clean(profile.Grant.Scopes), ProgramBindings: clean(profile.Grant.ProgramBindings)}
	if len(grant.Scopes) == 0 && len(grant.ProgramBindings) == 0 {
		return CapabilityGrant{}, fmt.Errorf("capability grant is empty")
	}
	return grant, nil
}

func LoadGrant(path string) (CapabilityGrant, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return CapabilityGrant{}, fmt.Errorf("read agent descriptor: %w", err)
	}
	return ParseGrant(raw)
}

func clean(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
