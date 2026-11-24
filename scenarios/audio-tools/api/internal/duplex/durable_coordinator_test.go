package duplex

import (
	"context"
	"errors"
	"testing"
)

type revokedDurableAdmission struct {
	durableAdmissionFixture
	rechecks int
}

func (revokedDurableAdmission) RequiresActiveConsentRevalidation(Request) bool { return true }
func (a *revokedDurableAdmission) RevalidateConsent(context.Context, Request) error {
	a.rechecks++
	return ErrConsentInvalid
}

type durableReservationFixture struct {
	dispatched bool
	beginErr   error
	authorized bool
	settled    []ProviderUsage
	retained   int
	released   int
}

func (r *durableReservationFixture) BeginDispatch(context.Context) (bool, error) {
	if r.beginErr != nil {
		return false, r.beginErr
	}
	if !r.authorized {
		return false, nil
	}
	r.dispatched = true
	return true, nil
}

func (r *durableReservationFixture) SettleProviderUsage(_ context.Context, u ProviderUsage) error {
	if !r.dispatched {
		return errors.New("settled before dispatch")
	}
	r.settled = append(r.settled, u)
	return nil
}

func (r *durableReservationFixture) RetainUncertain(context.Context) error {
	if !r.dispatched {
		return errors.New("retained before dispatch")
	}
	r.retained++
	return nil
}

func (r *durableReservationFixture) Settle(uint64) error {
	return errors.New("provider details required")
}
func (r *durableReservationFixture) Release() { r.released++ }

type durableAdmissionFixture struct{ reservation *durableReservationFixture }

func (a durableAdmissionFixture) Authorize(context.Context, Request) error { return nil }
func (a durableAdmissionFixture) Reserve(context.Context, Request) (Reservation, error) {
	return a.reservation, nil
}

type durableConnectionFixture struct {
	*fakeConnection
	usage     ProviderUsage
	confirmed bool
}

func (c durableConnectionFixture) FinalProviderUsage() (ProviderUsage, bool) {
	return c.usage, c.confirmed
}

type durableProviderFixture struct {
	connection Connection
	err        error
	opened     bool
}

func (p *durableProviderFixture) Open(context.Context, Request, Sink) (Connection, error) {
	p.opened = true
	return p.connection, p.err
}

func TestCoordinatorRequiresDispatchAndRetainsAmbiguousTerminalUsage(t *testing.T) {
	t.Run("revocation after dispatch prevents provider contact", func(t *testing.T) {
		reservation := &durableReservationFixture{authorized: true}
		admission := &revokedDurableAdmission{durableAdmissionFixture{reservation}, 0}
		provider := &durableProviderFixture{}
		coordinator, err := NewCoordinator(provider, admission)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := coordinator.Open(context.Background(), Request{}); !errors.Is(err, ErrConsentInvalid) {
			t.Fatalf("open error=%v, want consent invalid", err)
		}
		if provider.opened || admission.rechecks != 1 || reservation.retained != 1 || reservation.released != 0 {
			t.Fatalf("provider=%v rechecks=%d uncertain=%d released=%d", provider.opened, admission.rechecks, reservation.retained, reservation.released)
		}
	})

	t.Run("dispatch denied", func(t *testing.T) {
		reservation := &durableReservationFixture{}
		provider := &durableProviderFixture{}
		coordinator, err := NewCoordinator(provider, durableAdmissionFixture{reservation})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := coordinator.Open(context.Background(), Request{}); err == nil {
			t.Fatal("provider opened without dispatch authorization")
		}
		if provider.opened || reservation.released != 0 {
			t.Fatalf("provider=%v release=%d", provider.opened, reservation.released)
		}
	})
	t.Run("terminal provider receipt settles", func(t *testing.T) {
		reservation := &durableReservationFixture{authorized: true}
		provider := &durableProviderFixture{connection: durableConnectionFixture{fakeConnection: &fakeConnection{}, usage: ProviderUsage{ProviderReceiptID: "live-session-7", Model: "gpt-live-1.1", Provider: "openai-gpt-live", ActiveSeconds: 3, CostMicros: 12}, confirmed: true}}
		coordinator, err := NewCoordinator(provider, durableAdmissionFixture{reservation})
		if err != nil {
			t.Fatal(err)
		}
		handle, err := coordinator.Open(context.Background(), Request{})
		if err != nil {
			t.Fatal(err)
		}
		if err := handle.Close("done"); err != nil {
			t.Fatal(err)
		}
		if len(reservation.settled) != 1 || reservation.settled[0].ProviderReceiptID != "live-session-7" || reservation.retained != 0 || reservation.released != 0 {
			t.Fatalf("settled=%+v uncertain=%d released=%d", reservation.settled, reservation.retained, reservation.released)
		}
	})
	t.Run("missing terminal record retained", func(t *testing.T) {
		reservation := &durableReservationFixture{authorized: true}
		provider := &durableProviderFixture{connection: durableConnectionFixture{fakeConnection: &fakeConnection{}}}
		coordinator, err := NewCoordinator(provider, durableAdmissionFixture{reservation})
		if err != nil {
			t.Fatal(err)
		}
		handle, err := coordinator.Open(context.Background(), Request{})
		if err != nil {
			t.Fatal(err)
		}
		_ = handle.Close("connection-lost")
		if len(reservation.settled) != 0 || reservation.retained != 1 || reservation.released != 0 {
			t.Fatalf("settled=%+v uncertain=%d released=%d", reservation.settled, reservation.retained, reservation.released)
		}
	})
	t.Run("startup ambiguity retained", func(t *testing.T) {
		reservation := &durableReservationFixture{authorized: true}
		provider := &durableProviderFixture{err: errors.New("provider startup failed")}
		coordinator, err := NewCoordinator(provider, durableAdmissionFixture{reservation})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := coordinator.Open(context.Background(), Request{}); err == nil {
			t.Fatal("startup error lost")
		}
		if reservation.retained != 1 || reservation.released != 0 {
			t.Fatalf("uncertain=%d released=%d", reservation.retained, reservation.released)
		}
	})
}

var _ Connection = (*fakeConnection)(nil)
