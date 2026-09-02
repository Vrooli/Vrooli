package cliutil

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestLaunchCodingAgentFallsBackWhenAgentManagerUnavailable(t *testing.T) {
	// A token in the ambient environment would make the launcher adopt it and
	// skip attaching, so pin it empty to keep this test about the attach path.
	t.Setenv(AgentManagerIdentityTokenEnv, "")
	var ran bool
	err := LaunchCodingAgent(context.Background(), AgentLaunchRequest{
		Agent:   "codex",
		APIBase: "http://127.0.0.1:1",
		Args:    []string{"--safe-fixture", "value with spaces"},
		LookPath: func(name string) (string, error) {
			if name != "codex" {
				t.Fatalf("looked up %q, want codex", name)
			}
			return "/safe/fixture/codex", nil
		},
		RunChild: func(_ context.Context, path string, args, environment []string, _ io.Reader, _, _ io.Writer) error {
			ran = true
			if path != "/safe/fixture/codex" || !reflect.DeepEqual(args, []string{"--safe-fixture", "value with spaces"}) {
				t.Fatalf("child invocation = %q %q", path, args)
			}
			if token := environmentValue(environment, EnvIdentityToken); token != "" {
				t.Fatalf("fallback unexpectedly added identity token %q", token)
			}
			return nil
		},
		AttachTimeout: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("LaunchCodingAgent() error = %v", err)
	}
	if !ran {
		t.Fatal("child did not start after unavailable Agent Manager")
	}
}

func TestLaunchCodingAgentTreatsHTTP500AndHangAsFallback(t *testing.T) {
	// A token in the ambient environment would make the launcher adopt it and
	// skip attaching, so pin it empty to keep this test about the attach path.
	t.Setenv(AgentManagerIdentityTokenEnv, "")
	for _, tc := range []struct {
		name    string
		handler http.Handler
	}{
		{name: "http-500", handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "safe fixture", http.StatusInternalServerError)
		})},
		{name: "hang-timeout", handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { time.Sleep(200 * time.Millisecond) })},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer func() {
				// A deliberately hung fixture may keep the server handler alive
				// after the client-side timeout. Close active fixture connections
				// before closing the server so the test itself cannot hang.
				server.CloseClientConnections()
				server.Close()
			}()
			started := time.Now()
			ran := false
			err := LaunchCodingAgent(context.Background(), AgentLaunchRequest{
				Agent:         "claude-code",
				APIBase:       server.URL,
				AttachTimeout: 40 * time.Millisecond,
				LookPath: func(string) (string, error) {
					return "/safe/fixture/claude", nil
				},
				RunChild: func(_ context.Context, _ string, _ []string, environment []string, _ io.Reader, _, _ io.Writer) error {
					ran = true
					if token := environmentValue(environment, EnvIdentityToken); token != "" {
						t.Fatalf("failed attach added token %q", token)
					}
					return nil
				},
			})
			if err != nil {
				t.Fatalf("LaunchCodingAgent() error = %v", err)
			}
			if !ran {
				t.Fatal("child did not start")
			}
			if elapsed := time.Since(started); elapsed > 300*time.Millisecond {
				t.Fatalf("fallback took %s; want bounded local degradation", elapsed)
			}
		})
	}
}

func TestLaunchCodingAgentReportsAttachmentFailureClass(t *testing.T) {
	t.Setenv(AgentManagerIdentityTokenEnv, "")
	tests := []struct {
		name   string
		base   string
		handle http.Handler
		want   string
	}{
		{name: "bad-url", base: "://bad", want: "bad base URL"},
		{name: "unreachable", base: "http://127.0.0.1:1", want: "agent-manager unreachable"},
		{name: "non-2xx", handle: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) }), want: "non-2xx response"},
		{name: "malformed", handle: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, `{}`) }), want: "malformed response"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := tt.base
			var server *httptest.Server
			if tt.handle != nil {
				server = httptest.NewServer(tt.handle)
				defer server.Close()
				base = server.URL
			}
			result, err := LaunchCodingAgentResult(context.Background(), AgentLaunchRequest{
				Agent:         "codex",
				APIBase:       base,
				AttachTimeout: 50 * time.Millisecond,
				LookPath:      func(string) (string, error) { return "/safe/fixture/codex", nil },
				RunChild:      func(context.Context, string, []string, []string, io.Reader, io.Writer, io.Writer) error { return nil },
			})
			if err != nil {
				t.Fatalf("LaunchCodingAgentResult() error = %v", err)
			}
			if result.Tier != "tier-4" || result.AttachFailure != tt.want {
				t.Fatalf("result = %#v, want tier-4/%q", result, tt.want)
			}
		})
	}
}

func TestLaunchCodingAgentAttachesAddsOnlyTokenAndDetaches(t *testing.T) {
	// A token in the ambient environment would make the launcher adopt it and
	// skip attaching, so pin it empty to keep this test about the attach path.
	t.Setenv(AgentManagerIdentityTokenEnv, "")
	var mu sync.Mutex
	var attachBody map[string]string
	var childPath string
	var childArgs []string
	var childEnvironment []string
	var detachPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.URL.Path {
		case "/api/v1/runs/attach":
			if r.Method != http.MethodPost {
				t.Errorf("attach method = %s", r.Method)
			}
			if err := jsonDecode(r, &attachBody); err != nil {
				t.Errorf("decode attach: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"run":{"id":"safe-run-id"},"identity_token":"safe-token"}`)
		case "/api/v1/runs/safe-run-id/detach":
			detachPath = r.URL.Path
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	baseEnvironment := []string{"PATH=/safe", "VROOLI_AGENT_IDENTITY_TOKEN=old-token", "SAFE_VALUE=keep"}
	err := LaunchCodingAgent(context.Background(), AgentLaunchRequest{
		Agent:       "codex",
		TaskID:      "safe-task-id",
		APIBase:     server.URL,
		Args:        []string{"--unchanged", "literal value"},
		Environment: baseEnvironment,
		LookPath: func(name string) (string, error) {
			if name != "codex" {
				t.Fatalf("looked up %q, want codex", name)
			}
			return "/safe/fixture/codex", nil
		},
		RunChild: func(_ context.Context, path string, args, environment []string, _ io.Reader, _, _ io.Writer) error {
			childPath = path
			childArgs = append([]string(nil), args...)
			childEnvironment = append([]string(nil), environment...)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("LaunchCodingAgent() error = %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if !reflect.DeepEqual(attachBody, map[string]string{"task_id": "safe-task-id", "harness_kind": "codex", "harness_session_id": attachBody["harness_session_id"]}) || attachBody["harness_session_id"] == "" {
		t.Fatalf("attach body = %#v", attachBody)
	}
	if childPath != "/safe/fixture/codex" || !reflect.DeepEqual(childArgs, []string{"--unchanged", "literal value"}) {
		t.Fatalf("child invocation = %q %q", childPath, childArgs)
	}
	if got := environmentValue(childEnvironment, EnvIdentityToken); got != "safe-token" {
		t.Fatalf("child token = %q, want safe-token", got)
	}
	if len(childEnvironment) != len(baseEnvironment) {
		t.Fatalf("child environment length = %d, want %d", len(childEnvironment), len(baseEnvironment))
	}
	if detachPath != "/api/v1/runs/safe-run-id/detach" {
		t.Fatalf("detach path = %q", detachPath)
	}
}

func TestLaunchCodingAgentReportsAbsentBinaryWithoutHTTPCall(t *testing.T) {
	// A token in the ambient environment would make the launcher adopt it and
	// skip attaching, so pin it empty to keep this test about the attach path.
	t.Setenv(AgentManagerIdentityTokenEnv, "")
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer server.Close()
	err := LaunchCodingAgent(context.Background(), AgentLaunchRequest{
		Agent:   "opencode",
		APIBase: server.URL,
		LookPath: func(string) (string, error) {
			return "", errors.New("safe fixture binary absent")
		},
		RunChild: func(context.Context, string, []string, []string, io.Reader, io.Writer, io.Writer) error {
			t.Fatal("child started despite absent binary")
			return nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "safe fixture binary absent") {
		t.Fatalf("error = %v, want absent-binary error", err)
	}
	if called {
		t.Fatal("attach request sent despite absent binary")
	}
}

func TestLaunchCodingAgentUsesAntigravityBinaryName(t *testing.T) {
	// A token in the ambient environment would make the launcher adopt it and
	// skip attaching, so pin it empty to keep this test about the attach path.
	t.Setenv(AgentManagerIdentityTokenEnv, "")
	ran := false
	err := LaunchCodingAgent(context.Background(), AgentLaunchRequest{
		Agent: "antigravity",
		LookPath: func(name string) (string, error) {
			if name != "agy" {
				t.Fatalf("looked up %q, want agy", name)
			}
			return "/safe/fixture/agy", nil
		},
		RunChild: func(_ context.Context, path string, _ []string, _ []string, _ io.Reader, _, _ io.Writer) error {
			ran = true
			if path != "/safe/fixture/agy" {
				t.Fatalf("child path = %q, want fake agy path", path)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("LaunchCodingAgent() error = %v", err)
	}
	if !ran {
		t.Fatal("antigravity child did not start")
	}
}

func TestChildExitCode(t *testing.T) {
	if got := ChildExitCode(nil); got != 0 {
		t.Fatalf("ChildExitCode(nil) = %d", got)
	}
	if got := ChildExitCode(errors.New("not a child exit")); got != -1 {
		t.Fatalf("ChildExitCode(non-exit) = %d", got)
	}
}

func TestLauncherBaseUsesLocalDefaultWithoutControlPlaneLookup(t *testing.T) {
	t.Setenv(AgentManagerLauncherBaseEnv, "")
	t.Setenv("AGENT_MANAGER_API_URL", "")
	t.Setenv("AGENT_MANAGER_API_PORT", "")
	if got := agentManagerLauncherBase(AgentLaunchRequest{}); got != defaultAgentManagerLauncherBase {
		t.Fatalf("launcher base = %q, want %q", got, defaultAgentManagerLauncherBase)
	}
}

func TestRunNativeChildPreservesRequestedArgv0(t *testing.T) {
	var stdout bytes.Buffer
	run := runNativeChild(AgentLaunchRequest{}, "codex")
	if err := run(context.Background(), "/bin/sh", []string{"-c", "printf %s \"$0\""}, nil, nil, &stdout, io.Discard); err != nil {
		t.Fatalf("runNativeChild() error = %v", err)
	}
	if got := stdout.String(); got != "codex" {
		t.Fatalf("child argv[0] = %q, want codex", got)
	}
}

func environmentValue(environment []string, key string) string {
	prefix := key + "="
	for _, entry := range environment {
		if strings.HasPrefix(entry, prefix) {
			return strings.TrimPrefix(entry, prefix)
		}
	}
	return ""
}

func jsonDecode(r *http.Request, target *map[string]string) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}
