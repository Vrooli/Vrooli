package inspect

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func inspectTestClient(baseURL string) *Client {
	apiClient := cliutil.NewAPIClient(
		cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}),
		func() cliutil.APIBaseOptions {
			return cliutil.APIBaseOptions{Override: baseURL}
		},
		nil,
	)
	return NewClient(apiClient)
}

func TestLogsBuildsExpectedQuery(t *testing.T) {
	var got url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/deployments/dep-123/logs" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		got = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"deployment_id":"dep-123","logs":[],"total_count":0}`))
	}))
	defer server.Close()

	client := inspectTestClient(server.URL)
	_, _, err := client.Logs("dep-123", LogsOptions{
		Source: "caddy",
		Level:  "error",
		Search: "connect refused",
		Tail:   42,
		Since:  "1h",
	})
	if err != nil {
		t.Fatalf("Logs returned error: %v", err)
	}
	if got.Get("source") != "caddy" || got.Get("level") != "error" || got.Get("search") != "connect refused" || got.Get("tail") != "42" || got.Get("since") != "1h" {
		t.Fatalf("unexpected query params: %v", got)
	}
}

func TestParseLogsArgsPreservesFiltersAndBounds(t *testing.T) {
	id, opts, jsonOutput, maxBytes, showHelp, err := parseLogsArgs([]string{
		"dep-123", "--source", "caddy", "--level", "warn", "--search", "connect refused",
		"--tail", "42", "--since", "1h", "--max-bytes", "4096", "--json",
	})
	if err != nil {
		t.Fatalf("parseLogsArgs returned error: %v", err)
	}
	if id != "dep-123" || opts.Source != "caddy" || opts.Level != "warn" || opts.Search != "connect refused" || opts.Tail != 42 || opts.Since != "1h" {
		t.Fatalf("unexpected parsed request: id=%q opts=%+v", id, opts)
	}
	if !jsonOutput || maxBytes != 4096 || showHelp {
		t.Fatalf("unexpected output options: json=%v maxBytes=%d help=%v", jsonOutput, maxBytes, showHelp)
	}
}

func TestParseLogsArgsRejectsInvalidTail(t *testing.T) {
	for _, tail := range []string{"0", "-1", "many", "2001"} {
		t.Run(tail, func(t *testing.T) {
			_, _, _, _, _, err := parseLogsArgs([]string{"dep-123", "--tail", tail})
			if err == nil {
				t.Fatalf("parseLogsArgs accepted invalid tail %q", tail)
			}
		})
	}
}

func TestParseLogsArgsHelpStopsFurtherParsing(t *testing.T) {
	_, _, _, _, showHelp, err := parseLogsArgs([]string{"--help", "--tail", "bad"})
	if err != nil || !showHelp {
		t.Fatalf("help should short-circuit later flags: help=%v err=%v", showHelp, err)
	}
}

func TestParseDeploymentArgs(t *testing.T) {
	id, jsonOutput, showHelp := parseDeploymentArgs([]string{"dep-456", "--json"})
	if id != "dep-456" || !jsonOutput || showHelp {
		t.Fatalf("unexpected parsed deployment args: id=%q json=%v help=%v", id, jsonOutput, showHelp)
	}
	id, jsonOutput, showHelp = parseDeploymentArgs([]string{"--help", "dep-789"})
	if id != "" || jsonOutput || !showHelp {
		t.Fatalf("help should short-circuit parsing: id=%q json=%v help=%v", id, jsonOutput, showHelp)
	}
}

func TestWriteDriftPreservesDetailsAndSeverity(t *testing.T) {
	var output strings.Builder
	writeDrift(&output, DriftResponse{
		DeploymentID: "dep-123",
		HasDrift:     true,
		CheckedAt:    "now",
		DriftItems: []DriftItem{{
			Path: "service.port", Type: "modified", Severity: "warning",
			Expected: "80", Actual: "8080", Message: "port changed",
		}},
	})
	for _, want := range []string{"Drift Detection for Deployment: dep-123", "DRIFT DETECTED: 1 item(s)", "[W] service.port (modified)", "Expected: 80", "Actual:   8080", "Message:  port changed", "Checked at: now"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("drift output does not contain %q:\n%s", want, output.String())
		}
	}
}

func TestWriteLiveStatePreservesStatusAndTargetDetails(t *testing.T) {
	var output strings.Builder
	writeLiveState(&output, LiveStateResponse{
		DeploymentID: "dep-789",
		State: LiveState{
			Running: true, Healthy: false, Uptime: "5m", PublicIP: "203.0.113.4",
			CPUPercent: "10%", ErrorMessage: "health check failed",
		},
		Processes: []ProcessInfo{{PID: 42, Name: "worker", Status: "running", CPUPercent: "1%", MemoryMB: "32"}},
		Resources: []ResourceStatus{{Name: "postgres", Type: "database", Status: "running", Port: 5432, Healthy: true}},
	})
	for _, want := range []string{"Live State for Deployment: dep-789", "[R] Running: true  [!] Healthy: false", "Uptime:   5m", "Public IP: 203.0.113.4", "Error:    health check failed", "worker", "postgres", "5432", "yes"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("live state output does not contain %q:\n%s", want, output.String())
		}
	}
}

func TestWriteMetricsIncludesSystemAndCommandResults(t *testing.T) {
	var output strings.Builder
	writeMetrics(&output, MetricsDebugResponse{
		DeploymentID: "dep-321",
		Result: MetricsDebugResult{
			Collector: "native", OSID: "ubuntu", OSVersion: "24.04", Error: "partial result",
			System: SystemState{
				CPU: CPUInfo{UsagePercent: 12.5, Cores: 4, LoadAverage: []float64{1, 2, 3}},
				SSH: SSHHealth{Connected: true, LatencyMs: 12},
			},
			Commands: []MetricsDebugCommand{{ID: "probe", Command: "uptime", ExitCode: 1, DurationMs: 8, Error: "permission denied"}},
		},
	})
	for _, want := range []string{"Metrics Debug for Deployment: dep-321", "Collector: native  OS: ubuntu 24.04", "auth_mode=unknown key_auth=unknown", "CPU: usage=12.5% cores=4 load=[1.00 2.00 3.00]", "Error: partial result", "probe", "ERR", "permission denied"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("metrics output does not contain %q:\n%s", want, output.String())
		}
	}
}

func TestParseFilesArgsAndWriteDirectoryListing(t *testing.T) {
	id, opts, jsonOutput, showHelp := parseFilesArgs([]string{"dep-654", "/var/log", "--content", "--json"})
	if id != "dep-654" || opts.Path != "/var/log" || !opts.Content || !jsonOutput || showHelp {
		t.Fatalf("unexpected parsed file args: id=%q opts=%+v json=%v help=%v", id, opts, jsonOutput, showHelp)
	}
	var output strings.Builder
	writeFiles(&output, FilesOptions{}, FilesResponse{
		Path:  "/var/log",
		Files: []FileInfo{{Name: "app", IsDirectory: true, Size: 2048, ModTime: "today", Mode: "drwxr-xr-x"}},
	})
	for _, want := range []string{"Directory: /var/log", "MODE", "2.0K", "app/"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("file listing does not contain %q:\n%s", want, output.String())
		}
	}
}

func TestFilesContentQuery(t *testing.T) {
	var got url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/deployments/dep-456/files/content" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		got = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"deployment_id":"dep-456","path":"/etc/caddy/Caddyfile","content":"ok"}`))
	}))
	defer server.Close()

	client := inspectTestClient(server.URL)
	_, _, err := client.Files("dep-456", FilesOptions{
		Path:    "/etc/caddy/Caddyfile",
		Content: true,
	})
	if err != nil {
		t.Fatalf("Files returned error: %v", err)
	}
	if got.Get("path") != "/etc/caddy/Caddyfile" {
		t.Fatalf("unexpected query params: %v", got)
	}
}

func TestTruncateAndSafeLoadHelpers(t *testing.T) {
	if got := truncate("short", 10); got != "short" {
		t.Fatalf("truncate short=%q", got)
	}
	if got := truncate("1234567890", 7); got != "1234..." {
		t.Fatalf("truncate long=%q", got)
	}
	if got := safeLoad([]float64{0.1, 0.2, 0.3}, 1); got != 0.2 {
		t.Fatalf("safeLoad existing=%v", got)
	}
	if got := safeLoad([]float64{0.1}, 3); got != 0 {
		t.Fatalf("safeLoad out-of-range=%v", got)
	}
}

func TestLiveStateSupportsEnvelopeShape(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/deployments/dep-789/live-state" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"result":{
				"ok":true,
				"timestamp":"2026-02-08T23:00:00Z",
				"processes":{
					"scenarios":[
						{"id":"landing-page-business-suite","status":"running","pid":123,"resources":{"cpu_percent":1.2,"memory_mb":42}}
					],
					"resources":[
						{"id":"postgres","status":"running","pid":456}
					]
				},
				"system":{
					"cpu":{"usage_percent":2.5},
					"memory":{"usage_percent":45.0},
					"disk":{"used_gb":17,"total_gb":23},
					"swap":{},
					"ssh":{"public_key_fingerprint":"SHA256:test"},
					"uptime_seconds":100
				}
			},
			"timestamp":"2026-02-08T23:00:01Z"
		}`))
	}))
	defer server.Close()

	client := inspectTestClient(server.URL)
	_, resp, err := client.LiveState("dep-789")
	if err != nil {
		t.Fatalf("LiveState returned error: %v", err)
	}

	if resp.DeploymentID != "dep-789" {
		t.Fatalf("DeploymentID = %q, want dep-789", resp.DeploymentID)
	}
	if !resp.State.Running {
		t.Fatalf("expected running=true")
	}
	if !resp.State.Healthy {
		t.Fatalf("expected healthy=true")
	}
	if !strings.Contains(resp.State.CPUPercent, "2.5") {
		t.Fatalf("unexpected CPUPercent: %q", resp.State.CPUPercent)
	}
	if len(resp.Processes) != 1 || resp.Processes[0].Name != "landing-page-business-suite" {
		t.Fatalf("unexpected processes: %+v", resp.Processes)
	}
	if len(resp.Resources) != 1 || resp.Resources[0].Name != "postgres" {
		t.Fatalf("unexpected resources: %+v", resp.Resources)
	}
}
