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
	calls   int
	fail    bool
	entered chan struct{}
	release chan struct{}
}

func (r *recoveryRemediator) Supported(destinationreadiness.PreparationAction) (bool, string) {
	return true, ""
}

func (r *recoveryRemediator) Remediate(context.Context, destinationreadiness.Plan, bool) (destinationreadiness.RemediationOutcome, error) {
	r.calls++
	if r.entered != nil {
		r.entered <- struct{}{}
		<-r.release
	}
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
	if m.journal.ID == "" {
		return destinationreadiness.RecoveryJournal{}, destinationreadiness.ErrRecoveryNotFound
	}
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

func TestConcurrentResumeRecoveryExecutesEachHostStepOnce(t *testing.T) {
	svc, journal, remediator := recoveryFixture(t)
	remediator.entered = make(chan struct{}, 1)
	remediator.release = make(chan struct{})
	store := &memoryRecoveryStore{journal: journal}
	type result struct {
		journal destinationreadiness.RecoveryJournal
		err     error
	}
	results := make(chan result, 2)
	resume := func() {
		got, err := svc.ResumeRecovery(context.Background(), store, journal, map[int]string{0: "confirm-unmount", 1: "confirm-mount"}, false, false)
		results <- result{journal: got, err: err}
	}
	go resume()
	<-remediator.entered
	go resume()
	close(remediator.release)
	first, second := <-results, <-results
	if first.err != nil || second.err != nil {
		t.Fatalf("resume errors: first=%v second=%v", first.err, second.err)
	}
	if remediator.calls != 2 {
		t.Fatalf("host steps executed %d times, want one execution per step (2 total)", remediator.calls)
	}
	if store.journal.State != destinationreadiness.RecoveryCompleted || store.journal.Current != 2 {
		t.Fatalf("durable journal did not finish: %+v", store.journal)
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

func TestRecoveryStateTransitionMatrix(t *testing.T) {
	states := []destinationreadiness.RecoveryState{
		destinationreadiness.RecoveryPending,
		destinationreadiness.RecoveryRunning,
		destinationreadiness.RecoveryPaused,
		destinationreadiness.RecoveryCompleted,
		destinationreadiness.RecoveryFailed,
	}
	events := []destinationreadiness.RecoveryEvent{
		destinationreadiness.RecoveryResume,
		destinationreadiness.RecoveryPause,
		destinationreadiness.RecoveryComplete,
		destinationreadiness.RecoveryDryRunReset,
	}
	legal := map[struct {
		state destinationreadiness.RecoveryState
		event destinationreadiness.RecoveryEvent
	}]destinationreadiness.RecoveryState{
		{destinationreadiness.RecoveryPending, destinationreadiness.RecoveryResume}:      destinationreadiness.RecoveryRunning,
		{destinationreadiness.RecoveryRunning, destinationreadiness.RecoveryResume}:      destinationreadiness.RecoveryRunning,
		{destinationreadiness.RecoveryPaused, destinationreadiness.RecoveryResume}:       destinationreadiness.RecoveryRunning,
		{destinationreadiness.RecoveryFailed, destinationreadiness.RecoveryResume}:       destinationreadiness.RecoveryRunning,
		{destinationreadiness.RecoveryRunning, destinationreadiness.RecoveryPause}:       destinationreadiness.RecoveryPaused,
		{destinationreadiness.RecoveryRunning, destinationreadiness.RecoveryComplete}:    destinationreadiness.RecoveryCompleted,
		{destinationreadiness.RecoveryRunning, destinationreadiness.RecoveryDryRunReset}: destinationreadiness.RecoveryPending,
	}
	for _, state := range states {
		for _, event := range events {
			journal := destinationreadiness.RecoveryJournal{State: state}
			got, ok := state.Transition(event)
			want, wantOK := legal[struct {
				state destinationreadiness.RecoveryState
				event destinationreadiness.RecoveryEvent
			}{state, event}]
			if ok != wantOK || (ok && got != want) || (!ok && got != state) {
				t.Errorf("Transition(%s,%s) = %s,%v; want %s,%v", state, event, got, ok, want, wantOK)
			}
			err := journal.Transition(event)
			if wantOK {
				if err != nil || journal.State != want {
					t.Errorf("journal transition %s + %s = %s, %v; want %s", state, event, journal.State, err, want)
				}
			} else if err == nil || journal.State != state {
				t.Errorf("invalid journal transition %s + %s mutated to %s (err=%v)", state, event, journal.State, err)
			}
		}
	}
	unknown := destinationreadiness.RecoveryJournal{State: "future-state"}
	if err := unknown.Transition(destinationreadiness.RecoveryResume); err == nil || unknown.State != "future-state" {
		t.Fatalf("unknown-state transition changed journal: %+v err=%v", unknown, err)
	}
}

func TestRecoveryStepTransitionMatrix(t *testing.T) {
	statuses := []destinationreadiness.RecoveryStepStatus{
		destinationreadiness.RecoveryStepPending,
		destinationreadiness.RecoveryStepInFlight,
		destinationreadiness.RecoveryStepBlocked,
		destinationreadiness.RecoveryStepValidated,
		destinationreadiness.RecoveryStepCompleted,
	}
	events := []destinationreadiness.RecoveryStepEvent{
		destinationreadiness.RecoveryStepBegin,
		destinationreadiness.RecoveryStepBlock,
		destinationreadiness.RecoveryStepValidate,
		destinationreadiness.RecoveryStepComplete,
	}
	legal := map[struct {
		status destinationreadiness.RecoveryStepStatus
		event  destinationreadiness.RecoveryStepEvent
	}]destinationreadiness.RecoveryStepStatus{
		{destinationreadiness.RecoveryStepPending, destinationreadiness.RecoveryStepBegin}:     destinationreadiness.RecoveryStepInFlight,
		{destinationreadiness.RecoveryStepInFlight, destinationreadiness.RecoveryStepBlock}:    destinationreadiness.RecoveryStepBlocked,
		{destinationreadiness.RecoveryStepInFlight, destinationreadiness.RecoveryStepValidate}: destinationreadiness.RecoveryStepValidated,
		{destinationreadiness.RecoveryStepInFlight, destinationreadiness.RecoveryStepComplete}: destinationreadiness.RecoveryStepCompleted,
		{destinationreadiness.RecoveryStepBlocked, destinationreadiness.RecoveryStepBegin}:     destinationreadiness.RecoveryStepInFlight,
		{destinationreadiness.RecoveryStepValidated, destinationreadiness.RecoveryStepBegin}:   destinationreadiness.RecoveryStepInFlight,
	}
	for _, status := range statuses {
		for _, event := range events {
			step := destinationreadiness.RecoveryStep{Status: status}
			want, ok := legal[struct {
				status destinationreadiness.RecoveryStepStatus
				event  destinationreadiness.RecoveryStepEvent
			}{status, event}]
			err := step.Transition(event)
			if ok {
				if err != nil || step.Status != want {
					t.Errorf("step transition %s + %s = %s, %v; want %s", status, event, step.Status, err, want)
				}
			} else if err == nil || step.Status != status {
				t.Errorf("invalid step transition %s + %s mutated to %s (err=%v)", status, event, step.Status, err)
			}
		}
	}
}

func TestFileRecoveryStoreRejectsUnknownLifecycleValues(t *testing.T) {
	_, journal, _ := recoveryFixture(t)
	store := &destinationreadiness.FileRecoveryStore{Root: t.TempDir()}
	journal.State = "future-state"
	if err := store.Save(context.Background(), journal); err == nil {
		t.Fatal("unknown journal state must not be persisted")
	}
	_, journal, _ = recoveryFixture(t)
	journal.Steps[0].Status = "future-step"
	if err := store.Save(context.Background(), journal); err == nil {
		t.Fatal("unknown step status must not be persisted")
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
	if info.Mode().Perm() != 0o600 {
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

func TestFileRecoveryEpisodePersistsUntilMountedThenStartsFresh(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	identity := destinationreadiness.DeviceIdentity{DevicePath: "/dev/sdz1", Filesystem: "ntfs", UUID: "uuid-elements", Serial: "serial-elements"}
	store := &destinationreadiness.FileRecoveryStore{Root: root}
	first, err := store.ObserveEpisode(ctx, "elements", identity, false, "")
	if err != nil || first.RecoveryID == "" {
		t.Fatalf("first unmounted observation=(%+v, %v)", first, err)
	}
	repeated, err := store.ObserveEpisode(ctx, "elements", identity, false, "")
	if err != nil || repeated.RecoveryID != first.RecoveryID {
		t.Fatalf("repeated unmounted observation=(%+v, %v), want stable recovery ID %q", repeated, err, first.RecoveryID)
	}

	// A new store models a DBM process restart. It must retain the active outage
	// episode and must not create a second decision correlation.
	store = &destinationreadiness.FileRecoveryStore{Root: root}
	restarted, err := store.ObserveEpisode(ctx, "elements", identity, false, "")
	if err != nil || restarted.RecoveryID != first.RecoveryID {
		t.Fatalf("post-restart observation=(%+v, %v), want recovery ID %q", restarted, err, first.RecoveryID)
	}

	mounted, err := store.ObserveEpisode(ctx, "elements", identity, true, "")
	if err != nil || !mounted.LastMounted || mounted.RecoveryID != "" {
		t.Fatalf("mounted observation=(%+v, %v), want closed episode", mounted, err)
	}
	second, err := store.ObserveEpisode(ctx, "elements", identity, false, "")
	if err != nil || second.RecoveryID == "" || second.RecoveryID == first.RecoveryID {
		t.Fatalf("second outage=(%+v, %v), want a fresh recovery ID after remount", second, err)
	}
	secondRestart, err := (&destinationreadiness.FileRecoveryStore{Root: root}).ObserveEpisode(ctx, "elements", identity, false, "")
	if err != nil || secondRestart.RecoveryID != second.RecoveryID {
		t.Fatalf("second post-restart observation=(%+v, %v), want recovery ID %q", secondRestart, err, second.RecoveryID)
	}
}

func TestFileRecoveryEpisodeAdoptsIncompleteLegacyJournalAndAdvancesCompletedOne(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	identity := destinationreadiness.DeviceIdentity{DevicePath: "/dev/sdz1", Mountpoint: "/media/Elements", Filesystem: "ntfs", UUID: "uuid-elements", Serial: "serial-elements"}
	store := &destinationreadiness.FileRecoveryStore{Root: root}
	_, legacy, _ := recoveryFixture(t)
	legacy.ID = "legacy-recovery-id"
	legacy.Identity = identity
	legacy.Steps[0].Plan.Identity = identity
	legacy.Steps[0].Plan.Location = identity.Mountpoint
	legacy.Steps[0].Plan.TargetPath = identity.Mountpoint
	legacy.Steps[1].Plan.Identity = identity
	legacy.Steps[1].Plan.Location = identity.Mountpoint
	legacy.Steps[1].Plan.TargetPath = identity.Mountpoint
	legacy.Current = 0
	legacy.State = destinationreadiness.RecoveryPaused
	legacy.Steps[0].Status = "blocked"
	if err := store.Save(ctx, legacy); err != nil {
		t.Fatalf("save legacy journal: %v", err)
	}
	episode, err := store.ObserveEpisode(ctx, "elements", identity, false, legacy.ID)
	if err != nil || episode.RecoveryID != legacy.ID {
		t.Fatalf("legacy episode=(%+v, %v), want adopted journal %q", episode, err, legacy.ID)
	}

	_, completed, _ := recoveryFixture(t)
	completed.ID = episode.RecoveryID
	completed.Identity = identity
	completed.Steps[0].Plan.Identity = identity
	completed.Steps[0].Plan.Location = identity.Mountpoint
	completed.Steps[0].Plan.TargetPath = identity.Mountpoint
	completed.Steps[1].Plan.Identity = identity
	completed.Steps[1].Plan.Location = identity.Mountpoint
	completed.Steps[1].Plan.TargetPath = identity.Mountpoint
	completed.Current = len(completed.Steps)
	completed.State = destinationreadiness.RecoveryCompleted
	for index := range completed.Steps {
		completed.Steps[index].Status = "completed"
	}
	if err := store.Save(ctx, completed); err != nil {
		t.Fatalf("save completed journal: %v", err)
	}
	next, err := store.AdvanceUnmountedEpisode(ctx, "elements", completed.ID)
	if err != nil || next.RecoveryID == "" || next.RecoveryID == completed.ID || next.LastMounted {
		t.Fatalf("advanced episode=(%+v, %v), want fresh ID while still unmounted", next, err)
	}
	if _, err := store.AdvanceUnmountedEpisode(ctx, "elements", completed.ID); err == nil {
		t.Fatal("stale completion must not advance the same outage episode twice")
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
	if err := os.WriteFile(filepath.Join(root, otherID+".json"), data, 0o600); err != nil {
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
