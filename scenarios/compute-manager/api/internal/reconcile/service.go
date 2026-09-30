// Package reconcile compares provider inventory with durable local records.
// A sweep reports divergence and never destroys a resource.
package reconcile

import (
	"context"
	"fmt"
	"time"

	"compute-manager/internal/provider"
)

type Kind string

const (
	ProviderOnly Kind = "provider_only"
	LocalOnly    Kind = "local_only"
	StateDrift   Kind = "state_drift"
	CostDrift    Kind = "cost_divergence"
)

type Local struct {
	InstanceID    string
	ProviderID    string
	State         string
	ReservationID string
	CreatedAt     time.Time
}

type Finding struct {
	Kind       Kind
	ProviderID string
	Detail     string
}

type CostObservation struct {
	ProviderID, InstanceID string
	MeteredMinutes         int64
	ProviderMinutes        int64
	DeltaMinutes           int64
	Alarm                  bool
}

// CompareCost compares locally-derived lifecycle usage with provider
// accounting observations. It only reports; it never alters usage or money.
func CompareCost(metered map[string]int64, statements []provider.BillingStatement, threshold int64) []CostObservation {
	if threshold < 0 {
		threshold = 0
	}
	out := make([]CostObservation, 0, len(statements))
	for _, statement := range statements {
		local := metered[statement.ProviderInstanceID]
		delta := local - statement.Minutes
		if delta < 0 {
			delta = -delta
		}
		out = append(out, CostObservation{ProviderID: statement.Provider, InstanceID: statement.ProviderInstanceID, MeteredMinutes: local, ProviderMinutes: statement.Minutes, DeltaMinutes: delta, Alarm: delta > threshold})
	}
	return out
}

type Service struct {
	Provider provider.Provider
	// Settle closes usage for a locally-recorded instance that the provider no
	// longer has. It must not destroy provider resources.
	Settle func(context.Context, Local) error
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
			if s.Settle != nil {
				if err := s.Settle(ctx, item); err != nil {
					return nil, fmt.Errorf("settle local-only instance %s: %w", item.ProviderID, err)
				}
			}
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
