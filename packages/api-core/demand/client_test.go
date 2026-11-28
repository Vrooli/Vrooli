package demand

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestAcquireBuildsStableBoundedCommandAndDecodesLease(t *testing.T) {
	var got []string
	client := Client{Runner: func(_ context.Context, _ string, args ...string) ([]byte, error) {
		got = append([]string(nil), args...)
		return []byte(`{"lease":{"lease_id":"demand_abc","scenario":"search-hub","consumer_id":"sess-1","kind":"program","status":"active","expires_at":"2026-09-06T12:00:00Z"}}`), nil
	}}
	lease, err := client.Acquire(context.Background(), AcquireRequest{Scenario: "search-hub", ConsumerID: "sess-1", Kind: KindProgram, RequestID: "req-1", TTL: 10 * time.Minute})
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if lease.LeaseID != "demand_abc" {
		t.Fatalf("lease id = %q", lease.LeaseID)
	}
	joined := strings.Join(got, " ")
	for _, want := range []string{"scenario demand acquire", "--scenario search-hub", "--consumer sess-1", "--kind program", "--request-id req-1", "--ttl 10m", "--json"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("command %q missing %q", joined, want)
		}
	}
}

func TestStartScenarioUsesDemandManagedLifecycleFlag(t *testing.T) {
	var got []string
	client := Client{Runner: func(_ context.Context, _ string, args ...string) ([]byte, error) {
		got = append([]string(nil), args...)
		return nil, nil
	}}
	if err := client.StartScenario(context.Background(), "search-hub", "shadow"); err != nil {
		t.Fatalf("StartScenario: %v", err)
	}
	if strings.Join(got, " ") != "scenario start search-hub@shadow --demand-managed" {
		t.Fatalf("args = %v", got)
	}
}

func TestReleaseIncludesReasonAndSurfacesControlPlaneOutput(t *testing.T) {
	client := Client{Runner: func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if strings.Join(args, " ") != "scenario demand release --lease-id lease-1 --json --reason done" {
			t.Fatalf("args = %v", args)
		}
		return []byte(`{"lease":{"lease_id":"lease-1","status":"released"}}`), nil
	}}
	lease, err := client.Release(context.Background(), "lease-1", "done")
	if err != nil || lease.Status != "released" {
		t.Fatalf("Release = %#v, %v", lease, err)
	}
}

func TestAcquireReportsMalformedControlPlaneResponse(t *testing.T) {
	client := Client{Runner: func(context.Context, string, ...string) ([]byte, error) { return []byte(`{}`), nil }}
	if _, err := client.Acquire(context.Background(), AcquireRequest{Scenario: "x", ConsumerID: "c", Kind: KindProgram}); err == nil || !strings.Contains(err.Error(), "lease_id") {
		t.Fatalf("error = %v", err)
	}
}

type scopedStub struct {
	acquired bool
	released string
}

func (s *scopedStub) Acquire(_ context.Context, req AcquireRequest) (Lease, error) {
	s.acquired = true
	return Lease{LeaseID: req.LeaseID}, nil
}
func (s *scopedStub) Renew(context.Context, string, time.Duration) (Lease, error) {
	return Lease{}, nil
}
func (s *scopedStub) Release(_ context.Context, leaseID, _ string) (Lease, error) {
	s.released = leaseID
	return Lease{LeaseID: leaseID}, nil
}

func TestAcquireScopedProvidesBoundedCleanup(t *testing.T) {
	stub := &scopedStub{}
	lease, release, err := AcquireScoped(context.Background(), stub, AcquireRequest{LeaseID: "lease-1"}, "done")
	if err != nil {
		t.Fatalf("AcquireScoped: %v", err)
	}
	if !stub.acquired || lease.LeaseID != "lease-1" || release == nil {
		t.Fatalf("acquire result = %#v, release=%v", lease, release != nil)
	}
	release()
	if stub.released != lease.LeaseID {
		t.Fatalf("released = %q, want %q", stub.released, lease.LeaseID)
	}
}

type lifetimeStub struct {
	renew      func(context.Context) error
	releases   atomic.Int32
	releaseErr error
}

func (s *lifetimeStub) Acquire(_ context.Context, _ AcquireRequest) (Lease, error) {
	return Lease{LeaseID: "lifetime", Status: "active", ExpiresAt: time.Now().Add(time.Minute)}, nil
}
func (s *lifetimeStub) Renew(ctx context.Context, _ string, _ time.Duration) (Lease, error) {
	return Lease{LeaseID: "lifetime", Status: "active", ExpiresAt: time.Now().Add(time.Minute)}, s.renew(ctx)
}
func (s *lifetimeStub) Release(ctx context.Context, _, _ string) (Lease, error) {
	if ctx.Err() != nil {
		return Lease{}, ctx.Err()
	}
	s.releases.Add(1)
	return Lease{LeaseID: "lifetime"}, s.releaseErr
}

func TestLifetimeCancelsWorkOnRenewalFailureAndReleasesOnce(t *testing.T) {
	failure := errors.New("control plane unavailable")
	stub := &lifetimeStub{renew: func(context.Context) error { return failure }}
	hold, err := AcquireLifetime(context.Background(), stub, AcquireRequest{TTL: 30 * time.Millisecond}, "done")
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-hold.Context().Done():
	case <-time.After(time.Second):
		t.Fatal("renewal loss did not cancel work")
	}
	if !errors.Is(context.Cause(hold.Context()), failure) {
		t.Fatalf("cause: %v", context.Cause(hold.Context()))
	}
	if err := hold.Close(); err != nil {
		t.Fatal(err)
	}
	if err := hold.Close(); err != nil {
		t.Fatal(err)
	}
	if stub.releases.Load() != 1 {
		t.Fatalf("releases: %d", stub.releases.Load())
	}
}

func TestLifetimeCancellationInterruptsRenewalAndUsesFreshCleanupContext(t *testing.T) {
	entered := make(chan struct{})
	stub := &lifetimeStub{renew: func(ctx context.Context) error { close(entered); <-ctx.Done(); return ctx.Err() }}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	hold, err := AcquireLifetime(ctx, stub, AcquireRequest{TTL: 300 * time.Millisecond}, "done")
	if err != nil {
		t.Fatal(err)
	}
	defer hold.Close()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("renewal did not start")
	}
	cancel()
	if err := hold.Close(); err != nil {
		t.Fatal(err)
	}
	if stub.releases.Load() != 1 {
		t.Fatal("cancelled work stranded its hold")
	}
}

func TestLifetimeCloseReportsReleaseFailure(t *testing.T) {
	failure := errors.New("release failed")
	stub := &lifetimeStub{releaseErr: failure}
	hold, err := AcquireLifetime(context.Background(), stub, AcquireRequest{}, "done")
	if err != nil {
		t.Fatal(err)
	}
	if err := hold.Close(); !errors.Is(err, failure) {
		t.Fatalf("close: %v", err)
	}
}

func TestStableVariantLeaseIdentity(t *testing.T) {
	live := StableVariantLeaseID("worker", "target", "", "request")
	if live != StableVariantLeaseID("worker", "target", "live", "request") || live != StableLeaseID("worker", "target", "request") {
		t.Fatal("default variant identity changed")
	}
	if live == StableVariantLeaseID("worker", "target", "shadow", "request") {
		t.Fatal("variant identities collide")
	}
}

// The requested TTL is not authority for the granted lease lifetime.
type grantedLifetimeStub struct {
	lifetimeStub
	acquired Lease
	renewal  func(context.Context) (Lease, error)
}

func (s *grantedLifetimeStub) Acquire(context.Context, AcquireRequest) (Lease, error) {
	return s.acquired, nil
}
func (s *grantedLifetimeStub) Renew(ctx context.Context, _ string, _ time.Duration) (Lease, error) {
	return s.renewal(ctx)
}

func TestLifetimeRenewsFromGrantedExpiry(t *testing.T) {
	renewed := make(chan struct{}, 1)
	stub := &grantedLifetimeStub{acquired: Lease{LeaseID: "lifetime", Status: "active", ExpiresAt: time.Now().Add(300 * time.Millisecond)}}
	stub.renewal = func(context.Context) (Lease, error) {
		renewed <- struct{}{}
		return Lease{LeaseID: "lifetime", Status: "active", ExpiresAt: time.Now().Add(time.Minute)}, nil
	}
	hold, err := AcquireLifetime(context.Background(), stub, AcquireRequest{TTL: time.Hour}, "done")
	if err != nil {
		t.Fatal(err)
	}
	defer hold.Close()
	select {
	case <-renewed:
	case <-time.After(time.Second):
		t.Fatal("ignored the granted expiry")
	}
	if hold.Context().Err() != nil {
		t.Fatalf("hold cancelled: %v", context.Cause(hold.Context()))
	}
}

func TestLifetimeExpiryCancelsDuringUnresponsiveRenewal(t *testing.T) {
	entered, unblock := make(chan struct{}), make(chan struct{})
	stub := &grantedLifetimeStub{acquired: Lease{LeaseID: "lifetime", Status: "active", ExpiresAt: time.Now().Add(250 * time.Millisecond)}}
	stub.renewal = func(context.Context) (Lease, error) {
		close(entered)
		<-unblock
		return Lease{LeaseID: "lifetime", Status: "active", ExpiresAt: time.Now().Add(time.Minute)}, nil
	}
	hold, err := AcquireLifetime(context.Background(), stub, AcquireRequest{TTL: time.Hour}, "done")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { close(unblock); hold.Close() }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("renewal not entered")
	}
	select {
	case <-hold.Context().Done():
	case <-time.After(time.Second):
		t.Fatal("expired lease did not cancel work")
	}
	if !errors.Is(context.Cause(hold.Context()), ErrLeaseExpired) {
		t.Fatalf("cause=%v", context.Cause(hold.Context()))
	}
}

func TestLifetimeRejectsInvalidAcquisition(t *testing.T) {
	for _, lease := range []Lease{
		{}, {LeaseID: "lifetime", Status: "active"},
		{LeaseID: "lifetime", Status: "active", ExpiresAt: time.Now().Add(-time.Second)},
		{LeaseID: "lifetime", Status: "released", ExpiresAt: time.Now().Add(time.Hour)},
	} {
		stub := &grantedLifetimeStub{acquired: lease}
		hold, err := AcquireLifetime(context.Background(), stub, AcquireRequest{}, "done")
		if hold != nil || !errors.Is(err, ErrInvalidLease) {
			t.Fatalf("hold=%v error=%v", hold, err)
		}
	}
}

func TestLifetimeRejectsForeignRenewalIdentity(t *testing.T) {
	stub := &grantedLifetimeStub{acquired: Lease{LeaseID: "lifetime", Status: "active", ExpiresAt: time.Now().Add(time.Minute)}}
	stub.renewal = func(context.Context) (Lease, error) {
		return Lease{LeaseID: "foreign", Status: "active", ExpiresAt: time.Now().Add(time.Hour)}, nil
	}
	hold, err := AcquireLifetime(context.Background(), stub, AcquireRequest{TTL: 30 * time.Millisecond}, "done")
	if err != nil {
		t.Fatal(err)
	}
	defer hold.Close()
	select {
	case <-hold.Context().Done():
	case <-time.After(time.Second):
		t.Fatal("accepted foreign identity")
	}
	if !errors.Is(context.Cause(hold.Context()), ErrInvalidLease) {
		t.Fatalf("cause=%v", context.Cause(hold.Context()))
	}
}

func TestLifetimeReschedulesAfterShorterRenewal(t *testing.T) {
	failure := errors.New("second renewal reached")
	var calls atomic.Int32
	stub := &grantedLifetimeStub{acquired: Lease{LeaseID: "lifetime", Status: "active", ExpiresAt: time.Now().Add(300 * time.Millisecond)}}
	stub.renewal = func(context.Context) (Lease, error) {
		if calls.Add(1) == 1 {
			return Lease{LeaseID: "lifetime", Status: "active", ExpiresAt: time.Now().Add(120 * time.Millisecond)}, nil
		}
		return Lease{}, failure
	}
	hold, err := AcquireLifetime(context.Background(), stub, AcquireRequest{TTL: time.Hour}, "done")
	if err != nil {
		t.Fatal(err)
	}
	defer hold.Close()
	select {
	case <-hold.Context().Done():
	case <-time.After(time.Second):
		t.Fatal("renewal cadence ignored updated expiry")
	}
	if !errors.Is(context.Cause(hold.Context()), failure) {
		t.Fatalf("cause=%v", context.Cause(hold.Context()))
	}
}

func TestLifetimeDoesNotReleaseForeignAcquisition(t *testing.T) {
	stub := &grantedLifetimeStub{acquired: Lease{LeaseID: "foreign", Status: "active", ExpiresAt: time.Now().Add(time.Hour)}}
	hold, err := AcquireLifetime(context.Background(), stub, AcquireRequest{LeaseID: "owned"}, "done")
	if hold != nil || !errors.Is(err, ErrInvalidLease) {
		t.Fatalf("hold=%v err=%v", hold, err)
	}
	if stub.releases.Load() != 0 {
		t.Fatal("foreign identity was released")
	}
}
