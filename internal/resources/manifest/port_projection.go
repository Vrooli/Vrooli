package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode"
)

// PortProjection is reservation intent, not a validated runtime manifest. A
// resource may still be live while unrelated source runtime fields are invalid.
type PortProjection struct {
	Name  string         `json:"name"`
	Ports []ResourcePort `json:"ports"`
}

func LoadPortProjection(path, expectedName string) (PortProjection, error) {
	var projection PortProjection
	data, err := os.ReadFile(path)
	if err != nil {
		return projection, err
	}
	if err = json.Unmarshal(data, &projection); err != nil {
		return projection, fmt.Errorf("parse resource port projection: %w", err)
	}
	if projection.Name == "" || projection.Name != expectedName {
		return projection, fmt.Errorf("resource port projection name %q does not match directory %q", projection.Name, expectedName)
	}
	seen := map[string]bool{}
	for i, port := range projection.Ports {
		if port.Name == "" || strings.TrimSpace(port.Name) != port.Name || strings.IndexFunc(port.Name, unicode.IsControl) >= 0 {
			return projection, fmt.Errorf("resource %s port%d requires a nonempty control-free name without surrounding whitespace", expectedName, i)
		}
		if seen[port.Name] {
			return projection, fmt.Errorf("resource %s port name %q is duplicated", expectedName, port.Name)
		}
		seen[port.Name] = true
		if port.Host < 0 || port.Host > 65535 || port.Container < 0 || port.Container > 65535 {
			return projection, fmt.Errorf("resource %s port %q host/container must be0–65535", expectedName, port.Name)
		}
	}
	return projection, nil
}
