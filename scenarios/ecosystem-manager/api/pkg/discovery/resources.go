package discovery

import (
	"encoding/json"
	"log"
	"os/exec"
	"sort"

	"github.com/ecosystem-manager/api/pkg/tasks"
)

// DiscoverResources scans the filesystem for available resources
func DiscoverResources() ([]tasks.ResourceInfo, error) {
	var resources []tasks.ResourceInfo

	// Use vrooli command to get resources list with verbose output
	cmd := exec.Command("vrooli", "resource", "list", "--json", "--verbose")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Warning: Failed to run 'vrooli resource list --json --verbose': %v", err)
		// Try without verbose flag as fallback
		cmd = exec.Command("vrooli", "resource", "list", "--json")
		output, err = cmd.Output()
		if err != nil {
			log.Printf("Warning: Fallback failed too: %v", err)
			// Return empty list instead of error to prevent UI issues
			return resources, nil
		}
	}

	// Parse the JSON output
	var vrooliResources []map[string]interface{}
	if err := json.Unmarshal(output, &vrooliResources); err != nil {
		log.Printf("Warning: Failed to parse vrooli resource list output: %v", err)
		return resources, nil
	}

	// Convert to our ResourceInfo format
	for _, vr := range vrooliResources {
		resource := tasks.ResourceInfo{
			Name:        getStringField(vr, "Name"), // vrooli uses uppercase "Name"
			Path:        getStringField(vr, "path"),
			Port:        getIntField(vr, "port"),
			Category:    getStringField(vr, "category"),
			Description: getStringField(vr, "description"),
			Version:     getStringField(vr, "version"),
			Healthy:     getBoolField(vr, "Running"), // vrooli uses "Running" not "healthy"
		}

		// Skip empty entries
		if resource.Name != "" {
			resources = append(resources, resource)
		}
	}

	// Sort resources alphabetically by name
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].Name < resources[j].Name
	})

	log.Printf("Discovered %d resources via vrooli command", len(resources))
	return resources, nil
}

// Helper functions for safe field extraction
func getStringField(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getIntField(m map[string]interface{}, key string) int {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case float64:
			return int(v)
		case int:
			return v
		}
	}
	return 0
}

func getBoolField(m map[string]interface{}, key string) bool {
	if val, ok := m[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}
