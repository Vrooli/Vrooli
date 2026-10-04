// Responsibility: retain repository test declarations within their original package.
package database

import (
	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	_ "modernc.org/sqlite"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestTouchHeartbeat_StatusGuarded verifies the status-guarded heartbeat update:
// it bumps last_heartbeat for running/starting runs, but is a no-op (no clobber)
// for a parked or terminal run — so a heartbeat racing a park/stop transition
// can never resurrect the run.
func TestFreshRecoveryClaimFencesLifecycleWriters(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	db.DB.SetMaxOpenConns(1)
	repos := NewRepositories(db, logrus.New())
	claims := repos.Runs.(repository.RunFreshRecoveryClaimer)
	ctx := t.Context()
	newSource := func() *domain.Run {
		run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusFailed, RunMode: domain.RunModeInPlace}
		if err := repos.Runs.Create(ctx, run); err != nil {
			t.Fatal(err)
		}
		return run
	}
	t.Run("claim wins against stale and newly read continuation", func(t *testing.T) {
		source := newSource()
		hash := strings.Repeat("a", 64)
		if won, err := claims.ClaimFreshRecovery(ctx, source.ID, source.LifecycleVersion+1, hash, false); err != nil || won {
			t.Fatalf("stale version won: %v %v", won, err)
		}
		if won, err := claims.ClaimFreshRecovery(ctx, source.ID, source.LifecycleVersion, hash, false); err != nil || !won {
			t.Fatalf("claim failed: %v %v", won, err)
		}
		source.Status = domain.RunStatusRunning
		if err := repos.Runs.Update(ctx, source); err == nil {
			t.Fatal("stale continuation reactivated claimed source")
		}
		current, err := repos.Runs.Get(ctx, source.ID)
		if err != nil {
			t.Fatal(err)
		}
		if current.LifecycleVersion != 1 || current.Status != domain.RunStatusFailed {
			t.Fatalf("claim lost source lifecycle: %+v", current)
		}
		current.Status = domain.RunStatusRunning
		if err := repos.Runs.Update(ctx, current); err == nil {
			t.Fatal("newly read continuation reactivated claimed source")
		}
		if got, err := claims.GetFreshRecoveryClaim(ctx, source.ID); err != nil || got != hash {
			t.Fatalf("claim was not durable: %q %v", got, err)
		}
	})
	t.Run("continuation wins", func(t *testing.T) {
		source := newSource()
		version := source.LifecycleVersion
		source.Status = domain.RunStatusRunning
		if err := repos.Runs.Update(ctx, source); err != nil {
			t.Fatal(err)
		}
		if won, err := claims.ClaimFreshRecovery(ctx, source.ID, version, strings.Repeat("b", 64), false); err != nil || won {
			t.Fatalf("recovery overtook continuation: %v %v", won, err)
		}
	})
	t.Run("independent claim writers have one winner", func(t *testing.T) {
		source := newSource()
		var wg sync.WaitGroup
		wins := make(chan bool, 8)
		for range 8 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				other := NewRepositories(db, logrus.New()).Runs.(repository.RunFreshRecoveryClaimer)
				won, err := other.ClaimFreshRecovery(ctx, source.ID, source.LifecycleVersion, strings.Repeat("c", 64), false)
				if err != nil {
					t.Errorf("claim: %v", err)
				}
				wins <- won
			}()
		}
		wg.Wait()
		close(wins)
		count := 0
		for won := range wins {
			if won {
				count++
			}
		}
		if count != 1 {
			t.Fatalf("claim winners=%d, want 1", count)
		}
	})
}

func TestFreshRecoveryClaimRejectsAutomaticCancellation(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, logrus.New())
	claims := repos.Runs.(repository.RunFreshRecoveryClaimer)
	for _, status := range []domain.RunStatus{domain.RunStatusFailed, domain.RunStatusCancelled} {
		run := &domain.Run{ID: uuid.New(), Status: status, RunMode: domain.RunModeInPlace}
		now := time.Now()
		run.CancelRequestedAt = &now
		if err := repos.Runs.Create(t.Context(), run); err != nil {
			t.Fatal(err)
		}
		if won, err := claims.ClaimFreshRecovery(t.Context(), run.ID, run.LifecycleVersion, strings.Repeat("d", 64), false); err != nil || won {
			t.Fatalf("automatic cancellation recovery accepted: %v %v", won, err)
		}
		if won, err := claims.ClaimFreshRecovery(t.Context(), run.ID, run.LifecycleVersion, strings.Repeat("d", 64), true); err != nil || !won {
			t.Fatalf("explicit recovery refused: %v %v", won, err)
		}
	}
}

func TestFreshRecoveryClaimReleaseCannotForgetAcceptance(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, logrus.New())
	claims := repos.Runs.(repository.RunFreshRecoveryClaimer)
	for _, accepted := range []bool{false, true} {
		source := &domain.Run{ID: uuid.New(), Status: domain.RunStatusFailed, RunMode: domain.RunModeInPlace}
		if err := repos.Runs.Create(t.Context(), source); err != nil {
			t.Fatal(err)
		}
		hash := strings.Repeat("e", 64)
		if won, err := claims.ClaimFreshRecovery(t.Context(), source.ID, 0, hash, false); err != nil || !won {
			t.Fatalf("claim: %v %v", won, err)
		}
		if released, err := claims.ReleaseFreshRecoveryClaim(t.Context(), source.ID, 0, hash); err != nil || released {
			t.Fatalf("stale release accepted: %v %v", released, err)
		}
		if released, err := claims.ReleaseFreshRecoveryClaim(t.Context(), source.ID, 1, strings.Repeat("f", 64)); err != nil || released {
			t.Fatalf("different request released claim: %v %v", released, err)
		}
		if accepted {
			replacement := &domain.Run{ID: uuid.New(), Status: domain.RunStatusPending, RunMode: domain.RunModeInPlace, IdempotencyKey: "resume-from-failed:" + source.ID.String(), SourceRunIDs: []uuid.UUID{source.ID}}
			if err := repos.Runs.Create(t.Context(), replacement); err != nil {
				t.Fatal(err)
			}
			if err := repos.Runs.Delete(t.Context(), replacement.ID); err != nil {
				t.Fatal(err)
			}
		}
		released, err := claims.ReleaseFreshRecoveryClaim(t.Context(), source.ID, 1, hash)
		if err != nil || released == accepted {
			t.Fatalf("release=%v accepted=%v err=%v", released, accepted, err)
		}
		current, err := repos.Runs.Get(t.Context(), source.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !accepted && current.LifecycleVersion != 2 {
			t.Fatal("release did not fence the old claimant")
		}
	}
}

func TestRecoveryAttachmentRepairsOnlyCurrentTurnMetadata(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, logrus.New())
	ctx := t.Context()
	task := &domain.Task{ID: uuid.New(), Title: "recovery attachment", ScopePath: "/test", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	old, now := time.Now().Add(-time.Hour).UTC(), time.Now().UTC()
	exit := 0
	run := &domain.Run{
		ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusRunning, StartedAt: &old, EndedAt: &old, LastHeartbeat: &old, ExitCode: &exit, ErrorMsg: "old", TerminalClass: domain.RunTerminalClassInterruption, StopReason: domain.RunStopReasonTimeout,
		ProgressPercent: 42, TranscriptCursor: 87, RunnerPID: 12345, SessionID: "retained", Result: &domain.RunResult{FinalOutput: "prior evidence"}, Summary: &domain.RunSummary{TokensUsed: 123},
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	attacher := repos.Runs.(repository.RunRecoveryAttacher)
	if changed, err := attacher.AttachRecovery(ctx, run.ID, run.LifecycleVersion+1, now); err != nil || changed {
		t.Fatalf("wrong lifecycle attached: %v %v", changed, err)
	}
	if changed, err := attacher.AttachRecovery(ctx, run.ID, run.LifecycleVersion, now); err != nil || !changed {
		t.Fatalf("current lifecycle attachment failed: %v %v", changed, err)
	}
	got, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.EndedAt != nil || got.ExitCode != nil || got.ErrorMsg != "" || got.TerminalClass != "" || got.StopReason != "" || got.LastHeartbeat == nil || !got.LastHeartbeat.Equal(now) {
		t.Fatalf("legacy fields not repaired: %+v", got)
	}
	if got.LifecycleVersion != run.LifecycleVersion || got.ProgressPercent != 42 || got.TranscriptCursor != 87 || got.RunnerPID != 12345 || got.SessionID != "retained" || !got.StartedAt.Equal(old) || got.Result.FinalOutput != "prior evidence" || got.Summary.TokensUsed != 123 {
		t.Fatal("attachment altered identity/progress/evidence")
	}
	newer := now.Add(time.Minute)
	if _, err := repos.Runs.TouchHeartbeat(ctx, run.ID, newer); err != nil {
		t.Fatal(err)
	}
	if _, err := attacher.AttachRecovery(ctx, run.ID, run.LifecycleVersion, now); err != nil {
		t.Fatal(err)
	}
	got, _ = repos.Runs.Get(ctx, run.ID)
	if !got.LastHeartbeat.Equal(newer) {
		t.Fatal("attachment regressed newer heartbeat")
	}
	if _, err := repos.Runs.RequestCancellation(ctx, run.ID, now); err != nil {
		t.Fatal(err)
	}
	if changed, err := attacher.AttachRecovery(ctx, run.ID, run.LifecycleVersion, now); err != nil || changed {
		t.Fatalf("cancelled ownership was reattached: %v %v", changed, err)
	}
	got.Status = domain.RunStatusFailed
	if err := repos.Runs.Update(ctx, got); err != nil {
		t.Fatal(err)
	}
	if changed, err := attacher.AttachRecovery(ctx, run.ID, got.LifecycleVersion, now); err != nil || changed {
		t.Fatalf("terminal run reattached: %v %v", changed, err)
	}
}

func TestRecoveryOwnershipEpochFencesReplacedOwnerWrites(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, logrus.New())
	ctx := t.Context()
	run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusRunning, RunMode: domain.RunModeInPlace}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	claimer := repos.Runs.(repository.RunRecoveryOwnerClaimer)
	firstEpoch, claimed, err := claimer.ClaimRecoveryOwnership(ctx, run.ID, run.LifecycleVersion, "owner-a", time.Now().UTC())
	if err != nil || !claimed || firstEpoch != 1 {
		t.Fatalf("first ownership claim = epoch %d claimed %v err %v, want epoch 1", firstEpoch, claimed, err)
	}
	secondEpoch, claimed, err := claimer.ClaimRecoveryOwnership(ctx, run.ID, run.LifecycleVersion, "owner-b", time.Now().UTC())
	if err != nil || !claimed || secondEpoch != 2 {
		t.Fatalf("replacement ownership claim = epoch %d claimed %v err %v, want epoch 2", secondEpoch, claimed, err)
	}
	old, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	old.OwnerIdentity, old.OwnerEpoch, old.RunnerPID = "owner-a", firstEpoch, 111
	if updated, err := repos.Runs.UpdateRunnerStreamState(ctx, old); err != nil || updated {
		t.Fatalf("stale stream update = %v err %v, want rejected", updated, err)
	}
	current, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	current.OwnerIdentity, current.OwnerEpoch, current.RunnerPID = "owner-b", secondEpoch, 222
	if updated, err := repos.Runs.UpdateRunnerStreamState(ctx, current); err != nil || !updated {
		t.Fatalf("current stream update = %v err %v, want accepted", updated, err)
	}
	if got, err := repos.Runs.Get(ctx, run.ID); err != nil || got.OwnerIdentity != "owner-b" || got.OwnerEpoch != secondEpoch || got.RunnerPID != 222 {
		t.Fatalf("persisted owner = %+v err %v", got, err)
	}
}

func TestClearTerminalRunnerProcessIdentityIsGuarded(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, logrus.New())
	ctx := t.Context()
	run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusFailed, RunMode: domain.RunModeInPlace, RunnerPID: 111, RunnerPGID: 222}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	clearer := repos.Runs.(repository.RunRecoveryProcessIdentityClearer)
	if cleared, err := clearer.ClearTerminalRunnerProcessIdentity(ctx, run.ID, run.LifecycleVersion, run.OwnerEpoch, 999, 222); err != nil || cleared {
		t.Fatalf("mismatched identity clear = %v err %v, want rejected", cleared, err)
	}
	if cleared, err := clearer.ClearTerminalRunnerProcessIdentity(ctx, run.ID, run.LifecycleVersion, run.OwnerEpoch, 111, 222); err != nil || !cleared {
		t.Fatalf("exact identity clear = %v err %v, want accepted", cleared, err)
	}
	got, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.RunnerPID != 0 || got.RunnerPGID != 0 {
		t.Fatalf("terminal runner identity remained: pid=%d pgid=%d", got.RunnerPID, got.RunnerPGID)
	}
}

func TestRunLifecycleFencesOldAttemptWriters(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, logrus.New())
	ctx := t.Context()
	task := &domain.Task{ID: uuid.New(), Title: "attempt fence", ScopePath: "/test", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusRunning, RunMode: domain.RunModeInPlace, ApprovalState: domain.ApprovalStateNone}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	old, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	run.Status = domain.RunStatusFailed
	if err := repos.Runs.Update(ctx, run); err != nil {
		t.Fatal(err)
	}
	run.Status = domain.RunStatusRunning
	run.RunnerPID = 12345
	if err := repos.Runs.Update(ctx, run); err != nil {
		t.Fatal(err)
	}
	old.RunnerPID = 54321
	if changed, err := repos.Runs.UpdateRunnerStreamState(ctx, old); err != nil || changed {
		t.Errorf("old stream writer accepted: %v %v", changed, err)
	}
	old.Status = domain.RunStatusFailed
	old.ErrorMsg = "old executor finished"
	if err := repos.Runs.Update(ctx, old); err == nil {
		t.Error("old terminal writer accepted after continuation")
	}
	got, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.RunStatusRunning || got.RunnerPID != 12345 || got.ErrorMsg != "" {
		t.Fatalf("new attempt clobbered: %+v", got)
	}
	// An unrelated metadata write must not erase a newer heartbeat or cancel intent.
	stamp := time.Now().UTC()
	if _, err := repos.Runs.TouchHeartbeat(ctx, run.ID, stamp); err != nil {
		t.Fatal(err)
	}
	if _, err := repos.Runs.RequestCancellation(ctx, run.ID, stamp); err != nil {
		t.Fatal(err)
	}
	run.Label = "metadata"
	if err := repos.Runs.Update(ctx, run); err != nil {
		t.Fatal(err)
	}
	got, _ = repos.Runs.Get(ctx, run.ID)
	if got.LastHeartbeat == nil || got.CancelRequestedAt == nil {
		t.Fatal("full snapshot erased monotonic heartbeat/cancellation")
	}
}
