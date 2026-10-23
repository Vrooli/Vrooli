package surfaces

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
)

type Application struct {
	Scenario     string
	Label        string
	Description  string
	RuntimeState string
	IconPath     string
}

type Applications struct {
	Run         func(context.Context) ([]byte, error)
	ResolveURL  func(context.Context, string, string) (string, error)
	ResolvePort func(context.Context, string, string) (int, error)
	Branding    *ApplicationBranding
}

func DefaultApplications() *Applications {
	return &Applications{Run: func(ctx context.Context) ([]byte, error) {
		return exec.CommandContext(ctx, "vrooli", "scenario", "list", "--json").Output()
	}, ResolveURL: discovery.ResolveExternalURL, ResolvePort: discovery.ResolveScenarioPort, Branding: DefaultApplicationBranding()}
}

var scenarioName = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)

func (a *Applications) List(ctx context.Context) ([]Application, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	data, err := a.Run(ctx)
	if err != nil || len(data) > 2<<20 {
		return nil, fmt.Errorf("application inventory unavailable")
	}
	var inventory struct {
		Success   bool `json:"success"`
		Scenarios []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Status      string `json:"status"`
		} `json:"scenarios"`
	}
	if err := json.Unmarshal(data, &inventory); err != nil || !inventory.Success || len(inventory.Scenarios) > 512 {
		return nil, fmt.Errorf("invalid application inventory")
	}
	result := []Application{}
	for _, item := range inventory.Scenarios {
		if !scenarioName.MatchString(item.Name) {
			continue
		}
		app := Application{Scenario: item.Name, Label: strings.ReplaceAll(item.Name, "-", " "), Description: item.Description, RuntimeState: item.Status}
		if a.Branding != nil {
			if brand, err := a.Branding.Metadata(ctx, item.Name); err == nil {
				if brand.Label != "" {
					app.Label = brand.Label
				}
				app.IconPath = brand.IconPath
			}
		}
		result = append(result, app)
	}
	return result, nil
}

func (a *Applications) Resolve(ctx context.Context, name, host string) (Application, string, error) {
	if !scenarioName.MatchString(name) {
		return Application{}, "", fmt.Errorf("invalid application identity")
	}
	items, err := a.List(ctx)
	if err != nil {
		return Application{}, "", err
	}
	for _, item := range items {
		if item.Scenario != name {
			continue
		}
		if item.RuntimeState != "running" {
			return item, "", fmt.Errorf("application is stopped; start it through the control plane")
		}
		address, err := a.ResolveURL(ctx, name, host)
		return item, address, err
	}
	return Application{}, "", fmt.Errorf("unknown application")
}

// ResolveGateway keeps all browser traffic on the caller's existing Portal
// origin. It is opt-in; existing direct URL clients retain their contract.
func (a *Applications) ResolveGateway(ctx context.Context, name string) (Application, string, error) {
	if !scenarioName.MatchString(name) || name == "portal" {
		return Application{}, "", fmt.Errorf("application cannot be hosted through itself")
	}
	items, err := a.List(ctx)
	if err != nil {
		return Application{}, "", err
	}
	for _, item := range items {
		if item.Scenario != name {
			continue
		}
		if item.RuntimeState != "running" {
			return item, "", fmt.Errorf("application is stopped; start it through the control plane")
		}
		return item, "/apps/" + name + "/proxy/", nil
	}
	return Application{}, "", fmt.Errorf("unknown application")
}

// ProxyTarget exposes numeric owner-discovered ports only for a known running
// application. The shared api-base gateway remains the proxy implementation.
func (a *Applications) ProxyTarget(ctx context.Context, name string) (map[string]any, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if _, _, err := a.ResolveGateway(ctx, name); err != nil {
		return nil, err
	}
	if a.ResolvePort == nil {
		return nil, fmt.Errorf("application port discovery unavailable")
	}
	ui, err := a.ResolvePort(ctx, name, discovery.UIPortKey)
	if err != nil || ui < 1024 || ui > 65535 {
		return nil, fmt.Errorf("application UI unavailable")
	}
	ports := map[string]int{"UI_PORT": ui}
	if api, err := a.ResolvePort(ctx, name, "API_PORT"); err == nil && api >= 1024 && api <= 65535 {
		ports["API_PORT"] = api
	}
	return map[string]any{"id": name, "scenario_name": name, "port_mappings": ports}, nil
}
