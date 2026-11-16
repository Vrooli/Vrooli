package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	coredb "github.com/vrooli/api-core/database"
	testdb "github.com/vrooli/api-core/databasetest"
)

type blockingCandidateRunner struct {
	started chan struct{}
	release chan struct{}
	runs    atomic.Int32
}

type countingCandidateRunner struct{ runs atomic.Int32 }

func (r *countingCandidateRunner) Run(context.Context, CommandRunRequest) (CommandRunResult, error) {
	r.runs.Add(1)
	return CommandRunResult{Stdout: "checked"}, nil
}

func (r *blockingCandidateRunner) Run(ctx context.Context, _ CommandRunRequest) (CommandRunResult, error) {
	r.runs.Add(1)
	select {
	case r.started <- struct{}{}:
	default:
	}
	select {
	case <-ctx.Done():
		return CommandRunResult{}, ctx.Err()
	case <-r.release:
		return CommandRunResult{Stdout: "checked"}, nil
	}
}

func TestCandidateCheckJoinsAndReusesDurableReceipt(t *testing.T) {
	runner := &blockingCandidateRunner{started: make(chan struct{}, 2), release: make(chan struct{})}
	db := testdb.OpenSQLiteMemory(t)
	if err := ensureRepoSchema(db); err != nil {
		t.Fatalf("ensure repo schema: %v", err)
	}
	svc := NewPrecommitServiceWithRunner(db, runner)
	repo := t.TempDir()
	if _, err := svc.Save(authorizedHumanContext(), repo, PrecommitConfig{Enabled: true, Command: "check", RunBeforeCommit: true}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	git := NewFakeGitRunner().AddStagedFile("fixture.go")
	first, err := svc.StartCandidateCheck(context.Background(), repo, git, false)
	if err != nil || first.State != "running" || first.JobID == "" {
		t.Fatalf("first start = %#v, %v", first, err)
	}
	select {
	case <-runner.started:
	case <-time.After(time.Second):
		t.Fatal("check did not start")
	}
	joined, err := svc.StartCandidateCheck(context.Background(), repo, git, false)
	if err != nil || joined.JobID != first.JobID || joined.CandidateDigest != first.CandidateDigest {
		t.Fatalf("repeat start did not join: first=%#v joined=%#v err=%v", first, joined, err)
	}
	if got := runner.runs.Load(); got != 1 {
		t.Fatalf("runner started %d times, want once", got)
	}
	close(runner.release)
	deadline := time.Now().Add(2 * time.Second)
	var receipt CandidateCheck
	for time.Now().Before(deadline) {
		receipt, err = svc.GetCandidateCheck(context.Background(), repo, first.JobID)
		if err == nil && receipt.State != "running" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil || receipt.State != "passed" || receipt.Result == nil || receipt.Result.Stdout != "checked" {
		t.Fatalf("durable result = %#v, %v", receipt, err)
	}
	reused, err := svc.StartCandidateCheck(context.Background(), repo, git, false)
	if err != nil || reused.JobID != first.JobID || reused.State != "passed" || runner.runs.Load() != 1 {
		t.Fatalf("matching receipt not reused: %#v err=%v runs=%d", reused, err, runner.runs.Load())
	}
	// A freshly constructed service instance reads the persisted terminal receipt
	// after a UI reload or owner restart without invoking the check a second time.
	reloaded := NewPrecommitServiceWithRunner(db, runner)
	afterReload, err := reloaded.StartCandidateCheck(context.Background(), repo, git, false)
	if err != nil || afterReload.JobID != first.JobID || afterReload.State != "passed" || runner.runs.Load() != 1 {
		t.Fatalf("receipt did not survive service reload: %#v err=%v runs=%d", afterReload, err, runner.runs.Load())
	}
	git.Staged["fixture.go"] = "changed after receipt"
	fresh, err := svc.StartCandidateCheck(context.Background(), repo, git, false)
	if err != nil || fresh.JobID == first.JobID || fresh.CandidateDigest == first.CandidateDigest {
		t.Fatalf("changed index reused old identity: old=%#v fresh=%#v err=%v", first, fresh, err)
	}
}

func TestCandidateCheckConcurrentClientsJoinOneLiveWorker(t *testing.T) {
	runner := &blockingCandidateRunner{started: make(chan struct{}, 4), release: make(chan struct{})}
	db := testdb.OpenSQLiteMemory(t)
	if err := ensureRepoSchema(db); err != nil {
		t.Fatalf("ensure repo schema: %v", err)
	}
	svc := NewPrecommitServiceWithRunner(db, runner)
	repo := t.TempDir()
	if _, err := svc.Save(authorizedHumanContext(), repo, PrecommitConfig{Enabled: true, Command: "check", RunBeforeCommit: true}); err != nil {
		t.Fatalf("save config: %v", err)
	}
	git := NewFakeGitRunner().AddStagedFile("odd path/line\nbreak.txt")
	const clients = 12
	results := make(chan CandidateCheck, clients)
	errors := make(chan error, clients)
	var ready sync.WaitGroup
	ready.Add(clients)
	start := make(chan struct{})
	for range clients {
		go func() {
			ready.Done()
			<-start
			check, err := svc.StartCandidateCheck(context.Background(), repo, git, false)
			if err != nil {
				errors <- err
				return
			}
			results <- check
		}()
	}
	ready.Wait()
	close(start)
	first := <-results
	for range clients - 1 {
		select {
		case err := <-errors:
			t.Fatalf("parallel client start failed: %v", err)
		case check := <-results:
			if check.JobID != first.JobID || check.CandidateDigest != first.CandidateDigest {
				t.Fatalf("clients did not join same identity: first=%#v next=%#v", first, check)
			}
		}
	}
	if first.State != "running" || runner.runs.Load() != 1 {
		t.Fatalf("first joined result=%#v worker calls=%d", first, runner.runs.Load())
	}
	select {
	case <-runner.started:
	case <-time.After(time.Second):
		t.Fatal("joined worker did not start")
	}
	close(runner.release)
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		check, err := svc.GetCandidateCheck(context.Background(), repo, first.JobID)
		if err == nil && check.State == "passed" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("joined worker did not persist its terminal receipt")
}

func TestCandidateWorkerCapacityUsesValidatedEnvironmentSetting(t *testing.T) {
	for _, test := range []struct {
		value string
		want  int
	}{
		{value: "1", want: 1},
		{value: "8", want: 8},
		{value: "0", want: defaultCandidateWorkers},
		{value: "17", want: defaultCandidateWorkers},
		{value: "invalid", want: defaultCandidateWorkers},
	} {
		t.Run(test.value, func(t *testing.T) {
			t.Setenv("GCT_PRECOMMIT_MAX_WORKERS", test.value)
			if got := NewPrecommitServiceWithRunner(nil, nil).candidateMaxWork; got != test.want {
				t.Fatalf("configured capacity = %d, want %d", got, test.want)
			}
		})
	}
}

func TestCandidateCheckBackgroundWritesStayOnPinnedTestPool(t *testing.T) {
	primaryPath := filepath.Join(t.TempDir(), "primary.db")
	testPath := filepath.Join(t.TempDir(), "leased.db")
	routed, err := coredb.Open(context.Background(), coredb.Config{
		Driver: coredb.DriverSQLite, TestDriver: coredb.DriverSQLite,
		DSN: primaryPath, MaxOpenConns: 1, MaxIdleConns: 1,
	})
	if err != nil {
		t.Fatalf("open routed DB: %v", err)
	}
	t.Cleanup(func() { _ = routed.Close() })
	if err := ensureRepoSchema(routed.Primary()); err != nil {
		t.Fatalf("ensure primary schema: %v", err)
	}
	routed.SetTestPoolInitializer(func(_ context.Context, pool *sql.DB) error {
		return ensureRepoSchema(pool)
	})
	const leaseID = "candidate-check-pinned-test-pool"
	if err := routed.InstallTestPool(context.Background(), testPath, leaseID, time.Minute); err != nil {
		t.Fatalf("install test DB: %v", err)
	}
	t.Cleanup(func() { _ = routed.ClearTestPool(leaseID) })
	ctx, err := routed.PinTestPool(coredb.WithTestMode(authorizedHumanContext()), leaseID)
	if err != nil {
		t.Fatalf("pin test DB: %v", err)
	}

	runner := &countingCandidateRunner{}
	svc := NewPrecommitServiceWithRunner(routed, runner)
	repo := t.TempDir()
	if _, err := svc.Save(ctx, repo, PrecommitConfig{Enabled: true, Command: "check", RunBeforeCommit: true}); err != nil {
		t.Fatalf("save isolated config: %v", err)
	}
	git := NewFakeGitRunner().AddStagedFile("fixture.go")
	check, err := svc.StartCandidateCheck(ctx, repo, git, false)
	if err != nil || check.State != "running" {
		t.Fatalf("StartCandidateCheck = %#v, %v", check, err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		check, err = svc.GetCandidateCheck(ctx, repo, check.JobID)
		if err == nil && check.State != "running" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil || check.State != "passed" {
		t.Fatalf("test pool did not receive terminal receipt: %#v, %v", check, err)
	}
	var primaryRows int
	if err := routed.Primary().QueryRowContext(context.Background(), `SELECT COUNT(*) FROM git_precommit_candidate_checks`).Scan(&primaryRows); err != nil {
		t.Fatalf("count primary receipts: %v", err)
	}
	if primaryRows != 0 {
		t.Fatalf("asynchronous candidate receipt leaked to primary DB: %d rows", primaryRows)
	}
	if runner.runs.Load() != 1 {
		t.Fatalf("runner invoked %d times, want one", runner.runs.Load())
	}
}

func TestCandidateCheckCapacityIsBoundedAndCancellationIsDurable(t *testing.T) {
	runner := &blockingCandidateRunner{started: make(chan struct{}, 4), release: make(chan struct{})}
	db := testdb.OpenSQLiteMemory(t)
	if err := ensureRepoSchema(db); err != nil {
		t.Fatalf("ensure repo schema: %v", err)
	}
	svc := NewPrecommitServiceWithRunner(db, runner)
	svc.candidateMaxWork = 1
	firstRepo, secondRepo := t.TempDir(), t.TempDir()
	for _, repo := range []string{firstRepo, secondRepo} {
		if _, err := svc.Save(authorizedHumanContext(), repo, PrecommitConfig{Enabled: true, Command: "check", RunBeforeCommit: true}); err != nil {
			t.Fatalf("save config for %s: %v", repo, err)
		}
	}
	first, err := svc.StartCandidateCheck(context.Background(), firstRepo, NewFakeGitRunner().AddStagedFile("first.txt"), false)
	if err != nil || first.State != "running" {
		t.Fatalf("first candidate = %#v, %v", first, err)
	}
	select {
	case <-runner.started:
	case <-time.After(time.Second):
		t.Fatal("first bounded worker did not start")
	}
	second, err := svc.StartCandidateCheck(context.Background(), secondRepo, NewFakeGitRunner().AddStagedFile("second.txt"), false)
	if err != nil || second.State != "queued" {
		t.Fatalf("second candidate should queue at worker limit 1: %#v, %v", second, err)
	}
	time.Sleep(30 * time.Millisecond)
	if got := runner.runs.Load(); got != 1 {
		t.Fatalf("worker limit was exceeded: runner started %d checks", got)
	}
	cancelledQueue, err := svc.CancelCandidateCheck(context.Background(), secondRepo, second.JobID)
	if err != nil || cancelledQueue.State != "cancelled" {
		t.Fatalf("queued cancellation = %#v, %v", cancelledQueue, err)
	}
	cancelledActive, err := svc.CancelCandidateCheck(context.Background(), firstRepo, first.JobID)
	if err != nil || cancelledActive.State != "cancelled" || cancelledActive.Result == nil {
		t.Fatalf("active cancellation = %#v, %v", cancelledActive, err)
	}
	reloadedCancelled, err := svc.StartCandidateCheck(context.Background(), firstRepo, NewFakeGitRunner().AddStagedFile("first.txt"), false)
	if err != nil || reloadedCancelled.JobID != first.JobID || reloadedCancelled.State != "cancelled" {
		t.Fatalf("cancelled receipt was not stable across admission refresh: %#v err=%v", reloadedCancelled, err)
	}
	if got := runner.runs.Load(); got != 1 {
		t.Fatalf("cancelling a queued job started extra work: %d runner calls", got)
	}
	var states int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM git_precommit_candidate_checks WHERE state='cancelled'`).Scan(&states); err != nil {
		t.Fatal(err)
	}
	if states != 2 {
		t.Fatalf("durable cancelled receipts = %d, want 2", states)
	}
}

func TestCandidateCheckReclaimsOrphanedRunningReceiptAfterOwnerRestart(t *testing.T) {
	runner := &countingCandidateRunner{}
	db := testdb.OpenSQLiteMemory(t)
	if err := ensureRepoSchema(db); err != nil {
		t.Fatalf("ensure repo schema: %v", err)
	}
	repo := t.TempDir()
	firstOwner := NewPrecommitServiceWithRunner(db, runner)
	if _, err := firstOwner.Save(authorizedHumanContext(), repo, PrecommitConfig{Enabled: true, Command: "check", RunBeforeCommit: true}); err != nil {
		t.Fatalf("save config: %v", err)
	}
	git := NewFakeGitRunner().AddStagedFile("worktree with spaces/line\nbreak.txt")
	cfg, err := firstOwner.Get(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := candidateDigest(context.Background(), git, repo)
	if err != nil {
		t.Fatal(err)
	}
	config, err := precommitConfigDigest(context.Background(), repo, cfg)
	if err != nil {
		t.Fatal(err)
	}
	jobID := candidateJobID(repo, candidate, config)
	stamp := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := db.ExecContext(context.Background(), `INSERT INTO git_precommit_candidate_checks
		(job_id, repo_path, candidate_digest, config_digest, state, owner_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'running', 'stopped-owner', ?, ?)`, jobID, repo, candidate, config, stamp, stamp); err != nil {
		t.Fatalf("seed stale owner receipt: %v", err)
	}

	// A replacement owner has no process-local worker map. It must relaunch the
	// exact candidate identity; the stale durable `running` marker is not a pass.
	restartedOwner := NewPrecommitServiceWithRunner(db, runner)
	check, err := restartedOwner.StartCandidateCheck(context.Background(), repo, git, false)
	if err != nil || check.JobID != jobID || check.State != "running" {
		t.Fatalf("replacement owner did not reclaim identity: %#v err=%v", check, err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		check, err = restartedOwner.GetCandidateCheck(context.Background(), repo, jobID)
		if err == nil && check.State != "running" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil || check.State != "passed" || check.Result == nil || runner.runs.Load() != 1 {
		t.Fatalf("stale worker was not replaced by one terminal pass: %#v err=%v runs=%d", check, err, runner.runs.Load())
	}
}

func TestCandidateReceiptCannotBeReadFromAnotherRepository(t *testing.T) {
	svc := newTestPrecommitService(t)
	repo := t.TempDir()
	if _, err := svc.Save(authorizedHumanContext(), repo, PrecommitConfig{Enabled: true, Command: "true", RunBeforeCommit: true}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	check, err := svc.StartCandidateCheck(context.Background(), repo, NewFakeGitRunner().AddStagedFile("file"), false)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if _, err := svc.GetCandidateCheck(context.Background(), t.TempDir(), check.JobID); err == nil {
		t.Fatal("receipt was readable outside its repository scope")
	}
}

func TestPrecommitConfigFingerprintBindsDeclaredInputs(t *testing.T) {
	repo := t.TempDir()
	base := PrecommitConfig{
		Enabled: true, Command: "true", WorkingDirectory: ".", TimeoutSeconds: 30,
		RunBeforeCommit: true,
	}
	want, err := precommitConfigDigest(context.Background(), repo, base)
	if err != nil {
		t.Fatalf("fingerprint base config: %v", err)
	}
	if got, err := precommitConfigDigest(context.Background(), repo, base); err != nil || got != want {
		t.Fatalf("same declared config produced unstable fingerprint: %q then %q", want, got)
	}
	variants := []struct {
		name   string
		change func(*PrecommitConfig)
	}{
		{"command", func(cfg *PrecommitConfig) { cfg.Command = "false" }},
		{"working directory", func(cfg *PrecommitConfig) { cfg.WorkingDirectory = "subdir" }},
		{"timeout", func(cfg *PrecommitConfig) { cfg.TimeoutSeconds++ }},
	}
	for _, variant := range variants {
		t.Run(variant.name, func(t *testing.T) {
			changed := base
			variant.change(&changed)
			got, err := precommitConfigDigest(context.Background(), repo, changed)
			if err != nil {
				t.Fatalf("fingerprint changed config: %v", err)
			}
			if got == want {
				t.Fatalf("changed %s reused config identity %q", variant.name, got)
			}
		})
	}
}

func TestDeclaredPrecommitInputContentInvalidatesReceiptIdentity(t *testing.T) {
	repo := t.TempDir()
	inputPath := filepath.Join(repo, "tools", "check.sh")
	if err := os.MkdirAll(filepath.Dir(inputPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inputPath, []byte("#!/bin/sh\nexit 0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := PrecommitConfig{Enabled: true, Command: "./tools/check.sh", WorkingDirectory: repo, TimeoutSeconds: 30, RunBeforeCommit: true, DeclaredInputs: []string{"tools/check.sh"}}
	before, err := precommitConfigDigest(context.Background(), repo, cfg)
	if err != nil {
		t.Fatalf("initial fingerprint: %v", err)
	}
	if err := os.WriteFile(inputPath, []byte("#!/bin/sh\nexit 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	after, err := precommitConfigDigest(context.Background(), repo, cfg)
	if err != nil {
		t.Fatalf("updated fingerprint: %v", err)
	}
	if after == before {
		t.Fatal("changed command input reused a pre-change receipt identity")
	}
	if err := os.Remove(inputPath); err != nil {
		t.Fatal(err)
	}
	if _, err := precommitConfigDigest(context.Background(), repo, cfg); err == nil {
		t.Fatal("missing declared check input did not fail closed")
	}

	// Receipt lookup is keyed by the config digest, so changing only a declared
	// rules file must produce a distinct durable job identity for the same index.
	if err := os.WriteFile(inputPath, []byte("rules version one\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	svc := newTestPrecommitServiceWithRunner(t, &countingCandidateRunner{})
	if _, err := svc.Save(authorizedHumanContext(), repo, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	git := NewFakeGitRunner().AddStagedFile("src/main.go")
	first, err := svc.StartCandidateCheck(context.Background(), repo, git, false)
	if err != nil {
		t.Fatalf("start first candidate check: %v", err)
	}
	if err := os.WriteFile(inputPath, []byte("rules version two\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	second, err := svc.StartCandidateCheck(context.Background(), repo, git, false)
	if err != nil {
		t.Fatalf("start candidate check after rules change: %v", err)
	}
	if second.JobID == first.JobID || second.ConfigDigest == first.ConfigDigest || second.CandidateDigest != first.CandidateDigest {
		t.Fatalf("changed declared file reused receipt identity: first=%#v second=%#v", first, second)
	}
}

func TestDeclaredPrecommitInputsRejectPathEscape(t *testing.T) {
	for _, input := range []string{"../outside.sh", "/tmp/outside.sh", "C:/outside.cmd", "dir\\outside.sh"} {
		t.Run(input, func(t *testing.T) {
			if _, err := normalizePrecommitInputs([]string{input}); err == nil {
				t.Fatalf("accepted non-repository input path %q", input)
			}
		})
	}
}

func TestDeclaredPrecommitInputRejectsSymlinkEscapeAndNonRegularFiles(t *testing.T) {
	repo := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.sh")
	if err := os.WriteFile(outside, []byte("#!/bin/sh\nexit 0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(repo, "check.sh")); err != nil {
		t.Skipf("symlink setup unavailable: %v", err)
	}
	if _, err := hashPrecommitInputs(context.Background(), repo, []string{"check.sh"}); err == nil {
		t.Fatal("symlink resolving outside repository was accepted")
	}
	if err := os.Mkdir(filepath.Join(repo, "directory"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := hashPrecommitInputs(context.Background(), repo, []string{"directory"}); err == nil {
		t.Fatal("directory was accepted as a declared file input")
	}
}

func TestDeclaredPrecommitInputReadsStayBounded(t *testing.T) {
	repo := t.TempDir()
	path := filepath.Join(repo, "large.cfg")
	if err := os.WriteFile(path, make([]byte, maxPrecommitDeclaredInputBytes+1), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := hashPrecommitInputs(context.Background(), repo, []string{"large.cfg"}); err == nil {
		t.Fatal("oversize declared input was hashed")
	}
}

func TestNewServiceResumesPersistedRunningCandidateInsteadOfTrustingIt(t *testing.T) {
	db := testdb.OpenSQLiteMemory(t)
	if err := ensureRepoSchema(db); err != nil {
		t.Fatalf("ensure repo schema: %v", err)
	}
	repo := t.TempDir()
	git := NewFakeGitRunner().AddStagedFile("fixture.go")
	old := NewPrecommitServiceWithRunner(db, fakeCommandRunner{stdout: "old process"})
	if _, err := old.Save(authorizedHumanContext(), repo, PrecommitConfig{Enabled: true, Command: "check", RunBeforeCommit: true}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	cfg, err := old.Get(context.Background(), repo)
	if err != nil {
		t.Fatalf("Get config: %v", err)
	}
	candidate, err := candidateDigest(context.Background(), git, repo)
	if err != nil {
		t.Fatalf("candidate digest: %v", err)
	}
	config, err := precommitConfigDigest(context.Background(), repo, cfg)
	if err != nil {
		t.Fatalf("config digest: %v", err)
	}
	jobID := candidateJobID(repo, candidate, config)
	stamp := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := db.ExecContext(context.Background(), `INSERT INTO git_precommit_candidate_checks
		(job_id, repo_path, candidate_digest, config_digest, state, owner_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'running', 'dead-process', ?, ?)`, jobID, repo, candidate, config, stamp, stamp); err != nil {
		t.Fatalf("seed interrupted job: %v", err)
	}
	runner := &countingCandidateRunner{}
	restarted := NewPrecommitServiceWithRunner(db, runner)
	check, err := restarted.StartCandidateCheck(context.Background(), repo, git, false)
	if err != nil || check.JobID != jobID || check.State != "running" {
		t.Fatalf("restart attachment = %#v, %v", check, err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		check, err = restarted.GetCandidateCheck(context.Background(), repo, jobID)
		if err == nil && check.State != "running" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil || check.State != "passed" || check.Result == nil || check.Result.Stdout != "checked" || runner.runs.Load() != 1 {
		t.Fatalf("interrupted job did not recover as a fresh check: %#v err=%v runs=%d", check, err, runner.runs.Load())
	}
}

func TestTerminalFailureCanBeExplicitlyRetriedWithoutDuplicatingNormalAdmission(t *testing.T) {
	db := testdb.OpenSQLiteMemory(t)
	if err := ensureRepoSchema(db); err != nil {
		t.Fatalf("ensure repo schema: %v", err)
	}
	repo := t.TempDir()
	git := NewFakeGitRunner().AddStagedFile("fixture.go")
	setup := NewPrecommitServiceWithRunner(db, fakeCommandRunner{stdout: "setup"})
	if _, err := setup.Save(authorizedHumanContext(), repo, PrecommitConfig{Enabled: true, Command: "check", RunBeforeCommit: true}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	cfg, err := setup.Get(context.Background(), repo)
	if err != nil {
		t.Fatalf("Get config: %v", err)
	}
	candidate, err := candidateDigest(context.Background(), git, repo)
	if err != nil {
		t.Fatalf("candidate digest: %v", err)
	}
	config, err := precommitConfigDigest(context.Background(), repo, cfg)
	if err != nil {
		t.Fatalf("config digest: %v", err)
	}
	jobID := candidateJobID(repo, candidate, config)
	failed, _ := json.Marshal(PrecommitRunResult{Status: "failed", Summary: "scanner unavailable", Timestamp: time.Now().UTC()})
	stamp := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := db.ExecContext(context.Background(), `INSERT INTO git_precommit_candidate_checks
		(job_id, repo_path, candidate_digest, config_digest, state, owner_id, result_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'failed', 'prior-attempt', ?, ?, ?)`, jobID, repo, candidate, config, string(failed), stamp, stamp); err != nil {
		t.Fatalf("seed failed result: %v", err)
	}
	runner := &countingCandidateRunner{}
	svc := NewPrecommitServiceWithRunner(db, runner)
	reused, err := svc.StartCandidateCheck(context.Background(), repo, git, false)
	if err != nil || reused.State != "failed" || reused.JobID != jobID || runner.runs.Load() != 0 {
		t.Fatalf("normal admission did not reuse terminal failure: %#v err=%v runs=%d", reused, err, runner.runs.Load())
	}
	retried, err := svc.StartCandidateCheck(context.Background(), repo, git, true)
	if err != nil || retried.State != "running" || retried.JobID != jobID {
		t.Fatalf("explicit retry did not join a new attempt: %#v err=%v", retried, err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		retried, err = svc.GetCandidateCheck(context.Background(), repo, jobID)
		if err == nil && retried.State != "running" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil || retried.State != "passed" || runner.runs.Load() != 1 {
		t.Fatalf("explicit retry result = %#v err=%v runs=%d", retried, err, runner.runs.Load())
	}
}

func TestConcurrentExplicitRetriesJoinOneNewAttempt(t *testing.T) {
	db := testdb.OpenSQLiteMemory(t)
	if err := ensureRepoSchema(db); err != nil {
		t.Fatalf("ensure repo schema: %v", err)
	}
	repo := t.TempDir()
	git := NewFakeGitRunner().AddStagedFile("fixture.go")
	setup := NewPrecommitServiceWithRunner(db, fakeCommandRunner{stdout: "setup"})
	if _, err := setup.Save(authorizedHumanContext(), repo, PrecommitConfig{Enabled: true, Command: "check", RunBeforeCommit: true}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	cfg, err := setup.Get(context.Background(), repo)
	if err != nil {
		t.Fatalf("Get config: %v", err)
	}
	candidate, err := candidateDigest(context.Background(), git, repo)
	if err != nil {
		t.Fatalf("candidate digest: %v", err)
	}
	config, err := precommitConfigDigest(context.Background(), repo, cfg)
	if err != nil {
		t.Fatalf("config digest: %v", err)
	}
	jobID := candidateJobID(repo, candidate, config)
	failed, _ := json.Marshal(PrecommitRunResult{Status: "failed", Summary: "prior failure", Timestamp: time.Now().UTC()})
	stamp := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := db.ExecContext(context.Background(), `INSERT INTO git_precommit_candidate_checks
		(job_id, repo_path, candidate_digest, config_digest, state, owner_id, result_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'failed', 'prior-attempt', ?, ?, ?)`, jobID, repo, candidate, config, string(failed), stamp, stamp); err != nil {
		t.Fatalf("seed failed result: %v", err)
	}
	runner := &blockingCandidateRunner{started: make(chan struct{}, 2), release: make(chan struct{})}
	svc := NewPrecommitServiceWithRunner(db, runner)
	const callers = 12
	checks := make(chan CandidateCheck, callers)
	errs := make(chan error, callers)
	for i := 0; i < callers; i++ {
		go func() {
			check, err := svc.StartCandidateCheck(context.Background(), repo, git, true)
			checks <- check
			errs <- err
		}()
	}
	select {
	case <-runner.started:
	case <-time.After(time.Second):
		t.Fatal("explicit retry did not start")
	}
	for i := 0; i < callers; i++ {
		check, err := <-checks, <-errs
		if err != nil || check.JobID != jobID || check.State != "running" {
			t.Fatalf("concurrent retry %d = %#v, %v", i, check, err)
		}
	}
	if got := runner.runs.Load(); got != 1 {
		t.Fatalf("concurrent retries launched %d attempts, want one", got)
	}
	close(runner.release)
	deadline := time.Now().Add(time.Second)
	var receipt CandidateCheck
	for time.Now().Before(deadline) {
		receipt, err = svc.GetCandidateCheck(context.Background(), repo, jobID)
		if err == nil && receipt.State != "running" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil || receipt.State != "passed" || runner.runs.Load() != 1 {
		t.Fatalf("joined retry result = %#v, %v (runs=%d)", receipt, err, runner.runs.Load())
	}
}

func TestFinalAdmissionRerunsWhenStageChangesDuringCheck(t *testing.T) {
	runner := &blockingCandidateRunner{started: make(chan struct{}, 3), release: make(chan struct{})}
	svc := newTestPrecommitServiceWithRunner(t, runner)
	repo := t.TempDir()
	if _, err := svc.Save(authorizedHumanContext(), repo, PrecommitConfig{Enabled: true, Command: "check", RunBeforeCommit: true}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	git := NewFakeGitRunner().AddStagedFile("fixture.go")
	type answer struct {
		result PrecommitRunResult
		err    error
	}
	done := make(chan answer, 1)
	go func() {
		result, err := svc.AwaitCandidateCheck(context.Background(), repo, git)
		done <- answer{result: result, err: err}
	}()
	select {
	case <-runner.started:
	case <-time.After(time.Second):
		t.Fatal("first check did not start")
	}
	git.Staged["fixture.go"] = "concurrent stage change"
	close(runner.release)
	select {
	case got := <-done:
		if got.err != nil || got.result.Status != "passed" {
			t.Fatalf("final admission = %#v, %v", got.result, got.err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("final admission did not settle")
	}
	if got := runner.runs.Load(); got != 2 {
		t.Fatalf("stage change should force a fresh check, ran %d times", got)
	}
}

func TestCandidateReceiptQualifiesRealTemporaryGitCommit(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo with spaces")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	runGit("init", "-q")
	runGit("config", "user.name", "RH3 Fixture")
	runGit("config", "user.email", "rh3@example.invalid")
	if err := os.WriteFile(filepath.Join(repo, "sample.txt"), []byte("candidate contents\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit("add", "--", "sample.txt")
	counterPath := filepath.Join(t.TempDir(), "check-runs")
	command := "printf x >> " + strconv.Quote(counterPath)
	svc := newTestPrecommitServiceWithRunner(t, ShellCommandRunner{})
	if _, err := svc.Save(authorizedHumanContext(), repo, PrecommitConfig{Enabled: true, Command: command, RunBeforeCommit: true}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := InstallHook(authorizedHumanContext(), repo, command); err != nil {
		t.Fatalf("InstallHook: %v", err)
	}
	hookPath := filepath.Join(repo, ".git", "hooks", "pre-commit")
	git := &ExecGitRunner{}
	initialTree, err := git.WriteTree(context.Background(), repo)
	if err != nil {
		t.Fatalf("initial WriteTree: %v", err)
	}
	runHook := func(expectedTree string) {
		t.Helper()
		cmd := exec.Command("/bin/sh", hookPath)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), "VROOLI_GCT_PRECOMMIT_TREE="+expectedTree)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("managed hook: %v\n%s", err, out)
		}
	}
	runHook(initialTree)
	if _, err := os.Stat(counterPath); !os.IsNotExist(err) {
		t.Fatalf("matching candidate repeated the check: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "race.txt"), []byte("new stage\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit("add", "--", "race.txt")
	runHook(initialTree)
	if out, err := os.ReadFile(counterPath); err != nil || string(out) != "x" {
		t.Fatalf("stale receipt did not run a fresh hook check: %v", err)
	}
	if err := os.Remove(counterPath); err != nil {
		t.Fatal(err)
	}
	result, err := CreateCommit(authorizedHumanContext(), CommitDeps{
		Git: git, RepoDir: repo, Precommit: svc,
	}, CommitRequest{Message: "test: qualify durable candidate receipt"})
	if err != nil || result == nil || !result.Success || result.Hash == "" {
		t.Fatalf("CreateCommit = %#v, %v", result, err)
	}
	rows, err := svc.db.QueryContext(context.Background(), `SELECT state, result_json FROM git_precommit_candidate_checks WHERE repo_path = ?`, repo)
	if err != nil {
		t.Fatalf("read candidate receipt: %v", err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatal("successful commit did not leave a durable candidate receipt")
	}
	var state, encoded string
	if err := rows.Scan(&state, &encoded); err != nil || state != "passed" || encoded == "" {
		t.Fatalf("receipt = state %q result %q err %v", state, encoded, err)
	}
	if rows.Next() {
		t.Fatal("expected exactly one receipt for the candidate identity")
	}
	if out, err := os.ReadFile(counterPath); err != nil || string(out) != "x" {
		t.Fatalf("matching managed hook repeated the actual check: contents %q err %v", out, err)
	}
}
