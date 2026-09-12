package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/authz"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/identity"
)

// registeredRoutes walks the live router and returns every (method, template)
// pair that has a handler. Prefix mounts report method "*".
func registeredRoutes(t *testing.T, router *mux.Router) []authz.Route {
	t.Helper()
	var out []authz.Route
	err := router.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		if route.GetHandler() == nil {
			return nil
		}
		template, err := route.GetPathTemplate()
		if err != nil {
			return nil
		}
		methods, err := route.GetMethods()
		if err != nil || len(methods) == 0 {
			methods = []string{"*"}
		}
		for _, method := range methods {
			out = append(out, authz.Route{Method: method, Path: template})
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk router: %v", err)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key() < out[j].Key() })
	return out
}

// TestRouteCensusMatchesAuthorizationTable [REQ:STC-P0-013] fails when a
// registered route is missing from the table, when a public row is outside
// the allowlist, or when a required table row is no longer registered.
func TestRouteCensusMatchesAuthorizationTable(t *testing.T) {
	srv := newCensusServer(t, "authz-census")
	registered := registeredRoutes(t, srv.router)
	if len(registered) < 80 {
		t.Fatalf("census walked only %d routes; the router is not fully registered", len(registered))
	}
	seen := map[string]struct{}{}
	for _, route := range registered {
		entry, ok := authz.Lookup(route.Method, route.Path)
		if !ok {
			t.Errorf("route %s is not classified in authz.Table", route.Key())
			continue
		}
		seen[entry.Key()] = struct{}{}
		if entry.Effect == authz.EffectPublic {
			if _, allowed := authz.PublicAllowlist[entry.Key()]; !allowed {
				t.Errorf("route %s is public but not allowlisted", route.Key())
			}
		}
	}
	for _, entry := range authz.Table {
		if entry.Optional {
			continue
		}
		if _, ok := seen[entry.Key()]; !ok {
			t.Errorf("table row %s is not registered on the live router", entry.Key())
		}
	}
	if findings := authz.Validate(); len(findings) > 0 {
		t.Fatalf("table invariants: %v", findings)
	}
}

// TestPublicHealthDisclosesNothingTargetSpecific [REQ:STC-P0-012] keeps the
// public allowlist honest: the health body carries no deployment or target data.
func TestPublicHealthDisclosesNothingTargetSpecific(t *testing.T) {
	srv := newTestServer()
	srv.testProvider.Anonymous = true
	for _, path := range []string{"/health", "/api/v1/health"} {
		rec := httptest.NewRecorder()
		srv.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK && rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s status = %d", path, rec.Code)
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("%s body: %v", path, err)
		}
		for _, forbidden := range []string{"deployment", "target", "host", "manifest", "secret"} {
			for key := range body {
				if strings.Contains(strings.ToLower(key), forbidden) {
					t.Fatalf("%s discloses %q", path, key)
				}
			}
		}
	}
}

func concretePath(route authz.Route) string {
	path := route.Path
	for placeholder, value := range map[string]string{
		"{id}": "dep-1", "{sha256}": "0123abcd", "{key}": "API_KEY", "{scenario}": "demo-app",
		"{action}": "start", "{invId}": "inv-1", "{taskId}": "task-1", "{operation_id}": "op-1",
	} {
		path = strings.ReplaceAll(path, placeholder, value)
	}
	if route.Prefix {
		path += "Probe"
	}
	return path
}

func concreteMethod(route authz.Route) string {
	if route.Method == "*" {
		return http.MethodPost
	}
	return route.Method
}

func boundaryRequest(route authz.Route, headers map[string]string) *http.Request {
	req := httptest.NewRequest(concreteMethod(route), "http://example.com"+concretePath(route), strings.NewReader(`{"canary":"canary-secret-9f3a"}`))
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	return req
}

// newCensusServer wires the real repository (so the Connect mounts exist) and
// seeds one production deployment "dep-1" on 203.0.113.10.
func newCensusServer(t *testing.T, name string) *Server {
	t.Helper()
	srv, repo := newIdentityTestServer(t, name)
	seedDeployment(t, repo, "dep-1", "demo-app", "production", "203.0.113.10", "demo.example")
	return srv
}

func nonPublicRoutes() []authz.Route {
	var out []authz.Route
	for _, route := range authz.Table {
		if route.Effect != authz.EffectPublic && !route.Optional {
			out = append(out, route)
		}
	}
	return out
}

// TestAnonymousIsRefusedOnEveryRouteWithoutEffects [REQ:STC-P0-012] sends an
// anonymous request to every non-public route through the real router and
// asserts a typed 401 with zero SSH calls and no handler side effects.
func TestAnonymousIsRefusedOnEveryRouteWithoutEffects(t *testing.T) {
	srv := newCensusServer(t, "authz-anonymous")
	sshFake := srv.sshRunner.(*FakeSSHRunner)
	srv.testProvider.Anonymous = true
	for _, route := range nonPublicRoutes() {
		rec := httptest.NewRecorder()
		srv.Router().ServeHTTP(rec, boundaryRequest(route, nil))
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s: status=%d body=%s", route.Key(), rec.Code, rec.Body.String())
			continue
		}
		typed := apierrors.FromHTTP(rec.Code, rec.Body.Bytes())
		if typed.Code != apierrors.CodeUnauthenticated || typed.NextAction == nil {
			t.Errorf("%s: typed=%+v", route.Key(), typed)
		}
		if strings.Contains(rec.Body.String(), "203.0.113.10") {
			t.Errorf("%s: host disclosed to anonymous caller", route.Key())
		}
	}
	if len(sshFake.Calls) != 0 {
		t.Fatalf("anonymous requests reached SSH: %v", sshFake.Calls)
	}
}

// TestWrongTargetGrantIsRefusedOnEveryTargetRoute [REQ:STC-P0-013] proves a
// principal granted a different environment cannot read, mutate, or reveal
// anything about a production deployment, including secrets.
func TestWrongTargetGrantIsRefusedOnEveryTargetRoute(t *testing.T) {
	srv := newCensusServer(t, "authz-wrong-target")
	sshFake := srv.sshRunner.(*FakeSSHRunner)
	staging := testOperator()
	staging.Subject = "staging-operator"
	staging.Source = identity.SourceScenarioAuthenticator
	srv.authz = newTestEnforcerWith(srv, staging, testEnforcerOptions{Policy: authz.PolicyDocument{Principals: map[string]authz.Grant{
		"staging-operator": {Environments: []string{"staging"}},
	}}})
	srv.router = mux.NewRouter()
	srv.authzInstalled = false
	srv.setupRoutes()
	for _, route := range nonPublicRoutes() {
		if !route.TargetBound && !route.BodyTarget {
			continue
		}
		rec := httptest.NewRecorder()
		srv.Router().ServeHTTP(rec, boundaryRequest(route, map[string]string{"Origin": "http://example.com"}))
		if rec.Code != http.StatusForbidden || apierrors.FromHTTP(rec.Code, rec.Body.Bytes()).Code != apierrors.CodeForbiddenTarget {
			t.Errorf("%s: status=%d body=%s", route.Key(), rec.Code, rec.Body.String())
		}
		for _, secret := range []string{"203.0.113.10", "production", "demo.example", "host:"} {
			if strings.Contains(rec.Body.String(), secret) {
				t.Errorf("%s: disclosed %q", route.Key(), secret)
			}
		}
	}
	if len(sshFake.Calls) != 0 {
		t.Fatalf("wrong-target requests reached SSH: %v", sshFake.Calls)
	}
}

// TestCrossOriginMutationIsRefusedOnEveryRoute [REQ:STC-P0-014] sends a
// cross-origin request to every state-changing route.
func TestCrossOriginMutationIsRefusedOnEveryRoute(t *testing.T) {
	srv := newCensusServer(t, "authz-cross-origin")
	sshFake := srv.sshRunner.(*FakeSSHRunner)
	for _, route := range nonPublicRoutes() {
		method := concreteMethod(route)
		if method == http.MethodGet && route.Effect != authz.EffectInteractive {
			continue
		}
		rec := httptest.NewRecorder()
		srv.Router().ServeHTTP(rec, boundaryRequest(route, map[string]string{"Origin": "https://attacker.example"}))
		if rec.Code != http.StatusForbidden || apierrors.FromHTTP(rec.Code, rec.Body.Bytes()).Code != apierrors.CodeForbiddenOrigin {
			t.Errorf("%s: status=%d body=%s", route.Key(), rec.Code, rec.Body.String())
		}
	}
	if len(sshFake.Calls) != 0 {
		t.Fatalf("cross-origin requests reached SSH: %v", sshFake.Calls)
	}
}

// TestTerminalUpgradeIsRefusedBeforeAnySession [REQ:STC-P0-014] covers the
// WebSocket route: no principal, foreign origin, and missing origin are all
// refused before an SSH session could start.
func TestTerminalUpgradeIsRefusedBeforeAnySession(t *testing.T) {
	srv := newCensusServer(t, "authz-terminal")
	terminal, ok := authz.Lookup(http.MethodGet, "/api/v1/deployments/{id}/terminal")
	if !ok {
		t.Fatal("terminal route missing from table")
	}
	for _, tc := range []struct {
		name      string
		anonymous bool
		headers   map[string]string
		want      int
		code      string
	}{
		{"no principal", true, map[string]string{"Origin": "http://example.com"}, http.StatusUnauthorized, apierrors.CodeUnauthenticated},
		{"foreign origin", false, map[string]string{"Origin": "https://attacker.example"}, http.StatusForbidden, apierrors.CodeForbiddenOrigin},
		{"missing origin", false, nil, http.StatusForbidden, apierrors.CodeForbiddenOrigin},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv.testProvider.Anonymous = tc.anonymous
			req := boundaryRequest(terminal, tc.headers)
			req.Header.Set("Connection", "Upgrade")
			req.Header.Set("Upgrade", "websocket")
			rec := httptest.NewRecorder()
			srv.Router().ServeHTTP(rec, req)
			if rec.Code != tc.want || apierrors.FromHTTP(rec.Code, rec.Body.Bytes()).Code != tc.code {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

// TestServicePrincipalSeesOnlyDeclaredRows [REQ:STC-P0-013] proves a service
// principal with every scope is still confined to service-allowed rows.
func TestServicePrincipalSeesOnlyDeclaredRows(t *testing.T) {
	srv := newCensusServer(t, "authz-service")
	sshFake := srv.sshRunner.(*FakeSSHRunner)
	svc := testOperator()
	svc.Kind = identity.ActorService
	svc.Subject = "svc:monitor"
	svc.Source = identity.SourceScenarioAuthenticator
	srv.authz = newTestEnforcerWith(srv, svc, testEnforcerOptions{Policy: authz.PolicyDocument{Principals: map[string]authz.Grant{"svc:monitor": {Environments: []string{"*"}}}}})
	srv.router = mux.NewRouter()
	srv.authzInstalled = false
	srv.setupRoutes()
	for _, route := range nonPublicRoutes() {
		if route.RemoteReach && route.ServiceAllowed {
			// Allowed read rows that reach the target would call the SSH fake;
			// the boundary decision is what this test proves, so skip the effect.
			continue
		}
		rec := httptest.NewRecorder()
		srv.Router().ServeHTTP(rec, boundaryRequest(route, map[string]string{"Origin": "http://example.com"}))
		code := apierrors.FromHTTP(rec.Code, rec.Body.Bytes()).Code
		if route.ServiceAllowed {
			if rec.Code == http.StatusUnauthorized || (rec.Code == http.StatusForbidden && code == apierrors.CodeForbiddenScope) {
				t.Errorf("%s: service-allowed row refused: %d %s", route.Key(), rec.Code, rec.Body.String())
			}
			continue
		}
		if rec.Code != http.StatusForbidden || code != apierrors.CodeForbiddenScope {
			t.Errorf("%s: service principal admitted: %d %s", route.Key(), rec.Code, rec.Body.String())
		}
	}
	if len(sshFake.Calls) != 0 {
		t.Fatalf("service principal reached SSH: %v", sshFake.Calls)
	}
}

// TestRevokedPrincipalIsRefusedAtEffect [REQ:STC-P0-013] revokes the
// credential after admission and proves a gated host action refuses before
// any effect.
func TestRevokedPrincipalIsRefusedAtEffect(t *testing.T) {
	srv := newCensusServer(t, "authz-revoked")
	sshFake := srv.sshRunner.(*FakeSSHRunner)
	// The provider verifies once at admission; the second verification (the
	// effect recheck inside the gate) fails.
	srv.testProvider.RevokeAfterCalls = 1
	kill, _ := authz.Lookup(http.MethodPost, "/api/v1/deployments/{id}/actions/kill")
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, boundaryRequest(kill, map[string]string{"Origin": "http://example.com"}))
	if rec.Code != http.StatusForbidden || apierrors.FromHTTP(rec.Code, rec.Body.Bytes()).Code != apierrors.CodeForbiddenRevoked {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if len(sshFake.Calls) != 0 {
		t.Fatalf("revoked principal reached SSH: %v", sshFake.Calls)
	}
}

// TestAuditLineNeverCarriesRequestBody [REQ:STC-P0-012] plants a canary in a
// request body and asserts it never reaches the audit log.
func TestAuditLineNeverCarriesRequestBody(t *testing.T) {
	srv := newCensusServer(t, "authz-audit")
	var lines []string
	srv.authz = newTestEnforcerWith(srv, testOperator(), testEnforcerOptions{Logger: func(msg string, fields map[string]any) {
		lines = append(lines, fmt.Sprintf("%s %v", msg, fields))
	}})
	srv.router = mux.NewRouter()
	srv.authzInstalled = false
	srv.setupRoutes()
	history, _ := authz.Lookup(http.MethodPost, "/api/v1/deployments/{id}/history")
	rec := httptest.NewRecorder()
	req := boundaryRequest(history, map[string]string{"Authorization": "Bearer canary-token-77aa"})
	srv.Router().ServeHTTP(rec, req)
	if len(lines) == 0 {
		t.Fatal("no audit lines written")
	}
	for _, line := range lines {
		if strings.Contains(line, "canary-secret-9f3a") || strings.Contains(line, "canary-token-77aa") {
			t.Fatalf("audit leaked a credential: %s", line)
		}
	}
}
