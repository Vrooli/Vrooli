package onboard_test

import (
	"context"
	"testing"
	"time"

	"vrooli-bridge/internal/onboard"
	onboardmocks "vrooli-bridge/internal/onboard/mocks"
	"vrooli-bridge/internal/testutil/db"
	testmocks "vrooli-bridge/internal/testutil/mocks"

	"github.com/stretchr/testify/require"
	apiDB "github.com/vrooli/api-core/database"
)

type recordingMachineLinker struct{ correlation, nodeID string }

func (l *recordingMachineLinker) LinkCorrelatedNode(_ context.Context, correlationID, nodeID string) error {
	l.correlation, l.nodeID = correlationID, nodeID
	return nil
}
func (l *recordingMachineLinker) RecordCorrelatedTrust(_ context.Context, _ string, _ onboard.Conn) error {
	return nil
}

// [REQ:BRG-MEC-002] Terminal enrollment attempts are immutable and retries
// preserve a linked historic record rather than reopening it.
func TestEnrollmentAttemptRetryIsImmutableAndLinked(t *testing.T) {
	ctx := context.Background()
	d := db.NewSQLite(t)
	require.NoError(t, apiDB.EnsureSchemas(ctx, d, apiDB.SchemaProviderFunc(onboard.Schema)))
	repo := onboard.NewSQLiteRepository(d, testmocks.NewFakeClock(time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)))
	store := repo.(onboard.AttemptStore)
	first, err := onboard.NewAttempt("machine-1", map[string]string{"locator": "mac.local"})
	require.NoError(t, err)
	first, err = store.CreateAttempt(ctx, first)
	require.NoError(t, err)
	require.NoError(t, store.RecordCheckpoint(ctx, first.ID, "ssh_verified", "host key revalidated"))
	terminal, err := store.CompleteAttempt(ctx, first.ID, onboard.AttemptFailed, "service_install_failed", "launchd failure")
	require.NoError(t, err)
	require.Equal(t, onboard.AttemptFailed, terminal.State)
	_, err = store.CompleteAttempt(ctx, first.ID, onboard.AttemptSucceeded, "", "")
	require.Error(t, err)
	retry, err := store.RetryAttempt(ctx, first.ID, nil)
	require.NoError(t, err)
	require.NotEqual(t, first.ID, retry.ID)
	require.Equal(t, first.ID, retry.RetryOfAttemptID)
	require.Equal(t, first.InputSnapshot, retry.InputSnapshot)
}

func TestEnrollmentAttemptListForMachineIsNewestFirstAndDoesNotMixMachines(t *testing.T) {
	ctx := context.Background()
	d := db.NewSQLite(t)
	require.NoError(t, apiDB.EnsureSchemas(ctx, d, apiDB.SchemaProviderFunc(onboard.Schema)))
	store := onboard.NewSQLiteRepository(d, testmocks.NewFakeClock(time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC))).(onboard.AttemptStore)
	first, err := onboard.NewAttempt("machine-1", map[string]string{"host": "one"})
	require.NoError(t, err)
	first.CreatedAt = time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC)
	_, err = store.CreateAttempt(ctx, first)
	require.NoError(t, err)
	second, err := onboard.NewAttempt("machine-1", map[string]string{"host": "one"})
	require.NoError(t, err)
	second.CreatedAt = first.CreatedAt.Add(time.Minute)
	_, err = store.CreateAttempt(ctx, second)
	require.NoError(t, err)
	other, err := onboard.NewAttempt("machine-2", nil)
	require.NoError(t, err)
	_, err = store.CreateAttempt(ctx, other)
	require.NoError(t, err)

	listed, err := store.ListAttemptsForMachine(ctx, "machine-1")
	require.NoError(t, err)
	require.Equal(t, []string{second.ID, first.ID}, []string{listed[0].ID, listed[1].ID})
	for _, attempt := range listed {
		require.Equal(t, "machine-1", attempt.MachineID)
	}
}

func TestMachineEnrollmentCreatesAttemptBeforeSSHAndCompletesIt(t *testing.T) {
	ctx := context.Background()
	d := db.NewSQLite(t)
	require.NoError(t, apiDB.EnsureSchemas(ctx, d, apiDB.SchemaProviderFunc(onboard.Schema)))
	clk := testmocks.NewFakeClock(time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC))
	repo := onboard.NewSQLiteRepository(d, clk)
	driver := &onboardmocks.FakeSSHDriver{RunBootstrapMarkers: successMarkers(testNodeID)}
	svc := onboard.NewService(repo, driver, &onboardmocks.FakeCodeIssuer{Code: testCode}, &onboardmocks.FakeOnlineConfirmer{Online: true}, clk, onboard.WithEnrollmentResolver(fixedEnrollmentResolver{nodeID: testNodeID, paired: true}))
	decision, err := svc.StartMachineEnrollment(ctx, "machine-1", validInput())
	require.NoError(t, err)
	require.NotEmpty(t, decision.Attempt.ID)
	require.NotEmpty(t, decision.Attempt.CorrelationID)
	op := waitTerminal(t, svc, decision.Decision.OpID)
	require.Equal(t, decision.Attempt.CorrelationID, op.CorrelationID)
	store := repo.(onboard.AttemptStore)
	attempt, err := store.GetAttempt(ctx, decision.Attempt.ID)
	require.NoError(t, err)
	require.Equal(t, onboard.AttemptSucceeded, attempt.State)
	require.Equal(t, "enrolled", attempt.TerminalResult)
	require.Equal(t, "machine-machine-1", driver.CapturedKeyName)
}

func TestRetryMachineEnrollmentCreatesLinkedFreshAttempt(t *testing.T) {
	ctx := context.Background()
	d := db.NewSQLite(t)
	require.NoError(t, apiDB.EnsureSchemas(ctx, d, apiDB.SchemaProviderFunc(onboard.Schema)))
	clk := testmocks.NewFakeClock(time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC))
	repo := onboard.NewSQLiteRepository(d, clk)
	svc := onboard.NewService(repo, &onboardmocks.FakeSSHDriver{RunBootstrapMarkers: successMarkers(testNodeID)}, &onboardmocks.FakeCodeIssuer{Code: testCode}, &onboardmocks.FakeOnlineConfirmer{Online: true}, clk, onboard.WithEnrollmentResolver(fixedEnrollmentResolver{nodeID: testNodeID, paired: true}))
	first, err := svc.StartMachineEnrollment(ctx, "machine-1", validInput())
	require.NoError(t, err)
	_ = waitTerminal(t, svc, first.Decision.OpID)

	retry, err := svc.RetryMachineEnrollment(ctx, "machine-1", first.Attempt.ID, validInput())
	require.NoError(t, err)
	require.NotEqual(t, first.Attempt.ID, retry.Attempt.ID)
	require.Equal(t, first.Attempt.ID, retry.Attempt.RetryOfAttemptID)
	_ = waitTerminal(t, svc, retry.Decision.OpID)

	store := repo.(onboard.AttemptStore)
	prior, err := store.GetAttempt(ctx, first.Attempt.ID)
	require.NoError(t, err)
	current, err := store.GetAttempt(ctx, retry.Attempt.ID)
	require.NoError(t, err)
	require.Equal(t, onboard.AttemptSucceeded, prior.State)
	require.Equal(t, onboard.AttemptSucceeded, current.State)
}

func TestRetryMachineEnrollmentRejectsNonTerminalOrForeignAttempt(t *testing.T) {
	ctx := context.Background()
	d := db.NewSQLite(t)
	require.NoError(t, apiDB.EnsureSchemas(ctx, d, apiDB.SchemaProviderFunc(onboard.Schema)))
	clk := testmocks.NewFakeClock(time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC))
	repo := onboard.NewSQLiteRepository(d, clk)
	store := repo.(onboard.AttemptStore)
	pending, err := onboard.NewAttempt("machine-1", nil)
	require.NoError(t, err)
	pending, err = store.CreateAttempt(ctx, pending)
	require.NoError(t, err)
	svc := onboard.NewService(repo, &onboardmocks.FakeSSHDriver{}, &onboardmocks.FakeCodeIssuer{}, &onboardmocks.FakeOnlineConfirmer{}, clk)
	_, err = svc.RetryMachineEnrollment(ctx, "machine-1", pending.ID, validInput())
	require.ErrorContains(t, err, "must be terminal")
	_, err = svc.RetryMachineEnrollment(ctx, "machine-2", pending.ID, validInput())
	require.ErrorContains(t, err, "does not belong")
}

// [REQ:BRG-MEC-003] Pairing identity is persisted before a later service
// installation failure becomes terminal, so recovery never consumes marker text.
func TestPairedThenBootstrapFailureKeepsCorrelatedMachineLineage(t *testing.T) {
	ctx := context.Background()
	d := db.NewSQLite(t)
	require.NoError(t, apiDB.EnsureSchemas(ctx, d, apiDB.SchemaProviderFunc(onboard.Schema)))
	clk := testmocks.NewFakeClock(time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC))
	repo := onboard.NewSQLiteRepository(d, clk)
	linker := &recordingMachineLinker{}
	svc := onboard.NewService(repo,
		&onboardmocks.FakeSSHDriver{RunBootstrapExit: 1, RunBootstrapDiagnostics: "service install failed"},
		&onboardmocks.FakeCodeIssuer{Code: testCode}, &onboardmocks.FakeOnlineConfirmer{Online: false}, clk,
		onboard.WithEnrollmentResolver(onboard.EnrollmentResolverFunc(func(context.Context, string) (string, bool, error) { return testNodeID, true, nil })),
		onboard.WithMachineLinker(linker))
	decision, err := svc.StartMachineEnrollment(ctx, "machine-1", validInput())
	require.NoError(t, err)
	op := waitTerminal(t, svc, decision.Decision.OpID)
	require.Equal(t, onboard.StateFailed, op.State)
	require.Equal(t, onboard.FailureBootstrap, op.FailureReason)
	require.Equal(t, decision.Attempt.CorrelationID, linker.correlation)
	require.Equal(t, testNodeID, linker.nodeID)
	store := repo.(onboard.AttemptStore)
	attempt, err := store.GetAttempt(ctx, decision.Attempt.ID)
	require.NoError(t, err)
	require.Equal(t, onboard.AttemptFailed, attempt.State)
	require.Equal(t, string(onboard.FailureBootstrap), attempt.TerminalResult)
}
