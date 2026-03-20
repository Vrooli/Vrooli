package service

import (
	"context"
	"fmt"

	"tunnel-manager/adapter"
	"tunnel-manager/domain"
)

// ModeSwitcher handles switching between local and remote management modes. [REQ:CFAPI-006]
type ModeSwitcher struct {
	localCfg    *LocalConfigManager
	cfClient    *adapter.CFClient
	routeLister RouteLister
}

// NewModeSwitcher creates a mode switcher with local and remote managers.
func NewModeSwitcher(localCfg *LocalConfigManager, cfClient *adapter.CFClient, routeLister RouteLister) *ModeSwitcher {
	return &ModeSwitcher{
		localCfg:    localCfg,
		cfClient:    cfClient,
		routeLister: routeLister,
	}
}

// SwitchTo transitions to the target management mode. [REQ:CFAPI-006]
func (ms *ModeSwitcher) SwitchTo(ctx context.Context, target domain.ManagementMode) error {
	switch target {
	case domain.ModeRemote:
		return ms.switchToRemote(ctx)
	case domain.ModeLocal:
		return ms.switchToLocal(ctx)
	default:
		return domain.ErrValidation(fmt.Sprintf("unknown mode: %s", target))
	}
}

// switchToRemote reads local config, pushes to Cloudflare API. [REQ:CFAPI-006]
func (ms *ModeSwitcher) switchToRemote(ctx context.Context) error {
	if ms.cfClient == nil {
		return domain.ErrUnavailable("CF_API_TOKEN, CF_ACCOUNT_ID, and CF_TUNNEL_ID are required for remote mode")
	}

	// Get current routes from DB
	routes, err := ms.routeLister.List()
	if err != nil {
		return domain.ErrInternal("list routes for remote switch", err)
	}

	// Convert to CF rules and push
	rules := adapter.RoutesToCFRules(routes)
	if err := ms.cfClient.PushConfig(ctx, rules); err != nil {
		return domain.ErrInternal("push config to cloudflare", err)
	}

	return nil
}

// switchToLocal reads API config, writes to local config.yml. [REQ:CFAPI-006]
func (ms *ModeSwitcher) switchToLocal(ctx context.Context) error {
	if ms.localCfg == nil {
		return domain.ErrUnavailable("local config manager is required for local mode")
	}

	// Get routes from DB
	routes, err := ms.routeLister.List()
	if err != nil {
		return domain.ErrInternal("list routes for local switch", err)
	}

	// Try to read existing config to preserve non-ingress settings
	existing, _ := ms.localCfg.Parse()

	// Generate new config
	cfg := ms.localCfg.GenerateFromRoutes(routes, existing)

	// Write with backup
	if err := ms.localCfg.WriteWithBackup(cfg, 5); err != nil {
		return domain.ErrInternal("write config", err)
	}

	// Restart cloudflared
	if err := ms.localCfg.RestartCloudflared(ctx); err != nil {
		return domain.ErrInternal("restart after config change", err)
	}

	return nil
}
