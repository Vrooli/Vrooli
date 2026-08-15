package routing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	internalregistry "search-hub/internal/registry"
)

type quorumLister struct {
	providers []*registryv1.ProviderDescriptor
}

func (l quorumLister) List(context.Context, internalregistry.ListFilter) ([]*registryv1.ProviderDescriptor, error) {
	return l.providers, nil
}

type fakeDemotionStore struct {
	states []ProviderDemotionState
	saved  []ProviderDemotionState
}

func (f *fakeDemotionStore) Load(context.Context) ([]ProviderDemotionState, error) {
	return append([]ProviderDemotionState(nil), f.states...), nil
}

func (f *fakeDemotionStore) Save(_ context.Context, state ProviderDemotionState) error {
	f.saved = append(f.saved, state)
	return nil
}

func TestProviderBreakersOpenImmediatelyForStoppedScenario(t *testing.T) {
	now := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	p := newProviderBreakers(rerankBreakerConfig{FailureThreshold: 3, Cooldown: time.Minute, MaxCooldown: 5 * time.Minute})
	p.openImmediately("leaf", now)

	ok, note := p.allow("leaf", now.Add(time.Second))
	require.False(t, ok)
	require.Contains(t, note, "circuit unavailable")
	require.True(t, isScenarioNotRunning(errors.New("scenario not running")))
}

func TestProviderBreakersIncreaseCooldownAfterRepeatedFailure(t *testing.T) {
	now := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	p := newProviderBreakers(rerankBreakerConfig{FailureThreshold: 1, Cooldown: time.Minute, MaxCooldown: 5 * time.Minute})
	p.record("leaf", true, now)
	b := p.breaker("leaf")
	b.recordFailure(now.Add(time.Minute))

	b.mu.Lock()
	cooldown := b.cooldown
	b.mu.Unlock()
	require.Equal(t, 2*time.Minute, cooldown)
}

func TestProviderBreakersDemoteZeroYieldAndRestoreExplicitHit(t *testing.T) {
	p := newProviderBreakers(rerankBreakerConfig{FailureThreshold: 3, Cooldown: time.Minute})
	for i := int64(0); i < defaultZeroYieldMinimumRoutes; i++ {
		p.recordResult("leaf", 0, false, false, time.Now())
	}
	demoted, routed, hits := p.demotion("leaf")
	require.True(t, demoted)
	require.Equal(t, defaultZeroYieldMinimumRoutes, routed)
	require.Zero(t, hits)
	require.False(t, p.eligibleAutomatic("leaf", time.Now()))

	p.recordResult("leaf", 1, false, true, time.Now())
	demoted, _, hits = p.demotion("leaf")
	require.False(t, demoted)
	require.EqualValues(t, 0, hits, "explicit recovery resets the evidence window")
	require.True(t, p.eligibleAutomatic("leaf", time.Now()))
}

func TestProviderBreakersTrackExplicitZeroYieldWithoutDemoting(t *testing.T) {
	p := newProviderBreakers(rerankBreakerConfig{FailureThreshold: 3, Cooldown: time.Minute})
	for i := int64(0); i < defaultZeroYieldMinimumRoutes; i++ {
		p.recordResult("leaf", 0, false, true, time.Now())
	}
	state := p.state("leaf")
	require.NotNil(t, state)
	require.EqualValues(t, defaultZeroYieldMinimumRoutes, state.emptyStreak)
	require.False(t, state.demoted, "explicit callers must never trigger automatic demotion")
}

func TestProviderBreakersFailureRecoveryProbeClaimsAndReleasesSlot(t *testing.T) {
	now := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	p := newProviderBreakers(rerankBreakerConfig{FailureThreshold: 1, Cooldown: time.Minute})
	p.openImmediately("leaf", now)
	require.False(t, p.beginFailureRecoveryProbe("leaf", now.Add(30*time.Second)))
	require.True(t, p.beginFailureRecoveryProbe("leaf", now.Add(time.Minute)))
	require.False(t, p.beginFailureRecoveryProbe("leaf", now.Add(time.Minute)))
	p.finishFailureRecoveryProbe("leaf", now.Add(time.Minute), "probe failed")
	state := p.state("leaf")
	require.NotNil(t, state)
	require.False(t, state.probation)
	open, note := p.status("leaf", now.Add(time.Minute+time.Second))
	require.True(t, open)
	require.Contains(t, note, "circuit open")
}

func TestProviderBreakersFailureRecoveryReleasePersists(t *testing.T) {
	now := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	store := &fakeDemotionStore{states: []ProviderDemotionState{{
		ProviderID: "leaf", Demoted: true, DecayDeadline: now,
	}}}
	p := newProviderBreakers(rerankBreakerConfig{FailureThreshold: 1, Cooldown: time.Minute}, store)
	p.restore(context.Background())
	// A transport probe can claim probation after the decay window; its
	// completion must persist the released slot, not only mutate memory.
	p.openImmediately("leaf", now)
	require.True(t, p.beginFailureRecoveryProbe("leaf", now.Add(time.Minute)))
	store.saved = nil
	p.finishFailureRecoveryProbe("leaf", now.Add(time.Minute), "provider unavailable")
	require.NotEmpty(t, store.saved)
	require.False(t, store.saved[len(store.saved)-1].Probation)
}

func TestProviderBreakersExpiredProbationIsCleared(t *testing.T) {
	now := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	store := &fakeDemotionStore{}
	p := newProviderBreakers(rerankBreakerConfig{FailureThreshold: 1, Cooldown: time.Minute}, store)
	p.stats["leaf"] = &providerYieldStats{demoted: true, probation: true, decayDeadline: now}
	require.True(t, p.clearExpiredProbation("leaf", now))
	require.False(t, p.state("leaf").probation)
	require.Contains(t, store.saved[len(store.saved)-1].Trigger, "expired recovery probation cleared")
}

func TestProviderBreakersIgnoreDegradedZeroHitForDemotion(t *testing.T) {
	p := newProviderBreakers(rerankBreakerConfig{FailureThreshold: 3, Cooldown: time.Minute})
	for i := int64(0); i < defaultZeroYieldMinimumRoutes+2; i++ {
		p.recordResult("leaf", 0, true, false, time.Now())
	}
	demoted, routed, hits := p.demotion("leaf")
	require.False(t, demoted)
	require.Zero(t, routed)
	require.Zero(t, hits)
}

func TestProviderBreakersTransportErrorsOpenCircuitWithoutDemotion(t *testing.T) {
	now := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	p := newProviderBreakers(rerankBreakerConfig{FailureThreshold: 3, Cooldown: time.Minute})
	for i := int64(0); i < defaultZeroYieldMinimumRoutes; i++ {
		p.recordResult("leaf", 0, true, false, now.Add(time.Duration(i)*time.Second))
		p.record("leaf", true, now.Add(time.Duration(i)*time.Second))
	}

	demoted, routed, hits := p.demotion("leaf")
	require.False(t, demoted)
	require.Zero(t, routed)
	require.Zero(t, hits)
	open, note := p.status("leaf", now.Add(5*time.Second))
	require.True(t, open)
	require.Contains(t, note, "circuit open")
}

func TestProviderBreakersProbeAfterDecayCanRecover(t *testing.T) {
	now := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	p := newProviderBreakers(rerankBreakerConfig{FailureThreshold: 3, Cooldown: time.Minute})
	for i := int64(0); i < defaultZeroYieldMinimumRoutes; i++ {
		p.recordResult("leaf", 0, false, false, now)
	}
	require.False(t, p.eligibleAutomatic("leaf", now.Add(time.Minute)))
	require.True(t, p.eligibleAutomatic("leaf", now.Add(defaultZeroYieldDemotionWindow+time.Second)))
	p.recordResult("leaf", 1, false, false, now.Add(defaultZeroYieldDemotionWindow+time.Second))
	demoted, _, _ := p.demotion("leaf")
	require.False(t, demoted)
}

func TestProviderBreakersDegradedProbationProbeReleasesAndRestartsDecay(t *testing.T) {
	now := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	p := newProviderBreakers(rerankBreakerConfig{FailureThreshold: 3, Cooldown: time.Minute})
	for i := int64(0); i < defaultZeroYieldMinimumRoutes; i++ {
		p.recordResult("leaf", 0, false, false, now)
	}
	probeAt := now.Add(defaultZeroYieldDemotionWindow + time.Second)
	require.True(t, p.eligibleAutomatic("leaf", probeAt))
	require.True(t, p.state("leaf").probation)

	p.recordResult("leaf", 0, true, false, probeAt)
	state := p.state("leaf")
	require.NotNil(t, state)
	require.True(t, state.demoted)
	require.False(t, state.probation)
	require.Equal(t, probeAt.Add(defaultZeroYieldDemotionWindow), state.decayDeadline)
	require.False(t, p.eligibleAutomatic("leaf", probeAt.Add(time.Second)))
	require.True(t, p.eligibleAutomatic("leaf", state.decayDeadline))
}

func TestProviderBreakersAutomaticProbationClaimPersists(t *testing.T) {
	now := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	store := &fakeDemotionStore{}
	p := newProviderBreakers(rerankBreakerConfig{FailureThreshold: 3, Cooldown: time.Minute}, store)
	for i := int64(0); i < defaultZeroYieldMinimumRoutes; i++ {
		p.recordResult("leaf", 0, false, false, now)
	}
	store.saved = nil
	require.True(t, p.eligibleAutomatic("leaf", now.Add(defaultZeroYieldDemotionWindow)))
	require.Len(t, store.saved, 1)
	require.True(t, store.saved[0].Probation)
}

func TestProviderBreakersRestoreReconcilesPersistedProbation(t *testing.T) {
	now := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	store := &fakeDemotionStore{states: []ProviderDemotionState{{
		ProviderID: "leaf", Routed: defaultZeroYieldMinimumRoutes, Demoted: true,
		Probation: true, DecayDeadline: now,
	}}}
	p := newProviderBreakers(rerankBreakerConfig{FailureThreshold: 3, Cooldown: time.Minute}, store)
	require.Equal(t, 1, p.restore(context.Background()))
	require.False(t, p.state("leaf").probation)
	require.Len(t, store.saved, 1)
	require.False(t, store.saved[0].Probation)
}

func TestProviderRecoveryState(t *testing.T) {
	now := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		stats     *providerYieldStats
		demoted   bool
		want      string
		wantStuck bool
	}{
		{name: "healthy", want: "healthy"},
		{name: "demoted", demoted: true, stats: &providerYieldStats{decayDeadline: now.Add(time.Hour)}, want: "demoted"},
		{name: "probe due", demoted: true, stats: &providerYieldStats{decayDeadline: now}, want: "probe_due"},
		{name: "probing", demoted: true, stats: &providerYieldStats{probation: true, decayDeadline: now.Add(time.Hour)}, want: "probing"},
		{name: "stuck", demoted: true, stats: &providerYieldStats{probation: true, decayDeadline: now}, want: "stuck", wantStuck: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, stuck := providerRecoveryState(tt.stats, tt.demoted, now)
			require.Equal(t, tt.want, got)
			require.Equal(t, tt.wantStuck, stuck)
		})
	}
}

func TestProviderBreakersPersistAndRestoreDemotionEvidence(t *testing.T) {
	now := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	store := &fakeDemotionStore{}
	p := newProviderBreakers(rerankBreakerConfig{FailureThreshold: 3, Cooldown: time.Minute}, store)
	for i := int64(0); i < defaultZeroYieldMinimumRoutes; i++ {
		p.recordResult("leaf", 0, false, false, now)
	}
	if len(store.saved) == 0 || !store.saved[len(store.saved)-1].Demoted {
		t.Fatalf("demotion was not persisted: %+v", store.saved)
	}
	store.states = []ProviderDemotionState{store.saved[len(store.saved)-1]}
	restored := newProviderBreakers(rerankBreakerConfig{FailureThreshold: 3, Cooldown: time.Minute}, store)
	restored.restore(context.Background())
	demoted, routed, hits := restored.demotion("leaf")
	if !demoted || routed != defaultZeroYieldMinimumRoutes || hits != 0 {
		t.Fatalf("restored demotion = %v routed=%d hits=%d", demoted, routed, hits)
	}
}

func TestProviderBreakersRepromoteClearsOnlyGradedEmptyEvidence(t *testing.T) {
	now := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	store := &fakeDemotionStore{}
	p := newProviderBreakers(rerankBreakerConfig{FailureThreshold: 1, Cooldown: time.Minute}, store)
	for i := int64(0); i < defaultZeroYieldMinimumRoutes; i++ {
		p.recordResult("leaf", 0, false, false, now)
	}
	p.record("leaf", true, now)
	require.True(t, p.repromote("leaf", now))
	demoted, routed, hits := p.demotion("leaf")
	require.False(t, demoted)
	require.Zero(t, routed)
	require.Zero(t, hits)
	ok, _ := p.allow("leaf", now)
	require.False(t, ok, "transport circuit state is preserved by repromotion")
	require.Contains(t, store.saved[len(store.saved)-1].Trigger, "operator repromotion")
}

func TestCircuitOpenQuorumExcludesRecoveryProbe(t *testing.T) {
	now := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	providers := []*registryv1.ProviderDescriptor{
		{ProviderId: "one", State: registryv1.ProviderState_PROVIDER_STATE_ACTIVE},
		{ProviderId: "two", State: registryv1.ProviderState_PROVIDER_STATE_ACTIVE},
		{ProviderId: "three", State: registryv1.ProviderState_PROVIDER_STATE_ACTIVE},
	}
	breakers := newProviderBreakers(rerankBreakerConfig{FailureThreshold: 1, Cooldown: time.Minute})
	breakers.openImmediately("one", now)
	breakers.openImmediately("two", now)
	router := &Router{
		deps: Deps{
			Lister: quorumLister{providers: providers},
			Now:    func() time.Time { return now },
		},
		providerBreakers: breakers,
	}
	share, breached, err := router.CircuitOpenQuorum(context.Background())
	require.NoError(t, err)
	require.InDelta(t, 2.0/3.0, share, 1e-9)
	require.True(t, breached)

	now = now.Add(time.Minute)
	share, breached, err = router.CircuitOpenQuorum(context.Background())
	require.NoError(t, err)
	require.Zero(t, share, "cooldown-elapsed recovery probes are reachable, not circuit-open")
	require.False(t, breached)
}
