package provision

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"compute-manager/internal/intent"
	"compute-manager/internal/meter"
	"compute-manager/internal/provider"
	_ "modernc.org/sqlite"
)

type ledgerDB struct{ queries []string }

func (d *ledgerDB) ExecContext(_ context.Context, query string, _ ...any) (sql.Result, error) {
	d.queries = append(d.queries, query)
	return nil, nil
}

func (*ledgerDB) QueryContext(context.Context, string, ...any) (*sql.Rows, error) { return nil, nil }

type store struct {
	records map[string]intent.Record
	order   *[]string
}

func (s *store) Create(_ context.Context, r intent.Record) (intent.Record, error) {
	if r.ID == "" {
		r.ID = r.IdempotencyKey
	}
	s.records[r.IdempotencyKey] = r
	*s.order = append(*s.order, "intent")
	return r, nil
}
func (s *store) GetByIDOrKey(_ context.Context, key string) (intent.Record, error) {
	r, ok := s.records[key]
	if !ok {
		return intent.Record{}, intent.ErrNotFound
	}
	return r, nil
}
func (s *store) Update(_ context.Context, r intent.Record) error {
	s.records[r.IdempotencyKey] = r
	return nil
}

type credits struct {
	order    *[]string
	refuse   bool
	released bool
}

func (c *credits) ReserveCredits(context.Context, string, string, float64, time.Duration) (meter.Reservation, error) {
	*c.order = append(*c.order, "reserve")
	if c.refuse {
		return meter.Reservation{}, meter.ErrInsufficientCredits
	}
	return meter.Reservation{ID: "r1"}, nil
}
func (c *credits) ReleaseReservation(context.Context, string) error         { c.released = true; return nil }
func (*credits) FinalizeReservation(context.Context, string, float64) error { return nil }

type orderedProvider struct {
	provider.Fake
	order *[]string
	fail  bool
}

func (p *orderedProvider) Create(ctx context.Context, s provider.Spec) (provider.Instance, error) {
	*p.order = append(*p.order, "provider")
	if p.fail {
		return provider.Instance{}, errors.New("provider unavailable")
	}
	return p.Fake.Create(ctx, s)
}

func TestCreateOrdersIntentReservationProvider(t *testing.T) {
	order := []string{}
	st := &store{records: map[string]intent.Record{}, order: &order}
	cr := &credits{order: &order}
	p := &orderedProvider{order: &order}
	svc := Service{Intents: intent.Service{Store: st, Provider: p}, Meter: meter.Service{Credits: cr}, Provider: p, Window: time.Hour}
	_, _, err := svc.Create(context.Background(), intent.Request{IdempotencyKey: "one", RequestedBy: "owner", Spec: provider.Spec{Region: "r", Size: "s"}}, 60)
	if err != nil {
		t.Fatal(err)
	}
	if got := order; len(got) != 3 || got[0] != "intent" || got[1] != "reserve" || got[2] != "provider" {
		t.Fatalf("order = %v", got)
	}
}

func TestCreateReleasesHoldWhenProviderFails(t *testing.T) {
	order := []string{}
	st := &store{records: map[string]intent.Record{}, order: &order}
	cr := &credits{order: &order}
	p := &orderedProvider{order: &order, fail: true}
	svc := Service{Intents: intent.Service{Store: st, Provider: p}, Meter: meter.Service{Credits: cr}, Provider: p, Window: time.Hour}
	_, _, err := svc.Create(context.Background(), intent.Request{IdempotencyKey: "two", RequestedBy: "owner", Spec: provider.Spec{Region: "r", Size: "s"}}, 60)
	if err == nil || !cr.released {
		t.Fatalf("err=%v released=%t", err, cr.released)
	}
}

func TestCreatePersistsHeldReservationAndLinksInstance(t *testing.T) { // [REQ:COMPUTEM-P0-006]
	order := []string{}
	db := &ledgerDB{}
	st := &store{records: map[string]intent.Record{}, order: &order}
	cr := &credits{order: &order}
	p := &orderedProvider{order: &order}
	svc := Service{Intents: intent.Service{Store: st, Provider: p}, Meter: meter.Service{Credits: cr}, Provider: p, Window: time.Hour, DB: db}
	_, created, err := svc.Create(context.Background(), intent.Request{IdempotencyKey: "ledger", RequestedBy: "owner", Spec: provider.Spec{Region: "r", Size: "s"}}, 60)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || len(db.queries) != 2 || !strings.Contains(db.queries[0], "INSERT INTO reservations") || !strings.Contains(db.queries[1], "UPDATE reservations") {
		t.Fatalf("created=%+v queries=%v", created, db.queries)
	}
}

type renewingCredits struct {
	order []string
	next  int
}

func (c *renewingCredits) ReserveCredits(context.Context, string, string, float64, time.Duration) (meter.Reservation, error) {
	c.next++
	id := fmt.Sprintf("next-%d", c.next)
	c.order = append(c.order, "reserve:"+id)
	return meter.Reservation{ID: id}, nil
}
func (c *renewingCredits) ReleaseReservation(_ context.Context, id string) error {
	c.order = append(c.order, "release:"+id)
	return nil
}
func (*renewingCredits) FinalizeReservation(context.Context, string, float64) error { return nil }

func TestRenewReservationsHoldsSuccessorBeforeReleasingPredecessor(t *testing.T) { // [REQ:COMPUTEM-P0-006]
	db, err := sql.Open("sqlite", "file:renew-reservations?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE instances (id TEXT PRIMARY KEY,tenant TEXT,reservation_id TEXT,state TEXT); CREATE TABLE reservations (id TEXT PRIMARY KEY,instance_id TEXT,supersedes TEXT,meter_key TEXT,state TEXT,held_at TEXT,settled_at TEXT,quantity INTEGER); INSERT INTO instances VALUES ('i1','owner','old','running'); INSERT INTO reservations VALUES ('old','i1','','compute_minutes','held','2026-09-04T00:00:00Z','',60);`)
	if err != nil {
		t.Fatal(err)
	}
	credits := &renewingCredits{}
	if err := (Service{DB: db, Meter: meter.Service{Credits: credits}, Window: time.Hour}).RenewReservations(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(credits.order, ","), "reserve:next-1,release:old"; got != want {
		t.Fatalf("credit order = %q, want %q", got, want)
	}
	var supersedes, state string
	if err := db.QueryRow(`SELECT supersedes,state FROM reservations WHERE id='next-1'`).Scan(&supersedes, &state); err != nil {
		t.Fatal(err)
	}
	if supersedes != "old" || state != "held" {
		t.Fatalf("successor = %q/%q, want old/held", supersedes, state)
	}
	var current string
	if err := db.QueryRow(`SELECT reservation_id FROM instances WHERE id='i1'`).Scan(&current); err != nil || current != "next-1" {
		t.Fatalf("instance reservation = %q, %v", current, err)
	}
}
