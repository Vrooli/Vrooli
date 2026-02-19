package main

import (
	"context"
	"fmt"
)

// ManagementMode represents the tunnel management mode. [REQ:CFAPI-006]
type ManagementMode string

const (
	ModeLocal  ManagementMode = "local"
	ModeRemote ManagementMode = "remote"
)

// ModeSwitcher handles switching between local and remote management modes. [REQ:CFAPI-006]
type ModeSwitcher struct {
	localCfg   *LocalConfigManager
	cfClient   *CFClient
	routeSvc   *RouteService
}

// NewModeSwitcher creates a mode switcher with local and remote managers.
func NewModeSwitcher(localCfg *LocalConfigManager, cfClient *CFClient, routeSvc *RouteService) *ModeSwitcher {
	return &ModeSwitcher{
		localCfg: localCfg,
		cfClient: cfClient,
		routeSvc: routeSvc,
	}
}

// SwitchTo transitions to the target management mode. [REQ:CFAPI-006]
func (ms *ModeSwitcher) SwitchTo(ctx context.Context, target ManagementMode) error {
	switch target {
	case ModeRemote:
		return ms.switchToRemote(ctx)
	case ModeLocal:
		return ms.switchToLocal(ctx)
	default:
		return fmt.Errorf("unknown mode: %s", target)
	}
}

// switchToRemote reads local config, pushes to Cloudflare API. [REQ:CFAPI-006]
func (ms *ModeSwitcher) switchToRemote(ctx context.Context) error {
	if ms.cfClient == nil {
		return fmt.Errorf("CF_API_TOKEN, CF_ACCOUNT_ID, and CF_TUNNEL_ID are required for remote mode")
	}

	// Get current routes from DB
	routes, err := ms.routeSvc.List()
	if err != nil {
		return fmt.Errorf("list routes: %w", err)
	}

	// Convert to CF rules and push
	rules := RoutesToCFRules(routes)
	if err := ms.cfClient.PushConfig(ctx, rules); err != nil {
		return fmt.Errorf("push config to cloudflare: %w", err)
	}

	return nil
}

// switchToLocal reads API config, writes to local config.yml. [REQ:CFAPI-006]
func (ms *ModeSwitcher) switchToLocal(ctx context.Context) error {
	if ms.localCfg == nil {
		return fmt.Errorf("local config manager is required for local mode")
	}

	// Get routes from DB
	routes, err := ms.routeSvc.List()
	if err != nil {
		return fmt.Errorf("list routes: %w", err)
	}

	// Try to read existing config to preserve non-ingress settings
	existing, _ := ms.localCfg.Parse()

	// Generate new config
	cfg := ms.localCfg.GenerateFromRoutes(routes, existing)

	// Write with backup
	if err := ms.localCfg.WriteWithBackup(cfg, 5); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	// Restart cloudflared
	if err := ms.localCfg.RestartCloudflared(ctx); err != nil {
		return fmt.Errorf("restart after config change: %w", err)
	}

	return nil
}
