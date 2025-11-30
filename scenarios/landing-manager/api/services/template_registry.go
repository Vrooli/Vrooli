// Package services contains the business logic for the landing-manager API.
package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"landing-manager/util"
)

// Template represents a landing page template with full metadata
type Template struct {
	ID                  string                 `json:"id"`
	Name                string                 `json:"name"`
	Description         string                 `json:"description"`
	Version             string                 `json:"version"`
	Metadata            map[string]interface{} `json:"metadata"`
	Sections            map[string]interface{} `json:"sections"`
	MetricsHooks        []MetricHook           `json:"metrics_hooks"`
	CustomizationSchema map[string]interface{} `json:"customization_schema"`
	FrontendAesthetics  map[string]interface{} `json:"frontend_aesthetics,omitempty"`
	TechStack           map[string]interface{} `json:"tech_stack,omitempty"`
	GeneratedStructure  map[string]interface{} `json:"generated_structure,omitempty"`
}

// TemplateCatalog represents the template registry index
type TemplateCatalog struct {
	Version     string                  `json:"version"`
	Description string                  `json:"description"`
	Categories  map[string]CategoryInfo `json:"categories"`
	Templates   map[string]TemplateRef  `json:"templates"`
}

// CategoryInfo describes a template category
type CategoryInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Templates   []string `json:"templates"`
}

// TemplateRef is a reference to a template in the catalog
type TemplateRef struct {
	Path         string   `json:"path"`
	Category     string   `json:"category"`
	Version      string   `json:"version"`
	SectionsUsed []string `json:"sections_used"`
}

// MetricHook defines an analytics event type
type MetricHook struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	EventType      string   `json:"event_type"`
	RequiredFields []string `json:"required_fields"`
}

// TemplateRegistry handles template operations (listing, fetching, caching)
type TemplateRegistry struct {
	templatesDir  string
	cache         map[string]Template
	cacheMux      sync.RWMutex
	cacheExpiry   time.Time
	cacheDuration time.Duration
}

// NewTemplateRegistry creates a template registry instance
func NewTemplateRegistry() *TemplateRegistry {
	// Check for test environment override
	if testDir := os.Getenv("TEMPLATES_DIR"); testDir != "" {
		return &TemplateRegistry{
			templatesDir:  testDir,
			cache:         make(map[string]Template),
			cacheDuration: 5 * time.Minute,
		}
	}

	// Determine templates directory relative to binary location
	execPath, err := os.Executable()
	if err != nil {
		execPath = "."
	}
	execDir := filepath.Dir(execPath)
	templatesDir := filepath.Join(execDir, "templates")

	// Fallback to current directory + templates if binary path templates don't exist
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		templatesDir = "templates"
	}

	return &TemplateRegistry{
		templatesDir:  templatesDir,
		cache:         make(map[string]Template),
		cacheDuration: 5 * time.Minute,
	}
}

// NewTemplateRegistryWithDir creates a template registry with a specific directory (for testing)
func NewTemplateRegistryWithDir(dir string) *TemplateRegistry {
	return &TemplateRegistry{
		templatesDir:  dir,
		cache:         make(map[string]Template),
		cacheDuration: 5 * time.Minute,
	}
}

// GetTemplatesDir returns the templates directory path
func (tr *TemplateRegistry) GetTemplatesDir() string {
	return tr.templatesDir
}

// ListTemplates returns all available templates (with caching)
func (tr *TemplateRegistry) ListTemplates() ([]Template, error) {
	// Check if cache is valid
	if tr.cache != nil {
		tr.cacheMux.RLock()
		cacheValid := time.Now().Before(tr.cacheExpiry) && len(tr.cache) > 0
		tr.cacheMux.RUnlock()

		if cacheValid {
			tr.cacheMux.RLock()
			templates := make([]Template, 0, len(tr.cache))
			for _, tmpl := range tr.cache {
				templates = append(templates, tmpl)
			}
			tr.cacheMux.RUnlock()
			return templates, nil
		}
	}

	var templates []Template
	newCache := make(map[string]Template)

	// Try catalog-based structure first (screaming architecture)
	catalogPath := filepath.Join(tr.templatesDir, "catalog.json")
	if _, err := os.Stat(catalogPath); err == nil {
		catalog, err := tr.loadCatalog(catalogPath)
		if err == nil {
			for id, ref := range catalog.Templates {
				templatePath := filepath.Join(tr.templatesDir, ref.Path)
				template, err := tr.loadTemplate(templatePath)
				if err != nil {
					util.LogStructured("template_load_error", map[string]interface{}{
						"id":    id,
						"path":  ref.Path,
						"error": err.Error(),
					})
					continue
				}
				templates = append(templates, template)
				newCache[template.ID] = template
			}
		}
	}

	// Fall back to legacy flat structure if no catalog templates found
	if len(templates) == 0 {
		entries, err := os.ReadDir(tr.templatesDir)
		if err != nil {
			return nil, fmt.Errorf("failed to read templates directory: %w", err)
		}

		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
				continue
			}
			if entry.Name() == "catalog.json" {
				continue
			}

			templatePath := filepath.Join(tr.templatesDir, entry.Name())
			template, err := tr.loadTemplate(templatePath)
			if err != nil {
				util.LogStructured("template_load_error", map[string]interface{}{
					"file":  entry.Name(),
					"error": err.Error(),
				})
				continue
			}

			templates = append(templates, template)
			newCache[template.ID] = template
		}
	}

	// Update cache
	if tr.cache != nil {
		tr.cacheMux.Lock()
		tr.cache = newCache
		tr.cacheExpiry = time.Now().Add(tr.cacheDuration)
		tr.cacheMux.Unlock()
	}

	return templates, nil
}

// GetTemplate returns a specific template by ID (with caching)
func (tr *TemplateRegistry) GetTemplate(id string) (*Template, error) {
	// Check cache first
	if tr.cache != nil {
		tr.cacheMux.RLock()
		if time.Now().Before(tr.cacheExpiry) {
			if cached, ok := tr.cache[id]; ok {
				tr.cacheMux.RUnlock()
				return &cached, nil
			}
		}
		tr.cacheMux.RUnlock()
	}

	// Try catalog-based lookup first
	var templatePath string
	catalogPath := filepath.Join(tr.templatesDir, "catalog.json")
	if _, err := os.Stat(catalogPath); err == nil {
		catalog, err := tr.loadCatalog(catalogPath)
		if err == nil {
			if ref, ok := catalog.Templates[id]; ok {
				templatePath = filepath.Join(tr.templatesDir, ref.Path)
			}
		}
	}

	// Fall back to legacy flat structure
	if templatePath == "" {
		templatePath = filepath.Join(tr.templatesDir, id+".json")
	}

	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("template not found: %s", id)
	}

	template, err := tr.loadTemplate(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
	}

	// Update cache
	if tr.cache != nil {
		tr.cacheMux.Lock()
		tr.cache[id] = template
		if tr.cacheExpiry.IsZero() || time.Now().After(tr.cacheExpiry) {
			tr.cacheExpiry = time.Now().Add(tr.cacheDuration)
		}
		tr.cacheMux.Unlock()
	}

	return &template, nil
}

// loadCatalog reads and parses the template catalog index
func (tr *TemplateRegistry) loadCatalog(path string) (*TemplateCatalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read catalog: %w", err)
	}

	var catalog TemplateCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("failed to parse catalog JSON: %w", err)
	}

	return &catalog, nil
}

// loadTemplate reads and parses a template JSON file
func (tr *TemplateRegistry) loadTemplate(path string) (Template, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Template{}, fmt.Errorf("failed to read file: %w", err)
	}

	var template Template
	if err := json.Unmarshal(data, &template); err != nil {
		return Template{}, fmt.Errorf("failed to parse template JSON: %w", err)
	}

	return template, nil
}
