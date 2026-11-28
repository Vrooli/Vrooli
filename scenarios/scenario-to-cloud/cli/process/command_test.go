package process

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func processTestClient(baseURL string) *Client {
	apiClient := cliutil.NewAPIClient(
		cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}),
		func() cliutil.APIBaseOptions {
			return cliutil.APIBaseOptions{Override: baseURL}
		},
		nil,
	)
	return NewClient(apiClient)
}

func TestRunKillRejectsInvalidPID(t *testing.T) {
	err := Run(nil, []string{"kill", "dep-123", "not-a-number"})
	if err == nil {
		t.Fatal("expected invalid PID error")
	}
	if !strings.Contains(err.Error(), "invalid PID") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunRestartRejectsInvalidType(t *testing.T) {
	err := Run(nil, []string{"restart", "dep-123", "worker", "name"})
	if err == nil {
		t.Fatal("expected invalid type error")
	}
	if !strings.Contains(err.Error(), "type must be 'scenario' or 'resource'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseRestartArgsAndValidation(t *testing.T) {
	parsed, showHelp := parseRestartArgs([]string{"dep-5", "resource", "postgres", "--json"})
	if showHelp || parsed.deploymentID != "dep-5" || parsed.typeName != "resource" || parsed.name != "postgres" || !parsed.jsonOutput {
		t.Fatalf("unexpected restart args: %+v help=%v", parsed, showHelp)
	}
	if err := validateRestartArgs(parsed); err != nil {
		t.Fatalf("valid restart args rejected: %v", err)
	}
	parsed.typeName = "worker"
	if err := validateRestartArgs(parsed); err == nil {
		t.Fatal("invalid restart type was accepted")
	}
}

func TestParseKillArgsPreservesSignalDefaultAndOverride(t *testing.T) {
	parsed, showHelp, err := parseKillArgs([]string{"dep-6", "42"})
	if err != nil || showHelp || parsed.signal != "SIGTERM" || parsed.pid != 42 {
		t.Fatalf("unexpected default kill args: %+v help=%v err=%v", parsed, showHelp, err)
	}
	parsed, showHelp, err = parseKillArgs([]string{"dep-6", "42", "--signal", "SIGKILL", "--json"})
	if err != nil || showHelp || parsed.signal != "SIGKILL" || !parsed.jsonOutput {
		t.Fatalf("unexpected overridden kill args: %+v help=%v err=%v", parsed, showHelp, err)
	}
}

func TestRunVPSActionCarriesCleanupConfirmation(t *testing.T) {
	var got VPSActionRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/deployments/dep-7/actions/vps" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"action":"cleanup"}`))
	}))
	defer server.Close()
	client := processTestClient(server.URL)
	if err := runVPSAction(client, []string{"dep-7", "cleanup", "--cleanup-level", "3", "--confirmation", "CLEAN"}); err != nil {
		t.Fatalf("runVPSAction returned error: %v", err)
	}
	if got.Action != "cleanup" || got.CleanupLevel != 3 || got.Confirmation != "CLEAN" {
		t.Fatalf("unexpected VPS action request: %+v", got)
	}
}

func TestRunControlDefaultsTypeToAll(t *testing.T) {
	var gotReq ControlRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/deployments/dep-123/actions/process" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"action":"restart","results":[]}`))
	}))
	defer server.Close()

	client := processTestClient(server.URL)
	if err := Run(client, []string{"control", "dep-123", "restart"}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if gotReq.Type != "all" {
		t.Fatalf("expected default type 'all', got %q", gotReq.Type)
	}
	if gotReq.Action != "restart" {
		t.Fatalf("expected action restart, got %q", gotReq.Action)
	}
}

func TestParseAndValidateControlArgs(t *testing.T) {
	parsed, showHelp := parseControlArgs([]string{"dep-2", "stop", "resource", "postgres", "--json"})
	if showHelp || parsed.deploymentID != "dep-2" || parsed.request.Action != "stop" || parsed.request.Type != "resource" || parsed.request.Name != "postgres" || !parsed.jsonOutput {
		t.Fatalf("unexpected parsed args: %+v help=%v", parsed, showHelp)
	}
	if err := validateControlArgs(&parsed); err != nil {
		t.Fatalf("valid args rejected: %v", err)
	}
	missingType, _ := parseControlArgs([]string{"dep-2", "restart"})
	if err := validateControlArgs(&missingType); err != nil || missingType.request.Type != "all" {
		t.Fatalf("type default not applied: request=%+v err=%v", missingType.request, err)
	}
	invalid, _ := parseControlArgs([]string{"dep-2", "delete"})
	if err := validateControlArgs(&invalid); err == nil || !strings.Contains(err.Error(), "action must be") {
		t.Fatalf("invalid action was not rejected: %v", err)
	}
}

func TestWriteControlOutcomeIncludesPerProcessResults(t *testing.T) {
	var output strings.Builder
	writeControlOutcome(&output, ControlResponse{
		Success: true, Action: "restart",
		Results: []ProcessResult{
			{Type: "scenario", Name: "app", Success: true, PID: 42},
			{Type: "resource", Name: "postgres", Error: "permission denied"},
		},
	})
	for _, want := range []string{"Action 'restart' completed successfully", "[OK] scenario app (PID: 42)", "[FAILED] resource postgres - permission denied"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("control output does not contain %q:\n%s", want, output.String())
		}
	}
}
