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
			case strings.HasSuffix(path, "claude-stop-hook.sh"):
				return []byte("hook"), nil
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
		case strings.HasSuffix(path, "claude-stop-hook.sh"):
			return []byte("hook"), nil
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

func TestRegisterSkipsWhenResourceCLIIsAbsent(t *testing.T) {
	r := testRegistrar()
	r.lookup = func(string) (string, error) { return "", exec.ErrNotFound }
	if err := r.register(nil); err != nil {
		t.Fatalf("register: %v", err)
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
