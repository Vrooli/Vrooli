// Package recording owns committed browser observations.
package recording

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/domain"
	"github.com/vrooli/browser-automation-studio/services/recording/persistence"
)

var ErrRepositoryUnavailable = errors.New("recording journal unavailable")

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

// SessionConfig configures a new recording session.
type SessionConfig struct {
	// ProfileID optionally links this session to a SessionProfile.
	ProfileID string

	// ViewportWidth is the browser viewport width in pixels.
	ViewportWidth int

	// ViewportHeight is the browser viewport height in pixels.
	ViewportHeight int
}

// Service uses the repository as the sole history and sequence authority.
type Service struct {
	repo     persistence.Repository
	clock    schedule.Clock
	notifyMu sync.RWMutex
	onAction func(string, *persistence.UnifiedTimelineEntry)
}

type ServiceConfig struct {
	// OnAction receives only committed entries, once per newly appended identity.
	OnAction func(string, *persistence.UnifiedTimelineEntry)
	Clock    schedule.Clock
}

func NewService(repo persistence.Repository, config ServiceConfig) *Service {
	clk := config.Clock
	if clk == nil {
		clk = schedule.System()
	}
	return &Service{repo: repo, clock: clk, onAction: config.OnAction}
}

func (s *Service) CreateSession(ctx context.Context, cfg SessionConfig) (*domain.RecordingSession, error) {
	session := s.newSession(uuid.NewString(), cfg)
	if s.repo == nil {
		return nil, ErrRepositoryUnavailable
	}
	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *Service) newSession(id string, cfg SessionConfig) *domain.RecordingSession {
	return &domain.RecordingSession{
		ID: id, ProfileID: cfg.ProfileID, Status: domain.SessionStatusActive,
		ViewportWidth: cfg.ViewportWidth, ViewportHeight: cfg.ViewportHeight,
		CreatedAt: s.clock.Now(),
	}
}

func (s *Service) RegisterSession(ctx context.Context, id string, cfg SessionConfig) error {
	if s.repo == nil {
		return ErrRepositoryUnavailable
	}
	return s.repo.CreateSession(ctx, s.newSession(id, cfg))
}

// RecordAction acknowledges only durable writes. Event IDs, not URL/time guesses,
// identify retried observations; distinct navigations remain distinct history.
func (s *Service) RecordAction(ctx context.Context, id string, observed *driver.RecordedAction, pageID uuid.UUID, source ActionSource) error {
	if observed == nil || observed.ID == "" || observed.ActionType == "" {
		return errors.New("recording action requires identity and type")
	}
	ts, err := time.Parse(time.RFC3339Nano, observed.Timestamp)
	if err != nil {
		return fmt.Errorf("recording timestamp: %w", err)
	}
	action := s.convertDriverAction(observed, id, pageID, ts, source)
	return s.append(ctx, &persistence.UnifiedTimelineEntry{
		ID: action.ID, Type: persistence.TimelineEntryTypeAction, Timestamp: ts,
		SessionID: id, PageID: pageID, Action: action,
	})
}

func (s *Service) RecordPageEvent(ctx context.Context, id string, event *domain.PageEvent) error {
	if event == nil || event.ID == uuid.Nil {
		return errors.New("recording page event requires identity")
	}
	return s.append(ctx, &persistence.UnifiedTimelineEntry{
		ID: event.ID, Type: persistence.TimelineEntryTypePageEvent, Timestamp: event.Timestamp,
		SessionID: id, PageID: event.PageID, PageEvent: event,
	})
}

func (s *Service) append(ctx context.Context, entry *persistence.UnifiedTimelineEntry) error {
	if s.repo == nil {
		return ErrRepositoryUnavailable
	}
	inserted, err := s.repo.AppendTimelineEntry(ctx, entry)
	if err != nil {
		return err
	}
	s.notifyMu.RLock()
	notify := s.onAction
	s.notifyMu.RUnlock()
	if inserted && notify != nil {
		notify(entry.SessionID, entry)
	}
	return nil
}

func (s *Service) GetTimeline(ctx context.Context, q persistence.TimelineQuery) (*persistence.TimelineResponse, error) {
	if s.repo == nil {
		return nil, ErrRepositoryUnavailable
	}
	q.ApplyDefaults()
	return s.repo.GetTimeline(ctx, q)
}

func (s *Service) GetTimelineForPage(ctx context.Context, id string, page uuid.UUID, limit int) ([]persistence.UnifiedTimelineEntry, error) {
	result, err := s.GetTimeline(ctx, persistence.TimelineQuery{SessionID: id, PageID: &page, Limit: limit})
	if err != nil {
		return nil, err
	}
	return result.Entries, nil
}

func (s *Service) CloseSession(ctx context.Context, id string) error {
	if s.repo == nil {
		return ErrRepositoryUnavailable
	}
	return s.repo.CloseSession(ctx, id, s.clock.Now())
}

func (s *Service) GetSession(ctx context.Context, id string) (*domain.RecordingSession, error) {
	if s.repo == nil {
		return nil, ErrRepositoryUnavailable
	}
	return s.repo.GetSession(ctx, id)
}

func (s *Service) SetOnAction(callback func(string, *persistence.UnifiedTimelineEntry)) {
	s.notifyMu.Lock()
	defer s.notifyMu.Unlock()
	s.onAction = callback
}

// convertDriverAction converts a driver.RecordedAction to domain.RecordingAction.
func (s *Service) convertDriverAction(action *driver.RecordedAction, sessionID string, pageID uuid.UUID, ts time.Time, source ActionSource) *domain.RecordingAction {
	actionID, err := uuid.Parse(action.ID)
	if err != nil {
		// Capture sources may use opaque IDs; preserve retry identity per session.
		actionID = uuid.NewSHA1(uuid.NameSpaceOID, []byte(sessionID+":"+action.ID))
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
