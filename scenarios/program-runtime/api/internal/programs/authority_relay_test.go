//go:build linux

package programs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
)

func TestSealedKernelNativeReadForwardsVerifiedIdentityWithoutDisclosingIt(t *testing.T) {
	verify := provenance.VerifierFunc(func(token string) (*cliutil.VerifyResult, error) {
		return &cliutil.VerifyResult{Valid: token == "synthetic-proof", Claims: &cliutil.VerifiedClaims{RunID: "run-1"}}, nil
	})
	var ownerCtx context.Context
	provenance.Middleware(verify)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { ownerCtx = r.Context() })).ServeHTTP(httptest.NewRecorder(), func() *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		r.Header.Set(cliutil.HeaderAgentIdentityToken, "synthetic-proof")
		return r
	}())
	ownerCtx = context.WithValue(ownerCtx, oneShotContextKey{}, true)
	owner := httptest.NewServer(provenance.Middleware(verify)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !provenance.FromContext(r.Context()).IsVerifiedAgent() {
			http.Error(w, "identity required", http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"thread":"active"}`))
	})))
	defer owner.Close()
	kernel, err := filepath.Abs("../../../kernel/host/engine.py")
	require.NoError(t, err)
	runner := NewSubprocessRunnerWithBindings(kernel, nil, owner.URL+"/internal/program-runtime/bindings/execute")
	defer runner.Close()
	source := fmt.Sprintf(`import json, urllib.request, bridge_transport
req = urllib.request.Request(%q, data=json.dumps({"session_id":"s","program_id":"p","binding_id":"web-console/session/native-thread-state"}).encode(), method="POST")
with bridge_transport.open_bridge_request(req, 5) as response:
    print(response.read().decode())`, owner.URL+"/internal/program-runtime/bindings/execute")
	result, err := runner.ExecuteWithMetadataAndLimitsAndCaller(ownerCtx, "s", "p", "PROVENANCE_AGENT", source, Caller{}, false, ExecutionLimits{})
	require.NoError(t, err)
	require.Contains(t, result.Stdout, `"active"`)
	require.NotContains(t, result.Stdout, "synthetic-proof")
}

func TestSealedDelegationRelaysVerifiedPairWithoutExposingProofs(t *testing.T) {
	var valid atomic.Bool
	valid.Store(true)
	verifier := bridgePairVerifier{allowed: &valid}
	var ownerCtx context.Context
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	request.Header.Set(cliutil.HeaderAgentIdentityToken, "baseline")
	request.Header.Set(cliutil.HeaderAgentOperationToken, "operation")
	provenance.Middleware(verifier)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { ownerCtx = r.Context() })).ServeHTTP(httptest.NewRecorder(), request)
	ownerCtx = context.WithValue(ownerCtx, oneShotContextKey{}, true)
	var calls atomic.Int32
	owner := httptest.NewServer(provenance.Middleware(verifier)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !provenance.FromContext(r.Context()).IsVerifiedAgent() || provenance.ForwardedOperationToken(r.Context()) != "operation" {
			http.Error(w, "verified pair required", http.StatusForbidden)
			return
		}
		calls.Add(1)
		if r.URL.Path == "/internal/program-runtime/agent/start" {
			_, _ = w.Write([]byte(`{"execution_id":"child-1"}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":"completed"}`))
	})))
	defer owner.Close()
	kernel, err := filepath.Abs("../../../kernel/host/engine.py")
	require.NoError(t, err)
	runner := NewSubprocessRunnerWithBindings(kernel, nil, owner.URL+"/internal/program-runtime/bindings/execute", owner.URL+"/internal/program-runtime/agent/execute")
	defer runner.Close()
	result, err := runner.ExecuteWithMetadataAndLimitsAndCaller(ownerCtx, "s", "p", "PROVENANCE_AGENT", `h = agent.start(owner="fixture", workflow_key="fixture/work")
print(agent.collect(h).head(1))`, Caller{}, false, ExecutionLimits{})
	require.NoError(t, err)
	require.Contains(t, result.Stdout, "completed")
	require.NotContains(t, result.Stdout, "baseline")
	require.NotContains(t, result.Stdout, "operation")
	require.EqualValues(t, 2, calls.Load())
}

func TestDelegationOwnerDenialRemainsUncertain(t *testing.T) {
	owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "operation grant denied", http.StatusForbidden)
	}))
	defer owner.Close()
	relay, err := newAuthorityRelay(context.Background(), owner.URL+"/internal/program-runtime/bindings/execute", "s", "p")
	require.NoError(t, err)
	require.NoError(t, relay.start(os.Getpid()))
	client := &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", relay.path)
	}}}
	response, err := client.Post("http://local/internal/program-runtime/agent/start", "application/json", strings.NewReader(`{"session_id":"s","program_id":"p"}`))
	require.NoError(t, err)
	_ = response.Body.Close()
	require.Equal(t, http.StatusForbidden, response.StatusCode)
	require.ErrorIs(t, relay.close(), ErrAuthorityEffectUncertain)
}

func TestNativeReadFailureDoesNotReportUncertainCheckpoint(t *testing.T) {
	owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "denied", http.StatusUnauthorized)
	}))
	defer owner.Close()
	relay, err := newAuthorityRelay(context.Background(), owner.URL+"/internal/program-runtime/bindings/execute", "s", "p")
	require.NoError(t, err)
	require.NoError(t, relay.start(os.Getpid()))
	client := &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", relay.path)
	}}}
	response, err := client.Post("http://local/internal/program-runtime/bindings/execute", "application/json", strings.NewReader(`{"session_id":"s","program_id":"p","binding_id":"web-console/session/native-thread-state"}`))
	require.NoError(t, err)
	_ = response.Body.Close()
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
	require.NoError(t, relay.close())
}

func TestSealedKernelPreservesNativeReadOwnerDenial(t *testing.T) {
	verify := provenance.VerifierFunc(func(token string) (*cliutil.VerifyResult, error) {
		return &cliutil.VerifyResult{Valid: token == "synthetic-proof", Claims: &cliutil.VerifiedClaims{RunID: "run-1"}}, nil
	})
	var ownerCtx context.Context
	provenance.Middleware(verify)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { ownerCtx = r.Context() })).ServeHTTP(httptest.NewRecorder(), func() *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		r.Header.Set(cliutil.HeaderAgentIdentityToken, "synthetic-proof")
		return r
	}())
	ownerCtx = context.WithValue(ownerCtx, oneShotContextKey{}, true)
	owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "access denied", http.StatusForbidden)
	}))
	defer owner.Close()
	kernel, err := filepath.Abs("../../../kernel/host/engine.py")
	require.NoError(t, err)
	runner := NewSubprocessRunnerWithBindings(kernel, nil, owner.URL+"/internal/program-runtime/bindings/execute")
	defer runner.Close()
	source := fmt.Sprintf(`import json, urllib.request, urllib.error, bridge_transport
req = urllib.request.Request(%q, data=json.dumps({"session_id":"s","program_id":"p","binding_id":"web-console/session/native-thread-state"}).encode(), method="POST")
try:
    bridge_transport.open_bridge_request(req, 5)
except urllib.error.HTTPError as exc:
    print(exc.code, exc.read().decode())`, owner.URL+"/internal/program-runtime/bindings/execute")
	result, err := runner.ExecuteWithMetadataAndLimitsAndCaller(ownerCtx, "s", "p", "PROVENANCE_AGENT", source, Caller{}, false, ExecutionLimits{})
	require.NoError(t, err)
	require.Contains(t, result.Stdout, "403 access denied")
}

func TestAuthorityRelayRequiresExactPeerRouteAndLiveProof(t *testing.T) {
	_, err := newAuthorityRelay(context.Background(), "http://localhost:1234/internal/program-runtime/bindings/execute", "session-1", "program-1")
	require.Error(t, err, "relay must not resolve hostname destinations")
	var active atomic.Bool
	active.Store(true)
	verify := provenance.VerifierFunc(func(token string) (*cliutil.VerifyResult, error) {
		return &cliutil.VerifyResult{Valid: token == "synthetic-proof" && active.Load(), Claims: &cliutil.VerifiedClaims{RunID: "run-1"}}, nil
	})
	var ownerCtx context.Context
	provenance.Middleware(verify)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { ownerCtx = r.Context() })).ServeHTTP(httptest.NewRecorder(), func() *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		r.Header.Set(cliutil.HeaderAgentIdentityToken, "synthetic-proof")
		return r
	}())
	var calls atomic.Int32
	owner := httptest.NewServer(provenance.Middleware(verify)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if !provenance.FromContext(r.Context()).IsVerifiedAgent() {
			http.Error(w, "revoked", http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	})))
	defer owner.Close()
	relay, err := newAuthorityRelay(ownerCtx, owner.URL+"/internal/program-runtime/bindings/execute", "session-1", "program-1")
	require.NoError(t, err)
	require.NoError(t, relay.start(os.Getpid()))
	defer relay.close()
	client := &http.Client{Transport: &http.Transport{DisableKeepAlives: true, DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", relay.path)
	}}, Timeout: 3 * time.Second}
	do := func(method, path, body string) (int, string) {
		r, err := http.NewRequest(method, "http://local"+path, strings.NewReader(body))
		require.NoError(t, err)
		r.Header.Set(cliutil.HeaderAgentIdentityToken, "forged-proof")
		resp, err := client.Do(r)
		require.NoError(t, err)
		defer resp.Body.Close()
		payload, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		return resp.StatusCode, string(payload)
	}
	good := `{"session_id":"session-1","program_id":"program-1"}`
	status, body := do(http.MethodPost, "/internal/program-runtime/tasks/resolve", good)
	require.Equal(t, http.StatusOK, status)
	require.NotContains(t, body, "synthetic-proof")
	require.EqualValues(t, 1, calls.Load())
	status, body = do(http.MethodPost, "/internal/program-runtime/bindings/execute", `{"session_id":"session-1","program_id":"program-1","binding_id":"web-console/session/native-thread-state"}`)
	require.Equal(t, http.StatusOK, status)
	require.NotContains(t, body, "synthetic-proof")
	status, _ = do(http.MethodPost, "/internal/program-runtime/agent/start", good)
	require.Equal(t, http.StatusOK, status)
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodGet, "/internal/program-runtime/tasks/resolve", good},
		{http.MethodPost, "/internal/program-runtime/tasks/fragment_publish", good},
		{http.MethodPost, "/internal/program-runtime/agent/execute", good},
		{http.MethodPost, "/internal/program-runtime/agent/start", `{"session_id":"other","program_id":"program-1"}`},
		{http.MethodPost, "/internal/program-runtime/agent/collect", `{"session_id":"session-1","program_id":"other"}`},
		{http.MethodPost, "/internal/program-runtime/tasks/../tasks/resolve", good},
		{http.MethodPost, "/internal/program-runtime/tasks/resolve?x=1", good},
		{http.MethodPost, "/internal/program-runtime/tasks/resolve", `{"session_id":"other","program_id":"program-1"}`},
		{http.MethodPost, "/internal/program-runtime/tasks/resolve", `{"session_id":"session-1","SESSION_ID":"other","program_id":"program-1"}`},
		{http.MethodPost, "/internal/program-runtime/tasks/resolve", `{"session_id":"session-1","session_id":"other","program_id":"program-1"}`},
		{http.MethodPost, "/internal/program-runtime/bindings/execute", `{"session_id":"session-1","program_id":"program-1","binding_id":"web-console/session/native-thread-state","BINDING_ID":"other/write"}`},
		{http.MethodPost, "/internal/program-runtime/bindings/execute", `{"session_id":"session-1","program_id":"program-1","binding_id":"other/write"}`},
	} {
		status, _ := do(tc.method, tc.path, tc.body)
		require.Truef(t, status >= 400, "%s %s unexpectedly admitted", tc.method, tc.path)
	}
	require.EqualValues(t, 3, calls.Load(), "refused requests must not reach owner")
	active.Store(false)
	status, _ = do(http.MethodPost, "/internal/program-runtime/tasks/resolve", good)
	require.Equal(t, http.StatusUnauthorized, status, "receiving owner must reverify the carried proof")
	// A separate process cannot impersonate the exact kernel peer PID even if
	// it knows the socket path and shares the OS user.
	command := exec.Command("python3", "-c", `import socket,sys
s=socket.socket(socket.AF_UNIX); s.connect(sys.argv[1]); s.sendall(b'POST /internal/program-runtime/tasks/resolve HTTP/1.1\r\nHost: local\r\nContent-Length: 2\r\n\r\n{}'); print(s.recv(256).decode().splitlines()[0])`, relay.path)
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
	require.Contains(t, string(output), "403")
	path := relay.path
	relay.close()
	_, err = os.Stat(path)
	require.True(t, os.IsNotExist(err), "relay socket survives close")
	_, err = net.DialTimeout("unix", path, time.Second)
	require.Error(t, err)
}

func TestAuthorityRelayClosesSaturatedConnections(t *testing.T) {
	relay, err := newAuthorityRelay(context.Background(), "http://127.0.0.1:1/internal/program-runtime/bindings/execute", "s", "p")
	require.NoError(t, err)
	require.NoError(t, relay.start(os.Getpid()))
	connections := make([]net.Conn, 0, 16)
	for i := 0; i < 16; i++ {
		connection, err := net.DialTimeout("unix", relay.path, time.Second)
		require.NoError(t, err)
		connections = append(connections, connection)
	}
	done := make(chan struct{})
	go func() { relay.close(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("closing saturated relay blocked")
	}
	for _, connection := range connections {
		_ = connection.Close()
	}
	_, err = os.Stat(relay.path)
	require.True(t, os.IsNotExist(err))
}

func TestUnavailableOneShotRelayPreservesReadsAndRefusesCheckpoint(t *testing.T) {
	verify := provenance.VerifierFunc(func(token string) (*cliutil.VerifyResult, error) {
		return &cliutil.VerifyResult{Valid: token == "synthetic-proof", Claims: &cliutil.VerifiedClaims{RunID: "run-1"}}, nil
	})
	var ownerCtx context.Context
	provenance.Middleware(verify)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { ownerCtx = r.Context() })).ServeHTTP(httptest.NewRecorder(), func() *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		r.Header.Set(cliutil.HeaderAgentIdentityToken, "synthetic-proof")
		return r
	}())
	ownerCtx = context.WithValue(ownerCtx, oneShotContextKey{}, true)
	var ownerCalls atomic.Int32
	owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { ownerCalls.Add(1); w.WriteHeader(http.StatusOK) }))
	defer owner.Close()
	kernel, err := filepath.Abs("../../../kernel/host/engine.py")
	require.NoError(t, err)
	runner := NewSubprocessRunnerWithBindings(kernel, nil, owner.URL+"/internal/program-runtime/bindings/execute")
	defer runner.Close()
	runner.relayFactory = func(context.Context, string, string, string) (*authorityRelay, error) {
		return nil, errors.New("unsupported platform")
	}
	read, err := runner.ExecuteWithMetadataAndLimitsAndCaller(ownerCtx, "read-session", "read-program", "PROVENANCE_AGENT", `print("read succeeded")`, Caller{}, false, ExecutionLimits{})
	require.NoError(t, err)
	require.Contains(t, read.Stdout, "read succeeded")
	runner.KillSession("read-session")
	_, err = runner.ExecuteWithMetadataAndLimitsAndCaller(ownerCtx, "task-session", "task-program", "PROVENANCE_AGENT", `learn.task(operation="fixture.checkpoint")`, Caller{}, false, ExecutionLimits{})
	require.ErrorContains(t, err, "verified task checkpoint relay unavailable")
	require.Zero(t, ownerCalls.Load(), "a missing relay must not fall back to an anonymous owner request")
}

func TestAuthorityRelayDrainsOwnerEffectOrReportsUncertain(t *testing.T) {
	for _, timedOut := range []bool{false, true} {
		t.Run(map[bool]string{false: "drain", true: "uncertain"}[timedOut], func(t *testing.T) {
			started, release, committed := make(chan struct{}), make(chan struct{}), make(chan struct{})
			owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				close(started)
				<-release
				close(committed) // The owner may commit even if the client has cancelled.
				_, _ = w.Write([]byte(`{"ok":true}`))
			}))
			defer owner.Close()
			relay, err := newAuthorityRelay(context.Background(), owner.URL+"/internal/program-runtime/bindings/execute", "s", "p")
			require.NoError(t, err)
			if timedOut {
				relay.drainTimeout = 100 * time.Millisecond
			}
			require.NoError(t, relay.start(os.Getpid()))
			client := &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", relay.path)
			}}}
			requestDone := make(chan struct{})
			go func() {
				defer close(requestDone)
				response, err := client.Post("http://local/internal/program-runtime/tasks/begin", "application/json", strings.NewReader(`{"session_id":"s","program_id":"p"}`))
				if err == nil {
					_ = response.Body.Close()
				}
			}()
			select {
			case <-started:
			case <-time.After(2 * time.Second):
				t.Fatal("owner was not reached")
			}
			closed := make(chan error, 1)
			go func() { closed <- relay.close() }()
			if timedOut {
				require.ErrorIs(t, <-closed, ErrAuthorityEffectUncertain)
				close(release)
				<-committed
			} else {
				select {
				case err := <-closed:
					t.Fatalf("relay completed before owner: %v", err)
				case <-time.After(50 * time.Millisecond):
				}
				close(release)
				<-committed
				require.NoError(t, <-closed)
			}
			<-requestDone
		})
	}
}

func TestAuthorityRelayTreatsConflictAfterOwnerEffectAsUncertain(t *testing.T) {
	var committed atomic.Bool
	owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		committed.Store(true)
		http.Error(w, "conflict after write", http.StatusConflict)
	}))
	defer owner.Close()
	relay, err := newAuthorityRelay(context.Background(), owner.URL+"/internal/program-runtime/bindings/execute", "s", "p")
	require.NoError(t, err)
	require.NoError(t, relay.start(os.Getpid()))
	client := &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", relay.path)
	}}}
	response, err := client.Post("http://local/internal/program-runtime/tasks/complete", "application/json", strings.NewReader(`{"session_id":"s","program_id":"p"}`))
	require.NoError(t, err)
	_ = response.Body.Close()
	require.Equal(t, http.StatusConflict, response.StatusCode)
	require.True(t, committed.Load())
	require.ErrorIs(t, relay.close(), ErrAuthorityEffectUncertain)
}

func TestAuthorityRelayAcceptsOnlyTypedPreEffectResolveAbsence(t *testing.T) {
	for _, tc := range []struct {
		name, path, body string
		wantUncertain    bool
	}{
		{"typed-missing", "/internal/program-runtime/tasks/resolve", `{"error":"operation is not registered","reason":"operation_not_registered","effect":"none"}`, false},
		{"untyped", "/internal/program-runtime/tasks/resolve", `{"error":"operation is not registered"}`, true},
		{"wrong-reason", "/internal/program-runtime/tasks/resolve", `{"reason":"other","effect":"none"}`, true},
		{"wrong-action", "/internal/program-runtime/tasks/complete", `{"reason":"operation_not_registered","effect":"none"}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer owner.Close()
			relay, err := newAuthorityRelay(context.Background(), owner.URL+"/internal/program-runtime/bindings/execute", "s", "p")
			require.NoError(t, err)
			require.NoError(t, relay.start(os.Getpid()))
			client := &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", relay.path)
			}}}
			response, err := client.Post("http://local"+tc.path, "application/json", strings.NewReader(`{"session_id":"s","program_id":"p"}`))
			require.NoError(t, err)
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
			if tc.wantUncertain {
				require.ErrorIs(t, relay.close(), ErrAuthorityEffectUncertain)
			} else {
				require.NoError(t, relay.close())
			}
		})
	}
}

func TestSealedRunnerDoesNotReportSuccessForUncertainBackgroundCheckpoint(t *testing.T) {
	started, release, committed := make(chan struct{}), make(chan struct{}), make(chan struct{})
	owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		close(committed)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer owner.Close()
	verify := provenance.VerifierFunc(func(token string) (*cliutil.VerifyResult, error) {
		return &cliutil.VerifyResult{Valid: token == "synthetic-proof", Claims: &cliutil.VerifiedClaims{RunID: "run-1"}}, nil
	})
	var ownerCtx context.Context
	provenance.Middleware(verify)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { ownerCtx = r.Context() })).ServeHTTP(httptest.NewRecorder(), func() *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		r.Header.Set(cliutil.HeaderAgentIdentityToken, "synthetic-proof")
		return r
	}())
	ownerCtx = context.WithValue(ownerCtx, oneShotContextKey{}, true)
	kernel, err := filepath.Abs("../../../kernel/host/engine.py")
	require.NoError(t, err)
	runner := NewSubprocessRunnerWithBindings(kernel, nil, owner.URL+"/internal/program-runtime/bindings/execute")
	defer runner.Close()
	runner.relayFactory = func(ctx context.Context, target, session, program string) (*authorityRelay, error) {
		relay, err := newAuthorityRelay(ctx, target, session, program)
		if relay != nil {
			relay.drainTimeout = 100 * time.Millisecond
		}
		return relay, err
	}
	source := `import threading, time, urllib.request
from bridge_transport import open_bridge_request
def issue():
    try:
        req=urllib.request.Request(` + strconv.Quote(owner.URL+"/internal/program-runtime/tasks/begin") + `, data=b'{"session_id":"s","program_id":"p"}', method="POST")
        with open_bridge_request(req, timeout=2): pass
    except Exception: pass
threading.Thread(target=issue, daemon=True).start()
time.sleep(0.2)
print("reported-ok")`
	type execution struct {
		result Result
		err    error
	}
	finished := make(chan execution, 1)
	go func() {
		result, err := runner.ExecuteWithMetadataAndLimitsAndCaller(ownerCtx, "s", "p", "PROVENANCE_AGENT", source, Caller{}, false, ExecutionLimits{})
		finished <- execution{result, err}
	}()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		close(release)
		t.Fatal("owner was not reached")
	}
	got := <-finished
	require.ErrorIs(t, got.err, ErrAuthorityEffectUncertain)
	require.Contains(t, got.result.Stdout, "reported-ok")
	close(release)
	<-committed
}
