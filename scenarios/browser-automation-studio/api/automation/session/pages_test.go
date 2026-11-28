package session

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/domain"
)

func TestNewPageTrackerFromEventsReplaysOrderedPageState(t *testing.T) {
	t.Parallel()

	sessionID := "recording-replay"
	initialID, popupID := uuid.New(), uuid.New()
	start := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	closedAt := start.Add(4 * time.Second)
	activePopup := popupID
	events := []domain.PageEvent{
		{ID: uuid.New(), Type: domain.PageEventCreated, PageID: initialID, DriverPageID: "driver-initial", IsInitial: true, URL: "https://start.test", Timestamp: start},
		{ID: uuid.New(), Type: domain.PageEventCreated, PageID: popupID, DriverPageID: "driver-popup", URL: "https://popup.test", OpenerID: &initialID, Timestamp: start.Add(time.Second)},
		{ID: uuid.New(), Type: domain.PageEventNavigated, PageID: popupID, DriverPageID: "driver-popup", URL: "https://popup.test/next", Title: "Popup next", Timestamp: start.Add(2 * time.Second)},
		{ID: uuid.New(), Type: domain.PageEventActivated, PageID: popupID, DriverPageID: "driver-popup", ActivePageID: popupID, Timestamp: start.Add(3 * time.Second)},
		{ID: uuid.New(), Type: domain.PageEventClosed, PageID: initialID, DriverPageID: "driver-initial", ActivePageID: activePopup, Timestamp: closedAt},
	}

	tracker, err := NewPageTrackerFromEvents(sessionID, events)
	require.NoError(t, err)
	assert.Equal(t, initialID, tracker.GetInitialPageID())
	assert.Equal(t, popupID, tracker.GetActivePageID())
	assert.Equal(t, "driver-popup", tracker.GetDriverPageID(popupID))
	assert.Equal(t, popupID, *tracker.GetPageIDByDriverID("driver-popup"))
	require.NotNil(t, tracker.GetPageIDByDriverID("driver-initial"))
	assert.Equal(t, initialID, *tracker.GetPageIDByDriverID("driver-initial"))

	initial, ok := tracker.GetPage(initialID)
	require.True(t, ok)
	assert.Equal(t, domain.PageStatusClosed, initial.Status)
	require.NotNil(t, initial.ClosedAt)
	assert.Equal(t, closedAt, *initial.ClosedAt)

	popup, ok := tracker.GetPage(popupID)
	require.True(t, ok)
	assert.Equal(t, "https://popup.test/next", popup.URL)
	assert.Equal(t, "Popup next", popup.Title)
	assert.Equal(t, initialID, *popup.OpenerID)
	assert.Equal(t, sessionID, popup.SessionID)
}

func TestNewPageTrackerFromEventsRejectsIncompleteOrInconsistentJournal(t *testing.T) {
	t.Parallel()

	initialID := uuid.New()
	at := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	validInitial := domain.PageEvent{ID: uuid.New(), Type: domain.PageEventCreated, PageID: initialID, IsInitial: true, Timestamp: at}

	for _, tc := range []struct {
		name   string
		events []domain.PageEvent
	}{
		{name: "missing initial receipt", events: []domain.PageEvent{{ID: uuid.New(), Type: domain.PageEventCreated, PageID: uuid.New(), Timestamp: at}}},
		{name: "navigation before create", events: []domain.PageEvent{{ID: uuid.New(), Type: domain.PageEventNavigated, PageID: initialID, Timestamp: at}, validInitial}},
		{name: "unknown activation", events: []domain.PageEvent{validInitial, {ID: uuid.New(), Type: domain.PageEventActivated, PageID: uuid.New(), Timestamp: at.Add(time.Second)}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tracker, err := NewPageTrackerFromEvents("recording-replay", tc.events); err == nil || tracker != nil {
				t.Fatalf("replay accepted inconsistent journal: tracker=%v err=%v", tracker, err)
			}
		})
	}
}

func TestNewPageTracker(t *testing.T) {
	t.Parallel()

	pt := NewPageTracker("session-123", "https://example.com")

	// Should have exactly one page
	if pt.PageCount() != 1 {
		t.Errorf("expected 1 page, got %d", pt.PageCount())
	}

	// Initial page should be active
	initialPage := pt.GetActivePage()
	if initialPage == nil {
		t.Fatal("active page should not be nil")
	}
	if !initialPage.IsInitial {
		t.Error("initial page should have IsInitial = true")
	}
	if initialPage.URL != "https://example.com" {
		t.Errorf("expected URL 'https://example.com', got '%s'", initialPage.URL)
	}
	if initialPage.SessionID != "session-123" {
		t.Errorf("expected SessionID 'session-123', got '%s'", initialPage.SessionID)
	}
	if initialPage.Status != domain.PageStatusActive {
		t.Errorf("expected status 'active', got '%s'", initialPage.Status)
	}
}

func TestInitialDriverObservationPersistsBeforeRegisteringIdentity(t *testing.T) {
	pt := NewPageTracker("session-initial", "about:blank")
	initialID := pt.GetInitialPageID()
	at := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	event := domain.DriverPageEvent{
		SessionID: "session-initial", DriverPageID: "driver-initial", EventType: "initial",
		URL: "https://example.test/", Title: "Example", Timestamp: at.Format(time.RFC3339Nano),
	}
	before, active := pt.Snapshot(false)
	appendErr := fmt.Errorf("journal unavailable")
	if observation, err := pt.ApplyDriverEvent("session-initial", event, at, func(*domain.PageEvent) error { return appendErr }); err == nil || observation != nil {
		t.Fatalf("initial observation append failure result=(%+v,%v)", observation, err)
	}
	after, activeAfter := pt.Snapshot(false)
	if !assert.Equal(t, before, after) || active != activeAfter || pt.GetPageIDByDriverID("driver-initial") != nil || pt.GetDriverPageID(initialID) != "" {
		t.Fatal("failed initial observation changed tracker state or driver mapping")
	}

	var stored *domain.PageEvent
	appendCalls := 0
	persist := func(candidate *domain.PageEvent) error {
		appendCalls++
		if stored != nil && !reflect.DeepEqual(stored, candidate) {
			return fmt.Errorf("conflicting replay")
		}
		if stored == nil {
			copy := *candidate
			stored = &copy
		}
		return nil
	}
	observation, err := pt.ApplyDriverEvent("session-initial", event, at, persist)
	if err != nil {
		t.Fatal(err)
	}
	if observation == nil || observation.PageID != initialID || observation.DriverPageID != "driver-initial" || !observation.IsInitial {
		t.Fatalf("initial page journal identity missing: %+v", observation)
	}
	if got := pt.GetPageIDByDriverID("driver-initial"); got == nil || *got != initialID || pt.GetDriverPageID(initialID) != "driver-initial" {
		t.Fatal("initial mapping was not installed after successful journal append")
	}
	page, ok := pt.GetPage(initialID)
	if !ok || page.URL != event.URL || page.Title != event.Title || page.DriverPageID != event.DriverPageID {
		t.Fatalf("initial snapshot was not applied after append: %+v", page)
	}
	if _, err := pt.ApplyDriverEvent("session-initial", event, at, persist); err != nil {
		t.Fatalf("exact replay failed: %v", err)
	}
	if appendCalls != 2 {
		t.Fatalf("expected one initial append and one idempotent replay, got %d calls", appendCalls)
	}
	conflict := event
	conflict.URL = "https://example.test/conflict"
	beforeConflict, activeConflict := pt.Snapshot(false)
	if _, err := pt.ApplyDriverEvent("session-initial", conflict, at, persist); err == nil {
		t.Fatal("conflicting initial replay was accepted")
	}
	afterConflict, activeAfterConflict := pt.Snapshot(false)
	if !assert.Equal(t, beforeConflict, afterConflict) || activeConflict != activeAfterConflict {
		t.Fatal("conflicting initial replay mutated tracker state")
	}
}

func TestActivePageSelectionCommitsBeforeTrackerMutation(t *testing.T) {
	pt := NewPageTracker("session-selection", "about:blank")
	page := pt.AddPage(&domain.Page{DriverPageID: "driver-second", URL: "https://second.test"})
	initial := pt.GetActivePageID()
	at := time.Date(2026, 11, 21, 13, 0, 0, 0, time.UTC)
	appendErr := fmt.Errorf("journal unavailable")
	if err := pt.SetActivePageAt(page.ID, at, func(*domain.PageEvent) error { return appendErr }); err == nil {
		t.Fatal("selection append failure was accepted")
	}
	if got := pt.GetActivePageID(); got != initial {
		t.Fatalf("append failure changed active page: got %s want %s", got, initial)
	}
	var stored *domain.PageEvent
	appendCount := 0
	persist := func(candidate *domain.PageEvent) error {
		appendCount++
		copy := *candidate
		stored = &copy
		return nil
	}
	if err := pt.SetActivePageAt(page.ID, at, persist); err != nil {
		t.Fatal(err)
	}
	if stored == nil || stored.Type != domain.PageEventActivated || stored.PageID != page.ID || stored.ActivePageID != page.ID || stored.DriverPageID != "driver-second" {
		t.Fatalf("selection journal missing stable page state: %+v", stored)
	}
	if got := pt.GetActivePageID(); got != page.ID {
		t.Fatalf("selection was not applied after append: got %s want %s", got, page.ID)
	}
	if err := pt.SetActivePageAt(page.ID, at, persist); err != nil {
		t.Fatalf("exact selection replay should be a no-op: %v", err)
	}
	if appendCount != 1 {
		t.Fatalf("same selected page appended duplicate journal row: %d", appendCount)
	}
}

func TestPageCloseJournalCarriesResultingActiveSelection(t *testing.T) {
	pt := NewPageTracker("session-close-selection", "about:blank")
	initial := pt.GetActivePageID()
	other := pt.AddPage(&domain.Page{DriverPageID: "driver-second", URL: "https://second.test"})
	// Select the added tab through the compatibility helper to establish state.
	require.NoError(t, pt.SetActivePage(other.ID))
	at := time.Date(2026, 11, 21, 13, 1, 0, 0, time.UTC)
	appendErr := fmt.Errorf("journal unavailable")
	if _, err := pt.ClosePageAtPersist(other.ID, at, func(*domain.PageEvent) error { return appendErr }); err == nil {
		t.Fatal("close append failure was accepted")
	}
	page, ok := pt.GetPage(other.ID)
	if !ok || page.Status != domain.PageStatusActive || pt.GetActivePageID() != other.ID {
		t.Fatal("close append failure changed page lifecycle or active selection")
	}
	var stored *domain.PageEvent
	event, err := pt.ClosePageAtPersist(other.ID, at, func(candidate *domain.PageEvent) error { stored = candidate; return nil })
	require.NoError(t, err)
	if stored == nil || stored.ActivePageID != initial || event.ActivePageID != initial || pt.GetActivePageID() != initial {
		t.Fatalf("close journal did not capture resulting selection: stored=%+v event=%+v active=%s", stored, event, pt.GetActivePageID())
	}
}

func TestPageReceiptJournalPrecedesIdentityRegistration(t *testing.T) {
	pt := NewPageTracker("session-page-receipt", "about:blank")
	at := time.Date(2026, 11, 21, 13, 2, 0, 0, time.UTC)
	input := &domain.Page{DriverPageID: "driver-new", URL: "https://new.test", Title: "New", FaviconURL: "https://new.test/icon.ico"}
	appendErr := fmt.Errorf("journal unavailable")
	if page, err := pt.AddPageAt(input, at, func(*domain.PageEvent) error { return appendErr }); err == nil || page != nil {
		t.Fatal("page receipt append failure was accepted")
	}
	if pt.GetPageIDByDriverID(input.DriverPageID) != nil || pt.PageCount() != 1 {
		t.Fatal("failed page receipt append registered a partial page")
	}
	var stored *domain.PageEvent
	page, err := pt.AddPageAt(input, at, func(event *domain.PageEvent) error { stored = event; return nil })
	require.NoError(t, err)
	if stored == nil || stored.PageID != page.ID || stored.DriverPageID != input.DriverPageID || stored.URL != input.URL || stored.Title != input.Title || stored.Timestamp != at {
		t.Fatalf("page creation event lacks reconstruction identity: %+v", stored)
	}
	if id := pt.GetPageIDByDriverID(input.DriverPageID); id == nil || *id != page.ID {
		t.Fatal("page identity was not published after journal append")
	}
}

func TestPageMetadataJournalFailureLeavesSnapshotUnchanged(t *testing.T) {
	pt := NewPageTracker("session-metadata", "https://before.test")
	id := pt.GetInitialPageID()
	before, ok := pt.GetPage(id)
	require.True(t, ok)
	appendErr := fmt.Errorf("journal unavailable")
	favicon := "https://after.test/icon.ico"
	err := pt.UpdatePageInfoAt(id, "https://after.test", "After", &favicon, time.Now(), func(*domain.PageEvent) error { return appendErr })
	require.ErrorIs(t, err, appendErr)
	after, ok := pt.GetPage(id)
	require.True(t, ok)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("failed metadata append changed page snapshot: before=%+v after=%+v", before, after)
	}
}

func TestPageTracker_AddPage(t *testing.T) {
	t.Parallel()

	pt := NewPageTracker("session-123", "https://example.com")

	newPageID := uuid.New()
	newPage := &domain.Page{
		ID:        newPageID,
		SessionID: "session-123",
		URL:       "https://example.com/new",
		Title:     "New Page",
		CreatedAt: time.Now(),
		IsInitial: false,
		Status:    domain.PageStatusActive,
	}

	pt.AddPage(newPage)

	if pt.PageCount() != 2 {
		t.Errorf("expected 2 pages, got %d", pt.PageCount())
	}

	retrieved, ok := pt.GetPage(newPageID)
	if !ok {
		t.Fatal("page should exist after adding")
	}
	if retrieved.URL != "https://example.com/new" {
		t.Errorf("expected URL 'https://example.com/new', got '%s'", retrieved.URL)
	}
}

func TestPageTracker_SetActivePage(t *testing.T) {
	t.Parallel()

	pt := NewPageTracker("session-123", "https://example.com")

	// Add a second page
	newPageID := uuid.New()
	pt.AddPage(&domain.Page{
		ID:        newPageID,
		SessionID: "session-123",
		URL:       "https://example.com/new",
		CreatedAt: time.Now(),
		Status:    domain.PageStatusActive,
	})

	// Switch to new page
	err := pt.SetActivePage(newPageID)
	if err != nil {
		t.Fatalf("SetActivePage failed: %v", err)
	}

	if pt.GetActivePageID() != newPageID {
		t.Error("active page ID should have changed")
	}
}

func TestPageTracker_SetActivePage_Errors(t *testing.T) {
	t.Parallel()

	pt := NewPageTracker("session-123", "https://example.com")

	// Try to set a non-existent page as active
	nonExistentID := uuid.New()
	err := pt.SetActivePage(nonExistentID)
	if err == nil {
		t.Error("expected error for non-existent page")
	}

	// Add and close a page, then try to activate it
	closedPageID := uuid.New()
	pt.AddPage(&domain.Page{
		ID:        closedPageID,
		SessionID: "session-123",
		URL:       "https://example.com/closed",
		CreatedAt: time.Now(),
		Status:    domain.PageStatusActive,
	})
	_, _ = pt.ClosePage(closedPageID)

	err = pt.SetActivePage(closedPageID)
	if err == nil {
		t.Error("expected error for closed page")
	}
}

func TestPageTracker_ClosePage(t *testing.T) {
	t.Parallel()

	pt := NewPageTracker("session-123", "https://example.com")

	// Add a second page and make it active
	newPageID := uuid.New()
	pt.AddPage(&domain.Page{
		ID:        newPageID,
		SessionID: "session-123",
		URL:       "https://example.com/new",
		CreatedAt: time.Now(),
		Status:    domain.PageStatusActive,
	})
	_ = pt.SetActivePage(newPageID)

	// Close the active page
	_, err := pt.ClosePage(newPageID)
	if err != nil {
		t.Fatalf("ClosePage failed: %v", err)
	}

	// Verify page is closed
	page, _ := pt.GetPage(newPageID)
	if page.Status != domain.PageStatusClosed {
		t.Error("page should be closed")
	}
	if page.ClosedAt == nil {
		t.Error("ClosedAt should be set")
	}

	// Active page should have switched back to initial page
	if pt.GetActivePageID() == newPageID {
		t.Error("active page should have changed after closing")
	}
}

func TestPageTracker_SnapshotAll(t *testing.T) {
	t.Parallel()

	pt := NewPageTracker("session-123", "https://example.com")

	// Add pages with explicit creation times to verify chronological ordering.
	initialPages, _ := pt.Snapshot(false)
	if len(initialPages) != 1 {
		t.Fatalf("expected one initial page, got %d", len(initialPages))
	}
	baseCreatedAt := initialPages[0].CreatedAt
	expectedIDs := []uuid.UUID{initialPages[0].ID}
	for i := 0; i < 3; i++ {
		pageID := uuid.New()
		expectedIDs = append(expectedIDs, pageID)
		pt.AddPage(&domain.Page{
			ID:        pageID,
			SessionID: "session-123",
			URL:       "https://example.com/page",
			CreatedAt: baseCreatedAt.Add(time.Duration(i+1) * time.Second),
			Status:    domain.PageStatusActive,
		})
	}

	pages, _ := pt.Snapshot(false)
	if len(pages) != 4 {
		t.Errorf("expected 4 pages, got %d", len(pages))
	}

	// Verify pages are sorted by creation time
	for i := 1; i < len(pages); i++ {
		if pages[i].CreatedAt.Before(pages[i-1].CreatedAt) {
			t.Error("pages should be sorted by creation time")
		}
	}
	for i, wantID := range expectedIDs {
		if pages[i].ID != wantID {
			t.Errorf("page %d ID = %s, want chronological page %s", i, pages[i].ID, wantID)
		}
	}
}

func TestPageTracker_SnapshotOpen(t *testing.T) {
	t.Parallel()

	pt := NewPageTracker("session-123", "https://example.com")

	// Add and close a page
	closedPageID := uuid.New()
	pt.AddPage(&domain.Page{
		ID:        closedPageID,
		SessionID: "session-123",
		URL:       "https://example.com/closed",
		CreatedAt: time.Now(),
		Status:    domain.PageStatusActive,
	})
	_, _ = pt.ClosePage(closedPageID)

	// Add an open page
	pt.AddPage(&domain.Page{
		ID:        uuid.New(),
		SessionID: "session-123",
		URL:       "https://example.com/open",
		CreatedAt: time.Now(),
		Status:    domain.PageStatusActive,
	})

	openPages, _ := pt.Snapshot(true)
	if len(openPages) != 2 { // initial + one open
		t.Errorf("expected 2 open pages, got %d", len(openPages))
	}

	for _, p := range openPages {
		if p.Status != domain.PageStatusActive {
			t.Errorf("ListOpenPages should only return active pages, got status %s", p.Status)
		}
	}
}

func TestPageTracker_DriverPageMapping(t *testing.T) {
	t.Parallel()

	pt := NewPageTracker("session-123", "https://example.com")

	// Map the initial page to a driver ID
	initialID := pt.GetInitialPageID()
	pt.SetInitialPageDriverID("driver-page-1")

	// Verify mapping works
	retrievedID := pt.GetPageIDByDriverID("driver-page-1")
	if retrievedID == nil || *retrievedID != initialID {
		t.Error("should retrieve initial page ID by driver page ID")
	}

	driverID := pt.GetDriverPageID(initialID)
	if driverID != "driver-page-1" {
		t.Errorf("expected driver ID 'driver-page-1', got '%s'", driverID)
	}

	// Verify page struct was updated
	initialPage := pt.GetActivePage()
	if initialPage.DriverPageID != "driver-page-1" {
		t.Error("page DriverPageID should be updated")
	}
}

func TestPageTracker_AddPageMapsDriverIdentity(t *testing.T) {
	t.Parallel()

	pt := NewPageTracker("session-123", "https://example.com")

	newPageID := uuid.New()
	pt.AddPage(&domain.Page{
		ID:           newPageID,
		DriverPageID: "driver-page-2",
		SessionID:    "session-123",
		URL:          "https://example.com/new",
		CreatedAt:    time.Now(),
		Status:       domain.PageStatusActive,
	})

	// Verify bidirectional mapping
	retrievedVrooliID := pt.GetPageIDByDriverID("driver-page-2")
	if retrievedVrooliID == nil || *retrievedVrooliID != newPageID {
		t.Error("GetPageIDByDriverID should return the mapped Vrooli page ID")
	}

	retrievedDriverID := pt.GetDriverPageID(newPageID)
	if retrievedDriverID != "driver-page-2" {
		t.Errorf("GetDriverPageID should return 'driver-page-2', got '%s'", retrievedDriverID)
	}
}

func TestPageTracker_UpdatePageInfo(t *testing.T) {
	t.Parallel()

	pt := NewPageTracker("session-123", "https://example.com")
	initialID := pt.GetInitialPageID()

	pt.UpdatePageInfo(initialID, "https://example.com/updated", "Updated Title", nil)

	page := pt.GetActivePage()
	if page.URL != "https://example.com/updated" {
		t.Errorf("expected URL 'https://example.com/updated', got '%s'", page.URL)
	}
	if page.Title != "Updated Title" {
		t.Errorf("expected title 'Updated Title', got '%s'", page.Title)
	}
}

func TestPageTracker_FindOpenerPageID(t *testing.T) {
	t.Parallel()

	pt := NewPageTracker("session-123", "https://example.com")
	pt.SetInitialPageDriverID("driver-page-1")

	// Empty opener should return nil
	openerID := pt.FindOpenerPageID("")
	if openerID != nil {
		t.Error("empty opener should return nil")
	}

	// Valid opener should return the mapped page ID
	openerID = pt.FindOpenerPageID("driver-page-1")
	if openerID == nil {
		t.Fatal("should find opener page ID")
	}
	if *openerID != pt.GetInitialPageID() {
		t.Error("opener ID should match initial page ID")
	}

	// Unknown opener should return nil
	openerID = pt.FindOpenerPageID("unknown-page")
	if openerID != nil {
		t.Error("unknown opener should return nil")
	}
}

func TestPageTracker_OpenPageCount(t *testing.T) {
	t.Parallel()

	pt := NewPageTracker("session-123", "https://example.com")

	if pt.OpenPageCount() != 1 {
		t.Errorf("expected 1 open page, got %d", pt.OpenPageCount())
	}

	// Add a page
	newPageID := uuid.New()
	pt.AddPage(&domain.Page{
		ID:        newPageID,
		SessionID: "session-123",
		URL:       "https://example.com/new",
		CreatedAt: time.Now(),
		Status:    domain.PageStatusActive,
	})

	if pt.OpenPageCount() != 2 {
		t.Errorf("expected 2 open pages, got %d", pt.OpenPageCount())
	}

	// Close the page
	_, _ = pt.ClosePage(newPageID)

	if pt.OpenPageCount() != 1 {
		t.Errorf("expected 1 open page after close, got %d", pt.OpenPageCount())
	}

	// Total should still be 2
	if pt.PageCount() != 2 {
		t.Errorf("expected 2 total pages, got %d", pt.PageCount())
	}
}

func TestPageTracker_Concurrent(t *testing.T) {
	t.Parallel()

	pt := NewPageTracker("session-123", "https://example.com")

	done := make(chan bool)

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = pt.GetActivePage()
				_, _ = pt.Snapshot(false)
				_ = pt.PageCount()
			}
			done <- true
		}()
	}

	// Concurrent writes
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 50; j++ {
				newID := uuid.New()
				pt.AddPage(&domain.Page{
					ID:           newID,
					DriverPageID: "driver-" + newID.String(),
					SessionID:    "session-123",
					URL:          "https://example.com/concurrent",
					CreatedAt:    time.Now(),
					Status:       domain.PageStatusActive,
				})
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 15; i++ {
		<-done
	}
}

// [REQ:BAS-RH-J03] Concurrent receipt/callback registration cannot duplicate or relabel tabs.
func TestPageTrackerConcurrentDriverRegistration(t *testing.T) {
	pt := NewPageTracker("session", "https://original.test")
	pt.SetInitialPageDriverID("initial-driver")
	initial := pt.GetInitialPageID()
	ids := make(chan uuid.UUID, 16)
	for i := 0; i < cap(ids); i++ {
		go func() {
			page := pt.AddPage(&domain.Page{DriverPageID: "second-driver", URL: "https://second.test"})
			ids <- page.ID
		}()
	}
	var second uuid.UUID
	for i := 0; i < cap(ids); i++ {
		id := <-ids
		if i == 0 {
			second = id
		}
		if id != second {
			t.Fatalf("duplicate page identities: %s and %s", second, id)
		}
	}
	if pt.PageCount() != 2 {
		t.Fatalf("expected exactly two registered pages, got %d", pt.PageCount())
	}
	if got := pt.GetPageIDByDriverID("initial-driver"); got == nil || *got != initial {
		t.Fatal("initial mapping was overwritten")
	}
	if pt.GetDriverPageID(second) != "second-driver" {
		t.Fatal("missing reverse mapping")
	}
	snapshot := pt.AddPage(&domain.Page{DriverPageID: "second-driver", URL: "https://stale-callback.test"})
	snapshot.URL = "https://caller-mutation.test"
	stored, _ := pt.GetPage(second)
	if stored.URL != "https://second.test" {
		t.Fatalf("registration/caller changed established page: %s", stored.URL)
	}
}

// [REQ:BAS-RH-J03] Page receipts remain stable and cannot mutate the registry.
func TestPageTrackerDetachedReceipts(t *testing.T) {
	for _, kind := range []string{"page", "active", "all", "open", "registration", "duplicate registration"} {
		t.Run(kind, func(t *testing.T) {
			pt := NewPageTracker("snapshot", "https://initial.test")
			opener := pt.GetInitialPageID()
			receipt := pt.AddPage(&domain.Page{DriverPageID: "target", URL: "https://before.test", Title: "Before", OpenerID: &opener})
			id := receipt.ID
			require.NoError(t, pt.SetActivePage(id))
			switch kind {
			case "page":
				receipt, _ = pt.GetPage(id)
			case "active":
				receipt = pt.GetActivePage()
			case "all", "open":
				pages, _ := pt.Snapshot(false)
				if kind == "open" {
					pages, _ = pt.Snapshot(true)
				}
				for _, p := range pages {
					if p.ID == id {
						receipt = p
					}
				}
			case "duplicate registration":
				receipt = pt.AddPage(&domain.Page{DriverPageID: "target"})
			}
			pt.UpdatePageInfo(id, "https://updated.test", "Updated", nil)
			assert.Equal(t, "https://before.test", receipt.URL)
			assert.Equal(t, "Before", receipt.Title)
			receipt.URL = "https://caller.test"
			receipt.DriverPageID = "wrong"
			*receipt.OpenerID = uuid.New()
			stored, ok := pt.GetPage(id)
			require.True(t, ok)
			assert.Equal(t, "https://updated.test", stored.URL)
			assert.Equal(t, "Updated", stored.Title)
			assert.Equal(t, "target", stored.DriverPageID)
			assert.Equal(t, pt.GetInitialPageID(), *stored.OpenerID)
			assert.Equal(t, "target", pt.GetDriverPageID(id))
		})
	}
}

// [REQ:BAS-RH-J03] Both registration directions and closed-page reads own nested values.
func TestPageTrackerNestedValuesDetached(t *testing.T) {
	for _, kind := range []string{"input", "registration", "duplicate registration", "page", "all"} {
		t.Run(kind, func(t *testing.T) {
			pt := NewPageTracker("snapshot", "https://initial.test")
			opener, closed := pt.GetInitialPageID(), time.Unix(100, 0)
			input := &domain.Page{DriverPageID: "closed", OpenerID: &opener, ClosedAt: &closed, Status: domain.PageStatusClosed}
			receipt := pt.AddPage(input)
			id := receipt.ID
			switch kind {
			case "input":
				receipt = input
			case "duplicate registration":
				receipt = pt.AddPage(&domain.Page{DriverPageID: "closed"})
			case "page":
				receipt, _ = pt.GetPage(id)
			case "all":
				pages, _ := pt.Snapshot(false)
				for _, p := range pages {
					if p.ID == id {
						receipt = p
					}
				}
			}
			*receipt.OpenerID = uuid.New()
			*receipt.ClosedAt = time.Unix(200, 0)
			stored, ok := pt.GetPage(id)
			require.True(t, ok)
			assert.Equal(t, pt.GetInitialPageID(), *stored.OpenerID)
			assert.Equal(t, time.Unix(100, 0), *stored.ClosedAt)
		})
	}
}

// [REQ:BAS-RH-J03] Serialization after the read returns is safe during callbacks.
func TestPageTrackerConcurrentSerialization(t *testing.T) {
	pt := NewPageTracker("snapshot", "https://initial.test")
	id := pt.GetInitialPageID()
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 500; i++ {
			pt.UpdatePageInfo(id, fmt.Sprintf("https://changed.test/%d", i), "changed", nil)
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 500; i++ {
			pages, active := pt.Snapshot(false)
			_, err := json.Marshal(domain.PagesResponse{Pages: pages, ActivePageID: active.String()})
			assert.NoError(t, err)
		}
	}()
	close(start)
	wg.Wait()
}

// [REQ:BAS-RH-J03] A closed final page cannot remain the selected open page.
func TestPageTrackerCloseFinalPageClearsSelection(t *testing.T) {
	pt := NewPageTracker("snapshot", "https://initial.test")
	_, err := pt.ClosePage(pt.GetInitialPageID())
	require.NoError(t, err)
	assert.Equal(t, uuid.Nil, pt.GetActivePageID())
	assert.Nil(t, pt.GetActivePage())
	pages, active := pt.Snapshot(true)
	assert.Empty(t, pages)
	assert.Equal(t, uuid.Nil, active)
	page, exists := pt.GetPage(pt.GetInitialPageID())
	require.True(t, exists)
	assert.Equal(t, domain.PageStatusClosed, page.Status)
	require.Error(t, pt.SetActivePage(page.ID))
	missing, exists := pt.GetPage(uuid.New())
	assert.False(t, exists)
	assert.Nil(t, missing)
}
