package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"network-manager/internal/resolver"
)

const (
	adGuardFilteringStatusEndpoint = "/control/filtering/status"
	adGuardFilteringRulesEndpoint  = "/control/filtering/set_rules"
	adGuardProtectionEndpoint      = "/control/protection"
	adGuardStatusEndpoint          = "/control/status"
)

type AdGuardResolverPolicyAdapter struct {
	Backends resolver.Repository
	Secrets  resolver.SecretResolver
	HTTP     http.RoundTripper
	Timeout  time.Duration
}

var _ ResolverPolicyAdapter = AdGuardResolverPolicyAdapter{}

func NewAdGuardResolverPolicyAdapter(backends resolver.Repository, secrets resolver.SecretResolver) AdGuardResolverPolicyAdapter {
	if secrets == nil {
		secrets = resolver.NewCredentialAuthorityResolver()
	}
	return AdGuardResolverPolicyAdapter{Backends: backends, Secrets: secrets, Timeout: 10 * time.Second}
}

func (a AdGuardResolverPolicyAdapter) Preview(ctx context.Context, change Change) (AdapterPlan, error) {
	if err := a.validateSupported(change); err != nil {
		return AdapterPlan{
			Effects: []string{
				err.Error(),
				"No AdGuard Home policy mutation will be attempted for this preview.",
			},
			RollbackSupported: false,
		}, nil
	}
	if _, err := a.backend(ctx); err != nil {
		return AdapterPlan{
			Effects: []string{
				"AdGuard Home resolver backend is not configured with a secret reference.",
				"Applying this plan will return unsupported until resolver configuration is complete.",
			},
			RollbackSupported: false,
		}, nil
	}
	return AdapterPlan{
		Effects:           previewEffects(change),
		RollbackSupported: true,
	}, nil
}

func (a AdGuardResolverPolicyAdapter) Apply(ctx context.Context, change Change) (AdapterApplyResult, error) {
	if err := a.validateSupported(change); err != nil {
		return AdapterApplyResult{}, fmt.Errorf("%w: %s", ErrUnsupported, err.Error())
	}
	client, err := a.client(ctx)
	if err != nil {
		return AdapterApplyResult{}, fmt.Errorf("%w: %v", ErrUnsupported, err)
	}
	switch change.Action {
	case "allowlist", "denylist", "blocklist":
		return a.applyRules(ctx, client, change)
	case "pause_filtering", "resume_filtering":
		return a.applyProtection(ctx, client, change)
	default:
		return AdapterApplyResult{}, ErrUnsupported
	}
}

func (a AdGuardResolverPolicyAdapter) Rollback(ctx context.Context, change Change) (AdapterRollbackResult, error) {
	client, err := a.client(ctx)
	if err != nil {
		return AdapterRollbackResult{}, err
	}
	var handle adGuardRollbackHandle
	if err := json.Unmarshal([]byte(change.RollbackHandle), &handle); err != nil {
		return AdapterRollbackResult{}, fmt.Errorf("decode AdGuard rollback handle: %w", err)
	}
	switch handle.Kind {
	case "filtering_rules":
		if code, err := client.setRules(ctx, handle.UserRules); err != nil {
			return AdapterRollbackResult{}, fmt.Errorf("restore AdGuard filtering rules: %w", err)
		} else if code < 200 || code >= 300 {
			return AdapterRollbackResult{}, fmt.Errorf("restore AdGuard filtering rules returned HTTP %d", code)
		}
		return AdapterRollbackResult{Effects: []string{"Restored previous AdGuard Home user-defined filtering rules."}}, nil
	case "protection":
		if handle.ProtectionEnabled == nil {
			return AdapterRollbackResult{}, fmt.Errorf("rollback handle is missing previous protection state")
		}
		if code, err := client.setProtection(ctx, *handle.ProtectionEnabled, 0); err != nil {
			return AdapterRollbackResult{}, fmt.Errorf("restore AdGuard protection state: %w", err)
		} else if code < 200 || code >= 300 {
			return AdapterRollbackResult{}, fmt.Errorf("restore AdGuard protection state returned HTTP %d", code)
		}
		return AdapterRollbackResult{Effects: []string{"Restored previous AdGuard Home protection state."}}, nil
	default:
		return AdapterRollbackResult{}, fmt.Errorf("unsupported AdGuard rollback handle kind %q", handle.Kind)
	}
}

func (a AdGuardResolverPolicyAdapter) applyRules(ctx context.Context, client *adGuardPolicyClient, change Change) (AdapterApplyResult, error) {
	current, code, err := client.filteringStatus(ctx)
	if err != nil {
		return AdapterApplyResult{}, fmt.Errorf("read AdGuard filtering rules: %w", err)
	}
	if code < 200 || code >= 300 {
		return AdapterApplyResult{}, fmt.Errorf("read AdGuard filtering rules returned HTTP %d", code)
	}
	added := rulesForChange(change)
	if len(added) == 0 {
		return AdapterApplyResult{}, fmt.Errorf("%w: no valid filtering rules were supplied", ErrUnsupported)
	}
	next := appendUnique(current.UserRules, added...)
	if code, err := client.setRules(ctx, next); err != nil {
		return AdapterApplyResult{}, fmt.Errorf("apply AdGuard filtering rules: %w", err)
	} else if code < 200 || code >= 300 {
		return AdapterApplyResult{}, fmt.Errorf("apply AdGuard filtering rules returned HTTP %d", code)
	}
	handle := adGuardRollbackHandle{Kind: "filtering_rules", UserRules: current.UserRules}
	encoded, err := json.Marshal(handle)
	if err != nil {
		return AdapterApplyResult{}, err
	}
	return AdapterApplyResult{
		Effects: []string{
			fmt.Sprintf("Applied %d AdGuard Home user-defined filtering rule(s).", len(added)),
			"Rollback handle captured the previous user-defined filtering rules only.",
		},
		RollbackSupported: true,
		RollbackHandle:    string(encoded),
	}, nil
}

func (a AdGuardResolverPolicyAdapter) applyProtection(ctx context.Context, client *adGuardPolicyClient, change Change) (AdapterApplyResult, error) {
	status, code, err := client.status(ctx)
	if err != nil {
		return AdapterApplyResult{}, fmt.Errorf("read AdGuard protection state: %w", err)
	}
	if code < 200 || code >= 300 {
		return AdapterApplyResult{}, fmt.Errorf("read AdGuard protection state returned HTTP %d", code)
	}
	previous := protectionEnabled(status)
	enabled := change.Action == "resume_filtering"
	duration := pauseDuration(change.Values)
	if code, err := client.setProtection(ctx, enabled, duration); err != nil {
		return AdapterApplyResult{}, fmt.Errorf("update AdGuard protection state: %w", err)
	} else if code < 200 || code >= 300 {
		return AdapterApplyResult{}, fmt.Errorf("update AdGuard protection state returned HTTP %d", code)
	}
	handle := adGuardRollbackHandle{Kind: "protection", ProtectionEnabled: &previous}
	encoded, err := json.Marshal(handle)
	if err != nil {
		return AdapterApplyResult{}, err
	}
	effect := "Resumed AdGuard Home protection."
	if !enabled {
		effect = "Paused AdGuard Home protection."
		if duration > 0 {
			effect = fmt.Sprintf("Paused AdGuard Home protection for %s.", duration)
		}
	}
	return AdapterApplyResult{
		Effects:           []string{effect, "Rollback handle captured the previous protection state."},
		RollbackSupported: true,
		RollbackHandle:    string(encoded),
	}, nil
}

func (a AdGuardResolverPolicyAdapter) validateSupported(change Change) error {
	if !isGlobalTarget(change.Target) {
		return fmt.Errorf("AdGuard Home policy adapter currently supports only network/global targets; %s needs client mapping first", change.Target)
	}
	switch change.Action {
	case "allowlist", "denylist", "blocklist", "pause_filtering", "resume_filtering":
		return nil
	default:
		return fmt.Errorf("AdGuard Home policy adapter does not support action %q", change.Action)
	}
}

func (a AdGuardResolverPolicyAdapter) backend(ctx context.Context) (resolver.BackendConfig, error) {
	if a.Backends == nil {
		return resolver.BackendConfig{}, fmt.Errorf("resolver backend repository is unavailable")
	}
	cfg, err := a.Backends.GetBackend(ctx, resolver.AdGuardHomeBackend)
	if err != nil {
		return resolver.BackendConfig{}, err
	}
	if strings.TrimSpace(cfg.BaseURL) == "" || strings.TrimSpace(cfg.CredentialRef) == "" {
		return resolver.BackendConfig{}, fmt.Errorf("AdGuard Home backend is missing base_url or credential_ref")
	}
	return cfg, nil
}

func (a AdGuardResolverPolicyAdapter) client(ctx context.Context) (*adGuardPolicyClient, error) {
	cfg, err := a.backend(ctx)
	if err != nil {
		return nil, err
	}
	secrets := a.Secrets
	if secrets == nil {
		secrets = resolver.NewCredentialAuthorityResolver()
	}
	creds, err := secrets.ResolveAdGuardCredentials(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("resolve AdGuard Home credentials: %w", err)
	}
	return &adGuardPolicyClient{resolver.NewAdGuardHomeClient(cfg.BaseURL, creds, a.HTTP, a.Timeout)}, nil
}

type adGuardPolicyClient struct{ *resolver.AdGuardHomeClient }

type adGuardFilteringStatus struct {
	UserRules []string `json:"user_rules"`
}

type adGuardRulesUpdate struct {
	Rules []string `json:"rules"`
}

type adGuardProtectionUpdate struct {
	Enabled  bool   `json:"enabled"`
	Duration uint64 `json:"duration,omitempty"`
}

type adGuardStatus struct {
	ProtectionStatus *bool `json:"protection_status"`
	Protection       *bool `json:"protection"`
	Running          *bool `json:"running"`
}

type adGuardRollbackHandle struct {
	Kind              string   `json:"kind"`
	UserRules         []string `json:"user_rules,omitempty"`
	ProtectionEnabled *bool    `json:"protection_enabled,omitempty"`
}

func (c *adGuardPolicyClient) filteringStatus(ctx context.Context) (adGuardFilteringStatus, int, error) {
	return resolver.GetAdGuardJSON[adGuardFilteringStatus](ctx, c.AdGuardHomeClient, adGuardFilteringStatusEndpoint)
}

func (c *adGuardPolicyClient) status(ctx context.Context) (adGuardStatus, int, error) {
	return resolver.GetAdGuardJSON[adGuardStatus](ctx, c.AdGuardHomeClient, adGuardStatusEndpoint)
}

func (c *adGuardPolicyClient) setRules(ctx context.Context, rules []string) (int, error) {
	return c.requestStatus(ctx, http.MethodPost, adGuardFilteringRulesEndpoint, adGuardRulesUpdate{Rules: rules})
}

func (c *adGuardPolicyClient) setProtection(ctx context.Context, enabled bool, duration time.Duration) (int, error) {
	var millis uint64
	if duration > 0 {
		millis = uint64(duration / time.Millisecond)
	}
	return c.requestStatus(ctx, http.MethodPost, adGuardProtectionEndpoint, adGuardProtectionUpdate{Enabled: enabled, Duration: millis})
}

func (c *adGuardPolicyClient) requestStatus(ctx context.Context, method, endpoint string, body any) (int, error) {
	resp, err := c.Do(ctx, method, endpoint, body)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

func previewEffects(change Change) []string {
	switch change.Action {
	case "allowlist", "denylist", "blocklist":
		return []string{
			fmt.Sprintf("Preview will append %d AdGuard Home user-defined filtering rule(s).", len(rulesForChange(change))),
			"Rollback will restore the previous user-defined filtering rules.",
		}
	case "pause_filtering":
		return []string{
			"Preview will pause global AdGuard Home protection.",
			"Rollback will restore the previous protection state.",
		}
	case "resume_filtering":
		return []string{
			"Preview will resume global AdGuard Home protection.",
			"Rollback will restore the previous protection state.",
		}
	default:
		return nil
	}
}

func rulesForChange(change Change) []string {
	rules := make([]string, 0, len(change.Values))
	for _, value := range change.Values {
		value = strings.TrimSpace(value)
		if value == "" || strings.HasPrefix(value, "duration=") {
			continue
		}
		switch change.Action {
		case "allowlist":
			rules = append(rules, allowRule(value))
		case "denylist", "blocklist":
			rules = append(rules, blockRule(value))
		}
	}
	return cleanPolicyStrings(rules)
}

func allowRule(value string) string {
	if strings.HasPrefix(value, "@@") {
		return value
	}
	return "@@" + blockRule(value)
}

func blockRule(value string) string {
	if strings.Contains(value, "||") || strings.Contains(value, "^") || strings.HasPrefix(value, "/") {
		return value
	}
	return "||" + strings.Trim(value, ".") + "^"
}

func appendUnique(existing []string, additions ...string) []string {
	out := cleanPolicyStrings(existing)
	seen := make(map[string]struct{}, len(out)+len(additions))
	for _, value := range out {
		seen[value] = struct{}{}
	}
	for _, value := range additions {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func cleanPolicyStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func pauseDuration(values []string) time.Duration {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if !strings.HasPrefix(value, "duration=") {
			continue
		}
		parsed, err := time.ParseDuration(strings.TrimPrefix(value, "duration="))
		if err == nil && parsed > 0 {
			return parsed
		}
	}
	return 0
}

func isGlobalTarget(target string) bool {
	target = strings.TrimSpace(strings.ToLower(target))
	return target == "" || target == "network" || target == "global" || target == "all"
}

func protectionEnabled(status adGuardStatus) bool {
	if status.ProtectionStatus != nil {
		return *status.ProtectionStatus
	}
	if status.Protection != nil {
		return *status.Protection
	}
	if status.Running != nil {
		return *status.Running
	}
	return false
}
