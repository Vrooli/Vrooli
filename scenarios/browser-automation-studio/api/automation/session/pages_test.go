package session

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/domain"
)

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

	// Add more pages with staggered creation times
	for i := 0; i < 3; i++ {
		time.Sleep(1 * time.Millisecond) // Ensure different timestamps
		pt.AddPage(&domain.Page{
			ID:        uuid.New(),
			SessionID: "session-123",
			URL:       "https://example.com/page",
			CreatedAt: time.Now(),
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
