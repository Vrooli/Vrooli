package sessionprofile

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/scheduletest"
	"github.com/vrooli/browser-automation-studio/internal/testutil"
	"github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

func newTestService(t *testing.T) (*Service, *persistence.MockRepository, *scheduletest.FakeClock) {
	t.Helper()
	repo := persistence.NewMockRepository()
	mockClock := scheduletest.New(time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC))
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)

	svc := NewServiceWithConfig(repo, log, ServiceConfig{
		Clock: mockClock,
	})

	return svc, repo, mockClock
}

func TestService_CreateProfile(t *testing.T) {
	svc, repo, mockClock := newTestService(t)

	profile, err := svc.CreateProfile("My Test Profile")
	if err != nil {
		t.Fatalf("CreateProfile failed: %v", err)
	}

	if profile.Name != "My Test Profile" {
		t.Errorf("expected name 'My Test Profile', got '%s'", profile.Name)
	}
	if profile.ID == "" {
		t.Error("expected profile to have an ID")
	}
	if !profile.CreatedAt.Equal(mockClock.Now()) {
		t.Error("expected CreatedAt to use mock clock")
	}

	// Verify persisted
	if repo.Count() != 1 {
		t.Errorf("expected 1 profile in repo, got %d", repo.Count())
	}
}

func TestService_CreateProfile_AutoGenerateName(t *testing.T) {
	svc, _, _ := newTestService(t)

	profile, err := svc.CreateProfile("")
	if err != nil {
		t.Fatalf("CreateProfile failed: %v", err)
	}

	if profile.Name != "Session 1" {
		t.Errorf("expected auto-generated name 'Session 1', got '%s'", profile.Name)
	}
}

func TestService_GetOrCreateProfile(t *testing.T) {
	svc, _, _ := newTestService(t)

	// First call should create
	profile1, err := svc.GetOrCreateProfile("")
	if err != nil {
		t.Fatalf("GetOrCreateProfile failed: %v", err)
	}

	// Second call should return existing
	profile2, err := svc.GetOrCreateProfile("")
	if err != nil {
		t.Fatalf("GetOrCreateProfile failed: %v", err)
	}

	if profile1.ID != profile2.ID {
		t.Error("expected same profile to be returned")
	}
}

// [REQ:BAS-RH-J14] Failed recovery cannot silently replace an acknowledged identity.
func TestService_ProfileRecoveryPreservesIdentity(t *testing.T) {
	for _, fault := range []string{"missing-state", "corrupt-state", "wrong-key", "invalid-key", "corrupt-metadata"} {
		t.Run(fault, func(t *testing.T) {
			authority, err := testutil.ProfileCredentialAuthority()
			if err != nil {
				t.Fatal(err)
			}
			key, err := authority.Resolve("vrooli/browser-automation-studio", "session-profile-keyring")
			if err != nil {
				t.Fatal(err)
			}
			config := persistence.FileRepositoryConfig{Authority: func() (*credentialauthority.Authority, error) { return authority, nil }}
			root := t.TempDir()
			svc := NewService(persistence.NewFileRepositoryWithConfig(root, nil, config), nil)
			profile, err := svc.CreateProfile("Original identity")
			if err != nil {
				t.Fatal(err)
			}
			state := []byte(`{"cookies":[{"name":"identity","value":"synthetic-secret"}],"origins":[]}`)
			if _, err := svc.SaveStorageState(profile.ID, state); err != nil {
				t.Fatal(err)
			}
			readFiles := func() map[string]string {
				t.Helper()
				entries, err := os.ReadDir(root)
				if err != nil {
					t.Fatal(err)
				}
				files := make(map[string]string, len(entries))
				for _, entry := range entries {
					data, err := os.ReadFile(filepath.Join(root, entry.Name()))
					if err != nil {
						t.Fatal(err)
					}
					files[entry.Name()] = string(data)
				}
				return files
			}
			original := readFiles()
			protected := filepath.Join(root, string(profile.ID)+".json")
			switch fault {
			case "missing-state":
				err = os.WriteFile(protected, []byte(`{"version":1}`), 0o600)
			case "corrupt-state":
				err = os.WriteFile(protected, []byte("truncated"), 0o600)
			case "wrong-key":
				err = authority.Put("vrooli/browser-automation-studio", "session-profile-keyring", `{"active":1,"keys":{"1":"AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE="}}`)
			case "invalid-key":
				err = authority.Put("vrooli/browser-automation-studio", "session-profile-keyring", "invalid")
			case "corrupt-metadata":
				err = os.WriteFile(filepath.Join(root, string(profile.ID)+".json"), []byte("{"), 0o600)
			}
			if err != nil {
				t.Fatal(err)
			}
			before := readFiles()
			// A new service must not hide the failure through in-memory state.
			restarted := NewService(persistence.NewFileRepositoryWithConfig(root, nil, config), nil)
			if _, err := restarted.GetProfile(profile.ID); err == nil {
				t.Error("unreadable profile was treated as recovered")
			}
			if _, err := restarted.ListProfiles(); err == nil {
				t.Error("listing concealed the recovery failure")
			}
			if replacement, err := restarted.GetOrCreateProfile(""); err == nil || replacement != nil {
				t.Error("default resolution replaced or accepted an unreadable identity")
			}
			if !reflect.DeepEqual(before, readFiles()) {
				t.Fatal("failed recovery changed saved files")
			}
			// Repair only the injected fault; the original identity must return.
			if err := authority.Put("vrooli/browser-automation-studio", "session-profile-keyring", key); err != nil {
				t.Fatal(err)
			}
			for name, data := range original {
				if err := os.WriteFile(filepath.Join(root, name), []byte(data), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			recovered, err := restarted.GetOrCreateProfile("")
			if err != nil || recovered == nil || recovered.ID != profile.ID || string(recovered.StorageState) != string(state) {
				t.Fatalf("original identity did not recover: %v", err)
			}
		})
	}
}

func TestService_GetOrCreateProfile_SpecificID(t *testing.T) {
	svc, _, _ := newTestService(t)

	// Create a profile
	created, err := svc.CreateProfile("Test")
	if err != nil {
		t.Fatalf("CreateProfile failed: %v", err)
	}

	// Request it by ID
	profile, err := svc.GetOrCreateProfile(created.ID)
	if err != nil {
		t.Fatalf("GetOrCreateProfile failed: %v", err)
	}

	if profile.ID != created.ID {
		t.Error("expected same profile to be returned")
	}
}

func TestService_GetOrCreateProfile_NotFound(t *testing.T) {
	svc, _, _ := newTestService(t)

	_, err := svc.GetOrCreateProfile("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent profile ID")
	}
}

func TestService_RenameProfile(t *testing.T) {
	svc, _, mockClock := newTestService(t)

	// Create a profile
	created, _ := svc.CreateProfile("Original")
	originalUpdatedAt := created.UpdatedAt

	// Advance clock
	mockClock.Advance(time.Hour)

	// Rename it
	renamed, err := svc.RenameProfile(created.ID, "New Name")
	if err != nil {
		t.Fatalf("RenameProfile failed: %v", err)
	}

	if renamed.Name != "New Name" {
		t.Errorf("expected name 'New Name', got '%s'", renamed.Name)
	}
	if renamed.UpdatedAt.Equal(originalUpdatedAt) {
		t.Error("expected UpdatedAt to be updated")
	}
}

func TestService_DeleteProfile(t *testing.T) {
	svc, repo, _ := newTestService(t)

	// Create a profile
	created, _ := svc.CreateProfile("To Delete")

	// Delete it
	err := svc.DeleteProfile(created.ID)
	if err != nil {
		t.Fatalf("DeleteProfile failed: %v", err)
	}

	// Verify deleted
	if repo.Count() != 0 {
		t.Error("expected profile to be deleted")
	}
}

func TestService_TouchAndAssociateSession(t *testing.T) {
	svc, _, mockClock := newTestService(t)

	// Create a profile
	created, _ := svc.CreateProfile("Test")
	originalLastUsed := created.LastUsedAt

	// Advance clock
	mockClock.Advance(time.Hour)

	// Start session
	_, err := svc.Touch(created.ID)
	svc.SetActiveSession("browser-session-1", string(created.ID))
	if err != nil {
		t.Fatalf("Touch failed: %v", err)
	}

	// Verify session is tracked
	profileID := svc.GetActiveSession("browser-session-1")
	if profileID != string(created.ID) {
		t.Errorf("expected profile ID %s, got %s", created.ID, profileID)
	}

	// Verify last used was updated
	updated, _ := svc.GetProfile(created.ID)
	if updated.LastUsedAt.Equal(originalLastUsed) {
		t.Error("expected LastUsedAt to be updated")
	}
}

func TestService_PersistSessionState(t *testing.T) {
	svc, _, mockClock := newTestService(t)

	// Create a profile and start session
	created, _ := svc.CreateProfile("Test")
	svc.SetActiveSession("browser-session-1", string(created.ID))

	mockClock.Advance(time.Hour)

	// End session with state
	state := &persistence.SessionEndState{
		StorageState: []byte(`{"cookies":[]}`),
		OpenTabs: []persistence.TabState{
			{URL: "https://example.com", Title: "Example", Order: 0},
		},
	}

	ctx := context.Background()
	err := svc.PersistSessionState(ctx, "browser-session-1", func(context.Context, string) (*persistence.SessionEndState, error) { return state, nil })
	if err != nil {
		t.Fatalf("PersistSessionState failed: %v", err)
	}

	// The lifecycle owner detaches only after a successful save and browser close.
	svc.ClearActiveSession("browser-session-1")
	// Verify session is no longer tracked
	profileID := svc.GetActiveSession("browser-session-1")
	if profileID != "" {
		t.Error("expected session to be cleared")
	}

	// Verify state was saved
	updated, _ := svc.GetProfile(created.ID)
	if len(updated.StorageState) == 0 {
		t.Error("expected storage state to be saved")
	}
	if len(updated.OpenTabs) != 1 {
		t.Errorf("expected 1 tab, got %d", len(updated.OpenTabs))
	}
}

func TestService_PersistSessionState_LimitsTabs(t *testing.T) {
	svc, _, _ := newTestService(t)

	// Create a profile and start session
	created, _ := svc.CreateProfile("Test")
	svc.SetActiveSession("browser-session-1", string(created.ID))

	// Create more tabs than the limit
	tabs := make([]persistence.TabState, persistence.MaxRestoredTabs+10)
	for i := range tabs {
		tabs[i] = persistence.TabState{URL: "https://example.com", Order: i}
	}

	state := &persistence.SessionEndState{
		OpenTabs: tabs,
	}

	ctx := context.Background()
	err := svc.PersistSessionState(ctx, "browser-session-1", func(context.Context, string) (*persistence.SessionEndState, error) { return state, nil })
	if err != nil {
		t.Fatalf("PersistSessionState failed: %v", err)
	}

	// Verify tabs were limited
	updated, _ := svc.GetProfile(created.ID)
	if len(updated.OpenTabs) > persistence.MaxRestoredTabs {
		t.Errorf("expected at most %d tabs, got %d", persistence.MaxRestoredTabs, len(updated.OpenTabs))
	}
}

// [REQ:BAS-RH-J14] A cancelled waiter neither captures nor publishes an older
// snapshot; unrelated profiles continue while a browser capture is pending.
func TestService_ProfileCapturesSerializeAndRespectCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		svc, _, _ := newTestService(t)
		profile, _ := svc.CreateProfile("Main")
		other, _ := svc.CreateProfile("Independent")
		svc.SetActiveSession("main", string(profile.ID))
		svc.SetActiveSession("other", string(other.ID))
		started, release := make(chan struct{}), make(chan struct{})
		var captures atomic.Int32
		capture := func(context.Context, string) (*persistence.SessionEndState, error) {
			value := []byte(`{"marker":"new"}`)
			if captures.Add(1) == 1 {
				close(started)
				<-release
				value = []byte(`{"marker":"old"}`)
			}
			return &persistence.SessionEndState{StorageState: value}, nil
		}
		first, second := make(chan error, 1), make(chan error, 1)
		go func() { first <- svc.PersistSessionState(context.Background(), "main", capture) }()
		<-started
		ctx, cancel := context.WithCancel(context.Background())
		go func() { second <- svc.PersistSessionState(ctx, "main", capture) }()
		synctest.Wait()
		if captures.Load() != 1 {
			t.Error("overlapping snapshots entered capture")
		}
		cancel()
		if err := <-second; !errors.Is(err, context.Canceled) {
			t.Errorf("cancelled waiter = %v", err)
		}
		err := svc.PersistSessionState(context.Background(), "other", func(context.Context, string) (*persistence.SessionEndState, error) {
			return &persistence.SessionEndState{StorageState: []byte(`{"marker":"independent"}`)}, nil
		})
		if err != nil {
			t.Errorf("independent profile save: %v", err)
		}
		close(release)
		if err := <-first; err != nil {
			t.Fatal(err)
		}
		if err := svc.PersistSessionState(context.Background(), "main", capture); err != nil {
			t.Fatal(err)
		}
		stored, err := svc.GetProfile(profile.ID)
		if err != nil || string(stored.StorageState) != `{"marker":"new"}` || captures.Load() != 2 {
			t.Fatalf("final snapshot = %+v, captures=%d, error=%v", stored, captures.Load(), err)
		}
	})
}

type checkpointFaultRepository struct {
	*persistence.MockRepository
	fail atomic.Bool
}

func (r *checkpointFaultRepository) Update(id persistence.ProfileID, modify func(*persistence.SessionProfile) error) (*persistence.SessionProfile, error) {
	if r.fail.Load() {
		return nil, errors.New("checkpoint disk fault")
	}
	return r.MockRepository.Update(id, modify)
}

// [REQ:BAS-RH-J06] Automatic saves use the same aggregate writer, retain the
// last good state on failure, and finish before their lifecycle owner returns.
func TestService_PeriodicCheckpointDurability(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		repo := &checkpointFaultRepository{MockRepository: persistence.NewMockRepository()}
		svc := NewService(repo, nil)
		profile, _ := svc.CreateProfile("Periodic")
		svc.SetActiveSession("session", string(profile.ID))
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		var captures atomic.Int32
		capture := func(context.Context, string) (*persistence.SessionEndState, error) {
			captures.Add(1)
			return &persistence.SessionEndState{StorageState: []byte(`{"cookies":[{"name":"identity","value":"retained"}],"origins":[]}`)}, nil
		}
		go func() { defer close(done); svc.RunCheckpoints(ctx, capture) }()
		defer func() { cancel(); <-done }()
		synctest.Wait()
		time.Sleep(5 * time.Second)
		synctest.Wait()
		stored, err := svc.GetProfile(profile.ID)
		if err != nil || len(stored.StorageState) == 0 || captures.Load() == 0 || svc.CheckpointHealth() != nil {
			t.Fatalf("checkpoint missing after recovery window: profile=%+v err=%v health=%v", stored, err, svc.CheckpointHealth())
		}
		repo.fail.Store(true)
		time.Sleep(3 * time.Second)
		synctest.Wait()
		preserved, _ := svc.GetProfile(profile.ID)
		if !reflect.DeepEqual(preserved, stored) || svc.CheckpointHealth() == nil {
			t.Fatal("failed checkpoint changed state or reported healthy")
		}
		repo.fail.Store(false)
		time.Sleep(3 * time.Second)
		synctest.Wait()
		if err := svc.CheckpointHealth(); err != nil {
			t.Fatalf("checkpoint did not recover after disk fault: %v", err)
		}
		cancel()
		<-done
		before := captures.Load()
		time.Sleep(5 * time.Second)
		if captures.Load() != before {
			t.Error("checkpoint capture outlived its lifecycle")
		}
	})
}

func TestService_PeriodicCheckpointRejectsAmbiguousWriter(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		svc := NewService(persistence.NewMockRepository(), nil)
		profile, _ := svc.CreateProfile("Shared")
		svc.SetActiveSession("first", string(profile.ID))
		started, release := make(chan struct{}), make(chan struct{})
		var captures atomic.Int32
		capture := func(context.Context, string) (*persistence.SessionEndState, error) {
			if captures.Add(1) == 1 {
				close(started)
				<-release
			}
			return &persistence.SessionEndState{StorageState: []byte(`{"marker":"automatic"}`)}, nil
		}
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() { defer close(done); svc.RunCheckpoints(ctx, capture) }()
		defer func() { cancel(); <-done }()
		<-started
		// Joining during capture must also invalidate the automatic commit.
		svc.SetActiveSession("second", string(profile.ID))
		close(release)
		synctest.Wait()
		stored, _ := svc.GetProfile(profile.ID)
		if len(stored.StorageState) != 0 || svc.CheckpointHealth() == nil {
			t.Fatal("automatic save selected an ambiguous writer")
		}
		time.Sleep(3 * time.Second)
		synctest.Wait()
		if captures.Load() != 1 {
			t.Fatal("ambiguous automatic writers entered browser capture")
		}
		if err := svc.PersistSessionState(context.Background(), "second", func(context.Context, string) (*persistence.SessionEndState, error) {
			return &persistence.SessionEndState{StorageState: []byte(`{"marker":"manual"}`)}, nil
		}); err != nil {
			t.Fatalf("manual save was disabled: %v", err)
		}
		svc.ClearActiveSession("second")
		time.Sleep(3 * time.Second)
		synctest.Wait()
		stored, _ = svc.GetProfile(profile.ID)
		if string(stored.StorageState) != `{"marker":"automatic"}` || svc.CheckpointHealth() != nil {
			t.Fatal("unique writer did not resume checkpointing")
		}
	})
}

func TestService_CheckpointShutdownCancelsCapture(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		svc := NewService(persistence.NewMockRepository(), nil)
		profile, _ := svc.CreateProfile("Shutdown")
		svc.SetActiveSession("session", string(profile.ID))
		ctx, cancel := context.WithCancel(context.Background())
		started, done := make(chan struct{}), make(chan struct{})
		go func() {
			defer close(done)
			svc.RunCheckpoints(ctx, func(captureCtx context.Context, _ string) (*persistence.SessionEndState, error) {
				close(started)
				<-captureCtx.Done()
				return nil, captureCtx.Err()
			})
		}()
		<-started
		cancel()
		<-done
		stored, _ := svc.GetProfile(profile.ID)
		if len(stored.StorageState) != 0 {
			t.Fatal("cancelled capture committed state")
		}
	})
}

func TestService_AddHistoryEntry(t *testing.T) {
	svc, _, mockClock := newTestService(t)

	// Create a profile
	created, _ := svc.CreateProfile("Test")

	// Add history entries
	entry1 := persistence.HistoryEntry{
		ID:        "entry-1",
		URL:       "https://example.com",
		Title:     "Example",
		Timestamp: mockClock.Now().Format(time.RFC3339),
	}

	updated, err := svc.AddHistoryEntry(created.ID, entry1)
	if err != nil {
		t.Fatalf("AddHistoryEntry failed: %v", err)
	}

	if len(updated.History) != 1 {
		t.Errorf("expected 1 history entry, got %d", len(updated.History))
	}

	// Add another entry - should be prepended (newest first)
	mockClock.Advance(time.Minute)
	entry2 := persistence.HistoryEntry{
		ID:        "entry-2",
		URL:       "https://example.org",
		Title:     "Another",
		Timestamp: mockClock.Now().Format(time.RFC3339),
	}

	updated, err = svc.AddHistoryEntry(created.ID, entry2)
	if err != nil {
		t.Fatalf("AddHistoryEntry failed: %v", err)
	}

	if len(updated.History) != 2 {
		t.Errorf("expected 2 history entries, got %d", len(updated.History))
	}
	if updated.History[0].ID != "entry-2" {
		t.Error("expected newest entry first")
	}
}

func TestService_AddHistoryEntry_Pruning(t *testing.T) {
	svc, _, mockClock := newTestService(t)

	// Create a profile with custom settings
	created, _ := svc.CreateProfile("Test")
	created.HistorySettings = &persistence.HistorySettings{
		MaxEntries:    5,
		RetentionDays: 30,
	}
	if _, err := svc.UpdateHistorySettings(created.ID, created.HistorySettings); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Add more entries than the limit
	for i := 0; i < 10; i++ {
		entry := persistence.HistoryEntry{
			ID:        "entry-" + string(rune('0'+i)),
			URL:       "https://example.com/" + string(rune('0'+i)),
			Title:     "Page",
			Timestamp: mockClock.Now().Format(time.RFC3339),
		}
		if _, err := svc.AddHistoryEntry(created.ID, entry); err != nil {
			t.Fatalf("AddHistoryEntry failed: %v", err)
		}
		mockClock.Advance(time.Minute)
	}

	// Verify pruning occurred
	updated, _ := svc.GetProfile(created.ID)
	if len(updated.History) > 5 {
		t.Errorf("expected at most 5 history entries, got %d", len(updated.History))
	}
}

func TestService_ClearHistory(t *testing.T) {
	svc, _, mockClock := newTestService(t)

	// Create a profile with history
	created, _ := svc.CreateProfile("Test")
	entry := persistence.HistoryEntry{
		ID:        "entry-1",
		URL:       "https://example.com",
		Timestamp: mockClock.Now().Format(time.RFC3339),
	}
	if _, err := svc.AddHistoryEntry(created.ID, entry); err != nil {
		t.Fatalf("AddHistoryEntry failed: %v", err)
	}

	// Clear history
	updated, err := svc.ClearHistory(created.ID)
	if err != nil {
		t.Fatalf("ClearHistory failed: %v", err)
	}

	if len(updated.History) != 0 {
		t.Error("expected history to be cleared")
	}
}

func TestService_DeleteHistoryEntry(t *testing.T) {
	svc, _, mockClock := newTestService(t)

	// Create a profile with multiple history entries
	created, _ := svc.CreateProfile("Test")
	for i := 0; i < 3; i++ {
		entry := persistence.HistoryEntry{
			ID:        "entry-" + string(rune('0'+i)),
			URL:       "https://example.com/" + string(rune('0'+i)),
			Timestamp: mockClock.Now().Format(time.RFC3339),
		}
		if _, err := svc.AddHistoryEntry(created.ID, entry); err != nil {
			t.Fatalf("AddHistoryEntry failed: %v", err)
		}
	}

	// Delete the middle entry
	updated, err := svc.DeleteHistoryEntry(created.ID, "entry-1")
	if err != nil {
		t.Fatalf("DeleteHistoryEntry failed: %v", err)
	}

	if len(updated.History) != 2 {
		t.Errorf("expected 2 history entries, got %d", len(updated.History))
	}

	// Verify the right entry was deleted
	for _, e := range updated.History {
		if e.ID == "entry-1" {
			t.Error("entry-1 should have been deleted")
		}
	}
}

func TestService_DeleteHistoryEntry_NotFound(t *testing.T) {
	svc, _, _ := newTestService(t)

	created, _ := svc.CreateProfile("Test")

	_, err := svc.DeleteHistoryEntry(created.ID, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent entry")
	}
}

func TestService_ActiveSessionRegistry_Concurrent(t *testing.T) {
	svc, _, _ := newTestService(t)

	// Create profiles
	for i := 0; i < 5; i++ {
		if _, err := svc.CreateProfile("Profile " + string(rune('A'+i))); err != nil {
			t.Fatalf("CreateProfile failed: %v", err)
		}
	}
	profiles, _ := svc.ListProfiles()

	// Concurrent session operations
	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			sessionID := "session-" + string(rune('0'+n%10))
			profileID := profiles[n%len(profiles)].ID

			svc.SetActiveSession(sessionID, string(profileID))
			got := svc.GetActiveSession(sessionID)
			if got != string(profileID) {
				errors <- nil // Race condition acceptable in this test
			}
			svc.ClearActiveSession(sessionID)
		}(i)
	}

	wg.Wait()
	close(errors)
}

func TestActiveSessionRegistry_Operations(t *testing.T) {
	registry := NewActiveSessionRegistry()

	// Test Set and Get
	registry.Set("session-1", "profile-a", time.Now())
	if got := registry.Get("session-1"); got != "profile-a" {
		t.Errorf("expected profile-a, got %s", got)
	}

	// Test GetByProfile (reverse lookup)
	if got := registry.GetByProfile("profile-a"); got != "session-1" {
		t.Errorf("expected session-1, got %s", got)
	}

	// Test Clear
	registry.Clear("session-1")
	if got := registry.Get("session-1"); got != "" {
		t.Error("expected empty after clear")
	}

	// Test ClearForProfile
	registry.Set("session-2", "profile-b", time.Now())
	registry.Set("session-3", "profile-b", time.Now())
	registry.ClearForProfile("profile-b")
	if got := registry.Get("session-2"); got != "" {
		t.Error("expected session-2 to be cleared")
	}
	if got := registry.Get("session-3"); got != "" {
		t.Error("expected session-3 to be cleared")
	}
}

func TestService_PruneHistoryByTTL(t *testing.T) {
	svc, _, mockClock := newTestService(t)

	// Create profile with history settings
	created, _ := svc.CreateProfile("Test")
	created.HistorySettings = &persistence.HistorySettings{
		MaxEntries:    100,
		RetentionDays: 7, // 7 day TTL
	}
	if _, err := svc.UpdateHistorySettings(created.ID, created.HistorySettings); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Add entry that's 10 days old
	oldEntry := persistence.HistoryEntry{
		ID:        "old-entry",
		URL:       "https://old.com",
		Timestamp: mockClock.Now().AddDate(0, 0, -10).Format(time.RFC3339),
	}
	if _, err := svc.AddHistoryEntry(created.ID, oldEntry); err != nil {
		t.Fatalf("AddHistoryEntry failed: %v", err)
	}

	// Add recent entry
	recentEntry := persistence.HistoryEntry{
		ID:        "recent-entry",
		URL:       "https://recent.com",
		Timestamp: mockClock.Now().Format(time.RFC3339),
	}
	if _, err := svc.AddHistoryEntry(created.ID, recentEntry); err != nil {
		t.Fatalf("AddHistoryEntry failed: %v", err)
	}

	// Get history with pruning
	entries, _, err := svc.GetHistoryWithPruning(created.ID)
	if err != nil {
		t.Fatalf("GetHistoryWithPruning failed: %v", err)
	}

	// Only recent entry should remain after TTL pruning
	if len(entries) != 1 {
		t.Errorf("expected 1 entry after TTL pruning, got %d", len(entries))
	}
	if entries[0].ID != "recent-entry" {
		t.Error("expected recent entry to remain")
	}
}

func TestService_Touch(t *testing.T) {
	svc, _, mockClock := newTestService(t)

	created, _ := svc.CreateProfile("Test")
	originalLastUsed := created.LastUsedAt

	mockClock.Advance(time.Hour)

	touched, err := svc.Touch(created.ID)
	if err != nil {
		t.Fatalf("Touch failed: %v", err)
	}

	if touched.LastUsedAt.Equal(originalLastUsed) {
		t.Error("expected LastUsedAt to be updated")
	}
	if touched.UpdatedAt.Equal(created.UpdatedAt) {
		t.Error("expected UpdatedAt to be updated")
	}
}

func TestService_SaveStorageState(t *testing.T) {
	svc, _, _ := newTestService(t)

	created, _ := svc.CreateProfile("Test")

	storageState := []byte(`{"cookies":[{"name":"session","value":"abc123"}]}`)
	updated, err := svc.SaveStorageState(created.ID, storageState)
	if err != nil {
		t.Fatalf("SaveStorageState failed: %v", err)
	}

	if string(updated.StorageState) != string(storageState) {
		t.Error("storage state mismatch")
	}
}

func TestService_SaveOpenTabs(t *testing.T) {
	svc, _, _ := newTestService(t)

	created, _ := svc.CreateProfile("Test")

	tabs := []persistence.TabState{
		{URL: "https://example.com", Title: "Example", IsActive: true, Order: 0},
		{URL: "https://test.com", Title: "Test", Order: 1},
	}

	updated, err := svc.SaveOpenTabs(created.ID, tabs)
	if err != nil {
		t.Fatalf("SaveOpenTabs failed: %v", err)
	}

	if len(updated.OpenTabs) != 2 {
		t.Errorf("expected 2 tabs, got %d", len(updated.OpenTabs))
	}
}

func TestService_SaveOpenTabs_LimitsCount(t *testing.T) {
	svc, _, _ := newTestService(t)

	created, _ := svc.CreateProfile("Test")

	// Create more tabs than the limit
	tabs := make([]persistence.TabState, persistence.MaxRestoredTabs+10)
	for i := range tabs {
		tabs[i] = persistence.TabState{URL: "https://example.com", Order: i}
	}

	updated, err := svc.SaveOpenTabs(created.ID, tabs)
	if err != nil {
		t.Fatalf("SaveOpenTabs failed: %v", err)
	}

	if len(updated.OpenTabs) > persistence.MaxRestoredTabs {
		t.Errorf("expected at most %d tabs, got %d", persistence.MaxRestoredTabs, len(updated.OpenTabs))
	}
}

// =============================================================================
// Storage State Masking Tests
// =============================================================================

func TestService_MaskStorageState_HidesHttpOnlyCookies(t *testing.T) {
	svc, _, _ := newTestService(t)

	// Storage state with both httpOnly and non-httpOnly cookies
	storageState := []byte(`{
		"cookies": [
			{
				"name": "session_token",
				"value": "secret-session-value",
				"domain": ".example.com",
				"path": "/",
				"expires": 1735689600,
				"httpOnly": true,
				"secure": true,
				"sameSite": "Lax"
			},
			{
				"name": "preference",
				"value": "dark-mode",
				"domain": ".example.com",
				"path": "/",
				"expires": 1735689600,
				"httpOnly": false,
				"secure": false,
				"sameSite": "Lax"
			}
		],
		"origins": [
			{
				"origin": "https://example.com",
				"localStorage": [
					{"name": "theme", "value": "dark"},
					{"name": "lang", "value": "en"}
				]
			}
		]
	}`)

	masked, err := svc.MaskStorageState(storageState)
	if err != nil {
		t.Fatalf("MaskStorageState failed: %v", err)
	}

	// Verify we have 2 cookies
	if len(masked.Cookies) != 2 {
		t.Errorf("expected 2 cookies, got %d", len(masked.Cookies))
	}

	// Find httpOnly cookie
	var httpOnlyCookie, normalCookie *MaskedCookie
	for i := range masked.Cookies {
		if masked.Cookies[i].Name == "session_token" {
			httpOnlyCookie = &masked.Cookies[i]
		} else if masked.Cookies[i].Name == "preference" {
			normalCookie = &masked.Cookies[i]
		}
	}

	// httpOnly cookie value should be masked
	if httpOnlyCookie == nil {
		t.Fatal("expected to find session_token cookie")
	}
	if httpOnlyCookie.Value != "[HIDDEN]" {
		t.Errorf("expected httpOnly cookie value to be '[HIDDEN]', got '%s'", httpOnlyCookie.Value)
	}
	if !httpOnlyCookie.ValueMasked {
		t.Error("expected ValueMasked to be true for httpOnly cookie")
	}
	if !httpOnlyCookie.HttpOnly {
		t.Error("expected HttpOnly to be true")
	}

	// Non-httpOnly cookie value should be visible
	if normalCookie == nil {
		t.Fatal("expected to find preference cookie")
	}
	if normalCookie.Value != "dark-mode" {
		t.Errorf("expected cookie value 'dark-mode', got '%s'", normalCookie.Value)
	}
	if normalCookie.ValueMasked {
		t.Error("expected ValueMasked to be false for non-httpOnly cookie")
	}

	// Verify localStorage is passed through
	if len(masked.Origins) != 1 {
		t.Errorf("expected 1 origin, got %d", len(masked.Origins))
	}
	if masked.Origins[0].Origin != "https://example.com" {
		t.Errorf("expected origin 'https://example.com', got '%s'", masked.Origins[0].Origin)
	}
	if len(masked.Origins[0].LocalStorage) != 2 {
		t.Errorf("expected 2 localStorage items, got %d", len(masked.Origins[0].LocalStorage))
	}

	// Verify stats
	if masked.Stats.CookieCount != 2 {
		t.Errorf("expected cookie count 2, got %d", masked.Stats.CookieCount)
	}
	if masked.Stats.LocalStorageCount != 2 {
		t.Errorf("expected localStorage count 2, got %d", masked.Stats.LocalStorageCount)
	}
	if masked.Stats.OriginCount != 1 {
		t.Errorf("expected origin count 1, got %d", masked.Stats.OriginCount)
	}
}

func TestService_MaskStorageState_EmptyInput(t *testing.T) {
	svc, _, _ := newTestService(t)

	// Test nil input
	masked, err := svc.MaskStorageState(nil)
	if err != nil {
		t.Fatalf("MaskStorageState failed for nil input: %v", err)
	}
	if len(masked.Cookies) != 0 {
		t.Error("expected empty cookies for nil input")
	}
	if len(masked.Origins) != 0 {
		t.Error("expected empty origins for nil input")
	}

	// Test empty slice
	masked, err = svc.MaskStorageState([]byte{})
	if err != nil {
		t.Fatalf("MaskStorageState failed for empty input: %v", err)
	}
	if len(masked.Cookies) != 0 {
		t.Error("expected empty cookies for empty input")
	}
}

func TestService_MaskStorageState_InvalidJSON(t *testing.T) {
	svc, _, _ := newTestService(t)

	invalidJSON := []byte(`{invalid json}`)
	_, err := svc.MaskStorageState(invalidJSON)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestService_MaskStorageState_PreservesMetadata(t *testing.T) {
	svc, _, _ := newTestService(t)

	storageState := []byte(`{
		"cookies": [
			{
				"name": "test_cookie",
				"value": "test_value",
				"domain": ".test.com",
				"path": "/api",
				"expires": 1735689600.5,
				"httpOnly": false,
				"secure": true,
				"sameSite": "Strict"
			}
		],
		"origins": []
	}`)

	masked, err := svc.MaskStorageState(storageState)
	if err != nil {
		t.Fatalf("MaskStorageState failed: %v", err)
	}

	if len(masked.Cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(masked.Cookies))
	}

	cookie := masked.Cookies[0]
	if cookie.Domain != ".test.com" {
		t.Errorf("expected domain '.test.com', got '%s'", cookie.Domain)
	}
	if cookie.Path != "/api" {
		t.Errorf("expected path '/api', got '%s'", cookie.Path)
	}
	if cookie.Expires != 1735689600.5 {
		t.Errorf("expected expires 1735689600.5, got %f", cookie.Expires)
	}
	if !cookie.Secure {
		t.Error("expected Secure to be true")
	}
	if cookie.SameSite != "Strict" {
		t.Errorf("expected SameSite 'Strict', got '%s'", cookie.SameSite)
	}
}
