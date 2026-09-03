package provision

import (
	"context"
	"errors"
	"testing"
	"time"

	"compute-manager/internal/intent"
	"compute-manager/internal/meter"
	"compute-manager/internal/provider"
)

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
