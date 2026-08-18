package control

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	testutil "github.com/vrooli/cli-core/cliapptest"
)

func TestRegisteredRESTPathsUseScenarioRelativePaths(t *testing.T) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(source), "register.go"))
	if err != nil {
		t.Fatalf("read register.go: %v", err)
	}

	// ScenarioApp.Request adds /api/v1. A fully-qualified path here creates the
	// exact /api/v1/api/v1 regression that first-contact testing exposed.
	requestLiteral := regexp.MustCompile(`core\.Request\([^\n]*,\s*"([^"]+)"`)
	for _, match := range requestLiteral.FindAllStringSubmatch(string(data), -1) {
		path := match[1]
		if strings.HasPrefix(path, "/api/v1") {
			t.Fatalf("Request path %q is already API-prefixed; pass a scenario-relative path", path)
		}
		if !strings.HasPrefix(path, "/") {
			t.Fatalf("Request path %q must begin with /", path)
		}
	}
}

func TestConnectWatchReprobesUntilOnboardingIsReady(t *testing.T) {
	var requests atomic.Int32
	server := testutil.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/devices/connect" {
			t.Fatalf("request = %s %s, want POST /api/v1/devices/connect", r.Method, r.URL.Path)
		}
		if n := requests.Add(1); n == 1 {
			_, _ = w.Write([]byte(`{"kind":"android","rungs":[{"id":"host-node","status":"available","next_action":"No action required."},{"id":"android-sdk","status":"available","next_action":"No action required."},{"id":"usb-debugging","status":"unavailable","next_action":"Connect the phone."}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"kind":"android","rungs":[{"id":"host-node","status":"available","next_action":"No action required."},{"id":"android-sdk","status":"available","next_action":"No action required."},{"id":"usb-debugging","status":"available","next_action":"Device is authorized."}]}`))
	}))
	_ = server

	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name: "device-control-test", Version: "test", DefaultAPIBase: server.URL, AllowAnonymous: true,
	})
	if err != nil {
		t.Fatalf("NewStandardScenarioApp: %v", err)
	}
	var stdout bytes.Buffer
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "kind", Required: true},
		{Name: "watch", Bool: true},
		{Name: "watch-seconds", Default: "1"},
	}}
	ctx, err := cliapp.NewTestRunContextFromArgs(schema, []string{"--kind", "android", "--watch", "--json"}, core, &stdout, &stdout)
	if err != nil {
		t.Fatalf("parse context: %v", err)
	}
	if err := connect(ctx, core); err != nil {
		t.Fatalf("connect: %v", err)
	}
	var report struct {
		Rungs []struct {
			Status string `json:"status"`
		} `json:"rungs"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode output %q: %v", stdout.String(), err)
	}
	if len(report.Rungs) != 3 || report.Rungs[2].Status != "available" {
		t.Fatalf("report = %+v, want all three rungs available", report)
	}
	if requests.Load() < 2 {
		t.Fatalf("requests = %d, want at least two live probes", requests.Load())
	}
	if !strings.Contains(stdout.String(), "transitions") {
		t.Fatalf("watch output = %q, want transition history", stdout.String())
	}
}

func TestConnectRejectsInvalidWatchWindow(t *testing.T) {
	core := testutil.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"rungs":[]}`))
	}))
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "kind"}, {Name: "watch", Bool: true}, {Name: "watch-seconds", Default: "0"}}},
		Flags:  map[string]string{"kind": "android", "watch-seconds": "0"}, BoolFlags: map[string]bool{"watch": true},
	})
	if err := connect(ctx, core); err == nil || !strings.Contains(err.Error(), "watch-seconds must be a positive integer") {
		t.Fatalf("connect error = %v, want invalid watch window", err)
	}
}

func TestAgentStartSerializesBooleanFlags(t *testing.T) {
	server := testutil.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/agents/start" {
			t.Fatalf("request = %s %s, want POST /api/v1/agents/start", r.Method, r.URL.Path)
		}
		var request struct {
			Goal           string `json:"goal"`
			DeviceID       string `json:"device_id"`
			SkillAvailable bool   `json:"skill_available"`
			DryRun         bool   `json:"dry_run"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.Goal != "observe the TV" || request.DeviceID != "tv-1" || !request.SkillAvailable || !request.DryRun {
			t.Fatalf("request = %+v, want both boolean flags true", request)
		}
		_, _ = w.Write([]byte(`{"id":"agent-1","state":"completed"}`))
	}))
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name: "device-control-test", Version: "test", DefaultAPIBase: server.URL, AllowAnonymous: true,
	})
	if err != nil {
		t.Fatalf("NewStandardScenarioApp: %v", err)
	}
	var stdout bytes.Buffer
	var start cliapp.Command
	for _, candidate := range AgentGroup(core).Subcommands {
		if candidate.Name == "start" {
			start = candidate
			break
		}
	}
	if start.RunCtx == nil {
		t.Fatal("agent start command has no RunCtx handler")
	}
	ctx, err := cliapp.NewTestRunContextFromArgs(start.Args, []string{
		"--goal", "observe the TV", "--device", "tv-1", "--skill-available", "--dry-run", "--json",
	}, core, &stdout, &stdout)
	if err != nil {
		t.Fatalf("parse context: %v", err)
	}
	if err := start.RunCtx(ctx); err != nil {
		t.Fatalf("agent start: %v", err)
	}
}

func TestReadPairingPINReadsOneLineWithoutWaitingForEOF(t *testing.T) {
	var prompt bytes.Buffer
	pin, err := readPairingPIN(strings.NewReader("123456\nadditional input"), &prompt)
	if err != nil {
		t.Fatalf("readPairingPIN: %v", err)
	}
	if pin != "123456" {
		t.Fatalf("pin = %q, want one six-character line", pin)
	}
	if !strings.Contains(prompt.String(), "six-character hexadecimal pairing code") {
		t.Fatalf("prompt = %q, want an operator prompt", prompt.String())
	}
}

func TestPairInteractiveStartsHandshakeBeforeReadingPIN(t *testing.T) {
	server := testutil.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		switch r.URL.Path {
		case "/api/v1/devices/tv-1/pair/start":
			_, _ = w.Write([]byte(`{"pairing_id":"session-1"}`))
		case "/api/v1/devices/tv-1/pair/complete":
			var request struct {
				PairingID string `json:"pairing_id"`
				PIN       string `json:"pin"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode completion: %v", err)
			}
			if request.PairingID != "session-1" || request.PIN != "835B64" {
				t.Fatalf("completion = %+v, want session and PIN", request)
			}
			_, _ = w.Write([]byte(`{"paired":true,"device_id":"tv-1","outcome":"paired","transport":"android-tv-remote","detail":"certificate stored"}`))
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name: "device-control-test", Version: "test", DefaultAPIBase: server.URL, AllowAnonymous: true,
	})
	if err != nil {
		t.Fatalf("NewStandardScenarioApp: %v", err)
	}
	var stdout bytes.Buffer
	ctx, err := cliapp.NewTestRunContextFromArgs(cliapp.ArgSchema{}, []string{"--json"}, core, &stdout, &stdout)
	if err != nil {
		t.Fatalf("parse context: %v", err)
	}
	var prompt bytes.Buffer
	if err := pairInteractive(ctx, core, "tv-1", strings.NewReader("835B64\n"), &prompt); err != nil {
		t.Fatalf("pairInteractive: %v", err)
	}
	if !strings.Contains(prompt.String(), "six-character hexadecimal pairing code") {
		t.Fatalf("prompt = %q, want operator prompt after handshake start", prompt.String())
	}
	if !strings.Contains(stdout.String(), "\"paired\": true") {
		t.Fatalf("output = %q, want pairing result", stdout.String())
	}
}
