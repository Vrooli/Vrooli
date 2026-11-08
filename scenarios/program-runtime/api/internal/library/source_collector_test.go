package library

import (
	"bytes"
	"context"
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/authn"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/identity"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	"io"
	"net/http"
	"program-runtime/internal/contracts"
	"program-runtime/internal/localworkbridge"
	"program-runtime/internal/programs"
	"program-runtime/internal/sessions"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type ownerJWKSFixture struct{ key *rsa.PrivateKey }

func (d ownerJWKSFixture) Do(r *http.Request) (*http.Response, error) {
	if r.URL.String() != "https://fixture.invalid/jwks" {
		return nil, localworkbridge.ErrSourceAuthority
	}
	body := fmt.Sprintf(`{"keys":[{"kid":"fixture","kty":"RSA","alg":"RS256","n":%q,"e":"AQAB"}]}`, base64.RawURLEncoding.EncodeToString(d.key.N.Bytes()))
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
}

type ownerBindingFixture struct{ b localworkbridge.SourceBinding }

func (b *ownerBindingFixture) ReadSourceBinding(context.Context, string) (localworkbridge.SourceBinding, error) {
	return b.b, nil
}

type ownerBoundaryFunc func(context.Context, func(context.Context) error) error

func (f ownerBoundaryFunc) Within(ctx context.Context, fn func(context.Context) error) error {
	return f(ctx, fn)
}

type collectorRunner struct {
	calls  atomic.Int32
	source string
	mu     sync.Mutex
}

func (r *collectorRunner) Execute(_ context.Context, _, source string, _ bool) (programs.Result, error) {
	r.calls.Add(1)
	r.mu.Lock()
	r.source = source
	r.mu.Unlock()
	return programs.Result{Stdout: `{"ok":true}`}, nil
}
func TestSourceCollectorCanonicalAdmissionReplayObserveWaitAndReconcile(t *testing.T) {
	contract := contracts.Contract{ID: localworkbridge.SourceCollectorName, Source: "print(inputs)", WallMS: 1000, OutputBytes: 4096, Declaration: []byte(`{"helpers":{"program":"2"},"outputs":{}}`), Inputs: map[string]contracts.InputSpec{"row_limit": {Type: "integer"}, "team_ids": {Type: "array"}}}
	contract.Digest = contracts.ContentDigest(contract.Declaration, []byte(contract.Source), nil)
	now := time.Now()
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	require.NoError(t, err)
	policy := localworkbridge.SourcePolicy{Purpose: localworkbridge.SourceCollector, BindingID: "binding-1", BindingVersion: "v1", Subject: "bridge-service", Owner: "owner-1", SourceInstance: "swarminator", SourceEpoch: "epoch-1", CollectorDigest: contract.Digest, CollectorInputs: json.RawMessage(`{"row_limit":4,"team_ids":[]}`)}
	authority := &ownerBindingFixture{}
	verifier := authn.NewScenarioAuthenticatorProvider(authn.JWTConfig{Audience: "vrooli-local-work:collector", ExclusiveAudience: true, JWKSURL: "https://fixture.invalid/jwks", Doer: ownerJWKSFixture{key: key}})
	gate, err := localworkbridge.NewSourceGate(localworkbridge.SourceGateConfig{Enabled: true, Policy: policy, Verifier: verifier, Bindings: authority})
	require.NoError(t, err)
	authority.b = localworkbridge.SourceBinding{ID: policy.BindingID, Version: policy.BindingVersion, Subject: policy.Subject, Owner: policy.Owner, Audience: "vrooli-local-work:collector", ScopeSHA256: gate.ScopeSHA256(), NotBefore: now.Add(-time.Minute), ExpiresAt: now.Add(2 * time.Hour), Grant: &identity.OperationGrant{ID: "fixture-grant", Approver: policy.Owner, ExpiresAt: now.Add(2 * time.Hour), Operations: []string{"local-work.collector.admit", "local-work.collector.observe", "local-work.collector.wait"}, Targets: []string{"program-runtime:local-work/" + gate.ScopeSHA256() + "/*"}}}
	payload, err := json.Marshal(map[string]any{"sub": policy.Subject, "iss": "scenario-authenticator", "actor_kind": "service", "scope": []string{"local-work:collector"}, "aud": "vrooli-local-work:collector", "exp": now.Add(time.Hour).Unix()})
	require.NoError(t, err)
	input := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","kid":"fixture"}`)) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(input))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	require.NoError(t, err)
	token := input + "." + base64.RawURLEncoding.EncodeToString(signature)
	admitPermit := func(path, body string) *localworkbridge.SourcePermit {
		r, err := http.NewRequest("POST", path, strings.NewReader(body))
		require.NoError(t, err)
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Connect-Protocol-Version", "1")
		p, err := gate.Admit(context.Background(), r)
		require.NoError(t, err)
		return p
	}
	nativeDB := dbtest.NewSQLite(t)
	nativeStore := localworkbridge.NewNativeReceiptStore(nativeDB)
	nativeConfig := localworkbridge.NativeReceiptConfig{Enabled: true, Scope: localworkbridge.NativeReceiptScope{Owner: policy.Owner, Subject: policy.Subject, SourceInstance: policy.SourceInstance, SourceEpoch: policy.SourceEpoch, AllowedTypes: []string{"agent_manager.run.notification.v1"}}, Credentials: collectorNativeKey{}, Guard: &collectorNativeGuard{}, Store: nativeStore}
	native, err := localworkbridge.NewNativeReceiptOwner(nativeConfig)
	require.NoError(t, err)
	accepted, err := native.Accept(context.Background(), collectorNativeRequest("event-1"))
	require.NoError(t, err)
	require.True(t, accepted)
	selection, err := native.Select(context.Background())
	require.NoError(t, err)
	eventPolicy := policy
	eventPolicy.Purpose = localworkbridge.SourceEvents
	eventPolicy.BindingID = "events-binding"
	eventPolicy.CollectorDigest = ""
	eventPolicy.CollectorInputs = nil
	eventPolicy.EventIDs = selection.EventIDs()
	eventVerifier := authn.NewScenarioAuthenticatorProvider(authn.JWTConfig{Audience: "vrooli-local-work:events", ExclusiveAudience: true, JWKSURL: "https://fixture.invalid/jwks", Doer: ownerJWKSFixture{key: key}})
	eventsGate, err := localworkbridge.NewSourceGate(localworkbridge.SourceGateConfig{Enabled: true, Policy: eventPolicy, Verifier: eventVerifier, Bindings: authority})
	require.NoError(t, err)
	selected, err := native.BindSelection(context.Background(), selection, eventsGate, gate)
	require.NoError(t, err)
	requestKey := selected.RequestKey
	deadline := selected.AdmissionDeadline
	body := fmt.Sprintf(`{"name":%q,"inputs":{"row_limit":4,"team_ids":[]},"expectedDigest":%q,"provenance":"PROVENANCE_AGENT","async":true,"idempotencyKey":%q,"admissionDeadline":%q}`, localworkbridge.SourceCollectorName, policy.CollectorDigest, requestKey, deadline)
	permit := admitPermit("/vrooli.program_runtime.v1.library.LibraryService/RunDeclaredProgram", body)

	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(Schema), apidb.SchemaProviderFunc(programs.Schema), apidb.SchemaProviderFunc(sessions.Schema)))
	repo := NewRepository(db)
	require.NoError(t, repo.RetainDeclared(context.Background(), contract))
	manager := sessions.NewManager(sessions.Options{Store: db})
	runner := &collectorRunner{}
	var boundaryMu sync.Mutex
	held := false
	boundary := ownerBoundaryFunc(func(ctx context.Context, fn func(context.Context) error) error {
		boundaryMu.Lock()
		defer boundaryMu.Unlock()
		held = true
		defer func() { held = false }()
		return fn(ctx)
	})
	terminalCleaned := make(chan struct{})
	denySession := true
	denyAuthority := false
	c, err := NewSourceCollector(repo, manager, programs.Options{Store: db, Runner: runner, OnTerminal: func(context.Context, string) error { close(terminalCleaned); return nil }, PreflightSession: func(ctx context.Context, id, source string) []*programsv1.Diagnostic {
		if denySession {
			_, err := manager.Delete(ctx, id, "fixture invalidation before final insert")
			require.NoError(t, err)
		}
		if denyAuthority {
			authority.b.Revoked = true
		}
		return nil
	}}, boundary)
	require.NoError(t, err)
	nativeConfig.CollectorReconciler = c
	native, err = localworkbridge.NewNativeReceiptOwner(nativeConfig)
	require.NoError(t, err)
	observe := admitPermit("/vrooli.program_runtime.v1.library.LibraryService/GetDeclaredExecution", fmt.Sprintf(`{"name":%q,"idempotencyKey":%q}`, localworkbridge.SourceCollectorName, requestKey))
	absent, err := c.ObserveCollector(context.Background(), observe)
	require.NoError(t, err)
	require.False(t, absent.Found)
	_, err = c.AdmitCollector(context.Background(), permit)
	require.Error(t, err)
	var count int
	require.NoError(t, db.QueryRow("SELECT count(*) FROM programs").Scan(&count))
	require.Zero(t, count)
	require.Zero(t, runner.calls.Load())
	denySession = false
	denyAuthority = true
	_, err = c.AdmitCollector(context.Background(), permit)
	require.Error(t, err)
	require.NoError(t, db.QueryRow("SELECT count(*) FROM programs").Scan(&count))
	require.Zero(t, count)
	require.Zero(t, runner.calls.Load())
	denyAuthority = false
	authority.b.Revoked = false
	first, err := c.AdmitCollector(context.Background(), permit)
	require.NoError(t, err)
	require.True(t, first.Found)
	wait := admitPermit("/vrooli.program_runtime.v1.programs.ProgramService/WaitForProgram", fmt.Sprintf(`{"id":%q,"timeoutMillis":1000}`, first.Identity.ID))
	terminal, err := c.WaitCollector(context.Background(), wait)
	require.NoError(t, err)
	require.Equal(t, "PROGRAM_STATUS_SUCCEEDED", terminal.Status)
	select {
	case <-terminalCleaned:
	case <-time.After(time.Second):
		t.Fatal("one-shot cleanup did not finish")
	}
	replay, err := c.AdmitCollector(context.Background(), permit)
	require.NoError(t, err)
	require.Equal(t, first.Identity, replay.Identity)
	require.EqualValues(t, 1, runner.calls.Load())
	runner.mu.Lock()
	require.Contains(t, runner.source, `inputs = json.loads("{\"row_limit\":4,\"team_ids\":[]}")`)
	runner.mu.Unlock()
	var verified bool
	require.NoError(t, db.QueryRow("SELECT caller_identity_verified FROM programs WHERE id=?", first.Identity.ID).Scan(&verified))
	require.False(t, verified)
	visited := false
	require.NoError(t, c.WithTerminal(context.Background(), observe, func(_ context.Context, p localworkbridge.SourceBrokerProgram) error {
		require.True(t, held)
		require.Equal(t, `{"ok":true}`, p.Stdout)
		require.Equal(t, first.Identity, p.Identity)
		visited = true
		return nil
	}))
	require.True(t, visited)
	require.NoError(t, native.ReconcileCollector(context.Background(), selection, observe))
	none, err := native.Select(context.Background())
	require.NoError(t, err)
	require.Nil(t, none)
	accepted, err = native.Accept(context.Background(), collectorNativeRequest("event-1"))
	require.NoError(t, err)
	require.True(t, accepted)
	accepted, err = native.Accept(context.Background(), collectorNativeRequest("event-2"))
	require.NoError(t, err)
	require.True(t, accepted)
	later, err := native.Select(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"event-2"}, later.EventIDs())
	require.NoError(t, native.ReconcileCollector(context.Background(), selection, observe))
	retained, err := native.Select(context.Background())
	require.NoError(t, err)
	require.Equal(t, later, retained)
	// Fixture mutation models an abruptly stopped prior runtime, not a running
	// execution. Restart recovery must touch only the exact service-owned row.
	for _, status := range []string{"accepted", "running"} {
		_, err = db.Exec("UPDATE programs SET status=? WHERE id=?", status, first.Identity.ID)
		require.NoError(t, err)
		_, err = db.Exec("INSERT OR REPLACE INTO programs (id,session_id,source,provenance,status,created_at) VALUES ('unrelated','session','print(1)','1','running','fixture')")
		require.NoError(t, err)
		restarted, err := NewSourceCollector(repo, manager, programs.Options{Store: db, Runner: runner}, boundary)
		require.NoError(t, err)
		_, err = restarted.RecoverCollectorAfterStop(context.Background(), observe, nil)
		require.Error(t, err)
		blockedCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		_, err = restarted.RecoverCollectorAfterStop(blockedCtx, observe, collectorStoppedFixture{joined: make(chan struct{})})
		cancel()
		require.Error(t, err)
		var unchanged string
		require.NoError(t, db.QueryRow("SELECT status FROM programs WHERE id=?", first.Identity.ID).Scan(&unchanged))
		require.Equal(t, status, unchanged)
		recovered, err := restarted.RecoverCollectorAfterStop(context.Background(), observe, collectorStoppedFixture{joined: terminalCleaned})
		require.NoError(t, err)
		require.Equal(t, "PROGRAM_STATUS_FAILED", recovered.Status)
		require.Empty(t, recovered.Stdout)
		var unrelated, cause string
		require.NoError(t, db.QueryRow("SELECT status FROM programs WHERE id='unrelated'").Scan(&unrelated))
		require.Equal(t, "running", unrelated)
		require.NoError(t, db.QueryRow("SELECT failure_shape FROM programs WHERE id=?", first.Identity.ID).Scan(&cause))
		require.Equal(t, "runtime_interrupted", cause)
		require.EqualValues(t, 1, runner.calls.Load())
	}
	authority.b.Revoked = true
	_, err = c.ObserveCollector(context.Background(), observe)
	require.Error(t, err)
	_, err = c.AdmitCollector(context.Background(), permit)
	require.Error(t, err)
}

func TestSourceCollectorRejectsUnverifiedAndUnsupportedConstruction(t *testing.T) {
	_, err := NewSourceCollector(nil, nil, programs.Options{}, nil)
	require.Error(t, err)
	c := &SourceCollector{}
	_, err = c.AdmitCollector(context.Background(), &localworkbridge.SourcePermit{})
	require.Error(t, err)
	_, err = c.ObserveCollector(context.Background(), nil)
	require.Error(t, err)
	_, err = c.WaitCollector(context.Background(), nil)
	require.Error(t, err)
}

func TestSourceCollectorPreparationRejectsUnsafeContracts(t *testing.T) {
	base := contracts.Contract{ID: localworkbridge.SourceCollectorName, Source: "print(1)", WallMS: 120000, OutputBytes: 65536, Declaration: []byte(`{"helpers":{"program":"2"},"outputs":{}}`), Inputs: map[string]contracts.InputSpec{"row_limit": {Type: "integer", Default: json.RawMessage(`4`)}}}
	_, source, _, err := prepareSourceCollector(base, json.RawMessage(`{}`), localworkbridge.SourceCall{})
	require.NoError(t, err)
	require.Contains(t, source, "row_limit")
	for name, mutate := range map[string]func(*contracts.Contract){
		"invalid":   func(c *contracts.Contract) { c.ValidationError = "invalid" },
		"write":     func(c *contracts.Contract) { c.Bindings = []contracts.BindingRef{{ID: "write", Effect: "write"}} },
		"unbounded": func(c *contracts.Contract) { c.WallMS = 0 },
		"oversize":  func(c *contracts.Contract) { c.OutputBytes = 1000000 },
		"learning":  func(c *contracts.Contract) { c.Learning.Verbs = []string{"task"} },
		"missing":   func(c *contracts.Contract) { c.SourceMissing = true },
	} {
		t.Run(name, func(t *testing.T) {
			c := base
			mutate(&c)
			_, _, _, err := prepareSourceCollector(c, json.RawMessage(`{}`), localworkbridge.SourceCall{})
			require.Error(t, err)
		})
	}
	_, _, _, err = prepareSourceCollector(base, json.RawMessage(`{"unknown":1}`), localworkbridge.SourceCall{})
	require.Error(t, err)
}

type collectorNativeKey struct{}

func (collectorNativeKey) VerificationKey(context.Context, localworkbridge.NativeReceiptScope) ([]byte, error) {
	return bytes.Repeat([]byte{7}, 32), nil
}

type collectorNativeGuard struct{ mu sync.Mutex }

func (g *collectorNativeGuard) WithCurrent(ctx context.Context, _ localworkbridge.NativeReceiptScope, fn func(context.Context) error) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	return fn(ctx)
}
func collectorNativeRequest(id string) localworkbridge.Request {
	raw, _ := json.Marshal(map[string]any{"event_id": id, "event_type": "agent_manager.run.notification.v1", "source_scenario": "agent-manager", "delivered_at": "2026-11-07T11:59:59Z", "payload": map[string]any{"fixture": true}})
	mac := hmac.New(sha256.New, bytes.Repeat([]byte{7}, 32))
	mac.Write(raw)
	return localworkbridge.Request{Operation: "accept_native", Body: raw, EventID: id, IdempotencyKey: id, Signature: "sha256=" + hex.EncodeToString(mac.Sum(nil))}
}

type collectorStoppedFixture struct{ joined <-chan struct{} }

func (f collectorStoppedFixture) WithStoppedExecution(ctx context.Context, _ string, fn func(context.Context) error) error {
	select {
	case <-f.joined:
		return fn(ctx)
	case <-ctx.Done():
		return ctx.Err()
	}
}
