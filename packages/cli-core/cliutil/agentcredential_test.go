//go:build linux

package cliutil

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

func TestDynamicCredentialAdoptionAndRefusalDoNotChangeEnvironment(t *testing.T) {
	directory, err := os.MkdirTemp("", "agc-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(directory) })
	path := filepath.Join(directory, "broker.sock")
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	var state atomic.Int32
	state.Store(http.StatusNoContent)
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if int(state.Load()) == http.StatusOK {
			json.NewEncoder(w).Encode(map[string]string{"token": "approved-current"})
			return
		}
		w.WriteHeader(int(state.Load()))
	})}
	go server.Serve(listener)
	t.Cleanup(func() { server.Close() })
	if token, err := resolveAgentCredential(context.Background(), path, "inherited"); err != nil || token != "inherited" {
		t.Fatal("unbound process lost inherited identity")
	}
	state.Store(http.StatusOK)
	if token, err := resolveAgentCredential(context.Background(), path, "inherited"); err != nil || token != "approved-current" {
		t.Fatal("existing process failed to adopt current consent")
	}
	state.Store(http.StatusForbidden)
	token, err := resolveAgentCredential(context.Background(), path, "inherited")
	if err == nil || token != "" {
		t.Fatal("revocation fell back to older authority")
	}
	var refused *AgentAuthorizationError
	if !errors.As(err, &refused) {
		t.Fatalf("refusal is not reported as an authorization problem: %v", err)
	}
	server.Close()
	if _, err := resolveAgentCredential(context.Background(), path, "inherited"); err == nil {
		t.Fatal("a broken authority silently downgraded to inherited authority")
	}
}

func TestProcessBindingRefusesPIDReuseAndAnotherSession(t *testing.T) {
	root, err := ObserveProcessBinding(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if !ProcessDescendsFrom(os.Getpid(), root) {
		t.Fatal("process did not match its kernel identity")
	}
	root.Start += "-reused"
	if ProcessDescendsFrom(os.Getpid(), root) {
		t.Fatal("reused process identity accepted")
	}
	root, err = ObserveProcessBinding(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	root.Cgroup += "/other"
	if ProcessDescendsFrom(os.Getpid(), root) {
		t.Fatal("different session boundary accepted")
	}
}

func TestAmbientIdentityIsNotSentToRemoteDestinations(t *testing.T) {
	t.Setenv(EnvIdentityToken, "private-agent-proof")
	client := NewHTTPClient(HTTPClientOptions{})
	client.SetInvocationHeaderSource(InvocationHeaders("swarm-manager", "backlog update"))
	req := httptest.NewRequest(http.MethodPost, "https://unrelated.example.test/collect", nil)
	client.ApplyRequestHeaders(req)
	if req.Header.Get(HeaderAgentIdentityToken) != "" {
		t.Fatal("ambient identity left the local scenario transport")
	}
}

type credentialRedirectTransport func(*http.Request) (*http.Response, error)

func (f credentialRedirectTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRedirectCannotCarryAgentIdentityToAnExternalHost(t *testing.T) {
	seen := false
	next := credentialRedirectTransport(func(req *http.Request) (*http.Response, error) {
		if req.URL.Hostname() == "localhost" {
			return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": []string{"https://untrusted.example.test/collect"}}, Body: io.NopCloser(strings.NewReader("")), Request: req}, nil
		}
		seen = true
		if req.Header.Get(HeaderAgentIdentityToken) != "" {
			t.Error("redirect disclosed the credential of the trusted first hop")
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("")), Request: req}, nil
	})
	client := &http.Client{Transport: identityForwardingTransport{next: next}}
	req, _ := http.NewRequest(http.MethodGet, "http://localhost/start", nil)
	// An explicit first-hop identity avoids consulting any live authority.
	req.Header.Set(HeaderAgentIdentityToken, "request-local-proof")
	response, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if !seen {
		t.Fatal("redirect was not exercised")
	}
}

// Containment and owner transport are simulated here. Native child execution,
// attachment, lease lifetime and owned cleanup use their real launcher paths.
// No host scope or live grant is created.
func TestCommandSessionContainsWrapperBeforeAttach(t *testing.T) {
	// Keep Unix socket lookup below the platform path-length limit.
	namespace, err := os.MkdirTemp("", "command-boundary-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(namespace) })
	t.Setenv("TMPDIR", namespace)
	for _, mode := range []string{"contained", "child exit", "containment failure", "no container"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv(AgentSessionEnv, "")
			t.Setenv(EnvIdentityToken, "")
			container := withFakeContainer(t, false)
			container.failSelf = mode == "containment failure"
			if mode == "no container" {
				DefaultSessionContainer = nil
			}
			recorder := &recordingLeaseRecorder{}
			withLeaseRecorder(t, recorder)
			attached, detached := 0, 0
			client := &http.Client{Transport: credentialRedirectTransport(func(r *http.Request) (*http.Response, error) {
				switch r.URL.Path {
				case "/api/v1/runs/attach":
					attached++
					if mode != "no container" && len(container.selfCalls) != 1 {
						t.Error("approval caller was attached before its final containment")
					}
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}
					if len(body) != 3 || body["process_id"] != float64(os.Getpid()) || body["harness_kind"] != "command-session" {
						t.Fatalf("attachment changed caller identity: %v", body)
					}
					return &http.Response{StatusCode: 201, Body: io.NopCloser(strings.NewReader(`{"run":{"id":"owned-command-fixture"},"identity_token":"synthetic-bootstrap"}`))}, nil
				case "/api/v1/runs/owned-command-fixture/detach":
					detached++
					if r.Header.Get(HeaderAgentIdentityToken) != "synthetic-bootstrap" {
						t.Error("cleanup lost held attachment identity")
					}
					if len(recorder.stopped) != 0 {
						t.Error("lease ended before child cleanup")
					}
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("{}"))}, nil
				default:
					t.Fatalf("unexpected owner operation: %s", r.URL.Path)
					return nil, nil
				}
			})}
			var stdout bytes.Buffer
			exit := "0"
			if mode == "child exit" {
				exit = "23"
			}
			result, err := LaunchCommandSessionResult(context.Background(), AgentLaunchRequest{
				Args:    []string{"/bin/sh", "-c", "printf child-completed; exit " + exit},
				APIBase: "http://localhost", HTTPClient: client, Stdout: &stdout, Stderr: io.Discard,
			})
			if mode == "child exit" {
				if ChildExitCode(err) != 23 {
					t.Fatalf("child status lost: %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if stdout.String() != "child-completed" || attached != 1 || detached != 1 {
				t.Fatalf("command/owned cleanup: stdout=%q attach=%d detach=%d", stdout.String(), attached, detached)
			}
			if len(container.runs) != 0 {
				t.Error("child was placed into a second containment boundary")
			}
			if mode != "no container" {
				want, _ := testContainment()
				if !reflect.DeepEqual(container.selfLimits, []SessionContainment{want}) {
					t.Fatalf("wrapper ceilings=%+v", container.selfLimits)
				}
			}
			if !result.LeaseRecorded || len(recorder.created) != 1 || len(recorder.stopped) != 1 || recorder.stopped[0] != "owned-command-fixture:session ended" {
				t.Fatalf("lease lifecycle: %+v %+v", result, recorder)
			}
			lease := recorder.created[0]
			if lease.PID != os.Getpid() || lease.SessionID != "owned-command-fixture" || lease.Agent != "command-session" {
				t.Fatalf("lease changed caller: %+v", lease)
			}
			if mode == "contained" || mode == "child exit" {
				if lease.Scope != container.selfCalls[0] || result.ContainmentMethod != "transient-unit" || result.Scope != "cgroup:/x/"+lease.Scope+".scope" {
					t.Fatalf("boundary report: %+v %+v", lease, result)
				}
			} else if result.ContainmentMethod != ContainmentMethodNone || result.ContainmentFailure == "" {
				t.Fatalf("containment limitation hidden: %+v", result)
			}
		})
	}
}

// Real kernel parentage is checked across three test-process roles: wrapper,
// helper and requesting CLI. This does not invoke a provider CLI or a live grant;
// the test deliberately stays in its inherited cgroup.
func TestProcessBindingAllowsDescendantsButRejectsUnrelatedCaller(t *testing.T) {
	wrapper := exec.Command(os.Args[0], "-test.run=^TestProcessBindingChainHelper$")
	wrapper.Env = append(os.Environ(), "VROOLI_BINDING_CHAIN_LEVEL=wrapper")
	output, err := wrapper.CombinedOutput()
	if err != nil {
		t.Fatalf("kernel caller boundary: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "requesting-cli-verified") {
		t.Fatalf("requesting CLI assertion did not run: %s", output)
	}
}

func TestProcessBindingChainHelper(t *testing.T) {
	level := os.Getenv("VROOLI_BINDING_CHAIN_LEVEL")
	if level == "" {
		t.Skip("subprocess helper")
	}
	if level == "wrapper" {
		binding, err := ObserveProcessBinding(os.Getpid())
		if err != nil {
			t.Fatal(err)
		}
		raw, err := json.Marshal(binding)
		if err != nil {
			t.Fatal(err)
		}
		child := exec.Command(os.Args[0], "-test.run=^TestProcessBindingChainHelper$")
		child.Env = append(os.Environ(), "VROOLI_BINDING_CHAIN_LEVEL=helper", "VROOLI_BINDING_CHAIN_ROOT="+string(raw))
		output, err := child.CombinedOutput()
		if err != nil {
			t.Fatalf("helper: %v\n%s", err, output)
		}
		fmt.Print(string(output))
		return
	}
	var root ProcessBinding
	if err := json.Unmarshal([]byte(os.Getenv("VROOLI_BINDING_CHAIN_ROOT")), &root); err != nil {
		t.Fatal(err)
	}
	if !ProcessDescendsFrom(os.Getpid(), root) {
		t.Fatal("descendant in the caller boundary rejected")
	}
	// A parent of the wrapper is outside the approved tree despite matching UID,
	// boot and cgroup. Supplying/copying the root metadata does not authorize it.
	if ProcessDescendsFrom(ProcessParent(root.PID), root) {
		t.Fatal("unrelated caller accepted")
	}
	changed := root
	changed.Cgroup += "/different-session"
	if ProcessDescendsFrom(os.Getpid(), changed) {
		t.Fatal("changed cgroup binding accepted")
	}
	changed = root
	changed.Start += "-reused"
	if ProcessDescendsFrom(os.Getpid(), changed) {
		t.Fatal("reused PID accepted")
	}
	if level == "helper" {
		child := exec.Command(os.Args[0], "-test.run=^TestProcessBindingChainHelper$")
		child.Env = append(os.Environ(), "VROOLI_BINDING_CHAIN_LEVEL=cli")
		output, err := child.CombinedOutput()
		if err != nil {
			t.Fatalf("requesting CLI: %v\n%s", err, output)
		}
		fmt.Print(string(output))
		return
	}
	if level != "cli" {
		t.Fatalf("invalid helper level %q", level)
	}
	fmt.Println("requesting-cli-verified")
}

// Existing custody, with or without its compatibility session marker, must not
// move the caller or create a second child boundary to consume that custody.
func TestCommandSessionKeepsInheritedCallerBoundary(t *testing.T) {
	namespace, err := os.MkdirTemp("", "command-adopt-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(namespace) })
	t.Setenv("TMPDIR", namespace)
	for _, marker := range []string{"existing-session", ""} {
		t.Run("marker="+marker, func(t *testing.T) {
			t.Setenv(AgentSessionEnv, marker)
			t.Setenv(EnvIdentityToken, "synthetic-existing-custody")
			container := withFakeContainer(t, false)
			var stdout bytes.Buffer
			client := &http.Client{Transport: credentialRedirectTransport(func(r *http.Request) (*http.Response, error) {
				t.Errorf("adopted session attempted owner operation %s", r.URL.Path)
				return nil, errors.New("unexpected owner operation")
			})}
			result, err := LaunchCommandSessionResult(context.Background(), AgentLaunchRequest{
				Args: []string{"/bin/sh", "-c", "printf adopted-child"}, APIBase: "http://localhost", HTTPClient: client, Stdout: &stdout, Stderr: io.Discard,
			})
			if err != nil || stdout.String() != "adopted-child" || result.Tier != "tier-1" {
				t.Fatalf("adopted command: %v %+v %q", err, result, stdout.String())
			}
			if len(container.selfCalls) != 0 || len(container.runs) != 0 {
				t.Fatal("adopted custody crossed into new containment")
			}
		})
	}
}

func TestCommandSessionCleanupRefusesChangedCustody(t *testing.T) {
	namespace, err := os.MkdirTemp("", "command-cleanup-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(namespace) })
	t.Setenv("TMPDIR", namespace)
	t.Setenv(EnvIdentityToken, "")
	socket := AgentCredentialSocket()
	if err := os.MkdirAll(filepath.Dir(socket), 0o700); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	broker := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"token": "synthetic-new-consent"})
	})}
	go broker.Serve(listener)
	t.Cleanup(func() { broker.Close() })
	wireCalls := 0
	client := &http.Client{Transport: credentialRedirectTransport(func(r *http.Request) (*http.Response, error) {
		wireCalls++
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("{}"))}, nil
	})}
	err = detachCodingAgent(context.Background(), AgentLaunchRequest{RequireNativeIdentity: true, APIBase: "http://localhost", HTTPClient: client}, "owned-bootstrap-run", "synthetic-held-bootstrap")
	if err == nil || !strings.Contains(err.Error(), "native credential changed during exact request") || wireCalls != 0 {
		t.Fatalf("changed custody bypassed cleanup pin: err=%v wire=%d", err, wireCalls)
	}
}
