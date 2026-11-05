package packaged

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
)

func TestCaptureForwardsIndependentOptimizationToggles(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/packaged/capture" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"experiment_id":"e3-test","samples":[{},{}]}`))
	}))
	defer server.Close()

	ctx, output := cliapptest.NewCapturedRunContext(nil, captureSchema(), cliapptest.TestRunContextOptions{
		JSON:      true,
		Flags:     map[string]string{"scenario": "hello-desktop", "artifact": "/tmp/hello.AppImage", "ramp": "desktop", "strategy": "electron-optimized", "runs": "2", "condition": "cold", "experiment": "e3-test", "journey": "hello-journey"},
		BoolFlags: map[string]bool{"secure-custom-scheme-ui": true, "electron-fuses": true},
	})
	h := handlers{client: server.Client(), base: server.URL}
	if err := h.capture(ctx); err != nil {
		t.Fatal(err)
	}
	profile, ok := received["optimization_toggles"].(map[string]any)
	if !ok || profile["shared_runtime_readiness"] != false || profile["secure_custom_scheme_ui"] != true || profile["v8_snapshot_code_cache"] != false || profile["electron_fuses"] != true {
		t.Fatalf("optimization_toggles = %#v", received["optimization_toggles"])
	}
	if received["strategy"] != "electron-optimized" || received["journey"] != "hello-journey" || received["runs"] != float64(2) {
		t.Fatalf("capture request lost its strategy, journey or run count: %#v", received)
	}
	if output.String() != "{\"experiment_id\":\"e3-test\",\"samples\":[{},{}]}\n" {
		t.Fatalf("JSON response = %q", output.String())
	}
}

func TestCaptureRejectsNonIntegerRunCountBeforeCallingAPI(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { called = true }))
	defer server.Close()
	ctx, _ := cliapptest.NewCapturedRunContext(nil, captureSchema(), cliapptest.TestRunContextOptions{Flags: map[string]string{"runs": "many"}})
	h := handlers{client: server.Client(), base: server.URL}
	if err := h.capture(ctx); err == nil {
		t.Fatal("invalid run count unexpectedly succeeded")
	}
	if called {
		t.Fatal("invalid run count reached the packaged API")
	}
}

func TestReportCountsOnlyComparableCellsButListsEveryCell(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/packaged/report" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"schema_version":"packaged-comparison.v1","cells":[{"scenario":"hello-desktop","condition":"cold","experiment_id":"good","samples":10,"comparable":true,"metrics":{}},{"scenario":"web-console","condition":"warm","experiment_id":"partial","samples":3,"comparable":false,"metrics":{}}]}`))
	}))
	defer server.Close()

	ctx, output := cliapptest.NewCapturedRunContext(nil, captureSchema(), cliapptest.TestRunContextOptions{})
	h := handlers{client: server.Client(), base: server.URL}
	if err := h.report(ctx); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"1 comparable of 2 cells", "hello-desktop cold (good)", "web-console warm (partial)", "comparable=false"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("human report output %q missing %q", output.String(), want)
		}
	}
}

func TestReportDefaultsToAllStrategiesAndShowsResourceTradeoffs(t *testing.T) {
	group, err := Register(&cliapp.ScenarioApp{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var schema cliapp.ArgSchema
	for _, command := range group.Subcommands {
		if command.Name == "report" {
			schema = command.Args
		}
	}
	for _, flag := range schema.Flags {
		if flag.Name == "strategy" && flag.Default != "" {
			t.Fatalf("report hides optimized cells behind default strategy %q", flag.Default)
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("strategy") != "" || r.URL.Query().Get("experiment_id") != "screening" {
			t.Errorf("report filters = %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"cells":[{"scenario":"hello-desktop","strategy":"electron-optimized","condition":"warm","experiment_id":"screening","samples":1,"comparable":true,"metrics":{"app_interactive_ms":{"p50":1043},"steady_pss_mb":{"p50":307},"artifact_bytes":{"p50":161309408}}},{"scenario":"other","strategy":"electron-sidecar","metrics":{}}]}`))
	}))
	defer server.Close()
	ctx, output := cliapptest.NewCapturedRunContext(nil, schema, cliapptest.TestRunContextOptions{Flags: map[string]string{"experiment": "screening"}})
	if err := (handlers{client: server.Client(), base: server.URL}).report(ctx); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"electron-optimized", "1043.0 ms", "steady PSS p50 307.0 MB", "artifact p50 161309408.0 bytes", "steady PSS p50 unavailable", "not an adoption decision"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("report omits %q: %s", want, output.String())
		}
	}
}

func captureSchema() cliapp.ArgSchema {
	return cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "plan", Bool: true},
		{Name: "scenario"},
		{Name: "artifact"},
		{Name: "ramp"},
		{Name: "strategy"},
		{Name: "runs"},
		{Name: "condition"},
		{Name: "experiment"},
		{Name: "journey"},
		{Name: "shared-runtime-readiness", Bool: true},
		{Name: "secure-custom-scheme-ui", Bool: true},
		{Name: "v8-snapshot-code-cache", Bool: true},
		{Name: "electron-fuses", Bool: true},
	}}
}

func TestCapturePlanForwardsPreviewAndRendersOwnerCost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		if r.URL.Path != "/api/v1/packaged/capture/plan" {
			t.Errorf("preview did not select dedicated path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"samples":null,"planned_work":{"measurement_samples":2,"discarded_warmups":1,"measured_launch_operations":3,"nominal_protocol_demo_starts":6}}`))
	}))
	defer server.Close()
	ctx, output := cliapptest.NewCapturedRunContext(nil, captureSchema(), cliapptest.TestRunContextOptions{Flags: map[string]string{"runs": "2", "condition": "cold", "scenario": "hello-desktop"}, BoolFlags: map[string]bool{"plan": true}})
	if err := (handlers{client: server.Client(), base: server.URL}).capture(ctx); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Planned 2 cold packaged samples") || !strings.Contains(output.String(), "nominal_protocol_demo_starts") {
		t.Fatalf("preview output: %s", output.String())
	}
}

func TestCapturePlanNeverFallsBackToOldServerCapture(t *testing.T) {
	captures, requests := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path == "/api/v1/packaged/capture" {
			captures++
			_, _ = w.Write([]byte(`{"samples":[{}]}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	ctx, _ := cliapptest.NewCapturedRunContext(nil, captureSchema(), cliapptest.TestRunContextOptions{Flags: map[string]string{"runs": "2", "condition": "cold", "scenario": "hello-desktop"}, BoolFlags: map[string]bool{"plan": true}})
	err := (handlers{client: server.Client(), base: server.URL}).capture(ctx)
	if err == nil || !strings.Contains(err.Error(), "404") || captures != 0 || requests != 1 {
		t.Fatalf("old server preview was unsafe: err=%v captures=%d requests=%d", err, captures, requests)
	}
}
