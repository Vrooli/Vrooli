// Package resources owns resource discovery and the catalog-derived resource
// grouping used by onboarding. Transport handlers only project its typed
// responses onto Connect.
package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	vroolicli "github.com/vrooli/vrooli-cli-go"
	"github.com/vrooli/vrooli/internal/clock"
	"github.com/vrooli/vrooli/internal/operatorstate"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	resourcesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/resources"
)

type Scenario struct {
	Name    string
	Enabled bool
}

type ClosureMember struct {
	Name     string
	Required bool
}

type Service struct {
	Client         *vroolicli.Client
	Root           func() (string, error)
	LoadState      func(context.Context) (operatorstate.Document, error)
	LoadScenarios  func(context.Context) ([]Scenario, error)
	ResolveClosure func(string, []Scenario, operatorstate.Document) ([]ClosureMember, error)
	Now            func() time.Time
}

func (s Service) List(ctx context.Context) (*resourcesv1.ListResourcesResponse, error) {
	items, err := s.load(ctx)
	if err != nil {
		return nil, err
	}
	return &resourcesv1.ListResourcesResponse{Resources: items, Count: int32(len(items)), LoadedAt: s.now().UTC().Format(time.RFC3339)}, nil
}

func (s Service) Get(ctx context.Context, name string) (*resourcesv1.Resource, error) {
	items, err := s.load(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if strings.EqualFold(item.GetName(), name) {
			return item, nil
		}
	}
	return nil, fmt.Errorf("resource not found: %s", name)
}

func (s Service) Health(ctx context.Context) (*resourcesv1.GetResourceHealthResponse, error) {
	items, err := s.load(ctx)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC().Format(time.RFC3339)
	result := &resourcesv1.GetResourceHealthResponse{Total: int32(len(items)), CheckedAt: now}
	for _, item := range items {
		available := item.GetStatus() == "running"
		if available {
			result.HealthyCount++
		}
		result.Resources = append(result.Resources, &resourcesv1.ResourceHealth{Name: item.GetName(), Status: item.GetStatus(), Category: item.GetCategory(), Available: available, LastChecked: now})
	}
	return result, nil
}

func (s Service) Derived(ctx context.Context) (*resourcesv1.ListDerivedResourcesResponse, error) {
	if s.Root == nil || s.LoadState == nil || s.LoadScenarios == nil || s.ResolveClosure == nil {
		return nil, fmt.Errorf("resource catalog service is not configured")
	}
	root, err := s.Root()
	if err != nil {
		return nil, err
	}
	state, err := s.LoadState(ctx)
	if err != nil {
		return nil, err
	}
	models, err := loadCatalog(root, state)
	if err != nil {
		return nil, err
	}
	scenarios, err := s.LoadScenarios(ctx)
	if err != nil {
		return nil, err
	}
	closure, err := s.ResolveClosure(root, scenarios, state)
	if err != nil {
		return nil, err
	}
	byName := make(map[string]*resourcesv1.Resource, len(models))
	result := &resourcesv1.ListDerivedResourcesResponse{Resources: models, Count: int32(len(models)), Required: []*resourcesv1.Resource{}, Optional: []*resourcesv1.Resource{}, Standalone: []*resourcesv1.Resource{}}
	for _, item := range models {
		byName[item.GetName()] = item
	}
	for _, member := range closure {
		item, ok := byName[member.Name]
		if !ok {
			continue
		}
		copy := *item
		if member.Required {
			copy.Enabled = true
			result.Required = append(result.Required, &copy)
		} else {
			result.Optional = append(result.Optional, &copy)
		}
		delete(byName, member.Name)
	}
	for _, item := range byName {
		result.Standalone = append(result.Standalone, item)
	}
	sortResources(result.Required)
	sortResources(result.Optional)
	sortResources(result.Standalone)
	return result, nil
}

// Catalog returns immutable resource declarations for the selection service.
func (s Service) Catalog(ctx context.Context) ([]*resourcesv1.Resource, error) {
	if s.Root == nil || s.LoadState == nil {
		return nil, fmt.Errorf("resource catalog service is not configured")
	}
	root, err := s.Root()
	if err != nil {
		return nil, err
	}
	state, err := s.LoadState(ctx)
	if err != nil {
		return nil, err
	}
	return loadCatalog(root, state)
}

func (s Service) load(ctx context.Context) ([]*resourcesv1.Resource, error) {
	if s.Client == nil {
		return nil, fmt.Errorf("resource status client is not configured")
	}
	resp, err := s.Client.ResourceStatuses(ctx)
	if err != nil {
		return nil, fmt.Errorf("vrooli resource status failed: %w", err)
	}
	root, err := s.Root()
	if err != nil {
		return nil, err
	}
	result := make([]*resourcesv1.Resource, 0, len(resp.GetResources()))
	for _, item := range resp.GetResources() {
		name := item.GetResource().GetName()
		if strings.TrimSpace(name) == "" {
			continue
		}
		result = append(result, &resourcesv1.Resource{Name: name, Status: normalizeStatus(item), Category: categorize(root, name), Installed: item.GetInstalled()})
	}
	return result, nil
}

func normalizeStatus(item *cliv1.ResourceStatus) string {
	if item.GetRunning() {
		return "running"
	}
	status := strings.ToLower(strings.TrimSpace(item.GetHealth()))
	if status == "" {
		status = strings.ToLower(strings.TrimSpace(item.GetMessage()))
	}
	switch {
	case strings.Contains(status, "stopped"), strings.Contains(status, "not installed"):
		return "stopped"
	case item.GetInstalled():
		return "installed"
	default:
		return "stopped"
	}
}

func categorize(root, name string) string {
	data, err := os.ReadFile(filepath.Join(root, "resources", name, "resource.json"))
	if err == nil {
		var manifest struct {
			Category string `json:"category"`
		}
		if json.Unmarshal(data, &manifest) == nil && strings.TrimSpace(manifest.Category) != "" {
			return strings.TrimSpace(manifest.Category)
		}
	}
	return "general"
}

func loadCatalog(root string, state operatorstate.Document) ([]*resourcesv1.Resource, error) {
	entries, err := os.ReadDir(filepath.Join(root, "resources"))
	if err != nil {
		return nil, err
	}
	result := make([]*resourcesv1.Resource, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, "resources", entry.Name(), "resource.json")
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		var manifest struct {
			Name        string `json:"name"`
			DisplayName string `json:"display_name"`
			Description string `json:"description"`
			Category    string `json:"category"`
		}
		if err := json.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("decode %s: %w", path, err)
		}
		name := strings.TrimSpace(manifest.Name)
		if name == "" {
			name = entry.Name()
		}
		enabled := false
		if choice, ok := state.Resources[name]; ok && choice.Enabled != nil {
			enabled = *choice.Enabled
		}
		result = append(result, &resourcesv1.Resource{Name: name, DisplayName: manifest.DisplayName, Description: manifest.Description, Category: manifest.Category, Enabled: enabled, Installed: true})
	}
	sortResources(result)
	return result, nil
}

func sortResources(items []*resourcesv1.Resource) {
	sort.Slice(items, func(i, j int) bool { return items[i].GetName() < items[j].GetName() })
}
func (s Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return clock.Real{}.Now()
}
