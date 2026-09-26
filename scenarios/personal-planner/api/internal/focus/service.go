package focus

import (
	"context"
	"strings"
	"time"

	"github.com/vrooli/api-core/schedule"
)

type Service interface {
	Current(context.Context) (Session, bool, error)
	Start(context.Context, StartInput) (Session, error)
	Pause(context.Context, string, int64) (Session, error)
	Resume(context.Context, string, int64) (Session, error)
	End(context.Context, string, int64) (Session, error)
	SaveNote(context.Context, string, string) (SessionNote, error)
	ListNotes(context.Context, string) ([]SessionNote, error)
	RecordPauseEvent(context.Context, string, string) (PauseEvent, error)
	ListPauseEvents(context.Context, string) ([]PauseEvent, error)
	RecordActual(context.Context, RecordActualInput) (Actual, error)
	ListActuals(context.Context, string) ([]Actual, error)
	ListCorrections(context.Context, string, string, int) ([]Correction, error)
	CorrectActual(context.Context, CorrectActualInput) (Actual, error)
}

type service struct {
	repo  Repository
	clock schedule.Clock
}

func NewService(repo Repository, clock schedule.Clock) Service {
	return &service{repo: repo, clock: clock}
}

func (s *service) Current(ctx context.Context) (Session, bool, error) { return s.repo.Current(ctx) }

func (s *service) Start(ctx context.Context, in StartInput) (Session, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return Session{}, ErrInvalidFocus{"title", "required"}
	}
	mode := strings.TrimSpace(in.Mode)
	if mode == "" {
		mode = ModeOpen
	}
	if mode != ModeOpen && mode != ModePomodoro && mode != ModeTimed && mode != ModeUntimed {
		return Session{}, ErrInvalidFocus{"mode", "must be open, pomodoro, timed, or untimed"}
	}
	if _, ok, err := s.repo.Current(ctx); err != nil {
		return Session{}, err
	} else if ok {
		return Session{}, ErrSessionConflict{"only one running or paused session is allowed"}
	}
	now := s.clock.Now().UTC()
	return s.repo.Create(ctx, Session{WorkItemID: in.WorkItemID, Title: title, Mode: mode, State: StateRunning, StartedAt: now, ActiveStartedAt: &now, Revision: 1})
}

func (s *service) Pause(ctx context.Context, id string, revision int64) (Session, error) {
	return s.transition(ctx, id, revision, StatePaused)
}

func (s *service) Resume(ctx context.Context, id string, revision int64) (Session, error) {
	return s.transition(ctx, id, revision, StateRunning)
}

func (s *service) End(ctx context.Context, id string, revision int64) (Session, error) {
	return s.transition(ctx, id, revision, StateEnded)
}

func (s *service) SaveNote(ctx context.Context, sessionID, note string) (SessionNote, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return SessionNote{}, ErrInvalidFocus{"session_id", "required"}
	}
	note = strings.TrimSpace(note)
	if note == "" {
		return SessionNote{}, ErrInvalidFocus{"note", "required"}
	}
	if len([]rune(note)) > 2000 {
		return SessionNote{}, ErrInvalidFocus{"note", "must be 2000 characters or fewer"}
	}
	session, err := s.repo.Get(ctx, sessionID)
	if err != nil {
		return SessionNote{}, err
	}
	if session.State != StateEnded {
		return SessionNote{}, ErrInvalidFocus{"session_id", "notes are available after ending the session"}
	}
	return s.repo.SaveNote(ctx, SessionNote{SessionID: sessionID, LocalDate: session.StartedAt.In(s.clock.Now().Location()).Format("2006-01-02"), Note: note, UpdatedAt: s.clock.Now().UTC()})
}

func (s *service) ListNotes(ctx context.Context, localDate string) ([]SessionNote, error) {
	if localDate != "" {
		if _, err := time.ParseInLocation("2006-01-02", localDate, s.clock.Now().Location()); err != nil {
			return nil, ErrInvalidFocus{"local_date", "must be YYYY-MM-DD"}
		}
	}
	return s.repo.ListNotes(ctx, localDate)
}

func (s *service) RecordPauseEvent(ctx context.Context, sessionID, reason string) (PauseEvent, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return PauseEvent{}, ErrInvalidFocus{"session_id", "required"}
	}
	reason = strings.TrimSpace(reason)
	switch reason {
	case PauseInterrupted, PauseBlocked, PauseDistracted, PauseRest, PauseOther:
	default:
		return PauseEvent{}, ErrInvalidFocus{"reason", "must be interrupted, blocked, distracted, rest, or other"}
	}
	session, err := s.repo.Get(ctx, sessionID)
	if err != nil {
		return PauseEvent{}, err
	}
	if session.State != StatePaused {
		return PauseEvent{}, ErrInvalidFocus{"session_id", "pause reason requires a paused session"}
	}
	now := s.clock.Now().UTC()
	return s.repo.RecordPauseEvent(ctx, PauseEvent{SessionID: sessionID, LocalDate: now.In(s.clock.Now().Location()).Format("2006-01-02"), Reason: reason, RecordedAt: now})
}

func (s *service) ListPauseEvents(ctx context.Context, localDate string) ([]PauseEvent, error) {
	if localDate != "" {
		if _, err := time.ParseInLocation("2006-01-02", localDate, s.clock.Now().Location()); err != nil {
			return nil, ErrInvalidFocus{"local_date", "must be YYYY-MM-DD"}
		}
	}
	return s.repo.ListPauseEvents(ctx, localDate)
}

func (s *service) RecordActual(ctx context.Context, in RecordActualInput) (Actual, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return Actual{}, ErrInvalidFocus{"title", "required"}
	}
	if in.ReportedMinutes <= 0 {
		return Actual{}, ErrInvalidFocus{"reported_minutes", "must be positive"}
	}
	certainty := strings.TrimSpace(in.Certainty)
	if certainty == "" {
		certainty = "user_reported_approximate"
	}
	if certainty != "timed_observed" && certainty != "user_reported_approximate" {
		return Actual{}, ErrInvalidFocus{"certainty", "must be timed_observed or user_reported_approximate"}
	}
	if _, err := time.ParseInLocation("2006-01-02", in.LocalDate, s.clock.Now().Location()); err != nil {
		return Actual{}, ErrInvalidFocus{"local_date", "must be YYYY-MM-DD"}
	}
	return s.repo.CreateActual(ctx, Actual{WorkItemID: strings.TrimSpace(in.WorkItemID), Title: title, LocalDate: in.LocalDate, AllocationID: strings.TrimSpace(in.AllocationID), ReportedMinutes: in.ReportedMinutes, Certainty: certainty, Note: strings.TrimSpace(in.Note), CreatedAt: s.clock.Now().UTC(), Revision: 1})
}

func (s *service) ListActuals(ctx context.Context, localDate string) ([]Actual, error) {
	if localDate != "" {
		if _, err := time.ParseInLocation("2006-01-02", localDate, s.clock.Now().Location()); err != nil {
			return nil, ErrInvalidFocus{"local_date", "must be YYYY-MM-DD"}
		}
	}
	return s.repo.ListActuals(ctx, localDate)
}

func (s *service) ListCorrections(ctx context.Context, actualID, localDate string, limit int) ([]Correction, error) {
	if localDate != "" {
		if _, err := time.ParseInLocation("2006-01-02", localDate, s.clock.Now().Location()); err != nil {
			return nil, ErrInvalidFocus{"local_date", "must be YYYY-MM-DD"}
		}
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListCorrections(ctx, strings.TrimSpace(actualID), localDate, limit)
}

func (s *service) CorrectActual(ctx context.Context, in CorrectActualInput) (Actual, error) {
	if in.ID == "" || in.ReportedMinutes <= 0 {
		return Actual{}, ErrInvalidFocus{"actual", "id and positive reported_minutes are required"}
	}
	certainty := strings.TrimSpace(in.Certainty)
	if certainty == "" {
		certainty = "user_reported_approximate"
	}
	if certainty != "timed_observed" && certainty != "user_reported_approximate" {
		return Actual{}, ErrInvalidFocus{"certainty", "must be timed_observed or user_reported_approximate"}
	}
	return s.repo.CorrectActual(ctx, Actual{ID: in.ID, ReportedMinutes: in.ReportedMinutes, Certainty: certainty, Note: strings.TrimSpace(in.Note)}, in.ExpectedRevision)
}

func (s *service) transition(ctx context.Context, id string, revision int64, next string) (Session, error) {
	current, err := s.repo.Get(ctx, id)
	if err != nil {
		return Session{}, err
	}
	if current.Revision != revision {
		return Session{}, ErrRevisionConflict{id}
	}
	if next == StatePaused && current.State != StateRunning {
		return Session{}, ErrInvalidTransition{current.State, "pause"}
	}
	if next == StateRunning && current.State != StatePaused {
		return Session{}, ErrInvalidTransition{current.State, "resume"}
	}
	if next == StateEnded && current.State != StateRunning && current.State != StatePaused {
		return Session{}, ErrInvalidTransition{current.State, "end"}
	}
	now := s.clock.Now().UTC()
	if current.State == StateRunning && current.ActiveStartedAt != nil {
		current.ActiveSeconds += maxSeconds(now.Sub(*current.ActiveStartedAt))
	}
	current.WallSeconds = maxSeconds(now.Sub(current.StartedAt))
	current.ActiveStartedAt = nil
	current.State = next
	if next == StateRunning {
		current.ActiveStartedAt = &now
	}
	if next == StateEnded {
		current.EndedAt = now
	}
	return s.repo.Transition(ctx, current, revision)
}

func maxSeconds(duration time.Duration) int64 {
	if duration <= 0 {
		return 0
	}
	return int64(duration / time.Second)
}
