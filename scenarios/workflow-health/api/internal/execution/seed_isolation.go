package execution

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	seedspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/seeds"
	seedsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/seeds/seeds_v1connect"
)

// seam: SeedCoordinator acquires seeded storage from its Test Genie owner.
// Service consumes only the resulting lease and immutable input values.
type SeedCoordinator interface {
	AcquireSeed(context.Context, string, string, string) (SeedLease, error)
}

type SeedLease interface {
	IsolationLease
	InitialParams() map[string]any
}

// seedClient is the generated wire surface consumed by this adapter.
type seedClient interface {
	ApplySeed(context.Context, *connect.Request[seedspb.ApplySeedRequest]) (*connect.Response[seedspb.ApplySeedResponse], error)
	GetSeed(context.Context, *connect.Request[seedspb.GetSeedRequest]) (*connect.Response[seedspb.GetSeedResponse], error)
	ClearSeed(context.Context, *connect.Request[seedspb.ClearSeedRequest]) (*connect.Response[seedspb.ClearSeedResponse], error)
}

type SeedIsolation struct {
	ResolveOwner      func(context.Context, string) (string, error)
	NewClient         func(string) seedClient
	HeartbeatInterval time.Duration
}

var _ SeedCoordinator = (*SeedIsolation)(nil)

func NewSeedIsolation() *SeedIsolation {
	return &SeedIsolation{ResolveOwner: discovery.ResolveScenarioURLDefault, NewClient: func(base string) seedClient {
		return seedsconnect.NewSeedServiceClient(&http.Client{Timeout: 2 * time.Minute}, base)
	}, HeartbeatInterval: 10 * time.Second}
}

func (s *SeedIsolation) client(ctx context.Context) (seedClient, error) {
	if s == nil || s.ResolveOwner == nil || s.NewClient == nil {
		return nil, errors.New("seed owner client is not configured")
	}
	resolveCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	base, err := s.ResolveOwner(resolveCtx, "test-genie")
	if err != nil {
		return nil, errors.New("Test Genie seed owner discovery failed")
	}
	return s.NewClient(base), nil
}

func (s *SeedIsolation) AcquireSeed(ctx context.Context, scenario, targetDir, requestID string) (SeedLease, error) {
	if requestID == "" || strings.ContainsAny(scenario, ":@/") {
		return nil, errors.New("seeded validation requires a local live scenario and stable request identity")
	}
	request := &seedspb.ApplySeedRequest{Scenario: scenario, ScenarioPath: targetDir, RequestId: requestID}
	var response *connect.Response[seedspb.ApplySeedResponse]
	var err error
	// A second transport attempt re-resolves discovery and retains the same
	// request identity. It cannot create a second seed session at the owner.
	for attempt := 0; attempt < 2; attempt++ {
		client, clientErr := s.client(ctx)
		if clientErr != nil {
			return nil, clientErr
		}
		applyCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		response, err = client.ApplySeed(applyCtx, connect.NewRequest(request))
		cancel()
		if err == nil {
			break
		}
		if ctx.Err() != nil || (connect.CodeOf(err) != connect.CodeUnavailable && connect.CodeOf(err) != connect.CodeDeadlineExceeded) {
			return nil, errors.New("Test Genie seed apply failed")
		}
	}
	if err != nil || response == nil || response.Msg == nil {
		return nil, errors.New("Test Genie seed apply did not return a session")
	}
	msg := response.Msg
	evidence := msg.GetEvidence()
	if msg.GetSessionToken() == "" || !validSeedEvidence(evidence, scenario, "", seedspb.SeedState_SEED_STATE_ACTIVE) {
		if msg.GetSessionToken() != "" {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
			client, clientErr := s.client(cleanupCtx)
			if clientErr == nil {
				_, _ = client.ClearSeed(cleanupCtx, connect.NewRequest(&seedspb.ClearSeedRequest{Scenario: scenario, SessionToken: msg.GetSessionToken()}))
			}
			cancel()
		}
		return nil, errors.New("Test Genie did not prove a healthy paired seed lease")
	}
	interval := s.HeartbeatInterval
	if interval <= 0 {
		interval = 10 * time.Second
	}
	monitorCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	lease := &seedLease{owner: s, scenario: scenario, token: msg.GetSessionToken(), leaseID: evidence.GetLeaseId(), params: msg.GetInitialParams().AsMap(), cancel: cancel, stopped: make(chan struct{}), failed: make(chan struct{})}
	go lease.monitor(monitorCtx, interval)
	return lease, nil
}

func validSeedEvidence(e *seedspb.SeedEvidence, scenario, leaseID string, state seedspb.SeedState) bool {
	return e != nil && e.GetScenario() == scenario && e.GetLeaseId() != "" && (leaseID == "" || e.GetLeaseId() == leaseID) && e.GetPairedStorageInstalled() && e.GetState() == state && e.GetHeartbeatError() == "" && e.GetCleanupError() == ""
}

type seedLease struct {
	owner                    *SeedIsolation
	scenario, token, leaseID string
	params                   map[string]any
	cancel                   context.CancelFunc
	stopped, failed          chan struct{}
	mu                       sync.Mutex
	healthErr                error
	closeOnce                sync.Once
	result                   IsolationEvidence
}

var _ SeedLease = (*seedLease)(nil)
var _ leaseHealth = (*seedLease)(nil)

func (l *seedLease) InitialParams() map[string]any { return l.params }
func (l *seedLease) Evidence() IsolationEvidence {
	return IsolationEvidence{Installed: true, LeaseID: l.leaseID}
}
func (l *seedLease) Done() <-chan struct{} { return l.failed }
func (l *seedLease) Err() error            { l.mu.Lock(); defer l.mu.Unlock(); return l.healthErr }
func (l *seedLease) fail() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.healthErr == nil {
		l.healthErr = errors.New("Test Genie seed lease health is unavailable")
		close(l.failed)
	}
}
func (l *seedLease) monitor(ctx context.Context, interval time.Duration) {
	defer close(l.stopped)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			callCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			client, err := l.owner.client(callCtx)
			var response *connect.Response[seedspb.GetSeedResponse]
			if err == nil {
				response, err = client.GetSeed(callCtx, connect.NewRequest(&seedspb.GetSeedRequest{Scenario: l.scenario, SessionToken: l.token}))
			}
			cancel()
			if ctx.Err() != nil {
				return
			}
			if err != nil || response == nil || !validSeedEvidence(response.Msg.GetEvidence(), l.scenario, l.leaseID, seedspb.SeedState_SEED_STATE_ACTIVE) {
				l.fail()
				return
			}
		}
	}
}
func (l *seedLease) Close(parent context.Context) IsolationEvidence {
	l.closeOnce.Do(func() {
		l.cancel()
		<-l.stopped
		l.result = l.Evidence()
		if err := l.Err(); err != nil {
			l.result.HeartbeatError = err.Error()
		}
		ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), 30*time.Second)
		defer cancel()
		var response *connect.Response[seedspb.ClearSeedResponse]
		var err error
		for attempt := 0; attempt < 2; attempt++ {
			client, clientErr := l.owner.client(ctx)
			if clientErr != nil {
				err = clientErr
				break
			}
			response, err = client.ClearSeed(ctx, connect.NewRequest(&seedspb.ClearSeedRequest{Scenario: l.scenario, SessionToken: l.token}))
			if err == nil {
				break
			}
			if ctx.Err() != nil || (connect.CodeOf(err) != connect.CodeUnavailable && connect.CodeOf(err) != connect.CodeDeadlineExceeded) {
				break
			}
		}
		if err != nil || response == nil || response.Msg == nil {
			l.result.ClearError = "Test Genie seed cleanup could not be verified"
			return
		}

		e := response.Msg.GetEvidence()
		if !validSeedEvidence(e, l.scenario, l.leaseID, seedspb.SeedState_SEED_STATE_CLEARED) || e.GetStats() == nil {
			l.result.ClearError = "Test Genie seed cleanup has no successful paired routing proof"
		}
		if stats := e.GetStats(); stats != nil {
			l.result.TestPoolRequests = stats.GetTestPoolRequests()
			l.result.PrimaryDuringTestModeRequests = stats.GetPrimaryDuringTestModeRequests()
			l.result.TestRootWrites = stats.GetTestRootWrites()
			l.result.PrimaryRootWritesDuringTestMode = stats.GetPrimaryRootWritesDuringTestMode()
		}
		if err := l.Err(); err != nil {
			l.result.HeartbeatError = fmt.Sprint(err)
		}
	})
	return l.result
}
