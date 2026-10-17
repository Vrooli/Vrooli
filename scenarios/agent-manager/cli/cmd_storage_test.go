package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

type storageFixture struct {
	mu       sync.Mutex
	requests []string
	bodies   []map[string]any
	auth     []string
	polls    int
}

func (f *storageFixture) server(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.requests = append(f.requests, r.Method+" "+r.URL.Path)
		f.auth = append(f.auth, r.Header.Get("Authorization"))
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.bodies = append(f.bodies, body)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/storage/health":
			f.polls++
			state := "running"
			if f.polls > 1 {
				state = "succeeded"
			}
			_, _ = w.Write([]byte(`{"storage":{"fileBytes":10,"liveBytes":8,"freeBytes":2,"autoVacuum":"incremental","level":"ok","action":"none"},"compaction":{"state":"` + state + `","receipt":{"before":{"fileBytes":100},"after":{"fileBytes":10},"reclaimedBytes":90,"autoVacuumAfter":"incremental","quickCheck":"ok"}}}`))
		case "/api/v1/storage/reclaim":
			_, _ = w.Write([]byte(`{"dryRun":true,"action":"incremental_vacuum","before":{"freeBytes":2},"complete":false}`))
		case "/api/v1/storage/compact":
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"state":"running"}`))
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func storageApp(server *httptest.Server) *App {
	api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{DefaultBase: server.URL} }, func() string { return "owner-fixture" })
	return &App{services: NewServices(api)}
}

func TestStorageCLIUsesTheOwnerContract(t *testing.T) {
	previous := storagePollInterval
	storagePollInterval = time.Millisecond
	t.Cleanup(func() { storagePollInterval = previous })
	fixture := &storageFixture{}
	app := storageApp(fixture.server(t))

	for _, args := range [][]string{{"status", "--json"}, {"reclaim", "--dry-run", "--json"}, {"compact", "--reason", "planned compaction"}} {
		if err := app.cmdStorage(args); err != nil {
			t.Fatal(args, err)
		}
	}

	want := []string{"GET /api/v1/storage/health", "POST /api/v1/storage/reclaim", "POST /api/v1/storage/compact", "GET /api/v1/storage/health"}
	if strings.Join(fixture.requests, "\n") != strings.Join(want, "\n") {
		t.Fatalf("requests=%v", fixture.requests)
	}
	if fixture.bodies[1]["dry_run"] != true {
		t.Fatalf("reclaim lost dry_run: %v", fixture.bodies[1])
	}
	if fixture.bodies[2]["reason"] != "planned compaction" || fixture.auth[2] != "Bearer owner-fixture" {
		t.Fatalf("compact lost reason or owner credential: %v %q", fixture.bodies[2], fixture.auth[2])
	}
}

func TestStorageCompactValidatesBeforeAnyRequest(t *testing.T) {
	fixture := &storageFixture{}
	app := storageApp(fixture.server(t))
	t.Setenv(cliutil.EnvIdentityToken, "identified-agent-fixture")

	if err := app.cmdStorage([]string{"compact"}); err == nil || !strings.Contains(err.Error(), "--reason") {
		t.Fatalf("reasonless compact: %v", err)
	}
	if err := app.cmdStorage([]string{"compact", "--reason", "x", "--local-owner"}); err == nil || !strings.Contains(err.Error(), "identified agent") {
		t.Fatalf("agent elevation: %v", err)
	}
	if err := app.cmdStorage([]string{"defragment"}); err == nil {
		t.Fatal("unknown subcommand accepted")
	}
	if len(fixture.requests) != 0 {
		t.Fatalf("invalid commands reached the API: %v", fixture.requests)
	}
}
