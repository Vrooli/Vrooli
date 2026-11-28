package access_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/cli-core/cliutil"

	"money-ledger/internal/access"
	"money-ledger/internal/module"
	"money-ledger/internal/modules"
	"money-ledger/internal/server"
	"money-ledger/internal/testutil/authfixture"
	"money-ledger/internal/testutil/ledgertest"
)

// Every RPC the API serves is declared in the one policy, so a new RPC is
// refused until someone decides who may call it.
func TestEveryServedProcedureIsDeclared(t *testing.T) {
	served := map[string]bool{}
	for _, entry := range modules.AllProtoFiles() {
		services := entry.File.Services()
		for i := 0; i < services.Len(); i++ {
			methods := services.Get(i).Methods()
			for j := 0; j < methods.Len(); j++ {
				procedure := "/" + string(services.Get(i).FullName()) + "/" + string(methods.Get(j).Name())
				served[procedure] = true
				_, declared := access.Policy[procedure]
				require.True(t, declared, "%s is served but not declared in access.Policy", procedure)
			}
		}
	}
	for _, endpoint := range modules.AllEndpoints() {
		_, declared := access.Policy[endpoint.Path]
		require.True(t, declared, "%s %s is served but not declared in access.Policy", endpoint.Method, endpoint.Path)
		served[endpoint.Path] = true
	}
	for path, rule := range access.Policy {
		require.True(t, served[path] || path == "/api/v1/health", "access.Policy declares %s, which no module serves", path)
		if rule.Effect == access.Write {
			require.NotEmpty(t, rule.Callers, "write %s must name its callers", path)
		}
	}
}

// A route that reaches the server without a declaration is refused before its
// handler runs, whichever module mounts it.
func TestUndeclaredRouteIsRefused(t *testing.T) {
	reached := false
	undeclared := module.Module{Name: "undeclared", Mount: func(r *mux.Router) {
		r.HandleFunc("/api/v1/undeclared", func(w http.ResponseWriter, _ *http.Request) { reached = true }).Methods(http.MethodPost)
	}}
	srv := server.New(server.Deps{Clock: schedule.System(), Logger: log.New(io.Discard, "", 0), Gate: access.NewGate(authn.Config{})}, undeclared)
	live := httptest.NewServer(srv.Handler())
	defer live.Close()
	response, err := live.Client().Post(live.URL+"/api/v1/undeclared", "application/json", strings.NewReader("{}"))
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, http.StatusForbidden, response.StatusCode)
	require.False(t, reached)
}

// caller is one way a request can present itself.
type caller struct {
	name    string
	headers map[string]string
	actor   access.Actor // the actor the gate resolves, or "" when refused
	code    connect.Code // the refusal code when actor is ""
}

func callers(id *authfixture.Identity) []caller {
	hour := time.Now().Add(time.Hour)
	bearer := func(token string) map[string]string { return map[string]string{"Authorization": "Bearer " + token} }
	return []caller{
		{name: "anonymous", code: connect.CodeUnauthenticated},
		{name: "forged bearer", headers: bearer("not-a-signed-token"), code: connect.CodeUnauthenticated},
		{name: "expired owner session", headers: bearer(id.Token(authfixture.OwnerSubject, "human", authfixture.Audience, time.Now().Add(-time.Hour))), code: connect.CodeUnauthenticated},
		{name: "owner session for another audience", headers: bearer(id.Token(authfixture.OwnerSubject, "human", "scenario-authenticator:treasury", hour)), code: connect.CodeUnauthenticated},
		{name: "forged access assertion", headers: map[string]string{"Cf-Access-Jwt-Assertion": "forged"}, code: connect.CodeUnauthenticated},
		{name: "expired access assertion", headers: map[string]string{"Cf-Access-Jwt-Assertion": id.Access(authfixture.OwnerAccessSubject, "owner@example.test", time.Now().Add(-time.Hour))}, code: connect.CodeUnauthenticated},
		// Signing in is not ownership: the identity provider holds other
		// accounts, and they can grant themselves scopes.
		{name: "another person holding the write scope", headers: bearer(id.Person("someone-else", authfixture.WriteScope)), code: connect.CodePermissionDenied},
		{name: "another access user", headers: map[string]string{"Cf-Access-Jwt-Assertion": id.Access("someone-else", "else@example.test", hour)}, code: connect.CodePermissionDenied},
		{name: "undeclared service", headers: bearer(id.Service("someone-else", authfixture.WriteScope)), code: connect.CodePermissionDenied},
		{name: "treasury principal for another peer", headers: bearer(id.Service("treasury-agent-manager", authfixture.WriteScope)), code: connect.CodePermissionDenied},
		{name: "declared service without the write scope", headers: bearer(id.Service(authfixture.TreasuryWriter, "money-ledger:read")), code: connect.CodePermissionDenied},
		{name: "a person named like a declared service", headers: bearer(id.Person(authfixture.TreasuryWriter, authfixture.WriteScope)), code: connect.CodePermissionDenied},
		{name: "owner session (CLI --local-owner)", headers: bearer(id.Owner()), actor: access.Operator},
		{name: "owner access assertion (tunnel browser)", headers: map[string]string{"Cf-Access-Jwt-Assertion": id.OwnerAccess()}, actor: access.Operator},
		{name: "treasury", headers: bearer(id.Service(authfixture.TreasuryWriter, authfixture.WriteScope)), actor: access.Treasury},
		{name: "landing page business suite", headers: bearer(id.Service(authfixture.LandingPageWriter, authfixture.WriteScope)), actor: access.LandingPageBusinessSuite},
		{name: "compute manager invoice writer with only its exact scope", headers: bearer(id.Service(authfixture.ComputeManagerInvoiceWriter, authfixture.ComputeInvoiceScope)), actor: access.ComputeManagerInvoiceWriter},
		{name: "compute manager invoice writer with broad scope only", headers: bearer(id.Service(authfixture.ComputeManagerInvoiceWriter, authfixture.WriteScope)), code: connect.CodePermissionDenied},
	}
}

// Every restricted route admits exactly the actors its rule names and refuses
// every other caller before the handler runs, storing nothing.
func TestEveryRestrictedRouteAdmitsOnlyItsCallers(t *testing.T) {
	ledger := ledgertest.New(t)
	paths := make([]string, 0, len(access.Policy))
	for path, rule := range access.Policy {
		if !rule.Open() {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	require.NotEmpty(t, paths)
	for _, path := range paths {
		rule := access.Policy[path]
		for _, c := range callers(ledger.Identity) {
			t.Run(path+"/"+c.name, func(t *testing.T) {
				code := post(t, ledger.URL+path, c.headers)
				switch {
				case c.actor == "":
					require.Equal(t, c.code, code)
				case rule.Admits(c.actor):
					// Admitted: the empty request reaches the handler, which
					// rejects it on its merits, never as an identity refusal.
					require.NotEqual(t, connect.CodeUnauthenticated, code)
					require.NotEqual(t, connect.CodePermissionDenied, code)
				default:
					require.Equal(t, connect.CodePermissionDenied, code)
				}
			})
		}
	}
	for _, table := range []string{"postings", "crypto_receipts", "goals", "ledger_audit"} {
		require.Zero(t, ledger.Count(t, table), "%s must stay empty: no request above carried a valid write", table)
	}
	require.Equal(t, 1, ledger.Count(t, "adapters"), "only the built-in Compute Manager invoice adapter is present")
}

// Open reads need no identity, and a presented credential that fails cannot
// turn a read into a refusal.
func TestOpenReadsNeedNoIdentity(t *testing.T) {
	ledger := ledgertest.New(t)
	for path, rule := range access.Policy {
		if !rule.Open() || !strings.HasPrefix(path, "/vrooli.") {
			continue
		}
		for _, headers := range []map[string]string{nil, {"Authorization": "Bearer forged"}} {
			code := post(t, ledger.URL+path, headers)
			require.NotEqual(t, connect.CodeUnauthenticated, code, path)
			require.NotEqual(t, connect.CodePermissionDenied, code, path)
		}
	}
}

// An agent run is never the owner, even carrying the owner's own bearer:
// unverified, the shared middleware refuses it; verified, the gate does.
func TestAgentRunsCannotWrite(t *testing.T) {
	verifier := provenance.VerifierFunc(func(token string) (*cliutil.VerifyResult, error) {
		return &cliutil.VerifyResult{Valid: token == "run-token", Claims: &cliutil.VerifiedClaims{RunID: "run-1", Subject: authfixture.OwnerSubject, ExpiresAt: time.Now().Add(time.Hour).Unix()}}, nil
	})
	for _, test := range []struct {
		name  string
		wrap  []func(http.Handler) http.Handler
		token string
		code  connect.Code
	}{
		{"unverified agent token", nil, "run-token", connect.CodeUnauthenticated},
		{"verified agent run", []func(http.Handler) http.Handler{provenance.Middleware(verifier)}, "run-token", connect.CodePermissionDenied},
		{"forged agent token", []func(http.Handler) http.Handler{provenance.Middleware(verifier)}, "forged", connect.CodeUnauthenticated},
	} {
		t.Run(test.name, func(t *testing.T) {
			ledger := ledgertest.New(t, test.wrap...)
			for _, owner := range []map[string]string{
				{"Authorization": "Bearer " + ledger.Identity.Owner()},
				{"Cf-Access-Jwt-Assertion": ledger.Identity.OwnerAccess()},
			} {
				owner[cliutil.HeaderAgentIdentityToken] = test.token
				require.Equal(t, test.code, post(t, ledger.URL+"/vrooli.money_ledger.v1.ingest.IngestService/IngestEvent", owner))
			}
			require.Zero(t, ledger.Count(t, "postings"))
		})
	}
}

// post sends an empty Connect request and returns its code (0 on success).
func post(t *testing.T, url string, headers map[string]string) connect.Code {
	t.Helper()
	request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewBufferString("{}"))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Connect-Protocol-Version", "1")
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		return 0
	}
	var envelope struct {
		Code string `json:"code"`
	}
	payload, _ := io.ReadAll(response.Body)
	require.NoError(t, json.Unmarshal(payload, &envelope), "status %d: %s", response.StatusCode, payload)
	var code connect.Code
	require.NoError(t, code.UnmarshalText([]byte(envelope.Code)), "status %d: %s", response.StatusCode, payload)
	return code
}
