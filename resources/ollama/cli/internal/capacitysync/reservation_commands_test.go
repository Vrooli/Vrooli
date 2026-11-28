package capacitysync

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/vrooli/resources/ollama/cli/internal/ensure"
)

func TestReserveModelRightsRefusalHasNoReservationSideEffect(t *testing.T) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller did not resolve the test source")
	}
	repo := filepath.Clean(filepath.Join(filepath.Dir(source), "../../../../.."))
	reservationPath := filepath.Join(t.TempDir(), "reservation.json")
	h := handlersFor(fakePS{})
	h.ReservationFile = reservationPath
	h.GetEnv = func(key string) string {
		switch key {
		case "OLLAMA_MODEL_POLICY_PATH":
			return filepath.Join(repo, "resources", "ollama", "model-policy.json")
		case "VROOLI_MODEL_RIGHTS_CONTEXT_PATH":
			return filepath.Join(t.TempDir(), "missing-rights-context.json")
		default:
			return ""
		}
	}
	err := h.ReserveModel([]string{"--role", "chat.uncensored", "--json"})
	if err == nil || !strings.Contains(err.Error(), "rights") {
		t.Fatalf("ReserveModel error = %v, want fail-closed rights refusal", err)
	}
	if _, err := os.Stat(reservationPath); !os.IsNotExist(err) {
		t.Fatalf("reservation stat error = %v, rights refusal must precede reservation", err)
	}
}

func TestReservationCapacityRejectsNegativeAndOverflowingResidency(t *testing.T) {
	reservation := ModelReservation{Model: "target:8b", RequestedBytes: 8 << 30}
	for name, running := range map[string][]ensure.RunningModel{
		"negative value":     {{Name: "target:8b", SizeVRAM: 8 << 30}, {Name: "other:4b", SizeVRAM: -(2 << 30)}},
		"aggregate overflow": {{Name: "target:8b", SizeVRAM: int64(^uint64(0) >> 1)}, {Name: "other:4b", SizeVRAM: 1}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := reservationCapacityNeed(reservation, running); err == nil {
				t.Fatal("reservationCapacityNeed accepted invalid residency measurements")
			}
		})
	}
}

func TestConfirmModelReservationBindsTheAdmittedClaim(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reservation.json")
	store := ReservationStore{Path: path}
	reservation, err := store.Create("chat.default", "qwen3.5:9b", 8<<30, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	h := handlersFor(fakePS{})
	h.ReservationFile = path
	h.Stdout = &bytes.Buffer{}
	h.AdmitModel = func(_ context.Context, claimBytes, admissionBytes, observedBytes int64) (string, int64, bool, error) {
		if claimBytes != 8<<30 || admissionBytes != 8<<30 {
			t.Fatalf("admission claim/increment = %d/%d, want 8 GiB", claimBytes, admissionBytes)
		}
		if observedBytes != 0 {
			t.Fatalf("unexpected observed-only admission bytes = %d", observedBytes)
		}
		return "claim-1", claimBytes, true, nil
	}
	h.Exec = func(context.Context, string, ...string) ([]byte, error) {
		return []byte(`{"claims":[{"claim_id":"claim-1","owner_kind":"resource","owner_id":"ollama","resource_kind":"vram","status":"granted","amount_bytes":8589934592}]}`), nil
	}
	if err := h.ConfirmModelReservation([]string{"--id", reservation.ID, "--wait-seconds", "0", "--json"}); err != nil {
		t.Fatal(err)
	}
	confirmed, err := store.Read()
	if err != nil || confirmed == nil || confirmed.CapacityBytes != 8<<30 {
		t.Fatalf("confirmed reservation = %+v err=%v", confirmed, err)
	}
}

func TestConfirmModelReservationAdmitsBeforeStartingAStoppedResource(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reservation.json")
	store := ReservationStore{Path: path}
	reservation, err := store.Create("chat.default", "qwen3.5:9b", 8<<30, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	h := handlersFor(fakePS{err: errors.New("Ollama API is unavailable")})
	h.ReservationFile = path
	h.Stdout = &bytes.Buffer{}
	var admitted bool
	h.Exec = func(_ context.Context, name string, args ...string) ([]byte, error) {
		if len(args) >= 3 && args[0] == "resource" && args[1] == "status" {
			return []byte(`{"success":true,"running":false,"status":"stopped"}`), nil
		}
		amount := int64(0)
		if admitted {
			amount = 8 << 30
		}
		return []byte(`{"claims":[{"claim_id":"claim-1","owner_kind":"resource","owner_id":"ollama","resource_kind":"vram","status":"granted","amount_bytes":` + strconv.FormatInt(amount, 10) + `}]}`), nil
	}
	h.AdmitModel = func(_ context.Context, claimBytes, admissionBytes, observedBytes int64) (string, int64, bool, error) {
		if claimBytes != 8<<30 || admissionBytes != 8<<30 {
			t.Fatalf("pre-start admission = claim %d / admission %d, want target model estimate", claimBytes, admissionBytes)
		}
		if observedBytes != 0 {
			t.Fatalf("unexpected observed-only admission bytes = %d", observedBytes)
		}
		admitted = true
		return "claim-1", claimBytes, true, nil
	}
	if err := h.ConfirmModelReservation([]string{"--id", reservation.ID, "--wait-seconds", "0", "--json"}); err != nil {
		t.Fatal(err)
	}
	confirmed, err := store.Read()
	if err != nil || confirmed == nil || confirmed.CapacityBytes != 8<<30 {
		t.Fatalf("pre-start reservation = %+v err=%v", confirmed, err)
	}
}

func TestConfirmationPersistsAdmittedClaimBeforeFollowUpVerification(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reservation.json")
	store := ReservationStore{Path: path}
	reservation, err := store.Create("chat.default", "qwen3.5:9b", 8<<30, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	h := handlersFor(fakePS{})
	h.ReservationFile = path
	h.Stdout = &bytes.Buffer{}
	var listCalls atomic.Int32
	var released atomic.Bool
	h.Exec = func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name != "vrooli" || len(args) < 2 {
			return nil, errors.New("unexpected command")
		}
		switch {
		case args[0] == "capacity" && args[1] == "list":
			switch listCalls.Add(1) {
			case 1:
				return []byte(`{"claims":[]}`), nil
			case 2:
				return []byte(`not-json`), nil
			default:
				return []byte(`{"claims":[{"claim_id":"claim-new","owner_kind":"resource","owner_id":"ollama","resource_kind":"vram","status":"granted","amount_bytes":8589934592}]}`), nil
			}
		case args[0] == "resource" && args[1] == "status":
			return []byte(`{"success":true,"running":false,"status":"stopped"}`), nil
		case args[0] == "capacity" && args[1] == "release":
			released.Store(true)
			return []byte(`{"success":true}`), nil
		default:
			return nil, errors.New("unexpected command: " + strings.Join(args, " "))
		}
	}
	h.AdmitModel = func(context.Context, int64, int64, int64) (string, int64, bool, error) {
		return "claim-new", 8 << 30, true, nil
	}
	err = h.ConfirmModelReservation([]string{"--id", reservation.ID, "--wait-seconds", "0"})
	if err == nil || !strings.Contains(err.Error(), "decode Ollama broker capacity claims") {
		t.Fatalf("confirmation error = %v, want follow-up verification failure", err)
	}
	confirmed, err := store.Read()
	if err != nil || confirmed == nil || confirmed.ClaimID != "claim-new" || !confirmed.ReleaseClaimOnStopped {
		t.Fatalf("reservation after follow-up failure = %+v err=%v, want durable cleanup identity", confirmed, err)
	}
	if err := h.ReleaseModelReservation([]string{"--id", reservation.ID}); err != nil {
		t.Fatal(err)
	}
	if !released.Load() {
		t.Fatal("cleanup did not release the admitted claim after proving Ollama stopped")
	}
}

func TestInterruptedAdmissionCleanupDoesNotReleaseUnboundOwnerClaim(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reservation.json")
	store := ReservationStore{Path: path}
	reservation, err := store.Create("chat.default", "qwen3.5:9b", 8<<30, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	h := handlersFor(fakePS{})
	h.ReservationFile = path
	h.Stdout = &bytes.Buffer{}
	var listCalls atomic.Int32
	var released atomic.Bool
	h.Exec = func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name != "vrooli" || len(args) < 2 {
			return nil, errors.New("unexpected command")
		}
		switch {
		case args[0] == "capacity" && args[1] == "list":
			if listCalls.Add(1) < 3 {
				return []byte(`{"claims":[]}`), nil
			}
			return []byte(`{"claims":[{"claim_id":"claim-new","owner_kind":"resource","owner_id":"ollama","resource_kind":"vram","status":"granted","amount_bytes":8589934592}]}`), nil
		case args[0] == "resource" && args[1] == "status":
			return []byte(`{"success":true,"running":false,"status":"stopped"}`), nil
		case args[0] == "capacity" && args[1] == "release":
			released.Store(true)
			return []byte(`{"success":true}`), nil
		default:
			return nil, errors.New("unexpected command: " + strings.Join(args, " "))
		}
	}
	h.AdmitModel = func(context.Context, int64, int64, int64) (string, int64, bool, error) {
		// The broker committed the owner claim, then the command was canceled
		// before it could return the receipt to the reservation writer.
		return "claim-new", 8 << 30, true, errors.New("context canceled after broker commit")
	}
	err = h.ConfirmModelReservation([]string{"--id", reservation.ID, "--wait-seconds", "0"})
	if err == nil || !strings.Contains(err.Error(), "context canceled after broker commit") {
		t.Fatalf("confirmation error = %v, want simulated cancellation", err)
	}
	pending, err := store.Read()
	if err != nil || pending == nil || !pending.AdmissionPending {
		t.Fatalf("reservation after interrupted admission = %+v err=%v, want durable pending intent", pending, err)
	}
	if err := h.ReleaseModelReservation([]string{"--id", reservation.ID}); err != nil {
		t.Fatal(err)
	}
	if released.Load() {
		t.Fatal("interrupted-admission cleanup released a claim without its exact durable identity")
	}
	if current, err := store.Read(); err != nil || current != nil {
		t.Fatalf("reservation after interrupted cleanup = %+v err=%v, want marker removed", current, err)
	}
}

func TestConfirmModelReservationUsesStrictReplacementAdmissionForResidentModels(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reservation.json")
	store := ReservationStore{Path: path}
	reservation, err := store.Create("chat.default", "target:8b", 8<<30, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	h := handlersFor(fakePS{running: []ensure.RunningModel{{Name: "other:4b", SizeVRAM: 2 << 30}}})
	h.ReservationFile = path
	h.Stdout = &bytes.Buffer{}
	admitted := false
	h.Exec = func(context.Context, string, ...string) ([]byte, error) {
		amount := int64(4 << 30)
		if admitted {
			amount = 10 << 30
		}
		return []byte(`{"claims":[{"claim_id":"claim-1","owner_kind":"resource","owner_id":"ollama","resource_kind":"vram","status":"granted","amount_bytes":` + strconv.FormatInt(amount, 10) + `}]}`), nil
	}
	h.AdmitModel = func(_ context.Context, claimBytes, admissionBytes, observedBytes int64) (string, int64, bool, error) {
		if claimBytes != 10<<30 || admissionBytes != 8<<30 {
			t.Fatalf("strict replacement request = claim %d / admission %d, want aggregate 10 GiB / target 8 GiB", claimBytes, admissionBytes)
		}
		if observedBytes != 0 {
			t.Fatalf("unexpected observed-only admission bytes = %d", observedBytes)
		}
		admitted = true
		return "claim-1", claimBytes, true, nil
	}
	if err := h.ConfirmModelReservation([]string{"--id", reservation.ID, "--wait-seconds", "0", "--json"}); err != nil {
		t.Fatal(err)
	}
	confirmed, err := store.Read()
	if err != nil || confirmed == nil || confirmed.CapacityBytes != 10<<30 || confirmed.AdmissionBytes != 8<<30 {
		t.Fatalf("confirmed aggregate reservation = %+v err=%v", confirmed, err)
	}
}

func TestConfirmModelReservationReusesCoveredAlreadyLoadedTarget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reservation.json")
	store := ReservationStore{Path: path}
	reservation, err := store.Create("chat.default", "target:8b", 8<<30, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	h := handlersFor(fakePS{running: []ensure.RunningModel{
		{Name: "other:4b", SizeVRAM: 2 << 30},
		{Name: "target:8b", SizeVRAM: 8 << 30},
	}})
	h.ReservationFile = path
	h.Stdout = &bytes.Buffer{}
	h.Exec = func(context.Context, string, ...string) ([]byte, error) {
		return []byte(`{"claims":[{"claim_id":"claim-1","owner_kind":"resource","owner_id":"ollama","resource_kind":"vram","status":"granted","amount_bytes":10737418240}]}`), nil
	}
	h.AdmitModel = func(context.Context, int64, int64, int64) (string, int64, bool, error) {
		t.Fatal("already-loaded model should reuse its covered claim without incremental admission")
		return "", 0, false, nil
	}
	if err := h.ConfirmModelReservation([]string{"--id", reservation.ID, "--wait-seconds", "0", "--json"}); err != nil {
		t.Fatal(err)
	}
	confirmed, err := store.Read()
	if err != nil || confirmed == nil || confirmed.CapacityBytes != 10<<30 || confirmed.AdmissionBytes != 0 || confirmed.ClaimID != "claim-1" {
		t.Fatalf("already-loaded target reservation = %+v err=%v", confirmed, err)
	}
}

func TestConfirmModelReservationCorrectsAnUnderClaimedResidentFromGPUAttribution(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reservation.json")
	store := ReservationStore{Path: path}
	reservation, err := store.Create("chat.default", "target:8b", 8<<30, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	h := handlersFor(fakePS{running: []ensure.RunningModel{
		{Name: "other:4b", SizeVRAM: 2 << 30},
		{Name: "target:8b", SizeVRAM: 8 << 30},
	}})
	h.ReservationFile = path
	h.Stdout = &bytes.Buffer{}
	var admitted atomic.Bool
	h.Exec = func(context.Context, string, ...string) ([]byte, error) {
		amount := int64(8 << 30)
		if admitted.Load() {
			amount = 10 << 30
		}
		return []byte(`{"claims":[{"claim_id":"claim-1","owner_kind":"resource","owner_id":"ollama","resource_kind":"vram","status":"granted","amount_bytes":` + strconv.FormatInt(amount, 10) + `}]}`), nil
	}
	h.AdmitModel = func(_ context.Context, claimBytes, admissionBytes, observedBytes int64) (string, int64, bool, error) {
		if claimBytes != 10<<30 || admissionBytes != 0 || observedBytes != 10<<30 {
			t.Fatalf("observed-resident correction = claim %d / incremental %d / verified %d, want 10 / 0 / 10 GiB", claimBytes, admissionBytes, observedBytes)
		}
		admitted.Store(true)
		return "claim-1", 10 << 30, false, nil
	}
	if err := h.ConfirmModelReservation([]string{"--id", reservation.ID, "--wait-seconds", "0", "--json"}); err != nil {
		t.Fatal(err)
	}
	confirmed, err := store.Read()
	if err != nil || confirmed == nil || confirmed.CapacityBytes != 10<<30 || confirmed.AdmissionBytes != 0 || confirmed.ReleaseClaimOnStopped {
		t.Fatalf("corrected resident reservation = %+v err=%v", confirmed, err)
	}
	h.NewClient = func() psClient { return fakePS{} }
	h.Exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if len(args) >= 2 && args[0] == "capacity" && args[1] == "release" {
			t.Fatal("cleanup released a pre-existing claim after the corrected model unloaded")
		}
		return []byte(`{"claims":[{"claim_id":"claim-1","owner_kind":"resource","owner_id":"ollama","resource_kind":"vram","status":"granted","amount_bytes":10737418240}]}`), nil
	}
	if err := h.ReleaseModelReservation([]string{"--id", reservation.ID, "--json"}); err != nil {
		t.Fatal(err)
	}
}

func TestConfirmationWaitsForTheCompanionReconciliationLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reservation.json")
	store := ReservationStore{Path: path}
	reservation, err := store.Create("chat.default", "qwen3.5:9b", 8<<30, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	h := handlersFor(fakePS{})
	h.ReservationFile = path
	h.Stdout = &bytes.Buffer{}
	var queried atomic.Bool
	h.Exec = func(context.Context, string, ...string) ([]byte, error) {
		queried.Store(true)
		return []byte(`{"claims":[{"claim_id":"claim-1","owner_kind":"resource","owner_id":"ollama","resource_kind":"vram","status":"granted","amount_bytes":8589934592}]}`), nil
	}
	h.AdmitModel = func(_ context.Context, claimBytes, admissionBytes, observedBytes int64) (string, int64, bool, error) {
		if claimBytes != 8<<30 || admissionBytes != 8<<30 {
			t.Fatalf("admission claim/increment = %d/%d, want 8 GiB", claimBytes, admissionBytes)
		}
		if observedBytes != 0 {
			t.Fatalf("unexpected observed-only admission bytes = %d", observedBytes)
		}
		return "claim-1", claimBytes, true, nil
	}
	unlock, err := h.LockSync(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		done <- h.ConfirmModelReservation([]string{"--id", reservation.ID, "--wait-seconds", "1"})
	}()
	time.Sleep(30 * time.Millisecond)
	if queried.Load() {
		unlock()
		t.Fatal("confirmation read capacity while companion reconciliation held the lock")
	}
	unlock()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("confirmation did not continue after the companion lock was released")
	}
}

func TestConfirmationRevalidatesReservationAfterWaitingForSyncLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reservation.json")
	base := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	var now atomic.Int64
	now.Store(base.UnixNano())
	clock := func() time.Time { return time.Unix(0, now.Load()).UTC() }
	store := ReservationStore{Path: path, Now: clock}
	reservation, err := store.Create("chat.default", "qwen3.5:9b", 8<<30, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	h := handlersFor(fakePS{})
	h.ReservationFile = path
	h.Now = clock
	var admitted atomic.Bool
	h.AdmitModel = func(context.Context, int64, int64, int64) (string, int64, bool, error) {
		admitted.Store(true)
		return "claim-1", 8 << 30, true, nil
	}
	h.Exec = func(context.Context, string, ...string) ([]byte, error) {
		return []byte(`{"claims":[{"claim_id":"claim-1","owner_kind":"resource","owner_id":"ollama","resource_kind":"vram","status":"granted","amount_bytes":8589934592}]}`), nil
	}
	unlock, err := store.LockSync(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		done <- h.ConfirmModelReservation([]string{"--id", reservation.ID, "--wait-seconds", "1"})
	}()
	time.Sleep(30 * time.Millisecond)
	now.Store(base.Add(2 * time.Minute).UnixNano())
	unlock()
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "changed or expired") {
			t.Fatalf("confirmation error = %v, want expired reservation refusal", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("confirmation did not return after the sync lock was released")
	}
	if admitted.Load() {
		t.Fatal("expired reservation was admitted after waiting for the sync lock")
	}
}

func TestConfirmModelReservationRefusesAnUnderSizedClaim(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reservation.json")
	store := ReservationStore{Path: path}
	reservation, err := store.Create("chat.default", "qwen3.5:9b", 8<<30, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	h := handlersFor(fakePS{})
	h.ReservationFile = path
	h.Exec = func(context.Context, string, ...string) ([]byte, error) {
		return []byte(`{"claims":[{"claim_id":"claim-1","owner_kind":"resource","owner_id":"ollama","resource_kind":"vram","status":"granted","amount_bytes":4294967296}]}`), nil
	}
	h.AdmitModel = func(context.Context, int64, int64, int64) (string, int64, bool, error) {
		return "", 0, false, errors.New("insufficient no-reclaim capacity")
	}
	err = h.ConfirmModelReservation([]string{"--id", reservation.ID, "--wait-seconds", "0"})
	if err == nil || !strings.Contains(err.Error(), "insufficient no-reclaim capacity") {
		t.Fatalf("ConfirmModelReservation error = %v, want under-sized claim refusal", err)
	}
	if current, err := store.Read(); err != nil || current == nil || current.ID != reservation.ID {
		t.Fatalf("reservation after refused handoff = %+v err=%v, want retained", current, err)
	}
}

func TestReleaseModelReservationRetainsMarkerUntilResidencyIsCovered(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reservation.json")
	store := ReservationStore{Path: path}
	reservation, err := store.Create("chat.default", "target:8b", 8<<30, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	h := handlersFor(fakePS{running: []ensure.RunningModel{{Name: "target:8b", SizeVRAM: 8 << 30}}})
	h.ReservationFile = path
	h.Exec = func(context.Context, string, ...string) ([]byte, error) {
		return []byte(`{"claims":[{"claim_id":"claim-1","owner_kind":"resource","owner_id":"ollama","resource_kind":"vram","status":"granted","amount_bytes":4294967296}]}`), nil
	}
	err = h.ReleaseModelReservation([]string{"--id", reservation.ID})
	if err == nil || !strings.Contains(err.Error(), "below model reservation") {
		t.Fatalf("ReleaseModelReservation error = %v, want coverage refusal", err)
	}
	if current, err := store.Read(); err != nil || current == nil || current.ID != reservation.ID {
		t.Fatalf("reservation after refused release = %+v err=%v, want retained", current, err)
	}
}

func TestReleaseModelReservationChecksOtherModelsWhenTargetIsAbsent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reservation.json")
	store := ReservationStore{Path: path}
	reservation, err := store.Create("chat.default", "target:8b", 8<<30, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	h := handlersFor(fakePS{running: []ensure.RunningModel{{Name: "other:12b", SizeVRAM: 10 << 30}}})
	h.ReservationFile = path
	h.Exec = func(context.Context, string, ...string) ([]byte, error) {
		return []byte(`{"claims":[{"claim_id":"claim-1","owner_kind":"resource","owner_id":"ollama","resource_kind":"vram","status":"granted","amount_bytes":8589934592}]}`), nil
	}
	err = h.ReleaseModelReservation([]string{"--id", reservation.ID})
	if err == nil || !strings.Contains(err.Error(), "current model footprint") {
		t.Fatalf("ReleaseModelReservation error = %v, want uncovered other-model refusal", err)
	}
	if current, err := store.Read(); err != nil || current == nil || current.ID != reservation.ID {
		t.Fatalf("reservation after refused release = %+v err=%v, want retained", current, err)
	}
}

func TestReleaseModelReservationReleasesNewClaimWhenRuntimeIsStopped(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reservation.json")
	store := ReservationStore{Path: path}
	reservation, err := store.Create("chat.default", "target:8b", 8<<30, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Confirm(reservation.ID, "claim-1", 8<<30, 8<<30, true); err != nil {
		t.Fatal(err)
	}
	h := handlersFor(fakePS{err: errors.New("Ollama API is unavailable")})
	h.ReservationFile = path
	h.Stdout = &bytes.Buffer{}
	var released atomic.Bool
	h.Exec = func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name != "vrooli" || len(args) < 2 {
			return nil, errors.New("unexpected command")
		}
		switch {
		case args[0] == "resource" && args[1] == "status":
			return []byte(`{"success":true,"running":false,"status":"stopped"}`), nil
		case args[0] == "capacity" && args[1] == "list":
			return []byte(`{"claims":[{"claim_id":"claim-1","owner_kind":"resource","owner_id":"ollama","resource_kind":"vram","status":"granted","amount_bytes":8589934592}]}`), nil
		case args[0] == "capacity" && args[1] == "release":
			released.Store(true)
			return []byte(`{"success":true}`), nil
		default:
			return nil, errors.New("unexpected command: " + strings.Join(args, " "))
		}
	}
	if err := h.ReleaseModelReservation([]string{"--id", reservation.ID, "--json"}); err != nil {
		t.Fatal(err)
	}
	if !released.Load() {
		t.Fatal("stopped runtime cleanup did not release its newly created capacity claim")
	}
	if current, err := store.Read(); err != nil || current != nil {
		t.Fatalf("reservation after stopped cleanup = %+v err=%v, want removed", current, err)
	}
}

func TestReleaseModelReservationRetainsMarkerWhenStoppedStateIsUnproven(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reservation.json")
	store := ReservationStore{Path: path}
	reservation, err := store.Create("chat.default", "target:8b", 8<<30, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Confirm(reservation.ID, "claim-1", 8<<30, 8<<30, true); err != nil {
		t.Fatal(err)
	}
	h := handlersFor(fakePS{err: errors.New("Ollama API is unavailable")})
	h.ReservationFile = path
	h.Exec = func(context.Context, string, ...string) ([]byte, error) {
		return []byte(`{"success":false,"status":"unknown"}`), nil
	}
	err = h.ReleaseModelReservation([]string{"--id", reservation.ID})
	if err == nil || !strings.Contains(err.Error(), "did not prove") {
		t.Fatalf("release error = %v, want refusal without proof of stopped runtime", err)
	}
	if current, err := store.Read(); err != nil || current == nil || current.ID != reservation.ID {
		t.Fatalf("reservation after unproven stopped state = %+v err=%v, want retained", current, err)
	}
}
