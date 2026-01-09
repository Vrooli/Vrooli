package session

import (
	"testing"
	"time"

	"github.com/google/uuid"
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
	_ = pt.ClosePage(closedPageID)

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
	err := pt.ClosePage(newPageID)
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

func TestPageTracker_ListPages(t *testing.T) {
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

	pages := pt.ListPages()
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

func TestPageTracker_ListOpenPages(t *testing.T) {
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
	_ = pt.ClosePage(closedPageID)

	// Add an open page
	pt.AddPage(&domain.Page{
		ID:        uuid.New(),
		SessionID: "session-123",
		URL:       "https://example.com/open",
		CreatedAt: time.Now(),
		Status:    domain.PageStatusActive,
	})

	openPages := pt.ListOpenPages()
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

func TestPageTracker_MapDriverPageID(t *testing.T) {
	t.Parallel()

	pt := NewPageTracker("session-123", "https://example.com")

	newPageID := uuid.New()
	pt.AddPage(&domain.Page{
		ID:        newPageID,
		SessionID: "session-123",
		URL:       "https://example.com/new",
		CreatedAt: time.Now(),
		Status:    domain.PageStatusActive,
	})

	pt.MapDriverPageID("driver-page-2", newPageID)

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

	pt.UpdatePageInfo(initialID, "https://example.com/updated", "Updated Title")

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
	_ = pt.ClosePage(newPageID)

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
				_ = pt.ListPages()
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
					ID:        newID,
					SessionID: "session-123",
					URL:       "https://example.com/concurrent",
					CreatedAt: time.Now(),
					Status:    domain.PageStatusActive,
				})
				pt.MapDriverPageID("driver-"+newID.String(), newID)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 15; i++ {
		<-done
	}
}
