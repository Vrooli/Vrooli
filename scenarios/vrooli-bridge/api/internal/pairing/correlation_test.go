package pairing_test

import (
	"context"
	"encoding/base64"
	"sync"
	"testing"
	"time"

	"vrooli-bridge/internal/pairing"
	"vrooli-bridge/internal/testutil/db"
	"vrooli-bridge/internal/testutil/mocks"

	"github.com/stretchr/testify/require"
	apiDB "github.com/vrooli/api-core/database"
)

type correlatedRegistrar struct {
	mu    sync.Mutex
	nodes map[string]string
}

func (r *correlatedRegistrar) RegisterNode(_ context.Context, _ pairing.NodeFacts) (string, error) {
	return "legacy-node", nil
}

func (r *correlatedRegistrar) RegisterNodeWithCorrelation(_ context.Context, _ pairing.NodeFacts, correlation string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if id := r.nodes[correlation]; id != "" {
		return id, nil
	}
	id := "node-" + correlation
	r.nodes[correlation] = id
	return id, nil
}

func (r *correlatedRegistrar) FindNodeByPairingCorrelation(_ context.Context, correlation string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if id := r.nodes[correlation]; id != "" {
		return id, nil
	}
	return "", pairing.ErrCodeNotFound
}

func TestCorrelatedRedemptionConcurrentReplayConverges(t *testing.T) {
	ctx := context.Background()
	d := db.NewSQLite(t)
	require.NoError(t, apiDB.EnsureSchemas(ctx, d, apiDB.SchemaProviderFunc(pairing.Schema)))
	clock := mocks.NewFakeClock(time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC))
	registrar := &correlatedRegistrar{nodes: map[string]string{}}
	svc := pairing.NewService(pairing.NewSQLiteRepository(d, clock), registrar, clock)
	issued, err := svc.IssueCodeForEnrollment(ctx, "mac", nil, 0, "attempt-concurrent")
	require.NoError(t, err)
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	type outcome struct {
		id  string
		err error
	}
	results := make(chan outcome, 2)
	for range 2 {
		go func() {
			id, err := svc.Redeem(ctx, issued.Code, key, pairing.NodeFacts{OS: "darwin", Arch: "amd64"})
			results <- outcome{id, err}
		}()
	}
	first, second := <-results, <-results
	require.NoError(t, first.err)
	require.NoError(t, second.err)
	require.Equal(t, first.id, second.id)
	require.Len(t, registrar.nodes, 1)
}

// [REQ:BRG-MEC-002] A replayed correlated redemption converges to one Node,
// and its durable correlation can be reconciled after a process interruption.
func TestCorrelatedRedemptionIsReplaySafe(t *testing.T) {
	ctx := context.Background()
	d := db.NewSQLite(t)
	require.NoError(t, apiDB.EnsureSchemas(ctx, d, apiDB.SchemaProviderFunc(pairing.Schema)))
	clock := mocks.NewFakeClock(time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC))
	registrar := &correlatedRegistrar{nodes: map[string]string{}}
	svc := pairing.NewService(pairing.NewSQLiteRepository(d, clock), registrar, clock)
	issued, err := svc.IssueCodeForEnrollment(ctx, "mac", []string{"presence.read"}, 0, "attempt-1")
	require.NoError(t, err)
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	first, err := svc.Redeem(ctx, issued.Code, key, pairing.NodeFacts{OS: "darwin", Arch: "amd64"})
	require.NoError(t, err)
	second, err := svc.Redeem(ctx, issued.Code, key, pairing.NodeFacts{OS: "darwin", Arch: "amd64"})
	require.NoError(t, err)
	require.Equal(t, first, second)
	resumed, err := svc.ReconcileEnrollments(ctx)
	require.NoError(t, err)
	require.Zero(t, resumed)
}

// A new Bridge-owned onboarding attempt for a node with an intact key must
// converge to the existing registry identity rather than create a duplicate.
func TestCorrelatedRedemptionReusesActiveCredential(t *testing.T) {
	ctx := context.Background()
	d := db.NewSQLite(t)
	require.NoError(t, apiDB.EnsureSchemas(ctx, d, apiDB.SchemaProviderFunc(pairing.Schema)))
	clock := mocks.NewFakeClock(time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC))
	registrar := &correlatedRegistrar{nodes: map[string]string{}}
	repo := pairing.NewSQLiteRepository(d, clock)
	svc := pairing.NewService(repo, registrar, clock)
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))

	firstCode, err := svc.IssueCodeForEnrollment(ctx, "mac", nil, 0, "attempt-first")
	require.NoError(t, err)
	first, err := svc.Redeem(ctx, firstCode.Code, key, pairing.NodeFacts{OS: "darwin", Arch: "amd64"})
	require.NoError(t, err)

	secondCode, err := svc.IssueCodeForEnrollment(ctx, "mac", nil, 0, "attempt-reconcile")
	require.NoError(t, err)
	second, err := svc.Redeem(ctx, secondCode.Code, key, pairing.NodeFacts{OS: "darwin", Arch: "amd64"})
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Len(t, registrar.nodes, 1)
	resolved, paired, err := svc.ResolveEnrollment(ctx, "attempt-reconcile")
	require.NoError(t, err)
	require.True(t, paired)
	require.Equal(t, first, resolved)
}
