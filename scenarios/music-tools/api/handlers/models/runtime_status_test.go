package models

import (
	"github.com/gorilla/mux"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	domain "music-tools/internal/models"
)

func TestUnprobedDeclaredModelDoesNotClaimMissingInstallation(t *testing.T) {
	actual := wire(domain.Model{ID: "stabilityai/stable-audio-3-medium", Installed: false, Licence: domain.LicenceRestricted})
	if actual.Status != "runtime_unverified" || actual.LicenseLane != "restricted" {
		t.Fatalf("unprobed declaration falsely classified: %+v", actual)
	}
}

func TestSongModelCapabilitiesArePublicAndDoNotClaimInstallation(t *testing.T) {
	registry, err := domain.LoadSeed(false)
	if err != nil {
		t.Fatal(err)
	}
	for _, model := range registry.List() {
		if model.Runtime == nil {
			continue
		}
		actual := wire(model)
		if actual.Status != "runtime_unverified" || actual.SungLyrics == nil || !actual.SungLyrics.Supported || actual.SungLyrics.Source == "" || actual.VoiceReference.Supported || actual.CompositionControls == nil || actual.CompositionControls.MaxDurationSeconds != model.Runtime.MaxDuration {
			t.Fatalf("missing or misleading model capabilities: %+v", actual)
		}
	}
}

func TestModelLookupAcceptsCanonicalNamespacedIDs(t *testing.T) {
	registry, err := domain.LoadSeed(false)
	if err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	Module(registry).Mount(router)
	for _, path := range []string{"/api/v1/models/MiniMaxAI/MiniMax-Music3", "/api/v1/models/HeartMuLa%2FHeartMuLa-oss-3B-happy-new-year"} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "sung_lyrics") {
			t.Fatalf("namespaced lookup %s: %d %s", path, recorder.Code, recorder.Body.String())
		}
	}
}
