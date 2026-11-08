package isolation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestShouldRetainFromEnv(t *testing.T) {
	t.Setenv("TEST_GENIE_PLAYBOOKS_RETAIN", "1")
	if !ShouldRetainFromEnv() {
		t.Fatal("expected retain flag to honor environment override")
	}

	t.Setenv("TEST_GENIE_PLAYBOOKS_RETAIN", "0")
	if ShouldRetainFromEnv() {
		t.Fatal("expected retain flag to be false when env is not 1")
	}
}

func TestNewManagerAppliesDefaultTimeout(t *testing.T) {
	manager := NewManager(Config{})
	if manager.cfg.Timeout != 90*time.Second {
		t.Fatalf("expected default timeout of 90s, got %s", manager.cfg.Timeout)
	}
}

func TestBuildDBNameSanitizesAndTruncates(t *testing.T) {
	name := buildDBName("My Scenario!", strings.Repeat("run-id-", 20))
	if !strings.HasPrefix(name, "tg_pb_") {
		t.Fatalf("expected database name prefix, got %q", name)
	}
	if len(name) > 60 {
		t.Fatalf("expected database name to be truncated to 60 chars, got %d", len(name))
	}
	if strings.ContainsAny(name, " !") {
		t.Fatalf("expected database name to be sanitized, got %q", name)
	}
}

func TestMergeAndSanitize(t *testing.T) {
	merged := merge(map[string]string{"A": "1", "B": "2"}, map[string]string{"B": "override", "C": "3"})
	if merged["A"] != "1" || merged["B"] != "override" || merged["C"] != "3" {
		t.Fatalf("unexpected merged env map: %#v", merged)
	}

	if got := sanitize(" Test Genie/Playbooks "); got != "test_genie_playbooks" {
		t.Fatalf("expected sanitize to normalize punctuation and whitespace, got %q", got)
	}
}

func TestStartSQLiteProvisioning(t *testing.T) {
	manager := NewManager(Config{
		// A display name, not a slug: the manager must normalize it before
		// handing it to the storage seam, which requires a scenario identifier.
		ScenarioName:  "Test Genie",
		RequireSQLite: true,
		SQLiteEnvVars: []string{"APP_SQLITE_PATH"},
	})

	result, cleanup, err := manager.startSQLite(context.Background(), "run-id")
	if err != nil {
		t.Fatalf("startSQLite returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected sqlite start result")
	}

	// Isolation is expressed as a storage ROOT, not a database path. The root
	// is scenario-agnostic, so every scenario beneath it still resolves its own
	// separate file — which is why a leaked value isolates instead of colliding.
	root := result.env["VROOLI_STORAGE_ROOT"]
	if root == "" {
		t.Fatalf("expected an isolated storage root: %#v", result.env)
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("expected the isolated storage root to exist: %v", err)
	}

	// A scenario-scoped variable is still supplied: its name carries its owner,
	// so it cannot capture a sibling's database.
	scoped := result.env["APP_SQLITE_PATH"]
	if scoped == "" {
		t.Fatalf("expected the declared scenario-scoped variable to be set: %#v", result.env)
	}
	if !strings.HasPrefix(scoped, root) {
		t.Fatalf("declared path %q is not inside the isolated root %q", scoped, root)
	}
	if filepath.Ext(scoped) != ".db" {
		t.Fatalf("expected a .db file, got %s", scoped)
	}
	if scoped != result.info.Endpoint {
		t.Fatalf("the reported endpoint %q disagrees with the provisioned path %q", result.info.Endpoint, scoped)
	}

	// The generic pair must NOT be injected. Injecting it is what let a single
	// value redirect every scenario that inherited it.
	for _, key := range []string{"SQLITE_PATH", "SQLITE_DB"} {
		if v, ok := result.env[key]; ok {
			t.Fatalf("%s must not be injected (got %q): a generic database path is "+
				"inherited by every child process and redirects siblings", key, v)
		}
	}

	if err := cleanup(context.Background()); err != nil {
		t.Fatalf("cleanup returned error: %v", err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("expected the isolated storage root to be removed, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Dir(root)); !os.IsNotExist(err) {
		t.Fatalf("standalone private owner namespace was not retired: %v", err)
	}
	if err := cleanup(context.Background()); err != nil {
		t.Fatalf("standalone cleanup was not idempotent: %v", err)
	}
}

func TestSQLiteNameCollisionDoesNotAcquireOrDeleteForeignBacking(t *testing.T) {
	parent := t.TempDir()
	runID := "31c832dd-4761-4f80-aa9c-d43da34173df"
	foreign := filepath.Join(parent, "test-genie-playbooks-"+runID)
	if err := os.Mkdir(foreign, 0700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(foreign, "keep")
	if err := os.WriteFile(marker, []byte("foreign"), 0600); err != nil {
		t.Fatal(err)
	}
	allocations := 0
	manager := NewManager(Config{ScenarioName: "demo", RequireSQLite: true, RunID: runID, AllocationRoot: parent, OnAllocation: func(Allocation) error { allocations++; return nil }})
	result, err := manager.Prepare(t.Context())
	if err == nil || result == nil {
		t.Fatal("foreign allocation collision was not refused")
	}
	if err := result.Cleanup(t.Context()); err != nil {
		t.Fatal(err)
	}
	if allocations != 0 {
		t.Fatal("foreign backing acquired cleanup authority")
	}
	if raw, err := os.ReadFile(marker); err != nil || string(raw) != "foreign" {
		t.Fatal("foreign backing was changed")
	}
	if err := RetireAllocation(t.Context(), Allocation{Kind: "sqlite", RunID: runID, Root: foreign}); err == nil {
		t.Fatal("missing exact owner marker permitted foreign removal")
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("foreign backing deleted by recovery")
	}
}

func TestContainerRetirementRequiresExactNameLabelAndImage(t *testing.T) {
	for _, foreign := range []string{"label", "image", "name", "owned"} {
		t.Run(foreign, func(t *testing.T) {
			runID := "31c832dd-4761-4f80-aa9c-d43da34173df"
			name := "tg-seed-" + runID + "-redis"
			image := "sha256:" + strings.Repeat("a", 64)
			var deletes atomic.Int32
			fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "/_ping") {
					w.Header().Set("API-Version", "1.44")
					w.WriteHeader(200)
					return
				}
				if r.Method == http.MethodDelete {
					deletes.Add(1)
					if !strings.HasSuffix(r.URL.Path, "/containers/exact-private-id") {
						t.Error("delete selected a name instead of the verified immutable container ID")
					}
					w.WriteHeader(204)
					return
				}
				label, actualImage, actualName := runID, image, name
				if foreign == "label" {
					label = "different-owner"
				}
				if foreign == "image" {
					actualImage = "sha256:" + strings.Repeat("b", 64)
				}
				if foreign == "name" {
					actualName = "foreign-name"
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"Id": "exact-private-id", "Name": "/" + actualName, "Image": actualImage, "Config": map[string]any{"Labels": map[string]string{"vrooli.test-genie.seed-owner": label}}})
			}))
			defer fixture.Close()
			t.Setenv("DOCKER_HOST", "tcp://"+strings.TrimPrefix(fixture.URL, "http://"))
			t.Setenv("DOCKER_TLS_VERIFY", "")
			t.Setenv("DOCKER_CERT_PATH", "")
			t.Setenv("DOCKER_API_VERSION", "1.44")
			err := RetireAllocation(t.Context(), Allocation{Kind: "container", RunID: runID, Name: name, ImageID: image})
			if foreign == "owned" {
				if err != nil || deletes.Load() != 1 {
					t.Fatalf("exact owned container was not retired once: %v deletes=%d", err, deletes.Load())
				}
			} else if err == nil || deletes.Load() != 0 {
				t.Fatalf("foreign container retirement was not refused: %v deletes=%d", err, deletes.Load())
			}
		})
	}
}
func TestPrivateContainerUsesInstalledImageAndOwnerControlledLifetime(t *testing.T) {
	runID := "31c832dd-4761-4f80-aa9c-d43da34173df"
	name := "tg-seed-" + runID + "-redis"
	image := "sha256:" + strings.Repeat("a", 64)
	var created atomic.Bool
	var tracked atomic.Bool
	var pulls atomic.Int32
	var deletes atomic.Int32
	var fixture *httptest.Server
	fixture = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/_ping"):
			w.Header().Set("API-Version", "1.44")
			w.WriteHeader(200)
		case strings.Contains(r.URL.Path, "/images/create"):
			pulls.Add(1)
			http.Error(w, "image pull forbidden", 500)
		case strings.Contains(r.URL.Path, "/images/") && strings.HasSuffix(r.URL.Path, "/json"):
			_ = json.NewEncoder(w).Encode(map[string]any{"Id": image})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/containers/create"):
			if !tracked.Load() {
				t.Error("container create preceded durable allocation ownership")
			}
			if r.URL.Query().Get("name") != name {
				t.Error("container create omitted exact private name")
			}
			var config struct {
				Image      string
				Labels     map[string]string
				HostConfig struct {
					AutoRemove   bool
					PortBindings map[string][]struct{ HostIP, HostPort string }
				}
			}
			if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
				t.Error(err)
			}
			if config.Image != image || config.Labels["vrooli.test-genie.seed-owner"] != runID || config.HostConfig.AutoRemove {
				t.Error("private container identity or lifetime was not pinned")
			}
			bindings := config.HostConfig.PortBindings["6379/tcp"]
			if len(bindings) != 1 || bindings[0].HostIP != "127.0.0.1" {
				t.Error("private container listener was not loopback-only")
			}
			created.Store(true)
			w.WriteHeader(201)
			fmt.Fprint(w, `{"Id":"exact-private-id","Warnings":[]}`)
		case strings.HasSuffix(r.URL.Path, "/start"):
			w.WriteHeader(204)
		case r.Method == http.MethodDelete:
			deletes.Add(1)
			w.WriteHeader(204)
		case strings.Contains(r.URL.Path, "/containers/") && strings.HasSuffix(r.URL.Path, "/json"):
			if !created.Load() {
				w.WriteHeader(404)
				fmt.Fprint(w, `{"message":"not found"}`)
				return
			}
			parsed, _ := url.Parse(fixture.URL)
			_ = json.NewEncoder(w).Encode(map[string]any{"Id": "exact-private-id", "Name": "/" + name, "Image": image, "Config": map[string]any{"Labels": map[string]string{"vrooli.test-genie.seed-owner": runID}}, "NetworkSettings": map[string]any{"Ports": map[string]any{"6379/tcp": []map[string]string{{"HostIp": "127.0.0.1", "HostPort": parsed.Port()}}}}})
		default:
			t.Errorf("unexpected Docker owner request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(500)
		}
	}))
	defer fixture.Close()
	t.Setenv("DOCKER_HOST", "tcp://"+strings.TrimPrefix(fixture.URL, "http://"))
	t.Setenv("DOCKER_TLS_VERIFY", "")
	t.Setenv("DOCKER_CERT_PATH", "")
	t.Setenv("DOCKER_API_VERSION", "1.44")
	var allocation Allocation
	manager := NewManager(Config{OnAllocation: func(a Allocation) error { allocation = a; tracked.Store(true); return nil }})
	host, port, cleanup, err := manager.startPrivateContainer(t.Context(), runID, "redis", "redis:7-alpine", "6379/tcp", nil)
	if err != nil || host != "127.0.0.1" || port == "" || cleanup == nil {
		t.Fatalf("private allocation failed: %s:%s %v", host, port, err)
	}
	if allocation.ImageID != image || pulls.Load() != 0 || deletes.Load() != 0 {
		t.Fatal("allocator changed image or delegated premature deletion")
	}
	if err := cleanup(t.Context()); err != nil {
		t.Fatal(err)
	}
	if deletes.Load() != 1 {
		t.Fatal("owner did not retire its exact container")
	}
}
