package authz

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"scenario-to-cloud/apierrors"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
)

type fakeProvider struct {
	mu        sync.Mutex
	principal identity.Principal
	fail      error
	calls     int
}

func (f *fakeProvider) Source() identity.AuthSource { return identity.SourceScenarioAuthenticator }

func (f *fakeProvider) VerifyRequest(context.Context, *http.Request) (identity.Principal, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.fail != nil {
		return identity.Principal{}, f.fail
	}
	return f.principal, nil
}

func (f *fakeProvider) set(p identity.Principal, fail error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.principal, f.fail = p, fail
}

func human(subject string, scopes ...string) identity.Principal {
	return identity.Principal{Kind: identity.ActorHuman, Subject: subject, Verified: true, Source: identity.SourceScenarioAuthenticator, Scopes: scopes}
}

func service(subject string, scopes ...string) identity.Principal {
	return identity.Principal{Kind: identity.ActorService, Subject: subject, Verified: true, Source: identity.SourceScenarioAuthenticator, Scopes: scopes}
}

type harness struct {
	provider *fakeProvider
	enforcer *Enforcer
	router   *mux.Router
	handled  []string
	logs     []map[string]any
	targets  map[string]Target
	mu       sync.Mutex
}

func newHarness(t *testing.T, policy *Policy, tweak func(*Config)) *harness {
	t.Helper()
	h := &harness{provider: &fakeProvider{}, targets: map[string]Target{
		"dep-prod":  {DeploymentID: "dep-prod", Environment: "production", Key: "host:203.0.113.10"},
		"dep-stage": {DeploymentID: "dep-stage", Environment: "staging", Key: "machine:m-2"},
	}}
	cfg := Config{
		Authn:        authn.Config{Providers: []authn.Provider{h.provider}},
		Mode:         ModeShared,
		Policy:       policy,
		AllowedHosts: []string{"example.com"},
		BindLoopback: true,
		Logger: func(msg string, fields map[string]any) {
			h.mu.Lock()
			defer h.mu.Unlock()
			fields["msg"] = msg
			h.logs = append(h.logs, fields)
		},
	}
	if tweak != nil {
		tweak(&cfg)
	}
	enforcer, err := New(cfg, TargetResolverFunc(func(_ context.Context, id string) (Target, bool, error) {
		target, ok := h.targets[id]
		return target, ok, nil
	}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	h.enforcer = enforcer
	h.router = mux.NewRouter()
	h.router.Use(enforcer.Middleware)
	record := func(name string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			h.mu.Lock()
			h.handled = append(h.handled, name)
			h.mu.Unlock()
			w.WriteHeader(http.StatusOK)
		}
	}
	effect := func(name string, effect Effect) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if denied := enforcer.RequireEffect(r.Context(), mux.Vars(r)["id"], effect); denied != nil {
				apierrors.Write(w, denied)
				return
			}
			record(name)(w, r)
		}
	}
	h.router.HandleFunc("/health", record("health")).Methods("GET")
	api := h.router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/deployments", record("list")).Methods("GET")
	api.HandleFunc("/deployments/{id}", record("get")).Methods("GET")
	api.HandleFunc("/deployments/{id}/execute", effect("execute", EffectWorkloadMutation)).Methods("POST")
	api.HandleFunc("/deployments/{id}/secrets", effect("secrets", EffectSecret)).Methods("GET")
	api.HandleFunc("/deployments/{id}/terminal", record("terminal")).Methods("GET")
	api.HandleFunc("/preflight", record("preflight")).Methods("POST")
	api.HandleFunc("/bundle/build", effect("build", EffectWorkloadMutation)).Methods("POST")
	return h
}

func (h *harness) do(method, path string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "http://example.com"+path, strings.NewReader(`{"canary":"canary-secret-9f3a"}`))
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	rec := httptest.NewRecorder()
	h.router.ServeHTTP(rec, req)
	return rec
}

func codeOf(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	if rec.Code < 400 {
		return ""
	}
	return apierrors.FromHTTP(rec.Code, rec.Body.Bytes()).Code
}

func (h *harness) handledNames() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.handled...)
}

// [REQ:STC-P0-013] The table is internally consistent.
func TestTableValidates(t *testing.T) {
	if findings := Validate(); len(findings) > 0 {
		t.Fatalf("table findings: %v", findings)
	}
	for _, route := range Table {
		if route.Effect == EffectPublic {
			if _, ok := PublicAllowlist[route.Key()]; !ok {
				t.Fatalf("%s public outside allowlist", route.Key())
			}
		}
	}
	if _, ok := Lookup("POST", DeploymentsServicePrefix); !ok {
		t.Fatalf("prefix mount must match any method")
	}
	if _, ok := Lookup("GET", "/api/v1/not-a-route"); ok {
		t.Fatalf("unknown route must not resolve")
	}
}

// [REQ:STC-P0-012] Anonymous callers are refused on every non-public route
// before any handler runs.
func TestAnonymousIsRefusedBeforeHandlers(t *testing.T) {
	h := newHarness(t, NewStaticPolicy(PolicyDocument{}), nil)
	h.provider.set(identity.Principal{}, identity.NewFailure(identity.FailureMissing, identity.SourceScenarioAuthenticator))
	for _, tc := range []struct{ method, path string }{
		{"GET", "/api/v1/deployments"},
		{"GET", "/api/v1/deployments/dep-prod"},
		{"POST", "/api/v1/deployments/dep-prod/execute"},
		{"GET", "/api/v1/deployments/dep-prod/secrets"},
		{"GET", "/api/v1/deployments/dep-prod/terminal"},
		{"POST", "/api/v1/preflight"},
	} {
		rec := h.do(tc.method, tc.path, nil)
		if rec.Code != http.StatusUnauthorized || codeOf(t, rec) != apierrors.CodeUnauthenticated {
			t.Fatalf("%s %s: status=%d body=%s", tc.method, tc.path, rec.Code, rec.Body.String())
		}
		typed := apierrors.FromHTTP(rec.Code, rec.Body.Bytes())
		if typed.NextAction == nil || typed.NextAction.Kind != "sign_in" {
			t.Fatalf("%s %s: missing sign-in next action: %+v", tc.method, tc.path, typed)
		}
	}
	if got := h.handledNames(); len(got) != 0 {
		t.Fatalf("handlers ran for anonymous callers: %v", got)
	}
	if rec := h.do("GET", "/health", nil); rec.Code != http.StatusOK {
		t.Fatalf("public health must stay reachable: %d", rec.Code)
	}
}

// [REQ:STC-P0-013] Scope is enforced through the shared catalog grammar.
func TestScopeIsRequired(t *testing.T) {
	h := newHarness(t, NewStaticPolicy(PolicyDocument{Principals: map[string]Grant{"alice": {Environments: []string{"*"}}}}), nil)
	h.provider.set(human("alice", ScopeRead), nil)
	if rec := h.do("GET", "/api/v1/deployments/dep-prod", nil); rec.Code != http.StatusOK {
		t.Fatalf("read with read scope: %d %s", rec.Code, rec.Body.String())
	}
	rec := h.do("POST", "/api/v1/deployments/dep-prod/execute", nil)
	if rec.Code != http.StatusForbidden || codeOf(t, rec) != apierrors.CodeForbiddenScope {
		t.Fatalf("execute with read scope: %d %s", rec.Code, rec.Body.String())
	}
	h.provider.set(human("alice", Scenario+":*"), nil)
	if rec := h.do("POST", "/api/v1/deployments/dep-prod/execute", nil); rec.Code != http.StatusOK {
		t.Fatalf("execute with namespace wildcard: %d %s", rec.Code, rec.Body.String())
	}
	if got := h.handledNames(); len(got) != 2 {
		t.Fatalf("handled = %v", got)
	}
}

// [REQ:STC-P0-013] A grant for another target is refused without disclosing
// the deployment's environment, key or host; a missing id looks the same.
func TestWrongTargetGrantIsRefusedWithoutDisclosure(t *testing.T) {
	h := newHarness(t, NewStaticPolicy(PolicyDocument{Principals: map[string]Grant{"bob": {Environments: []string{"staging"}}}}), nil)
	h.provider.set(human("bob", ScopeRead, ScopeDestructive), nil)
	if rec := h.do("GET", "/api/v1/deployments/dep-stage", nil); rec.Code != http.StatusOK {
		t.Fatalf("granted environment: %d %s", rec.Code, rec.Body.String())
	}
	for _, path := range []string{"/api/v1/deployments/dep-prod", "/api/v1/deployments/dep-prod/secrets", "/api/v1/deployments/does-not-exist"} {
		rec := h.do("GET", path, nil)
		if rec.Code != http.StatusForbidden || codeOf(t, rec) != apierrors.CodeForbiddenTarget {
			t.Fatalf("%s: %d %s", path, rec.Code, rec.Body.String())
		}
		body := rec.Body.String()
		for _, secret := range []string{"203.0.113.10", "production", "host:", "dep-prod"} {
			if strings.Contains(body, secret) {
				t.Fatalf("%s disclosed %q: %s", path, secret, body)
			}
		}
	}
	// Body-addressed legacy routes are closed to restricted principals.
	rec := h.do("POST", "/api/v1/preflight", nil)
	if rec.Code != http.StatusForbidden || codeOf(t, rec) != apierrors.CodeForbiddenTarget {
		t.Fatalf("body target: %d %s", rec.Code, rec.Body.String())
	}
	if got := h.handledNames(); len(got) != 1 || got[0] != "get" {
		t.Fatalf("handled = %v", got)
	}
}

// [REQ:STC-P0-013] A credential revoked between admission and effect is
// refused at the effect boundary.
func TestRevokedBetweenAdmissionAndEffect(t *testing.T) {
	h := newHarness(t, NewStaticPolicy(PolicyDocument{Principals: map[string]Grant{"carol": {Machines: []string{"*"}}}}), nil)
	h.provider.set(human("carol", ScopeDestructive), nil)
	// The provider verifies at admission, then the handler's RequireEffect
	// re-verifies; flip the provider after the first call.
	first := true
	h.router = mux.NewRouter()
	h.router.Use(h.enforcer.Middleware)
	h.router.HandleFunc("/api/v1/deployments/{id}/execute", func(w http.ResponseWriter, r *http.Request) {
		if first {
			first = false
			h.provider.set(identity.Principal{}, identity.NewFailure(identity.FailureInvalid, identity.SourceScenarioAuthenticator))
		}
		if denied := h.enforcer.RequireEffect(r.Context(), mux.Vars(r)["id"], EffectWorkloadMutation); denied != nil {
			apierrors.Write(w, denied)
			return
		}
		h.handled = append(h.handled, "execute")
		w.WriteHeader(http.StatusOK)
	}).Methods("POST")
	rec := h.do("POST", "/api/v1/deployments/dep-prod/execute", nil)
	if rec.Code != http.StatusForbidden || codeOf(t, rec) != apierrors.CodeForbiddenRevoked {
		t.Fatalf("revoked credential: %d %s", rec.Code, rec.Body.String())
	}
	if len(h.handled) != 0 {
		t.Fatalf("effect ran after revocation")
	}

	// Operator revocation through the policy file is honoured on the next
	// request without a restart.
	dir := t.TempDir()
	path := filepath.Join(dir, "authz-policy.json")
	write := func(doc PolicyDocument) {
		raw, _ := json.Marshal(doc)
		if err := os.WriteFile(path, raw, 0o600); err != nil {
			t.Fatal(err)
		}
		// Ensure a distinct mtime for the reload check.
		future := time.Now().Add(2 * time.Second)
		_ = os.Chtimes(path, future, future)
	}
	write(PolicyDocument{Principals: map[string]Grant{"carol": {Machines: []string{"*"}}}})
	h2 := newHarness(t, NewPolicy(path), nil)
	h2.provider.set(human("carol", ScopeDestructive), nil)
	if rec := h2.do("POST", "/api/v1/deployments/dep-prod/execute", nil); rec.Code != http.StatusOK {
		t.Fatalf("before revoke: %d %s", rec.Code, rec.Body.String())
	}
	write(PolicyDocument{Principals: map[string]Grant{"carol": {Machines: []string{"*"}}}, Revoked: []string{"carol"}})
	rec = h2.do("POST", "/api/v1/deployments/dep-prod/execute", nil)
	if rec.Code != http.StatusForbidden || codeOf(t, rec) != apierrors.CodeForbiddenRevoked {
		t.Fatalf("after revoke: %d %s", rec.Code, rec.Body.String())
	}
}

// [REQ:STC-P0-014] Expired credentials are refused at admission, and an
// interactive session context is bounded by the credential expiry.
func TestExpiredCredential(t *testing.T) {
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	h := newHarness(t, NewStaticPolicy(PolicyDocument{Principals: map[string]Grant{"dan": {Environments: []string{"*"}}}}), func(c *Config) {
		c.Now = func() time.Time { return now }
	})
	p := human("dan", ScopeRead, ScopeDestructive)
	p.ExpiresAt = now.Add(-time.Minute)
	h.provider.set(p, nil)
	rec := h.do("GET", "/api/v1/deployments/dep-prod", nil)
	if rec.Code != http.StatusUnauthorized || codeOf(t, rec) != apierrors.CodeUnauthenticated {
		t.Fatalf("expired at admission: %d %s", rec.Code, rec.Body.String())
	}
	if reason := apierrors.FromHTTP(rec.Code, rec.Body.Bytes()).Details["reason"]; reason != string(identity.FailureExpired) {
		t.Fatalf("reason = %v", reason)
	}

	p.ExpiresAt = now.Add(time.Hour)
	h.provider.set(p, nil)
	var captured Decision
	h.router.HandleFunc("/api/v1/deployments/{id}/terminal-probe", func(w http.ResponseWriter, r *http.Request) {
		captured, _ = DecisionFromContext(r.Context())
	}).Methods("GET")
	Table = append(Table, Route{Method: "GET", Path: "/api/v1/deployments/{id}/terminal-probe", Effect: EffectRead, Scope: ScopeRead, TargetBound: true, Optional: true})
	index = buildIndex()
	t.Cleanup(func() {
		Table = Table[:len(Table)-1]
		index = buildIndex()
	})
	if rec := h.do("GET", "/api/v1/deployments/dep-prod/terminal-probe", nil); rec.Code != http.StatusOK {
		t.Fatalf("valid credential: %d %s", rec.Code, rec.Body.String())
	}
	ctx, cancel := h.enforcer.SessionContext(context.WithValue(context.Background(), decisionKey{}, captured))
	defer cancel()
	deadline, ok := ctx.Deadline()
	if !ok || !deadline.Equal(p.ExpiresAt) {
		t.Fatalf("session deadline = %v (%v), want %v", deadline, ok, p.ExpiresAt)
	}
}

// [REQ:STC-P0-014] Cross-origin state changes and unauthenticated or
// foreign-origin interactive upgrades are refused.
func TestBrowserBoundary(t *testing.T) {
	h := newHarness(t, NewStaticPolicy(PolicyDocument{Principals: map[string]Grant{"erin": {Environments: []string{"*"}}}}), func(c *Config) {
		c.AllowedOrigins = []string{"https://ops.example.net"}
	})
	h.provider.set(human("erin", ScopeRead, ScopeDestructive), nil)
	cases := []struct {
		name    string
		method  string
		path    string
		headers map[string]string
		want    int
		code    string
	}{
		{"same origin post", "POST", "/api/v1/deployments/dep-prod/execute", map[string]string{"Origin": "http://example.com"}, 200, ""},
		{"allowed origin post", "POST", "/api/v1/deployments/dep-prod/execute", map[string]string{"Origin": "https://ops.example.net"}, 200, ""},
		{"loopback origin on loopback bind", "POST", "/api/v1/deployments/dep-prod/execute", map[string]string{"Origin": "http://localhost:23013"}, 200, ""},
		{"cross origin post", "POST", "/api/v1/deployments/dep-prod/execute", map[string]string{"Origin": "https://attacker.example"}, 403, apierrors.CodeForbiddenOrigin},
		{"cross-site fetch metadata", "POST", "/api/v1/deployments/dep-prod/execute", map[string]string{"Sec-Fetch-Site": "cross-site"}, 403, apierrors.CodeForbiddenOrigin},
		{"non-browser post", "POST", "/api/v1/deployments/dep-prod/execute", nil, 200, ""},
		{"cross origin get is read only", "GET", "/api/v1/deployments/dep-prod", map[string]string{"Origin": "https://attacker.example"}, 200, ""},
		{"terminal without origin", "GET", "/api/v1/deployments/dep-prod/terminal", nil, 403, apierrors.CodeForbiddenOrigin},
		{"terminal foreign origin", "GET", "/api/v1/deployments/dep-prod/terminal", map[string]string{"Origin": "https://attacker.example"}, 403, apierrors.CodeForbiddenOrigin},
		{"terminal same origin", "GET", "/api/v1/deployments/dep-prod/terminal", map[string]string{"Origin": "http://example.com"}, 200, ""},
		{"foreign host header", "GET", "/api/v1/deployments/dep-prod", map[string]string{"Host": "rebind.attacker"}, 403, apierrors.CodeForbiddenHost},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "http://example.com"+tc.path, nil)
			for key, value := range tc.headers {
				if key == "Host" {
					req.Host = value
					continue
				}
				req.Header.Set(key, value)
			}
			rec := httptest.NewRecorder()
			h.router.ServeHTTP(rec, req)
			if rec.Code != tc.want || codeOf(t, rec) != tc.code {
				t.Fatalf("status=%d code=%q body=%s", rec.Code, codeOf(t, rec), rec.Body.String())
			}
		})
	}
	before := len(h.handledNames())
	h.provider.set(identity.Principal{}, identity.NewFailure(identity.FailureMissing, identity.SourceScenarioAuthenticator))
	rec := h.do("GET", "/api/v1/deployments/dep-prod/terminal", map[string]string{"Origin": "http://example.com"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous terminal: %d", rec.Code)
	}
	if len(h.handledNames()) != before {
		t.Fatalf("terminal handler ran without a principal")
	}
}

// [REQ:STC-P0-013] Service principals may use only rows that allow them.
func TestServicePrincipalIsLimitedToDeclaredRows(t *testing.T) {
	h := newHarness(t, NewStaticPolicy(PolicyDocument{Principals: map[string]Grant{"svc:monitor": {Environments: []string{"*"}}}}), nil)
	h.provider.set(service("svc:monitor", ScopeRead, ScopeWrite, ScopeDestructive), nil)
	if rec := h.do("GET", "/api/v1/deployments", nil); rec.Code != http.StatusOK {
		t.Fatalf("service list: %d %s", rec.Code, rec.Body.String())
	}
	if rec := h.do("GET", "/api/v1/deployments/dep-prod", nil); rec.Code != http.StatusOK {
		t.Fatalf("service get: %d %s", rec.Code, rec.Body.String())
	}
	for _, tc := range []struct{ method, path string }{
		{"POST", "/api/v1/deployments/dep-prod/execute"},
		{"GET", "/api/v1/deployments/dep-prod/secrets"},
		{"GET", "/api/v1/deployments/dep-prod/terminal"},
		{"POST", "/api/v1/preflight"},
		{"POST", "/api/v1/bundle/build"},
	} {
		rec := h.do(tc.method, tc.path, map[string]string{"Origin": "http://example.com"})
		if rec.Code != http.StatusForbidden || codeOf(t, rec) != apierrors.CodeForbiddenScope {
			t.Fatalf("service %s %s: %d %s", tc.method, tc.path, rec.Code, rec.Body.String())
		}
	}
	if got := h.handledNames(); len(got) != 2 {
		t.Fatalf("handled = %v", got)
	}
	for _, route := range Table {
		if route.ServiceAllowed && (route.Effectful() || route.NoStore) {
			t.Fatalf("%s must not be service-allowed", route.Key())
		}
	}
}

// [REQ:STC-P0-012] Bounded bodies, per-principal effect concurrency, and
// no-store on credential responses.
func TestBoundedRequests(t *testing.T) {
	h := newHarness(t, NewStaticPolicy(PolicyDocument{Principals: map[string]Grant{"fay": {Environments: []string{"*"}}}}), func(c *Config) {
		c.MaxBodyBytes = 64
		c.EffectConcurrency = 1
	})
	h.provider.set(human("fay", ScopeDestructive), nil)
	big := strings.Repeat("x", 128)
	req := httptest.NewRequest("POST", "http://example.com/api/v1/deployments/dep-prod/execute", strings.NewReader(big))
	rec := httptest.NewRecorder()
	h.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge || codeOf(t, rec) != apierrors.CodeRequestTooLarge {
		t.Fatalf("oversized body: %d %s", rec.Code, rec.Body.String())
	}
	rec = h.do("GET", "/api/v1/deployments/dep-prod/secrets", nil)
	if rec.Code != http.StatusOK || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("secrets no-store: %d %q", rec.Code, rec.Header().Get("Cache-Control"))
	}

	release := make(chan struct{})
	started := make(chan struct{})
	h.router = mux.NewRouter()
	h.router.Use(h.enforcer.Middleware)
	h.router.HandleFunc("/api/v1/deployments/{id}/execute", func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		w.WriteHeader(http.StatusOK)
	}).Methods("POST")
	go func() {
		h.do("POST", "/api/v1/deployments/dep-prod/execute", nil)
	}()
	<-started
	rec = h.do("POST", "/api/v1/deployments/dep-prod/execute", nil)
	close(release)
	if rec.Code != http.StatusTooManyRequests || codeOf(t, rec) != apierrors.CodeTooManyRequests {
		t.Fatalf("concurrency: %d %s", rec.Code, rec.Body.String())
	}
}

// [REQ:STC-P0-012] Audit lines carry actor, scope, target and outcome and
// never the request body or credential values.
func TestAuditNeverCarriesCredentials(t *testing.T) {
	h := newHarness(t, NewStaticPolicy(PolicyDocument{Principals: map[string]Grant{"gus": {Environments: []string{"*"}}}}), nil)
	h.provider.set(human("gus", ScopeDestructive), nil)
	rec := h.do("POST", "/api/v1/deployments/dep-prod/execute", map[string]string{"Authorization": "Bearer canary-token-77aa"})
	if rec.Code != http.StatusOK {
		t.Fatalf("execute: %d %s", rec.Code, rec.Body.String())
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	var effect map[string]any
	for _, line := range h.logs {
		if line["msg"] == "authz.effect" {
			effect = line
		}
		raw, _ := json.Marshal(line)
		for _, canary := range []string{"canary-secret-9f3a", "canary-token-77aa"} {
			if strings.Contains(string(raw), canary) {
				t.Fatalf("audit line leaked %q: %s", canary, raw)
			}
		}
	}
	if effect == nil {
		t.Fatalf("no authz.effect line: %v", h.logs)
	}
	for _, key := range []string{"actor", "source", "scope", "deployment_id", "target_key", "outcome"} {
		if _, ok := effect[key]; !ok {
			t.Fatalf("effect line missing %s: %v", key, effect)
		}
	}
	if effect["actor"] != "gus" || effect["deployment_id"] != "dep-prod" || effect["outcome"] != "admitted" {
		t.Fatalf("effect line = %v", effect)
	}
}

// [REQ:STC-P0-012] Configuration never resolves to an anonymous boundary.
func TestFromEnvironmentNeverAnonymous(t *testing.T) {
	env := func(values map[string]string) func(string) string {
		return func(key string) string { return values[key] }
	}
	cfg, err := FromEnvironment(env(map[string]string{}))
	if err != nil || cfg.Mode != ModePersonalLocal || len(cfg.Authn.Providers) != 1 || cfg.Authn.Providers[0].Source() != identity.SourcePersonalLocal {
		t.Fatalf("default: cfg=%+v err=%v", cfg, err)
	}
	if !cfg.BindLoopback || cfg.MaxBodyBytes != DefaultMaxBodyBytes {
		t.Fatalf("defaults: %+v", cfg)
	}
	if _, err := FromEnvironment(env(map[string]string{EnvAuthMode: "shared"})); err == nil {
		t.Fatalf("shared mode without providers must fail")
	}
	if _, err := FromEnvironment(env(map[string]string{EnvAuthMode: "anonymous"})); err == nil {
		t.Fatalf("unknown mode must fail")
	}
	if _, err := FromEnvironment(env(map[string]string{EnvBindAddress: "0.0.0.0"})); err == nil {
		t.Fatalf("non-loopback bind without allowed hosts must fail")
	}
	cfg, err = FromEnvironment(env(map[string]string{EnvBindAddress: "0.0.0.0", EnvAllowedHosts: "cloud.example.test", EnvAllowedOrigins: "https://cloud.example.test"}))
	if err != nil || cfg.BindLoopback || len(cfg.AllowedHosts) != 1 || len(cfg.AllowedOrigins) != 1 {
		t.Fatalf("exposed bind: cfg=%+v err=%v", cfg, err)
	}
	if _, err := New(Config{Authn: authn.Config{}, BindLoopback: true}, TargetResolverFunc(func(context.Context, string) (Target, bool, error) { return Target{}, false, nil })); err == nil {
		t.Fatalf("New must refuse an empty provider chain")
	}
}

// Personal-local owners are unrestricted by default; every other unnamed
// principal has no target grant.
func TestPolicyDefaults(t *testing.T) {
	policy := NewPolicy(filepath.Join(t.TempDir(), "missing.json"))
	owner := identity.Principal{Kind: identity.ActorHuman, Subject: "osuser:1000", Verified: true, Source: identity.SourcePersonalLocal}
	other := human("zed", ScopeRead)
	target := Target{DeploymentID: "d", Environment: "production", Key: "host:h"}
	if ok, _ := policy.Allows(owner, target); !ok {
		t.Fatalf("owner must be unrestricted without a policy file")
	}
	if ok, _ := policy.Allows(other, target); ok {
		t.Fatalf("unnamed shared principal must have no grant")
	}
	named := NewStaticPolicy(PolicyDocument{Principals: map[string]Grant{"osuser:1000": {Environments: []string{"staging"}}}})
	if ok, _ := named.Allows(owner, target); ok {
		t.Fatalf("a named owner is restricted to the named grant")
	}
	if ok, _ := named.Allows(owner, Target{Environment: "staging"}); !ok {
		t.Fatalf("named grant must cover its environment")
	}
	if ok, _ := NewStaticPolicy(PolicyDocument{Principals: map[string]Grant{"zed": {Machines: []string{"host:h"}}}}).Allows(other, target); !ok {
		t.Fatalf("machine grant must cover its key")
	}
}
