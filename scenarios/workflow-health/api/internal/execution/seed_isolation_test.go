package execution

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	routingpb "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing"
	seedspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/seeds"
	seedsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/seeds/seeds_v1connect"
	"google.golang.org/protobuf/types/known/structpb"
)

type seedOwnerFixture struct {
	seedsconnect.UnimplementedSeedServiceHandler
	mu                sync.Mutex
	applyKeys         []string
	clears            int
	failHealth        bool
	loseApplyResponse bool
}

var _ seedsconnect.SeedServiceHandler = (*seedOwnerFixture)(nil)

func (f *seedOwnerFixture) evidence(state seedspb.SeedState) *seedspb.SeedEvidence {
	return &seedspb.SeedEvidence{Scenario: "sample", LeaseId: "owner-lease", State: state, PairedStorageInstalled: true}
}
func (f *seedOwnerFixture) ApplySeed(_ context.Context, r *connect.Request[seedspb.ApplySeedRequest]) (*connect.Response[seedspb.ApplySeedResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.applyKeys = append(f.applyKeys, r.Msg.GetRequestId())
	if f.loseApplyResponse && len(f.applyKeys) == 1 {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("lost response"))
	}
	params, _ := structpb.NewStruct(map[string]any{"project": "/private/seeded-project"})
	return connect.NewResponse(&seedspb.ApplySeedResponse{SessionToken: "private-token", Evidence: f.evidence(seedspb.SeedState_SEED_STATE_ACTIVE), InitialParams: params}), nil
}
func (f *seedOwnerFixture) GetSeed(_ context.Context, r *connect.Request[seedspb.GetSeedRequest]) (*connect.Response[seedspb.GetSeedResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.Msg.GetSessionToken() != "private-token" {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("token"))
	}
	if f.failHealth {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("owner restarted"))
	}
	return connect.NewResponse(&seedspb.GetSeedResponse{Evidence: f.evidence(seedspb.SeedState_SEED_STATE_ACTIVE)}), nil
}
func (f *seedOwnerFixture) ClearSeed(_ context.Context, r *connect.Request[seedspb.ClearSeedRequest]) (*connect.Response[seedspb.ClearSeedResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.clears++
	if r.Msg.GetSessionToken() != "private-token" {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("token"))
	}
	e := f.evidence(seedspb.SeedState_SEED_STATE_CLEARED)
	e.Stats = &routingpb.LeaseStats{TestPoolRequests: 9, TestRootWrites: 2}
	return connect.NewResponse(&seedspb.ClearSeedResponse{Evidence: e}), nil
}
func newSeedOwner(t *testing.T, f *seedOwnerFixture) *SeedIsolation {
	t.Helper()
	path, h := seedsconnect.NewSeedServiceHandler(f)
	mux := http.NewServeMux()
	mux.Handle(path, h)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	owner := NewSeedIsolation()
	owner.ResolveOwner = func(context.Context, string) (string, error) { return server.URL, nil }
	owner.HeartbeatInterval = time.Millisecond
	return owner
}

func TestSeedOwnerLeaseKeepsInputsCountersAndOneClearAfterCallerCancellation(t *testing.T) {
	f := &seedOwnerFixture{loseApplyResponse: true}
	owner := newSeedOwner(t, f)
	ctx, cancel := context.WithCancel(t.Context())
	lease, err := owner.AcquireSeed(ctx, "sample", t.TempDir(), "request-identity")
	require.NoError(t, err)
	cancel()
	require.Equal(t, "/private/seeded-project", lease.InitialParams()["project"])
	evidence := lease.Close(ctx)
	require.Empty(t, evidence.ClearError)
	require.Equal(t, int64(9), evidence.TestPoolRequests)
	require.Equal(t, int64(2), evidence.TestRootWrites)
	require.Equal(t, evidence, lease.Close(t.Context()))
	f.mu.Lock()
	defer f.mu.Unlock()
	require.Equal(t, []string{"request-identity", "request-identity"}, f.applyKeys)
	require.Equal(t, 1, f.clears)
}
func TestSeedOwnerLossCancelsLeaseHealthWithoutSuccessFallback(t *testing.T) {
	f := &seedOwnerFixture{}
	owner := newSeedOwner(t, f)
	lease, err := owner.AcquireSeed(t.Context(), "sample", t.TempDir(), "health-request")
	require.NoError(t, err)
	f.mu.Lock()
	f.failHealth = true
	f.mu.Unlock()
	health := lease.(leaseHealth)
	select {
	case <-health.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("owner loss did not revoke healthy lease proof")
	}
	require.Error(t, health.Err())
	e := lease.Close(t.Context())
	require.NotEmpty(t, e.HeartbeatError)
}
func TestSelectedSeededCaseReceivesOwnerInputsAndUsesNoSecondLease(t *testing.T) {
	t.Setenv("VROOLI_STORAGE_ROOT", t.TempDir())
	root := makeExecutionFixture(t, true)
	p := filepath.Join(root, "bas", "cases", "dashboard.json")
	// Seed metadata is author-owned; this case must never execute with an empty
	// replacement lease or literal unresolved workspace input.
	raw, err := os.ReadFile(p)
	require.NoError(t, err)
	var flow map[string]any
	require.NoError(t, json.Unmarshal(raw, &flow))
	flow["metadata"].(map[string]any)["labels"].(map[string]any)["seed"] = "bas/seeds/seed.go"
	writeFixtureJSON(t, p, flow)
	owner := newSeedOwner(t, &seedOwnerFixture{})
	client := &fakeBASClient{}
	service := NewService(client)
	service.Seeds = owner
	unused := &fakeIsolation{}
	report, err := service.RunScenario(t.Context(), "sample", root, Options{IncludeExecution: true, RunID: "selected-seed", Isolation: unused, Selector: AssetSelector{CasesProvided: true, CasePaths: []string{"bas/cases/dashboard.json"}}})
	require.NoError(t, err)
	require.Equal(t, 1, client.executeCalls)
	require.False(t, unused.acquired)
	require.Equal(t, "/private/seeded-project", client.lastRequest.Parameters.InitialParams["project"])
	require.Equal(t, "owner-lease", client.lastRequest.Parameters.ExtraHeaders["X-Vrooli-Test-Lease-ID"])
	require.Equal(t, int64(9), report.Isolation.TestPoolRequests)
	require.Empty(t, report.Isolation.ClearError)
}

type refusedSeedCoordinator struct{ lease SeedLease }

func (c refusedSeedCoordinator) AcquireSeed(context.Context, string, string, string) (SeedLease, error) {
	if c.lease != nil {
		return c.lease, nil
	}
	return nil, errors.New("seed owner unavailable")
}

func TestSeededObserverRequiresSeedProofBeforeAnyBASExecution(t *testing.T) {
	for _, invalidProof := range []bool{false, true} {
		t.Run(fmt.Sprint(invalidProof), func(t *testing.T) {
			t.Setenv("VROOLI_STORAGE_ROOT", t.TempDir())
			root := makeExecutionFixture(t, false)
			p := filepath.Join(root, "bas", "cases", "dashboard.json")
			raw, err := os.ReadFile(p)
			require.NoError(t, err)
			var flow map[string]any
			require.NoError(t, json.Unmarshal(raw, &flow))
			labels := flow["metadata"].(map[string]any)["labels"].(map[string]any)
			labels["seed"] = "bas/seeds/seed.go"
			delete(labels, "routed_isolation")
			writeFixtureJSON(t, p, flow)
			client := &fakeBASClient{}
			service := NewService(client)
			coordinator := refusedSeedCoordinator{}
			if invalidProof {
				coordinator.lease = &invalidSeedLease{}
			}
			service.Seeds = coordinator
			report, err := service.RunScenario(t.Context(), "sample", root, Options{IncludeExecution: true, RunID: "seed-refusal", Selector: AssetSelector{CasesProvided: true, CasePaths: []string{"bas/cases/dashboard.json"}}})
			require.NoError(t, err)
			require.Zero(t, client.executeCalls)
			require.Equal(t, 1, report.Summary.Refused)
			require.NotEmpty(t, report.Isolation.InstallError)
		})
	}
}

type invalidSeedLease struct{}

func (*invalidSeedLease) InitialParams() map[string]any           { return nil }
func (*invalidSeedLease) Evidence() IsolationEvidence             { return IsolationEvidence{} }
func (*invalidSeedLease) Close(context.Context) IsolationEvidence { return IsolationEvidence{} }
