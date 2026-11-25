//go:build journey

package main

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/cli-core/agentcatalog/harness"
	"github.com/vrooli/vrooli/packages/capabilityprobe"
)

// TestJ8_FictionalHarnessInServedDesktop proves a new harness descriptor and
// model policy can be added as resource data and consumed by the production
// API and the already-built production UI. It deliberately does not install or
// invoke the fictional binary.
func TestJ8_FictionalHarnessInServedDesktop(t *testing.T) {
	evidence := os.Getenv("LAUNCHER_JOURNEY_EVIDENCE")
	if evidence == "" {
		t.Fatal("LAUNCHER_JOURNEY_EVIDENCE must name the production UI build and evidence directory")
	}
	dist := filepath.Join(evidence, "ui-dist")
	if _, err := os.Stat(filepath.Join(dist, "index.html")); err != nil {
		t.Fatalf("production UI build missing from %s: %v", dist, err)
	}
	if os.Getenv("AC6_FICTIONAL_CHILD") != "1" {
		bundleRoot := t.TempDir()
		resourceRoot := filepath.Join(bundleRoot, "catalog")
		writeNimbusFixture(t, resourceRoot)
		cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestJ8_FictionalHarnessInServedDesktop$")
		cmd.Env = append(os.Environ(),
			"AC6_FICTIONAL_CHILD=1",
			"VROOLI_RESOURCE_ROOT="+resourceRoot,
			"VROOLI_DESKTOP_MODE=true",
			"BUNDLE_ROOT="+bundleRoot,
			"VROOLI_ROOT=",
			"LAUNCHER_JOURNEY_EVIDENCE="+evidence,
		)
		output, err := cmd.CombinedOutput()
		t.Log(string(output))
		if err != nil {
			t.Fatalf("production-startup fictional harness child: %v", err)
		}
		return
	}

	// The child process started with the fixture resource root in its initial
	// environment. Its package init path must project Nimbus into the same
	// catalogue used by the shipped Web Console process.
	nimbus, ok := harness.Active().Lookup("nimbus")
	if !ok {
		t.Fatal("production startup did not load the data-defined Nimbus harness")
	}
	if nimbus.ModelSourceProtocol != "" || nimbus.ModelSourceLimitation != "Nimbus is an offline fictional showcase harness with no executable or provider protocol; it cannot make a credits-door request." {
		t.Fatalf("fictional Nimbus credits limitation is not explicit data: %+v", nimbus)
	}
	if !slices.ContainsFunc(capabilityprobe.AITools, func(def capabilityprobe.Definition) bool { return def.ID == "nimbus" && def.Label == "Nimbus" }) {
		t.Fatalf("production startup probe projection omitted Nimbus: %+v", capabilityprobe.AITools)
	}
	pythonSite, err := exec.Command("python3", "-c", "import site; print(site.getusersitepackages())").Output()
	if err != nil {
		t.Fatal(err)
	}

	home := t.TempDir()
	for _, class := range []string{"config", "data", "cache", "logs", "state"} {
		root := filepath.Join(home, class)
		if err := os.MkdirAll(root, 0o700); err != nil {
			t.Fatal(err)
		}
		t.Setenv("VROOLI_"+strings.ToUpper(class)+"_ROOT", root)
	}
	t.Setenv("HOME", home)
	t.Setenv("VROOLI_HOME", filepath.Join(home, ".vrooli"))
	t.Setenv("WC_TMUX_SOCKET", filepath.Join(home, "tmux.sock"))
	t.Setenv("WC_SESSION_STATE_ROOT", filepath.Join(home, "sessions"))
	for _, class := range []string{"CONFIG", "DATA", "STATE", "CACHE"} {
		t.Setenv("XDG_"+class+"_HOME", filepath.Join(home, "xdg-"+strings.ToLower(class)))
	}
	t.Setenv("AUDIO_TOOLS_URL", "http://127.0.0.1:1")

	dbPath := filepath.Join(home, "data", "web-console.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	srv := NewServer(database.NewFromPrimary(db))
	t.Cleanup(func() {
		srv.sweeper.Stop()
		srv.sessions.StopReattachWatchdog()
		srv.sessions.Shutdown()
		if srv.codexTailer != nil {
			srv.codexTailer.Stop()
		}
		if srv.claudeTailer != nil {
			srv.claudeTailer.Stop()
		}
		if srv.grokTailer != nil {
			srv.grokTailer.Stop()
		}
		if srv.opencodeWatcher != nil {
			srv.opencodeWatcher.Stop()
		}
		_ = db.Close()
	})

	// The launch card is ordinary profile data. Persist a user-owned shortcut
	// that references the fixture descriptor, then let the served UI discover
	// descriptor capabilities and model choices through the same API as a user.
	ctx := context.Background()
	defaults := srv.shortcuts.Effective(ctx)
	shortcuts := append([]ShortcutEntry(nil), defaults.Shortcuts...)
	shortcuts = append(shortcuts, ShortcutEntry{Label: "Nimbus", Command: "nimbus", Description: "Fictional data-only harness", AgentID: "nimbus"})
	srv.shortcuts.Upsert(ctx, "default", "service", "Default", shortcuts)
	if got, ok := srv.shortcuts.Get(ctx, "default"); !ok || len(got.Shortcuts) != len(shortcuts) {
		t.Fatalf("fictional shortcut profile was not persisted: %+v, found=%t", got, ok)
	}
	if effective := srv.shortcuts.Effective(ctx); !slices.ContainsFunc(effective.Shortcuts, func(entry ShortcutEntry) bool { return entry.AgentID == "nimbus" }) {
		t.Fatalf("effective shortcut data omitted Nimbus: %+v", effective)
	}

	api := srv.Handler()
	files := http.FileServer(http.Dir(dist))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/vrooli.") || r.URL.Path == "/health" {
			api.ServeHTTP(w, r)
			return
		}
		files.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)

	request := httptest.NewRequest(http.MethodPost, "/vrooli.web_console.v1.capabilities.CapabilitiesService/ListHarnesses", strings.NewReader(`{"targetId":"local"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	api.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"agentId":"nimbus"`) || !strings.Contains(response.Body.String(), `"supportsContextLimit":true`) {
		t.Fatalf("served API did not expose descriptor data: status=%d body=%s", response.Code, response.Body.String())
	}
	policyRequest := httptest.NewRequest(http.MethodPost, "/vrooli.web_console.v1.modelselection.ModelSelectionService/GetRoleModelChoices", strings.NewReader(`{"agentId":"nimbus","targetId":"local"}`))
	policyRequest.Header.Set("Content-Type", "application/json")
	policyResponse := httptest.NewRecorder()
	api.ServeHTTP(policyResponse, policyRequest)
	if policyResponse.Code != http.StatusOK || !strings.Contains(policyResponse.Body.String(), "nimbus-1") {
		t.Fatalf("served API did not expose policy data: status=%d body=%s", policyResponse.Code, policyResponse.Body.String())
	}

	cmd := exec.CommandContext(t.Context(), "python3", "../test/fictional_harness_showcase.py", server.URL, evidence)
	cmd.Env = append(os.Environ(), "PYTHONPATH="+strings.TrimSpace(string(pythonSite)))
	output, err := cmd.CombinedOutput()
	t.Log(string(output))
	if err != nil {
		t.Fatalf("served fictional harness UI showcase: %v", err)
	}
}

func writeNimbusFixture(t *testing.T, root string) {
	t.Helper()
	markerDir := filepath.Join(root, ".vrooli")
	if err := os.MkdirAll(markerDir, 0o700); err != nil {
		t.Fatal(err)
	}
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	marker, err := os.ReadFile(filepath.Join(repoRoot, "packages", "cli-core", "agentcatalog", "harness", "catalog-marker.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(markerDir, filepath.Base(harness.CatalogRootMarkerPath)), marker, 0o600); err != nil {
		t.Fatal(err)
	}
	// The bundle fixture carries the shipped resource manifests plus one new
	// data-only harness, matching the declarative catalog staged by the desktop
	// packager. This keeps lifecycle checks for built-in harnesses on real data.
	resources, err := filepath.Glob(filepath.Join(repoRoot, "resources", "*", "resource.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range resources {
		resource := filepath.Base(filepath.Dir(source))
		resourceDir := filepath.Join(root, "resources", resource)
		if err := os.MkdirAll(resourceDir, 0o700); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(source)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(resourceDir, "resource.json"), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	resourceDir := filepath.Join(root, "resources", "nimbus")
	if err := os.MkdirAll(resourceDir, 0o700); err != nil {
		t.Fatal(err)
	}
	descriptor := `{"name":"nimbus","cli":{"enabled":true,"command":"resource-nimbus","adapter":{"kind":"go_module","module_dir":"cli"},"source_build":{"kind":"go_module"},"invoke":{"kind":"installed_command","command":"resource-nimbus"},"freshness":{"inputs":["cli/**","resource.json"]}},"driver":"external-cli","binary":"nimbus","version_args":["--version"],"platforms":{"linux":"supported"},"execution":{"runner_type":"nimbus"},"harness":{"id":"nimbus","binary":"nimbus","display_name":"Nimbus","icon":"terminal","model_source_limitation":"Nimbus is an offline fictional showcase harness with no executable or provider protocol; it cannot make a credits-door request.","home_isolation":{"kind":"none"},"usage_windows":[{"id":"daily","minutes":1440}],"features":{"model_selection":{"flag":"--model","sources":["nimbus-plan"]},"context_limit":{"kind":"env","key":"NIMBUS_CONTEXT_TOKENS","minimum":300000},"provider_selection":false,"resume":false,"goal_mode":false,"mcp":false,"hooks":false}}}`
	policy := `{"schema_version":"v1","runner":"nimbus","provenance":{"source":"fictional-showcase-data","observed_at":"2099-01-01"},"billing":{"mode":"subscription","sources":[{"id":"nimbus-plan","label":"Nimbus Plan","billing_mode":"subscription"}]},"model_aliases":{"nimbus-1":{"display_name":"Nimbus R1"}},"roles":{"code.default":{"model":"nimbus-1","models":[{"model":"nimbus-1","source":"nimbus-plan","effort":null}],"description":"Fictional showcase model","capabilities":["code"]},"code.fast":{"model":"nimbus-1","description":"Fictional showcase model","capabilities":["code"]},"code.smart":{"model":"nimbus-1","description":"Fictional showcase model","capabilities":["code"]},"code.cheap":{"model":"nimbus-1","description":"Fictional showcase model","capabilities":["code"]}}}`
	for name, value := range map[string]string{"resource.json": descriptor, "model-policy.json": policy} {
		if err := os.WriteFile(filepath.Join(resourceDir, name), []byte(value), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
