package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Persona represents an agent customization profile
type Persona struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Prompt      string   `json:"prompt"`
	UseCases    []string `json:"use_cases"`
	Keywords    []string `json:"keywords"`
}

// PersonaCatalog represents the personas catalog structure
type PersonaCatalog struct {
	Personas []Persona              `json:"personas"`
	Metadata map[string]interface{} `json:"_metadata"`
}

// PersonaService handles persona operations
type PersonaService struct {
	personasDir string
}

// NewPersonaService creates a new persona service
func NewPersonaService(templatesDir string) *PersonaService {
	return &PersonaService{
		personasDir: filepath.Join(templatesDir, "..", "personas"),
	}
}

// GetPersonas retrieves all available agent personas from the catalog
func (ps *PersonaService) GetPersonas() ([]Persona, error) {
	catalogPath := filepath.Join(ps.personasDir, "catalog.json")

	data, err := os.ReadFile(catalogPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read persona catalog: %w", err)
	}

	var catalog PersonaCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("failed to parse persona catalog: %w", err)
	}

	return catalog.Personas, nil
}

// GetPersona retrieves a specific persona by ID
func (ps *PersonaService) GetPersona(id string) (*Persona, error) {
	personas, err := ps.GetPersonas()
	if err != nil {
		return nil, err
	}

	for _, p := range personas {
		if p.ID == id {
			return &p, nil
		}
	}

	return nil, fmt.Errorf("persona not found: %s", id)
}
