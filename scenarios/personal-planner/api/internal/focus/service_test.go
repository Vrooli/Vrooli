package focus

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/vrooli/api-core/schedule"
	_ "modernc.org/sqlite"
)

type fakeRepository struct {
	current     *Session
	created     Session
	actuals     []Actual
	corrections []Correction
	notes       []SessionNote
	pauses      []PauseEvent
}

func (r *fakeRepository) CreateActual(_ context.Context, actual Actual) (Actual, error) {
	actual.ID = "actual-1"
	r.actuals = append(r.actuals, actual)
	return actual, nil
}

func (r *fakeRepository) ListActuals(_ context.Context, localDate string) ([]Actual, error) {
	var result []Actual
	for _, actual := range r.actuals {
		if localDate == "" || actual.LocalDate == localDate {
			result = append(result, actual)
		}
	}
	return result, nil
}

func (r *fakeRepository) ListCorrections(_ context.Context, actualID, localDate string, limit int) ([]Correction, error) {
	result := make([]Correction, 0, len(r.corrections))
	for _, correction := range r.corrections {
		if (actualID == "" || correction.ActualID == actualID) && limit > 0 && len(result) < limit {
			result = append(result, correction)
		}
	}
	return result, nil
}

func (r *fakeRepository) CorrectActual(_ context.Context, update Actual, expectedRevision int64) (Actual, error) {
	for i := range r.actuals {
		if r.actuals[i].ID == update.ID {
			if r.actuals[i].Revision != expectedRevision {
				return Actual{}, ErrRevisionConflict{update.ID}
			}
			previous := r.actuals[i]
			r.actuals[i].ReportedMinutes = update.ReportedMinutes
			r.actuals[i].Certainty = update.Certainty
			r.actuals[i].Note = update.Note
			r.actuals[i].Revision++
			r.corrections = append(r.corrections, Correction{ID: "correction-1", ActualID: update.ID, PreviousMinutes: previous.ReportedMinutes, NewMinutes: update.ReportedMinutes, PreviousCertainty: previous.Certainty, NewCertainty: update.Certainty, Reason: update.Note})
			return r.actuals[i], nil
		}
	}
	return Actual{}, ErrActualNotFound{update.ID}
}

func (r *fakeRepository) Create(_ context.Context, session Session) (Session, error) {
	if session.ID == "" {
		session.ID = "session-1"
	}
	r.created = session
	r.current = &session
	return session, nil
}

func (r *fakeRepository) Current(context.Context) (Session, bool, error) {
	if r.current == nil || (r.current.State != StateRunning && r.current.State != StatePaused) {
		return Session{}, false, nil
	}
	return *r.current, true, nil
}

func (r *fakeRepository) Get(_ context.Context, id string) (Session, error) {
	if r.current == nil || r.current.ID != id {
		return Session{}, ErrSessionNotFound{id}
	}
	return *r.current, nil
}

func (r *fakeRepository) Transition(_ context.Context, session Session, revision int64) (Session, error) {
	if r.current == nil || r.current.Revision != revision {
		return Session{}, ErrRevisionConflict{session.ID}
	}
	session.Revision++
	r.current = &session
	return session, nil
}

func (r *fakeRepository) SaveNote(_ context.Context, note SessionNote) (SessionNote, error) {
	r.notes = append(r.notes, note)
	return note, nil
}

func (r *fakeRepository) ListNotes(_ context.Context, localDate string) ([]SessionNote, error) {
	result := []SessionNote{}
	for _, note := range r.notes {
		if localDate == "" || note.LocalDate == localDate {
			result = append(result, note)
		}
	}
	return result, nil
}

func (r *fakeRepository) RecordPauseEvent(_ context.Context, event PauseEvent) (PauseEvent, error) {
	if event.ID == "" {
		event.ID = "pause-1"
	}
	r.pauses = append(r.pauses, event)
	return event, nil
}

func (r *fakeRepository) ListPauseEvents(_ context.Context, localDate string) ([]PauseEvent, error) {
	result := []PauseEvent{}
	for _, event := range r.pauses {
		if localDate == "" || event.LocalDate == localDate {
			result = append(result, event)
		}
	}
	return result, nil
}

func TestServiceFocusLifecycleSeparatesActiveAndWallTime(t *testing.T) {
	clock := schedule.NewFake(time.Date(2026, 9, 19, 9, 0, 0, 0, time.UTC))
	repo := &fakeRepository{}
	service := NewService(repo, clock)

	session, err := service.Start(context.Background(), StartInput{Title: "Draft the launch story", Mode: ModeOpen})
	if err != nil {
		t.Fatal(err)
	}
	if session.State != StateRunning || session.Revision != 1 {
		t.Fatalf("unexpected start: %#v", session)
	}
	clock.Advance(30 * time.Minute)
	session, err = service.Pause(context.Background(), session.ID, session.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if session.ActiveSeconds != 1800 || session.WallSeconds != 1800 || session.State != StatePaused {
		t.Fatalf("unexpected pause: %#v", session)
	}
	clock.Advance(15 * time.Minute)
	session, err = service.Resume(context.Background(), session.ID, session.Revision)
	if err != nil {
		t.Fatal(err)
	}
	clock.Advance(20 * time.Minute)
	session, err = service.End(context.Background(), session.ID, session.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if session.ActiveSeconds != 3000 || session.WallSeconds != 3900 || session.State != StateEnded {
		t.Fatalf("unexpected end: %#v", session)
	}
	if _, ok, err := service.Current(context.Background()); err != nil || ok {
		t.Fatalf("ended session remained current: ok=%v err=%v", ok, err)
	}
	note, err := service.SaveNote(context.Background(), session.ID, "Ship the smallest useful version next.")
	if err != nil || note.LocalDate != "2026-09-19" {
		t.Fatalf("unexpected session note: %#v err=%v", note, err)
	}
	if got, err := service.ListNotes(context.Background(), "2026-09-19"); err != nil || len(got) != 1 || got[0].Note == "" {
		t.Fatalf("unexpected notes: %#v err=%v", got, err)
	}
}

func TestServiceRecordsPauseReasonAfterPause(t *testing.T) {
	clock := schedule.NewFake(time.Date(2026, 9, 19, 9, 0, 0, 0, time.UTC))
	repo := &fakeRepository{}
	service := NewService(repo, clock)
	session, err := service.Start(context.Background(), StartInput{Title: "Draft"})
	if err != nil {
		t.Fatal(err)
	}
	paused, err := service.Pause(context.Background(), session.ID, session.Revision)
	if err != nil {
		t.Fatal(err)
	}
	event, err := service.RecordPauseEvent(context.Background(), paused.ID, PauseInterrupted)
	if err != nil || event.Reason != PauseInterrupted || event.LocalDate != "2026-09-19" {
		t.Fatalf("unexpected pause event: %#v err=%v", event, err)
	}
}

func TestServiceRejectsSecondCurrentSessionAndInvalidTransitions(t *testing.T) {
	clock := schedule.NewFake(time.Unix(0, 0))
	repo := &fakeRepository{}
	service := NewService(repo, clock)
	first, err := service.Start(context.Background(), StartInput{Title: "First"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Start(context.Background(), StartInput{Title: "Second"}); err == nil {
		t.Fatal("expected exclusive-session conflict")
	}
	if _, err := service.Resume(context.Background(), first.ID, first.Revision); err == nil {
		t.Fatal("expected invalid resume transition")
	}
	if _, err := service.Pause(context.Background(), first.ID, first.Revision+1); err == nil {
		t.Fatal("expected revision conflict")
	}
}

func TestServiceRecordsAndCorrectsApproximateActual(t *testing.T) {
	clock := schedule.NewFake(time.Date(2026, 9, 19, 9, 0, 0, 0, time.UTC))
	repo := &fakeRepository{}
	service := NewService(repo, clock)
	actual, err := service.RecordActual(context.Background(), RecordActualInput{Title: "Review launch brief", LocalDate: "2026-09-19", ReportedMinutes: 45})
	if err != nil {
		t.Fatal(err)
	}
	if actual.Certainty != "user_reported_approximate" || actual.ReportedMinutes != 45 {
		t.Fatalf("unexpected actual: %#v", actual)
	}
	corrected, err := service.CorrectActual(context.Background(), CorrectActualInput{ID: actual.ID, ExpectedRevision: actual.Revision, ReportedMinutes: 30, Certainty: "timed_observed", Note: "Removed the interruption."})
	if err != nil {
		t.Fatal(err)
	}
	if corrected.ReportedMinutes != 30 || corrected.Revision != 2 || corrected.Note == "" {
		t.Fatalf("unexpected correction: %#v", corrected)
	}
	history, err := service.ListCorrections(context.Background(), actual.ID, "2026-09-19", 10)
	if err != nil || len(history) != 1 || history[0].PreviousMinutes != 45 || history[0].NewMinutes != 30 {
		t.Fatalf("unexpected correction history: %#v err=%v", history, err)
	}
}

func TestSQLiteActualCorrectionRetainsAuditAndEffectiveValue(t *testing.T) {
	db, err := sql.Open("sqlite", "file:focus-actuals-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE calendar_allocations (id TEXT PRIMARY KEY, work_item_id TEXT, local_date TEXT, start_minutes INTEGER, state TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO calendar_allocations(id, work_item_id, local_date, start_minutes, state) VALUES ('allocation-1', 'work-1', '2026-09-19', 540, 'accepted')`); err != nil {
		t.Fatal(err)
	}
	clock := schedule.NewFake(time.Date(2026, 9, 19, 9, 0, 0, 0, time.UTC))
	service := NewService(NewSQLiteRepository(db, clock), clock)
	actual, err := service.RecordActual(context.Background(), RecordActualInput{WorkItemID: "work-1", Title: "Review launch brief", LocalDate: "2026-09-19", ReportedMinutes: 45})
	if err != nil {
		t.Fatal(err)
	}
	if actual.AllocationID != "allocation-1" {
		t.Fatalf("expected automatic plan link, got %q", actual.AllocationID)
	}
	if _, err := service.CorrectActual(context.Background(), CorrectActualInput{ID: actual.ID, ExpectedRevision: actual.Revision, ReportedMinutes: 30, Note: "Removed interruption"}); err != nil {
		t.Fatal(err)
	}
	var effective, auditCount int64
	if err := db.QueryRow(`SELECT reported_minutes FROM manual_actuals WHERE id = ?`, actual.ID).Scan(&effective); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM actual_corrections WHERE actual_id = ?`, actual.ID).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if effective != 30 || auditCount != 1 {
		t.Fatalf("expected effective correction and one audit row, got minutes=%d audits=%d", effective, auditCount)
	}
	history, err := service.ListCorrections(context.Background(), actual.ID, "2026-09-19", 10)
	if err != nil || len(history) != 1 || history[0].PreviousMinutes != 45 || history[0].NewMinutes != 30 || history[0].Reason != "Removed interruption" {
		t.Fatalf("unexpected persisted correction history: %#v err=%v", history, err)
	}
}
