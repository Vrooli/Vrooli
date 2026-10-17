// Package recording owns committed browser observations.
package recording

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"

	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/automation/telemetry"
	"github.com/vrooli/browser-automation-studio/domain"
	"github.com/vrooli/browser-automation-studio/services/recording/persistence"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	"google.golang.org/protobuf/proto"
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
	driver.RedactSensitiveValues(observed)
	entry := telemetry.BuildRecordingTimelineEntry(observed)
	if entry.Timestamp == nil {
		return errors.New("recording timestamp is required")
	}
	entryID, err := uuid.Parse(entry.GetId())
	if err != nil {
		entryID = uuid.NewSHA1(uuid.NameSpaceOID, []byte(id+":"+entry.GetId()))
		entry.Id = entryID.String()
	}
	recordingSource := basbase.RecordingSource_RECORDING_SOURCE_AUTO
	switch source {
	case ActionSourceManual:
		recordingSource = basbase.RecordingSource_RECORDING_SOURCE_MANUAL
	case ActionSourceAI:
		// The shared proto source enum has no AI value; retain a review cue.
		needsConfirmation := true
		entry.Context.NeedsConfirmation = &needsConfirmation
	case ActionSourceAuto:
	default:
		return fmt.Errorf("unsupported action source %q", source)
	}
	entry.Context.Source = &recordingSource
	return s.RecordTimelineEntry(ctx, id, entry, pageID)
}

// RecordTimelineEntry persists the driver's canonical action proto without projecting it through RecordedAction.
func (s *Service) RecordTimelineEntry(ctx context.Context, sessionID string, observed *bastimeline.TimelineEntry, pageID uuid.UUID) error {
	if observed == nil || observed.GetId() == "" || observed.GetAction() == nil || observed.GetTimestamp() == nil {
		return errors.New("recording timeline entry requires identity, timestamp and action")
	}
	if err := observed.GetTimestamp().CheckValid(); err != nil {
		return fmt.Errorf("recording timestamp: %w", err)
	}
	entryID, err := uuid.Parse(observed.GetId())
	if err != nil {
		return fmt.Errorf("recording entry ID: %w", err)
	}
	canonical := proto.Clone(observed).(*bastimeline.TimelineEntry)
	if canonical.Context == nil {
		canonical.Context = &basbase.EventContext{}
	}
	switch origin := canonical.Context.GetOrigin().(type) {
	case *basbase.EventContext_SessionId:
		if origin.SessionId == "" {
			canonical.Context.Origin = &basbase.EventContext_SessionId{SessionId: sessionID}
		} else if origin.SessionId != sessionID {
			return fmt.Errorf("recording entry session %q does not match owner %q", origin.SessionId, sessionID)
		}
	case nil:
		canonical.Context.Origin = &basbase.EventContext_SessionId{SessionId: sessionID}
	default:
		return errors.New("recording entry has an execution origin")
	}
	driver.RedactSensitiveTimelineEntry(canonical)
	err = s.append(ctx, &persistence.UnifiedTimelineEntry{
		ID: entryID, Type: persistence.TimelineEntryTypeAction, Timestamp: canonical.Timestamp.AsTime(),
		SessionID: sessionID, PageID: pageID, Sequence: int(canonical.GetSequenceNum()), Entry: canonical,
	})
	if err == nil {
		observed.SequenceNum = canonical.GetSequenceNum()
	}
	return err
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
	if entry.Entry != nil {
		entry.Entry.SequenceNum = int32(entry.Sequence)
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
	response, err := s.repo.GetTimeline(ctx, q)
	if err != nil {
		return nil, err
	}
	for i := range response.Entries {
		if response.Entries[i].Entry != nil {
			driver.RedactSensitiveTimelineEntry(response.Entries[i].Entry)
		}
		if response.Entries[i].Action != nil {
			response.Entries[i].Action = redactStoredActionForRead(response.Entries[i].Action)
		}
	}
	return response, nil
}

// redactStoredActionForRead sanitizes the detached repository result, leaving
// existing saved bytes intact while preventing legacy values from reaching
// API readers. The maps are copied because some repository implementations
// return shallow in-memory views.
func redactStoredActionForRead(action *domain.RecordingAction) *domain.RecordingAction {
	copyAction := *action
	if action.ElementMeta != nil {
		meta := *action.ElementMeta
		if action.ElementMeta.Attributes != nil {
			meta.Attributes = make(map[string]string, len(action.ElementMeta.Attributes))
			for key, value := range action.ElementMeta.Attributes {
				meta.Attributes[key] = value
			}
		}
		copyAction.ElementMeta = &meta
	}
	if action.Payload != nil {
		copyAction.Payload = make(map[string]interface{}, len(action.Payload))
		for key, value := range action.Payload {
			copyAction.Payload[key] = value
		}
	}
	if copyAction.ElementMeta != nil {
		meta := copyAction.ElementMeta
		innerText := meta.InnerText
		if driver.RedactSensitiveFields(meta.TagName, &innerText, meta.Attributes, copyAction.Payload) {
			meta.InnerText = innerText
		}
	}
	return &copyAction
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
