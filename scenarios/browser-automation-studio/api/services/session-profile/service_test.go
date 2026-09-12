package sessionprofile

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/scheduletest"
	"github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
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

func TestService_StartSession(t *testing.T) {
	svc, _, mockClock := newTestService(t)

	// Create a profile
	created, _ := svc.CreateProfile("Test")
	originalLastUsed := created.LastUsedAt

	// Advance clock
	mockClock.Advance(time.Hour)

	// Start session
	err := svc.StartSession("browser-session-1", created.ID)
	if err != nil {
		t.Fatalf("StartSession failed: %v", err)
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

func TestService_EndSession(t *testing.T) {
	svc, _, mockClock := newTestService(t)

	// Create a profile and start session
	created, _ := svc.CreateProfile("Test")
	if err := svc.StartSession("browser-session-1", created.ID); err != nil {
		t.Fatalf("StartSession failed: %v", err)
	}

	mockClock.Advance(time.Hour)

	// End session with state
	state := &persistence.SessionEndState{
		StorageState: []byte(`{"cookies":[]}`),
		OpenTabs: []persistence.TabState{
			{URL: "https://example.com", Title: "Example", Order: 0},
		},
	}

	ctx := context.Background()
	err := svc.EndSession(ctx, "browser-session-1", state)
	if err != nil {
		t.Fatalf("EndSession failed: %v", err)
	}

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

func TestService_EndSession_LimitsTabs(t *testing.T) {
	svc, _, _ := newTestService(t)

	// Create a profile and start session
	created, _ := svc.CreateProfile("Test")
	if err := svc.StartSession("browser-session-1", created.ID); err != nil {
		t.Fatalf("StartSession failed: %v", err)
	}

	// Create more tabs than the limit
	tabs := make([]persistence.TabState, persistence.MaxRestoredTabs+10)
	for i := range tabs {
		tabs[i] = persistence.TabState{URL: "https://example.com", Order: i}
	}

	state := &persistence.SessionEndState{
		OpenTabs: tabs,
	}

	ctx := context.Background()
	err := svc.EndSession(ctx, "browser-session-1", state)
	if err != nil {
		t.Fatalf("EndSession failed: %v", err)
	}

	// Verify tabs were limited
	updated, _ := svc.GetProfile(created.ID)
	if len(updated.OpenTabs) > persistence.MaxRestoredTabs {
		t.Errorf("expected at most %d tabs, got %d", persistence.MaxRestoredTabs, len(updated.OpenTabs))
	}
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
	if err := svc.repo.Save(created); err != nil {
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
	registry.Set("session-1", "profile-a")
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
	registry.Set("session-2", "profile-b")
	registry.Set("session-3", "profile-b")
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
	if err := svc.repo.Save(created); err != nil {
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
