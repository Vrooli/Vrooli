package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRegisterReconcilesBothHooks(t *testing.T) {
	var calls [][]string
	stdout := &bytes.Buffer{}
	r := &registrar{
		lookup: func(string) (string, error) { return "/bin/resource-claude-code", nil },
		run: func(_ context.Context, name string, args ...string) ([]byte, error) {
			calls = append(calls, append([]string{name}, args...))
			return []byte(`{"status":"applied"}`), nil
		},
		sleep:       func(time.Duration) {},
		stdout:      stdout,
		stderr:      &bytes.Buffer{},
		getenv:      func(string) string { return "" },
		userHomeDir: func() (string, error) { return "/home/test", nil },
		workingDir:  func() (string, error) { return "/repo/scenarios/web-console", nil },
		readFile: func(path string) ([]byte, error) {
			switch {
			case strings.HasSuffix(path, "start-api.json"):
				return []byte(`{"port":19777}`), nil
			case strings.HasSuffix(path, "hook-token.txt"):
				return []byte("secret-token\n"), nil
			default:
				return nil, errors.New("unexpected path")
			}
		},
		maxAttempts: 5,
	}

	if err := r.register(nil); err != nil {
		t.Fatalf("register: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("calls = %d, want 2", len(calls))
	}
	if got := calls[0][1:7]; !reflect.DeepEqual(got, []string{"hooks", "reconcile", "--event", stopEvent, "--id", stopID}) {
		t.Fatalf("stop call prefix = %#v", got)
	}
	if got := calls[1][1:7]; !reflect.DeepEqual(got, []string{"hooks", "reconcile", "--event", promptEvent, "--id", promptID}) {
		t.Fatalf("prompt call prefix = %#v", got)
	}
	for _, call := range calls {
		payloadIndex := indexOf(call, "--hook-json")
		if payloadIndex < 0 || payloadIndex+1 >= len(call) {
			t.Fatalf("missing hook payload in %#v", call)
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(call[payloadIndex+1]), &payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		command := payload["command"].(string)
		if !strings.Contains(command, "'secret-token'") || !strings.Contains(command, "localhost:19777") {
			t.Fatalf("unsafe or incomplete command payload: %s", command)
		}
	}
}

func TestRegisterRetriesTokenWithoutSleepingInTest(t *testing.T) {
	reads := 0
	sleeps := 0
	r := testRegistrar()
	r.maxAttempts = 3
	r.sleep = func(time.Duration) { sleeps++ }
	r.readFile = func(path string) ([]byte, error) {
		switch {
		case strings.HasSuffix(path, "start-api.json"):
			return []byte(`{"port":19777}`), nil
		case strings.HasSuffix(path, "hook-token.txt"):
			reads++
			if reads < 3 {
				return nil, errors.New("not ready")
			}
			return []byte("token"), nil
		default:
			return nil, errors.New("unexpected path")
		}
	}
	if err := r.register(nil); err != nil {
		t.Fatalf("register: %v", err)
	}
	if reads != 3 || sleeps != 2 {
		t.Fatalf("reads=%d sleeps=%d, want 3 and 2", reads, sleeps)
	}
}

func TestRegisterReportsWhenResourceCLIIsAbsent(t *testing.T) {
	// This used to return nil. Reporting success when the tool that performs
	// registration is missing meant hooks were never reconciled and nothing
	// ever said so — a stale hook survived every restart, message capture
	// stayed broken, and the only symptom was an empty Messages view.
	r := testRegistrar()
	r.lookup = func(string) (string, error) { return "", exec.ErrNotFound }
	err := r.register(nil)
	if err == nil {
		t.Fatal("register must report a missing resource CLI, not claim success")
	}
	if !errors.Is(err, errResourceCLIMissing) {
		t.Fatalf("register error = %v, want it to wrap errResourceCLIMissing", err)
	}
}

func TestHookTokenPathHonorsExplicitOverride(t *testing.T) {
	r := testRegistrar()
	r.getenv = func(key string) string {
		if key == "WC_HOOK_TOKEN_PATH" {
			return "/custom/hook-token.txt"
		}
		return ""
	}
	got, err := r.hookTokenPath()
	if err != nil {
		t.Fatalf("hookTokenPath: %v", err)
	}
	if got != "/custom/hook-token.txt" {
		t.Fatalf("hookTokenPath = %q, want the override", got)
	}
}

func TestHookTokenPathResolvesThroughScenarioStorage(t *testing.T) {
	// The registrar and the API must agree on one path. They previously did
	// not: the API wrote to the scenario state class while this read
	// $XDG_STATE_HOME, and only a years-old file holding the same value kept
	// hooks working at all.
	r := testRegistrar()
	r.getenv = func(string) string { return "" }
	got, err := r.hookTokenPath()
	if err != nil {
		t.Fatalf("hookTokenPath: %v", err)
	}
	if filepath.Base(got) != hookTokenFileName {
		t.Fatalf("hookTokenPath = %q, want it to end in %q", got, hookTokenFileName)
	}
	if !strings.Contains(got, "web-console") {
		t.Fatalf("hookTokenPath = %q, want it scoped to the web-console scenario", got)
	}
}

func TestHookTokenRejectsAnEmptyFile(t *testing.T) {
	r := testRegistrar()
	r.getenv = func(key string) string {
		if key == "WC_HOOK_TOKEN_PATH" {
			return "/custom/hook-token.txt"
		}
		return ""
	}
	r.readFile = func(string) ([]byte, error) { return []byte("   \n"), nil }
	if _, err := r.hookToken(); err == nil {
		t.Fatal("an empty token file must be an error, not an empty token")
	}
}

func TestAPIPortFallsBackToEnvironment(t *testing.T) {
	r := testRegistrar()
	r.readFile = func(string) ([]byte, error) { return nil, errors.New("missing") }
	r.getenv = func(key string) string {
		if key == "API_PORT" {
			return "19876"
		}
		return ""
	}
	port, err := r.apiPort()
	if err != nil || port != 19876 {
		t.Fatalf("apiPort = %d, %v", port, err)
	}
}

func TestShellQuoteEscapesSingleQuotes(t *testing.T) {
	if got, want := shellQuote("a'b"), `'a'\''b'`; got != want {
		t.Fatalf("shellQuote = %q, want %q", got, want)
	}
}

func testRegistrar() *registrar {
	return &registrar{
		lookup: func(string) (string, error) { return "/bin/resource-claude-code", nil },
		run: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte(`{"status":"unchanged"}`), nil
		},
		sleep:       func(time.Duration) {},
		stdout:      &bytes.Buffer{},
		stderr:      &bytes.Buffer{},
		getenv:      func(string) string { return "" },
		userHomeDir: func() (string, error) { return "/home/test", nil },
		workingDir:  func() (string, error) { return filepath.Join("repo", "scenarios", "web-console"), nil },
		readFile:    func(string) ([]byte, error) { return nil, errors.New("missing") },
		maxAttempts: 1,
	}
}

func indexOf(values []string, wanted string) int {
	for index, value := range values {
		if value == wanted {
			return index
		}
	}
	return -1
}
