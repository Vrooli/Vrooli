package programs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/storage"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	"program-runtime/internal/localworkbridge"
)

type ownerBoundaryFixture struct {
	mu      sync.Mutex
	held    bool
	revoked bool
}

func (b *ownerBoundaryFixture) Within(ctx context.Context, fn func(context.Context) error) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.held = true
	defer func() { b.held = false }()
	return fn(ctx)
}
func ownerFixture(t *testing.T) (context.Context, *ownerBoundaryFixture, *programsv1.Program, localworkbridge.SourceProgramIdentity) {
	t.Helper()
	owner := localworkbridge.SourceProgramIdentity{Name: localworkbridge.SourceCollectorName, Digest: "digest", RequestKey: "local-work-fixture", Subject: "service", Owner: "owner", BindingID: "binding", BindingVersion: "v1", ScopeSHA256: "policy"}
	owner.ID = submissionID(Identity{ProgramName: owner.Name, IdempotencyKey: owner.RequestKey})
	b := &ownerBoundaryFixture{}
	check := func(context.Context) error {
		require.True(t, b.held, "final check must execute inside composite boundary")
		if b.revoked {
			return localworkbridge.ErrSourceAuthority
		}
		return nil
	}
	a := &serviceAdmission{boundary: b, deadline: time.Now().Add(time.Hour).Format(time.RFC3339Nano), recheck: check, identity: func(string) (localworkbridge.SourceProgramIdentity, error) { return owner, nil }, check: func(ctx context.Context, actual localworkbridge.SourceProgramIdentity) error {
		if actual != owner {
			return localworkbridge.ErrSourceAuthority
		}
		return check(ctx)
	}}
	ctx := context.WithValue(context.Background(), servicePermitKey{}, a)
	p := &programsv1.Program{Id: owner.ID, ProgramName: owner.Name, ProgramDigest: owner.Digest, RequestDigest: "intent", Status: programsv1.ProgramStatus_PROGRAM_STATUS_ACCEPTED}
	return ctx, b, p, owner
}
func TestServiceAdmissionOwnershipSurvivesSaveAndSQLiteReopen(t *testing.T) {
	d := newProgramsTestDB(t)
	repo := NewRepository(d)
	ctx, _, p, want := ownerFixture(t)
	created, err := repo.Create(ctx, p)
	require.NoError(t, err)
	require.True(t, created)
	var raw string
	require.NoError(t, d.QueryRow("SELECT service_owner_json FROM programs WHERE id=?", p.Id).Scan(&raw))
	var stored localworkbridge.SourceProgramIdentity
	require.NoError(t, json.Unmarshal([]byte(raw), &stored))
	require.Equal(t, want, stored)
	p.Status = programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED
	p.Stdout = "bounded result"
	require.NoError(t, repo.Save(context.Background(), p))
	var sequence int
	var name, path string
	require.NoError(t, d.QueryRow("PRAGMA database_list").Scan(&sequence, &name, &path))
	require.NoError(t, d.Close())
	dsn, err := storage.SQLiteDSNAt(path, storage.SQLiteTuning{TxLock: "immediate"})
	require.NoError(t, err)
	reopened, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = reopened.Close() })
	reopened.SetMaxOpenConns(1)
	restarted := NewRepository(reopened).(*sqliteRepository)
	got, owner, err := restarted.GetServiceAdmission(ctx, p.Id)
	require.NoError(t, err)
	require.Equal(t, want, owner)
	require.Equal(t, p.Stdout, got.Stdout)
	require.False(t, got.CallerIdentityVerified)
	created, err = restarted.Create(ctx, p)
	require.NoError(t, err)
	require.False(t, created)
}
func TestServiceAdmissionFinalDenialLeavesNoRow(t *testing.T) {
	for _, reason := range []string{"revoked", "deadline", "identity", "agent"} {
		t.Run(reason, func(t *testing.T) {
			d := newProgramsTestDB(t)
			r := NewRepository(d)
			ctx, b, p, _ := ownerFixture(t)
			switch reason {
			case "revoked":
				b.revoked = true
			case "deadline":
				serviceAdmissionFrom(ctx).deadline = time.Now().Add(-time.Second).Format(time.RFC3339Nano)
			case "identity":
				p.ProgramDigest = "other"
			case "agent":
				p.CallerIdentityVerified = true
			}
			created, err := r.Create(ctx, p)
			require.Error(t, err)
			require.False(t, created)
			var count int
			require.NoError(t, d.QueryRow("SELECT count(*) FROM programs").Scan(&count))
			require.Zero(t, count)
		})
	}
}
func TestServiceAdmissionCannotAdoptUnownedOrMismatchedRows(t *testing.T) {
	d := newProgramsTestDB(t)
	r := NewRepository(d).(*sqliteRepository)
	ctx, b, p, _ := ownerFixture(t)
	require.NoError(t, r.Save(context.Background(), p))
	created, err := r.Create(ctx, p)
	require.NoError(t, err)
	require.False(t, created)
	_, _, err = r.GetServiceAdmission(ctx, p.Id)
	require.ErrorIs(t, err, localworkbridge.ErrSourceAuthority)
	// Ordinary user/CLI reads are unchanged; no service privilege is inferred.
	_, err = r.Get(context.Background(), p.Id)
	require.NoError(t, err)
	_, err = d.Exec("DELETE FROM programs")
	require.NoError(t, err)
	created, err = r.Create(ctx, p)
	require.NoError(t, err)
	require.True(t, created)
	_, err = d.Exec("UPDATE programs SET service_owner_json = json_set(service_owner_json, '$.BindingVersion', 'v2') WHERE id=?", p.Id)
	require.NoError(t, err)
	_, _, err = r.GetServiceAdmission(ctx, p.Id)
	require.ErrorIs(t, err, localworkbridge.ErrSourceAuthority)
	b.revoked = true
	_, _, err = r.GetServiceAdmission(ctx, "missing")
	require.ErrorIs(t, err, localworkbridge.ErrSourceAuthority)
}
func TestServiceAdmissionRejectsMemoryAndUnverifiedPermit(t *testing.T) {
	ctx, _, p, _ := ownerFixture(t)
	created, err := newMemoryRepository().Create(ctx, p)
	require.Error(t, err)
	require.False(t, created)
	memory := newMemoryRepository()
	require.NoError(t, memory.Save(context.Background(), p))
	_, err = memory.Get(ctx, p.Id)
	require.Error(t, err)
	_, err = WithSourcePermit(context.Background(), &localworkbridge.SourcePermit{}, &ownerBoundaryFixture{})
	require.Error(t, err)
}

func TestServiceAdmissionConcurrentInsertKeepsOneOwnedRow(t *testing.T) {
	d := newProgramsTestDB(t)
	r := NewRepository(d)
	ctx, _, p, want := ownerFixture(t)
	var wg sync.WaitGroup
	results := make(chan bool, 12)
	errs := make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); created, err := r.Create(ctx, p); results <- created; errs <- err }()
	}
	wg.Wait()
	close(results)
	close(errs)
	count := 0
	for created := range results {
		if created {
			count++
		}
	}
	require.Equal(t, 1, count)
	for err := range errs {
		require.NoError(t, err)
	}
	_, owner, err := r.(*sqliteRepository).GetServiceAdmission(ctx, p.Id)
	require.NoError(t, err)
	require.Equal(t, want, owner)
}

func TestServiceAdmissionUsesExistingDeclaredExecutionAndReplay(t *testing.T) {
	d := newProgramsTestDB(t)
	runner := &admissionRunner{}
	s := NewService(Options{Store: d, Runner: runner})
	ctx, b, _, owner := ownerFixture(t)
	identity := Identity{ProgramName: owner.Name, ProgramDigest: owner.Digest, IdempotencyKey: owner.RequestKey, AdmissionDeadline: serviceAdmissionFrom(ctx).deadline}
	first, _, err := s.SubmitDeclared(ctx, "session", "print(1)", programsv1.Provenance_PROVENANCE_TEST, false, false, identity, Caller{})
	require.NoError(t, err)
	require.NotNil(t, first)
	require.EqualValues(t, 1, runner.calls.Load())
	again, _, err := s.SubmitDeclared(ctx, "session", "print(1)", programsv1.Provenance_PROVENANCE_TEST, false, false, identity, Caller{})
	require.NoError(t, err)
	require.Equal(t, first.Id, again.Id)
	require.EqualValues(t, 1, runner.calls.Load())
	result, err := s.SourceProgram(ctx, first.Id)
	require.NoError(t, err)
	require.True(t, result.Found)
	require.Equal(t, owner, result.Identity)
	b.revoked = true
	_, _, err = s.Wait(ctx, first.Id, time.Millisecond)
	require.ErrorIs(t, err, localworkbridge.ErrSourceAuthority)
	_, _, err = s.SubmitDeclared(ctx, "session", "print(1)", programsv1.Provenance_PROVENANCE_TEST, false, false, identity, Caller{})
	require.Error(t, err)
	require.EqualValues(t, 1, runner.calls.Load())
}

func TestSourceProgramProjectionNeverReleasesPartialOrOversizedOutput(t *testing.T) {
	for _, status := range []programsv1.ProgramStatus{programsv1.ProgramStatus_PROGRAM_STATUS_RUNNING, programsv1.ProgramStatus_PROGRAM_STATUS_FAILED, programsv1.ProgramStatus_PROGRAM_STATUS_CANCELLED} {
		result, err := sourceProgramResult(&programsv1.Program{Status: status, Stdout: "partial"}, localworkbridge.SourceProgramIdentity{})
		require.NoError(t, err)
		require.Empty(t, result.Stdout)
	}
	_, err := sourceProgramResult(&programsv1.Program{Status: programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, Stdout: string(make([]byte, 65537))}, localworkbridge.SourceProgramIdentity{})
	require.Error(t, err)
}

type deadlineBlockedOwnerSQL struct {
	SQLExecutor
	reached bool
	bounded bool
}

func (s *deadlineBlockedOwnerSQL) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	s.reached = true
	_, s.bounded = ctx.Deadline()
	if !s.bounded {
		return nil, errors.New("missing admission SQL deadline")
	}
	<-ctx.Done()
	return s.SQLExecutor.ExecContext(ctx, query, args...)
}
func TestServiceAdmissionDeadlineBoundsBlockedFinalSQL(t *testing.T) {
	db := newProgramsTestDB(t)
	blocked := &deadlineBlockedOwnerSQL{SQLExecutor: db}
	repo := NewRepository(blocked)
	ctx, _, p, _ := ownerFixture(t)
	serviceAdmissionFrom(ctx).deadline = time.Now().Add(30 * time.Millisecond).Format(time.RFC3339Nano)
	created, err := repo.Create(ctx, p)
	require.Error(t, err)
	require.False(t, created)
	require.True(t, blocked.reached)
	require.True(t, blocked.bounded)
	var count int
	require.NoError(t, db.QueryRow("SELECT count(*) FROM programs").Scan(&count))
	require.Zero(t, count)
}
