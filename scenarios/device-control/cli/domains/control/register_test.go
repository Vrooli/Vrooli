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
