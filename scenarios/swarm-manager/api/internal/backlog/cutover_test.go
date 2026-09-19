package backlog

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestLegacyWorkshopRoutesAreNotRegistered(t *testing.T) {
	h := NewHandler(t.TempDir(), t.TempDir())
	router := mux.NewRouter()
	h.RegisterRoutes(router)

	for _, target := range []string{
		"/api/v1/backlog/idea/example/research",
		"/api/v1/backlog/idea/example/workshop/reset",
		"/api/v1/backlog/idea/example/workshop/round",
		"/api/v1/backlog/idea/example/workshop/clarification/thread/action",
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, target, nil))
		if response.Code != http.StatusNotFound {
			t.Fatalf("legacy route %s returned %d; want 404", target, response.Code)
		}
	}
}

func TestImmutableBacklogSnapshotVersionChangesWithItem(t *testing.T) {
	item := BacklogItem{Name: "example", Title: "Before", Kind: KindIdea}
	before := immutableBacklogSnapshotVersion(item)
	item.Title = "After"
	if before == immutableBacklogSnapshotVersion(item) {
		t.Fatal("snapshot version did not change after item mutation")
	}
}
