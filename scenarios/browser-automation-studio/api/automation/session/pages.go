package session

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/domain"
)

// PageTracker manages multiple pages within a recording session.
// It maintains the mapping between Vrooli page IDs (UUIDs) and
// Playwright driver page IDs (strings).
type PageTracker struct {
	pages         map[uuid.UUID]*domain.Page
	activePageID  uuid.UUID
	initialPageID uuid.UUID

	// Bidirectional mapping between driver page IDs and Vrooli page IDs
	driverToVrooli map[string]uuid.UUID
	vrooliToDriver map[uuid.UUID]string

	mu sync.RWMutex
}

// NewPageTracker creates a new page tracker with an initial page.
// The initial page is created automatically and set as the active page.
func NewPageTracker(sessionID, initialURL string) *PageTracker {
	initialPageID := uuid.New()
	initialPage := &domain.Page{
		ID:        initialPageID,
		SessionID: sessionID,
		URL:       initialURL,
		CreatedAt: time.Now(),
		IsInitial: true,
		Status:    domain.PageStatusActive,
	}

	return &PageTracker{
		pages: map[uuid.UUID]*domain.Page{
			initialPageID: initialPage,
		},
		activePageID:   initialPageID,
		initialPageID:  initialPageID,
		driverToVrooli: make(map[string]uuid.UUID),
		vrooliToDriver: make(map[uuid.UUID]string),
	}
}

// AddPage atomically registers a page and its driver mapping. Creation receipts
// and asynchronous callbacks for the same browser page share one identity.
// Returned snapshots do not expose the registry's mutable storage.
func (pt *PageTracker) AddPage(page *domain.Page) *domain.Page {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	if id, ok := pt.driverToVrooli[page.DriverPageID]; ok {
		return clonePage(pt.pages[id])
	}
	stored := clonePage(page)
	if stored.ID == uuid.Nil {
		stored.ID = uuid.New()
	}
	stored.SessionID = pt.pages[pt.initialPageID].SessionID
	if stored.CreatedAt.IsZero() {
		stored.CreatedAt = time.Now()
	}
	if stored.Status == "" {
		stored.Status = domain.PageStatusActive
	}
	pt.pages[stored.ID] = stored
	if stored.DriverPageID != "" {
		pt.driverToVrooli[stored.DriverPageID] = stored.ID
		pt.vrooliToDriver[stored.ID] = stored.DriverPageID
	}
	return clonePage(stored)
}

// clonePage detaches every mutable field; callers must hold the lock when
// copying a record owned by the tracker.
func clonePage(page *domain.Page) *domain.Page {
	if page == nil {
		return nil
	}
	copy := *page
	if page.OpenerID != nil {
		opener := *page.OpenerID
		copy.OpenerID = &opener
	}
	if page.ClosedAt != nil {
		closed := *page.ClosedAt
		copy.ClosedAt = &closed
	}
	return &copy
}

// GetPage returns a detached page by its Vrooli ID.
func (pt *PageTracker) GetPage(pageID uuid.UUID) (*domain.Page, bool) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	page, ok := pt.pages[pageID]
	return clonePage(page), ok
}

// GetActivePageID returns the currently active page ID.
func (pt *PageTracker) GetActivePageID() uuid.UUID {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return pt.activePageID
}

// GetActivePage returns a detached active page, or nil if none is open.
func (pt *PageTracker) GetActivePage() *domain.Page {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return clonePage(pt.pages[pt.activePageID])
}

// GetInitialPageID returns the initial page ID.
func (pt *PageTracker) GetInitialPageID() uuid.UUID {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return pt.initialPageID
}

// SetActivePage changes the active page.
// Returns an error if the page doesn't exist or is closed.
func (pt *PageTracker) SetActivePage(pageID uuid.UUID) error {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	page, ok := pt.pages[pageID]
	if !ok {
		return fmt.Errorf("page %s not found in session", pageID)
	}
	if page.Status != domain.PageStatusActive {
		return fmt.Errorf("page %s is closed", pageID)
	}
	pt.activePageID = pageID
	return nil
}

// ClosePage records closure and returns its stable observation receipt.
// Repeated callbacks/commands preserve the original timestamp and journal ID.
func (pt *PageTracker) ClosePage(pageID uuid.UUID) (*domain.PageEvent, error) {
	return pt.ClosePageAt(pageID, time.Now())
}

// ClosePageAt records the browser-observed close time when the driver reports
// lifecycle events; repeated receipts retain the first close time.
func (pt *PageTracker) ClosePageAt(pageID uuid.UUID, closedAt time.Time) (*domain.PageEvent, error) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	page, ok := pt.pages[pageID]
	if !ok {
		return nil, fmt.Errorf("page %s not found", pageID)
	}

	if page.ClosedAt == nil {
		if closedAt.IsZero() {
			closedAt = time.Now()
		}
		page.ClosedAt = &closedAt
	}
	page.Status = domain.PageStatusClosed

	// If active page closed, switch to another open page
	if pt.activePageID == pageID {
		pt.activePageID = uuid.Nil
		for id, p := range pt.pages {
			if p.Status == domain.PageStatusActive {
				pt.activePageID = id
				break
			}
		}
	}
	return &domain.PageEvent{
		ID: uuid.NewSHA1(pageID, []byte("closed")), Type: domain.PageEventClosed,
		PageID: pageID, Timestamp: *page.ClosedAt,
	}, nil
}

// Snapshot captures detached pages and their selected ID under one lock.
// Closed pages are included unless openOnly is true. Sorting private copies
// after unlocking keeps serialization and sorting outside the registry lock.
func (pt *PageTracker) Snapshot(openOnly bool) ([]*domain.Page, uuid.UUID) {
	pt.mu.RLock()
	pages := make([]*domain.Page, 0, len(pt.pages))
	for _, p := range pt.pages {
		if !openOnly || p.Status == domain.PageStatusActive {
			pages = append(pages, clonePage(p))
		}
	}
	active := pt.activePageID
	pt.mu.RUnlock()
	sort.Slice(pages, func(i, j int) bool {
		return pages[i].CreatedAt.Before(pages[j].CreatedAt)
	})
	return pages, active
}

// PageCount returns the total number of pages (including closed).
func (pt *PageTracker) PageCount() int {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return len(pt.pages)
}

// OpenPageCount returns the number of open (not closed) pages.
func (pt *PageTracker) OpenPageCount() int {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	count := 0
	for _, p := range pt.pages {
		if p.Status == domain.PageStatusActive {
			count++
		}
	}
	return count
}

// GetPageIDByDriverID returns the Vrooli page ID for a driver page ID.
func (pt *PageTracker) GetPageIDByDriverID(driverPageID string) *uuid.UUID {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	if id, ok := pt.driverToVrooli[driverPageID]; ok {
		return &id
	}
	return nil
}

// GetDriverPageID returns the driver page ID for a Vrooli page ID.
func (pt *PageTracker) GetDriverPageID(vrooliPageID uuid.UUID) string {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return pt.vrooliToDriver[vrooliPageID]
}

// UpdatePageInfo updates browser metadata. Missing icons survive same-URL title
// observations; a different document or explicit empty icon clears stale metadata.
func (pt *PageTracker) UpdatePageInfo(pageID uuid.UUID, url, title string, faviconURL *string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if page, ok := pt.pages[pageID]; ok {
		if url != "" && url != page.URL {
			page.FaviconURL = ""
		}
		if faviconURL != nil {
			page.FaviconURL = *faviconURL
		}
		if url != "" {
			page.URL = url
		}
		if title != "" {
			page.Title = title
		}
	}
}

// SetInitialPageDriverID sets the driver page ID for the initial page.
// This should be called when the driver reports the initial page.
func (pt *PageTracker) SetInitialPageDriverID(driverPageID string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.driverToVrooli[driverPageID] = pt.initialPageID
	pt.vrooliToDriver[pt.initialPageID] = driverPageID

	if page, ok := pt.pages[pt.initialPageID]; ok {
		page.DriverPageID = driverPageID
	}
}

// FindOpenerPageID finds the Vrooli page ID for an opener driver page ID.
func (pt *PageTracker) FindOpenerPageID(openerDriverPageID string) *uuid.UUID {
	if openerDriverPageID == "" {
		return nil
	}
	return pt.GetPageIDByDriverID(openerDriverPageID)
}
