package lifecycle

import (
	"context"
	"errors"
	"fmt"

	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func (r *Runner) acquireDependencyDemandLease(ctx context.Context, consumer, dependency scenario.Scenario) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	store, err := r.runtimeDeps().runtimeRegistry(ctx, r.Home)
	if err != nil {
		return "", fmt.Errorf("open runtime registry for dependency demand: %w", err)
	}
	defer store.Close()
	consumerID := scenarioruntime.DependencyConsumerID(consumer.Slug, consumer.Variant)
	leaseID := scenarioruntime.DependencyLeaseID(consumerID, dependency.Slug, dependency.Variant)
	_, err = store.AcquireDemandLease(ctx, scenarioruntime.DemandLease{
		LeaseID:    leaseID,
		Scenario:   dependency.Slug,
		Variant:    dependency.Variant,
		ConsumerID: consumerID,
		Kind:       scenarioruntime.DemandLeaseDependency,
		RequestID:  "dependency-hold",
	}, scenarioruntime.DefaultDemandLeaseTTL)
	if err != nil {
		return "", fmt.Errorf("acquire dependency demand for %s: %w", dependency.Slug, err)
	}
	return leaseID, nil
}

func (r *Runner) releaseDemandLease(ctx context.Context, leaseID, reason string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	store, err := r.runtimeDeps().runtimeRegistry(ctx, r.Home)
	if err != nil {
		return err
	}
	defer store.Close()
	_, err = store.ReleaseDemandLease(ctx, leaseID, reason)
	if errors.Is(err, scenarioruntime.ErrDemandLeaseExpired) || errors.Is(err, scenarioruntime.ErrNotFound) {
		return nil
	}
	return err
}

func (r *Runner) releaseDependencyLeasesForConsumer(ctx context.Context, scenarioName, variant, reason string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	store, err := r.runtimeDeps().runtimeRegistry(ctx, r.Home)
	if err != nil {
		return err
	}
	defer store.Close()
	consumerID := scenarioruntime.DependencyConsumerID(scenarioName, variant)
	leases, err := store.ListDemandLeases(ctx, scenarioruntime.DemandLeaseFilter{ConsumerID: consumerID, Statuses: []string{scenarioruntime.DemandLeaseActive}})
	if err != nil {
		return err
	}
	var releaseErr error
	for _, lease := range leases {
		if lease.Kind != scenarioruntime.DemandLeaseDependency {
			continue
		}
		if _, err := store.ReleaseDemandLease(ctx, lease.LeaseID, reason); err != nil && !errors.Is(err, scenarioruntime.ErrDemandLeaseExpired) && !errors.Is(err, scenarioruntime.ErrNotFound) {
			releaseErr = errors.Join(releaseErr, err)
		}
	}
	return releaseErr
}

func (r *Runner) releaseStartSessionDemand(ctx context.Context, session *startSession) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if session == nil {
		return nil
	}
	var releaseErr error
	for _, leaseID := range session.dependencyLeases() {
		if err := r.releaseDemandLease(ctx, leaseID, "scenario start failed"); err != nil {
			releaseErr = errors.Join(releaseErr, err)
		}
	}
	return releaseErr
}

func (r *Runner) releaseNewStartSessionDemand(ctx context.Context, session *startSession, before map[string]struct{}, reason string) error {
	if session == nil {
		return nil
	}
	newIDs := make([]string, 0)
	for _, leaseID := range session.dependencyLeases() {
		if _, existed := before[leaseID]; existed {
			continue
		}
		newIDs = append(newIDs, leaseID)
	}
	var releaseErr error
	for _, leaseID := range newIDs {
		if err := r.releaseDemandLease(ctx, leaseID, reason); err != nil {
			releaseErr = errors.Join(releaseErr, err)
		}
	}
	session.forgetDependencyLeases(newIDs)
	return releaseErr
}
