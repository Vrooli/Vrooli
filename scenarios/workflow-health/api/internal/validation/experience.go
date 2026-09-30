package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"workflow-health/internal/workflows"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	contractv1 "github.com/vrooli/vrooli/packages/proto/gen/go/experience-manager/v1/contract"
	contractconnect "github.com/vrooli/vrooli/packages/proto/gen/go/experience-manager/v1/contract/contract_v1connect"
)

// ReadinessProfileFetcher is the only contract boundary Workflow Health uses.
// Experience Manager owns compilation; Workflow Health never reads authored
// experience files or maintains a parallel readiness registry.
type ReadinessProfileFetcher func(context.Context, string) (*readinessProfile, error)

type readinessProfile struct {
	Version string          `json:"version"`
	Pages   []readinessPage `json:"pages"`
}

type readinessPage struct {
	ID      string            `json:"id"`
	Routes  []string          `json:"routes"`
	Regions []readinessRegion `json:"regions"`
}

type readinessRegion struct {
	ID       string `json:"id"`
	Required bool   `json:"required"`
	Binding  struct {
		TestID   string `json:"testid"`
		Selector string `json:"selector"`
	} `json:"binding"`
	Lifecycle struct {
		Kind   string   `json:"kind"`
		States []string `json:"states"`
	} `json:"lifecycle"`
}

func newExperienceManagerFetcher() ReadinessProfileFetcher {
	resolver := discovery.NewResolver(discovery.ResolverConfig{})
	client := &http.Client{Timeout: 15 * time.Second}
	return func(ctx context.Context, scenario string) (*readinessProfile, error) {
		base, err := resolver.ResolveScenarioURLDefault(ctx, "experience-manager")
		if err != nil {
			return nil, err
		}
		resp, err := contractconnect.NewContractServiceClient(client, base).GetReadinessProfile(ctx, connect.NewRequest(&contractv1.GetReadinessProfileRequest{Scenario: scenario}))
		if err != nil {
			return nil, err
		}
		var profile readinessProfile
		if err := json.Unmarshal([]byte(resp.Msg.GetProfileJson()), &profile); err != nil {
			return nil, fmt.Errorf("decode readiness profile: %w", err)
		}
		return &profile, nil
	}
}

func experienceCoverageFindings(catalog *workflows.ScenarioWorkflowCatalog, profile *readinessProfile) []Finding {
	if catalog == nil || profile == nil || len(profile.Pages) == 0 {
		return nil
	}
	pages := map[string]readinessPage{}
	for _, page := range profile.Pages {
		for _, route := range page.Routes {
			pages[route] = page
		}
	}
	assetsByRoute := map[string][]workflows.WorkflowAsset{}
	var findings []Finding
	for _, asset := range catalog.Assets {
		if asset.ParseError != "" {
			continue
		}
		for _, ref := range asset.Routes {
			if ref.Scenario != "" && ref.Scenario != catalog.Scenario {
				continue
			}
			route, _, ok := matchingReadinessPage(ref.Path, pages)
			if !ok {
				findings = append(findings, finding(CodeExperienceRouteMissing, asset.Path, asset.ID, fmt.Sprintf("Workflow route %q is not declared by the adopted experience profile.", ref.Path), false))
				continue
			}
			assetsByRoute[route] = append(assetsByRoute[route], asset)
		}
	}
	for route, assets := range assetsByRoute {
		page := pages[route]
		for _, region := range page.Regions {
			if !region.Required || (strings.TrimSpace(region.Binding.TestID) == "" && strings.TrimSpace(region.Binding.Selector) == "") {
				continue
			}
			anchor := assets[0]
			if !assetsCoverRegion(assets, region) {
				findings = append(findings, finding(CodeExperienceBindingMissing, anchor.Path, anchor.ID, fmt.Sprintf("Workflow route %q does not collectively assert required region %q through its declared runtime binding.", route, region.ID), false))
				continue
			}
			if region.Lifecycle.Kind == "async" && !assetsCoverLifecycle(assets, region) {
				findings = append(findings, finding(CodeExperienceStateMissing, anchor.Path, anchor.ID, fmt.Sprintf("Workflow route %q does not collectively cover loading plus a declared terminal lifecycle state for async region %q.", route, region.ID), false))
			}
		}
	}
	return findings
}

// matchingReadinessPage matches BAS navigation paths against Experience Manager
// route declarations. Experience contracts permit :params, while workflow URLs
// commonly include query-only state such as an example or active tab.
func matchingReadinessPage(rawPath string, pages map[string]readinessPage) (string, readinessPage, bool) {
	path := rawPath
	if before, _, found := strings.Cut(path, "?"); found {
		path = before
	}
	if before, _, found := strings.Cut(path, "#"); found {
		path = before
	}
	if page, ok := pages[path]; ok {
		return path, page, true
	}
	for route, page := range pages {
		if readinessRouteMatches(route, path) {
			return route, page, true
		}
	}
	return "", readinessPage{}, false
}

func readinessRouteMatches(pattern, path string) bool {
	patternParts := splitRoute(pattern)
	pathParts := splitRoute(path)
	if len(patternParts) != len(pathParts) {
		return false
	}
	for index, part := range patternParts {
		if strings.HasPrefix(part, ":") {
			if pathParts[index] == "" {
				return false
			}
			continue
		}
		if part != pathParts[index] {
			return false
		}
	}
	return true
}

func splitRoute(route string) []string {
	if route == "/" {
		return nil
	}
	return strings.Split(strings.Trim(route, "/"), "/")
}

func assetsCoverRegion(assets []workflows.WorkflowAsset, region readinessRegion) bool {
	for _, asset := range assets {
		if assetCoversRegion(asset, region) {
			return true
		}
	}
	return false
}

func assetsCoverLifecycle(assets []workflows.WorkflowAsset, region readinessRegion) bool {
	loading, terminal := false, false
	for _, asset := range assets {
		for _, ref := range asset.Selectors {
			raw := ref.Raw
			if !strings.Contains(raw, "data-experience-state") {
				continue
			}
			if strings.Contains(raw, "loading") {
				loading = true
			}
			for _, state := range region.Lifecycle.States {
				if state != "loading" && state != "static" && strings.Contains(raw, state) {
					terminal = true
				}
			}
		}
	}
	return loading && terminal
}

func assetCoversLifecycle(asset workflows.WorkflowAsset, region readinessRegion) bool {
	loading := false
	terminal := false
	for _, ref := range asset.Selectors {
		raw := ref.Raw
		if !strings.Contains(raw, "data-experience-state") {
			continue
		}
		if strings.Contains(raw, "loading") {
			loading = true
		}
		for _, state := range region.Lifecycle.States {
			if state != "loading" && state != "static" && strings.Contains(raw, state) {
				terminal = true
			}
		}
	}
	return loading && terminal
}

func assetCoversRegion(asset workflows.WorkflowAsset, region readinessRegion) bool {
	for _, ref := range asset.Selectors {
		raw := ref.Raw
		if region.Binding.Selector != "" && strings.Contains(raw, region.Binding.Selector) {
			return true
		}
		if region.Binding.TestID != "" && strings.Contains(raw, region.Binding.TestID) {
			return true
		}
	}
	return false
}
