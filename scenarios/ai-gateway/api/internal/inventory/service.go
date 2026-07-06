package inventory

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"ai-gateway/internal/providers"
)

type ProviderAdapter interface {
	ListRoles(ctx context.Context) (providers.Inventory, error)
	Smoke(ctx context.Context) providers.SmokeResult
}

type Service struct {
	adapters map[string]ProviderAdapter
	order    []string
}

func NewService(adapters []providers.Adapter) *Service {
	wrapped := make([]ProviderAdapter, 0, len(adapters))
	for _, adapter := range adapters {
		wrapped = append(wrapped, adapter)
	}
	return NewServiceWithAdapters(wrapped)
}

func NewServiceWithAdapters(adapters []ProviderAdapter) *Service {
	svc := &Service{adapters: map[string]ProviderAdapter{}}
	for _, adapter := range adapters {
		name := providerName(adapter)
		if name == "" {
			continue
		}
		svc.adapters[name] = adapter
		svc.order = append(svc.order, name)
	}
	sort.Strings(svc.order)
	return svc
}

func (s *Service) ListProviderRoles(ctx context.Context, provider string) ([]providers.Role, []string) {
	var roles []providers.Role
	var warnings []string
	selected := s.selectedProviders(provider)
	if len(selected) == 0 && strings.TrimSpace(provider) != "" {
		return nil, []string{fmt.Sprintf("%s: provider is not configured", strings.TrimSpace(provider))}
	}
	for _, name := range selected {
		inventory, err := s.adapters[name].ListRoles(ctx)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		roles = append(roles, inventory.Roles...)
		warnings = append(warnings, inventory.Warnings...)
	}
	sort.Slice(roles, func(i, j int) bool {
		if roles[i].Provider == roles[j].Provider {
			return roles[i].Role < roles[j].Role
		}
		return roles[i].Provider < roles[j].Provider
	})
	return roles, warnings
}

func (s *Service) SmokeProvider(ctx context.Context, provider string) providers.SmokeResult {
	selected := s.selectedProviders(provider)
	if len(selected) == 0 {
		return providers.SmokeResult{Provider: strings.TrimSpace(provider), Status: "unavailable", Code: "unknown_provider", Message: "provider is not configured", ExitCode: -1}
	}
	if len(selected) > 1 {
		return providers.SmokeResult{Provider: "all", Status: "available", Code: "ok", Message: "use a specific provider for detailed smoke status"}
	}
	return s.adapters[selected[0]].Smoke(ctx)
}

func (s *Service) selectedProviders(provider string) []string {
	provider = strings.TrimSpace(strings.ToLower(provider))
	if provider == "" || provider == "all" {
		return append([]string{}, s.order...)
	}
	if _, ok := s.adapters[provider]; !ok {
		return nil
	}
	return []string{provider}
}

func providerName(adapter ProviderAdapter) string {
	if concrete, ok := adapter.(providers.Adapter); ok {
		return strings.TrimSpace(strings.ToLower(concrete.Provider))
	}
	if concrete, ok := adapter.(*providers.Adapter); ok {
		return strings.TrimSpace(strings.ToLower(concrete.Provider))
	}
	return ""
}
