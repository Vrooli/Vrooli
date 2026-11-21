//go:build linux

package lightingruntime

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/vrooli/api-core/lightingbinding"
	"github.com/vrooli/vrooli/internal/process"
)

func TestNativeScenarioOwnerUsesExactGenerationAndLostACKRetry(t *testing.T) {
	identity := lightingbinding.Identity{Scenario: "lighting-control", Variant: "live", RuntimeInstanceID: "fixture", Generation: "generation-one", PID: os.Getpid()}
	var mu sync.Mutex
	state := lightingbinding.State{Version: lightingbinding.Version, Identity: identity}
	var effects, calls int
	var applied lightingbinding.Request
	var receipt lightingbinding.Receipt
	socket := nativeFixtureSocket(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode(state)
			return
		}
		var request lightingbinding.Request
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		calls++
		if request.Identity != identity {
			t.Error("request targeted wrong generation")
		}
		if calls == 1 {
			applied = request
			effects++
			state.Revision++
			state.Binding = request.Binding
			receipt = lightingbinding.Receipt{State: state, OperationID: request.OperationID}
			// Close after committing without delivering an HTTP acknowledgment.
			conn, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Error(err)
				return
			}
			_ = conn.Close()
			return
		}
		if request.OperationID != applied.OperationID || request.ExpectedRevision != applied.ExpectedRevision {
			t.Error("lost acknowledgment retried with a different operation")
		}
		_ = json.NewEncoder(w).Encode(receipt)
	}))
	var starts, verified, closed int
	owner := &nativeScenarioOwner{ensureStarted: func(context.Context) error { starts++; return nil }, resolve: func() (*nativeScenarioTarget, error) {
		return &nativeScenarioTarget{identity: identity, socket: socket, verify: func() error { verified++; return nil }, close: func() { closed++ }}, nil
	}}
	backend := Backend{Endpoint: netip.MustParseAddrPort("127.0.0.1:6742"), Incarnation: "owned-backend"}
	if err := owner.EnsureLightingReconciler(context.Background(), backend); err != nil {
		t.Fatal(err)
	}
	if effects != 1 || calls != 2 || starts != 1 || verified < 3 || closed != 1 {
		t.Fatalf("effects=%d calls=%d starts=%d verified=%d close=%d", effects, calls, starts, verified, closed)
	}
	// A reconstructed root owner reads current binding and does no duplicate
	// SDK work. Its authority still comes from the fresh pinned resolver.
	if err := owner.EnsureLightingReconciler(context.Background(), backend); err != nil {
		t.Fatal(err)
	}
	if effects != 1 || calls != 2 {
		t.Fatal("recovery repeated effective policy")
	}
}

func TestNativeScenarioOwnerRejectsReplacementAndDoesNotStartOnQuiesce(t *testing.T) {
	identity := lightingbinding.Identity{Scenario: "lighting-control", Variant: "live", RuntimeInstanceID: "fixture", Generation: "one", PID: os.Getpid()}
	var mutations int
	socket := nativeFixtureSocket(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			mutations++
		}
		_ = json.NewEncoder(w).Encode(lightingbinding.State{Version: lightingbinding.Version, Identity: identity})
	}))
	owner := &nativeScenarioOwner{ensureStarted: func(context.Context) error { t.Fatal("quiesce started a scenario"); return nil }, resolve: func() (*nativeScenarioTarget, error) {
		return &nativeScenarioTarget{identity: identity, socket: socket, verify: func() error { return errors.New("fixture owner replaced") }, close: func() {}}, nil
	}}
	if err := owner.QuiesceLightingReconciler(context.Background()); err == nil || mutations != 0 {
		t.Fatal("replacement admitted mutation")
	}
	owner.resolve = func() (*nativeScenarioTarget, error) { return nil, errNoLightingScenario }
	if err := owner.QuiesceLightingReconciler(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestNativeScenarioOwnerRejectsWrongIdentityBeforeMutation(t *testing.T) {
	identity := lightingbinding.Identity{Scenario: "lighting-control", Variant: "live", RuntimeInstanceID: "fixture", Generation: "one", PID: os.Getpid()}
	wrong := identity
	wrong.Generation = "previous"
	socket := nativeFixtureSocket(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Error("foreign generation received mutation")
		}
		_ = json.NewEncoder(w).Encode(lightingbinding.State{Version: lightingbinding.Version, Identity: wrong})
	}))
	owner := &nativeScenarioOwner{resolve: func() (*nativeScenarioTarget, error) {
		return &nativeScenarioTarget{identity: identity, socket: socket, verify: func() error { return nil }, close: func() {}}, nil
	}}
	if err := owner.QuiesceLightingReconciler(context.Background()); err == nil {
		t.Fatal("foreign generation accepted")
	}
}

func TestSelectLightingRecordRejectsAmbiguityAndPreservesUncertainty(t *testing.T) {
	if _, err := selectLightingRecord(nil); !errors.Is(err, errNoLightingScenario) {
		t.Fatal(err)
	}
	if _, err := selectLightingRecord([]process.Record{{Step: "start-api"}, {Step: "start-api"}}); err == nil || errors.Is(err, errNoLightingScenario) {
		t.Fatal("ambiguous owner became absence")
	}
	got, err := selectLightingRecord([]process.Record{{Step: "start-api", Status: process.RecordStatusExitedVerified}, {Step: "start-api", PID: 7, Generation: "current"}})
	if err != nil || got.PID != 7 {
		t.Fatalf("current owner %#v %v", got, err)
	}
	if _, err := NewNativeScenarioOwner(t.TempDir(), nil); err == nil {
		t.Fatal("missing lifecycle owner accepted")
	}
}

func nativeFixtureSocket(t *testing.T, handler http.Handler) string {
	t.Helper()
	directory, err := os.MkdirTemp("", "lighting-owner-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(directory) })
	socket := filepath.Join(directory, "fixture.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: handler}
	done := make(chan struct{})
	go func() { defer close(done); _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close(); <-done })
	return socket
}
