// Package resourceport resolves host ports from resource.json, the only
// production port authority. It is intentionally independent of the control
// plane so scenario APIs can use it without shelling out to a legacy registry.
package resourceport

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

type manifest struct {
	Ports []struct {
		Name      string `json:"name"`
		Host      int    `json:"host"`
		Container int    `json:"container"`
	} `json:"ports"`
}

func Resolve(root, resource, portName string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "resources", resource, "resource.json"))
	if err != nil {
		return "", fmt.Errorf("read %s manifest: %w", resource, err)
	}
	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return "", fmt.Errorf("parse %s manifest: %w", resource, err)
	}
	for _, port := range m.Ports {
		if port.Name == portName || (portName == "" && len(m.Ports) == 1) {
			value := port.Host
			if value == 0 {
				value = port.Container
			}
			if value > 0 {
				return strconv.Itoa(value), nil
			}
		}
	}
	return "", fmt.Errorf("resource %s has no port %q", resource, portName)
}
