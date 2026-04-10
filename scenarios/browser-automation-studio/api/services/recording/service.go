// Package recording provides the unified recording service.
//
// DOC: docs/architecture/recording.md
package recording

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/domain"
	"github.com/vrooli/browser-automation-studio/internal/clock"
	"github.com/vrooli/browser-automation-studio/services/recording/persistence"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
)

// duplicateNavigateThreshold is the time window for detecting duplicate navigate actions.
const duplicateNavigateThreshold = 500 * time.Millisecond

// ActionSource represents the provenance of a recorded action.
type ActionSource string

const (
	// ActionSourceAuto indicates the action was captured automatically by the driver.
	ActionSourceAuto ActionSource = "auto"
	// ActionSourceManual indicates the action was manually entered by a user.
	ActionSourceManual ActionSource = "manual"
	// ActionSourceAI indicates the action was performed by AI navigation.
	ActionSourceAI ActionSource = "ai"
)

// SessionTimeline is the in-memory cache for an active session's timeline.
type SessionTimeline struct {
	// SessionID identifies this timeline's session.
	SessionID string

	// Entries is the chronologically-ordered list of timeline entries.
	Entries []persistence.UnifiedTimelineEntry

	// Sequence tracks the next sequence number for ordering.
	Sequence int
}

// SessionConfig configures a new recording session.
type SessionConfig struct {
	// ProfileID optionally links this session to a SessionProfile.
	ProfileID string

	// ViewportWidth is the browser viewport width in pixels.
	ViewportWidth int

	// ViewportHeight is the browser viewport height in pixels.
	ViewportHeight int
}

// Service provides unified recording functionality.
// It combines in-memory timeline management (for fast WebSocket updates) with
// persistent storage (for durability and historical queries).
//
// All recorded actions and page events flow through this service, regardless
// of how they were initiated (manual recording, AI navigation, playback).
type Service struct {
	repo  persistence.Repository
	wsHub wsHub.HubInterface
	log   *logrus.Logger
	clock clock.Clock

	// Hot cache for active sessions
	cache   map[string]*SessionTimeline
	cacheMu sync.RWMutex

	// Callback for action notifications (WebSocket broadcast)
	onAction func(sessionID string, entry *persistence.UnifiedTimelineEntry)
}

// ServiceConfig configures the recording service.
type ServiceConfig struct {
	// OnAction is called when a new entry is recorded (for WebSocket broadcast).
	OnAction func(sessionID string, entry *persistence.UnifiedTimelineEntry)

	// Clock provides time operations. If nil, uses the real system clock.
	Clock clock.Clock
}

// NewService creates a new unified recording service.
func NewService(repo persistence.Repository, hub wsHub.HubInterface, log *logrus.Logger, config ServiceConfig) *Service {
	clk := config.Clock
	if clk == nil {
		clk = clock.New()
	}
	return &Service{
		repo:     repo,
		wsHub:    hub,
		log:      log,
		clock:    clk,
		cache:    make(map[string]*SessionTimeline),
		onAction: config.OnAction,
	}
}

// CreateSession creates a new recording session.
func (s *Service) CreateSession(ctx context.Context, cfg SessionConfig) (*domain.RecordingSession, error) {
	session := &domain.RecordingSession{
		ID:             uuid.New().String(),
		ProfileID:      cfg.ProfileID,
		Status:         domain.SessionStatusActive,
		ViewportWidth:  cfg.ViewportWidth,
		ViewportHeight: cfg.ViewportHeight,
		CreatedAt:      s.clock.Now(),
	}

	// Persist to database if repo is available
	if s.repo != nil {
		if err := s.repo.CreateSession(ctx, session); err != nil {
			s.log.WithError(err).Error("Failed to create recording session in database")
			return nil, err
		}
	}

	// Initialize hot cache
	s.cacheMu.Lock()
	s.cache[session.ID] = &SessionTimeline{
		SessionID: session.ID,
		Entries:   make([]persistence.UnifiedTimelineEntry, 0),
		Sequence:  1,
	}
	s.cacheMu.Unlock()

	s.log.WithField("session_id", session.ID).Info("Created recording session")
	return session, nil
}

// RegisterSession registers an existing session ID from an external source (like live-capture driver).
// This is used when the session is created by the driver but needs to be tracked for timeline persistence.
func (s *Service) RegisterSession(ctx context.Context, sessionID string, cfg SessionConfig) error {
	session := &domain.RecordingSession{
		ID:             sessionID,
		ProfileID:      cfg.ProfileID,
		Status:         domain.SessionStatusActive,
		ViewportWidth:  cfg.ViewportWidth,
		ViewportHeight: cfg.ViewportHeight,
		CreatedAt:      s.clock.Now(),
	}

	// Persist to database if repo is available
	if s.repo != nil {
		if err := s.repo.CreateSession(ctx, session); err != nil {
			s.log.WithError(err).Error("Failed to register recording session in database")
			return err
		}
	}

	// Initialize hot cache
	s.cacheMu.Lock()
	s.cache[sessionID] = &SessionTimeline{
		SessionID: sessionID,
		Entries:   make([]persistence.UnifiedTimelineEntry, 0),
		Sequence:  1,
	}
	s.cacheMu.Unlock()

	s.log.WithField("session_id", sessionID).Info("Registered recording session")
	return nil
}

// RecordAction records a browser action to the timeline.
// This is the primary entry point for all action recording, ensuring
// consistent handling regardless of action source.
func (s *Service) RecordAction(ctx context.Context, sessionID string, driverAction *driver.RecordedAction, pageID uuid.UUID, source ActionSource) error {
	// Parse timestamp
	ts, err := time.Parse(time.RFC3339Nano, driverAction.Timestamp)
	if err != nil {
		ts = s.clock.Now()
	}

	// Check for duplicate navigate actions
	if s.isDuplicateNavigate(sessionID, driverAction, ts) {
		s.log.WithFields(logrus.Fields{
			"session_id":  sessionID,
			"action_type": driverAction.ActionType,
			"url":         driverAction.URL,
		}).Debug("Skipping duplicate navigate action")
		return nil
	}

	// Convert to domain action
	action := s.convertDriverAction(driverAction, sessionID, pageID, ts, source)

	// Get next sequence number
	s.cacheMu.Lock()
	timeline, ok := s.cache[sessionID]
	if !ok {
		timeline = &SessionTimeline{
			SessionID: sessionID,
			Entries:   make([]persistence.UnifiedTimelineEntry, 0),
			Sequence:  1,
		}
		s.cache[sessionID] = timeline
	}
	sequence := timeline.Sequence
	timeline.Sequence++
	s.cacheMu.Unlock()

	// Create timeline entry
	entry := persistence.UnifiedTimelineEntry{
		ID:        uuid.New(),
		Type:      persistence.TimelineEntryTypeAction,
		Timestamp: ts,
		SessionID: sessionID,
		PageID:    pageID,
		Sequence:  sequence,
		Action:    action,
	}

	// Add to hot cache
	s.cacheMu.Lock()
	s.cache[sessionID].Entries = append(s.cache[sessionID].Entries, entry)
	// Keep cache bounded
	if len(s.cache[sessionID].Entries) > 1000 {
		s.cache[sessionID].Entries = s.cache[sessionID].Entries[500:]
	}
	s.cacheMu.Unlock()

	// Persist to database if available
	if s.repo != nil {
		if err := s.repo.SaveTimelineEntry(ctx, &entry); err != nil {
			s.log.WithError(err).Warn("Failed to persist action to database")
			// Continue anyway - in-memory timeline is authoritative during session
		}
	}

	// Broadcast to WebSocket listeners
	if s.onAction != nil {
		s.onAction(sessionID, &entry)
	}

	s.log.WithFields(logrus.Fields{
		"session_id":  sessionID,
		"action_type": driverAction.ActionType,
		"sequence":    sequence,
	}).Debug("Recorded action")

	return nil
}

// RecordPageEvent records a page lifecycle event to the timeline.
func (s *Service) RecordPageEvent(ctx context.Context, sessionID string, event *domain.PageEvent) error {
	// Get next sequence number
	s.cacheMu.Lock()
	timeline, ok := s.cache[sessionID]
	if !ok {
		timeline = &SessionTimeline{
			SessionID: sessionID,
			Entries:   make([]persistence.UnifiedTimelineEntry, 0),
			Sequence:  1,
		}
		s.cache[sessionID] = timeline
	}
	sequence := timeline.Sequence
	timeline.Sequence++
	s.cacheMu.Unlock()

	// Create timeline entry
	entry := persistence.UnifiedTimelineEntry{
		ID:        uuid.New(),
		Type:      persistence.TimelineEntryTypePageEvent,
		Timestamp: event.Timestamp,
		SessionID: sessionID,
		PageID:    event.PageID,
		Sequence:  sequence,
		PageEvent: event,
	}

	// Add to hot cache
	s.cacheMu.Lock()
	s.cache[sessionID].Entries = append(s.cache[sessionID].Entries, entry)
	s.cacheMu.Unlock()

	// Persist to database if available
	if s.repo != nil {
		if err := s.repo.SaveTimelineEntry(ctx, &entry); err != nil {
			s.log.WithError(err).Warn("Failed to persist page event to database")
		}
	}

	// Broadcast to WebSocket listeners
	if s.onAction != nil {
		s.onAction(sessionID, &entry)
	}

	s.log.WithFields(logrus.Fields{
		"session_id": sessionID,
		"event_type": event.Type,
		"page_id":    event.PageID,
		"sequence":   sequence,
	}).Debug("Recorded page event")

	return nil
}

// GetTimeline returns the timeline for a session.
func (s *Service) GetTimeline(ctx context.Context, query persistence.TimelineQuery) (*persistence.TimelineResponse, error) {
	query.ApplyDefaults()

	// Try hot cache first for active sessions
	s.cacheMu.RLock()
	cached, hasCached := s.cache[query.SessionID]
	s.cacheMu.RUnlock()

	if hasCached && query.Since == nil && query.PageID == nil && len(query.EntryTypes) == 0 {
		// Fast path: return from cache
		entries := cached.Entries
		if len(entries) > query.Limit {
			entries = entries[len(entries)-query.Limit:]
		}

		return &persistence.TimelineResponse{
			Entries:    entries,
			HasMore:    len(cached.Entries) > query.Limit,
			TotalCount: len(cached.Entries),
		}, nil
	}

	// For filtered queries, still use cache if available but apply filters
	if hasCached {
		filtered := s.filterEntries(cached.Entries, query)
		hasMore := len(filtered) > query.Limit
		if hasMore {
			filtered = filtered[:query.Limit]
		}

		return &persistence.TimelineResponse{
			Entries:    filtered,
			HasMore:    hasMore,
			TotalCount: len(cached.Entries),
		}, nil
	}

	// Fall back to database query
	if s.repo != nil {
		return s.repo.GetTimeline(ctx, query)
	}

	return &persistence.TimelineResponse{
		Entries:    []persistence.UnifiedTimelineEntry{},
		HasMore:    false,
		TotalCount: 0,
	}, nil
}

// GetTimelineForPage returns timeline entries for a specific page.
func (s *Service) GetTimelineForPage(ctx context.Context, sessionID string, pageID uuid.UUID, limit int) ([]persistence.UnifiedTimelineEntry, error) {
	query := persistence.TimelineQuery{
		SessionID: sessionID,
		PageID:    &pageID,
		Limit:     limit,
	}

	resp, err := s.GetTimeline(ctx, query)
	if err != nil {
		return nil, err
	}

	return resp.Entries, nil
}

// GetTimelineCount returns the total number of entries for a session.
func (s *Service) GetTimelineCount(sessionID string) int {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	if timeline, ok := s.cache[sessionID]; ok {
		return len(timeline.Entries)
	}
	return 0
}

// CloseSession marks a session as closed.
func (s *Service) CloseSession(ctx context.Context, sessionID string) error {
	// Persist to database if available
	if s.repo != nil {
		if err := s.repo.CloseSession(ctx, sessionID, s.clock.Now()); err != nil {
			s.log.WithError(err).Error("Failed to close recording session in database")
			return err
		}
	}

	// Clear hot cache
	s.cacheMu.Lock()
	delete(s.cache, sessionID)
	s.cacheMu.Unlock()

	s.log.WithField("session_id", sessionID).Info("Closed recording session")
	return nil
}

// ClearSession removes a session from the hot cache without persisting.
// Used when a session ends without explicit close.
func (s *Service) ClearSession(sessionID string) {
	s.cacheMu.Lock()
	delete(s.cache, sessionID)
	s.cacheMu.Unlock()
}

// GetSession retrieves a session by ID.
func (s *Service) GetSession(ctx context.Context, sessionID string) (*domain.RecordingSession, error) {
	if s.repo == nil {
		return nil, nil
	}
	return s.repo.GetSession(ctx, sessionID)
}

// SetOnAction sets the callback for new entry notifications.
func (s *Service) SetOnAction(callback func(sessionID string, entry *persistence.UnifiedTimelineEntry)) {
	s.onAction = callback
}

// WarmCache loads recent entries for a session into the hot cache.
func (s *Service) WarmCache(ctx context.Context, sessionID string) error {
	if s.repo == nil {
		return nil
	}

	query := persistence.TimelineQuery{
		SessionID: sessionID,
		Limit:     100,
	}

	resp, err := s.repo.GetTimeline(ctx, query)
	if err != nil {
		return err
	}

	// Find max sequence number
	maxSeq := 0
	for _, entry := range resp.Entries {
		if entry.Sequence > maxSeq {
			maxSeq = entry.Sequence
		}
	}

	s.cacheMu.Lock()
	s.cache[sessionID] = &SessionTimeline{
		SessionID: sessionID,
		Entries:   resp.Entries,
		Sequence:  maxSeq + 1,
	}
	s.cacheMu.Unlock()

	return nil
}

// isDuplicateNavigate checks if a navigate action is a duplicate.
func (s *Service) isDuplicateNavigate(sessionID string, action *driver.RecordedAction, ts time.Time) bool {
	if action.ActionType != "navigate" || action.URL == "" {
		return false
	}

	s.cacheMu.RLock()
	timeline, ok := s.cache[sessionID]
	s.cacheMu.RUnlock()

	if !ok || len(timeline.Entries) == 0 {
		return false
	}

	// Check recent entries for duplicate navigate to same URL
	checkCount := 5
	if len(timeline.Entries) < checkCount {
		checkCount = len(timeline.Entries)
	}

	for i := len(timeline.Entries) - 1; i >= len(timeline.Entries)-checkCount && i >= 0; i-- {
		entry := timeline.Entries[i]
		if entry.Type == persistence.TimelineEntryTypeAction && entry.Action != nil {
			if entry.Action.ActionType == "navigate" &&
				entry.Action.URL == action.URL &&
				ts.Sub(entry.Timestamp) < duplicateNavigateThreshold {
				return true
			}
		}
	}

	return false
}

// convertDriverAction converts a driver.RecordedAction to domain.RecordingAction.
func (s *Service) convertDriverAction(action *driver.RecordedAction, sessionID string, pageID uuid.UUID, ts time.Time, source ActionSource) *domain.RecordingAction {
	// Parse action ID
	var actionID uuid.UUID
	if action.ID != "" {
		parsed, err := uuid.Parse(action.ID)
		if err != nil {
			actionID = uuid.New()
		} else {
			actionID = parsed
		}
	} else {
		actionID = uuid.New()
	}

	result := &domain.RecordingAction{
		ID:          actionID,
		SessionID:   sessionID,
		PageID:      pageID,
		SequenceNum: action.SequenceNum,
		ActionType:  action.ActionType,
		Timestamp:   ts,
		DurationMs:  action.DurationMs,
		URL:         action.URL,
		PageTitle:   action.PageTitle,
		Confidence:  action.Confidence,
		Source:      domain.ActionSource(source),
		CreatedAt:   s.clock.Now(),
	}

	// Convert selector
	if action.Selector != nil {
		result.Selector = &domain.SelectorSet{
			Primary: action.Selector.Primary,
		}
		if len(action.Selector.Candidates) > 0 {
			result.Selector.Candidates = make([]domain.SelectorCandidate, len(action.Selector.Candidates))
			for i, c := range action.Selector.Candidates {
				result.Selector.Candidates[i] = domain.SelectorCandidate{
					Type:        c.Type,
					Value:       c.Value,
					Confidence:  c.Confidence,
					Specificity: c.Specificity,
				}
			}
		}
	}

	// Convert element metadata
	if action.ElementMeta != nil {
		result.ElementMeta = &domain.ElementMeta{
			TagName:   action.ElementMeta.TagName,
			ID:        action.ElementMeta.ID,
			ClassName: action.ElementMeta.ClassName,
			InnerText: action.ElementMeta.InnerText,
			IsVisible: action.ElementMeta.IsVisible,
			IsEnabled: action.ElementMeta.IsEnabled,
			Role:      action.ElementMeta.Role,
			AriaLabel: action.ElementMeta.AriaLabel,
		}
		if action.ElementMeta.Attributes != nil {
			result.ElementMeta.Attributes = make(map[string]string)
			for k, v := range action.ElementMeta.Attributes {
				result.ElementMeta.Attributes[k] = v
			}
		}
	}

	// Convert bounding box
	if action.BoundingBox != nil {
		result.BoundingBox = &domain.BoundingBox{
			X:      action.BoundingBox.X,
			Y:      action.BoundingBox.Y,
			Width:  action.BoundingBox.Width,
			Height: action.BoundingBox.Height,
		}
	}

	// Copy payload
	if action.Payload != nil {
		result.Payload = make(map[string]interface{})
		for k, v := range action.Payload {
			result.Payload[k] = v
		}
	}

	return result
}

// Ensure Service implements ActionRecorder interface at compile time.
var _ ActionRecorder = (*Service)(nil)

// RecordActionUnified records an action with unified persistence and broadcast.
// This replaces the dual-write pattern by handling both operations atomically
// with full observability into what succeeded and what failed.
//
// DOC: docs/architecture/recording.md#unified-recording
func (s *Service) RecordActionUnified(ctx context.Context, req RecordActionRequest) (*ActionRecordResult, error) {
	// Generate correlation ID if not provided
	correlationID := req.CorrelationID
	if correlationID == "" {
		correlationID = GenerateCorrelationID(req.SessionID)
	}

	result := &ActionRecordResult{
		CorrelationID: correlationID,
		Errors:        make([]ActionRecordError, 0),
	}

	// Validate request
	if req.Action == nil {
		result.Errors = append(result.Errors, ActionRecordError{
			Stage:   "validation",
			Message: "action is nil",
		})
		return result, nil
	}

	// Parse timestamp
	ts, err := time.Parse(time.RFC3339Nano, req.Action.Timestamp)
	if err != nil {
		ts = s.clock.Now()
	}

	// Check for duplicate navigate actions
	if s.isDuplicateNavigate(req.SessionID, req.Action, ts) {
		s.log.WithFields(logrus.Fields{
			"correlation_id": correlationID,
			"session_id":     req.SessionID,
			"action_type":    req.Action.ActionType,
			"url":            req.Action.URL,
		}).Debug("Skipping duplicate navigate action")
		// Not an error, just a no-op
		return result, nil
	}

	// Convert to domain action
	action := s.convertDriverAction(req.Action, req.SessionID, req.PageID, ts, req.Source)
	result.ActionID = action.ID

	// Get next sequence number
	s.cacheMu.Lock()
	timeline, ok := s.cache[req.SessionID]
	if !ok {
		timeline = &SessionTimeline{
			SessionID: req.SessionID,
			Entries:   make([]persistence.UnifiedTimelineEntry, 0),
			Sequence:  1,
		}
		s.cache[req.SessionID] = timeline
	}
	sequence := timeline.Sequence
	timeline.Sequence++
	s.cacheMu.Unlock()

	result.SequenceNum = sequence

	// Create timeline entry
	entry := persistence.UnifiedTimelineEntry{
		ID:        action.ID,
		Type:      persistence.TimelineEntryTypeAction,
		Timestamp: ts,
		SessionID: req.SessionID,
		PageID:    req.PageID,
		Sequence:  sequence,
		Action:    action,
	}

	// Step 1: Add to hot cache (always succeeds)
	s.cacheMu.Lock()
	s.cache[req.SessionID].Entries = append(s.cache[req.SessionID].Entries, entry)
	// Keep cache bounded
	if len(s.cache[req.SessionID].Entries) > 1000 {
		s.cache[req.SessionID].Entries = s.cache[req.SessionID].Entries[500:]
	}
	s.cacheMu.Unlock()

	// Step 2: Persist to database (may fail)
	if s.repo != nil {
		if err := s.repo.SaveTimelineEntry(ctx, &entry); err != nil {
			s.log.WithError(err).WithFields(logrus.Fields{
				"correlation_id": correlationID,
				"session_id":     req.SessionID,
				"action_id":      action.ID,
			}).Warn("Failed to persist action to database")
			result.Errors = append(result.Errors, ActionRecordError{
				Stage:   "persistence",
				Err:     err,
				Message: "failed to save to database",
			})
		} else {
			result.Persisted = true
		}
	} else {
		// No repo configured - consider persisted (in-memory only mode)
		result.Persisted = true
	}

	// Step 3: Broadcast to WebSocket (may fail or have no subscribers)
	if s.wsHub != nil {
		wsEntry := s.toWebSocketEntry(&entry)
		broadcastResult := s.wsHub.BroadcastRecordingEntry(req.SessionID, wsEntry)
		result.SubscriberCount = broadcastResult.SubscriberCount
		result.SentCount = broadcastResult.SentCount
		result.DroppedCount = broadcastResult.DroppedCount
		result.BroadcastSent = broadcastResult.SentCount > 0

		if broadcastResult.DroppedCount > 0 {
			result.Errors = append(result.Errors, ActionRecordError{
				Stage:   "broadcast",
				Message: "some clients had full buffers",
			})
		}
	}

	// Log unified result for observability
	s.log.WithFields(logrus.Fields{
		"correlation_id":   correlationID,
		"session_id":       req.SessionID,
		"action_id":        action.ID,
		"action_type":      req.Action.ActionType,
		"sequence":         sequence,
		"persisted":        result.Persisted,
		"broadcast_sent":   result.BroadcastSent,
		"subscriber_count": result.SubscriberCount,
		"sent_count":       result.SentCount,
		"dropped_count":    result.DroppedCount,
		"error_count":      len(result.Errors),
	}).Debug("Action recorded (unified)")

	return result, nil
}

// RecordPageEventUnified records a page event with unified persistence and broadcast.
func (s *Service) RecordPageEventUnified(ctx context.Context, req RecordPageEventRequest) (*ActionRecordResult, error) {
	// Generate correlation ID if not provided
	correlationID := req.CorrelationID
	if correlationID == "" {
		correlationID = GenerateCorrelationID(req.SessionID)
	}

	result := &ActionRecordResult{
		CorrelationID: correlationID,
		Errors:        make([]ActionRecordError, 0),
	}

	// Validate request
	if req.Event == nil {
		result.Errors = append(result.Errors, ActionRecordError{
			Stage:   "validation",
			Message: "event is nil",
		})
		return result, nil
	}

	result.ActionID = req.Event.ID

	// Get next sequence number
	s.cacheMu.Lock()
	timeline, ok := s.cache[req.SessionID]
	if !ok {
		timeline = &SessionTimeline{
			SessionID: req.SessionID,
			Entries:   make([]persistence.UnifiedTimelineEntry, 0),
			Sequence:  1,
		}
		s.cache[req.SessionID] = timeline
	}
	sequence := timeline.Sequence
	timeline.Sequence++
	s.cacheMu.Unlock()

	result.SequenceNum = sequence

	// Create timeline entry
	entry := persistence.UnifiedTimelineEntry{
		ID:        req.Event.ID,
		Type:      persistence.TimelineEntryTypePageEvent,
		Timestamp: req.Event.Timestamp,
		SessionID: req.SessionID,
		PageID:    req.Event.PageID,
		Sequence:  sequence,
		PageEvent: req.Event,
	}

	// Step 1: Add to hot cache
	s.cacheMu.Lock()
	s.cache[req.SessionID].Entries = append(s.cache[req.SessionID].Entries, entry)
	s.cacheMu.Unlock()

	// Step 2: Persist to database
	if s.repo != nil {
		if err := s.repo.SaveTimelineEntry(ctx, &entry); err != nil {
			s.log.WithError(err).WithFields(logrus.Fields{
				"correlation_id": correlationID,
				"session_id":     req.SessionID,
				"event_id":       req.Event.ID,
			}).Warn("Failed to persist page event to database")
			result.Errors = append(result.Errors, ActionRecordError{
				Stage:   "persistence",
				Err:     err,
				Message: "failed to save to database",
			})
		} else {
			result.Persisted = true
		}
	} else {
		result.Persisted = true
	}

	// Step 3: Broadcast page event via WebSocket
	if s.wsHub != nil {
		s.wsHub.BroadcastPageEvent(req.SessionID, req.Event)
		// Page events don't return BroadcastResult yet, assume sent
		result.BroadcastSent = true
	}

	// Log unified result
	s.log.WithFields(logrus.Fields{
		"correlation_id": correlationID,
		"session_id":     req.SessionID,
		"event_id":       req.Event.ID,
		"event_type":     req.Event.Type,
		"sequence":       sequence,
		"persisted":      result.Persisted,
		"broadcast_sent": result.BroadcastSent,
	}).Debug("Page event recorded (unified)")

	return result, nil
}

// toWebSocketEntry converts a persistence entry to WebSocket format.
func (s *Service) toWebSocketEntry(entry *persistence.UnifiedTimelineEntry) *wsHub.UnifiedTimelineEntry {
	if entry == nil || entry.Action == nil {
		return nil
	}

	// Build selector if present
	var selector map[string]any
	if entry.Action.Selector != nil && entry.Action.Selector.Primary != "" {
		selector = map[string]any{
			"primary": entry.Action.Selector.Primary,
		}
	}

	return &wsHub.UnifiedTimelineEntry{
		ID:        entry.ID.String(),
		Type:      string(entry.Type),
		Timestamp: entry.Timestamp.Format(time.RFC3339Nano),
		PageID:    entry.PageID.String(),
		Action: &wsHub.TimelineAction{
			ID:          entry.Action.ID.String(),
			ActionType:  entry.Action.ActionType,
			SequenceNum: entry.Action.SequenceNum,
			Timestamp:   entry.Timestamp.Format(time.RFC3339Nano),
			Confidence:  entry.Action.Confidence,
			URL:         entry.Action.URL,
			PageTitle:   entry.Action.PageTitle,
			Selector:    selector,
			Payload:     entry.Action.Payload,
		},
	}
}

// filterEntries applies query filters to a slice of entries.
func (s *Service) filterEntries(entries []persistence.UnifiedTimelineEntry, query persistence.TimelineQuery) []persistence.UnifiedTimelineEntry {
	var filtered []persistence.UnifiedTimelineEntry

	for _, entry := range entries {
		// Filter by page ID
		if query.PageID != nil && entry.PageID != *query.PageID {
			continue
		}

		// Filter by timestamp
		if query.Since != nil && !entry.Timestamp.After(*query.Since) {
			continue
		}

		// Filter by entry type
		if len(query.EntryTypes) > 0 {
			found := false
			for _, t := range query.EntryTypes {
				if entry.Type == t {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		filtered = append(filtered, entry)
	}

	// Sort by timestamp
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp.Before(filtered[j].Timestamp)
	})

	return filtered
}

