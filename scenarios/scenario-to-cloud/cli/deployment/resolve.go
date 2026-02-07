package deployment

import (
	"fmt"
	"strings"
	"time"

	internalmanifest "scenario-to-cloud/cli/internal/manifest"
)

// ManifestSelector captures the identifying fields used to map a manifest to a deployment.
type ManifestSelector struct {
	ScenarioID string `json:"scenario_id,omitempty"`
	Host       string `json:"host"`
	Domain     string `json:"domain,omitempty"`
}

// ReadSelectorFromManifest extracts scenario and VPS target fields from a manifest file.
func ReadSelectorFromManifest(path string) (ManifestSelector, error) {
	raw, err := internalmanifest.ReadJSONFile(path)
	if err != nil {
		return ManifestSelector{}, err
	}

	scenarioID := strings.TrimSpace(getNestedString(raw, "scenario", "id"))
	host := strings.TrimSpace(getNestedString(raw, "target", "vps", "host"))
	domain := strings.TrimSpace(getNestedString(raw, "edge", "domain"))

	if scenarioID == "" {
		return ManifestSelector{}, fmt.Errorf("manifest is missing required field: scenario.id")
	}
	if host == "" {
		return ManifestSelector{}, fmt.Errorf("manifest is missing required field: target.vps.host")
	}

	return ManifestSelector{
		ScenarioID: scenarioID,
		Host:       host,
		Domain:     domain,
	}, nil
}

// ResolveLatestBySelector returns the newest deployment matching scenario + host.
func ResolveLatestBySelector(client *Client, selector ManifestSelector) (*DeploymentSummary, error) {
	host := strings.ToLower(strings.TrimSpace(selector.Host))
	domain := strings.ToLower(strings.TrimSpace(selector.Domain))
	scenarioID := strings.TrimSpace(selector.ScenarioID)
	if host == "" {
		return nil, fmt.Errorf("host is required")
	}

	opts := ListOptions{}
	if scenarioID != "" {
		opts.ScenarioID = scenarioID
	}

	_, listResp, err := client.List(opts)
	if err != nil {
		return nil, err
	}

	var latest *DeploymentSummary
	for i := range listResp.Deployments {
		candidate := listResp.Deployments[i]
		if strings.ToLower(strings.TrimSpace(candidate.Host)) != host {
			continue
		}
		if scenarioID != "" && strings.TrimSpace(candidate.ScenarioID) != scenarioID {
			continue
		}
		if domain != "" && strings.ToLower(strings.TrimSpace(candidate.Domain)) != domain {
			continue
		}
		if latest == nil || candidate.CreatedAt.After(latest.CreatedAt) {
			copy := candidate
			latest = &copy
		}
	}

	return latest, nil
}

func getNestedString(m map[string]interface{}, path ...string) string {
	var current interface{} = m
	for i, key := range path {
		obj, ok := current.(map[string]interface{})
		if !ok {
			return ""
		}
		value, ok := obj[key]
		if !ok {
			return ""
		}
		if i == len(path)-1 {
			s, ok := value.(string)
			if !ok {
				return ""
			}
			return s
		}
		current = value
	}
	return ""
}

func formatOptionalTime(t *time.Time) string {
	if t == nil {
		return "n/a"
	}
	return t.UTC().Format(time.RFC3339)
}
