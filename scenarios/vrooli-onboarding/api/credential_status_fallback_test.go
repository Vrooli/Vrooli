package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

type credentialStatusRecoveryStore struct {
	mu            sync.Mutex
	values        map[string]string
	unavailable   bool
	reads         int
	writes        int
	probeStarted  chan struct{}
	probeRelease  chan struct{}
	probeReturned chan struct{}
	probeOnce     sync.Once
}

func (s *credentialStatusRecoveryStore) Put(service, key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writes++
	s.values[service+"\x00"+key] = value
	return nil
}

func (s *credentialStatusRecoveryStore) Get(service, key string) (string, error) {
	if service == "vrooli.securestore.probe" && s.probeRelease != nil {
		s.probeOnce.Do(func() {
			close(s.probeStarted)
			<-s.probeRelease
			close(s.probeReturned)
		})
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reads++
	if s.unavailable {
		return "", errors.Join(credentialauthority.ErrProviderUnavailable, errors.New("fixture store temporarily locked"))
	}
	value, ok := s.values[service+"\x00"+key]
	if !ok {
		return "", credentialauthority.ErrNotFound
	}
	return value, nil
}

func (s *credentialStatusRecoveryStore) Delete(service, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.values, service+"\x00"+key)
	return nil
}

// TestCredentialStatusNamesAnUnusableStoreInsteadOfTheTransport pins what
// minimouse showed: with no credential store, the client refused to exist, and
// every credential read "credential transport is unavailable". Status is
// metadata-only, so readiness now asks the authority directly and the store's
// own condition and fix reach the operator.
func TestCredentialStatusNamesAnUnusableStoreInsteadOfTheTransport(t *testing.T) {
	previous := onboardingAuthority
	onboardingAuthority = func() (*credentialauthority.Authority, error) {
		return credentialauthority.Unavailable("no credential store on this host; run `vrooli credentials store init`")
	}
	t.Cleanup(func() { onboardingAuthority = previous })

	output, err := onboardingStatusJSON(context.Background(), "vrooli/openrouter", "api-key")
	if err != nil {
		t.Fatalf("status error = %v, want the store's condition", err)
	}
	var status struct {
		Configured     bool   `json:"configured"`
		ProviderState  string `json:"provider_state"`
		ProviderDetail string `json:"provider_detail"`
	}
	if err := json.Unmarshal(output, &status); err != nil {
		t.Fatalf("status is not JSON: %v (%s)", err, output)
	}
	if status.Configured || status.ProviderState == "" || status.ProviderState == "available" {
		t.Fatalf("status = %+v, want an unusable provider state", status)
	}
	if !strings.Contains(status.ProviderDetail, "vrooli credentials store init") {
		t.Fatalf("provider detail = %q, want the store's own remediation", status.ProviderDetail)
	}
}

func TestCredentialReadinessSkipsStoreWhenStatusProbeIsDisabled(t *testing.T) {
	store := &credentialStatusRecoveryStore{values: map[string]string{}}
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	previous := onboardingAuthority
	onboardingAuthority = func() (*credentialauthority.Authority, error) { return authority, nil }
	t.Cleanup(func() { onboardingAuthority = previous })

	items := credentialReadinessForRefsWithOptions(context.Background(), []credentialclient.CredentialRef{{LogicalID: "vrooli/demo", Field: "token", Required: true}}, false, false)
	if len(items) != 1 || store.reads != 0 {
		t.Fatalf("items=%+v provider reads=%d; non-probing readiness must not touch the credential store", items, store.reads)
	}
}

func TestCredentialReadinessSkipsProviderWhenRequestIsAlreadyCanceled(t *testing.T) {
	store := &credentialStatusRecoveryStore{values: map[string]string{}}
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	previous := onboardingAuthority
	authorityCalls := 0
	onboardingAuthority = func() (*credentialauthority.Authority, error) {
		authorityCalls++
		return authority, nil
	}
	t.Cleanup(func() { onboardingAuthority = previous })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	items := credentialReadinessForRefsWithOptions(ctx, []credentialclient.CredentialRef{{LogicalID: "vrooli/demo", Field: "token", Required: true}}, true, false)
	if len(items) != 1 || store.reads != 0 || authorityCalls != 0 {
		t.Fatalf("canceled readiness items=%+v store reads=%d authority calls=%d; want no provider initialization", items, store.reads, authorityCalls)
	}
}

func TestCredentialReadinessBoundsSlowProviderInitialization(t *testing.T) {
	store := &credentialStatusRecoveryStore{
		values:        map[string]string{},
		probeStarted:  make(chan struct{}),
		probeRelease:  make(chan struct{}),
		probeReturned: make(chan struct{}),
	}
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	previous := onboardingAuthority
	onboardingAuthority = func() (*credentialauthority.Authority, error) { return authority, nil }
	t.Cleanup(func() {
		select {
		case <-store.probeRelease:
		default:
			close(store.probeRelease)
		}
		onboardingAuthority = previous
	})

	started := time.Now()
	items := credentialReadinessForRefsWithOptions(context.Background(), []credentialclient.CredentialRef{{LogicalID: "vrooli/demo", Field: "token", Required: true}}, true, false)
	if elapsed := time.Since(started); elapsed > credentialStatusBudget+500*time.Millisecond {
		t.Fatalf("readiness waited %v for provider initialization, budget is %v", elapsed, credentialStatusBudget)
	}
	select {
	case <-store.probeStarted:
	default:
		t.Fatal("readiness returned without attempting the injected provider probe")
	}
	if len(items) != 1 || items[0].Status != credentialStatusPending || items[0].EvidenceNextAction != "retry-credential-verification" {
		t.Fatalf("slow-provider readiness = %+v; want retryable pending state", items)
	}
	close(store.probeRelease)
	select {
	case <-store.probeReturned:
	case <-time.After(time.Second):
		t.Fatal("injected provider probe did not finish after release")
	}
}

func TestCredentialReadinessCoalescesConcurrentProviderInitialization(t *testing.T) {
	store := &credentialStatusRecoveryStore{
		values:        map[string]string{},
		probeStarted:  make(chan struct{}),
		probeRelease:  make(chan struct{}),
		probeReturned: make(chan struct{}),
	}
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	previous := onboardingAuthority
	var authorityCallsMu sync.Mutex
	authorityCalls := 0
	onboardingAuthority = func() (*credentialauthority.Authority, error) {
		authorityCallsMu.Lock()
		defer authorityCallsMu.Unlock()
		authorityCalls++
		return authority, nil
	}
	t.Cleanup(func() {
		select {
		case <-store.probeRelease:
		default:
			close(store.probeRelease)
		}
		onboardingAuthority = previous
	})
	refs := []credentialclient.CredentialRef{{LogicalID: "vrooli/demo", Field: "token", Required: true}}
	results := make(chan []credentialReadiness, 2)
	go func() { results <- credentialReadinessForRefsWithOptions(context.Background(), refs, true, false) }()
	select {
	case <-store.probeStarted:
	case <-time.After(time.Second):
		t.Fatal("first readiness request did not start provider initialization")
	}
	go func() { results <- credentialReadinessForRefsWithOptions(context.Background(), refs, true, false) }()

	for request := 0; request < 2; request++ {
		select {
		case items := <-results:
			if len(items) != 1 || items[0].Status != credentialStatusPending {
				t.Fatalf("request %d result = %+v; want retryable pending while the shared provider is slow", request+1, items)
			}
		case <-time.After(2 * credentialStatusBudget):
			t.Fatalf("request %d did not respect the provider status budget", request+1)
		}
	}
	close(store.probeRelease)
	select {
	case <-store.probeReturned:
	case <-time.After(time.Second):
		t.Fatal("shared provider initialization did not finish after release")
	}
	authorityCallsMu.Lock()
	calls := authorityCalls
	authorityCallsMu.Unlock()
	if calls != 1 {
		t.Fatalf("overlapping readiness requests initialized the authority %d times; want one shared attempt", calls)
	}
}

func TestCredentialReadinessShowsProviderLossThenRecoversStoredValue(t *testing.T) {
	store := &credentialStatusRecoveryStore{values: map[string]string{}}
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := credentialauthority.ParseIdentity("vrooli/demo")
	if err != nil {
		t.Fatal(err)
	}
	const retained = "fixture-only-value"
	if err := authority.Put(identity, "token", retained); err != nil {
		t.Fatal(err)
	}
	store.mu.Lock()
	initialWrites := store.writes
	store.mu.Unlock()
	var authorityCallsMu sync.Mutex
	authorityCalls := 0
	previous := onboardingAuthority
	onboardingAuthority = func() (*credentialauthority.Authority, error) {
		authorityCallsMu.Lock()
		authorityCalls++
		authorityCallsMu.Unlock()
		return authority, nil
	}
	t.Cleanup(func() { onboardingAuthority = previous })
	ref := []credentialclient.CredentialRef{{LogicalID: string(identity), Field: "token", Required: true}}

	store.mu.Lock()
	store.unavailable = true
	store.mu.Unlock()
	recheckCredentialAuthority()
	degraded := credentialReadinessForRefsWithOptions(context.Background(), ref, true, false)
	if len(degraded) != 1 || degraded[0].ProviderState != "unavailable" || degraded[0].Status != "unsupported" {
		t.Fatalf("provider-loss readiness = %+v; want an explicit unavailable state", degraded)
	}
	authorityCallsMu.Lock()
	degradedCalls := authorityCalls
	authorityCallsMu.Unlock()
	if degradedCalls != 3 {
		t.Fatalf("unavailable readiness initialized authority %d times; want one admission, one recheck and one metadata fallback", degradedCalls)
	}

	store.mu.Lock()
	store.unavailable = false
	store.mu.Unlock()
	recheckCredentialAuthority()
	recovered := credentialReadinessForRefsWithOptions(context.Background(), ref, true, false)
	if len(recovered) != 1 || recovered[0].Status != "configured" || recovered[0].ProviderState != "available" {
		t.Fatalf("recovered readiness = %+v; want the same credential configured", recovered)
	}
	authorityCallsMu.Lock()
	recoveredCalls := authorityCalls
	authorityCallsMu.Unlock()
	if recoveredCalls != degradedCalls+2 {
		t.Fatalf("recovered readiness made %d additional authority constructions; want one recheck and one client admission", recoveredCalls-degradedCalls)
	}
	value, err := authority.Resolve(identity, "token")
	if err != nil || value != retained {
		t.Fatalf("stored value after recovery = %q, err=%v", value, err)
	}
	store.mu.Lock()
	writes := store.writes
	store.mu.Unlock()
	if writes != initialWrites {
		t.Fatalf("status transition added %d credential-store writes; want only the original user write", writes-initialWrites)
	}
}
