package calendar

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Event is a native calendar entry. All-day dates are civil dates with an
// exclusive end; timed values are RFC3339 instants interpreted in Timezone.
type Event struct {
	ID                 string    `json:"id"`
	Title              string    `json:"title"`
	Subject            string    `json:"subject"`
	Notes              string    `json:"notes"`
	Availability       string    `json:"availability"`
	Timezone           string    `json:"timezone"`
	AllDay             bool      `json:"all_day"`
	StartDate          string    `json:"start_date"`
	EndDateExclusive   string    `json:"end_date_exclusive"`
	StartAt            string    `json:"start_at"`
	EndAt              string    `json:"end_at"`
	Provider           string    `json:"provider"`
	ProviderCalendarID string    `json:"provider_calendar_id"`
	ProviderEventID    string    `json:"provider_event_id"`
	OccurrenceID       string    `json:"occurrence_id"`
	Revision           int64     `json:"revision"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type CreateEventInput struct {
	Event
	IdempotencyKey string `json:"idempotency_key"`
}

type UpdateEventInput struct {
	Event
	ExpectedRevision int64 `json:"expected_revision"`
}

type EventRepository interface {
	ListEvents(context.Context, string, string) ([]Event, error)
	GetEvent(context.Context, string) (Event, error)
	GetEventByIdempotencyKey(context.Context, string) (Event, error)
	CreateEvent(context.Context, CreateEventInput) (Event, error)
	UpdateEvent(context.Context, UpdateEventInput) (Event, error)
}

type EventService interface {
	List(context.Context, string, string) ([]Event, error)
	Get(context.Context, string) (Event, error)
	GetByIdempotencyKey(context.Context, string) (Event, error)
	Create(context.Context, CreateEventInput) (Event, error)
	Update(context.Context, UpdateEventInput) (Event, error)
}

type eventService struct{ repo EventRepository }

func NewEventService(repo EventRepository) EventService { return &eventService{repo: repo} }

func (s *eventService) List(ctx context.Context, start, end string) ([]Event, error) {
	if !validCivilDate(start) || !validCivilDate(end) || end < start {
		return nil, ErrInvalidAllocation{"date_range", "must be an ordered YYYY-MM-DD range"}
	}
	return s.repo.ListEvents(ctx, start, end)
}

func (s *eventService) Get(ctx context.Context, id string) (Event, error) {
	if strings.TrimSpace(id) == "" {
		return Event{}, ErrInvalidAllocation{"event_id", "required"}
	}
	return s.repo.GetEvent(ctx, id)
}

func (s *eventService) GetByIdempotencyKey(ctx context.Context, key string) (Event, error) {
	if strings.TrimSpace(key) == "" {
		return Event{}, ErrInvalidAllocation{"idempotency_key", "required"}
	}
	return s.repo.GetEventByIdempotencyKey(ctx, key)
}

func (s *eventService) Create(ctx context.Context, in CreateEventInput) (Event, error) {
	if strings.TrimSpace(in.IdempotencyKey) == "" {
		return Event{}, ErrInvalidAllocation{"idempotency_key", "required"}
	}
	if err := validateEvent(in.Event); err != nil {
		return Event{}, err
	}
	return s.repo.CreateEvent(ctx, in)
}

func (s *eventService) Update(ctx context.Context, in UpdateEventInput) (Event, error) {
	if strings.TrimSpace(in.ID) == "" {
		return Event{}, ErrInvalidAllocation{"event_id", "required"}
	}
	if in.ExpectedRevision <= 0 {
		return Event{}, ErrInvalidAllocation{"expected_revision", "must be positive"}
	}
	if err := validateEvent(in.Event); err != nil {
		return Event{}, err
	}
	return s.repo.UpdateEvent(ctx, in)
}

func validateEvent(e Event) error {
	if strings.TrimSpace(e.Title) == "" {
		return ErrInvalidAllocation{"title", "required"}
	}
	if e.Availability != "busy" && e.Availability != "free" {
		return ErrInvalidAllocation{"availability", "must be busy or free"}
	}
	if _, err := time.LoadLocation(e.Timezone); err != nil {
		return ErrInvalidAllocation{"timezone", "must be an IANA timezone"}
	}
	if e.AllDay {
		if !validCivilDate(e.StartDate) || !validCivilDate(e.EndDateExclusive) || e.EndDateExclusive <= e.StartDate {
			return ErrInvalidAllocation{"date_range", "all-day events require an ordered start date and exclusive end date"}
		}
		if e.StartAt != "" || e.EndAt != "" {
			return ErrInvalidAllocation{"time_range", "all-day events cannot include timed instants"}
		}
		return nil
	}
	start, startErr := time.Parse(time.RFC3339Nano, e.StartAt)
	end, endErr := time.Parse(time.RFC3339Nano, e.EndAt)
	if startErr != nil || endErr != nil || !end.After(start) {
		return ErrInvalidAllocation{"time_range", "timed events require an ordered RFC3339 instant interval"}
	}
	if e.StartDate != "" || e.EndDateExclusive != "" {
		return ErrInvalidAllocation{"date_range", "timed events use instants, not all-day dates"}
	}
	return nil
}

func validCivilDate(value string) bool {
	t, err := time.Parse("2006-01-02", value)
	return err == nil && t.Format("2006-01-02") == value
}

type ErrEventNotFound struct{ ID string }

func (e ErrEventNotFound) Error() string { return fmt.Sprintf("calendar event %q not found", e.ID) }

type ErrEventRevisionConflict struct{ Expected, Current int64 }

func (e ErrEventRevisionConflict) Error() string {
	return fmt.Sprintf("calendar event changed from revision %d to %d; reload before editing", e.Expected, e.Current)
}

type ErrEventIdempotencyConflict struct{ Key string }

func (e ErrEventIdempotencyConflict) Error() string {
	return fmt.Sprintf("idempotency key %q was already used for a different event", e.Key)
}

type ErrEventIdentityConflict struct{ Provider, CalendarID, EventID, OccurrenceID string }

func (e ErrEventIdentityConflict) Error() string {
	return fmt.Sprintf("provider event identity %s/%s/%s/%s already exists with different content", e.Provider, e.CalendarID, e.EventID, e.OccurrenceID)
}
