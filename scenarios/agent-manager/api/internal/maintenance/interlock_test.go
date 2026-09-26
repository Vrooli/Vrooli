package maintenance

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/api-core/identity"
	platform "github.com/vrooli/platform-go"
)

func TestMaintenanceRootLockAllowsGETButRefusesResumeBeforeGateMutex(t *testing.T) {
	gate, _ := openGate(t, filepath.Join(t.TempDir(), "gate.db"))
	state, err := gate.Enter(t.Context(), "owner", "rollout")
	if err != nil {
		t.Fatal(err)
	}
	lock := testInterlock(t)
	// An independent descriptor uses exactly the control plane's native lock.
	if err := os.MkdirAll(filepath.Dir(lock.path), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(lock.path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	unlock, err := platform.LockFile(f, true)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	h, err := NewHandler(gate, emptyInventory, ownerVerifier{identity.Principal{Verified: true, Kind: identity.ActorHuman, Subject: "owner", Scopes: []string{"*"}}}, lock)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, AdmissionPath, nil))
	var standing Standing
	if err := json.Unmarshal(response.Body.Bytes(), &standing); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || !standing.Drained || !standing.Closed || standing.Revision != state.Revision || standing.LifecycleInterlock != ScenarioLockV1 || standing.Inventory == nil || standing.Inventory.Remaining == nil || *standing.Inventory.Remaining != 0 {
		t.Fatalf("root proof=%d %s", response.Code, response.Body.String())
	}
	// If resume takes gate.mu first this request deadlocks. The test timeout
	// bounds that regression; no actual lifecycle action or shared lock is used.
	gate.mu.Lock()
	request := httptest.NewRequest(http.MethodPost, AdmissionPath+"/resume", strings.NewReader(`{"revision":1}`))
	request.Header.Set("Authorization", "Bearer owner")
	response = httptest.NewRecorder()
	h.ServeHTTP(response, request)
	gate.mu.Unlock()
	if response.Code != http.StatusConflict {
		t.Fatalf("resume=%d %s", response.Code, response.Body.String())
	}
	current, err := gate.Status(t.Context())
	if err != nil || !current.Closed || current.Revision != state.Revision {
		t.Fatalf("resume changed fence: %+v %v", current, err)
	}
}

type interlockCheckingStore struct {
	Store
	lock *ScenarioInterlock
	t    *testing.T
}

func (s interlockCheckingStore) Save(ctx context.Context, state State, revision int64) error {
	if !state.Closed {
		f, err := os.OpenFile(s.lock.path, os.O_RDWR, 0o644)
		if err != nil {
			return err
		}
		defer f.Close()
		unlock, err := platform.LockFile(f, true)
		if err == nil {
			unlock()
			s.t.Error("resume persistence occurred without root lock")
		}
		if !errors.Is(err, platform.ErrLockUnavailable) {
			s.t.Errorf("lock contention=%v", err)
		}
	}
	return s.Store.Save(ctx, state, revision)
}

func TestMaintenanceResumeHoldsRootLockThroughPersistence(t *testing.T) {
	gate, _ := openGate(t, filepath.Join(t.TempDir(), "gate.db"))
	lock := testInterlock(t)
	gate.store = interlockCheckingStore{Store: gate.store, lock: lock, t: t}
	if _, err := gate.Enter(t.Context(), "owner", "rollout"); err != nil {
		t.Fatal(err)
	}
	h, err := NewHandler(gate, emptyInventory, ownerVerifier{identity.Principal{Verified: true, Kind: identity.ActorHuman, Subject: "owner", Scopes: []string{"*"}}}, lock)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, AdmissionPath+"/resume", strings.NewReader(`{"revision":1}`))
	request.Header.Set("Authorization", "Bearer owner")
	response := httptest.NewRecorder()
	h.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("resume=%d %s", response.Code, response.Body.String())
	}
	release, err := lock.acquire(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	release()
}

func TestMaintenanceGETCannotDrainAcrossRevisionChangeOrUnknownInventory(t *testing.T) {
	for _, unknown := range []bool{false, true} {
		gate, _ := openGate(t, filepath.Join(t.TempDir(), "gate.db"))
		state, err := gate.Enter(t.Context(), "owner", "rollout")
		if err != nil {
			t.Fatal(err)
		}
		h, err := NewHandler(gate, func(ctx context.Context) (Inventory, error) {
			inv, _ := emptyInventory(ctx)
			if unknown {
				inv.Unknown = []string{"descendant exclusion unavailable"}
				return inv, nil
			}
			if _, err := gate.Resume(ctx, "owner", state.Revision); err != nil {
				t.Fatal(err)
			}
			if _, err := gate.Enter(ctx, "owner", "second rollout"); err != nil {
				t.Fatal(err)
			}
			return inv, nil
		}, ownerVerifier{}, testInterlock(t))
		if err != nil {
			t.Fatal(err)
		}
		response := httptest.NewRecorder()
		h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, AdmissionPath, nil))
		if response.Code == http.StatusOK || strings.Contains(response.Body.String(), `"drained":true`) {
			t.Fatalf("false proof=%d %s", response.Code, response.Body.String())
		}
	}
}
