package capture

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	actionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	contractv1 "github.com/vrooli/vrooli/packages/proto/gen/go/experience-manager/v1/contract"
	contractconnect "github.com/vrooli/vrooli/packages/proto/gen/go/experience-manager/v1/contract/contract_v1connect"
)

type experienceReadinessResolver struct {
	resolver *discovery.Resolver
	client   *http.Client
}

// NewReadinessProfileResolver constructs the production resolver for the
// Experience Manager-owned profile RPC.
func NewReadinessProfileResolver() ReadinessProfileResolver {
	return &experienceReadinessResolver{resolver: discovery.NewResolver(discovery.ResolverConfig{}), client: &http.Client{Timeout: 5 * time.Second}}
}

func (r *experienceReadinessResolver) ResolveReadinessWaits(ctx context.Context, scenario, route string) (ReadinessResolution, error) {
	base, err := r.resolver.ResolveScenarioURLDefault(ctx, "experience-manager")
	if err != nil {
		return ReadinessResolution{}, err
	}
	resp, err := contractconnect.NewContractServiceClient(r.client, base).GetReadinessProfile(ctx, connect.NewRequest(&contractv1.GetReadinessProfileRequest{Scenario: scenario}))
	if err != nil {
		return ReadinessResolution{}, err
	}
	var profile readinessProfile
	if err := json.Unmarshal([]byte(resp.Msg.GetProfileJson()), &profile); err != nil {
		return ReadinessResolution{}, fmt.Errorf("decode readiness profile: %w", err)
	}
	resolution := ReadinessResolution{ProfileVersion: resp.Msg.GetProfileVersion(), Route: route}
	for _, page := range profile.Pages {
		if !containsRoute(page.Routes, route) {
			continue
		}
		resolution.RouteMatched = true
		for _, region := range page.Regions {
			if !region.Required {
				continue
			}
			selector := strings.TrimSpace(region.Binding.Selector)
			if selector == "" && strings.TrimSpace(region.Binding.TestID) != "" {
				selector = `[data-testid="` + strings.TrimSpace(region.Binding.TestID) + `"]`
			}
			if selector != "" {
				selector = terminalReadinessSelector(selector, region.Lifecycle.Kind, region.Lifecycle.States)
				resolution.Waits = append(resolution.Waits, &actionsv1.WaitParams{WaitFor: &actionsv1.WaitParams_Selector{Selector: selector}})
				resolution.RequiredSurfaceIDs = append(resolution.RequiredSurfaceIDs, region.ID)
			}
		}
		return resolution, nil
	}
	return resolution, nil
}

type readinessProfile struct {
	Pages []struct {
		Routes  []string `json:"routes"`
		Regions []struct {
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
		} `json:"regions"`
	} `json:"pages"`
}

// terminalReadinessSelector turns an async surface binding into a selector for
// one of its declared terminal states. This prevents a surface's initial DOM
// mount (usually loading) from being mistaken for functional readiness.
func terminalReadinessSelector(binding, kind string, states []string) string {
	if kind != "async" {
		return binding
	}
	var selectors []string
	for _, state := range states {
		state = strings.TrimSpace(state)
		if state == "" || state == "loading" || state == "static" {
			continue
		}
		selectors = append(selectors, binding+`[data-experience-state="`+state+`"]`)
	}
	if len(selectors) == 0 {
		return binding
	}
	return strings.Join(selectors, ", ")
}

func containsRoute(routes []string, route string) bool {
	path := strings.SplitN(strings.SplitN(route, "#", 2)[0], "?", 2)[0]
	for _, candidate := range routes {
		if routeTemplateMatches(candidate, path) {
			return true
		}
	}
	return false
}

// routeTemplateMatches recognizes the same colon-parameter shape authored in
// experience page routes. It deliberately matches whole path segments only:
// /assets/:id accepts /assets/Button but not /assets/Button/history, keeping
// undeclared subroutes on BAS's generic fallback path.
func routeTemplateMatches(template, path string) bool {
	if template == path {
		return true
	}
	templateParts := strings.Split(strings.Trim(template, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	if len(templateParts) != len(pathParts) {
		return false
	}
	for index, part := range templateParts {
		if strings.HasPrefix(part, ":") {
			if strings.TrimPrefix(part, ":") == "" || pathParts[index] == "" {
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
