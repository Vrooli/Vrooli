package system

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"scenario-to-desktop/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func newTestClient(handler http.Handler) support.Dependencies {
	server := httptest.NewServer(handler)
	app, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:             "scenario-to-desktop-test",
		Version:          "test",
		Description:      "test",
		DefaultAPIBase:   server.URL,
		AllowAnonymous:   true,
		CommandGroups:    func(*cliapp.ScenarioApp) []cliapp.CommandGroup { return nil },
		SubcommandGroups: func(*cliapp.ScenarioApp) []cliapp.SubcommandGroup { return nil },
	})
	if err != nil {
		panic(err)
	}
	return support.Dependencies{Core: func() *cliapp.ScenarioApp { return app }}
}

// jsonHandler returns an http.HandlerFunc that responds with the given JSON body.
func jsonHandler(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}
}

// --- Status ---

func TestStatus_Success(t *testing.T) {
	var requestPaths []string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPaths = append(requestPaths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/health") {
			_, _ = w.Write([]byte(`{"status":"healthy"}`))
		} else {
			_, _ = w.Write([]byte(`{"service":{"name":"std","version":"1.0","status":"running"},"statistics":{"total_builds":10,"active_builds":1,"completed_builds":8,"failed_builds":1}}`))
		}
	})

	cmds := New(newTestClient(handler))
	err := cmds.Status([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(requestPaths) != 2 {
		t.Errorf("expected 2 API calls (health + status), got %d", len(requestPaths))
	}
}

func TestStatus_JSONOutput(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/health") {
			_, _ = w.Write([]byte(`{"status":"healthy"}`))
		} else {
			_, _ = w.Write([]byte(`{"service":{"name":"std"}}`))
		}
	})

	cmds := New(newTestClient(handler))
	err := cmds.Status([]string{"--json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStatus_HealthCheckFails(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/health") {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":"unhealthy"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})

	cmds := New(newTestClient(handler))
	err := cmds.Status([]string{})
	if err == nil {
		t.Fatal("expected error when health check fails")
	}
	if !strings.Contains(err.Error(), "health check failed") {
		t.Errorf("error = %q, want to contain 'health check failed'", err.Error())
	}
}

func TestStatus_InvalidFlag(t *testing.T) {
	cmds := New(newTestClient(http.NotFoundHandler()))
	err := cmds.Status([]string{"--unknown"})
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
}

// --- TemplatesList ---

func TestTemplatesList_Success(t *testing.T) {
	handler := jsonHandler(`{"templates":[{"name":"Basic","type":"basic","description":"Simple wrapper","complexity":"low"},{"name":"Kiosk","type":"kiosk","description":"Kiosk mode","complexity":"medium"}]}`)

	cmds := New(newTestClient(handler))
	err := cmds.TemplatesList([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTemplatesList_JSONOutput(t *testing.T) {
	handler := jsonHandler(`{"templates":[]}`)

	cmds := New(newTestClient(handler))
	err := cmds.TemplatesList([]string{"--json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTemplatesList_UnmarshalFallback(t *testing.T) {
	// When response doesn't match expected struct, it should fall back to PrintJSON
	handler := jsonHandler(`{"unexpected":"format"}`)

	cmds := New(newTestClient(handler))
	// Should not error — falls back to raw JSON printing
	err := cmds.TemplatesList([]string{})
	if err != nil {
		t.Fatalf("unexpected error on unmarshal fallback: %v", err)
	}
}

func TestTemplatesList_APIError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"server error"}`))
	})

	cmds := New(newTestClient(handler))
	err := cmds.TemplatesList([]string{})
	if err == nil {
		t.Fatal("expected error from API failure")
	}
}

// --- TemplateGet ---

func TestTemplateGet_MissingType(t *testing.T) {
	cmds := New(newTestClient(http.NotFoundHandler()))
	err := cmds.TemplateGet([]string{})
	if err == nil {
		t.Fatal("expected error for missing template type")
	}
	if !strings.Contains(err.Error(), "usage:") {
		t.Errorf("error = %q, want usage message", err.Error())
	}
}

func TestTemplateGet_Success(t *testing.T) {
	var receivedPath string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"type":"kiosk","name":"Kiosk Template"}`))
	})

	cmds := New(newTestClient(handler))
	err := cmds.TemplateGet([]string{"kiosk"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(receivedPath, "/templates/kiosk") {
		t.Errorf("path = %q, want to contain '/templates/kiosk'", receivedPath)
	}
}

func TestTemplateGet_JSONOutput(t *testing.T) {
	handler := jsonHandler(`{"type":"basic"}`)

	cmds := New(newTestClient(handler))
	err := cmds.TemplateGet([]string{"basic", "--json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- RecordsList ---

func TestRecordsList_Success(t *testing.T) {
	handler := jsonHandler(`{"records":[{"record":{"id":"abc-123","scenario_name":"my-app","status":"complete"},"build_state":"built"}]}`)

	cmds := New(newTestClient(handler))
	err := cmds.RecordsList([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRecordsList_EmptyRecords(t *testing.T) {
	handler := jsonHandler(`{"records":[]}`)

	cmds := New(newTestClient(handler))
	err := cmds.RecordsList([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRecordsList_UnmarshalFallback(t *testing.T) {
	handler := jsonHandler(`{"not_records":"value"}`)

	cmds := New(newTestClient(handler))
	err := cmds.RecordsList([]string{})
	if err != nil {
		t.Fatalf("unexpected error on unmarshal fallback: %v", err)
	}
}

func TestRecordsList_JSONOutput(t *testing.T) {
	handler := jsonHandler(`{"records":[]}`)

	cmds := New(newTestClient(handler))
	err := cmds.RecordsList([]string{"--json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- RecordsMove ---

func TestRecordsMove_MissingID(t *testing.T) {
	cmds := New(newTestClient(http.NotFoundHandler()))
	err := cmds.RecordsMove([]string{})
	if err == nil {
		t.Fatal("expected error for missing record ID")
	}
	if !strings.Contains(err.Error(), "usage:") {
		t.Errorf("error = %q, want usage message", err.Error())
	}
}

func TestRecordsMove_DefaultTarget(t *testing.T) {
	var receivedBody map[string]interface{}
	var receivedPath string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"moved":true}`))
	})

	cmds := New(newTestClient(handler))
	err := cmds.RecordsMove([]string{"record-123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(receivedPath, "/desktop/records/record-123/move") {
		t.Errorf("path = %q, want to contain '/desktop/records/record-123/move'", receivedPath)
	}
	if receivedBody["target"] != "destination" {
		t.Errorf("target = %v, want 'destination'", receivedBody["target"])
	}
}

func TestRecordsMove_CustomPath(t *testing.T) {
	var receivedBody map[string]interface{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"moved":true}`))
	})

	cmds := New(newTestClient(handler))
	err := cmds.RecordsMove([]string{"record-123", "--target", "custom", "--path", "/opt/apps"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedBody["target"] != "custom" {
		t.Errorf("target = %v, want 'custom'", receivedBody["target"])
	}
	if receivedBody["destination_path"] != "/opt/apps" {
		t.Errorf("destination_path = %v, want '/opt/apps'", receivedBody["destination_path"])
	}
}

func TestRecordsMove_JSONOutput(t *testing.T) {
	handler := jsonHandler(`{"moved":true}`)

	cmds := New(newTestClient(handler))
	err := cmds.RecordsMove([]string{"record-123", "--json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- RecordsDelete ---

func TestRecordsDelete_MissingScenario(t *testing.T) {
	cmds := New(newTestClient(http.NotFoundHandler()))
	err := cmds.RecordsDelete([]string{})
	if err == nil {
		t.Fatal("expected error for missing scenario")
	}
	if !strings.Contains(err.Error(), "usage:") {
		t.Errorf("error = %q, want usage message", err.Error())
	}
}

func TestRecordsDelete_Success(t *testing.T) {
	var receivedPath string
	var receivedMethod string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		receivedMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"deleted":true}`))
	})

	cmds := New(newTestClient(handler))
	err := cmds.RecordsDelete([]string{"my-app"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", receivedMethod)
	}
	if !strings.Contains(receivedPath, "/desktop/delete/my-app") {
		t.Errorf("path = %q, want to contain '/desktop/delete/my-app'", receivedPath)
	}
}

func TestRecordsDelete_JSONOutput(t *testing.T) {
	handler := jsonHandler(`{"deleted":true}`)

	cmds := New(newTestClient(handler))
	err := cmds.RecordsDelete([]string{"my-app", "--json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Download ---

func TestDownload_MissingArgs(t *testing.T) {
	cmds := New(newTestClient(http.NotFoundHandler()))

	tests := []struct {
		name string
		args []string
	}{
		{"no args", []string{}},
		{"only scenario", []string{"my-app"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := cmds.Download(tc.args)
			if err == nil {
				t.Fatal("expected error for missing arguments")
			}
			if !strings.Contains(err.Error(), "usage:") {
				t.Errorf("error = %q, want usage message", err.Error())
			}
		})
	}
}

func TestDownload_PlatformExtensions(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write([]byte("binary-content"))
	})

	tests := []struct {
		platform    string
		expectedExt string
	}{
		{"win", ".exe"},
		{"mac", ".zip"},
		{"linux", ".AppImage"},
	}

	for _, tc := range tests {
		t.Run(tc.platform, func(t *testing.T) {
			tmpDir := t.TempDir()
			cmds := New(newTestClient(handler))
			err := cmds.Download([]string{"my-app", tc.platform, "--output", filepath.Join(tmpDir, "out"+tc.expectedExt)})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			data, err := os.ReadFile(filepath.Join(tmpDir, "out"+tc.expectedExt))
			if err != nil {
				t.Fatalf("failed to read output: %v", err)
			}
			if string(data) != "binary-content" {
				t.Errorf("content = %q, want 'binary-content'", string(data))
			}
		})
	}
}

func TestDownload_DefaultFilename(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("bin"))
	})

	// Use a temp dir to avoid writing to cwd
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	cmds := New(newTestClient(handler))
	err := cmds.Download([]string{"my-app", "win"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedFile := filepath.Join(tmpDir, "my-app-win.exe")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("expected file %q to be created", expectedFile)
	}
}

func TestDownload_APIError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"not found"}`))
	})

	cmds := New(newTestClient(handler))
	err := cmds.Download([]string{"nonexistent", "win", "--output", filepath.Join(t.TempDir(), "out.exe")})
	if err == nil {
		t.Fatal("expected error from API failure")
	}
}

// --- DesktopStatus ---

func TestDesktopStatus_Success(t *testing.T) {
	handler := jsonHandler(`{
		"scenarios":[
			{"name":"app-a","display_name":"App A","version":"1.0","built":true,"platforms":["win","linux"],"build_artifacts":[{"platform":"win","file_name":"app.exe","size_bytes":1024}]},
			{"name":"app-b","display_name":"","version":"","built":false,"platforms":[],"build_artifacts":[]}
		],
		"stats":{"total":2,"with_desktop":1,"built":1,"web_only":1}
	}`)

	cmds := New(newTestClient(handler))
	err := cmds.DesktopStatus([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDesktopStatus_NameFilter(t *testing.T) {
	handler := jsonHandler(`{
		"scenarios":[
			{"name":"app-a","display_name":"","version":"1.0","built":true,"platforms":["win"],"build_artifacts":[]},
			{"name":"app-b","display_name":"","version":"2.0","built":true,"platforms":["linux"],"build_artifacts":[]}
		],
		"stats":{"total":2,"with_desktop":2,"built":2,"web_only":0}
	}`)

	cmds := New(newTestClient(handler))
	// Filtering by name should only show app-a
	err := cmds.DesktopStatus([]string{"--name", "app-a"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDesktopStatus_FilterNoMatch(t *testing.T) {
	handler := jsonHandler(`{
		"scenarios":[{"name":"app-a","display_name":"","version":"1.0","built":true,"platforms":[],"build_artifacts":[]}],
		"stats":{"total":1,"with_desktop":1,"built":1,"web_only":0}
	}`)

	cmds := New(newTestClient(handler))
	// Filter that matches nothing should print "No scenarios found"
	err := cmds.DesktopStatus([]string{"--name", "nonexistent"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDesktopStatus_EmptyScenarios(t *testing.T) {
	handler := jsonHandler(`{"scenarios":[],"stats":{"total":0,"with_desktop":0,"built":0,"web_only":0}}`)

	cmds := New(newTestClient(handler))
	err := cmds.DesktopStatus([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDesktopStatus_JSONOutput(t *testing.T) {
	handler := jsonHandler(`{"scenarios":[],"stats":{}}`)

	cmds := New(newTestClient(handler))
	err := cmds.DesktopStatus([]string{"--json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDesktopStatus_UnexpectedPositionalArgs(t *testing.T) {
	cmds := New(newTestClient(http.NotFoundHandler()))
	err := cmds.DesktopStatus([]string{"some-arg"})
	if err == nil {
		t.Fatal("expected error for unexpected positional args")
	}
	if !strings.Contains(err.Error(), "usage:") {
		t.Errorf("error = %q, want usage message", err.Error())
	}
}

func TestDesktopStatus_UnmarshalFallback(t *testing.T) {
	handler := jsonHandler(`{"unexpected":"data"}`)

	cmds := New(newTestClient(handler))
	err := cmds.DesktopStatus([]string{})
	if err != nil {
		t.Fatalf("unexpected error on unmarshal fallback: %v", err)
	}
}

func TestDesktopStatus_ArtifactFallbackPath(t *testing.T) {
	// When file_name is empty, should fall back to relative_path
	handler := jsonHandler(`{
		"scenarios":[{"name":"app-a","display_name":"","version":"1.0","built":true,"platforms":["linux"],
			"build_artifacts":[{"platform":"linux","file_name":"","size_bytes":2048,"relative_path":"dist/app.AppImage"}]}],
		"stats":{"total":1,"with_desktop":1,"built":1,"web_only":0}
	}`)

	cmds := New(newTestClient(handler))
	err := cmds.DesktopStatus([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- WineCheck ---

func TestWineCheck_Installed(t *testing.T) {
	handler := jsonHandler(`{"installed":true,"version":"9.0","usable":true,"install_method":"flatpak","available_install_methods":[]}`)

	cmds := New(newTestClient(handler))
	err := cmds.WineCheck([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWineCheck_NotInstalled(t *testing.T) {
	handler := jsonHandler(`{"installed":false,"version":"","usable":false,"install_method":"","available_install_methods":["flatpak","appimage"]}`)

	cmds := New(newTestClient(handler))
	err := cmds.WineCheck([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWineCheck_InstalledButNotUsable(t *testing.T) {
	handler := jsonHandler(`{"installed":true,"version":"8.0","usable":false,"install_method":"apt","available_install_methods":[]}`)

	cmds := New(newTestClient(handler))
	err := cmds.WineCheck([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWineCheck_JSONOutput(t *testing.T) {
	handler := jsonHandler(`{"installed":true,"version":"9.0","usable":true}`)

	cmds := New(newTestClient(handler))
	err := cmds.WineCheck([]string{"--json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWineCheck_UnmarshalFallback(t *testing.T) {
	handler := jsonHandler(`{"bad":"data"}`)

	cmds := New(newTestClient(handler))
	err := cmds.WineCheck([]string{})
	if err != nil {
		t.Fatalf("unexpected error on unmarshal fallback: %v", err)
	}
}

// --- WineInstall ---

func TestWineInstall_MissingMethod(t *testing.T) {
	cmds := New(newTestClient(http.NotFoundHandler()))
	err := cmds.WineInstall([]string{})
	if err == nil {
		t.Fatal("expected error for missing method")
	}
	if !strings.Contains(err.Error(), "usage:") {
		t.Errorf("error = %q, want usage message", err.Error())
	}
}

func TestWineInstall_Success(t *testing.T) {
	var receivedBody map[string]interface{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"install_id":"inst-001","status":"started","status_url":"/wine/status/inst-001"}`))
	})

	cmds := New(newTestClient(handler))
	err := cmds.WineInstall([]string{"--method", "flatpak"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedBody["method"] != "flatpak" {
		t.Errorf("method = %v, want 'flatpak'", receivedBody["method"])
	}
}

func TestWineInstall_JSONOutput(t *testing.T) {
	handler := jsonHandler(`{"install_id":"inst-001","status":"started"}`)

	cmds := New(newTestClient(handler))
	err := cmds.WineInstall([]string{"--method", "appimage", "--json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWineInstall_UnmarshalFallback(t *testing.T) {
	handler := jsonHandler(`{"unexpected":"format"}`)

	cmds := New(newTestClient(handler))
	err := cmds.WineInstall([]string{"--method", "flatpak"})
	if err != nil {
		t.Fatalf("unexpected error on unmarshal fallback: %v", err)
	}
}

// --- WineStatus ---

func TestWineStatus_MissingInstallID(t *testing.T) {
	cmds := New(newTestClient(http.NotFoundHandler()))
	err := cmds.WineStatus([]string{})
	if err == nil {
		t.Fatal("expected error for missing install ID")
	}
	if !strings.Contains(err.Error(), "usage:") {
		t.Errorf("error = %q, want usage message", err.Error())
	}
}

func TestWineStatus_Success(t *testing.T) {
	var receivedPath string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"install_id":"inst-001","status":"complete","progress":100}`))
	})

	cmds := New(newTestClient(handler))
	err := cmds.WineStatus([]string{"inst-001"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(receivedPath, "/system/wine/install/status/inst-001") {
		t.Errorf("path = %q, want to contain '/system/wine/install/status/inst-001'", receivedPath)
	}
}

func TestWineStatus_JSONOutput(t *testing.T) {
	handler := jsonHandler(`{"install_id":"inst-001","status":"running"}`)

	cmds := New(newTestClient(handler))
	err := cmds.WineStatus([]string{"inst-001", "--json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
