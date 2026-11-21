package meter

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

type fakeCredits struct{ called bool }

func (f *fakeCredits) ReserveCredits(context.Context, string, string, float64, time.Duration) (Reservation, error) {
	f.called = true
	return Reservation{}, ErrInsufficientCredits
}

func TestCheckCeilingUsesHeldReservations(t *testing.T) { // [REQ:COMPUTEM-P1-002]
	db, err := sql.Open("sqlite", "file:meter-ceiling?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE instance_intents (id TEXT PRIMARY KEY,requested_by TEXT); CREATE TABLE reservations (id TEXT PRIMARY KEY,intent_id TEXT,state TEXT,quantity INTEGER); INSERT INTO instance_intents VALUES ('intent-1','owner'); INSERT INTO reservations VALUES ('reservation-1','intent-1','held',50);`)
	if err != nil {
		t.Fatal(err)
	}
	svc := Service{DB: db, TenantCeilingMinutes: 60}
	if err := svc.CheckCeiling(context.Background(), "owner", 11); !errors.Is(err, ErrCeilingExceeded) {
		t.Fatalf("error = %v, want ceiling exceeded", err)
	}
	if err := svc.CheckCeiling(context.Background(), "owner", 9); err != nil {
		t.Fatalf("under-ceiling request = %v", err)
	}
}

func TestAcquireCeilingHoldSerializesConcurrentProcesses(t *testing.T) { // [REQ:COMPUTEM-P1-002]
	db, err := sql.Open("sqlite", "file:meter-ceiling-concurrent?mode=memory&cache=shared&_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE tenant_ceiling_holds (id TEXT PRIMARY KEY,tenant TEXT NOT NULL,quantity INTEGER NOT NULL,expires_at TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	svc := Service{DB: db, TenantCeilingMinutes: 60, Now: func() time.Time { return time.Date(2026, 9, 4, 3, 0, 0, 0, time.UTC) }}
	start := make(chan struct{})
	type result struct {
		id  string
		err error
	}
	results := make(chan result, 2)
	for range 2 {
		go func() {
			<-start
			id, err := svc.AcquireCeilingHold(context.Background(), "tenant", 40, time.Hour)
			results <- result{id: id, err: err}
		}()
	}
	close(start)
	first, second := <-results, <-results
	if (first.err == nil) == (second.err == nil) {
		t.Fatalf("concurrent results = %v, %v; want exactly one ceiling hold", first.err, second.err)
	}
	if first.err == nil {
		_ = svc.ReleaseCeilingHold(context.Background(), first.id)
	}
	if second.err == nil {
		_ = svc.ReleaseCeilingHold(context.Background(), second.id)
	}
}
func (*fakeCredits) ReleaseReservation(context.Context, string) error           { return nil }
func (*fakeCredits) FinalizeReservation(context.Context, string, float64) error { return nil }

func TestReserveReturnsTypedRefusal(t *testing.T) { // [REQ:COMPUTEM-P0-006]
	f := &fakeCredits{}
	_, err := (Service{Credits: f}).Reserve(context.Background(), "owner", 60, time.Hour)
	if !errors.Is(err, ErrInsufficientCredits) {
		t.Fatalf("error = %v, want insufficient credits", err)
	}
	if !f.called {
		t.Fatal("reservation service was not called")
	}
}
