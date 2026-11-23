package eventbus

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestDiscoveredPublisherRecoversEndpointAndRetainsExactEventIntent(t *testing.T) {
	var receivedMu sync.Mutex
	var receivedIDs []string
	newOwner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/events" {
			http.Error(w, "unsupported route", http.StatusNotFound)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var envelope struct {
			EventID string `json:"eventId"`
		}
		if err := json.Unmarshal(body, &envelope); err != nil || envelope.EventID == "" {
			t.Errorf("event envelope=%s err=%v", body, err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		receivedMu.Lock()
		receivedIDs = append(receivedIDs, envelope.EventID)
		receivedMu.Unlock()
		w.WriteHeader(http.StatusAccepted)
	}))
	defer newOwner.Close()

	oldOwner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	oldURL := oldOwner.URL
	var resolveCalls int
	client := newDiscoveredClient("", func() string {
		resolveCalls++
		switch resolveCalls {
		case 1:
			return "" // Registry does not yet have a healthy owner.
		case 2:
			return oldURL
		default:
			return newOwner.URL
		}
	})
	if client.Enabled() {
		t.Fatal("initial missing Events route unexpectedly resolved")
	}
	if !client.Enabled() || resolveCalls != 2 {
		t.Fatalf("publisher did not recover after initial discovery miss: enabled=%v resolutions=%d", client.Enabled(), resolveCalls)
	}

	intent := DomainEvent{
		Source: "vrooli-onboarding", EventType: "onboarding.profile.saved.v1",
		Payload:  map[string]any{"profile_id": "profile-test", "revision": 7},
		Occurred: time.Date(2026, 11, 21, 21, 0, 0, 0, time.UTC),
	}
	seed, err := json.Marshal(intent)
	if err != nil {
		t.Fatal(err)
	}
	expectedID := digestBytes(seed)
	intentPath := filepath.Join(t.TempDir(), "pending-event.json")
	if err := os.WriteFile(intentPath, seed, 0600); err != nil {
		t.Fatal(err)
	}

	oldOwner.Close()
	intentBytes, err := os.ReadFile(intentPath)
	if err != nil {
		t.Fatal(err)
	}
	var pending DomainEvent
	if err := json.Unmarshal(intentBytes, &pending); err != nil {
		t.Fatal(err)
	}
	if err := client.PublishDomainEvent(context.Background(), pending); err == nil {
		t.Fatal("stale discovered endpoint failure was hidden")
	}
	retained, err := os.ReadFile(intentPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(retained) != string(seed) {
		t.Fatal("failed publish changed retained event intent")
	}

	if err := client.PublishDomainEvent(context.Background(), pending); err != nil {
		t.Fatalf("same retained event did not publish after endpoint recovery: %v", err)
	}
	if resolveCalls != 3 {
		t.Fatalf("recovered endpoint resolutions=%d want 3", resolveCalls)
	}
	receivedMu.Lock()
	defer receivedMu.Unlock()
	if len(receivedIDs) != 1 {
		t.Fatalf("recovered owner received %d events: %v", len(receivedIDs), receivedIDs)
	}
	if receivedIDs[0] != expectedID {
		t.Fatalf("recovered event identity=%q want %q", receivedIDs[0], expectedID)
	}
}

func TestDiscoveredClientKeepsExplicitEndpointPinned(t *testing.T) {
	called := false
	client := newDiscoveredClient(" http://events.example.test ", func() string {
		called = true
		return "http://registry.example.test"
	})
	if got := client.baseURL(); got != "http://events.example.test" || called {
		t.Fatalf("explicit endpoint=%q resolver-called=%v", got, called)
	}
	client.invalidateEndpoint("http://events.example.test")
	if got := client.baseURL(); got != "http://events.example.test" || called {
		t.Fatalf("explicit endpoint changed after invalidation: %q resolver-called=%v", got, called)
	}
}
