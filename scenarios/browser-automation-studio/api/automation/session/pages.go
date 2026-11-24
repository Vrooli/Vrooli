package session

import (
	"fmt"
	"sort"
	"strings"
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

// NewPageTrackerFromEvents reconstructs the in-memory page index from an
// already ordered durable page-event stream. It does not consult or mutate a
// browser lease; callers remain responsible for proving that any live driver
// session is the same session before attaching this tracker to it.
func NewPageTrackerFromEvents(sessionID string, events []domain.PageEvent) (*PageTracker, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, fmt.Errorf("page replay requires session identity")
	}
	if len(events) == 0 {
		return nil, fmt.Errorf("page replay requires a complete event stream")
	}
	tracker := &PageTracker{
		pages:          make(map[uuid.UUID]*domain.Page),
		driverToVrooli: make(map[string]uuid.UUID),
		vrooliToDriver: make(map[uuid.UUID]string),
	}
	initialSeen := false
	for index := range events {
		event := events[index]
		if event.ID == uuid.Nil || event.PageID == uuid.Nil || event.Timestamp.IsZero() {
			return nil, fmt.Errorf("page replay event %d is missing identity or timestamp", index)
		}
		page := tracker.pages[event.PageID]
		switch event.Type {
		case domain.PageEventCreated:
			if page != nil {
				return nil, fmt.Errorf("page replay creates page %s more than once", event.PageID)
			}
			if event.IsInitial {
				if initialSeen {
					return nil, fmt.Errorf("page replay contains multiple initial pages")
				}
				initialSeen = true
				tracker.initialPageID = event.PageID
			}
			favicon := ""
			if event.FaviconURL != nil {
				favicon = *event.FaviconURL
			}
			page = &domain.Page{
				ID: event.PageID, SessionID: sessionID, URL: event.URL, Title: event.Title,
				FaviconURL: favicon, OpenerID: cloneUUID(event.OpenerID), CreatedAt: event.Timestamp,
				IsInitial: event.IsInitial, Status: domain.PageStatusActive, DriverPageID: event.DriverPageID,
			}
			tracker.pages[page.ID] = page
			if err := tracker.bindReplayDriverPage(page.ID, event.DriverPageID); err != nil {
				return nil, fmt.Errorf("page replay event %d: %w", index, err)
			}
			if event.IsInitial {
				tracker.activePageID = event.PageID
			}
		case domain.PageEventNavigated:
			if page == nil || page.Status != domain.PageStatusActive {
				return nil, fmt.Errorf("page replay navigates page %s before creation or after closure", event.PageID)
			}
			if event.DriverPageID != "" {
				if err := tracker.bindReplayDriverPage(page.ID, event.DriverPageID); err != nil {
					return nil, fmt.Errorf("page replay event %d: %w", index, err)
				}
			}
			page.URL, page.Title = event.URL, event.Title
			if event.FaviconURL != nil {
				page.FaviconURL = *event.FaviconURL
			} else if event.URL != "" {
				page.FaviconURL = ""
			}
		case domain.PageEventActivated:
			activeID := event.ActivePageID
			if activeID == uuid.Nil {
				activeID = event.PageID
			}
			active := tracker.pages[activeID]
			if active == nil || active.Status != domain.PageStatusActive {
				return nil, fmt.Errorf("page replay activates unknown or closed page %s", activeID)
			}
			if event.DriverPageID != "" {
				if err := tracker.bindReplayDriverPage(active.ID, event.DriverPageID); err != nil {
					return nil, fmt.Errorf("page replay event %d: %w", index, err)
				}
			}
			tracker.activePageID = activeID
		case domain.PageEventClosed:
			if page == nil {
				return nil, fmt.Errorf("page replay closes page %s before creation", event.PageID)
			}
			if page.Status == domain.PageStatusClosed {
				return nil, fmt.Errorf("page replay closes page %s more than once", event.PageID)
			}
			closedAt := event.Timestamp
			page.ClosedAt, page.Status = &closedAt, domain.PageStatusClosed
			activeID := event.ActivePageID
			if activeID == uuid.Nil {
				tracker.activePageID = uuid.Nil
			} else {
				active := tracker.pages[activeID]
				if active == nil || active.Status != domain.PageStatusActive || activeID == page.ID {
					return nil, fmt.Errorf("page replay close selects unknown or closed page %s", activeID)
				}
				tracker.activePageID = activeID
			}
		default:
			return nil, fmt.Errorf("page replay event %d has unsupported type %q", index, event.Type)
		}
	}
	if !initialSeen {
		return nil, fmt.Errorf("page replay event stream has no initial page receipt")
	}
	return tracker, nil
}

func (pt *PageTracker) bindReplayDriverPage(pageID uuid.UUID, driverPageID string) error {
	if driverPageID == "" {
		return nil
	}
	if existing, ok := pt.driverToVrooli[driverPageID]; ok && existing != pageID {
		return fmt.Errorf("driver page %q is assigned to multiple page identities", driverPageID)
	}
	if existing := pt.vrooliToDriver[pageID]; existing != "" && existing != driverPageID {
		return fmt.Errorf("page %s has conflicting driver identities", pageID)
	}
	pt.driverToVrooli[driverPageID] = pageID
	pt.vrooliToDriver[pageID] = driverPageID
	if page := pt.pages[pageID]; page != nil {
		page.DriverPageID = driverPageID
	}
	return nil
}

func cloneUUID(value *uuid.UUID) *uuid.UUID {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
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

// AddPageAt journals the exact browser page receipt before publishing its
// identity into the in-memory registry. Callback-first registration reuses the
// already committed driver lifecycle event and identity.
func (pt *PageTracker) AddPageAt(page *domain.Page, createdAt time.Time, persist func(*domain.PageEvent) error) (*domain.Page, error) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	if page == nil || page.DriverPageID == "" {
		return nil, fmt.Errorf("page receipt requires driver page identity")
	}
	if id, ok := pt.driverToVrooli[page.DriverPageID]; ok {
		if existing := pt.pages[id]; existing != nil {
			return clonePage(existing), nil
		}
		return nil, fmt.Errorf("driver page %q has stale registry identity", page.DriverPageID)
	}
	initial := pt.pages[pt.initialPageID]
	if initial == nil {
		return nil, fmt.Errorf("page tracking has no initial page")
	}
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	stored := clonePage(page)
	if stored.ID == uuid.Nil {
		stored.ID = stableDriverPageUUID(initial.SessionID, stored.DriverPageID)
	}
	if _, exists := pt.pages[stored.ID]; exists {
		return nil, fmt.Errorf("page identity %s is already registered", stored.ID)
	}
	stored.SessionID = initial.SessionID
	stored.CreatedAt = createdAt
	if stored.Status == "" {
		stored.Status = domain.PageStatusActive
	}
	event := &domain.PageEvent{
		ID: domain.StablePageEventID(stored.ID, domain.PageEventCreated, createdAt), Type: domain.PageEventCreated,
		PageID: stored.ID, DriverPageID: stored.DriverPageID, IsInitial: stored.IsInitial,
		URL: stored.URL, Title: stored.Title, FaviconURL: stringPointer(stored.FaviconURL), OpenerID: stored.OpenerID, Timestamp: createdAt,
	}
	if persist == nil {
		return nil, fmt.Errorf("page receipt requires durable journal owner")
	}
	if err := persist(event); err != nil {
		return nil, err
	}
	pt.pages[stored.ID] = stored
	pt.driverToVrooli[stored.DriverPageID] = stored.ID
	pt.vrooliToDriver[stored.ID] = stored.DriverPageID
	return clonePage(stored), nil
}

func stringPointer(value string) *string { return &value }

// ApplyDriverEvent validates and journals a driver lifecycle observation before
// publishing it to the in-memory page registry. The callback runs while the
// tracker lock is held so concurrent lifecycle events cannot overtake the
// durable journal commit. A failed append leaves tracker state unchanged.
func (pt *PageTracker) ApplyDriverEvent(ownerSessionID string, event domain.DriverPageEvent, occurredAt time.Time, persist func(*domain.PageEvent) error) (*domain.PageEvent, error) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	initial := pt.pages[pt.initialPageID]
	if initial == nil {
		return nil, fmt.Errorf("page tracking has no initial page")
	}
	if ownerSessionID == "" {
		return nil, fmt.Errorf("page lifecycle event requires owner session identity")
	}
	if initial.SessionID != "" && ownerSessionID != initial.SessionID {
		return nil, fmt.Errorf("route session %q does not match tracker owner %q", ownerSessionID, initial.SessionID)
	}
	if event.EventType == "initial" {
		id, exists := pt.driverToVrooli[event.DriverPageID]
		page := initial
		if exists {
			page = pt.pages[id]
			if page == nil {
				return nil, fmt.Errorf("driver page %q has stale registry identity", event.DriverPageID)
			}
		} else {
			if initial.DriverPageID != "" && initial.DriverPageID != event.DriverPageID {
				return nil, fmt.Errorf("initial driver page %q conflicts with registered page %q", event.DriverPageID, initial.DriverPageID)
			}
			id = pt.initialPageID
		}
		if page == nil || page.Status != domain.PageStatusActive {
			return nil, fmt.Errorf("driver page %q is not open", event.DriverPageID)
		}
		if page.DriverPageID != "" && page.DriverPageID != event.DriverPageID {
			return nil, fmt.Errorf("driver page %q conflicts with registered page %q", event.DriverPageID, page.DriverPageID)
		}
		if event.VrooliPageID != "" && event.VrooliPageID != id.String() {
			return nil, fmt.Errorf("driver page identity %q does not match registered page %s", event.VrooliPageID, id)
		}
		nextURL, nextTitle, nextFavicon := page.URL, page.Title, page.FaviconURL
		if event.URL != "" {
			if event.URL != page.URL {
				nextFavicon = ""
			}
			nextURL = event.URL
		}
		if event.Title != "" {
			nextTitle = event.Title
		}
		if event.FaviconURL != nil {
			nextFavicon = *event.FaviconURL
		}
		favicon := nextFavicon
		observation := &domain.PageEvent{
			ID: domain.StablePageEventID(id, domain.PageEventCreated, occurredAt), Type: domain.PageEventCreated,
			PageID: id, DriverPageID: event.DriverPageID, IsInitial: page.IsInitial,
			URL: nextURL, Title: nextTitle, FaviconURL: &favicon, OpenerID: page.OpenerID, ActivePageID: id, Timestamp: occurredAt,
		}
		if persist == nil {
			return nil, fmt.Errorf("initial page observation requires durable journal owner")
		}
		if err := persist(observation); err != nil {
			return nil, err
		}
		if !exists {
			pt.driverToVrooli[event.DriverPageID] = id
			pt.vrooliToDriver[id] = event.DriverPageID
			page.DriverPageID = event.DriverPageID
		}
		page.URL, page.Title, page.FaviconURL = nextURL, nextTitle, nextFavicon
		pt.activePageID = id
		return observation, nil
	}

	var pageID uuid.UUID
	var page *domain.Page
	var observation *domain.PageEvent
	var closedActivePageID uuid.UUID
	switch event.EventType {
	case "created":
		if id, ok := pt.driverToVrooli[event.DriverPageID]; ok {
			pageID, page = id, pt.pages[id]
		} else {
			pageID = stableDriverPageUUID(ownerSessionID, event.DriverPageID)
			openerID := pt.driverToVrooli[event.OpenerDriverPageID]
			var opener *uuid.UUID
			if openerID != uuid.Nil {
				opener = &openerID
			}
			favicon := ""
			if event.FaviconURL != nil {
				favicon = *event.FaviconURL
			}
			page = &domain.Page{ID: pageID, SessionID: ownerSessionID, URL: event.URL, Title: event.Title, FaviconURL: favicon, OpenerID: opener, DriverPageID: event.DriverPageID, CreatedAt: occurredAt, Status: domain.PageStatusActive}
		}
		if page == nil || page.Status != domain.PageStatusActive {
			return nil, fmt.Errorf("driver page %q is not open", event.DriverPageID)
		}
		if event.VrooliPageID != "" && event.VrooliPageID != pageID.String() {
			return nil, fmt.Errorf("driver page identity %q does not match registered page %s", event.VrooliPageID, pageID)
		}
		var opener *uuid.UUID
		if openerID := pt.driverToVrooli[event.OpenerDriverPageID]; openerID != uuid.Nil {
			opener = &openerID
		} else if page.OpenerID != nil {
			opener = page.OpenerID
		}
		createdAt := occurredAt
		if page.CreatedAt.After(time.Time{}) {
			createdAt = page.CreatedAt
		}
		observation = &domain.PageEvent{ID: domain.StablePageEventID(pageID, domain.PageEventCreated, createdAt), Type: domain.PageEventCreated, PageID: pageID, DriverPageID: event.DriverPageID, IsInitial: page.IsInitial, URL: event.URL, Title: event.Title, FaviconURL: event.FaviconURL, OpenerID: opener, Timestamp: createdAt}
	case "navigated":
		id, ok := pt.driverToVrooli[event.DriverPageID]
		if !ok {
			return nil, fmt.Errorf("driver page %q is not registered", event.DriverPageID)
		}
		pageID, page = id, pt.pages[id]
		if page == nil || page.Status != domain.PageStatusActive {
			return nil, fmt.Errorf("driver page %q is closed", event.DriverPageID)
		}
		if event.VrooliPageID != "" && event.VrooliPageID != pageID.String() {
			return nil, fmt.Errorf("driver page identity %q does not match registered page %s", event.VrooliPageID, pageID)
		}
		observation = &domain.PageEvent{ID: domain.StablePageEventID(pageID, domain.PageEventNavigated, occurredAt), Type: domain.PageEventNavigated, PageID: pageID, DriverPageID: event.DriverPageID, IsInitial: page.IsInitial, URL: event.URL, Title: event.Title, FaviconURL: event.FaviconURL, Timestamp: occurredAt}
	case "closed":
		id, ok := pt.driverToVrooli[event.DriverPageID]
		if !ok {
			return nil, fmt.Errorf("driver page %q is not registered", event.DriverPageID)
		}
		pageID, page = id, pt.pages[id]
		if page == nil {
			return nil, fmt.Errorf("driver page %q is not registered", event.DriverPageID)
		}
		if event.VrooliPageID != "" && event.VrooliPageID != pageID.String() {
			return nil, fmt.Errorf("driver page identity %q does not match registered page %s", event.VrooliPageID, pageID)
		}
		closedAt := occurredAt
		if page.ClosedAt != nil {
			closedAt = *page.ClosedAt
		}
		closedActivePageID = pt.activePageID
		if closedActivePageID == pageID {
			closedActivePageID = uuid.Nil
			candidates := make([]*domain.Page, 0, len(pt.pages))
			for candidateID, candidate := range pt.pages {
				if candidateID != pageID && candidate.Status == domain.PageStatusActive {
					candidates = append(candidates, candidate)
				}
			}
			sort.Slice(candidates, func(i, j int) bool { return candidates[i].CreatedAt.Before(candidates[j].CreatedAt) })
			if len(candidates) != 0 {
				closedActivePageID = candidates[0].ID
			}
		}
		observation = &domain.PageEvent{ID: domain.StablePageEventID(pageID, domain.PageEventClosed, closedAt), Type: domain.PageEventClosed, PageID: pageID, DriverPageID: event.DriverPageID, IsInitial: page.IsInitial, ActivePageID: closedActivePageID, Timestamp: closedAt}
	default:
		return nil, fmt.Errorf("unsupported driver page event type %q", event.EventType)
	}

	if persist == nil {
		return nil, fmt.Errorf("page lifecycle event requires durable journal owner")
	}
	if err := persist(observation); err != nil {
		return nil, err
	}

	switch event.EventType {
	case "created":
		if pt.pages[pageID] == nil {
			pt.pages[pageID] = clonePage(page)
			pt.driverToVrooli[event.DriverPageID] = pageID
			pt.vrooliToDriver[pageID] = event.DriverPageID
		} else {
			if page.OpenerID == nil && observation.OpenerID != nil {
				opener := *observation.OpenerID
				page.OpenerID = &opener
			}
			if event.URL != "" && event.URL != page.URL {
				page.FaviconURL = ""
			}
			if event.FaviconURL != nil {
				page.FaviconURL = *event.FaviconURL
			}
			if event.URL != "" {
				page.URL = event.URL
			}
			if event.Title != "" {
				page.Title = event.Title
			}
		}
	case "navigated":
		if event.URL != "" && event.URL != page.URL {
			page.FaviconURL = ""
		}
		if event.FaviconURL != nil {
			page.FaviconURL = *event.FaviconURL
		}
		if event.URL != "" {
			page.URL = event.URL
		}
		if event.Title != "" {
			page.Title = event.Title
		}
	case "closed":
		if page.ClosedAt == nil {
			closedAt := occurredAt
			page.ClosedAt = &closedAt
		}
		page.Status = domain.PageStatusClosed
		if pt.activePageID == pageID {
			pt.activePageID = closedActivePageID
		}
	}
	return observation, nil
}

func stableDriverPageUUID(sessionID, driverPageID string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte("bas-page:"+sessionID+"\x00"+driverPageID))
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

// SetActivePageAt commits a durable selection observation before changing the
// active-page pointer. Production callers provide the existing timeline owner.
func (pt *PageTracker) SetActivePageAt(pageID uuid.UUID, selectedAt time.Time, persist func(*domain.PageEvent) error) error {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	page, ok := pt.pages[pageID]
	if !ok {
		return fmt.Errorf("page %s not found in session", pageID)
	}
	if page.Status != domain.PageStatusActive {
		return fmt.Errorf("page %s is closed", pageID)
	}
	if pt.activePageID == pageID {
		return nil
	}
	if selectedAt.IsZero() {
		selectedAt = time.Now()
	}
	event := &domain.PageEvent{
		ID: domain.StablePageEventID(pageID, domain.PageEventActivated, selectedAt), Type: domain.PageEventActivated,
		PageID: pageID, DriverPageID: page.DriverPageID, IsInitial: page.IsInitial,
		ActivePageID: pageID, Timestamp: selectedAt,
	}
	if persist == nil {
		return fmt.Errorf("active page selection requires durable journal owner")
	}
	if err := persist(event); err != nil {
		return err
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
	return pt.ClosePageAtPersist(pageID, closedAt, nil)
}

// ClosePageAtPersist records both closure and the resulting active selection
// before mutating either field in the tracker.
func (pt *PageTracker) ClosePageAtPersist(pageID uuid.UUID, closedAt time.Time, persist func(*domain.PageEvent) error) (*domain.PageEvent, error) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	page, ok := pt.pages[pageID]
	if !ok {
		return nil, fmt.Errorf("page %s not found", pageID)
	}

	if page.ClosedAt != nil {
		closedAt = *page.ClosedAt
	} else if closedAt.IsZero() {
		closedAt = time.Now()
	}
	activePageID := pt.activePageID
	if pt.activePageID == pageID {
		activePageID = uuid.Nil
		candidates := make([]*domain.Page, 0, len(pt.pages))
		for id, candidate := range pt.pages {
			if id != pageID && candidate.Status == domain.PageStatusActive {
				candidates = append(candidates, candidate)
			}
		}
		sort.Slice(candidates, func(i, j int) bool { return candidates[i].CreatedAt.Before(candidates[j].CreatedAt) })
		if len(candidates) != 0 {
			activePageID = candidates[0].ID
		}
	}
	event := &domain.PageEvent{
		ID: domain.StablePageEventID(pageID, domain.PageEventClosed, closedAt), Type: domain.PageEventClosed,
		PageID: pageID, DriverPageID: page.DriverPageID, IsInitial: page.IsInitial, ActivePageID: activePageID, Timestamp: closedAt,
	}
	if persist != nil {
		if err := persist(event); err != nil {
			return nil, err
		}
	}
	if page.ClosedAt == nil {
		page.ClosedAt = &closedAt
	}
	page.Status = domain.PageStatusClosed
	pt.activePageID = activePageID
	return event, nil
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

// UpdatePageInfoAt persists metadata observations before changing the page
// snapshot. It is used when a browser receipt enriches an identity registered
// by an earlier callback.
func (pt *PageTracker) UpdatePageInfoAt(pageID uuid.UUID, url, title string, faviconURL *string, observedAt time.Time, persist func(*domain.PageEvent) error) error {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	page, ok := pt.pages[pageID]
	if !ok {
		return fmt.Errorf("page %s not found", pageID)
	}
	nextURL, nextTitle, nextFavicon := page.URL, page.Title, page.FaviconURL
	if url != "" && url != page.URL {
		nextFavicon = ""
	}
	if faviconURL != nil {
		nextFavicon = *faviconURL
	}
	if url != "" {
		nextURL = url
	}
	if title != "" {
		nextTitle = title
	}
	if nextURL == page.URL && nextTitle == page.Title && nextFavicon == page.FaviconURL {
		return nil
	}
	if observedAt.IsZero() {
		observedAt = time.Now()
	}
	favicon := nextFavicon
	event := &domain.PageEvent{
		ID: domain.StablePageEventID(pageID, domain.PageEventNavigated, observedAt), Type: domain.PageEventNavigated,
		PageID: pageID, DriverPageID: page.DriverPageID, IsInitial: page.IsInitial,
		URL: nextURL, Title: nextTitle, FaviconURL: &favicon, ActivePageID: pt.activePageID, Timestamp: observedAt,
	}
	if persist == nil {
		return fmt.Errorf("page metadata update requires durable journal owner")
	}
	if err := persist(event); err != nil {
		return err
	}
	page.URL, page.Title, page.FaviconURL = nextURL, nextTitle, nextFavicon
	return nil
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
