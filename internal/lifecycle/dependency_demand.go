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

func (r *Runner) snapshotDependencyDemandLeases(ctx context.Context, consumer scenario.Scenario, session *startSession) error {
	if session == nil {
		return nil
	}
	consumerID := scenarioruntime.DependencyConsumerID(consumer.Slug, consumer.Variant)
	store, err := r.runtimeDeps().runtimeRegistry(ctx, r.Home)
	if err != nil {
		return fmt.Errorf("open runtime registry to snapshot dependency demand for %s: %w", consumerID, err)
	}
	defer store.Close()
	instances, err := store.ListInstances(ctx, scenarioruntime.InstanceFilter{
		Scenario: consumer.Slug,
		Variant:  consumer.Variant,
		Statuses: scenarioruntime.ActiveInstanceStatuses(),
	})
	if err != nil {
		return fmt.Errorf("list active dependency consumer %s: %w", consumerID, err)
	}
	if len(instances) == 0 {
		session.rememberPriorDependencyLeases(consumerID, nil)
		return nil
	}
	leases, err := store.ListDemandLeases(ctx, scenarioruntime.DemandLeaseFilter{
		ConsumerID: consumerID,
		Statuses:   []string{scenarioruntime.DemandLeaseActive},
	})
	if err != nil {
		return fmt.Errorf("list prior dependency demand for %s: %w", consumerID, err)
	}
	leaseIDs := make([]string, 0, len(leases))
	for _, lease := range leases {
		if lease.Kind == scenarioruntime.DemandLeaseDependency {
			leaseIDs = append(leaseIDs, lease.LeaseID)
		}
	}
	session.rememberPriorDependencyLeases(consumerID, leaseIDs)
	return nil
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

func (r *Runner) releaseDependencyDemandIfCandidate(ctx context.Context, session *startSession, consumer scenario.Scenario, leaseID, reason string) error {
	consumerID := scenarioruntime.DependencyConsumerID(consumer.Slug, consumer.Variant)
	if session.shouldPreservePriorDependencyLease(consumerID, leaseID) {
		return nil
	}
	return r.releaseDemandLease(ctx, leaseID, reason)
}

func (r *Runner) releaseDependencyLeasesForConsumer(ctx context.Context, scenarioName, variant, reason string) error {
	return r.releaseDependencyLeasesForConsumerExcept(ctx, scenarioName, variant, reason, nil)
}

func (r *Runner) releaseDependencyLeasesForConsumerExcept(ctx context.Context, scenarioName, variant, reason string, keep map[string]struct{}) error {
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
		if _, retained := keep[lease.LeaseID]; retained {
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
		if session.shouldPreserveRecordedDependencyLease(leaseID) {
			continue
		}
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
	released := make([]string, 0, len(newIDs))
	for _, leaseID := range newIDs {
		if session.shouldPreserveRecordedDependencyLease(leaseID) {
			continue
		}
		if err := r.releaseDemandLease(ctx, leaseID, reason); err != nil {
			releaseErr = errors.Join(releaseErr, err)
		} else {
			released = append(released, leaseID)
		}
	}
	session.forgetDependencyLeases(released)
	return releaseErr
}
