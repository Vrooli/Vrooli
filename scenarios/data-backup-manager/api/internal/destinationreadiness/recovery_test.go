package destinationreadiness_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"data-backup-manager/internal/destinationreadiness"
)

type recoveryInspector struct {
	identity destinationreadiness.DeviceIdentity
}

func (i recoveryInspector) Inspect(context.Context, string) (destinationreadiness.Inspection, error) {
	return destinationreadiness.Inspection{Identity: i.identity, LocationExists: true, LocationIsDirectory: true, Mounted: true}, nil
}
func (i recoveryInspector) InspectDevice(context.Context, destinationreadiness.DeviceIdentity) (destinationreadiness.Inspection, error) {
	return destinationreadiness.Inspection{Identity: i.identity, LocationExists: true, LocationIsDirectory: true, Mounted: true}, nil
}

type recoveryRemediator struct {
	calls int
	fail  bool
}

func (r *recoveryRemediator) Supported(destinationreadiness.PreparationAction) (bool, string) {
	return true, ""
}
func (r *recoveryRemediator) Remediate(context.Context, destinationreadiness.Plan, bool) (destinationreadiness.RemediationOutcome, error) {
	r.calls++
	if r.fail {
		return destinationreadiness.RemediationOutcome{Status: "refused", RefusalReason: "operator must inspect first"}, errors.New("operator must inspect first")
	}
	return destinationreadiness.RemediationOutcome{Status: "changed", Changed: true}, nil
}

type memoryRecoveryStore struct {
	journal destinationreadiness.RecoveryJournal
	saves   int
}

func (m *memoryRecoveryStore) Load(context.Context, string) (destinationreadiness.RecoveryJournal, error) {
	return m.journal, nil
}
func (m *memoryRecoveryStore) Save(_ context.Context, j destinationreadiness.RecoveryJournal) error {
	m.journal = j
	m.saves++
	return nil
}

func recoveryFixture(t *testing.T) (destinationreadiness.Service, destinationreadiness.RecoveryJournal, *recoveryRemediator) {
	t.Helper()
	identity := destinationreadiness.DeviceIdentity{DevicePath: "/dev/sda1", Filesystem: "ntfs", UUID: "uuid-1", TotalBytes: 100}
	inspector := recoveryInspector{identity: identity}
	remediator := &recoveryRemediator{}
	svc := destinationreadiness.NewService(inspector, nil).WithRemediator(remediator)
	plans := []destinationreadiness.Plan{
		{ID: "p1", Action: destinationreadiness.ActionUnmount, Location: "/media/Elements", TargetPath: "/media/Elements", Identity: identity, RequiresConfirm: true, ConfirmationPhrase: "confirm-unmount", Supported: true},
		{ID: "p2", Action: destinationreadiness.ActionMountReadWrite, Location: "/media/Elements", TargetPath: "/media/Elements", Identity: identity, RequiresConfirm: true, ConfirmationPhrase: "confirm-mount", Supported: true},
	}
	j, err := destinationreadiness.NewRecoveryJournal("/media/Elements", identity, plans, time.Unix(1, 0))
	if err != nil {
		t.Fatal(err)
	}
	return *svc, j, remediator
}

func TestResumeRecoveryPersistsEachStepAndIsIdempotent(t *testing.T) {
	svc, journal, remediator := recoveryFixture(t)
	store := &memoryRecoveryStore{}
	got, err := svc.ResumeRecovery(context.Background(), store, journal, map[int]string{0: "confirm-unmount", 1: "confirm-mount"}, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != destinationreadiness.RecoveryCompleted || got.Current != 2 || remediator.calls != 2 {
		t.Fatalf("journal=%+v calls=%d", got, remediator.calls)
	}
	// A restart/retry loads the completed journal and must not call the host.
	got, err = svc.ResumeRecovery(context.Background(), store, got, nil, false, false)
	if err != nil || remediator.calls != 2 || got.State != destinationreadiness.RecoveryCompleted {
		t.Fatalf("retry journal=%+v calls=%d err=%v", got, remediator.calls, err)
	}
	if store.saves < 5 {
		t.Fatalf("saves=%d, expected durable transitions before/after steps", store.saves)
	}
}

func TestResumeRecoveryPausesWithoutAdvancingOnRefusal(t *testing.T) {
	svc, journal, remediator := recoveryFixture(t)
	remediator.fail = true
	store := &memoryRecoveryStore{}
	got, err := svc.ResumeRecovery(context.Background(), store, journal, map[int]string{0: "confirm-unmount"}, false, false)
	if err == nil || got.State != destinationreadiness.RecoveryPaused || got.Current != 0 || got.Steps[0].Status != "blocked" {
		t.Fatalf("journal=%+v err=%v", got, err)
	}
}

func TestResumeRecoveryDryRunDoesNotAdvanceCursor(t *testing.T) {
	svc, journal, remediator := recoveryFixture(t)
	store := &memoryRecoveryStore{}
	got, err := svc.ResumeRecovery(context.Background(), store, journal, map[int]string{0: "confirm-unmount", 1: "confirm-mount"}, false, true)
	if err != nil || got.Current != 0 || got.State != destinationreadiness.RecoveryPending || remediator.calls != 2 {
		t.Fatalf("journal=%+v calls=%d err=%v", got, remediator.calls, err)
	}
}

func TestResumeRecoveryPausesAfterAmbiguousInFlightAction(t *testing.T) {
	svc, journal, remediator := recoveryFixture(t)
	journal.Steps[0].Status = "in_flight"
	store := &memoryRecoveryStore{}
	got, err := svc.ResumeRecovery(context.Background(), store, journal, map[int]string{0: "confirm-unmount"}, false, false)
	if err == nil || got.State != destinationreadiness.RecoveryPaused || remediator.calls != 0 {
		t.Fatalf("journal=%+v calls=%d err=%v", got, remediator.calls, err)
	}
	if !strings.Contains(got.LastError, "may have completed") {
		t.Fatalf("last error = %q", got.LastError)
	}
}

func TestFileRecoveryStoreUsesAtomicPrivateJournal(t *testing.T) {
	_, journal, _ := recoveryFixture(t)
	root := t.TempDir()
	store := &destinationreadiness.FileRecoveryStore{Root: root}
	if err := store.Save(context.Background(), journal); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load(context.Background(), journal.ID)
	if err != nil || got.ID != journal.ID || got.Version != journal.Version {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	info, err := os.Stat(filepath.Join(root, journal.ID+".json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("journal mode=%o", info.Mode().Perm())
	}
}

func TestFileRecoveryStoreDistinguishesMissingJournal(t *testing.T) {
	store := &destinationreadiness.FileRecoveryStore{Root: t.TempDir()}
	_, err := store.Load(context.Background(), "missing")
	if !errors.Is(err, destinationreadiness.ErrRecoveryNotFound) {
		t.Fatalf("error = %v, want ErrRecoveryNotFound", err)
	}
}

func TestNewRecoveryJournalRejectsRedirectedStep(t *testing.T) {
	identity := destinationreadiness.DeviceIdentity{DevicePath: "/dev/sda1", Filesystem: "ntfs", UUID: "uuid-1", TotalBytes: 100}
	_, err := destinationreadiness.NewRecoveryJournal("/media/Elements", identity, []destinationreadiness.Plan{
		{Action: destinationreadiness.ActionUnmount, Location: "/media/Other", TargetPath: "/media/Other", Identity: identity},
	}, time.Unix(1, 0))
	if err == nil {
		t.Fatal("expected a step targeting another mount location to be rejected")
	}
}

func TestNewRecoveryJournalRequiresStrongIdentityOnEveryStep(t *testing.T) {
	journalIdentity := destinationreadiness.DeviceIdentity{DevicePath: "/dev/sda1", Filesystem: "ntfs", UUID: "uuid-1", TotalBytes: 100}
	stepIdentity := destinationreadiness.DeviceIdentity{DevicePath: "/dev/sda1", Filesystem: "ntfs", TotalBytes: 100}
	_, err := destinationreadiness.NewRecoveryJournal("/media/Elements", journalIdentity, []destinationreadiness.Plan{
		{Action: destinationreadiness.ActionUnmount, Location: "/media/Elements", TargetPath: "/media/Elements", Identity: stepIdentity},
	}, time.Unix(1, 0))
	if err == nil {
		t.Fatal("expected a step without UUID or serial to be rejected")
	}
}

func TestFileRecoveryStoreRejectsJournalIDMismatch(t *testing.T) {
	_, journal, _ := recoveryFixture(t)
	root := t.TempDir()
	store := &destinationreadiness.FileRecoveryStore{Root: root}
	if err := store.Save(context.Background(), journal); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, journal.ID+".json"))
	if err != nil {
		t.Fatal(err)
	}
	const otherID = "000000000000000000000000"
	if err := os.WriteFile(filepath.Join(root, otherID+".json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	_, err = store.Load(context.Background(), otherID)
	if err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("error = %v, want filename mismatch", err)
	}
}

func TestFileRecoveryStoreRejectsCursorSkippingPendingStep(t *testing.T) {
	_, journal, _ := recoveryFixture(t)
	journal.Current = 1
	store := &destinationreadiness.FileRecoveryStore{Root: t.TempDir()}
	if err := store.Save(context.Background(), journal); err == nil {
		t.Fatal("expected cursor/status mismatch to be rejected")
	}
}
