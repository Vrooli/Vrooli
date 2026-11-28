//go:build linux

package orchestration

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	runidentity "agent-manager/internal/identity"
	"github.com/google/uuid"
	"github.com/vrooli/cli-core/cliutil"
)

type treasurySessionBrokerFixture struct {
	current, rotated, rotationRequest string
}

type blockedCredentialResponseWriter struct {
	entered chan struct{}
	resume  chan struct{}
	header  http.Header
}

func (w *blockedCredentialResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}
func (w *blockedCredentialResponseWriter) WriteHeader(int) {}
func (w *blockedCredentialResponseWriter) Write(p []byte) (int, error) {
	close(w.entered)
	<-w.resume
	return len(p), nil
}

func TestTreasurySessionBrokerLeaseCoversResponseDelivery(t *testing.T) {
	var gate sync.RWMutex
	gate.RLock()
	w := &blockedCredentialResponseWriter{entered: make(chan struct{}), resume: make(chan struct{})}
	responseDone := make(chan struct{})
	go func() { writeTreasurySessionKeyResponse(w, strings.Repeat("a", 64), gate.RUnlock); close(responseDone) }()
	<-w.entered
	terminalDone := make(chan struct{})
	go func() { gate.Lock(); gate.Unlock(); close(terminalDone) }()
	select {
	case <-terminalDone:
		t.Fatal("terminal cleanup passed the broker lease before response delivery")
	case <-time.After(30 * time.Millisecond):
	}
	close(w.resume)
	<-responseDone
	select {
	case <-terminalDone:
	case <-time.After(time.Second):
		t.Fatal("terminal cleanup did not proceed after broker delivery")
	}
}

func (f *treasurySessionBrokerFixture) ResolveOrMint(context.Context, uuid.UUID) (string, error) {
	return f.current, nil
}

func (f *treasurySessionBrokerFixture) Acquire(ctx context.Context, id uuid.UUID) (string, func(), error) {
	key, err := f.ResolveOrMint(ctx, id)
	return key, func() {}, err
}

func (f *treasurySessionBrokerFixture) Rotate(_ context.Context, _ uuid.UUID, request string) (string, error) {
	if f.rotationRequest == request {
		return f.rotated, nil
	}
	f.rotationRequest = request
	return f.rotated, nil
}

func (f *treasurySessionBrokerFixture) AcquireRotation(ctx context.Context, id uuid.UUID, request string) (string, func(), error) {
	key, err := f.Rotate(ctx, id, request)
	return key, func() {}, err
}

func (f *treasurySessionBrokerFixture) RevokeRunCredentialUse(context.Context, uuid.UUID, string) error {
	return nil
}

func TestTreasurySessionKeyBrokerRequiresOrdinaryRunAndReplaysRotation(t *testing.T) {
	o, ctx, _, run, _ := newCredentialPairFixture(t)
	claims := runidentity.Claims{
		RunID: run.ID, TaskID: run.TaskID, Subject: run.OwnerSubject,
		Scopes: []string{"treasury:write"}, IssuedAt: time.Now().Unix(), ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}
	baselineToken, err := runidentity.GenerateToken(&claims, []byte(pairTestSecret))
	if err != nil {
		t.Fatal(err)
	}
	run.IdentityTokenHash = runidentity.HashToken(baselineToken)
	if err := o.runs.Update(ctx, run); err != nil {
		t.Fatal(err)
	}
	store := &treasurySessionBrokerFixture{current: "a1" + strings.Repeat("1", 62), rotated: "b2" + strings.Repeat("2", 62)}
	o.treasurySessionKeys = store

	dir, err := os.MkdirTemp("/tmp", "amb-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(dir, "identity.sock")
	stop, err := o.startAuthorizationBrokerAt(socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = stop() })
	if _, err := o.startAuthorizationBrokerAt(socket); err == nil || !strings.Contains(err.Error(), "already running") {
		t.Fatalf("a second Agent Manager owner was allowed to take the shared credential socket: %v", err)
	}
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	t.Cleanup(transport.CloseIdleConnections)
	request := func(query string) (int, string) {
		t.Helper()
		req, err := http.NewRequest(http.MethodGet, "http://local-agent-manager/credential?purpose=treasury_session_key"+query, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set(cliutil.HeaderAgentIdentityToken, baselineToken)
		response, err := (&http.Client{Transport: transport}).Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var body struct {
			Key  string `json:"treasury_session_private_key"`
			Code string `json:"code"`
		}
		_ = json.NewDecoder(response.Body).Decode(&body)
		return response.StatusCode, body.Key + body.Code
	}

	status, initial := request("")
	if status != http.StatusOK || initial != store.current {
		t.Fatalf("initial run key resolve status=%d value=%q", status, initial)
	}
	status, rotated := request("&rotation_request=rotate-1")
	if status != http.StatusOK || rotated != store.rotated {
		t.Fatalf("run key rotation status=%d value=%q", status, rotated)
	}
	status, replayed := request("&rotation_request=rotate-1")
	if status != http.StatusOK || replayed != store.rotated {
		t.Fatalf("idempotent run key rotation replay status=%d value=%q", status, replayed)
	}
	status, duplicate := request("&rotation_request=rotate-1&rotation_request=rotate-1")
	if status != http.StatusBadRequest || duplicate != "" {
		t.Fatalf("duplicate rotation parameter status=%d value=%q", status, duplicate)
	}
}
