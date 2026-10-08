package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	heartbeatv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/heartbeat"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestNewApp_APIPortEnvVars_DoNotIncludeGenericAPIPort(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	portEnvVars := app.core.APIBaseOptions().PortEnvVars
	if len(portEnvVars) == 0 {
		t.Fatal("expected APIPortEnvVars to be configured")
	}
	if portEnvVars[0] != "PROMPT_MANAGER_API_PORT" {
		t.Fatalf("first APIPortEnvVar = %q, want %q", portEnvVars[0], "PROMPT_MANAGER_API_PORT")
	}
	for _, key := range portEnvVars {
		if key == "API_PORT" {
			t.Fatalf("unexpected generic API_PORT in APIPortEnvVars: %v", portEnvVars)
		}
	}
}

func TestSkillHelpDoesNotRequireAPI(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	output := captureStdout(t, func() {
		if err := app.Run([]string{"skill", "--help"}); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	})
	if !strings.Contains(output, "prompt-manager skill <subcommand> [args]") {
		t.Fatalf("expected skill help output, got %q", output)
	}
}

func TestBugCapturePreservesRequiredAttributionAlongsideInvocationHeaders(t *testing.T) {
	var attribution, invocation string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","readiness":true}`))
		case "/vrooli.prompt_manager.v1.heartbeat.HeartbeatService/CaptureBug":
			attribution = r.Header.Get("X-Vrooli-Attribution")
			invocation = r.Header.Get("X-Vrooli-Invocation-ID")
			w.Header().Set("Content-Type", "application/proto")
			data, _ := structpb.NewValue(map[string]any{"disposition": "published", "knowledge": map[string]any{"id": "knw-test", "topic": "bug-inbox/code-defect/test"}})
			encoded, _ := proto.Marshal(&heartbeatv1.JsonResponse{Data: data})
			_, _ = w.Write(encoded)
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	app.core.Config.APIBase = server.URL
	if err := app.Run([]string{"team", "bug-capture", "scenario-qa", "--title=test", "--signal-type=code-defect", "--severity=minor", "--repro=start", "--expected=publish", "--actual=dropped", "--description=header regression"}); err != nil {
		t.Fatalf("bug-capture: %v", err)
	}
	if attribution == "" {
		t.Fatal("bug-capture dropped X-Vrooli-Attribution")
	}
	decoded, err := base64.StdEncoding.DecodeString(attribution)
	if err != nil {
		t.Fatalf("decode bug-capture attribution: %v", err)
	}
	var info map[string]any
	if err := json.Unmarshal(decoded, &info); err != nil {
		t.Fatalf("unmarshal bug-capture attribution: %v", err)
	}
	if info["kind"] != "writer-skill" || info["source_skill_id"] != "report-bug" || info["team_id"] != "scenario-qa" {
		t.Fatalf("bug-capture attribution = %v, want writer-skill/report-bug/scenario-qa", info)
	}
	if invocation == "" {
		t.Fatal("bug-capture dropped X-Vrooli-Invocation-ID")
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = old
	}()

	done := make(chan string, 1)
	go func() {
		var builder strings.Builder
		_, _ = io.Copy(&builder, r)
		done <- builder.String()
	}()

	fn()
	_ = w.Close()
	return <-done
}
