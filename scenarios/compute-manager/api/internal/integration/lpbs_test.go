package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLPBSCreditsUsesReservationLifecycle(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/usage/reservations" {
			_, _ = w.Write([]byte(`{"reservation_id":"res-1"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	c := &LPBSCredits{BaseURL: server.URL}
	reservation, err := c.ReserveCredits(context.Background(), "owner", "compute_minutes", 60, time.Hour)
	if err != nil || reservation.ID != "res-1" {
		t.Fatalf("reserve = %+v, %v", reservation, err)
	}
	if err := c.FinalizeReservation(context.Background(), reservation.ID, 42); err != nil {
		t.Fatal(err)
	}
	if err := c.ReleaseReservation(context.Background(), reservation.ID); err != nil {
		t.Fatal(err)
	}
	if len(paths) != 3 || paths[1] != "/api/v1/usage/reservations/res-1/finalize" || paths[2] != "/api/v1/usage/reservations/res-1/release" {
		t.Fatalf("paths = %#v", paths)
	}
}
