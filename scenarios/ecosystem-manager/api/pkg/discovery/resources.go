package discovery

import (
	"encoding/json"
	"log"
	"os/exec"
	"sort"
	"strings"

	"github.com/ecosystem-manager/api/pkg/tasks"
)

// DiscoverResources gets all available resources from vrooli CLI
func DiscoverResources() ([]tasks.ResourceInfo, error) {
	var resources []tasks.ResourceInfo

	// Get all resources from vrooli CLI (now includes unregistered resources)
	cmd := exec.Command("vrooli", "resource", "list", "--json", "--verbose")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Warning: Failed to run 'vrooli resource list --json --verbose': %v", err)
		// Try without verbose flag as fallback
		cmd = exec.Command("vrooli", "resource", "list", "--json")
		output, err = cmd.Output()
		if err != nil {
			log.Printf("Error: Failed to get vrooli resources: %v", err)
			return resources, err
		}
	}

	var vrooliResources []map[string]interface{}
	if err := json.Unmarshal(output, &vrooliResources); err != nil {
		log.Printf("Error: Failed to parse vrooli resource list output: %v", err)
		return resources, err
	}

	for _, vr := range vrooliResources {
		resourceName := getStringField(vr, "Name")
		if resourceName != "" {
			status := getStringField(vr, "Status")
			isRegistered := status != "[UNREGISTERED]" && status != "[MISSING]"

			resource := tasks.ResourceInfo{
				Name:        resourceName,
				Path:        getStringField(vr, "Path"),
				Port:        getIntField(vr, "Port"),
				Category:    inferResourceCategory(resourceName), // Still infer category from name
				Description: getStringField(vr, "Description"),
				Version:     getStringField(vr, "Version"),
				Healthy:     getBoolField(vr, "Running"),
				Status:      status,
			}

			// If no version provided and unregistered, mark it
			if resource.Version == "" && !isRegistered {
				resource.Version = "unregistered"
			}

			resources = append(resources, resource)
		}
	}

	// Sort resources alphabetically by name
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].Name < resources[j].Name
	})

	log.Printf("Discovered %d resources from vrooli CLI", len(resources))
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

// inferResourceCategory attempts to categorize a resource based on its name
func inferResourceCategory(name string) string {
	lower := strings.ToLower(name)

	if strings.Contains(lower, "postgres") || strings.Contains(lower, "mysql") || strings.Contains(lower, "redis") {
		return "database"
	}
	if strings.Contains(lower, "n8n") || strings.Contains(lower, "workflow") {
		return "automation"
	}
	if strings.Contains(lower, "docker") || strings.Contains(lower, "k8s") || strings.Contains(lower, "kubernetes") {
		return "infrastructure"
	}
	if strings.Contains(lower, "ollama") || strings.Contains(lower, "ai") || strings.Contains(lower, "ml") {
		return "ai"
	}
	if strings.Contains(lower, "minio") || strings.Contains(lower, "storage") {
		return "storage"
	}
	if strings.Contains(lower, "monitoring") || strings.Contains(lower, "metrics") {
		return "monitoring"
	}

	return "misc" // default
}
