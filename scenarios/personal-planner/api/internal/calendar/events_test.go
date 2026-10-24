package calendar

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/schedule"
	_ "modernc.org/sqlite"
)

func eventOwnerContext(subject string) context.Context {
	return identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: subject, Verified: true})
}

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
	ownerCtx := eventOwnerContext("owner-a")
	created, err := repo.CreateEvent(ownerCtx, input)
	if err != nil {
		t.Fatal(err)
	}
	statusReadback, err := repo.GetEventByIdempotencyKey(ownerCtx, input.IdempotencyKey)
	if err != nil || statusReadback.ID != created.ID {
		t.Fatalf("idempotency lookup did not recover the durable event: event=%#v err=%v", statusReadback, err)
	}
	retried, err := repo.CreateEvent(ownerCtx, input)
	if err != nil || retried.ID != created.ID || retried.Revision != 1 {
		t.Fatalf("idempotent retry created another event: event=%#v err=%v", retried, err)
	}
	providerReplay := input
	providerReplay.IdempotencyKey = "synthetic-provider-replay"
	replayed, err := repo.CreateEvent(ownerCtx, providerReplay)
	if err != nil || replayed.ID != created.ID {
		t.Fatalf("provider identity replay created another durable event: event=%#v err=%v", replayed, err)
	}
	changedReplay := providerReplay
	changedReplay.IdempotencyKey = "synthetic-provider-changed"
	changedReplay.Event.Title = "Different source payload"
	if _, err := repo.CreateEvent(ownerCtx, changedReplay); err == nil {
		t.Fatal("expected an existing provider identity with changed content to conflict")
	}
	clock.Advance(time.Minute)
	edit := UpdateEventInput{ExpectedRevision: 1, Event: Event{ID: created.ID, Title: "South Jersey edited", Subject: "owner", AllDay: true,
		StartDate: "2026-10-04", EndDateExclusive: "2026-10-07", Timezone: "America/New_York", Availability: "free"}}
	updated, err := repo.UpdateEvent(ownerCtx, edit)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != created.ID || updated.Revision != 2 || updated.ProviderEventID != "fixture-event-1" || updated.EndDateExclusive != "2026-10-07" {
		t.Fatalf("edit changed durable identity or event semantics: %#v", updated)
	}
	if _, err := repo.UpdateEvent(ownerCtx, edit); err == nil {
		t.Fatal("expected stale event revision to conflict")
	}
	reopened, err := NewSQLiteRepository(db, clock).(EventRepository).GetEvent(ownerCtx, created.ID)
	if err != nil || reopened.Title != "South Jersey edited" || reopened.Revision != 2 {
		t.Fatalf("reopened event=%#v err=%v", reopened, err)
	}
	listed, err := repo.ListEvents(ownerCtx, "2026-10-06", "2026-10-06")
	if err != nil || len(listed) != 1 || listed[0].ID != created.ID {
		t.Fatalf("overlap list=%#v err=%v", listed, err)
	}
	listed, err = repo.ListEvents(ownerCtx, "2026-10-07", "2026-10-07")
	if err != nil || len(listed) != 0 {
		t.Fatalf("exclusive all-day end was included: events=%#v err=%v", listed, err)
	}
	dst, err := repo.CreateEvent(ownerCtx, CreateEventInput{IdempotencyKey: "synthetic-dst-persist", Event: Event{
		Title: "DST fold", Timezone: "America/New_York", Availability: "busy",
		StartAt: "2026-11-01T05:30:00Z", EndAt: "2026-11-01T06:30:00Z",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if dst.StartAt != "2026-11-01T05:30:00Z" || dst.EndAt != "2026-11-01T06:30:00Z" {
		t.Fatalf("persisted DST fold instants changed: %#v", dst)
	}
	dstDay, err := repo.ListEvents(ownerCtx, "2026-11-01", "2026-11-01")
	if err != nil || len(dstDay) != 1 || dstDay[0].ID != dst.ID {
		t.Fatalf("DST fold event missing from its New York civil date: events=%#v err=%v", dstDay, err)
	}
}

func TestSQLiteEventsAreBoundToVerifiedOwnerAndLeaveLegacyRowsUnowned(t *testing.T) {
	db, err := sql.Open("sqlite", "file:calendar-event-owners-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	// Simulate a pre-existing row from the additive migration boundary. It has
	// no owner relation and must remain invisible and immutable to all actors.
	_, err = db.Exec(`INSERT INTO calendar_events
		(id,title,subject,availability,timezone,all_day,start_date,end_date_exclusive,range_start,range_end,revision,created_at,updated_at)
		VALUES ('legacy-event','Legacy private event','claimed-owner','busy','America/New_York',1,'2026-10-04','2026-10-05','2026-10-04','2026-10-05',7,'2026-10-01T00:00:00Z','2026-10-01T00:00:00Z')`)
	if err != nil {
		t.Fatal(err)
	}
	repo := NewSQLiteRepository(db, schedule.System()).(EventRepository)
	ownerA, ownerB := eventOwnerContext("verified-owner-a"), eventOwnerContext("verified-owner-b")
	created, err := repo.CreateEvent(ownerA, CreateEventInput{IdempotencyKey: "owner-key", Event: Event{
		Title: "Owned event", Subject: "verified-owner-b", Availability: "busy", Timezone: "America/New_York",
		AllDay: true, StartDate: "2026-10-04", EndDateExclusive: "2026-10-06",
	}})
	if err != nil || created.ID == "" || created.Revision != 1 {
		t.Fatalf("create by verified owner: event=%#v err=%v", created, err)
	}
	replayed, err := repo.CreateEvent(ownerA, CreateEventInput{IdempotencyKey: "owner-key", Event: Event{
		Title: "Owned event", Subject: "verified-owner-b", Availability: "busy", Timezone: "America/New_York",
		AllDay: true, StartDate: "2026-10-04", EndDateExclusive: "2026-10-06",
	}})
	if err != nil || replayed.ID != created.ID || replayed.Revision != created.Revision {
		t.Fatalf("same-owner retry changed event identity/revision: event=%#v err=%v", replayed, err)
	}
	if _, err := repo.CreateEvent(ownerB, CreateEventInput{IdempotencyKey: "owner-key", Event: created}); err == nil {
		t.Fatal("cross-owner idempotency key returned/adopted another owner's event")
	}
	if _, err := repo.GetEvent(ownerB, created.ID); err == nil {
		t.Fatal("different owner read event")
	} else {
		var missing ErrEventNotFound
		if !errors.As(err, &missing) {
			t.Fatalf("cross-owner get error=%T %v, want not found", err, err)
		}
	}
	if _, err := repo.GetEventByIdempotencyKey(ownerB, "owner-key"); err == nil {
		t.Fatal("different owner read event by idempotency key")
	}
	otherEvents, err := repo.ListEvents(ownerB, "2026-10-04", "2026-10-06")
	if err != nil || len(otherEvents) != 0 {
		t.Fatalf("different owner list=%#v err=%v, want empty", otherEvents, err)
	}
	updated := created
	updated.Title = "unauthorized change"
	if _, err := repo.UpdateEvent(ownerB, UpdateEventInput{Event: updated, ExpectedRevision: 1}); err == nil {
		t.Fatal("different owner updated event")
	} else {
		var missing ErrEventNotFound
		if !errors.As(err, &missing) {
			t.Fatalf("cross-owner update error=%T %v, want not found before revision check", err, err)
		}
	}
	if _, err := repo.GetEvent(ownerA, "legacy-event"); err == nil {
		t.Fatal("authenticated owner saw unowned legacy event")
	}
	legacyEdit := Event{ID: "legacy-event", Title: "changed legacy", Availability: "busy", Timezone: "America/New_York", AllDay: true, StartDate: "2026-10-04", EndDateExclusive: "2026-10-05"}
	if _, err := repo.UpdateEvent(ownerA, UpdateEventInput{Event: legacyEdit, ExpectedRevision: 1}); err == nil {
		t.Fatal("authenticated owner changed unowned legacy event")
	}
	legacyCollision := CreateEventInput{IdempotencyKey: "legacy-provider-key", Event: Event{
		Title: "Legacy private event", Subject: "claimed-owner", Availability: "busy", Timezone: "America/New_York",
		AllDay: true, StartDate: "2026-10-04", EndDateExclusive: "2026-10-05", Provider: "fixture",
		ProviderCalendarID: "legacy-calendar", ProviderEventID: "legacy-provider-id",
	}}
	if _, err := db.Exec(`UPDATE calendar_events SET provider='fixture',provider_calendar_id='legacy-calendar',provider_event_id='legacy-provider-id' WHERE id='legacy-event'`); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateEvent(ownerA, legacyCollision); err == nil {
		t.Fatal("provider identity collision adopted an unowned legacy row")
	} else {
		var collision ErrEventIdentityConflict
		if !errors.As(err, &collision) {
			t.Fatalf("legacy provider collision error=%T %v, want identity conflict", err, err)
		}
	}
	var legacyTitle string
	if err := db.QueryRow(`SELECT title FROM calendar_events WHERE id='legacy-event'`).Scan(&legacyTitle); err != nil || legacyTitle != "Legacy private event" {
		t.Fatalf("legacy row changed: title=%q err=%v", legacyTitle, err)
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
	dbPath := filepath.Join(t.TempDir(), "calendar-manifest-fixtures.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLiteRepository(db, schedule.NewFake(time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC))).(EventRepository)
	want := make(map[string]Event, len(manifest.Events))
	ownerCtx := eventOwnerContext("manifest-owner")
	for i, fixture := range manifest.Events {
		availability := "busy"
		if fixture.SemanticCategory == "girlfriend_awareness" || fixture.SemanticCategory == "family_awareness" {
			availability = "free"
		}
		event := Event{Title: fixture.Title, Subject: fixture.Subject, Notes: fixture.Notes, Availability: availability,
			Timezone: fixture.Timezone, AllDay: fixture.AllDay, StartDate: fixture.StartDate, EndDateExclusive: fixture.EndDateExclusive,
			Provider: "fixture", ProviderCalendarID: fixture.GoogleCalendarID, ProviderEventID: fixture.GoogleEventID}
		created, err := repo.CreateEvent(ownerCtx, CreateEventInput{Event: event, IdempotencyKey: "manifest-fixture-" + fixture.GoogleEventID})
		if err != nil {
			t.Fatalf("fixture %d (%s): %v", i, fixture.GoogleEventID, err)
		}
		edited := created
		edited.Notes = "fixture edit readback"
		updated, err := repo.UpdateEvent(ownerCtx, UpdateEventInput{Event: edited, ExpectedRevision: created.Revision})
		if err != nil {
			t.Fatalf("edit fixture %d (%s): %v", i, fixture.GoogleEventID, err)
		}
		if updated.ID != created.ID || updated.Revision != 2 || updated.Provider != created.Provider || updated.ProviderCalendarID != created.ProviderCalendarID || updated.ProviderEventID != created.ProviderEventID || updated.StartDate != created.StartDate || updated.EndDateExclusive != created.EndDateExclusive {
			t.Fatalf("edit changed fixture identity or civil-date semantics: before=%#v after=%#v", created, updated)
		}
		want[fixture.GoogleEventID] = updated
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	// Reopen from the isolated file before checking the durable IDs and edits.
	repo = NewSQLiteRepository(db, schedule.NewFake(time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC))).(EventRepository)
	for providerEventID, expected := range want {
		got, err := repo.GetEvent(ownerCtx, expected.ID)
		if err != nil {
			t.Fatalf("read back fixture %s by durable ID %s: %v", providerEventID, expected.ID, err)
		}
		if got.ID != expected.ID || got.Revision != 2 || got.Title != expected.Title || got.Notes != "fixture edit readback" || got.ProviderEventID != providerEventID || got.StartDate != expected.StartDate || got.EndDateExclusive != expected.EndDateExclusive || got.Subject != expected.Subject || got.Availability != expected.Availability {
			t.Errorf("reopened fixture readback changed identity, revision, or semantics: %#v", got)
		}
	}
	got, err := repo.ListEvents(ownerCtx, "2026-10-01", "2026-12-02")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 21 {
		t.Fatalf("durable fixture count=%d, want 21", len(got))
	}
	for _, event := range got {
		original, ok := want[event.ProviderEventID]
		if !ok || event.ID != original.ID || event.Revision != 2 || event.Title != original.Title || event.StartDate != original.StartDate || event.EndDateExclusive != original.EndDateExclusive || event.Subject != original.Subject || event.Availability != original.Availability {
			t.Errorf("fixture identity or civil-date semantics changed: %#v", event)
		}
	}
	october31, err := repo.ListEvents(ownerCtx, "2026-10-31", "2026-10-31")
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
	november1, err := repo.ListEvents(ownerCtx, "2026-11-01", "2026-11-01")
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range november1 {
		if event.Title == "South Jersey" && (event.StartDate != "2026-11-01" || event.EndDateExclusive != "2026-11-04") {
			t.Fatalf("New York DST date-only event shifted: %#v", event)
		}
	}
}
