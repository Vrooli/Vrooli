package calendar

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/api-core/schedule"
	_ "modernc.org/sqlite"
)

type eventRepoStub struct {
	event  Event
	input  CreateEventInput
	update UpdateEventInput
}

func (r *eventRepoStub) ListEvents(context.Context, string, string) ([]Event, error) {
	return []Event{r.event}, nil
}
func (r *eventRepoStub) GetEvent(context.Context, string) (Event, error) { return r.event, nil }
func (r *eventRepoStub) GetEventByIdempotencyKey(context.Context, string) (Event, error) {
	return r.event, nil
}
func (r *eventRepoStub) CreateEvent(_ context.Context, input CreateEventInput) (Event, error) {
	r.input = input
	return input.Event, nil
}
func (r *eventRepoStub) UpdateEvent(_ context.Context, input UpdateEventInput) (Event, error) {
	r.update = input
	return input.Event, nil
}

func TestEventServicePreservesCivilDatesAndExclusiveEnd(t *testing.T) {
	repo := &eventRepoStub{}
	event := Event{Title: "South Jersey", Subject: "owner", AllDay: true,
		StartDate: "2026-10-04", EndDateExclusive: "2026-10-07", Timezone: "America/New_York", Availability: "free"}
	got, err := NewEventService(repo).Create(context.Background(), CreateEventInput{Event: event, IdempotencyKey: "synthetic-1"})
	if err != nil {
		t.Fatal(err)
	}
	if got.StartDate != "2026-10-04" || got.EndDateExclusive != "2026-10-07" || got.StartAt != "" || got.EndAt != "" {
		t.Fatalf("date-only event gained or lost temporal data: %#v", got)
	}
}

func TestEventServicePreservesDistinctInstantsAcrossNewYorkDSTFold(t *testing.T) {
	repo := &eventRepoStub{}
	event := Event{Title: "DST fold", Timezone: "America/New_York", Availability: "busy",
		StartAt: "2026-11-01T05:30:00Z", EndAt: "2026-11-01T06:30:00Z"}
	got, err := NewEventService(repo).Create(context.Background(), CreateEventInput{Event: event, IdempotencyKey: "synthetic-dst"})
	if err != nil {
		t.Fatal(err)
	}
	if got.StartAt != "2026-11-01T05:30:00Z" || got.EndAt != "2026-11-01T06:30:00Z" {
		t.Fatalf("DST fold instants changed: %#v", got)
	}
}

func TestEventServiceRejectsNonExclusiveAllDayRangeAndAcceptsRevisionedEdit(t *testing.T) {
	repo := &eventRepoStub{}
	service := NewEventService(repo)
	invalid := Event{Title: "bad range", AllDay: true, StartDate: "2026-10-04", EndDateExclusive: "2026-10-04", Timezone: "America/New_York", Availability: "free"}
	if _, err := service.Create(context.Background(), CreateEventInput{Event: invalid, IdempotencyKey: "synthetic-invalid"}); err == nil {
		t.Fatal("expected empty all-day interval to be rejected")
	}
	valid := Event{ID: "event-1", Title: "edited", AllDay: true, StartDate: "2026-10-04", EndDateExclusive: "2026-10-05", Timezone: "America/New_York", Availability: "free"}
	if _, err := service.Update(context.Background(), UpdateEventInput{Event: valid, ExpectedRevision: 4}); err != nil {
		t.Fatal(err)
	}
	if repo.update.ExpectedRevision != 4 || repo.update.ID != "event-1" {
		t.Fatalf("edit lost durable identity or revision: %#v", repo.update)
	}
}

func TestEventServiceRejectsInvalidCivilDateRange(t *testing.T) {
	if _, err := NewEventService(&eventRepoStub{}).List(context.Background(), "2026-02-30", "2026-03-01"); err == nil {
		t.Fatal("expected invalid civil date to be rejected")
	}
}

func TestSQLiteEventsRetainIdentityRevisionAndIdempotentCreateAcrossReopen(t *testing.T) {
	db, err := sql.Open("sqlite", "file:calendar-native-events-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	clock := schedule.NewFake(time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC))
	repo := NewSQLiteRepository(db, clock).(EventRepository)
	input := CreateEventInput{IdempotencyKey: "synthetic-create-1", Event: Event{
		Title: "South Jersey", Subject: "owner", AllDay: true, StartDate: "2026-10-04", EndDateExclusive: "2026-10-07",
		Timezone: "America/New_York", Availability: "free", Provider: "fixture", ProviderCalendarID: "fixture-calendar",
		ProviderEventID: "fixture-event-1",
	}}
	created, err := repo.CreateEvent(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	statusReadback, err := repo.GetEventByIdempotencyKey(context.Background(), input.IdempotencyKey)
	if err != nil || statusReadback.ID != created.ID {
		t.Fatalf("idempotency lookup did not recover the durable event: event=%#v err=%v", statusReadback, err)
	}
	retried, err := repo.CreateEvent(context.Background(), input)
	if err != nil || retried.ID != created.ID || retried.Revision != 1 {
		t.Fatalf("idempotent retry created another event: event=%#v err=%v", retried, err)
	}
	providerReplay := input
	providerReplay.IdempotencyKey = "synthetic-provider-replay"
	replayed, err := repo.CreateEvent(context.Background(), providerReplay)
	if err != nil || replayed.ID != created.ID {
		t.Fatalf("provider identity replay created another durable event: event=%#v err=%v", replayed, err)
	}
	changedReplay := providerReplay
	changedReplay.IdempotencyKey = "synthetic-provider-changed"
	changedReplay.Event.Title = "Different source payload"
	if _, err := repo.CreateEvent(context.Background(), changedReplay); err == nil {
		t.Fatal("expected an existing provider identity with changed content to conflict")
	}
	clock.Advance(time.Minute)
	edit := UpdateEventInput{ExpectedRevision: 1, Event: Event{ID: created.ID, Title: "South Jersey edited", Subject: "owner", AllDay: true,
		StartDate: "2026-10-04", EndDateExclusive: "2026-10-07", Timezone: "America/New_York", Availability: "free"}}
	updated, err := repo.UpdateEvent(context.Background(), edit)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != created.ID || updated.Revision != 2 || updated.ProviderEventID != "fixture-event-1" || updated.EndDateExclusive != "2026-10-07" {
		t.Fatalf("edit changed durable identity or event semantics: %#v", updated)
	}
	if _, err := repo.UpdateEvent(context.Background(), edit); err == nil {
		t.Fatal("expected stale event revision to conflict")
	}
	reopened, err := NewSQLiteRepository(db, clock).(EventRepository).GetEvent(context.Background(), created.ID)
	if err != nil || reopened.Title != "South Jersey edited" || reopened.Revision != 2 {
		t.Fatalf("reopened event=%#v err=%v", reopened, err)
	}
	listed, err := repo.ListEvents(context.Background(), "2026-10-06", "2026-10-06")
	if err != nil || len(listed) != 1 || listed[0].ID != created.ID {
		t.Fatalf("overlap list=%#v err=%v", listed, err)
	}
	listed, err = repo.ListEvents(context.Background(), "2026-10-07", "2026-10-07")
	if err != nil || len(listed) != 0 {
		t.Fatalf("exclusive all-day end was included: events=%#v err=%v", listed, err)
	}
	dst, err := repo.CreateEvent(context.Background(), CreateEventInput{IdempotencyKey: "synthetic-dst-persist", Event: Event{
		Title: "DST fold", Timezone: "America/New_York", Availability: "busy",
		StartAt: "2026-11-01T05:30:00Z", EndAt: "2026-11-01T06:30:00Z",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if dst.StartAt != "2026-11-01T05:30:00Z" || dst.EndAt != "2026-11-01T06:30:00Z" {
		t.Fatalf("persisted DST fold instants changed: %#v", dst)
	}
	dstDay, err := repo.ListEvents(context.Background(), "2026-11-01", "2026-11-01")
	if err != nil || len(dstDay) != 1 || dstDay[0].ID != dst.ID {
		t.Fatalf("DST fold event missing from its New York civil date: events=%#v err=%v", dstDay, err)
	}
}

func TestSQLiteCalendarManifestRetainsAllAcceptedFixtureIdentities(t *testing.T) {
	manifestPath := filepath.Join("..", "..", "..", "docs", "internal", "goal", "sources", "calendar-manifest.json")
	encoded, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Events []struct {
			Title            string `json:"title"`
			Subject          string `json:"subject"`
			AllDay           bool   `json:"all_day"`
			Timezone         string `json:"timezone"`
			StartDate        string `json:"start_date"`
			EndDateExclusive string `json:"end_date_exclusive"`
			Notes            string `json:"notes"`
			GoogleCalendarID string `json:"google_calendar_id"`
			GoogleEventID    string `json:"google_event_id"`
			SemanticCategory string `json:"semantic_category"`
		} `json:"events"`
	}
	if err := json.Unmarshal(encoded, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Events) != 21 {
		t.Fatalf("accepted calendar manifest has %d identities, want 21", len(manifest.Events))
	}
	db, err := sql.Open("sqlite", "file:calendar-manifest-fixtures?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLiteRepository(db, schedule.NewFake(time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC))).(EventRepository)
	want := make(map[string]Event, len(manifest.Events))
	for i, fixture := range manifest.Events {
		availability := "busy"
		if fixture.SemanticCategory == "girlfriend_awareness" || fixture.SemanticCategory == "family_awareness" {
			availability = "free"
		}
		event := Event{Title: fixture.Title, Subject: fixture.Subject, Notes: fixture.Notes, Availability: availability,
			Timezone: fixture.Timezone, AllDay: fixture.AllDay, StartDate: fixture.StartDate, EndDateExclusive: fixture.EndDateExclusive,
			Provider: "fixture", ProviderCalendarID: fixture.GoogleCalendarID, ProviderEventID: fixture.GoogleEventID}
		created, err := repo.CreateEvent(context.Background(), CreateEventInput{Event: event, IdempotencyKey: "manifest-fixture-" + fixture.GoogleEventID})
		if err != nil {
			t.Fatalf("fixture %d (%s): %v", i, fixture.GoogleEventID, err)
		}
		want[fixture.GoogleEventID] = created
	}
	got, err := repo.ListEvents(context.Background(), "2026-10-01", "2026-12-02")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 21 {
		t.Fatalf("durable fixture count=%d, want 21", len(got))
	}
	for _, event := range got {
		original, ok := want[event.ProviderEventID]
		if !ok || event.ID != original.ID || event.Revision != 1 || event.StartDate != original.StartDate || event.EndDateExclusive != original.EndDateExclusive || event.Subject != original.Subject || event.Availability != original.Availability {
			t.Errorf("fixture identity or civil-date semantics changed: %#v", event)
		}
	}
	october31, err := repo.ListEvents(context.Background(), "2026-10-31", "2026-10-31")
	if err != nil {
		t.Fatal(err)
	}
	var hike, candy int
	for _, event := range october31 {
		if event.Title == "Foliage hike" {
			hike++
		}
		if event.Title == "Handing out candy at girlfriend’s parents’ house" {
			candy++
		}
	}
	if hike != 1 || candy != 1 {
		t.Fatalf("same-day overlapping personal events were lost: hike=%d candy=%d events=%#v", hike, candy, october31)
	}
	november1, err := repo.ListEvents(context.Background(), "2026-11-01", "2026-11-01")
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range november1 {
		if event.Title == "South Jersey" && (event.StartDate != "2026-11-01" || event.EndDateExclusive != "2026-11-04") {
			t.Fatalf("New York DST date-only event shifted: %#v", event)
		}
	}
}
