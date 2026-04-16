package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/discovery"

	rulespkg "scenario-auditor/rules"
)

type appMonitorInteropProvider struct {
	client     *http.Client
	ruleLookup map[string]rulespkg.Rule
}

func newAppMonitorInteropProvider() externalRuleProvider {
	provider := &appMonitorInteropProvider{
		client:     &http.Client{Timeout: 30 * time.Second},
		ruleLookup: make(map[string]rulespkg.Rule),
	}
	for _, rule := range provider.Rules() {
		provider.ruleLookup[rule.ID] = rule
	}
	return provider
}

func (p *appMonitorInteropProvider) ID() string {
	return "app-monitor"
}

func (p *appMonitorInteropProvider) Name() string {
	return "App Monitor Interop"
}

func (p *appMonitorInteropProvider) Rules() []rulespkg.Rule {
	return []rulespkg.Rule{
		{ID: "interop_api_base_dep", Name: "API base dependency", Description: "Ensure the UI declares the API base interop dependency", Category: "interop", Severity: "critical", Enabled: true, Standard: "ui-interop-deps"},
		{ID: "interop_iframe_bridge_dep", Name: "Iframe bridge dependency", Description: "Ensure the UI declares the iframe bridge interop dependency", Category: "interop", Severity: "critical", Enabled: true, Standard: "ui-interop-deps"},
		{ID: "interop_hardcoded_localhost", Name: "No hardcoded localhost", Description: "Ensure the UI does not hardcode localhost URLs for API calls", Category: "interop", Severity: "high", Enabled: true, Standard: "ui-interop-api"},
		{ID: "interop_relative_base", Name: "Relative Vite base", Description: "Ensure the Vite base config uses a relative path for proxy compatibility", Category: "interop", Severity: "critical", Enabled: true, Standard: "ui-interop-assets"},
		{ID: "interop_router_basename", Name: "Proxy-aware router", Description: "Ensure the UI router uses a proxy-aware basename", Category: "interop", Severity: "high", Enabled: true, Standard: "ui-interop-routing"},
		{ID: "interop_no_custom_server", Name: "Standard scenario server", Description: "Ensure the scenario uses the standard server instead of a custom one", Category: "interop", Severity: "medium", Enabled: true, Standard: "ui-interop-server"},
		{ID: "interop_bridge_init", Name: "Bridge initialization", Description: "Ensure the iframe bridge is initialized at app startup", Category: "interop", Severity: "critical", Enabled: true, Standard: "ui-interop-bridge"},
		{ID: "interop_resolve_api_base_single", Name: "Single API base resolution", Description: "Ensure the API base URL is resolved in at most 2 production files", Category: "interop", Severity: "high", Enabled: true, Standard: "ui-interop-api"},
		{ID: "interop_shortcut_relay", Name: "Shortcut iframe relay", Description: "Ensure keyboard shortcuts are relayed through the iframe bridge", Category: "interop", Severity: "medium", Enabled: true, Standard: "ui-interop-keyboard"},
		{ID: "interop_no_scattered_keydown", Name: "Centralized keyboard handling", Description: "Ensure keydown listeners are centralized rather than scattered", Category: "interop", Severity: "medium", Enabled: true, Standard: "ui-interop-keyboard"},
		{ID: "interop_bridge_app_id", Name: "Bridge appId parameter", Description: "Ensure the bridge initialization includes the appId parameter", Category: "interop", Severity: "medium", Enabled: true, Standard: "ui-interop-bridge"},
		{ID: "interop_protective_comments", Name: "Protective comments", Description: "Ensure interop-critical code has protective comments to prevent accidental removal", Category: "interop", Severity: "low", Enabled: true, Standard: "ui-interop-docs"},
		{ID: "interop_iframe_guard", Name: "Iframe guard", Description: "Ensure bridge initialization is guarded with window.parent !== window", Category: "interop", Severity: "high", Enabled: true, Standard: "ui-interop-bridge"},
		{ID: "interop_capture_enabled", Name: "Capture settings enabled", Description: "Ensure captureLogs and captureNetwork are not disabled in bridge init", Category: "interop", Severity: "medium", Enabled: true, Standard: "ui-interop-bridge"},
		{ID: "interop_proxy_base_preserved", Name: "Proxy base preservation", Description: "Ensure resolveApiBase output is not rebuilt with window.location.origin", Category: "interop", Severity: "high", Enabled: true, Standard: "ui-interop-api"},
		{ID: "interop_secure_tunnel", Name: "Secure UI tunnel", Description: "Ensure custom server files route API calls through proxyToApi", Category: "interop", Severity: "high", Enabled: true, Standard: "ui-interop-server"},
		{ID: "interop_standard_server", Name: "Standard server functions", Description: "Ensure server files use startScenarioServer or createScenarioServer from @vrooli/api-base/server", Category: "interop", Severity: "medium", Enabled: true, Standard: "ui-interop-server"},
	}
}

type interopStandardsResponse struct {
	EntityName string                      `json:"entity_name"`
	Violations []interopStandardsViolation `json:"violations"`
}

type interopStandardsViolation struct {
	RuleID         string         `json:"rule_id"`
	Severity       string         `json:"severity"`
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	FilePath       string         `json:"file_path"`
	Recommendation string         `json:"recommendation"`
	Metadata       map[string]any `json:"metadata"`
}

func (p *appMonitorInteropProvider) Run(ctx context.Context, scenarioName string, ruleIDs []string) ([]StandardsViolation, error) {
	cleaned := strings.TrimSpace(scenarioName)
	if cleaned == "" {
		return nil, nil
	}

	requested := make(map[string]struct{})
	for _, id := range ruleIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		requested[id] = struct{}{}
	}

	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, "app-monitor")
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/api/v1/quality/scenario/%s/standards", baseURL, url.PathEscape(cleaned))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		payload, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("app-monitor responded with %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}

	var parsed interopStandardsResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	var violations []StandardsViolation
	for _, violation := range parsed.Violations {
		if len(requested) > 0 {
			if _, ok := requested[violation.RuleID]; !ok {
				continue
			}
		}
		ruleMeta := p.ruleLookup[violation.RuleID]
		violations = append(violations, StandardsViolation{
			ID:             uuid.New().String(),
			ScenarioName:   cleaned,
			Type:           violation.RuleID,
			Severity:       strings.ToLower(violation.Severity),
			Title:          violation.Title,
			Description:    violation.Description,
			FilePath:       violation.FilePath,
			Recommendation: violation.Recommendation,
			Standard:       ruleMeta.Standard,
			DiscoveredAt:   time.Now().Format(time.RFC3339),
			Metadata:       violation.Metadata,
		})
	}

	return violations, nil
}
