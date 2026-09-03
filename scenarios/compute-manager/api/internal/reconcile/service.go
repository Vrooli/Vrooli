// Package reconcile compares provider inventory with durable local records.
// A sweep reports divergence and never destroys a resource.
package reconcile

import (
	"context"
	"fmt"

	"compute-manager/internal/provider"
)

type Kind string

const (
	ProviderOnly Kind = "provider_only"
	LocalOnly    Kind = "local_only"
	StateDrift   Kind = "state_drift"
)

type Local struct {
	ProviderID string
	State      string
}

type Finding struct {
	Kind       Kind
	ProviderID string
	Detail     string
}

type Service struct {
	Provider provider.Provider
}

func (s Service) Sweep(ctx context.Context, local []Local) ([]Finding, error) {
	if s.Provider == nil {
		return nil, fmt.Errorf("reconciliation provider is unavailable")
	}
	observed, err := s.Provider.List(ctx)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]provider.Instance, len(observed))
	for _, item := range observed {
		byID[item.ID] = item
	}
	localByID := make(map[string]Local, len(local))
	findings := make([]Finding, 0)
	for _, item := range local {
		localByID[item.ProviderID] = item
		observedItem, ok := byID[item.ProviderID]
		if !ok {
			findings = append(findings, Finding{Kind: LocalOnly, ProviderID: item.ProviderID, Detail: "local instance is absent at provider"})
			continue
		}
		if item.State != "" && item.State != "running" {
			findings = append(findings, Finding{Kind: StateDrift, ProviderID: item.ProviderID, Detail: fmt.Sprintf("local state %s differs from provider instance %s", item.State, observedItem.ID)})
		}
	}
	for _, item := range observed {
		if _, ok := localByID[item.ID]; !ok {
			findings = append(findings, Finding{Kind: ProviderOnly, ProviderID: item.ID, Detail: "provider instance is absent from local records"})
		}
	}
	return findings, nil
}
