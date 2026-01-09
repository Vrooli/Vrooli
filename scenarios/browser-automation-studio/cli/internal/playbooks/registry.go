package playbooks

import (
	"encoding/json"
	"fmt"
	"os"
)

type Registry struct {
	Scenario  string          `json:"scenario"`
	Playbooks []RegistryEntry `json:"playbooks"`
}

type RegistryEntry struct {
	Order        string   `json:"order"`
	Reset        string   `json:"reset"`
	Requirements []string `json:"requirements"`
	File         string   `json:"file"`
	Description  string   `json:"description"`
}

func LoadRegistry(path string) (*Registry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read registry: %w", err)
	}
	var registry Registry
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("parse registry: %w", err)
	}
	return &registry, nil
}
