package meter

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeCredits struct{ called bool }

func (f *fakeCredits) ReserveCredits(context.Context, string, string, float64, time.Duration) (Reservation, error) {
	f.called = true
	return Reservation{}, ErrInsufficientCredits
}
func (*fakeCredits) ReleaseReservation(context.Context, string) error           { return nil }
func (*fakeCredits) FinalizeReservation(context.Context, string, float64) error { return nil }

func TestReserveReturnsTypedRefusal(t *testing.T) {
	f := &fakeCredits{}
	_, err := (Service{Credits: f}).Reserve(context.Background(), "owner", 60, time.Hour)
	if !errors.Is(err, ErrInsufficientCredits) {
		t.Fatalf("error = %v, want insufficient credits", err)
	}
	if !f.called {
		t.Fatal("reservation service was not called")
	}
}
