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

// AddPage adds a new page to the tracker.
func (pt *PageTracker) AddPage(page *domain.Page) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.pages[page.ID] = page
}

// GetPage returns a page by its Vrooli ID.
func (pt *PageTracker) GetPage(pageID uuid.UUID) (*domain.Page, bool) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	page, ok := pt.pages[pageID]
	return page, ok
}

// GetActivePageID returns the currently active page ID.
func (pt *PageTracker) GetActivePageID() uuid.UUID {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return pt.activePageID
}

// GetActivePage returns the currently active page.
func (pt *PageTracker) GetActivePage() *domain.Page {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return pt.pages[pt.activePageID]
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

// ClosePage marks a page as closed.
// If the closed page was active, switches to another open page.
func (pt *PageTracker) ClosePage(pageID uuid.UUID) error {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	page, ok := pt.pages[pageID]
	if !ok {
		return fmt.Errorf("page %s not found", pageID)
	}

	now := time.Now()
	page.ClosedAt = &now
	page.Status = domain.PageStatusClosed

	// If active page closed, switch to another open page
	if pt.activePageID == pageID {
		for id, p := range pt.pages {
			if p.Status == domain.PageStatusActive {
				pt.activePageID = id
				break
			}
		}
	}
	return nil
}

// ListPages returns all pages sorted by creation time.
func (pt *PageTracker) ListPages() []*domain.Page {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	pages := make([]*domain.Page, 0, len(pt.pages))
	for _, p := range pt.pages {
		pages = append(pages, p)
	}
	sort.Slice(pages, func(i, j int) bool {
		return pages[i].CreatedAt.Before(pages[j].CreatedAt)
	})
	return pages
}

// ListOpenPages returns all pages that are not closed.
func (pt *PageTracker) ListOpenPages() []*domain.Page {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	pages := make([]*domain.Page, 0)
	for _, p := range pt.pages {
		if p.Status == domain.PageStatusActive {
			pages = append(pages, p)
		}
	}
	sort.Slice(pages, func(i, j int) bool {
		return pages[i].CreatedAt.Before(pages[j].CreatedAt)
	})
	return pages
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

// MapDriverPageID creates a bidirectional mapping between driver page ID and Vrooli page ID.
func (pt *PageTracker) MapDriverPageID(driverPageID string, vrooliPageID uuid.UUID) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.driverToVrooli[driverPageID] = vrooliPageID
	pt.vrooliToDriver[vrooliPageID] = driverPageID

	// Also update the page's DriverPageID field
	if page, ok := pt.pages[vrooliPageID]; ok {
		page.DriverPageID = driverPageID
	}
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

// UpdatePageInfo updates the URL and title of a page.
func (pt *PageTracker) UpdatePageInfo(pageID uuid.UUID, url, title string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if page, ok := pt.pages[pageID]; ok {
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
