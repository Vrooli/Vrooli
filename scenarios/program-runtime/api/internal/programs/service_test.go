package programs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
)

type failTerminalProgramWrite struct{ SQLExecutor }

func (f failTerminalProgramWrite) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if strings.HasPrefix(query, "INSERT INTO programs") && len(args) > 4 && args[4] == "succeeded" {
		return nil, errors.New("terminal receipt write failed")
	}
	return f.SQLExecutor.ExecContext(ctx, query, args...)
}

type admissionRunner struct {
	calls atomic.Int32
	err   error
}

func (r *admissionRunner) Execute(context.Context, string, string, bool) (Result, error) {
	r.calls.Add(1)
	return Result{Stdout: "retained result"}, r.err
}

func TestCloseDeclaredAdmissionWinsAgainstDelayedSubmissionAndSurvivesRestart(t *testing.T) {
	db := newProgramsTestDB(t)
	runner := &admissionRunner{}
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	t.Cleanup(func() { once.Do(func() { close(release) }) })
	s := NewService(Options{Store: db, Runner: runner, Preflight: func(string) []*programsv1.Diagnostic { close(entered); <-release; return nil }})
	identity := Identity{ProgramName: "fixture.once", ProgramDigest: "digest", IdempotencyKey: "close-race", AdmissionDeadline: time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)}
	provenance := programsv1.Provenance_PROVENANCE_TEST
	result := make(chan *programsv1.Program, 1)
	errors := make(chan error, 1)
	go func() {
		p, _, err := s.SubmitDeclared(t.Context(), "session", "effect()", provenance, false, false, identity, Caller{})
		result <- p
		errors <- err
	}()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("submission did not enter preflight")
	}
	owner := NewService(Options{Store: db, Runner: runner})
	closed, err := owner.CloseDeclaredAdmission(t.Context(), "effect()", provenance, false, identity, Caller{})
	require.NoError(t, err)
	require.Equal(t, programsv1.ProgramStatus_PROGRAM_STATUS_CANCELLED, closed.Status)
	require.Empty(t, closed.SessionId)
	once.Do(func() { close(release) })
	require.NoError(t, <-errors)
	require.Equal(t, closed.Id, (<-result).Id)
	restarted := NewService(Options{Store: db, Runner: runner})
	replay, _, err := restarted.SubmitDeclared(t.Context(), "", "effect()", provenance, false, false, identity, Caller{})
	require.NoError(t, err)
	require.Equal(t, closed.Id, replay.Id)
	require.Equal(t, "admission_closed", replay.FailureShape)
	_, err = restarted.CloseDeclaredAdmission(t.Context(), "different()", provenance, false, identity, Caller{})
	require.ErrorIs(t, err, ErrRequestConflict)
	_, err = restarted.CloseDeclaredAdmission(t.Context(), "effect()", provenance, false, Identity{}, Caller{})
	require.ErrorIs(t, err, ErrInvalidRequestKey)
	require.Zero(t, runner.calls.Load())
}

func TestDeclaredKeyRejectsChangedIntentAndRetainsFailedExecution(t *testing.T) {
	runner := &admissionRunner{err: errors.New("owner failed after a possible effect")}
	s := NewService(Options{Runner: runner, Store: newProgramsTestDB(t)})
	identity := Identity{ProgramName: "fixture.once", ProgramDigest: "digest", IdempotencyKey: "owner-attempt", AdmissionDeadline: time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)}
	caller := Caller{RunID: "owner", AgentProfile: "profile", SkillID: "skill", Harness: "harness"}
	source := "effect()"
	provenance := programsv1.Provenance_PROVENANCE_TEST
	original, _, err := s.SubmitDeclared(t.Context(), "s", source, provenance, false, false, identity, caller)
	require.NoError(t, err)
	require.Equal(t, programsv1.ProgramStatus_PROGRAM_STATUS_FAILED, original.Status)
	s.validateSession = func(string) bool { t.Error("replay allocated or validated a new session"); return false }
	replayed, _, err := s.SubmitDeclared(t.Context(), "", source, provenance, false, false, identity, caller)
	require.NoError(t, err)
	require.Equal(t, original.Id, replayed.Id)
	require.Equal(t, original.FailureDetail, replayed.FailureDetail)
	for _, change := range []string{"source", "digest", "provenance", "output", "run", "profile", "skill", "harness", "deadline", "grants"} {
		t.Run(change, func(t *testing.T) {
			i, c, text, p, expanded := identity, caller, source, provenance, false
			switch change {
			case "source":
				text += "changed()"
			case "digest":
				i.ProgramDigest = "other"
			case "provenance":
				p = programsv1.Provenance_PROVENANCE_OPERATOR
			case "output":
				expanded = true
			case "run":
				c.RunID = "other"
			case "profile":
				c.AgentProfile = "other"
			case "skill":
				c.SkillID = "other"
			case "harness":
				c.Harness = "other"
			case "deadline":
				i.AdmissionDeadline = time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339Nano)
			case "grants":
				i.Grants = []string{"binding:workspace-sandbox/change/promote"}
			}
			_, _, err := s.SubmitDeclared(t.Context(), "", text, p, expanded, false, i, c)
			require.ErrorIs(t, err, ErrRequestConflict)
		})
	}
	require.EqualValues(t, 1, runner.calls.Load())
}

func TestDeclaredUsageReceiptSettlesBeforeSessionReclamationAndPersists(t *testing.T) {
	db := newProgramsTestDB(t)
	var snapshots atomic.Int32
	s := NewService(Options{Store: db, Runner: &admissionRunner{}, UsageReceipt: func(context.Context, string) (UsageReceipt, error) {
		snapshots.Add(1)
		return UsageReceipt{Tokens: 12, ChargeMicros: 34, AccountingComplete: true, ChargeMeasured: true, Basis: "dedicated_declared_session"}, nil
	}})
	identity := Identity{ProgramName: "fixture.accounted", ProgramDigest: "digest", IdempotencyKey: "accounting-receipt", AdmissionDeadline: time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)}
	p, _, err := s.SubmitDeclared(t.Context(), "session", "local()", programsv1.Provenance_PROVENANCE_TEST, false, false, identity, Caller{})
	require.NoError(t, err)
	require.Equal(t, int32(1), snapshots.Load())
	restarted := NewService(Options{Store: db})
	got, err := restarted.Get(t.Context(), p.Id)
	require.NoError(t, err)
	require.Equal(t, int64(12), got.GetUsageTokens())
	require.Equal(t, int64(34), got.GetUsageChargeMicros())
	require.True(t, got.GetUsageAccountingComplete())
	require.True(t, got.GetUsageChargeMeasured())
	require.Equal(t, "dedicated_declared_session", got.GetUsageBasis())
}

func TestDeclaredUsagePersistenceFailureRemainsUnknownToCallerAndRestart(t *testing.T) {
	db := newProgramsTestDB(t)
	s := NewService(Options{Store: failTerminalProgramWrite{SQLExecutor: db}, Runner: &admissionRunner{}, UsageReceipt: func(context.Context, string) (UsageReceipt, error) {
		return UsageReceipt{Tokens: 12, ChargeMicros: 34, AccountingComplete: true, ChargeMeasured: true, Basis: "dedicated_declared_session"}, nil
	}})
	identity := Identity{ProgramName: "fixture.write-fails", ProgramDigest: "digest", IdempotencyKey: "accounting-write-failure", AdmissionDeadline: time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)}
	p, _, err := s.SubmitDeclared(t.Context(), "session", "local()", programsv1.Provenance_PROVENANCE_TEST, false, false, identity, Caller{})
	require.NoError(t, err)
	require.False(t, p.GetUsageAccountingComplete())
	require.False(t, p.GetUsageChargeMeasured())
	restarted := NewService(Options{Store: db})
	stored, err := restarted.Get(t.Context(), p.Id)
	require.NoError(t, err)
	require.False(t, stored.GetUsageAccountingComplete())
	require.NotEqual(t, programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, stored.GetStatus())
}

func TestDeclaredKeyConcurrentAdmissionHasOneExecution(t *testing.T) {
	for _, persistent := range []bool{false, true} {
		t.Run(fmt.Sprint("sqlite=", persistent), func(t *testing.T) {
			const callers = 12
			runner := &admissionRunner{}
			var store SQLExecutor
			if persistent {
				store = newProgramsTestDB(t)
			}
			memory := newMemoryRepository()
			ready, release := make(chan struct{}, callers), make(chan struct{})
			var once sync.Once
			t.Cleanup(func() { once.Do(func() { close(release) }) })
			results := make(chan *programsv1.Program, callers)
			errors := make(chan error, callers)
			identity := Identity{ProgramName: "fixture.once", ProgramDigest: "digest", IdempotencyKey: "concurrent", AdmissionDeadline: time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)}
			for n := range callers {
				s := NewService(Options{Store: store, Runner: runner, Preflight: func(string) []*programsv1.Diagnostic { ready <- struct{}{}; <-release; return nil }})
				if !persistent {
					s.repo = memory
				}
				go func() {
					p, _, err := s.SubmitDeclared(t.Context(), fmt.Sprint("s", n), "effect()", programsv1.Provenance_PROVENANCE_TEST, false, false, identity, Caller{})
					results <- p
					errors <- err
				}()
			}
			for range callers {
				select {
				case <-ready:
				case <-time.After(5 * time.Second):
					t.Fatal("concurrent preflight did not rendezvous")
				}
			}
			once.Do(func() { close(release) })
			var id, session string
			for range callers {
				require.NoError(t, <-errors)
				p := <-results
				if id == "" {
					id, session = p.Id, p.SessionId
				}
				require.Equal(t, id, p.Id)
				require.Equal(t, session, p.SessionId, "loser overwrote the winner's identity")
			}
			require.EqualValues(t, 1, runner.calls.Load())
		})
	}
}

func TestDeclaredKeyInterruptedOwnerNeverReexecutes(t *testing.T) {
	db := newProgramsTestDB(t)
	identity := Identity{ProgramName: "fixture.once", ProgramDigest: "digest", IdempotencyKey: "interrupted", AdmissionDeadline: time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)}
	provenance := programsv1.Provenance_PROVENANCE_TEST
	for _, status := range []programsv1.ProgramStatus{programsv1.ProgramStatus_PROGRAM_STATUS_ACCEPTED, programsv1.ProgramStatus_PROGRAM_STATUS_RUNNING} {
		identity.IdempotencyKey = status.String()
		original := &programsv1.Program{Id: submissionID(identity), SessionId: "dead", Source: "effect()", Provenance: provenance, Status: status, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano), ProgramName: identity.ProgramName, ProgramDigest: identity.ProgramDigest, OutputLimitBytes: 4096, RequestDigest: declaredRequestDigest("effect()", provenance, false, identity, Caller{})}
		created, err := NewRepository(db).Create(t.Context(), original)
		require.NoError(t, err)
		require.True(t, created)
		runner := &admissionRunner{}
		restarted := NewService(Options{Store: db, Runner: runner})
		n, err := restarted.RecoverInterrupted(t.Context())
		require.NoError(t, err)
		require.EqualValues(t, 1, n)
		p, _, err := restarted.SubmitDeclared(t.Context(), "", original.Source, provenance, false, false, identity, Caller{})
		require.NoError(t, err)
		require.Equal(t, original.Id, p.Id)
		require.Equal(t, programsv1.FailureCause_FAILURE_CAUSE_RUNTIME_INTERRUPTED, p.FailureCause)
		require.Zero(t, runner.calls.Load())
	}
}

func TestDeclaredKeyPreflightFailureIsNotRetried(t *testing.T) {
	runner := &admissionRunner{}
	s := NewService(Options{Runner: runner, Preflight: func(string) []*programsv1.Diagnostic {
		return []*programsv1.Diagnostic{{Severity: "error", Message: "blocked"}}
	}})
	identity := Identity{ProgramName: "fixture.once", ProgramDigest: "digest", IdempotencyKey: "preflight", AdmissionDeadline: time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)}
	first, _, err := s.SubmitDeclared(t.Context(), "s", "effect()", programsv1.Provenance_PROVENANCE_TEST, true, false, identity, Caller{})
	require.NoError(t, err)
	require.Equal(t, programsv1.ProgramStatus_PROGRAM_STATUS_FAILED, first.Status)
	s.preflight = nil
	second, _, err := s.SubmitDeclared(t.Context(), "s", "effect()", programsv1.Provenance_PROVENANCE_TEST, true, false, identity, Caller{})
	require.NoError(t, err)
	require.Equal(t, first.Id, second.Id)
	require.Equal(t, first.Status, second.Status)
	require.Zero(t, runner.calls.Load())
	identity.IdempotencyKey, identity.AdmissionDeadline = "", ""
	first, _, err = s.SubmitDeclared(t.Context(), "s", "effect()", programsv1.Provenance_PROVENANCE_TEST, true, false, identity, Caller{})
	require.NoError(t, err)
	second, _, err = s.SubmitDeclared(t.Context(), "s", "effect()", programsv1.Provenance_PROVENANCE_TEST, true, false, identity, Caller{})
	require.NoError(t, err)
	require.NotEqual(t, first.Id, second.Id, "unkeyed requests remain independent")
	require.EqualValues(t, 2, runner.calls.Load())
}

func TestDeclaredKeyRejectsInvalidOrExpiredAdmissionBeforeEffects(t *testing.T) {
	now := time.Now().UTC()
	valid := Identity{ProgramName: "fixture.once", ProgramDigest: "digest", IdempotencyKey: "key", AdmissionDeadline: now.Add(time.Hour).Format(time.RFC3339Nano)}
	for _, problem := range []string{"long key", "padded key", "control key", "missing pin", "missing deadline", "invalid deadline", "expired", "distant", "deadline without key", "preflight expired"} {
		t.Run(problem, func(t *testing.T) {
			i, clock, want := valid, now, ErrInvalidRequestKey
			switch problem {
			case "long key":
				i.IdempotencyKey = strings.Repeat("k", 129)
			case "padded key":
				i.IdempotencyKey = " key"
			case "control key":
				i.IdempotencyKey = "key\x00"
			case "missing pin":
				i.ProgramDigest = ""
			case "missing deadline":
				i.AdmissionDeadline = ""
			case "invalid deadline":
				i.AdmissionDeadline = "later"
			case "expired":
				i.AdmissionDeadline, want = now.Format(time.RFC3339Nano), ErrRequestExpired
			case "distant":
				i.AdmissionDeadline = now.Add(25 * time.Hour).Format(time.RFC3339Nano)
			case "deadline without key":
				i.IdempotencyKey = ""
			case "preflight expired":
				want = ErrRequestExpired
			}
			runner := &admissionRunner{}
			s := NewService(Options{Runner: runner, Clock: func() time.Time { return clock }, Preflight: func(string) []*programsv1.Diagnostic {
				if problem == "preflight expired" {
					clock = clock.Add(2 * time.Hour)
				}
				return nil
			}})
			_, _, err := s.SubmitDeclared(t.Context(), "s", "effect()", programsv1.Provenance_PROVENANCE_TEST, false, false, i, Caller{})
			require.ErrorIs(t, err, want)
			require.Zero(t, runner.calls.Load())
			require.Empty(t, s.List(t.Context(), "", true))
		})
	}
}

type fakeRunner struct {
	result Result
	err    error
}

type blockingRunner struct {
	started chan struct{}
	release chan struct{}
}

func (r *blockingRunner) Execute(context.Context, string, string, bool) (Result, error) {
	close(r.started)
	<-r.release
	return Result{Stdout: "done\n", ContextBytes: 5, AgentBytes: 5}, nil
}

func (r fakeRunner) Execute(context.Context, string, string, bool) (Result, error) {
	return r.result, r.err
}

func TestRetainsProgramSourceAndFailureDetail(t *testing.T) { // [REQ:PRT-P1-006]
	s := NewService(Options{Runner: fakeRunner{err: errors.New("field title: invalid")}})
	p, err := s.Submit(context.Background(), "s1", "raise ValueError()", programsv1.Provenance_PROVENANCE_AGENT, false)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(context.Background(), p.Id)
	if err != nil || got.Source != "raise ValueError()" || got.FailureDetail == "" {
		t.Fatalf("program=%+v err=%v", got, err)
	}
}

func TestSuccessfulAgentProgramIsRetained(t *testing.T) {
	s := NewService(Options{Runner: fakeRunner{result: Result{Stdout: "ok"}}})
	p, err := s.Submit(context.Background(), "s1", "print('ok')", programsv1.Provenance_PROVENANCE_AGENT, false)
	if err != nil {
		t.Fatal(err)
	}
	if p.GetStatus() != programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED {
		t.Fatalf("status=%v", p.GetStatus())
	}
}

func TestAsyncSubmissionReturnsAcceptedAndPublishesTerminalState(t *testing.T) {
	runner := &blockingRunner{started: make(chan struct{}), release: make(chan struct{})}
	s := NewService(Options{Runner: runner, ValidateSession: func(string) bool { return true }})
	p, err := s.Submit(context.Background(), "s1", "print('done')", programsv1.Provenance_PROVENANCE_AGENT, false, true)
	if err != nil || p.Status != programsv1.ProgramStatus_PROGRAM_STATUS_ACCEPTED {
		t.Fatalf("accepted program=%v err=%v", p, err)
	}
	<-runner.started
	running, err := s.Get(context.Background(), p.Id)
	if err != nil || running.Status != programsv1.ProgramStatus_PROGRAM_STATUS_RUNNING {
		t.Fatalf("running program=%v err=%v", running, err)
	}
	close(runner.release)
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		finished, getErr := s.Get(context.Background(), p.Id)
		if getErr == nil && finished.Status == programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED {
			if finished.Stdout != "done\n" {
				t.Fatalf("stdout=%q", finished.Stdout)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("async program did not reach succeeded state")
}

func TestRecurringFailureShapesAreDerivable(t *testing.T) { // [REQ:PRT-P1-006]
	s := NewService(Options{Runner: fakeRunner{err: errors.New("field title: invalid")}})
	for i := 0; i < 3; i++ {
		if _, err := s.Submit(context.Background(), "s1", "x", programsv1.Provenance_PROVENANCE_AGENT, false); err != nil {
			t.Fatal(err)
		}
	}
	shapes := s.MineFailures(context.Background(), false)
	if len(shapes) != 1 || shapes[0].Count != 3 {
		t.Fatalf("shapes=%v", shapes)
	}
}

func TestSubmissionRecordsProvenance(t *testing.T) { // [REQ:PRT-P1-008]
	s := NewService(Options{})
	p, err := s.Submit(context.Background(), "s1", "x=1", programsv1.Provenance_PROVENANCE_OPERATOR, false)
	if err != nil {
		t.Fatal(err)
	}
	if p.Provenance != programsv1.Provenance_PROVENANCE_OPERATOR {
		t.Fatalf("provenance=%v", p.Provenance)
	}
}

func TestListFilteredHonorsProvenanceAndTimeWindow(t *testing.T) {
	now := time.Now().UTC()
	service := NewService(Options{})
	service.repo = newMemoryRepository()
	for _, item := range []*programsv1.Program{
		{Id: "agent", SessionId: "s", Provenance: programsv1.Provenance_PROVENANCE_AGENT, CreatedAt: now.Add(-time.Hour).Format(time.RFC3339Nano)},
		{Id: "operator", SessionId: "s", Provenance: programsv1.Provenance_PROVENANCE_OPERATOR, CreatedAt: now.Format(time.RFC3339Nano)},
	} {
		if err := service.repo.Save(context.Background(), item); err != nil {
			t.Fatal(err)
		}
	}
	got := service.ListFiltered(context.Background(), "s", true, "PROVENANCE_OPERATOR", now.Add(-time.Minute), time.Time{}, 0)
	if len(got) != 1 || got[0].GetId() != "operator" {
		t.Fatalf("filtered=%v", got)
	}
}

func TestPortfolioStatsReportsUnavailableContractIndex(t *testing.T) { // [REQ:PRT-P1-006]
	service := NewService(Options{})
	service.repo = newMemoryRepository()
	require := &programsv1.Program{Id: "named", ProgramName: "demo.program", ProgramDigest: "digest", Status: programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, Provenance: programsv1.Provenance_PROVENANCE_AGENT, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	if err := service.repo.Save(context.Background(), require); err != nil {
		t.Fatal(err)
	}
	response, err := service.PortfolioStats(context.Background(), &programsv1.PortfolioStatsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if response.GetContractIndexReason() == "" || response.GetRows()[0].GetDeclaredWallMillis() != 0 || len(response.GetNeverExecuted()) != 0 {
		t.Fatalf("response=%v", response)
	}
}

func TestProtectedNameMisuseGetsTypedFailureCause(t *testing.T) {
	service := NewService(Options{
		Preflight: func(string) []*programsv1.Diagnostic {
			return []*programsv1.Diagnostic{{Severity: "error", Message: "import \"vrooli\" is unavailable"}}
		},
	})
	program, err := service.Submit(context.Background(), "s", "import vrooli", programsv1.Provenance_PROVENANCE_AGENT, false)
	if err != nil {
		t.Fatal(err)
	}
	if program.GetFailureCause() != programsv1.FailureCause_FAILURE_CAUSE_PROTECTED_NAME_MISUSE {
		t.Fatalf("cause=%s", program.GetFailureCause())
	}
}

func TestMiningExcludesOperatorProgramsByDefault(t *testing.T) { // [REQ:PRT-P1-008]
	s := NewService(Options{Runner: fakeRunner{err: errors.New("same")}})
	_, _ = s.Submit(context.Background(), "s1", "x", programsv1.Provenance_PROVENANCE_OPERATOR, false)
	_, _ = s.Submit(context.Background(), "s1", "x", programsv1.Provenance_PROVENANCE_AGENT, false)
	shapes := s.MineFailures(context.Background(), false)
	if len(shapes) != 1 || shapes[0].Count != 1 {
		t.Fatalf("shapes=%v", shapes)
	}
}

func TestDeadlineFailureUsesStableFailureShape(t *testing.T) { // [REQ:PRT-P1-005]
	s := NewService(Options{Runner: fakeRunner{err: &DeadlineExceededError{Limit: 2 * time.Second}}})
	p, err := s.Submit(context.Background(), "s1", "while True: pass", programsv1.Provenance_PROVENANCE_AGENT, false)
	if err != nil {
		t.Fatal(err)
	}
	if p.Status != programsv1.ProgramStatus_PROGRAM_STATUS_FAILED || p.FailureShape != "deadline_exceeded" || !strings.Contains(p.FailureDetail, "2s") {
		t.Fatalf("program=%+v", p)
	}
}
