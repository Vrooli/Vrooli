package expiry

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"compute-manager/internal/provider"
	_ "modernc.org/sqlite"
)

func TestSweeperDestroysExpiredInstanceAndSettlesReservation(t *testing.T) { // [REQ:COMPUTEM-P0-004] [REQ:COMPUTEM-P0-006]
	db, err := sql.Open("sqlite", "file:expiry-sweeper?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE instances (id TEXT PRIMARY KEY,provider_instance_id TEXT,state TEXT,expires_at TEXT,reservation_id TEXT,created_at TEXT,destroyed_at TEXT); INSERT INTO instances VALUES ('i1','fake-1','running','2026-09-03T00:30:00Z','res-1','2026-09-03T00:00:00Z','')`)
	if err != nil {
		t.Fatal(err)
	}
	fake := &provider.Fake{Instances: map[string]provider.Instance{"fake-1": {ID: "fake-1"}}}
	var settledID string
	var settledAmount float64
	s := Sweeper{DB: db, Provider: fake, Now: func() time.Time { return time.Date(2026, 9, 3, 1, 0, 0, 0, time.UTC) }, Finalize: func(_ context.Context, id string, amount float64) error {
		settledID, settledAmount = id, amount
		return nil
	}}
	if err := s.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if fake.DestroyCalls != 1 || settledID != "res-1" || settledAmount != 60 {
		t.Fatalf("destroy/settle = %d/%s/%v, want 1/res-1/60", fake.DestroyCalls, settledID, settledAmount)
	}
	var state string
	if err := db.QueryRow(`SELECT state FROM instances WHERE id='i1'`).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "destroyed" {
		t.Fatalf("state = %s, want destroyed", state)
	}
}
