package resources

import (
	"context"
	"errors"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"os"
	"strings"
	"testing"
	"time"
)

func TestOpenRGBUpdateDisabledRequiresObservedRetirement(t *testing.T) {
	u := updateTestInitial(t)
	u.Resource = openRGBResourceName
	for _, c := range []*ManagedUpdateContract{&u.Target, &u.Rollback} {
		c.Manifest.Name = openRGBResourceName
		c.Artifact.Resource = openRGBResourceName
		c.Digest = c.digest()
	}
	if _, err := transitionManagedUpdate(u, UpdateFinishDisabled, u.UpdatedAt.Add(time.Second)); err == nil {
		t.Fatal("event manufactured retirement evidence")
	}
	u.PrimaryStopVerified, u.PrimaryQuiescent, u.CandidateQuiescent = true, true, true
	got, err := transitionManagedUpdate(u, UpdateFinishDisabled, u.UpdatedAt.Add(time.Second))
	if err != nil || got.Phase != UpdateResourceDisabled || !got.Phase.Terminal() || !got.AbortRequested {
		t.Fatalf("disabled closure: %s %v", got.Phase, err)
	}
	unrelated := updateTestInitial(t)
	unrelated.AbortRequested, unrelated.PrimaryStopVerified, unrelated.PrimaryQuiescent, unrelated.CandidateQuiescent = true, true, true, true
	if _, err := transitionManagedUpdate(unrelated, UpdateFinishDisabled, unrelated.UpdatedAt.Add(time.Second)); err == nil {
		t.Fatal("unrelated resource accepted OpenRGB disable")
	}
}

func TestOpenRGBUpdateDisableDoesNotCommitWithForeignRestoredListener(t *testing.T) {
	u := updateTestInitial(t)
	u.Resource = openRGBResourceName
	for _, c := range []*ManagedUpdateContract{&u.Target, &u.Rollback} {
		c.Manifest.Name = openRGBResourceName
		c.Artifact.Resource = openRGBResourceName
		c.Digest = c.digest()
	}
	u.OriginalScope = updateOriginalScopeFixture(u) // injected stopper supplies exact receipt
	u.Restored.Intent = updateTestIntent("restored", u.Rollback)
	ledger := managedUpdateLedger{Records: []managedUpdateEntry{{State: u}}}
	committed := false
	cleanupCalls := 0
	cleanup := func(context.Context, ManagedServiceUpdate, ManagedUpdateProcess) error { cleanupCalls++; return nil }
	stop := func(context.Context, ManagedServiceUpdate) (managedUpdateOriginalStopReceipt, error) {
		return managedUpdateOriginalStopReceipt{RootGone: true, TreeQuiescent: true}, nil
	}
	foreign := errors.New("foreign listener")
	finish := func(_ context.Context, got ManagedServiceUpdate) error {
		committed = true
		if !freshUpdateQuiescence(got.Restored, time.Now().UTC()) {
			t.Fatal("missing observed restored receipt")
		}
		return nil
	}
	err := retireOpenRGBUpdateRecords(context.Background(), ledger, func(context.Context, ManagedServiceUpdate, ManagedUpdateProcess) error { return nil }, cleanup, stop, func(int) error { return foreign }, finish)
	if !errors.Is(err, foreign) || committed {
		t.Fatalf("foreign listener was committed: %v %v", err, committed)
	}
	err = retireOpenRGBUpdateRecords(context.Background(), ledger, func(context.Context, ManagedServiceUpdate, ManagedUpdateProcess) error { return nil }, cleanup, stop, func(port int) error {
		if port != u.Restored.Intent.Runtime.Port {
			t.Fatal("wrong port")
		}
		return nil
	}, finish)
	if err != nil || !committed || cleanupCalls != 8 {
		t.Fatalf("retry retirement: %v committed=%v calls=%d", err, committed, cleanupCalls)
	}
}

func TestOpenRGBUpdateDelayedWitnessCannotExecAfterDisableRevocation(t *testing.T) {
	x := newOpenRGBExecFixture(t)
	if _, err := x.store.Prepare(t.Context(), x.fence); err != nil {
		t.Fatal(err)
	}
	observed, resume := make(chan struct{}), make(chan struct{})
	access := x.access
	x.access = func() (hostreqkit.OpenRGBLinuxEvidence, error) { close(observed); <-resume; return access() }
	done := make(chan error, 1)
	go func() { done <- x.run() }()
	<-observed
	if _, err := x.store.Revoke(t.Context(), x.fence); err != nil {
		close(resume)
		t.Fatal(err)
	}
	close(resume)
	if err := <-done; !errors.Is(err, ErrManagedUpdateExecRevoked) {
		t.Fatalf("delayed witness advanced after opt-out: %v", err)
	}
	if x.replaced != 0 {
		t.Fatal("payload executed after revocation")
	}
	invocation, err := renderManagedUpdateInvocation(x.update.Rollback, x.fence.Intent.Runtime, x.update.ID, x.fence.Intent.InstanceID, "fixture-token")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(invocation.ConfigPath); !os.IsNotExist(err) {
		t.Fatalf("configuration materialized after revoked claim: %v", err)
	}
}

func TestOpenRGBUpdateDisabledExecStopsBeforeLedgerOrNativeLookup(t *testing.T) {
	c, _, _ := openRGBHeadlessFixture(t)
	if err := c.RunManagedUpdateExec(t.Context(), openRGBResourceName, "absent-request", "absent-role"); err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("disabled wrapper reached ledger/native owner: %v", err)
	}
}

func TestOpenRGBUpdateDisableRevokesAllRolesBeforePartialCleanup(t *testing.T) {
	u := updateTestInitial(t)
	u.Resource = openRGBResourceName
	u.OriginalScope = updateOriginalScopeFixture(u)
	ledger := managedUpdateLedger{Records: []managedUpdateEntry{{State: u}}}
	revoked, cleaned, originalStopped, committed := 0, 0, false, false
	failed := errors.New("first role stop uncertain")
	err := retireOpenRGBUpdateRecords(t.Context(), ledger, func(context.Context, ManagedServiceUpdate, ManagedUpdateProcess) error { revoked++; return nil }, func(context.Context, ManagedServiceUpdate, ManagedUpdateProcess) error {
		if revoked != 4 {
			t.Fatal("native stop preceded complete wrapper revocation")
		}
		cleaned++
		if cleaned == 1 {
			return failed
		}
		return nil
	}, func(context.Context, ManagedServiceUpdate) (managedUpdateOriginalStopReceipt, error) {
		originalStopped = true
		return managedUpdateOriginalStopReceipt{RootGone: true, TreeQuiescent: true}, nil
	}, func(int) error { return nil }, func(context.Context, ManagedServiceUpdate) error { committed = true; return nil })
	if !errors.Is(err, failed) || cleaned != 4 || !originalStopped || committed {
		t.Fatalf("partial cleanup skipped owners or committed: %v cleaned=%d original=%v committed=%v", err, cleaned, originalStopped, committed)
	}
}

func TestOpenRGBUpdateDisablePreservesCombinedRetirementFailures(t *testing.T) {
	u := updateTestInitial(t)
	u.Resource = openRGBResourceName
	u.OriginalScope = updateOriginalScopeFixture(u)
	ledger := managedUpdateLedger{Records: []managedUpdateEntry{{State: u}}}
	revokeErr, cleanupErr, originalErr := errors.New("uncertain revoke persistence"), errors.New("role stop failed"), errors.New("original stop failed")
	revokes, cleanups := 0, 0
	err := retireOpenRGBUpdateRecords(t.Context(), ledger, func(context.Context, ManagedServiceUpdate, ManagedUpdateProcess) error {
		revokes++
		if revokes == 1 {
			return revokeErr
		}
		return nil
	}, func(context.Context, ManagedServiceUpdate, ManagedUpdateProcess) error {
		cleanups++
		if cleanups == 1 {
			return cleanupErr
		}
		return nil
	}, func(context.Context, ManagedServiceUpdate) (managedUpdateOriginalStopReceipt, error) {
		return managedUpdateOriginalStopReceipt{}, originalErr
	}, func(int) error { t.Fatal("unretired role reached listener publication"); return nil }, func(context.Context, ManagedServiceUpdate) error {
		t.Fatal("partial cleanup reached terminal publication")
		return nil
	})
	if !errors.Is(err, revokeErr) || !errors.Is(err, cleanupErr) || !errors.Is(err, originalErr) || revokes != 4 || cleanups != 4 {
		t.Fatalf("retirement evidence discarded: %v revokes=%d cleanups=%d", err, revokes, cleanups)
	}
}

func TestOpenRGBRetirementObservationCannotChangeImmutableAdmission(t *testing.T) {
	x := newOpenRGBExecFixture(t)
	old := x.update
	good := updateTestClone(t, old)
	good.PrimaryStopVerified, good.PrimaryQuiescent, good.CandidateQuiescent = true, true, true
	good.Restored.Quiescence = &ManagedUpdateQuiescence{InstanceID: good.Restored.Intent.InstanceID, OwnershipTokenHash: good.Restored.Intent.OwnershipTokenHash, ObservedAt: time.Now().UTC(), NativeAbsent: true, ProcessesAbsent: true, ListenerAbsent: true}
	if err := validOpenRGBUpdateRetirementObservation(old, good); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*ManagedServiceUpdate){
		func(u *ManagedServiceUpdate) { u.AbortRequested = !u.AbortRequested },
		func(u *ManagedServiceUpdate) { u.SourceRevision = updateTestDigest("different source") },
		func(u *ManagedServiceUpdate) { u.Target.Artifact.Path += ".changed" },
		func(u *ManagedServiceUpdate) { u.Restored.Intent.InstanceID = "different-native" },
		func(u *ManagedServiceUpdate) { u.Phase = UpdateRestoring },
	} {
		changed := updateTestClone(t, good)
		change(&changed)
		if err := validOpenRGBUpdateRetirementObservation(old, changed); err == nil {
			t.Fatal("retirement changed immutable admitted inputs")
		}
	}
	store, err := newManagedUpdateStore(openRGBResourceName)
	if err != nil {
		t.Fatal(err)
	}
	// Seed an already admitted restoration checkpoint, as loaded after caller loss.
	if err := store.save(managedUpdateLedger{SchemaVersion: 1, Resource: old.Resource, Records: []managedUpdateEntry{{State: old}}}); err != nil {
		t.Fatal(err)
	}
	result, err := store.Commit(t.Context(), good, UpdateFinishDisabled)
	if err != nil || result.Phase != UpdateResourceDisabled || !result.AbortRequested {
		t.Fatalf("real store refused observed retirement: %+v %v", result, err)
	}
}
