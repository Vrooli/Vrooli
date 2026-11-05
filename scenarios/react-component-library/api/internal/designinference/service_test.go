package designinference

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
)

type gatewayFunc func(context.Context, *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error)

func (f gatewayFunc) Run(ctx context.Context, r *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
	return f(ctx, r)
}
func request() Request {
	return Request{Role: "extract.structured", MaxOutputTokens: 4096, Source: "Source document", Schema: `{"type":"object"}`, Instruction: "Extract requirements", Context: []byte(`{"revision":"one"}`)}
}
func TestConcurrentDispatchIsDurableAndReplayDoesNotChargeAgain(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(db)
	var calls atomic.Int32
	entered, release := make(chan struct{}), make(chan struct{})
	client := gatewayFunc(func(ctx context.Context, r *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
		calls.Add(1)
		require.Equal(t, "extract.structured", r.Msg.Role)
		require.Equal(t, int32(4096), r.Msg.MaxOutputTokens)
		close(entered)
		<-release
		return connect.NewResponse(&inferencev1.RunResponse{ValueJson: `{"requirements":[]}`, Validated: true, Provider: "local", Model: "fixture", Usage: &inferencev1.Usage{InputTokens: 11, OutputTokens: 3}}), nil
	})
	svc := &Service{Repository: repo, Client: client}
	var first Operation
	var firstErr error
	var done sync.WaitGroup
	done.Add(1)
	go func() { defer done.Done(); first, firstErr = svc.Run(ctx, "key", request()) }()
	<-entered
	// A second service instance shares only authoritative storage.
	second, err := (&Service{Repository: NewSQLiteRepository(db), Client: client}).Run(ctx, "key", request())
	require.NoError(t, err)
	require.Equal(t, "dispatch_unknown", second.State)
	close(release)
	done.Wait()
	require.NoError(t, firstErr)
	require.Equal(t, "completed", first.State)
	replay, err := svc.Run(ctx, "key", request())
	require.NoError(t, err)
	require.Equal(t, first, replay)
	require.Equal(t, int32(1), calls.Load())
	require.Contains(t, replay.ResponseJSON, "inputTokens")
	changed := request()
	changed.Source = "Changed source"
	_, err = svc.Run(ctx, "key", changed)
	require.ErrorIs(t, err, ErrConflict)
}
func TestLostResponseAndCrashReservationAreNeverRedispatched(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(db)
	calls := 0
	svc := &Service{Repository: repo, Client: gatewayFunc(func(context.Context, *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
		calls++
		return nil, errors.New("response lost")
	})}
	op, err := svc.Run(ctx, "lost", request())
	require.NoError(t, err)
	require.Equal(t, "dispatch_unknown", op.State)
	_, err = svc.Run(ctx, "lost", request())
	require.NoError(t, err)
	require.Equal(t, 1, calls)
	prepared, err := repo.Create(ctx, "crash", request())
	require.NoError(t, err)
	claimed, err := repo.Claim(ctx, prepared.ID)
	require.NoError(t, err)
	require.True(t, claimed)
	resumed, err := svc.Run(ctx, "crash", request())
	require.NoError(t, err)
	require.Equal(t, "dispatch_unknown", resumed.State)
	require.Equal(t, 1, calls)
}
func TestReceivedFailureIsDurableAndResolutionFailureCanResume(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(db)
	svc := &Service{Repository: repo, Resolve: func(context.Context) (Gateway, error) { return nil, errors.New("unavailable") }}
	op, err := svc.Run(ctx, "key", request())
	require.Error(t, err)
	require.Equal(t, "prepared", op.State)
	svc.Client = gatewayFunc(func(context.Context, *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
		return connect.NewResponse(&inferencev1.RunResponse{Error: &inferencev1.InferenceError{Code: inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_VALIDATION_FAILED, Message: "Invalid result"}}), nil
	})
	op, err = svc.Run(ctx, "key", request())
	require.NoError(t, err)
	require.Equal(t, "failed", op.State)
	require.Contains(t, op.ResponseJSON, "VALIDATION_FAILED")
}

func TestReceivedResponseSurvivesCallerCancellation(t *testing.T) {
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(Schema)))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc := &Service{Repository: NewSQLiteRepository(db), Client: gatewayFunc(func(context.Context, *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
		cancel()
		return connect.NewResponse(&inferencev1.RunResponse{Validated: true, ValueJson: `{}`}), nil
	})}
	op, err := svc.Run(ctx, "cancel-observer", request())
	require.NoError(t, err)
	require.Equal(t, "completed", op.State)
	restored, err := NewSQLiteRepository(db).Get(context.Background(), op.ID)
	require.NoError(t, err)
	require.Equal(t, op, restored)
}

func TestInferenceScopeBindsScenarioAndPageWithoutBreakingHistoricalReads(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(db)
	historical, err := repo.Create(ctx, "historical", request())
	require.NoError(t, err)
	restored, err := repo.Get(ctx, historical.ID)
	require.NoError(t, err)
	require.Equal(t, historical, restored)
	scoped := request()
	scoped.Scope = &Scope{Scenario: "demo", Page: "home"}
	_, err = repo.Create(ctx, "scoped", scoped)
	require.NoError(t, err)
	for _, scope := range []*Scope{{Scenario: "other", Page: "home"}, {Scenario: "demo", Page: "other"}} {
		changed := scoped
		changed.Scope = scope
		_, err = repo.Create(ctx, "scoped", changed)
		require.ErrorIs(t, err, ErrConflict)
	}
	_, err = repo.Create(ctx, "historical", scoped)
	require.ErrorIs(t, err, ErrConflict)
}
