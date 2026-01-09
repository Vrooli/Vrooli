package archiveingestion

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSessionProfileStore_ActiveSessionTracking(t *testing.T) {
	// Create a temp directory for the test store
	tempDir, err := os.MkdirTemp("", "session-profiles-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store := NewSessionProfileStore(tempDir, nil)

	t.Run("SetActiveSession and GetActiveSession", func(t *testing.T) {
		// Initially should return empty
		got := store.GetActiveSession("session-1")
		if got != "" {
			t.Errorf("GetActiveSession() = %q, want empty string", got)
		}

		// Set a session-profile mapping
		store.SetActiveSession("session-1", "profile-a")

		// Should now return the profile
		got = store.GetActiveSession("session-1")
		if got != "profile-a" {
			t.Errorf("GetActiveSession() = %q, want %q", got, "profile-a")
		}

		// Different session should still return empty
		got = store.GetActiveSession("session-2")
		if got != "" {
			t.Errorf("GetActiveSession(session-2) = %q, want empty string", got)
		}
	})

	t.Run("SetActiveSession ignores empty values", func(t *testing.T) {
		// Empty session ID should not create mapping
		store.SetActiveSession("", "profile-b")
		got := store.GetActiveSession("")
		if got != "" {
			t.Errorf("GetActiveSession('') = %q, want empty string", got)
		}

		// Empty profile ID should not create mapping
		store.SetActiveSession("session-3", "")
		got = store.GetActiveSession("session-3")
		if got != "" {
			t.Errorf("GetActiveSession(session-3) = %q, want empty string", got)
		}
	})

	t.Run("ClearActiveSession removes mapping and returns profile ID", func(t *testing.T) {
		store.SetActiveSession("session-4", "profile-c")

		// Clear should return the profile ID
		cleared := store.ClearActiveSession("session-4")
		if cleared != "profile-c" {
			t.Errorf("ClearActiveSession() = %q, want %q", cleared, "profile-c")
		}

		// Should now return empty
		got := store.GetActiveSession("session-4")
		if got != "" {
			t.Errorf("GetActiveSession() after clear = %q, want empty string", got)
		}

		// Clearing again should return empty
		cleared = store.ClearActiveSession("session-4")
		if cleared != "" {
			t.Errorf("ClearActiveSession() second call = %q, want empty string", cleared)
		}
	})

	t.Run("ClearSessionsForProfile removes all sessions for a profile", func(t *testing.T) {
		// Set up multiple sessions pointing to different profiles
		store.SetActiveSession("sess-a1", "profile-x")
		store.SetActiveSession("sess-a2", "profile-x")
		store.SetActiveSession("sess-b1", "profile-y")
		store.SetActiveSession("sess-b2", "profile-y")
		store.SetActiveSession("sess-c1", "profile-z")

		// Clear all sessions for profile-x
		store.ClearSessionsForProfile("profile-x")

		// Sessions for profile-x should be gone
		if got := store.GetActiveSession("sess-a1"); got != "" {
			t.Errorf("GetActiveSession(sess-a1) = %q, want empty string", got)
		}
		if got := store.GetActiveSession("sess-a2"); got != "" {
			t.Errorf("GetActiveSession(sess-a2) = %q, want empty string", got)
		}

		// Sessions for other profiles should remain
		if got := store.GetActiveSession("sess-b1"); got != "profile-y" {
			t.Errorf("GetActiveSession(sess-b1) = %q, want %q", got, "profile-y")
		}
		if got := store.GetActiveSession("sess-b2"); got != "profile-y" {
			t.Errorf("GetActiveSession(sess-b2) = %q, want %q", got, "profile-y")
		}
		if got := store.GetActiveSession("sess-c1"); got != "profile-z" {
			t.Errorf("GetActiveSession(sess-c1) = %q, want %q", got, "profile-z")
		}
	})

	t.Run("ClearSessionsForProfile handles non-existent profile", func(t *testing.T) {
		// Should not panic or error
		store.ClearSessionsForProfile("non-existent-profile")
	})
}

func TestSessionProfileStore_Integration(t *testing.T) {
	// Create a temp directory for the test store
	tempDir, err := os.MkdirTemp("", "session-profiles-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store := NewSessionProfileStore(tempDir, nil)

	t.Run("Create profile and track active session", func(t *testing.T) {
		// Create a profile
		profile, err := store.Create("Test Profile")
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if profile.ID == "" {
			t.Error("Create() returned profile with empty ID")
		}
		if profile.Name != "Test Profile" {
			t.Errorf("Create() profile.Name = %q, want %q", profile.Name, "Test Profile")
		}

		// Verify profile file was created
		profilePath := filepath.Join(tempDir, profile.ID+".json")
		if _, err := os.Stat(profilePath); os.IsNotExist(err) {
			t.Errorf("Profile file not created at %s", profilePath)
		}

		// Track active session with this profile
		store.SetActiveSession("browser-session-123", profile.ID)

		// Verify we can retrieve the mapping
		got := store.GetActiveSession("browser-session-123")
		if got != profile.ID {
			t.Errorf("GetActiveSession() = %q, want %q", got, profile.ID)
		}

		// Clear when "closing" the browser session
		cleared := store.ClearActiveSession("browser-session-123")
		if cleared != profile.ID {
			t.Errorf("ClearActiveSession() = %q, want %q", cleared, profile.ID)
		}

		// Profile should still exist (active session tracking is separate from profile storage)
		retrieved, err := store.Get(profile.ID)
		if err != nil {
			t.Errorf("Get() error = %v", err)
		}
		if retrieved == nil {
			t.Error("Get() returned nil profile")
		}
	})

	t.Run("Delete profile clears active sessions", func(t *testing.T) {
		// Create a profile
		profile, err := store.Create("Deletable Profile")
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// Set up multiple browser sessions using this profile
		store.SetActiveSession("del-sess-1", profile.ID)
		store.SetActiveSession("del-sess-2", profile.ID)

		// Clear active sessions for this profile (simulating what would happen before delete)
		store.ClearSessionsForProfile(profile.ID)

		// Delete the profile
		if err := store.Delete(profile.ID); err != nil {
			t.Errorf("Delete() error = %v", err)
		}

		// Active sessions should be gone
		if got := store.GetActiveSession("del-sess-1"); got != "" {
			t.Errorf("GetActiveSession(del-sess-1) after delete = %q, want empty", got)
		}
		if got := store.GetActiveSession("del-sess-2"); got != "" {
			t.Errorf("GetActiveSession(del-sess-2) after delete = %q, want empty", got)
		}

		// Profile should no longer exist
		_, err = store.Get(profile.ID)
		if err == nil {
			t.Error("Get() after delete should return error")
		}
	})
}

func TestSessionProfileStore_Concurrency(t *testing.T) {
	// Create a temp directory for the test store
	tempDir, err := os.MkdirTemp("", "session-profiles-concurrent-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store := NewSessionProfileStore(tempDir, nil)

	t.Run("Concurrent active session operations", func(t *testing.T) {
		const numGoroutines = 50
		const numOperations = 100

		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				for j := 0; j < numOperations; j++ {
					sessionID := "concurrent-session"
					profileID := "concurrent-profile"

					// Mix of operations
					switch j % 4 {
					case 0:
						store.SetActiveSession(sessionID, profileID)
					case 1:
						_ = store.GetActiveSession(sessionID)
					case 2:
						_ = store.ClearActiveSession(sessionID)
					case 3:
						store.ClearSessionsForProfile(profileID)
					}
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// If we got here without deadlock or panic, the test passes
	})
}
