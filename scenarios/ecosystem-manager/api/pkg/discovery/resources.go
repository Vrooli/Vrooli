package discovery

import (
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ecosystem-manager/api/pkg/tasks"
)

// DiscoverResources scans the filesystem for available resources
func DiscoverResources() ([]tasks.ResourceInfo, error) {
	var resources []tasks.ResourceInfo

	// Step 1: Find the Vrooli root directory and scan filesystem for all resources
	vrooliRoot, err := findVrooliRoot()
	if err != nil {
		log.Printf("Warning: Could not find Vrooli root directory: %v", err)
		return resources, nil
	}

	resourcesDir := filepath.Join(vrooliRoot, "resources")
	log.Printf("Scanning for resources in: %s", resourcesDir)

	// Get all resource folders from filesystem
	allResourceFolders := make(map[string]string) // name -> path
	if _, err := os.Stat(resourcesDir); err == nil {
		entries, err := os.ReadDir(resourcesDir)
		if err != nil {
			log.Printf("Warning: Failed to read resources directory: %v", err)
		} else {
			for _, entry := range entries {
				if entry.IsDir() {
					resourceName := entry.Name()
					resourcePath := filepath.Join(resourcesDir, resourceName)
					allResourceFolders[resourceName] = resourcePath
				}
			}
		}
	}

	// Step 2: Get registered resources from vrooli CLI (with status info)
	registeredResources := make(map[string]tasks.ResourceInfo)
	cmd := exec.Command("vrooli", "resource", "list", "--json", "--verbose")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Warning: Failed to run 'vrooli resource list --json --verbose': %v", err)
		// Try without verbose flag as fallback
		cmd = exec.Command("vrooli", "resource", "list", "--json")
		output, err = cmd.Output()
		if err != nil {
			log.Printf("Warning: Failed to get vrooli resources: %v", err)
		}
	}

	if err == nil {
		var vrooliResources []map[string]interface{}
		if err := json.Unmarshal(output, &vrooliResources); err == nil {
			for _, vr := range vrooliResources {
				resourceName := getStringField(vr, "Name")
				if resourceName != "" {
					registeredResources[resourceName] = tasks.ResourceInfo{
						Name:        resourceName,
						Path:        getStringField(vr, "path"),
						Port:        getIntField(vr, "port"),
						Category:    getStringField(vr, "category"),
						Description: getStringField(vr, "description"),
						Version:     getStringField(vr, "version"),
						Healthy:     getBoolField(vr, "Running"),
					}
				}
			}
		}
	}

	// Step 3: Merge filesystem and CLI data
	for resourceName, resourcePath := range allResourceFolders {
		if registeredInfo, exists := registeredResources[resourceName]; exists {
			// Use registered info with CLI status
			resources = append(resources, registeredInfo)
		} else {
			// Create info from filesystem scan for unregistered resources
			resourceInfo := extractResourceInfoFromFilesystem(resourceName, resourcePath)
			resources = append(resources, resourceInfo)
		}
	}

	// Sort resources alphabetically by name
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].Name < resources[j].Name
	})

	log.Printf("Discovered %d resources (%d registered, %d unregistered)", 
		len(resources), len(registeredResources), len(allResourceFolders)-len(registeredResources))
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

// findVrooliRoot finds the Vrooli root directory by walking up the directory tree
func findVrooliRoot() (string, error) {
	// Start from current working directory
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up the directory tree looking for resources/ folder
	dir := pwd
	for {
		resourcesPath := filepath.Join(dir, "resources")
		if info, err := os.Stat(resourcesPath); err == nil && info.IsDir() {
			return dir, nil
		}
		
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached the root directory
			break
		}
		dir = parent
	}

	return "", os.ErrNotExist
}

// extractResourceInfoFromFilesystem creates ResourceInfo from filesystem scan
func extractResourceInfoFromFilesystem(name, path string) tasks.ResourceInfo {
	resource := tasks.ResourceInfo{
		Name:        name,
		Path:        path,
		Category:    inferResourceCategory(name),
		Description: extractResourceDescription(path),
		Healthy:     false, // Unregistered resources are considered unhealthy
		Version:     "unregistered",
	}

	return resource
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

// extractResourceDescription tries to get description from resource files
func extractResourceDescription(resourcePath string) string {
	// Try to read README.md
	readmePath := filepath.Join(resourcePath, "README.md")
	if data, err := os.ReadFile(readmePath); err == nil {
		content := string(data)
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !strings.HasPrefix(trimmed, "#") && len(trimmed) > 10 {
				if len(trimmed) > 100 {
					return trimmed[:100] + "..."
				}
				return trimmed
			}
		}
	}

	// Try to read service.json for description
	servicePath := filepath.Join(resourcePath, ".vrooli", "service.json")
	if data, err := os.ReadFile(servicePath); err == nil {
		var serviceConfig map[string]interface{}
		if json.Unmarshal(data, &serviceConfig) == nil {
			if desc := getStringField(serviceConfig, "description"); desc != "" {
				return desc
			}
		}
	}

	return "Unregistered resource - no description available"
}
