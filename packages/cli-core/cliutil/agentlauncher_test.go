package cliutil

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/envkit-go"
	"github.com/vrooli/vrooli/packages/proto/descriptorimage"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
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
	source, err := descriptorimage.New(descriptorimage.Config{DescriptorPath: filepath.Join("..", "..", "proto", "gen", "descriptor", "image.binpb")})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := source.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	descriptor, err := snapshot.Files.FindDescriptorByName("agent_manager.v1.AttachRunRequest")
	if err != nil {
		t.Fatal(err)
	}
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
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read attach: %v", err)
				http.Error(w, "failed to read attach request", http.StatusBadRequest)
				return
			}
			// Match Agent Manager's strict decoding of the published request.
			// A permissive map-only fixture concealed unsupported launch fields.
			request := dynamicpb.NewMessage(descriptor.(protoreflect.MessageDescriptor))
			if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(body, request); err != nil {
				t.Errorf("attach does not match the API contract: %v", err)
				http.Error(w, "invalid attach request", http.StatusBadRequest)
				return
			}
			if err := json.Unmarshal(body, &attachBody); err != nil {
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
	err = LaunchCodingAgent(context.Background(), AgentLaunchRequest{
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
	// The attach carries only fields admitted by the API. Tree and containment
	// metadata remain on the editor lease, independently of attachment.
	if !reflect.DeepEqual(attachBody, map[string]string{"task_id": "safe-task-id", "harness_kind": "codex", "harness_session_id": attachBody["harness_session_id"]}) || attachBody["harness_session_id"] == "" {
		t.Fatalf("attach body = %#v", attachBody)
	}
	if childPath != "/safe/fixture/codex" || !reflect.DeepEqual(childArgs, []string{"--unchanged", "literal value"}) {
		t.Fatalf("child invocation = %q %q", childPath, childArgs)
	}
	if got := environmentValue(childEnvironment, EnvIdentityToken); got != "safe-token" {
		t.Fatalf("child token = %q, want safe-token", got)
	}
	// Beyond the base environment the launcher adds exactly the token, the
	// toolchain floor and the session marker; nothing else may leak into the
	// child. The marker is what tells a later launcher stage that this
	// session is already open, so it records no second lease and mints no
	// second scope.
	if got := environmentValue(childEnvironment, AgentSessionEnv); !strings.HasPrefix(got, "vrooli-agent-") {
		t.Fatalf("child session marker = %q, want the session scope", got)
	}
	for _, entry := range childEnvironment {
		key, _, _ := strings.Cut(entry, "=")
		if environmentValue(baseEnvironment, key) != "" || key == EnvIdentityToken {
			continue
		}
		switch key {
		case AgentSessionEnv:
		case LaunchRepoRootEnv, LaunchSourceRootEnv:
		case envkit.GoFlagsKey, envkit.GoMaxProcsKey, envkit.PnpmChildConcurrencyKey, envkit.PnpmWorkspaceConcurrencyKey:
		default:
			t.Fatalf("child environment gained %q beyond the token and the toolchain floor", entry)
		}
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

func jsonDecode(r *http.Request, target *map[string]string) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}

func TestProtectedLaunchBrokerRefusalBeforeAnyBootstrapOrChild(t *testing.T) {
	testProtectedSessionBroker(t, LaunchCodingAgentResult)
}
func TestProtectedCommandSessionBrokerRefusalBeforeAnyBootstrapOrChild(t *testing.T) {
	testProtectedSessionBroker(t, func(ctx context.Context, r AgentLaunchRequest) (LaunchResult, error) {
		r.Args = append([]string{"fixture-command"}, r.Args...)
		return LaunchCommandSessionResult(ctx, r)
	})
}
func testProtectedSessionBroker(t *testing.T, launch func(context.Context, AgentLaunchRequest) (LaunchResult, error)) {
	for _, mode := range []string{"refused", "unavailable", "changed", "same", "postattach refused", "postattach forwarding refusal"} {
		t.Run(mode, func(t *testing.T) {
			// Namespace fixed before the first lookup; never changed to evade refusal.
			namespace, nsErr := os.MkdirTemp("", "protected-launch-")
			if nsErr != nil {
				t.Fatal(nsErr)
			}
			defer os.RemoveAll(namespace)
			t.Setenv("TMPDIR", namespace)
			t.Setenv(AgentSessionEnv, "")
			t.Setenv(EnvIdentityToken, "old-private-fixture-leaf")
			if strings.HasPrefix(mode, "postattach") {
				t.Setenv(EnvIdentityToken, "")
			}
			socket := AgentCredentialSocket()
			if os.MkdirAll(filepath.Dir(socket), 0700) != nil {
				t.Fatal("broker directory")
			}
			calls, attach, detach, child := 0, 0, 0, 0
			if mode != "unavailable" {
				listener, err := net.Listen("unix", socket)
				if err != nil {
					t.Fatal(err)
				}
				broker := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					if mode == "refused" || (mode == "postattach refused" && calls > 1) || (mode == "postattach forwarding refusal" && calls > 2) {
						w.WriteHeader(403)
						_, _ = io.WriteString(w, `{"code":"revoked"}`)
						return
					}
					if strings.HasPrefix(mode, "postattach") {
						w.WriteHeader(204)
						return
					}
					leaf := "old-private-fixture-leaf"
					if mode == "changed" && calls > 1 {
						leaf = "changed-private-fixture-leaf"
					}
					_ = json.NewEncoder(w).Encode(map[string]string{"token": leaf})
				})}
				go broker.Serve(listener)
				defer broker.Close()
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "/attach") {
					attach++
					w.WriteHeader(201)
					_, _ = io.WriteString(w, `{"run":{"id":"owned-disposable-attempt"},"identity_token":"new-private-fixture-leaf"}`)
					return
				}
				if strings.HasSuffix(r.URL.Path, "/detach") {
					detach++
					w.WriteHeader(200)
					return
				}
				t.Error("unexpected wire")
			}))
			defer server.Close()
			transport := http.RoundTripper(&http.Transport{})
			if mode == "postattach forwarding refusal" {
				transport = identityForwardingTransport{next: transport}
			}
			_, err := launch(context.Background(), AgentLaunchRequest{Agent: "codex", RequireNativeIdentity: true, APIBase: server.URL, HTTPClient: &http.Client{Transport: transport},
				LookPath: func(string) (string, error) { return "/disposable/codex", nil },
				RunChild: func(_ context.Context, _ string, _ []string, env []string, _ io.Reader, _, _ io.Writer) error {
					child++
					if environmentValue(env, EnvIdentityToken) != "old-private-fixture-leaf" {
						t.Error("custody changed")
					}
					return nil
				},
			})
			if mode == "same" {
				if err != nil || child != 1 || attach != 0 || detach != 0 {
					t.Fatal("existing protected custody", err, attach, detach, child)
				}
				return
			}
			if err == nil || child != 0 {
				t.Fatal("protected refusal launched child", err, child)
			}
			if mode == "postattach refused" {
				if attach != 1 || detach != 0 || !strings.Contains(err.Error(), "cleanup unconfirmed") {
					t.Fatal("denied owned cleanup not reported", err, attach, detach)
				}
			} else if mode == "postattach forwarding refusal" {
				if attach != 1 || detach != 0 || !strings.Contains(err.Error(), "owned attachment owned-disposable-attempt cleanup unconfirmed") {
					t.Fatal("protected cleanup refusal hidden", err, attach, detach)
				}
			} else if attach != 0 || detach != 0 {
				t.Fatal("denial bootstrap effects", attach, detach)
			}
			if mode == "refused" || strings.HasPrefix(mode, "postattach") {
				var denied *AgentAuthorizationError
				if !errors.As(err, &denied) || denied.Code != "revoked" {
					t.Fatal("exact broker refusal lost", err)
				}
			}
		})
	}
}

func TestProtectedLaunchNoAnonymousFallback(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	t.Setenv(AgentSessionEnv, "")
	t.Setenv(EnvIdentityToken, "")
	child := false
	_, err := LaunchCodingAgentResult(context.Background(), AgentLaunchRequest{Agent: "codex", RequireNativeIdentity: true, APIBase: "http://127.0.0.1:1", LookPath: func(string) (string, error) { return "/disposable/codex", nil }, RunChild: func(context.Context, string, []string, []string, io.Reader, io.Writer, io.Writer) error {
		child = true
		return nil
	}})
	if err == nil || child {
		t.Fatal("protected startup fell back", err, child)
	}
}

func TestProtectedInformationalInvocationRequiresExistingCustody(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	t.Setenv(AgentSessionEnv, "")
	t.Setenv(EnvIdentityToken, "")
	ran := false
	_, err := LaunchCodingAgentResult(context.Background(), AgentLaunchRequest{Agent: "codex", RequireNativeIdentity: true, Args: []string{"--version"}, LookPath: func(string) (string, error) { return "/disposable/codex", nil }, RunChild: func(context.Context, string, []string, []string, io.Reader, io.Writer, io.Writer) error {
		ran = true
		return nil
	}})
	if err == nil || ran {
		t.Fatal("protected informational invocation escaped custody", err, ran)
	}
}

func TestProtectedCommandSessionAdoptsInheritedCustody(t *testing.T) {
	namespace, err := os.MkdirTemp("", "cmd-fixture-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(namespace)
	t.Setenv("TMPDIR", namespace)
	t.Setenv(EnvIdentityToken, "synthetic-inherited-leaf")
	t.Setenv(AgentSessionEnv, "existing-session")
	calls := 0
	result, err := LaunchCommandSessionResult(context.Background(), AgentLaunchRequest{
		Args: []string{"fixture-command", "--help"}, APIBase: "http://127.0.0.1:1",
		LookPath: func(string) (string, error) { return "/synthetic/command", nil },
		RunChild: func(_ context.Context, _ string, args, environment []string, _ io.Reader, _, _ io.Writer) error {
			calls++
			if len(args) != 1 || args[0] != "--help" || environmentValue(environment, EnvIdentityToken) != "synthetic-inherited-leaf" {
				t.Error("inherited command custody/argv")
			}
			return nil
		},
	})
	if err != nil || calls != 1 || result.Tier != "tier-1" || result.AttachFailure != "" || result.Scope != "existing-session" {
		t.Fatal("protected inherited classification", result, err, calls)
	}
}

func TestProtectedCommandSessionCleanupRefusesRedirect(t *testing.T) {
	namespace, err := os.MkdirTemp("", "cleanup-fixture-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(namespace)
	t.Setenv("TMPDIR", namespace)
	t.Setenv(EnvIdentityToken, "")
	secondRequests := 0
	second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { secondRequests++; w.WriteHeader(200) }))
	defer second.Close()
	for _, status := range []int{307, 308} {
		firstRequests := 0
		first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			firstRequests++
			if r.Header.Get(HeaderAgentIdentityToken) != "synthetic-cleanup-leaf" {
				t.Error("missing held cleanup identity")
			}
			w.Header().Set("Location", second.URL+"/unexpected")
			w.WriteHeader(status)
		}))
		err := detachCodingAgent(context.Background(), AgentLaunchRequest{RequireNativeIdentity: true, APIBase: first.URL}, "synthetic-owned-id", "synthetic-cleanup-leaf")
		first.Close()
		if err == nil || !strings.Contains(err.Error(), "protected cleanup redirect refused") || firstRequests != 1 || secondRequests != 0 {
			t.Fatal("redirect leaked cleanup", err, firstRequests, secondRequests)
		}
	}
}
