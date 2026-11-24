package capacitysync

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/vrooli/resources/ollama/cli/internal/ensure"
)

func TestReservationStoreCreateConfirmAndRemove(t *testing.T) {
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	store := ReservationStore{Path: filepath.Join(t.TempDir(), "reservation.json"), Now: func() time.Time { return now }}
	reservation, err := store.Create("chat.default", "model:8b", 10<<30, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if reservation.ID == "" || reservation.RequestedBytes != 10<<30 || !reservation.ExpiresAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("reservation = %+v", reservation)
	}
	if _, err := store.Create("chat.other", "model:4b", 4<<30, time.Hour); err == nil {
		t.Fatal("concurrent reservation should be refused")
	}
	confirmed, err := store.Confirm(reservation.ID, "claim-1", 12<<30, 8<<30, true)
	if err != nil || confirmed.ClaimID != "claim-1" || confirmed.CapacityBytes != 12<<30 || confirmed.AdmissionBytes != 8<<30 || !confirmed.ReleaseClaimOnStopped {
		t.Fatalf("confirmed = %+v err=%v", confirmed, err)
	}
	if err := store.Remove("wrong-id"); err == nil {
		t.Fatal("wrong reservation id should not remove the active marker")
	}
	if err := store.Remove(reservation.ID); err != nil {
		t.Fatal(err)
	}
	if current, err := store.Read(); err != nil || current != nil {
		t.Fatalf("reservation after removal = %+v err=%v", current, err)
	}
}

func TestReservationStoreExpiresAndAllowsNextReservation(t *testing.T) {
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	store := ReservationStore{Path: filepath.Join(t.TempDir(), "reservation.json"), Now: func() time.Time { return now }}
	first, err := store.Create("chat.default", "model:8b", 10<<30, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Minute)
	if current, err := store.Read(); err != nil || current != nil {
		t.Fatalf("expired reservation = %+v err=%v", current, err)
	}
	second, err := store.Create("chat.other", "model:4b", 4<<30, time.Minute)
	if err != nil || second.ID == first.ID {
		t.Fatalf("second reservation = %+v err=%v", second, err)
	}
}

func TestConcurrentReservationCreationHasOneWinner(t *testing.T) {
	store := ReservationStore{Path: filepath.Join(t.TempDir(), "reservation.json")}
	const attempts = 12
	var wg sync.WaitGroup
	var mu sync.Mutex
	winners := 0
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := store.Create("chat.default", "model:8b", int64(8+i)<<30, time.Hour)
			if err == nil {
				mu.Lock()
				winners++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	if winners != 1 {
		t.Fatalf("successful concurrent reservations = %d, want exactly one", winners)
	}
	if current, err := store.Read(); err != nil || current == nil {
		t.Fatalf("active reservation = %+v err=%v", current, err)
	}
}

func TestObserveKeepsReservationClaimedUntilModelResidency(t *testing.T) {
	reservation := ModelReservation{
		ID: "r1", Role: "chat.default", Model: "target:8b", RequestedBytes: 10 << 30,
		CapacityBytes: 12 << 30, ExpiresAt: time.Now().Add(time.Hour),
	}
	path := filepath.Join(t.TempDir(), "reservation.json")
	store := ReservationStore{Path: path}
	if err := store.write(reservation); err != nil {
		t.Fatal(err)
	}
	client := fakePS{running: []ensure.RunningModel{{Name: "other:4b", SizeVRAM: 2 << 30}}}
	h := handlersFor(client)
	h.ReservationFile = path
	footprint, err := h.Observe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if footprint.Bytes != 12<<30 || footprint.Activity != "active" {
		t.Fatalf("footprint while target is not loaded = %+v", footprint)
	}
	if !footprint.AdmissionRequired {
		t.Fatal("pending model reservation must require broker admission")
	}
	h = handlersFor(fakePS{running: []ensure.RunningModel{
		{Name: "other:4b", SizeVRAM: 2 << 30},
		{Name: "target:8b", SizeVRAM: 10 << 30},
	}})
	h.ReservationFile = path
	footprint, err = h.Observe(context.Background())
	if err != nil || footprint.Bytes != 12<<30 || footprint.Activity != "active" {
		t.Fatalf("footprint after target load = %+v err=%v", footprint, err)
	}
}

func TestObserveReservationReadFailureNeverLooksLikeZero(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reservation.json")
	if err := os.WriteFile(path, []byte("not-json"), 0o600); err != nil {
		t.Fatal(err)
	}
	h := handlersFor(fakePS{})
	h.ReservationFile = path
	if _, err := h.Observe(context.Background()); err == nil {
		t.Fatal("malformed reservation should preserve accounting instead of reporting zero")
	}
}

func TestUnconfirmedReservationAddsTargetToExistingResidentFootprint(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reservation.json")
	store := ReservationStore{Path: path}
	reservation, err := store.Create("chat.default", "target:8b", 10<<30, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	h := handlersFor(fakePS{running: []ensure.RunningModel{{Name: "other:4b", SizeVRAM: 2 << 30}}})
	h.ReservationFile = path
	footprint, err := h.Observe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if want := int64(12 << 30); footprint.Bytes != want || footprint.ClaimPreferredBytes != want || footprint.ClaimFloorBytes != want {
		t.Fatalf("unconfirmed footprint = %+v, want an exact aggregate claim of %d bytes for %s", footprint, want, reservation.Model)
	}
	if !footprint.AdmissionRequired || footprint.ObservedBytes != 2<<30 {
		t.Fatalf("unconfirmed admission state = %+v", footprint)
	}
}

func TestConfirmedReservationNeverLetsOtherResidentModelsFallOutOfClaim(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reservation.json")
	store := ReservationStore{Path: path}
	reservation := ModelReservation{ID: "r1", Role: "chat.default", Model: "target:8b", RequestedBytes: 10 << 30, CapacityBytes: 12 << 30, ExpiresAt: time.Now().Add(time.Hour)}
	if err := store.write(reservation); err != nil {
		t.Fatal(err)
	}
	h := handlersFor(fakePS{running: []ensure.RunningModel{
		{Name: "other:4b", SizeVRAM: 2 << 30},
		{Name: "target:8b", SizeVRAM: 12 << 30},
	}})
	h.ReservationFile = path
	footprint, err := h.Observe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if footprint.Bytes != 14<<30 || footprint.ObservedBytes != 14<<30 || !footprint.AdmissionRequired || footprint.ClaimPreferredBytes != 14<<30 {
		t.Fatalf("observed over-estimate footprint = %+v, want a guarded 14 GiB aggregate", footprint)
	}
}
