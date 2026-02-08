package preflight

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSelectRequirementEstimate(t *testing.T) {
	t.Parallel()

	tier, estimate, ok := selectRequirementEstimate(map[string]struct {
		EstimatedRequirements struct {
			RAMMB    float64 `json:"ram_mb"`
			DiskMB   float64 `json:"disk_mb"`
			CPUCores float64 `json:"cpu_cores"`
		} `json:"estimated_requirements"`
	}{
		"server": {
			EstimatedRequirements: struct {
				RAMMB    float64 `json:"ram_mb"`
				DiskMB   float64 `json:"disk_mb"`
				CPUCores float64 `json:"cpu_cores"`
			}{RAMMB: 1024, DiskMB: 2048, CPUCores: 2},
		},
		"tier-4-saas": {
			EstimatedRequirements: struct {
				RAMMB    float64 `json:"ram_mb"`
				DiskMB   float64 `json:"disk_mb"`
				CPUCores float64 `json:"cpu_cores"`
			}{RAMMB: 1536, DiskMB: 1024, CPUCores: 1.5},
		},
	})
	if !ok {
		t.Fatal("expected estimate to be selected")
	}
	if tier != "tier-4-saas" {
		t.Fatalf("expected preferred tier tier-4-saas, got %s", tier)
	}
	if estimate.RAMMB != 1536 {
		t.Fatalf("expected RAMMB=1536, got %f", estimate.RAMMB)
	}
}

func TestFetchScenarioRequirementsFromAnalyzer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/scenarios/landing-page-business-suite/deployment" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		fmt.Fprint(w, `{
			"aggregates": {
				"tier-4-saas": {
					"estimated_requirements": {"ram_mb": 2048, "disk_mb": 6144, "cpu_cores": 2}
				}
			},
			"metadata_gaps": {"total_gaps": 0}
		}`)
	}))
	defer server.Close()

	prevResolver := resolveScenarioURLDefault
	prevClientFactory := httpClientFactory
	resolveScenarioURLDefault = func(ctx context.Context, scenarioSlug string) (string, error) {
		_ = ctx
		if scenarioSlug != "scenario-dependency-analyzer" {
			return "", fmt.Errorf("unexpected scenario slug: %s", scenarioSlug)
		}
		return server.URL, nil
	}
	httpClientFactory = func() *http.Client { return server.Client() }
	t.Cleanup(func() {
		resolveScenarioURLDefault = prevResolver
		httpClientFactory = prevClientFactory
	})

	estimate, err := fetchScenarioRequirementsFromAnalyzer(context.Background(), "landing-page-business-suite")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if estimate == nil {
		t.Fatal("expected non-nil estimate")
	}
	if estimate.RAMKB != 2097152 {
		t.Fatalf("expected RAMKB=2097152, got %d", estimate.RAMKB)
	}
	if estimate.DiskKB != 6291456 {
		t.Fatalf("expected DiskKB=6291456, got %d", estimate.DiskKB)
	}
	if estimate.Tier != "tier-4-saas" {
		t.Fatalf("expected tier-4-saas, got %s", estimate.Tier)
	}
	if estimate.Source != "scenario-dependency-analyzer" {
		t.Fatalf("expected analyzer source, got %s", estimate.Source)
	}
}
