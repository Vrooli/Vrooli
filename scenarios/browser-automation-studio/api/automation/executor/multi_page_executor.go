// Package executor provides workflow execution functionality.
package executor

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/engine"
)

// PageMatchingStrategy defines how to match runtime pages to recorded pages.
type PageMatchingStrategy string

const (
	// MatchByOpener matches pages by their opener relationship.
	MatchByOpener PageMatchingStrategy = "opener"
	// MatchByURL matches pages by URL pattern.
	MatchByURL PageMatchingStrategy = "url"
	// MatchByOrder matches pages by creation order.
	MatchByOrder PageMatchingStrategy = "order"
)

// MultiPageConfig configures multi-page execution behavior.
type MultiPageConfig struct {
	// PageWaitTimeout is how long to wait for an expected page to open.
	PageWaitTimeout time.Duration
	// MatchStrategy is the primary strategy for matching runtime pages to recorded pages.
	MatchStrategy PageMatchingStrategy
	// EnableFallback enables fallback matching when primary strategy fails.
	EnableFallback bool
}

// DefaultMultiPageConfig returns sensible defaults for multi-page execution.
func DefaultMultiPageConfig() MultiPageConfig {
	return MultiPageConfig{
		PageWaitTimeout: 30 * time.Second,
		MatchStrategy:   MatchByOpener,
		EnableFallback:  true,
	}
}

// RuntimePage represents a browser page detected at runtime during execution.
type RuntimePage struct {
	ID        string    // Driver-assigned page ID
	URL       string    // Current URL
	Title     string    // Page title
	OpenerID  string    // ID of the page that opened this one (empty for initial)
	CreatedAt time.Time // When the page was detected
	Closed    bool      // True if the page has been closed
}

// PageMatcher manages page tracking and matching during multi-page execution.
type PageMatcher struct {
	mu sync.RWMutex

	// plan contains page definitions from the recorded workflow
	plan *contracts.ExecutionPlan

	// runtimePages maps driver page IDs to runtime page info
	runtimePages map[string]*RuntimePage

	// recordedToRuntime maps recorded page UUIDs to driver page IDs
	recordedToRuntime map[uuid.UUID]string

	// runtimeToRecorded maps driver page IDs to recorded page UUIDs
	runtimeToRecorded map[string]uuid.UUID

	// pendingPages are recorded pages not yet matched to runtime pages
	pendingPages map[uuid.UUID]*contracts.PageDefinition

	// config for matching behavior
	config MultiPageConfig

	log *logrus.Logger
}

// NewPageMatcher creates a page matcher from an execution plan.
func NewPageMatcher(plan *contracts.ExecutionPlan, config MultiPageConfig, log *logrus.Logger) *PageMatcher {
	if log == nil {
		log = logrus.StandardLogger()
	}

	pm := &PageMatcher{
		plan:              plan,
		runtimePages:      make(map[string]*RuntimePage),
		recordedToRuntime: make(map[uuid.UUID]string),
		runtimeToRecorded: make(map[string]uuid.UUID),
		pendingPages:      make(map[uuid.UUID]*contracts.PageDefinition),
		config:            config,
		log:               log,
	}

	// Initialize pending pages from plan (excluding initial page)
	for i := range plan.Pages {
		page := &plan.Pages[i]
		if !page.IsInitial {
			pm.pendingPages[page.ID] = page
		}
	}

	return pm
}

// RegisterInitialPage registers the initial page with a runtime ID.
func (pm *PageMatcher) RegisterInitialPage(driverPageID, url string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	initialPage := contracts.GetInitialPage(pm.plan)
	if initialPage == nil {
		return errors.New("no initial page defined in plan")
	}

	pm.runtimePages[driverPageID] = &RuntimePage{
		ID:        driverPageID,
		URL:       url,
		CreatedAt: time.Now(),
	}

	pm.recordedToRuntime[initialPage.ID] = driverPageID
	pm.runtimeToRecorded[driverPageID] = initialPage.ID

	pm.log.WithFields(logrus.Fields{
		"recorded_page_id": initialPage.ID,
		"driver_page_id":   driverPageID,
		"url":              url,
	}).Debug("Registered initial page")

	return nil
}

// OnPageCreated handles a new page being created at runtime.
// Returns the matched recorded page ID if a match is found.
func (pm *PageMatcher) OnPageCreated(driverPageID, url, title, openerID string) *uuid.UUID {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.runtimePages[driverPageID] = &RuntimePage{
		ID:        driverPageID,
		URL:       url,
		Title:     title,
		OpenerID:  openerID,
		CreatedAt: time.Now(),
	}

	// Try to match to a pending recorded page
	matchedPageID := pm.matchPage(driverPageID, url, openerID)
	if matchedPageID != nil {
		pm.recordedToRuntime[*matchedPageID] = driverPageID
		pm.runtimeToRecorded[driverPageID] = *matchedPageID
		delete(pm.pendingPages, *matchedPageID)

		pm.log.WithFields(logrus.Fields{
			"recorded_page_id": matchedPageID,
			"driver_page_id":   driverPageID,
			"url":              url,
		}).Debug("Matched runtime page to recorded page")
	} else {
		pm.log.WithFields(logrus.Fields{
			"driver_page_id":     driverPageID,
			"url":                url,
			"pending_page_count": len(pm.pendingPages),
		}).Debug("Runtime page created but no match found")
	}

	return matchedPageID
}

// OnPageClosed handles a page being closed at runtime.
func (pm *PageMatcher) OnPageClosed(driverPageID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if page, ok := pm.runtimePages[driverPageID]; ok {
		page.Closed = true
	}
}

// GetDriverPageID returns the runtime driver page ID for a recorded page.
func (pm *PageMatcher) GetDriverPageID(recordedPageID uuid.UUID) (string, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	driverID, ok := pm.recordedToRuntime[recordedPageID]
	return driverID, ok
}

// WaitForPage waits for a recorded page to appear at runtime.
func (pm *PageMatcher) WaitForPage(ctx context.Context, recordedPageID uuid.UUID) (string, error) {
	// Check if already matched
	if driverID, ok := pm.GetDriverPageID(recordedPageID); ok {
		return driverID, nil
	}

	// Wait with timeout
	timeout := pm.config.PageWaitTimeout
	deadline := time.Now().Add(timeout)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			if driverID, ok := pm.GetDriverPageID(recordedPageID); ok {
				return driverID, nil
			}
			if time.Now().After(deadline) {
				return "", fmt.Errorf("timeout waiting for page %s after %v", recordedPageID, timeout)
			}
		}
	}
}

// matchPage attempts to match a runtime page to a pending recorded page.
// This is the core matching logic that implements the page matching strategy.
func (pm *PageMatcher) matchPage(driverPageID, url, openerID string) *uuid.UUID {
	if len(pm.pendingPages) == 0 {
		return nil
	}

	// Strategy 1: Match by opener
	if pm.config.MatchStrategy == MatchByOpener || pm.config.EnableFallback {
		if openerID != "" {
			// Find which recorded page the opener corresponds to
			if openerRecordedID, ok := pm.runtimeToRecorded[openerID]; ok {
				// Look for a pending page opened by this recorded page
				for _, pending := range pm.pendingPages {
					if pending.OpenerID != nil && *pending.OpenerID == openerRecordedID {
						return &pending.ID
					}
				}
			}
		}
	}

	// Strategy 2: Match by URL pattern
	if pm.config.MatchStrategy == MatchByURL || pm.config.EnableFallback {
		for _, pending := range pm.pendingPages {
			if pending.URLPattern != "" {
				matched, _ := regexp.MatchString(pending.URLPattern, url)
				if matched {
					return &pending.ID
				}
			}
		}
	}

	// Strategy 3: Match by order (fallback when only one pending page)
	if pm.config.EnableFallback && len(pm.pendingPages) == 1 {
		for _, pending := range pm.pendingPages {
			return &pending.ID
		}
	}

	return nil
}

// GetAllRuntimePages returns all tracked runtime pages.
func (pm *PageMatcher) GetAllRuntimePages() []*RuntimePage {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	pages := make([]*RuntimePage, 0, len(pm.runtimePages))
	for _, page := range pm.runtimePages {
		pageCopy := *page
		pages = append(pages, &pageCopy)
	}
	return pages
}

// HasPendingPages returns true if there are recorded pages not yet matched.
func (pm *PageMatcher) HasPendingPages() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return len(pm.pendingPages) > 0
}

// MultiPageSession wraps an engine session with multi-page awareness.
type MultiPageSession struct {
	session     engine.EngineSession
	pageMatcher *PageMatcher
	log         *logrus.Logger

	// currentPageID tracks the current active page (driver ID)
	currentPageID string
	mu            sync.RWMutex
}

// NewMultiPageSession creates a multi-page aware session wrapper.
func NewMultiPageSession(session engine.EngineSession, pageMatcher *PageMatcher, log *logrus.Logger) *MultiPageSession {
	if log == nil {
		log = logrus.StandardLogger()
	}
	return &MultiPageSession{
		session:     session,
		pageMatcher: pageMatcher,
		log:         log,
	}
}

// SetInitialPage sets the initial page as the current active page.
func (mps *MultiPageSession) SetInitialPage(driverPageID string) {
	mps.mu.Lock()
	defer mps.mu.Unlock()
	mps.currentPageID = driverPageID
}

// SwitchToPage switches the active page to the one matching the recorded page ID.
func (mps *MultiPageSession) SwitchToPage(ctx context.Context, recordedPageID uuid.UUID) error {
	// Get the runtime driver page ID
	driverPageID, ok := mps.pageMatcher.GetDriverPageID(recordedPageID)
	if !ok {
		// Wait for the page to appear
		var err error
		driverPageID, err = mps.pageMatcher.WaitForPage(ctx, recordedPageID)
		if err != nil {
			return fmt.Errorf("page %s not found: %w", recordedPageID, err)
		}
	}

	mps.mu.Lock()
	defer mps.mu.Unlock()

	// Already on this page?
	if mps.currentPageID == driverPageID {
		return nil
	}

	// Switch via driver - get the session with page switching capability
	if switcher, ok := mps.session.(PageSwitcher); ok {
		if err := switcher.SetActivePage(ctx, driverPageID); err != nil {
			return fmt.Errorf("switch to page %s: %w", recordedPageID, err)
		}
	} else {
		return fmt.Errorf("session does not support page switching")
	}

	mps.currentPageID = driverPageID
	mps.log.WithFields(logrus.Fields{
		"recorded_page_id": recordedPageID,
		"driver_page_id":   driverPageID,
	}).Debug("Switched active page")

	return nil
}

// Run executes an instruction on the appropriate page.
func (mps *MultiPageSession) Run(ctx context.Context, instr contracts.CompiledInstruction) (contracts.StepOutcome, error) {
	// If instruction has a page ID, switch to that page first
	if instr.PageID != nil {
		if err := mps.SwitchToPage(ctx, *instr.PageID); err != nil {
			return contracts.StepOutcome{
				Success: false,
				Failure: &contracts.StepFailure{
					Kind:    contracts.FailureKindEngine,
					Message: fmt.Sprintf("failed to switch to page %s: %v", *instr.PageID, err),
				},
			}, nil
		}
	}

	// Delegate to underlying session
	return mps.session.Run(ctx, instr)
}

// Close closes the underlying session.
func (mps *MultiPageSession) Close(ctx context.Context) error {
	return mps.session.Close(ctx)
}

// Reset resets the underlying session.
func (mps *MultiPageSession) Reset(ctx context.Context) error {
	return mps.session.Reset(ctx)
}

// PageSwitcher is implemented by sessions that support page switching.
type PageSwitcher interface {
	SetActivePage(ctx context.Context, driverPageID string) error
}

// PageEventHandler handles page lifecycle events during execution.
type PageEventHandler struct {
	pageMatcher *PageMatcher
	log         *logrus.Logger
}

// NewPageEventHandler creates a handler for page events during execution.
func NewPageEventHandler(pageMatcher *PageMatcher, log *logrus.Logger) *PageEventHandler {
	if log == nil {
		log = logrus.StandardLogger()
	}
	return &PageEventHandler{
		pageMatcher: pageMatcher,
		log:         log,
	}
}

// HandlePageCreated processes a page_created event from the driver.
func (h *PageEventHandler) HandlePageCreated(pageID, url, title, openerID string) {
	matchedID := h.pageMatcher.OnPageCreated(pageID, url, title, openerID)
	if matchedID != nil {
		h.log.WithFields(logrus.Fields{
			"page_id":    pageID,
			"matched_to": matchedID,
			"url":        url,
		}).Info("Page created and matched to recorded page")
	} else {
		h.log.WithFields(logrus.Fields{
			"page_id": pageID,
			"url":     url,
		}).Warn("Page created but not matched to any recorded page")
	}
}

// HandlePageClosed processes a page_closed event from the driver.
func (h *PageEventHandler) HandlePageClosed(pageID string) {
	h.pageMatcher.OnPageClosed(pageID)
	h.log.WithField("page_id", pageID).Debug("Page closed")
}

// HandlePageNavigated processes a page_navigated event from the driver.
func (h *PageEventHandler) HandlePageNavigated(pageID, url, title string) {
	h.log.WithFields(logrus.Fields{
		"page_id": pageID,
		"url":     url,
		"title":   title,
	}).Debug("Page navigated")
}

// EnsureMultiPagePlan ensures the plan is multi-page aware (v2 format).
// For v1 plans, this upgrades them to v2 with a single implicit page.
func EnsureMultiPagePlan(plan *contracts.ExecutionPlan) *contracts.ExecutionPlan {
	if plan == nil {
		return nil
	}

	// Already v2?
	if plan.SchemaVersion == contracts.ExecutionPlanSchemaVersionV2 {
		return plan
	}

	// Upgrade v1 to v2
	return contracts.MigratePlanV1ToV2(plan)
}
