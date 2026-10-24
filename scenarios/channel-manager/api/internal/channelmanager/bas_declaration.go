package channelmanager

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

const basConsumerDeclarationSchema = "browser-automation-studio.consumer-declaration/v1"

// BASProfileDeclaration is Channel Manager's non-secret reference to a
// profile class it is allowed to bind. The runtime BAS profile reference is
// supplied separately by an operator and remains opaque to this scenario.
type BASProfileDeclaration struct {
	Key              string            `json:"key"`
	WorkflowRef      string            `json:"workflowRef"`
	AllowedVariables []string          `json:"allowedVariables"`
	Preferences      map[string]string `json:"preferences"`
}

// LoadBASProfileDeclarations reads the scenario-owned consumer declaration.
// It deliberately accepts no browser state, credentials, proxy values, or
// runtime profile identifiers: those belong in BAS protected storage.
func LoadBASProfileDeclarations(path string) (map[string]BASProfileDeclaration, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read BAS consumer declaration: %w", err)
	}
	lower := strings.ToLower(string(raw))
	for _, forbidden := range []string{"cookie", "password", "credential", "token", "runtimestate", "runtimeprofile", "profileid", "proxy"} {
		if strings.Contains(lower, forbidden) {
			return nil, fmt.Errorf("BAS consumer declaration contains forbidden %q", forbidden)
		}
	}
	var declaration struct {
		SchemaVersion string                  `json:"schemaVersion"`
		Profiles      []BASProfileDeclaration `json:"profiles"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&declaration); err != nil {
		return nil, fmt.Errorf("decode BAS consumer declaration: %w", err)
	}
	if declaration.SchemaVersion != basConsumerDeclarationSchema || len(declaration.Profiles) == 0 {
		return nil, fmt.Errorf("BAS consumer declaration must use %s and declare at least one profile", basConsumerDeclarationSchema)
	}
	profiles := make(map[string]BASProfileDeclaration, len(declaration.Profiles))
	for _, profile := range declaration.Profiles {
		if strings.TrimSpace(profile.Key) == "" || strings.TrimSpace(profile.WorkflowRef) == "" {
			return nil, errors.New("BAS consumer declaration profile requires key and workflowRef")
		}
		if _, exists := profiles[profile.Key]; exists {
			return nil, fmt.Errorf("BAS consumer declaration duplicates profile key %q", profile.Key)
		}
		profiles[profile.Key] = profile
	}
	return profiles, nil
}
