package discovery

import (
	"context"
	"log"
	"sort"
	"strings"

	"github.com/ecosystem-manager/api/pkg/tasks"
	vroolicli "github.com/vrooli/vrooli-cli-go"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

var defaultClient = vroolicli.New()

// DiscoverResources gets all available resources from vrooli CLI
func DiscoverResources() ([]tasks.ResourceInfo, error) {
	return discoverResources(defaultClient)
}

func discoverResources(client *vroolicli.Client) ([]tasks.ResourceInfo, error) {
	var resources []tasks.ResourceInfo

	resp, err := client.ListResources(context.Background())
	if err != nil {
		log.Printf("Error: Failed to get vrooli resources: %v", err)
		return resources, err
	}

	for _, vr := range resp.GetResources() {
		if vr.GetName() == "" {
			continue
		}
		resources = append(resources, tasks.ResourceInfo{
			Name:     vr.GetName(),
			Path:     vr.GetPath(),
			Category: inferResourceCategory(vr.GetName()), // Still infer category from name
			Healthy:  vr.GetEnabled(),
			Status:   resourceStatus(vr),
		})
	}

	// Sort resources alphabetically by name
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].Name < resources[j].Name
	})

	log.Printf("Discovered %d resources from vrooli CLI", len(resources))
	return resources, nil
}

// resourceStatus derives a human-facing status string from the typed resource's
// registration flags (the CLI reports flags, not a literal status field).
func resourceStatus(vr *cliv1.Resource) string {
	switch {
	case !vr.GetExists():
		return "[MISSING]"
	case !vr.GetRegistered():
		return "[UNREGISTERED]"
	case !vr.GetEnabled():
		return "disabled"
	default:
		return "enabled"
	}
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
