package operatorcapability

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type verifyingFixtureProvider struct {
	descriptorFixtureProvider
	receipts []EvidenceReference
}

func (p verifyingFixtureProvider) Verify(context.Context, VerificationRequest) ([]EvidenceReference, error) {
	return append([]EvidenceReference(nil), p.receipts...), nil
}

type blockingVerifyingFixtureProvider struct {
	verifyingFixtureProvider
	started chan<- struct{}
	release <-chan struct{}
}

type flakyVerifyingFixtureProvider struct {
	verifyingFixtureProvider
	remainingFailures int32
	calls             int32
}

func (p *flakyVerifyingFixtureProvider) Verify(ctx context.Context, request VerificationRequest) ([]EvidenceReference, error) {
	atomic.AddInt32(&p.calls, 1)
	if atomic.LoadInt32(&p.remainingFailures) > 0 {
		atomic.AddInt32(&p.remainingFailures, -1)
		return nil, &VerificationError{Code: "provider_rate_limited", Retryable: true, RetryAfter: time.Nanosecond, NextAction: "retry-after-provider-backoff", Cause: errors.New("fixture rate limit")}
	}
	return p.verifyingFixtureProvider.Verify(ctx, request)
}

func (p blockingVerifyingFixtureProvider) Verify(ctx context.Context, request VerificationRequest) ([]EvidenceReference, error) {
	select {
	case p.started <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	select {
	case <-p.release:
		return p.verifyingFixtureProvider.Verify(ctx, request)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func TestRegistryVerifyBoundsConcurrentProviderWork(t *testing.T) {
	descriptor := fixtureDescriptor("fixture/concurrency", "Concurrency fixture", SensitivityOperator, nil)
	descriptor.Evidence = EvidenceContract{Kinds: []string{"fixture"}, RequiredFields: []string{"target_id", "operation", "observed_at", "expires_at", "verified"}, SecretFree: true}
	now := time.Now().UTC()
	receipt := EvidenceReference{Kind: "fixture", ArtifactIdentity: "fixture-concurrency", TargetID: "host-1", Operation: "readiness-check", ObservedAt: now, ExpiresAt: now.Add(time.Minute), Verified: true}
	started := make(chan struct{}, maxConcurrentVerifications)
	release := make(chan struct{})
	provider := blockingVerifyingFixtureProvider{verifyingFixtureProvider: verifyingFixtureProvider{descriptorFixtureProvider: descriptorFixtureProvider{descriptor: descriptor}, receipts: []EvidenceReference{receipt}}, started: started, release: release}
	registry, err := NewRegistry(provider)
	if err != nil {
		t.Fatal(err)
	}

	var workers sync.WaitGroup
	for i := 0; i < maxConcurrentVerifications; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			_, _ = registry.Verify(context.Background(), VerificationRequest{CapabilityID: descriptor.ID, TargetID: "host-1", Operation: "readiness-check", Timeout: time.Second, Effect: EffectBudget{Class: EffectReadOnly}})
		}()
	}
	for i := 0; i < maxConcurrentVerifications; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("provider did not reach the bounded concurrency limit")
		}
	}

	queuedContext, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	if _, err := registry.Verify(queuedContext, VerificationRequest{CapabilityID: descriptor.ID, TargetID: "host-1", Operation: "readiness-check", Timeout: time.Second, Effect: EffectBudget{Class: EffectReadOnly}}); err == nil || !strings.Contains(err.Error(), "queue capability verification") {
		t.Fatalf("queued verification was not bounded by context: %v", err)
	}
	close(release)
	workers.Wait()
}

func TestRegistryVerifyBindsEvidenceToCredentialAndTarget(t *testing.T) {
	descriptor := fixtureDescriptor("fixture/verify", "Verification fixture", SensitivitySecret, nil)
	descriptor.Evidence = EvidenceContract{Kinds: []string{"fixture"}, Stages: []VerificationStage{VerificationAuthentication}, RequiredFields: []string{"owner", "credential_ref", "context_digest", "catalog_revision", "configuration_revision", "provider_adapter_version", "target_id", "operation", "stage", "observed_at", "expires_at", "verified"}, SecretFree: true}
	now := time.Now().UTC()
	verificationContext := map[string]string{"purpose": "release", "mode": "staging"}
	provider := verifyingFixtureProvider{
		descriptorFixtureProvider: descriptorFixtureProvider{descriptor: descriptor},
		receipts: []EvidenceReference{{
			SchemaVersion: EvidenceSchemaVersion, Kind: "fixture", ArtifactIdentity: "fixture-evidence-1", TargetID: "host-1", Operation: "readiness-check",
			Stage:         VerificationAuthentication,
			CredentialRef: &CredentialEvidenceRef{LogicalID: "vrooli/test", Field: "token", Version: "authority-version-4"}, ObservedAt: now, ExpiresAt: now.Add(15 * time.Minute), Status: "exercised", Verified: true,
			Owner: "fixture.owner", ContextDigest: VerificationContextDigest(verificationContext), CatalogRevision: "catalog-2", ConfigurationRevision: "config-9", ProviderAdapterVersion: "adapter-1",
		}},
	}
	registry, err := NewRegistry(provider)
	if err != nil {
		t.Fatal(err)
	}
	receipts, err := registry.Verify(context.Background(), VerificationRequest{
		CapabilityID: "fixture/verify", CredentialRef: provider.receipts[0].CredentialRef, TargetID: "host-1", Operation: "readiness-check",
		Context: verificationContext, CatalogRevision: "catalog-2", ConfigurationRevision: "config-9", ProviderAdapterVersion: "adapter-1",
		Effect: EffectBudget{Class: EffectReadOnly},
	})
	if err != nil || len(receipts) != 1 || receipts[0].CapabilityID != "fixture/verify" {
		t.Fatalf("Verify() = %+v, %v", receipts, err)
	}
	if !receipts[0].FreshAt(now.Add(time.Minute)) || receipts[0].FreshAt(now.Add(16*time.Minute)) {
		t.Fatalf("receipt freshness did not respect its observed/expiry window: %+v", receipts[0])
	}
	if receipts[0].Owner != descriptor.Owner || receipts[0].ContextDigest != VerificationContextDigest(verificationContext) {
		t.Fatalf("receipt context binding = %+v", receipts[0])
	}
	if _, err := registry.Verify(context.Background(), VerificationRequest{CapabilityID: descriptor.ID, CredentialRef: provider.receipts[0].CredentialRef, TargetID: "host-1", Operation: "readiness-check", Context: map[string]string{"purpose": "release", "mode": "production"}, CatalogRevision: "catalog-2", ConfigurationRevision: "config-9", ProviderAdapterVersion: "adapter-1", Effect: EffectBudget{Class: EffectReadOnly}}); err == nil || !strings.Contains(err.Error(), "context") {
		t.Fatalf("verification accepted evidence for a different context: %v", err)
	}

	provider.receipts[0].Stage = ""
	registry, _ = NewRegistry(provider)
	if _, err := registry.Verify(context.Background(), VerificationRequest{CapabilityID: "fixture/verify", CredentialRef: provider.receipts[0].CredentialRef, TargetID: "host-1", Operation: "readiness-check", Effect: EffectBudget{Class: EffectReadOnly}}); err == nil || !strings.Contains(err.Error(), "missing a valid stage") {
		t.Fatalf("verification accepted an unclassified receipt: %v", err)
	}
	provider.receipts[0].Stage = VerificationAuthentication

	provider.receipts[0].TargetID = "other-host"
	registry, _ = NewRegistry(provider)
	if _, err := registry.Verify(context.Background(), VerificationRequest{CapabilityID: "fixture/verify", CredentialRef: provider.receipts[0].CredentialRef, TargetID: "host-1", Operation: "readiness-check", Effect: EffectBudget{Class: EffectReadOnly}}); err == nil {
		t.Fatal("verification accepted evidence for a different target")
	}
	provider.receipts[0].TargetID = "host-1"
	provider.receipts[0].Environment = "production"
	provider.receipts[0].AccountIdentity = "account-a"
	registry, _ = NewRegistry(provider)
	if _, err := registry.Verify(context.Background(), VerificationRequest{CapabilityID: "fixture/verify", CredentialRef: provider.receipts[0].CredentialRef, TargetID: "host-1", Environment: "staging", AccountIdentity: "account-a", Operation: "readiness-check", Effect: EffectBudget{Class: EffectReadOnly}}); err == nil {
		t.Fatal("verification accepted evidence for a different environment")
	}
	if _, err := registry.Verify(context.Background(), VerificationRequest{CapabilityID: "fixture/verify", CredentialRef: provider.receipts[0].CredentialRef, TargetID: "host-1", Environment: "production", AccountIdentity: "account-b", Operation: "readiness-check", Effect: EffectBudget{Class: EffectReadOnly}}); err == nil {
		t.Fatal("verification accepted evidence for a different account")
	}
}

func TestVerificationBoundsAndProbeURLSafety(t *testing.T) {
	for _, raw := range []string{
		"http://provider.example.test/check", "https://user:password@provider.example.test/check", "https://localhost/check",
		"https://127.0.0.1/check", "https://10.0.0.3/check", "https://provider.example.test/check#fragment",
	} {
		if err := ValidateProbeURL(raw); err == nil {
			t.Fatalf("ValidateProbeURL(%q) accepted an unsafe endpoint", raw)
		}
	}
	if err := ValidateProbeURL("https://provider.example.test/check"); err != nil {
		t.Fatalf("valid provider URL rejected: %v", err)
	}
	request := VerificationRequest{CapabilityID: "fixture/check", TargetID: "host-1", Operation: "check", Context: map[string]string{"api_token": "redacted"}, Effect: EffectBudget{Class: EffectBoundedWrite, MaxOperations: 17}}
	if err := request.Validate(); err == nil || !strings.Contains(err.Error(), "secret metadata") {
		t.Fatalf("sensitive verification context was not rejected: %v", err)
	}
	request.Context = map[string]string{}
	if err := request.Validate(); err == nil || !strings.Contains(err.Error(), "effect budget") {
		t.Fatalf("unbounded effect budget was not rejected: %v", err)
	}
}

func TestRegistryVerifyWithRetryBoundsAttemptsAndAvoidsEffectReplay(t *testing.T) {
	descriptor := fixtureDescriptor("fixture/retry", "Retry fixture", SensitivityOperator, nil)
	descriptor.Evidence = EvidenceContract{Kinds: []string{"fixture"}, RequiredFields: []string{"target_id", "operation", "observed_at", "expires_at", "verified"}, SecretFree: true}
	now := time.Now().UTC()
	provider := &flakyVerifyingFixtureProvider{
		verifyingFixtureProvider: verifyingFixtureProvider{descriptorFixtureProvider: descriptorFixtureProvider{descriptor: descriptor}, receipts: []EvidenceReference{{Kind: "fixture", ArtifactIdentity: "fixture-retry", TargetID: "host-1", Operation: "readiness-check", ObservedAt: now, ExpiresAt: now.Add(time.Minute), Verified: true}}},
		remainingFailures:        2,
	}
	registry, err := NewRegistry(provider)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.VerifyWithRetry(context.Background(), VerificationRequest{CapabilityID: descriptor.ID, TargetID: "host-1", Operation: "readiness-check", Effect: EffectBudget{Class: EffectReadOnly}}); err != nil {
		t.Fatalf("read-only verification did not recover from bounded transient failures: %v", err)
	}
	if calls := atomic.LoadInt32(&provider.calls); calls != 3 {
		t.Fatalf("provider calls = %d, want bounded three attempts", calls)
	}

	provider.remainingFailures = 1
	provider.calls = 0
	if _, err := registry.VerifyWithRetry(context.Background(), VerificationRequest{CapabilityID: descriptor.ID, TargetID: "host-1", Operation: "readiness-check", Effect: EffectBudget{Class: EffectBoundedWrite, MaxOperations: 1, CleanupPolicy: "delete fixture"}}); err == nil {
		t.Fatal("bounded-write verification silently retried an effectful failure")
	}
	if calls := atomic.LoadInt32(&provider.calls); calls != 1 {
		t.Fatalf("bounded-write provider calls = %d, want one attempt", calls)
	}
}

func TestRegistryVerifyRejectsStaleEvidenceWhenDescriptorRequiresFreshness(t *testing.T) {
	descriptor := fixtureDescriptor("fixture/stale", "Stale verification fixture", SensitivitySecret, nil)
	descriptor.Evidence = EvidenceContract{
		Kinds: []string{"fixture"}, RequiredFields: []string{"observed_at", "expires_at", "verified"},
		SecretFree: true, Freshness: "must be current",
	}
	provider := verifyingFixtureProvider{
		descriptorFixtureProvider: descriptorFixtureProvider{descriptor: descriptor},
		receipts: []EvidenceReference{{
			SchemaVersion: EvidenceSchemaVersion, Kind: "fixture", ArtifactIdentity: "fixture-stale",
			TargetID: "host-1", Operation: "readiness-check", ObservedAt: time.Now().Add(-time.Hour), ExpiresAt: time.Now().Add(-time.Minute), Verified: true,
		}},
	}
	registry, err := NewRegistry(provider)
	if err != nil {
		t.Fatal(err)
	}
	_, err = registry.Verify(context.Background(), VerificationRequest{
		CapabilityID: "fixture/stale", TargetID: "host-1", Operation: "readiness-check",
		Effect: EffectBudget{Class: EffectReadOnly},
	})
	if err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("stale evidence was accepted: %v", err)
	}
}
