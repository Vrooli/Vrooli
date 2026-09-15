package preflight

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
)

var (
	resolveScenarioURLDefault = discovery.ResolveScenarioURLDefault
	httpClientFactory         = func() *http.Client { return &http.Client{Timeout: 5 * time.Second} }
)

// ScenarioRequirements contains analyzer-derived resource requirements.
type ScenarioRequirements struct {
	RAMKB      int64
	DiskKB     int64
	CPUCores   float64
	Tier       string
	Source     string
	Confidence string
}

// ScenarioRequirementsFetcher resolves scenario requirements for preflight sizing.
type ScenarioRequirementsFetcher func(ctx context.Context, scenarioID string) (*ScenarioRequirements, error)

// fetchScenarioRequirementsFromAnalyzer retrieves requirements from scenario-dependency-analyzer.
func fetchScenarioRequirementsFromAnalyzer(ctx context.Context, scenarioID string) (*ScenarioRequirements, error) {
	baseURL, err := resolveScenarioURLDefault(ctx, "scenario-dependency-analyzer")
	if err != nil {
		return nil, fmt.Errorf("resolve scenario-dependency-analyzer URL: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/v1/scenarios/%s/deployment?refresh=true", strings.TrimSuffix(baseURL, "/"), url.PathEscape(scenarioID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create analyzer requirements request: %w", err)
	}

	client := httpClientFactory()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("analyzer requirements request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("analyzer requirements returned status %d", resp.StatusCode)
	}

	var payload struct {
		Aggregates map[string]struct {
			EstimatedRequirements struct {
				RAMMB    float64 `json:"ram_mb"`
				DiskMB   float64 `json:"disk_mb"`
				CPUCores float64 `json:"cpu_cores"`
			} `json:"estimated_requirements"`
		} `json:"aggregates"`
		MetadataGaps *struct {
			TotalGaps int `json:"total_gaps"`
		} `json:"metadata_gaps,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode analyzer requirements response: %w", err)
	}

	tier, estimate, ok := selectRequirementEstimate(payload.Aggregates)
	if !ok {
		return nil, fmt.Errorf("analyzer report has no numeric requirement estimates")
	}

	confidence := "unknown"
	if payload.MetadataGaps != nil {
		if payload.MetadataGaps.TotalGaps == 0 {
			confidence = "medium"
		} else {
			confidence = "low"
		}
	}

	return &ScenarioRequirements{
		RAMKB:      int64(math.Ceil(estimate.RAMMB * 1024)),
		DiskKB:     int64(math.Ceil(estimate.DiskMB * 1024)),
		CPUCores:   estimate.CPUCores,
		Tier:       tier,
		Source:     "scenario-dependency-analyzer",
		Confidence: confidence,
	}, nil
}

type requirementEstimate struct {
	RAMMB    float64
	DiskMB   float64
	CPUCores float64
}

func (r requirementEstimate) hasNumericValues() bool {
	return r.RAMMB > 0 || r.DiskMB > 0 || r.CPUCores > 0
}

func selectRequirementEstimate(aggregates map[string]struct {
	EstimatedRequirements struct {
		RAMMB    float64 `json:"ram_mb"`
		DiskMB   float64 `json:"disk_mb"`
		CPUCores float64 `json:"cpu_cores"`
	} `json:"estimated_requirements"`
},
) (string, requirementEstimate, bool) {
	if len(aggregates) == 0 {
		return "", requirementEstimate{}, false
	}

	preferredTiers := []string{"tier-4-saas", "saas", "server", "tier-1-local", "local"}
	for _, tier := range preferredTiers {
		aggregate, ok := aggregates[tier]
		if !ok {
			continue
		}
		estimate := requirementEstimate{
			RAMMB:    aggregate.EstimatedRequirements.RAMMB,
			DiskMB:   aggregate.EstimatedRequirements.DiskMB,
			CPUCores: aggregate.EstimatedRequirements.CPUCores,
		}
		if estimate.hasNumericValues() {
			return tier, estimate, true
		}
	}

	var (
		selectedTier string
		selected     requirementEstimate
		found        bool
	)
	for tier, aggregate := range aggregates {
		estimate := requirementEstimate{
			RAMMB:    aggregate.EstimatedRequirements.RAMMB,
			DiskMB:   aggregate.EstimatedRequirements.DiskMB,
			CPUCores: aggregate.EstimatedRequirements.CPUCores,
		}
		if !estimate.hasNumericValues() {
			continue
		}
		if !found || estimate.RAMMB > selected.RAMMB || (estimate.RAMMB == selected.RAMMB && estimate.DiskMB > selected.DiskMB) {
			selectedTier = tier
			selected = estimate
			found = true
		}
	}

	return selectedTier, selected, found
}
