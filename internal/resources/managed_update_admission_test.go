package resources

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/capacity"
)

func TestManagedUpdateAdmissionReplayDoesNotNeedCurrentSourceOrAuthority(t *testing.T) {
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", filepath.Join(t.TempDir(), "state"))
	u := updateTestInitial(t)
	store, err := newManagedUpdateStore(u.Resource)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Create(context.Background(), u, false); err != nil {
		t.Fatal(err)
	}
	c := NewController(t.TempDir(), t.TempDir())
	opts := ManagedServiceUpdateAdmission{RequestID: u.RequestID, ExpectedVersion: u.Target.Artifact.Version, ExpectedSHA256: u.Target.Artifact.SHA256}
	replayed, reused, err := c.AdmitManagedServiceUpdate(context.Background(), u.Resource, opts)
	if err != nil || !reused || replayed.ID != u.ID || replayed.Revision != u.Revision || replayed.Candidate.Intent != nil {
		t.Fatalf("replay changed or redispatched an admitted operation: %+v %t %v", replayed, reused, err)
	}
	opts.ExpectedSHA256 = updateTestDigest("different-release")
	if _, _, err := c.AdmitManagedServiceUpdate(context.Background(), u.Resource, opts); !errors.Is(err, ErrManagedUpdateConflict) {
		t.Fatalf("request reused for a different release: %v", err)
	}
	current, err := LookupManagedServiceUpdate(u.Resource, u.RequestID)
	if err != nil || current.Revision != u.Revision || current.Phase != UpdatePrepared {
		t.Fatalf("rejected replay changed checkpoint: %+v %v", current, err)
	}
}

func TestManagedUpdateAdmissionCapacityBudgetCannotChange(t *testing.T) {
	u := updateTestInitial(t)
	budget := managedUpdateCapacityBudget(u)
	budget.LedgerPath = capacity.LedgerReference(filepath.Join(t.TempDir(), "capacity.db"))
	u.Capacity = &budget
	s := updateTestStore(t, u)
	if _, _, err := s.Create(context.Background(), u, false); err != nil {
		t.Fatal(err)
	}
	changed := updateTestClone(t, u)
	changed.Capacity = nil
	if _, _, err := s.Create(context.Background(), changed, false); !errors.Is(err, ErrManagedUpdateConflict) {
		t.Fatalf("admission replay removed required capacity budget: %v", err)
	}
	for _, change := range []func(*ManagedUpdateCapacityBudget){func(b *ManagedUpdateCapacityBudget) { b.OwnerID = "foreign" }, func(b *ManagedUpdateCapacityBudget) { b.InstanceID = "foreign" }, func(b *ManagedUpdateCapacityBudget) { b.AmountBytes-- }} {
		changed = updateTestClone(t, u)
		change(changed.Capacity)
		if err := changed.Validate(); err == nil {
			t.Fatal("checkpoint accepted mismatched operation capacity budget")
		}
	}
}
