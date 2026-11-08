package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/artifactlock"
	"github.com/vrooli/vrooli/internal/config"
)

func updateTestStoreAt(path string, now time.Time) *managedUpdateStore {
	return &managedUpdateStore{resource: "cloudflared", path: path, now: func() time.Time { return now }, write: config.WriteOwnedFileDurable, lock: func(ctx context.Context) (func(), error) {
		return artifactlock.AcquireWithPathContext(ctx, "update-test:"+path, func(string) (string, error) { return path + ".lock", nil })
	}}
}

func updateTestStore(t *testing.T, u ManagedServiceUpdate) *managedUpdateStore {
	t.Helper()
	return updateTestStoreAt(filepath.Join(t.TempDir(), managedUpdateLedgerFile), u.UpdatedAt)
}

func updateStoreTrace(t *testing.T, s *managedUpdateStore, u ManagedServiceUpdate, trace []updateTraceStep) ManagedServiceUpdate {
	t.Helper()
	for _, step := range trace {
		observed := updateTestClone(t, u)
		at := u.UpdatedAt.Add(time.Second)
		if step.observe != nil {
			step.observe(&observed, at)
		}
		s.now = func() time.Time { return at }
		next, err := s.Commit(context.Background(), observed, step.event)
		if err != nil {
			t.Fatalf("store %s: %v", step.event, err)
		}
		if next.Phase != step.phase {
			t.Fatalf("store %s reached %s, want %s", step.event, next.Phase, step.phase)
		}
		resource := s.resource
		s = updateTestStoreAt(s.path, at) // Lose all in-memory owner state after every commit.
		s.resource = resource
		loaded, err := s.Lookup(u.RequestID)
		if err != nil {
			t.Fatal(err)
		}
		if loaded.Revision != next.Revision || loaded.Phase != next.Phase {
			t.Fatal("restart lost committed checkpoint")
		}
		replayed, err := s.Commit(context.Background(), observed, step.event)
		if err != nil {
			t.Fatalf("lost acknowledgement replay: %v", err)
		}
		if replayed.Revision != next.Revision {
			t.Fatal("replay advanced revision twice")
		}
		u = loaded
	}
	return u
}

func TestManagedUpdateStoreCrashReloadAndReplayAtEveryCheckpoint(t *testing.T) {
	for name, trace := range map[string][]updateTraceStep{"promotion": updateSuccessTrace(), "restoration": updateRestorationTrace(), "retry acknowledged restoration": updateRetryRestorationTrace(true), "retry unacknowledged restoration": updateRetryRestorationTrace(false)} {
		t.Run(name, func(t *testing.T) {
			u := updateTestInitial(t)
			s := updateTestStore(t, u)
			if _, replayed, err := s.Create(context.Background(), u, false); err != nil || replayed {
				t.Fatalf("initial admission: replay=%v err=%v", replayed, err)
			}
			last := updateStoreTrace(t, s, u, trace)
			if !last.Phase.Terminal() {
				t.Fatal("trace did not terminate")
			}
			if _, err := s.Active(); !errors.Is(err, ErrManagedUpdateNotFound) {
				t.Fatalf("terminal operation retained active fence: %v", err)
			}
			replay := updateTestClone(t, u)
			replay.ID = "another-generated-id"
			replay.CreatedAt = replay.CreatedAt.Add(time.Second)
			replay.UpdatedAt = replay.CreatedAt
			got, replayed, err := s.Create(context.Background(), replay, false)
			if err != nil || !replayed || got.ID != u.ID || got.Revision != last.Revision {
				t.Fatalf("lost create reply redispatched update: %+v replay=%v err=%v", got, replayed, err)
			}
		})
	}
}

func TestManagedUpdateStorePersistenceFailureNeverAcknowledgesNextEffect(t *testing.T) {
	for _, afterRename := range []bool{false, true} {
		t.Run(fmt.Sprintf("after_rename_%v", afterRename), func(t *testing.T) {
			u := updateTestInitial(t)
			s := updateTestStore(t, u)
			if _, _, err := s.Create(context.Background(), u, false); err != nil {
				t.Fatal(err)
			}
			for _, step := range updateSuccessTrace() {
				observed := updateTestClone(t, u)
				at := u.UpdatedAt.Add(time.Second)
				step.observe(&observed, at)
				s.now = func() time.Time { return at }
				injected := errors.New("injected durable commit acknowledgement loss")
				s.write = func(path string, data []byte, mode os.FileMode) error {
					if afterRename {
						if err := config.WriteOwnedFileDurable(path, data, mode); err != nil {
							return err
						}
					}
					return injected
				}
				acknowledged, err := s.Commit(context.Background(), observed, step.event)
				if !errors.Is(err, ErrManagedUpdateCommitUnknown) || !errors.Is(err, injected) || acknowledged.ID != "" {
					t.Fatalf("failed commit acknowledged %s: %+v %v", step.event, acknowledged, err)
				}
				// Recreate the repository as another control-plane process would.
				s = updateTestStoreAt(s.path, at)
				recorded, err := s.Lookup(u.RequestID)
				if err != nil {
					t.Fatal(err)
				}
				expected := u.Revision
				if afterRename {
					expected++
				}
				if recorded.Revision != expected {
					t.Fatalf("partial checkpoint: revision=%d expected=%d", recorded.Revision, expected)
				}
				next, err := s.Commit(context.Background(), observed, step.event)
				if err != nil {
					t.Fatalf("same-request reconciliation: %v", err)
				}
				if next.Revision != u.Revision+1 || next.Phase != step.phase {
					t.Fatal("reconciliation duplicated or lost operation")
				}
				u = next
			}
		})
	}
}

func TestManagedUpdateStoreRestoreRetirementSurvivesLostCommitAcknowledgement(t *testing.T) {
	for _, afterRename := range []bool{false, true} {
		t.Run(fmt.Sprintf("after_rename_%v", afterRename), func(t *testing.T) {
			u := updateTestInitial(t)
			s := updateTestStore(t, u)
			if _, _, err := s.Create(context.Background(), u, false); err != nil {
				t.Fatal(err)
			}
			trace := updateRetryRestorationTrace(true)
			u = updateStoreTrace(t, s, u, trace[:11])
			at := u.UpdatedAt.Add(time.Second)
			observed := updateTestClone(t, u)
			observed.Restored.Quiescence = updateTestQuiescence(observed.Restored, at)
			s.now = func() time.Time { return at }
			injected := errors.New("retirement directory sync acknowledgement lost")
			s.write = func(path string, data []byte, mode os.FileMode) error {
				if afterRename {
					if err := config.WriteOwnedFileDurable(path, data, mode); err != nil {
						return err
					}
				}
				return injected
			}
			if ack, err := s.Commit(context.Background(), observed, UpdateRetireRestore); !errors.Is(err, ErrManagedUpdateCommitUnknown) || !errors.Is(err, injected) || ack.ID != "" {
				t.Fatalf("uncertain retirement acknowledged replacement launch: %+v %v", ack, err)
			}
			s = updateTestStoreAt(s.path, at)
			got, err := s.Commit(context.Background(), observed, UpdateRetireRestore)
			if err != nil {
				t.Fatal(err)
			}
			if len(got.RetiredRestores) != 1 || got.Revision != u.Revision+1 || got.Restored.Intent != nil || got.CandidateQuiescent {
				t.Fatal("retirement replay duplicated history, lost checkpoint or removed candidate")
			}
			forged := updateTestClone(t, got)
			forged.RetiredRestores[0].Process.Intent.InstanceID = "forged-retired"
			forged.Restored.Intent = updateTestIntent("restored-second", forged.Rollback)
			if _, err := s.Commit(context.Background(), forged, UpdateLaunchRestore); !errors.Is(err, ErrManagedUpdateConflict) {
				t.Fatalf("retained attempt history could be overwritten: %v", err)
			}
			last := updateStoreTrace(t, s, got, trace[12:])
			if last.Phase != UpdateFailedRestored || len(last.RetiredRestores) != 1 {
				t.Fatal("replacement restoration did not recover service")
			}
		})
	}
}

func TestManagedUpdateStoreFailedAdmissionReplaysWithoutDuplicate(t *testing.T) {
	u := updateTestInitial(t)
	s := updateTestStore(t, u)
	s.write = func(path string, data []byte, mode os.FileMode) error {
		if err := config.WriteOwnedFileDurable(path, data, mode); err != nil {
			return err
		}
		return errors.New("acknowledgement lost")
	}
	if got, _, err := s.Create(context.Background(), u, false); !errors.Is(err, ErrManagedUpdateCommitUnknown) || got.ID != "" {
		t.Fatalf("uncertain admission acknowledged: %+v %v", got, err)
	}
	s = updateTestStoreAt(s.path, u.UpdatedAt)
	got, replayed, err := s.Create(context.Background(), u, false)
	if err != nil || !replayed || got.ID != u.ID {
		t.Fatalf("admission could not reconcile same request: %+v %v %v", got, replayed, err)
	}
	ledger, err := s.read()
	if err != nil || len(ledger.Records) != 1 {
		t.Fatalf("duplicate admission after unknown commit: %d %v", len(ledger.Records), err)
	}
}

func TestManagedUpdateStoreCompetingOwnersCannotReplaceContractsOrIdentity(t *testing.T) {
	u := updateTestInitial(t)
	s := updateTestStore(t, u)
	if _, _, err := s.Create(context.Background(), u, false); err != nil {
		t.Fatal(err)
	}
	observed := updateTestClone(t, u)
	observed.Candidate.Intent = updateTestIntent("candidate", u.Target)
	s.now = func() time.Time { return u.UpdatedAt.Add(time.Second) }
	const count = 8
	var wg sync.WaitGroup
	results := make(chan error, count)
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.Commit(context.Background(), observed, UpdateLaunchCandidate)
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.Lookup(u.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Revision != 2 {
		t.Fatalf("competing identical retries advanced %d revisions", got.Revision)
	}
	stale := updateTestClone(t, observed)
	stale.Candidate.Intent = updateTestIntent("different", u.Target)
	if _, err := s.Commit(context.Background(), stale, UpdateLaunchCandidate); !errors.Is(err, ErrManagedUpdateRevision) {
		t.Fatalf("different stale observation replaced checkpoint: %v", err)
	}
	for name, change := range map[string]func(*ManagedServiceUpdate){
		"contract": func(u *ManagedServiceUpdate) {
			u.Target.Manifest.ManagedService.Arguments[0] = "forged"
			u.Target.Digest = u.Target.digest()
		},
		"source":          func(u *ManagedServiceUpdate) { u.SourceRevision = updateTestDigest("different") },
		"launch identity": func(u *ManagedServiceUpdate) { u.Candidate.Intent = updateTestIntent("different", u.Target) },
		"phase skip":      func(u *ManagedServiceUpdate) { u.Phase = UpdateSucceeded },
		"abort authority": func(u *ManagedServiceUpdate) { u.AbortRequested = true },
	} {
		t.Run(name, func(t *testing.T) {
			bad := updateTestClone(t, got)
			change(&bad)
			if _, err := s.Commit(context.Background(), bad, UpdateCandidateStarted); err == nil {
				t.Fatal("immutable state replacement admitted")
			}
		})
	}
}

func TestManagedUpdateStoreCannotRebindCommittedNativeInvocation(t *testing.T) {
	u := updateTestInitial(t)
	s := updateTestStore(t, u)
	if _, _, err := s.Create(context.Background(), u, false); err != nil {
		t.Fatal(err)
	}
	u = updateStoreTrace(t, s, u, updateSuccessTrace()[:2])
	forged := updateTestClone(t, u)
	forged.Candidate.NativeRef.NativeInvocationID = updateTestDigest("another-manager-invocation")[:32]
	forged.Candidate.Proof = updateTestProof(*forged.Candidate.State, u.UpdatedAt.Add(time.Second))
	s.now = func() time.Time { return u.UpdatedAt.Add(time.Second) }
	if _, err := s.Commit(context.Background(), forged, UpdateCandidateVerified); !errors.Is(err, ErrManagedUpdateConflict) {
		t.Fatalf("native invocation changed under an existing resource process: %v", err)
	}
	recorded, err := s.Lookup(u.RequestID)
	if err != nil || recorded.Candidate.NativeRef.NativeInvocationID != u.Candidate.NativeRef.NativeInvocationID {
		t.Fatal("committed manager identity was replaced")
	}
}

func TestManagedUpdateStoreFailedReleaseNeedsExplicitNewRequest(t *testing.T) {
	u := updateTestInitial(t)
	s := updateTestStore(t, u)
	if _, _, err := s.Create(context.Background(), u, false); err != nil {
		t.Fatal(err)
	}
	trace := []updateTraceStep{
		{UpdateFailure, UpdatePreserving, func(u *ManagedServiceUpdate, _ time.Time) { u.Error = "candidate rejected" }},
		{UpdateOriginalVerified, UpdatePreserveCommitting, func(u *ManagedServiceUpdate, at time.Time) { u.OriginalProof = updateTestProof(u.Original, at) }},
		{UpdateFinishPreserved, UpdateFailedPreserved, func(u *ManagedServiceUpdate, at time.Time) {
			u.CandidateQuiescent = true
			u.OriginalProof = updateTestProof(u.Original, at)
		}},
	}
	updateStoreTrace(t, s, u, trace)
	retry := updateTestClone(t, u)
	retry.ID = "new-update"
	retry.RequestID = "new-request"
	if _, _, err := s.Create(context.Background(), retry, false); !errors.Is(err, ErrManagedUpdateConflict) {
		t.Fatalf("failed release auto retried: %v", err)
	}
	if got, replayed, err := s.Create(context.Background(), retry, true); err != nil || replayed || got.ID != retry.ID {
		t.Fatalf("explicit retry rejected: %+v %v %v", got, replayed, err)
	}
	third := updateTestClone(t, retry)
	third.ID = "third-update"
	third.RequestID = "third-request"
	if _, _, err := s.Create(context.Background(), third, true); !errors.Is(err, ErrManagedUpdateConflict) {
		t.Fatalf("explicit retry bypassed active operation: %v", err)
	}
}

func TestManagedUpdateStoreRejectsCorruptUnknownAndOversizedState(t *testing.T) {
	for name, corrupt := range map[string]func(*managedUpdateLedger){
		"wrong schema":      func(l *managedUpdateLedger) { l.SchemaVersion = 99 },
		"wrong resource":    func(l *managedUpdateLedger) { l.Resource = "another" },
		"duplicate request": func(l *managedUpdateLedger) { l.Records = append(l.Records, l.Records[0]) },
		"competing active": func(l *managedUpdateLedger) {
			another := l.Records[0]
			another.State.ID = "other"
			another.State.RequestID = "other"
			l.Records = append(l.Records, another)
		},
		"contract corruption": func(l *managedUpdateLedger) {
			l.Records[0].State.Rollback.Artifact.SHA256 = updateTestDigest("tampered")
		},
		"lost mutation identity": func(l *managedUpdateLedger) { l.Records[0].State.Revision = 2 },
	} {
		t.Run(name, func(t *testing.T) {
			u := updateTestInitial(t)
			s := updateTestStore(t, u)
			l := managedUpdateLedger{SchemaVersion: 1, Resource: u.Resource, Records: []managedUpdateEntry{{State: u}}}
			corrupt(&l)
			if err := s.save(l); err != nil {
				t.Fatal(err)
			}
			if _, err := s.Active(); err == nil {
				t.Fatal("corruption allowed caller to treat update as absent")
			}
			if _, _, err := s.Create(context.Background(), u, false); err == nil {
				t.Fatal("corruption overwritten by admission")
			}
		})
	}
	for name, data := range map[string][]byte{"unknown fields": []byte(`{"schema_version":1,"resource":"cloudflared","records":[],"allow_force":true}`), "trailing document": []byte(`{"schema_version":1,"resource":"cloudflared","records":[]} {}`), "oversized": []byte(strings.Repeat(" ", managedUpdateMaxBytes+1))} {
		t.Run(name, func(t *testing.T) {
			u := updateTestInitial(t)
			s := updateTestStore(t, u)
			if err := os.WriteFile(s.path, data, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := s.Active(); err == nil {
				t.Fatal("invalid state was treated as no active update")
			}
		})
	}
}

func TestManagedUpdateStoreConcurrentProcessAdmission(t *testing.T) {
	if os.Getenv("VROOLI_MANAGED_UPDATE_TEST_HELPER") == "1" {
		data, err := os.ReadFile(os.Getenv("VROOLI_UPDATE_TEST_INITIAL"))
		if err != nil {
			t.Fatal(err)
		}
		var u ManagedServiceUpdate
		if err := json.Unmarshal(data, &u); err != nil {
			t.Fatal(err)
		}
		u.ID = os.Getenv("VROOLI_UPDATE_TEST_REQUEST")
		u.RequestID = u.ID
		s := updateTestStoreAt(os.Getenv("VROOLI_UPDATE_TEST_LEDGER"), u.UpdatedAt)
		_, _, err = s.Create(context.Background(), u, false)
		if err == nil {
			fmt.Println("ADMITTED")
		} else if errors.Is(err, ErrManagedUpdateConflict) {
			fmt.Println("CONFLICT")
		} else {
			t.Fatal(err)
		}
		return
	}
	u := updateTestInitial(t)
	root := t.TempDir()
	initialPath := filepath.Join(root, "initial.json")
	data, err := json.Marshal(u)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(initialPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	ledgerPath := filepath.Join(root, managedUpdateLedgerFile)
	results := make(chan string, 2)
	failures := make(chan error, 2)
	for _, id := range []string{"owner-a", "owner-b"} {
		cmd := exec.Command(os.Args[0], "-test.run=^TestManagedUpdateStoreConcurrentProcessAdmission$")
		cmd.Env = append(os.Environ(), "VROOLI_MANAGED_UPDATE_TEST_HELPER=1", "VROOLI_UPDATE_TEST_INITIAL="+initialPath, "VROOLI_UPDATE_TEST_LEDGER="+ledgerPath, "VROOLI_UPDATE_TEST_REQUEST="+id)
		go func() {
			out, err := cmd.CombinedOutput()
			if err != nil {
				failures <- fmt.Errorf("helper failed: %w %s", err, out)
			} else {
				failures <- nil
			}
			results <- string(out)
		}()
	}
	admitted, conflicts := 0, 0
	for i := 0; i < 2; i++ {
		if err := <-failures; err != nil {
			t.Fatal(err)
		}
		result := <-results
		if strings.Contains(result, "ADMITTED") {
			admitted++
		}
		if strings.Contains(result, "CONFLICT") {
			conflicts++
		}
	}
	if admitted != 1 || conflicts != 1 {
		t.Fatalf("two processes admitted=%d conflict=%d", admitted, conflicts)
	}
	s := updateTestStoreAt(ledgerPath, u.UpdatedAt)
	l, err := s.read()
	if err != nil || len(l.Records) != 1 {
		t.Fatalf("competing processes lost ledger: %d %v", len(l.Records), err)
	}
}

func TestManagedUpdateStoreReconstructionMustReconfirmDurability(t *testing.T) {
	u := updateTestInitial(t)
	s := updateTestStore(t, u)
	if _, _, err := s.Create(context.Background(), u, false); err != nil {
		t.Fatal(err)
	}
	s = updateTestStoreAt(s.path, u.UpdatedAt)
	s.write = func(string, []byte, os.FileMode) error { return errors.New("directory durability unavailable") }
	if got, err := s.Confirm(context.Background(), u.RequestID, u.Revision); !errors.Is(err, ErrManagedUpdateCommitUnknown) || got.ID != "" {
		t.Fatalf("reconstruction acknowledged nondurable effect intent: %+v %v", got, err)
	}
	s = updateTestStoreAt(s.path, u.UpdatedAt)
	if _, err := s.Confirm(context.Background(), u.RequestID, u.Revision+1); !errors.Is(err, ErrManagedUpdateRevision) {
		t.Fatalf("reconstruction accepted other revision: %v", err)
	}
	got, err := s.Confirm(context.Background(), u.RequestID, u.Revision)
	if err != nil || got.ID != u.ID || got.Revision != u.Revision {
		t.Fatalf("confirmation changed operation: %+v %v", got, err)
	}
}

func TestManagedUpdateStoreCapacityRetainsDeduplicationKeys(t *testing.T) {
	u := updateTestInitial(t)
	s := updateTestStore(t, u)
	terminal := updateRunTrace(t, u, updateSuccessTrace())
	last := terminal[len(terminal)-1]
	ledger := managedUpdateLedger{SchemaVersion: 1, Resource: u.Resource}
	for i := 0; i < managedUpdateMaxRecords; i++ {
		record := updateTestClone(t, last)
		record.ID = fmt.Sprintf("old-update-%d", i)
		record.RequestID = fmt.Sprintf("old-request-%d", i)
		ledger.Records = append(ledger.Records, managedUpdateEntry{State: record, LastBaseRevision: record.Revision - 1, LastInputDigest: updateTestDigest("mutation")})
	}
	if err := s.save(ledger); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Create(context.Background(), u, true); err == nil {
		t.Fatal("full ledger silently evicted a replay key")
	}
	recorded, err := s.Lookup("old-request-0")
	if err != nil || recorded.Phase != UpdateSucceeded {
		t.Fatalf("capacity failure lost original receipt: %+v %v", recorded, err)
	}
	old := updateTestClone(t, u)
	old.ID = "new-generated-id"
	old.RequestID = "old-request-0"
	got, replayed, err := s.Create(context.Background(), old, false)
	if err != nil || !replayed || got.ID != "old-update-0" {
		t.Fatalf("full ledger redispatched old request: %+v %v %v", got, replayed, err)
	}
}
