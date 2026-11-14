package localworkbridge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
)

type sourceVerifierStub struct {
	p     identity.Principal
	calls int
	err   error
}

func (s *sourceVerifierStub) Source() identity.AuthSource {
	return identity.SourceScenarioAuthenticator
}
func (s *sourceVerifierStub) VerifyRequest(_ context.Context, r *http.Request) (identity.Principal, error) {
	s.calls++
	if len(r.Header) != 1 || r.Header.Get("Authorization") != "Bearer fixture-not-a-credential" {
		return identity.Principal{}, errors.New("unexpected carrier")
	}
	return s.p, s.err
}

type sourceBindingStub struct {
	b      SourceBinding
	calls  int
	err    error
	onRead func()
}

func (s *sourceBindingStub) ReadSourceBinding(_ context.Context, id string) (SourceBinding, error) {
	s.calls++
	if s.onRead != nil {
		s.onRead()
	}
	if id != s.b.ID {
		return SourceBinding{}, ErrSourceAuthority
	}
	return s.b, s.err
}
func sourceFixture(t *testing.T, purpose string) (*SourceGate, *sourceVerifierStub, *sourceBindingStub) {
	t.Helper()
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	p := SourcePolicy{Purpose: purpose, BindingID: "binding-1", BindingVersion: "v1", Subject: "bridge-service", Owner: "owner-1", SourceInstance: "swarminator", SourceEpoch: "epoch-1"}
	if purpose == SourceEvents {
		p.EventIDs = []string{"event-1", "event-2"}
	} else {
		p.CollectorDigest = strings.Repeat("a", 64)
		p.CollectorInputs = json.RawMessage(`{"row_limit":4,"team_ids":[]}`)
	}
	v := &sourceVerifierStub{p: identity.Principal{Verified: true, Kind: identity.ActorService, Subject: p.Subject, Source: identity.SourceScenarioAuthenticator, Issuer: "scenario-authenticator", Audience: "vrooli-local-work:" + purpose, Scopes: []string{"local-work:" + purpose}, ExpiresAt: now.Add(time.Hour)}}
	b := &sourceBindingStub{}
	g, err := NewSourceGate(SourceGateConfig{Enabled: true, Policy: p, Verifier: sourceCanonicalVerifier(p.Purpose, now), Bindings: b})
	if err != nil {
		t.Fatal(err)
	}
	g.clock = func() time.Time { return now }
	g.verifier = v
	domain := "program-runtime"
	ops := []string{"local-work.collector.admit", "local-work.collector.observe", "local-work.collector.wait"}
	if purpose == SourceEvents {
		domain = "vrooli-events"
		ops = []string{"local-work.event.read"}
	}
	b.b = SourceBinding{ID: p.BindingID, Version: p.BindingVersion, Subject: p.Subject, Owner: p.Owner, Audience: v.p.Audience, ScopeSHA256: g.ScopeSHA256(), NotBefore: now.Add(-time.Hour), ExpiresAt: now.Add(2 * time.Hour), Grant: &identity.OperationGrant{ID: "grant-1", Approver: p.Owner, ExpiresAt: now.Add(2 * time.Hour), Operations: ops, Targets: []string{domain + ":local-work/" + g.ScopeSHA256() + "/*"}}}
	return g, v, b
}
func sourceRequest(method, path, body string) *http.Request {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer fixture-not-a-credential")
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Connect-Protocol-Version", "1")
	return r
}
func sourceRunBody() string {
	return `{"name":"program-runtime.local-work-snapshot","inputs":{"row_limit":4,"team_ids":[]},"expectedDigest":"` + strings.Repeat("a", 64) + `","provenance":"PROVENANCE_AGENT","async":true,"idempotencyKey":"local-work-` + strings.Repeat("b", 64) + `","admissionDeadline":"2026-11-07T12:05:00Z"}`
}
func sourceObserveBody() string {
	return `{"name":"program-runtime.local-work-snapshot","idempotencyKey":"local-work-` + strings.Repeat("b", 64) + `"}`
}
func TestSourceGateHappyPaths(t *testing.T) {
	for _, tt := range []struct{ name, purpose, method, path, body string }{
		{"event", SourceEvents, "GET", "/api/v1/events?event_id=event-1&limit=2", ""},
		{"admit", SourceCollector, "POST", sourceLibrary + "RunDeclaredProgram", sourceRunBody()},
		{"observe", SourceCollector, "POST", sourceLibrary + "GetDeclaredExecution", sourceObserveBody()},
		{"wait", SourceCollector, "POST", sourcePrograms + "WaitForProgram", `{"id":"prog_123","timeoutMillis":90000}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g, v, b := sourceFixture(t, tt.purpose)
			p, err := g.Admit(context.Background(), sourceRequest(tt.method, tt.path, tt.body))
			if err != nil {
				t.Fatal(err)
			}
			if p.Call().Operation == "" || p.Owner() != "owner-1" || p.Subject() != "bridge-service" || v.calls != 1 || b.calls != 1 {
				t.Fatal("missing verified binding")
			}
			if err = p.Recheck(context.Background()); err != nil || b.calls != 2 {
				t.Fatalf("live recheck: %v", err)
			}
		})
	}
}

type sourcePanicReader struct{}

func (sourcePanicReader) Read([]byte) (int, error) { panic("disabled body read") }
func TestSourceGateDisabledNoIO(t *testing.T) {
	g, err := NewSourceGate(SourceGateConfig{})
	if err != nil {
		t.Fatal(err)
	}
	r := sourceRequest("GET", "/api/v1/events", "")
	r.Body = io.NopCloser(sourcePanicReader{})
	if _, err = g.Admit(context.Background(), r); !errors.Is(err, ErrDisabled) {
		t.Fatal(err)
	}
}
func TestSourceGateRejectsEventRequests(t *testing.T) {
	for name, modify := range map[string]func(*http.Request){
		"post": func(r *http.Request) { r.Method = "POST" }, "delete": func(r *http.Request) { r.Method = "DELETE" }, "head": func(r *http.Request) { r.Method = "HEAD" }, "list": func(r *http.Request) { r.URL.RawQuery = "limit=2" }, "other id": func(r *http.Request) { r.URL.RawQuery = "event_id=event-3&limit=2" }, "extra query": func(r *http.Request) { r.URL.RawQuery += "&type=secret" }, "duplicate id": func(r *http.Request) { r.URL.RawQuery += "&event_id=event-2" }, "duplicate limit": func(r *http.Request) { r.URL.RawQuery += "&limit=1" }, "range": func(r *http.Request) { r.URL.RawQuery += "&since=0" }, "changed limit": func(r *http.Request) { r.URL.RawQuery = "event_id=event-1&limit=100" }, "bad query": func(r *http.Request) { r.URL.RawQuery += "&bad=%zz" }, "subscription": func(r *http.Request) { r.URL.Path = "/api/v1/events/subscribe" }, "raw path": func(r *http.Request) { r.URL.RawPath = "/api/v1/%65vents" }, "absolute URL": func(r *http.Request) { r.URL.Host = "other"; r.URL.Scheme = "http" }, "body": func(r *http.Request) { r.Body = io.NopCloser(strings.NewReader("x")) }, "chunked": func(r *http.Request) { r.TransferEncoding = []string{"chunked"} },
	} {
		t.Run(name, func(t *testing.T) {
			g, v, b := sourceFixture(t, SourceEvents)
			r := sourceRequest("GET", "/api/v1/events?event_id=event-1&limit=2", "")
			modify(r)
			if _, err := g.Admit(context.Background(), r); err == nil {
				t.Fatal("admitted invalid query")
			}
			if v.calls != 0 || b.calls != 0 {
				t.Fatal("invalid shape reached authority")
			}
		})
	}
}
func TestSourceGateRejectsCollectorRequests(t *testing.T) {
	changes := map[string]func(map[string]any){"name": func(v map[string]any) { v["name"] = "program-runtime.other" }, "digest": func(v map[string]any) { v["expectedDigest"] = strings.Repeat("c", 64) }, "inputs": func(v map[string]any) { v["inputs"] = map[string]any{"row_limit": 5, "team_ids": []any{}} }, "extra input": func(v map[string]any) { v["inputs"].(map[string]any)["research_root"] = "/home" }, "null inputs": func(v map[string]any) { v["inputs"] = nil }, "async": func(v map[string]any) { v["async"] = false }, "operator": func(v map[string]any) { v["provenance"] = "PROVENANCE_OPERATOR" }, "caller": func(v map[string]any) { v["caller"] = map[string]any{"runId": "victim"} }, "grants": func(v map[string]any) { v["grants"] = []any{"binding:other"} }, "source": func(v map[string]any) { v["source"] = "evil()" }, "alias": func(v map[string]any) { v["expected_digest"] = v["expectedDigest"]; delete(v, "expectedDigest") }, "key": func(v map[string]any) { v["idempotencyKey"] = "other-key" }, "expired deadline": func(v map[string]any) { v["admissionDeadline"] = "2026-11-07T11:59:00Z" }, "long deadline": func(v map[string]any) { v["admissionDeadline"] = "2026-11-07T12:11:00Z" }, "missing deadline": func(v map[string]any) { delete(v, "admissionDeadline") }}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			g, _, _ := sourceFixture(t, SourceCollector)
			var v map[string]any
			json.Unmarshal([]byte(sourceRunBody()), &v)
			change(v)
			raw, _ := json.Marshal(v)
			if _, err := g.Admit(context.Background(), sourceRequest("POST", sourceLibrary+"RunDeclaredProgram", string(raw))); err == nil {
				t.Fatal("admitted changed intent")
			}
		})
	}
	for name, body := range map[string]string{"duplicate": strings.Replace(sourceRunBody(), `"async":true`, `"async":false,"async":true`, 1), "nested duplicate": strings.Replace(sourceRunBody(), `"row_limit":4`, `"row_limit":20,"row_limit":4`, 1), "trailing": sourceRunBody() + `{}`, "oversize": strings.Repeat(" ", 16385) + sourceRunBody(), "array": "[]", "null": "null"} {
		t.Run(name, func(t *testing.T) {
			g, _, _ := sourceFixture(t, SourceCollector)
			if _, err := g.Admit(context.Background(), sourceRequest("POST", sourceLibrary+"RunDeclaredProgram", body)); err == nil {
				t.Fatal("admitted malformed body")
			}
		})
	}
	for _, path := range []string{sourcePrograms + "SubmitProgram", sourceLibrary + "CloseDeclaredAdmission", sourceLibrary + "ListLibrary", sourcePrograms + "GetProgram", sourcePrograms + "ListPrograms", "/internal/program-runtime/bindings/execute"} {
		t.Run(path, func(t *testing.T) {
			g, _, _ := sourceFixture(t, SourceCollector)
			if _, err := g.Admit(context.Background(), sourceRequest("POST", path, sourceRunBody())); err == nil {
				t.Fatal("unrelated RPC allowed")
			}
		})
	}
}
func TestSourceGateRejectsWaitBounds(t *testing.T) {
	for _, body := range []string{`{"id":"prog_123","timeoutMillis":0}`, `{"id":"prog_123","timeoutMillis":90001}`, `{"id":"prog_123","timeoutMillis":"10"}`, `{"id":"prog_123","timeoutMillis":1.5}`, `{"id":"../victim","timeoutMillis":1}`, `{"id":"prog_123","timeoutMillis":1,"includeMaterialized":true}`} {
		g, _, _ := sourceFixture(t, SourceCollector)
		if _, err := g.Admit(context.Background(), sourceRequest("POST", sourcePrograms+"WaitForProgram", body)); err == nil {
			t.Fatalf("admitted %s", body)
		}
	}
}
func TestSourceGateRejectsIdentityEscalation(t *testing.T) {
	for name, change := range map[string]func(*identity.Principal){"unverified": func(p *identity.Principal) { p.Verified = false }, "human": func(p *identity.Principal) { p.Kind = identity.ActorHuman }, "agent": func(p *identity.Principal) { p.Kind = identity.ActorAgent }, "subject": func(p *identity.Principal) { p.Subject = "victim" }, "provider": func(p *identity.Principal) { p.Source = identity.SourcePersonalLocal }, "issuer": func(p *identity.Principal) { p.Issuer = "other" }, "audience": func(p *identity.Principal) { p.Audience = "scenario-authenticator:program-runtime" }, "scope": func(p *identity.Principal) { p.Scopes = []string{"program-runtime:write"} }, "extra scope": func(p *identity.Principal) { p.Scopes = append(p.Scopes, "program-runtime:read") }, "role": func(p *identity.Principal) { p.Roles = []string{"admin"} }, "token grant": func(p *identity.Principal) { p.Grant = &identity.OperationGrant{ID: "broad"} }, "expiry missing": func(p *identity.Principal) { p.ExpiresAt = time.Time{} }, "expiry elapsed": func(p *identity.Principal) { p.ExpiresAt = time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC) }, "expiry broad": func(p *identity.Principal) { p.ExpiresAt = p.ExpiresAt.Add(7 * 24 * time.Hour) }} {
		t.Run(name, func(t *testing.T) {
			g, v, _ := sourceFixture(t, SourceCollector)
			change(&v.p)
			if _, err := g.Admit(context.Background(), sourceRequest("POST", sourceLibrary+"RunDeclaredProgram", sourceRunBody())); err == nil {
				t.Fatal("escalated identity allowed")
			}
		})
	}
}
func TestSourceGateRejectsAuthorityChanges(t *testing.T) {
	for name, change := range map[string]func(*SourceBinding){"revoked": func(b *SourceBinding) { b.Revoked = true }, "owner": func(b *SourceBinding) { b.Owner = "victim" }, "subject": func(b *SourceBinding) { b.Subject = "victim" }, "version": func(b *SourceBinding) { b.Version = "v2" }, "audience": func(b *SourceBinding) { b.Audience = "other" }, "scope digest": func(b *SourceBinding) { b.ScopeSHA256 = strings.Repeat("0", 64) }, "not yet": func(b *SourceBinding) { b.NotBefore = b.NotBefore.Add(2 * time.Hour) }, "expired": func(b *SourceBinding) { b.ExpiresAt = b.NotBefore }, "long window": func(b *SourceBinding) { b.ExpiresAt = b.NotBefore.Add(8 * 24 * time.Hour) }, "grant missing": func(b *SourceBinding) { b.Grant = nil }, "grant approver": func(b *SourceBinding) { b.Grant.Approver = "other" }, "grant operation": func(b *SourceBinding) { b.Grant.Operations = []string{"unrelated"} }, "grant target": func(b *SourceBinding) { b.Grant.Targets = []string{"program-runtime:other/*"} }, "grant expired": func(b *SourceBinding) { b.Grant.ExpiresAt = b.NotBefore }, "grant beyond binding": func(b *SourceBinding) { b.Grant.ExpiresAt = b.ExpiresAt.Add(time.Second) }} {
		t.Run(name, func(t *testing.T) {
			g, _, b := sourceFixture(t, SourceCollector)
			p, err := g.Admit(context.Background(), sourceRequest("POST", sourceLibrary+"RunDeclaredProgram", sourceRunBody()))
			if err != nil {
				t.Fatal(err)
			}
			change(&b.b)
			if err := p.Recheck(context.Background()); err == nil {
				t.Fatal("authority change ignored")
			}
		})
	}
	t.Run("unavailable", func(t *testing.T) {
		g, _, b := sourceFixture(t, SourceEvents)
		b.err = errors.New("private store detail")
		if _, err := g.Admit(context.Background(), sourceRequest("GET", "/api/v1/events?event_id=event-1&limit=2", "")); err != ErrSourceAuthority {
			t.Fatal("authority failure leaked or allowed")
		}
	})
	t.Run("expiry during lookup", func(t *testing.T) {
		g, _, b := sourceFixture(t, SourceEvents)
		now := g.clock()
		b.onRead = func() { g.clock = func() time.Time { return now.Add(3 * time.Hour) } }
		if _, err := g.Admit(context.Background(), sourceRequest("GET", "/api/v1/events?event_id=event-1&limit=2", "")); err == nil {
			t.Fatal("lookup time not rechecked")
		}
	})
}
func TestSourceGateUsesCanonicalPairedRules(t *testing.T) {
	g, _, b := sourceFixture(t, SourceEvents)
	target := b.b.Grant.Targets[0]
	b.b.Grant.Rules = []identity.OperationRule{{Policy: "read", Operations: []string{"local-work.event.read"}, Targets: []string{"vrooli-events:other/*"}}, {Policy: "other", Operations: []string{"other"}, Targets: []string{target}}}
	r := func() *http.Request { return sourceRequest("GET", "/api/v1/events?event_id=event-1&limit=2", "") }
	if _, err := g.Admit(context.Background(), r()); err == nil {
		t.Fatal("cross-paired operation/target")
	}
	b.b.Grant.Rules = append(b.b.Grant.Rules, identity.OperationRule{Policy: "selected", Operations: []string{"local-work.event.read"}, Targets: []string{target}})
	if _, err := g.Admit(context.Background(), r()); err != nil {
		t.Fatal("valid covered multi-policy grant rejected", err)
	}
}
func TestSourceGateRejectsAlternateCarriers(t *testing.T) {
	for _, header := range []string{"Cookie", "X-Vrooli-Agent-Identity", "X-Vrooli-Test-Mode", "X-Forwarded-For", "X-HTTP-Method-Override", "X-Vrooli-Operation-Grant", "Content-Encoding"} {
		t.Run(header, func(t *testing.T) {
			g, v, _ := sourceFixture(t, SourceEvents)
			r := sourceRequest("GET", "/api/v1/events?event_id=event-1&limit=2", "")
			r.Header.Set(header, "untrusted")
			if _, err := g.Admit(context.Background(), r); err == nil || v.calls != 0 {
				t.Fatal("alternate carrier reached verifier")
			}
		})
	}
	g, _, _ := sourceFixture(t, SourceEvents)
	r := sourceRequest("GET", "/api/v1/events?event_id=event-1&limit=2", "")
	r.Header.Add("Authorization", "Bearer other")
	if _, err := g.Admit(context.Background(), r); err == nil {
		t.Fatal("duplicate authorization allowed")
	}
}
func TestSourceGateResponseOwnership(t *testing.T) {
	t.Run("events", func(t *testing.T) {
		g, _, b := sourceFixture(t, SourceEvents)
		p, err := g.Admit(context.Background(), sourceRequest("GET", "/api/v1/events?event_id=event-1&limit=2", ""))
		if err != nil {
			t.Fatal(err)
		}
		good := SourceEventIdentity{ID: "event-1", Scenario: "agent-manager", Type: "agent_manager.run.notification.v1"}
		if err = p.CheckEvent(context.Background(), good); err != nil {
			t.Fatal(err)
		}
		for _, bad := range []SourceEventIdentity{{ID: "event-2", Scenario: good.Scenario, Type: good.Type}, {ID: good.ID, Scenario: "other", Type: good.Type}, {ID: good.ID, Scenario: good.Scenario, Type: "agent_manager.run.secret.v1"}} {
			if p.CheckEvent(context.Background(), bad) == nil {
				t.Fatal("wrong canonical identity released")
			}
		}
		b.b.Revoked = true
		if p.CheckEvent(context.Background(), good) == nil {
			t.Fatal("revoked after read released")
		}
	})
	t.Run("programs", func(t *testing.T) {
		g, _, b := sourceFixture(t, SourceCollector)
		p, err := g.Admit(context.Background(), sourceRequest("POST", sourcePrograms+"WaitForProgram", `{"id":"prog_123","timeoutMillis":90000}`))
		if err != nil {
			t.Fatal(err)
		}
		good := SourceProgramIdentity{ID: "prog_123", Name: SourceCollectorName, Digest: g.policy.CollectorDigest, RequestKey: "local-work-" + strings.Repeat("b", 64), Subject: p.Subject(), Owner: p.Owner(), BindingID: g.policy.BindingID, BindingVersion: g.policy.BindingVersion, ScopeSHA256: g.digest}
		if err = p.CheckProgram(context.Background(), good); err != nil {
			t.Fatal(err)
		}
		for name, change := range map[string]func(*SourceProgramIdentity){"id": func(r *SourceProgramIdentity) { r.ID = "prog_other" }, "name": func(r *SourceProgramIdentity) { r.Name = "other" }, "digest": func(r *SourceProgramIdentity) { r.Digest = strings.Repeat("c", 64) }, "subject": func(r *SourceProgramIdentity) { r.Subject = "victim" }, "owner": func(r *SourceProgramIdentity) { r.Owner = "victim" }, "binding": func(r *SourceProgramIdentity) { r.BindingID = "other" }, "version": func(r *SourceProgramIdentity) { r.BindingVersion = "old" }, "scope": func(r *SourceProgramIdentity) { r.ScopeSHA256 = "other" }, "unattributed": func(r *SourceProgramIdentity) { *r = SourceProgramIdentity{ID: r.ID} }} {
			t.Run(name, func(t *testing.T) {
				bad := good
				change(&bad)
				if p.CheckProgram(context.Background(), bad) == nil {
					t.Fatal("foreign/unattributed record allowed")
				}
			})
		}
		b.b.Revoked = true
		if p.CheckProgram(context.Background(), good) == nil {
			t.Fatal("revoked during wait released")
		}
	})
}
func TestSourceGateImmutableAndCancelled(t *testing.T) {
	g, _, b := sourceFixture(t, SourceCollector)
	cfg := SourceGateConfig{Enabled: true, Policy: g.policy, Verifier: sourceCanonicalVerifier(g.policy.Purpose, g.clock()), Bindings: b}
	copyGate, err := NewSourceGate(cfg)
	if err != nil {
		t.Fatal(err)
	}
	before := copyGate.ScopeSHA256()
	cfg.Policy.CollectorInputs[0] = '['
	if copyGate.ScopeSHA256() != before || !bytes.HasPrefix(copyGate.policy.CollectorInputs, []byte("{")) {
		t.Fatal("caller mutation changed binding")
	}
	g, _, _ = sourceFixture(t, SourceCollector)
	p, err := g.Admit(context.Background(), sourceRequest("POST", sourceLibrary+"RunDeclaredProgram", sourceRunBody()))
	if err != nil {
		t.Fatal(err)
	}
	call := p.Call()
	call.Inputs[0] = '['
	if p.Call().Inputs[0] != '{' {
		t.Fatal("permit aliases input")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if p.Recheck(ctx) == nil {
		t.Fatal("cancelled permission allowed")
	}
	if _, err = g.Admit(ctx, sourceRequest("POST", sourceLibrary+"RunDeclaredProgram", sourceRunBody())); err == nil {
		t.Fatal("cancelled request admitted")
	}
}
func TestSourceGateRejectsInvalidConfiguration(t *testing.T) {
	for name, change := range map[string]func(*SourceGateConfig){"no verifier": func(c *SourceGateConfig) { c.Verifier = nil }, "no authority": func(c *SourceGateConfig) { c.Bindings = nil }, "uncommissioned": func(c *SourceGateConfig) { c.Policy.SourceEpoch = "uncommissioned" }, "other host": func(c *SourceGateConfig) { c.Policy.SourceInstance = "other" }, "bad digest": func(c *SourceGateConfig) { c.Policy.CollectorDigest = "bad" }, "unknown input": func(c *SourceGateConfig) { c.Policy.CollectorInputs = json.RawMessage(`{"source":"evil"}`) }, "duplicate input": func(c *SourceGateConfig) {
		c.Policy.CollectorInputs = json.RawMessage(`{"row_limit":4,"row_limit":20}`)
	}, "event confused": func(c *SourceGateConfig) { c.Policy.EventIDs = []string{"event-1"} }, "purpose": func(c *SourceGateConfig) { c.Policy.Purpose = "admin" }, "owner empty": func(c *SourceGateConfig) { c.Policy.Owner = "" }} {
		t.Run(name, func(t *testing.T) {
			g, _, b := sourceFixture(t, SourceCollector)
			cfg := SourceGateConfig{Enabled: true, Policy: g.policy, Verifier: sourceCanonicalVerifier(g.policy.Purpose, g.clock()), Bindings: b}
			change(&cfg)
			if _, err := NewSourceGate(cfg); err == nil {
				t.Fatal("invalid config allowed")
			}
		})
	}
}

func TestSourceGateHeaderAndBodyShapeBounds(t *testing.T) {
	for name, change := range map[string]func(*http.Request){
		"case duplicate": func(r *http.Request) { r.Header["authorization"] = []string{"Bearer other"} },
		"large header":   func(r *http.Request) { r.Header.Set("User-Agent", strings.Repeat("a", 8193)) },
		"newline header": func(r *http.Request) { r.Header.Set("Accept", "application/json\r\nX-Test: 1") },
		"empty bearer":   func(r *http.Request) { r.Header.Set("Authorization", "Bearer ") },
		"token spaces":   func(r *http.Request) { r.Header.Set("Authorization", "Bearer first second") },
		"token comma":    func(r *http.Request) { r.Header.Set("Authorization", "Bearer first,second") },
		"missing token":  func(r *http.Request) { r.Header.Del("Authorization") },
		"form body":      func(r *http.Request) { r.Header.Set("Content-Type", "application/x-www-form-urlencoded") },
		"protocol":       func(r *http.Request) { r.Header.Set("Connect-Protocol-Version", "2") },
		"query on rpc":   func(r *http.Request) { r.URL.RawQuery = "extra=1" },
		"wrong method":   func(r *http.Request) { r.Method = "GET" },
	} {
		t.Run(name, func(t *testing.T) {
			g, v, _ := sourceFixture(t, SourceCollector)
			r := sourceRequest("POST", sourceLibrary+"RunDeclaredProgram", sourceRunBody())
			change(r)
			if _, err := g.Admit(context.Background(), r); err == nil || v.calls != 0 {
				t.Fatal("invalid transport shape reached verification")
			}
		})
	}
}
func TestSourceGateRetainedRequestKeyAndExactGrant(t *testing.T) {
	g, _, b := sourceFixture(t, SourceCollector)
	p, err := g.Admit(context.Background(), sourceRequest("POST", sourceLibrary+"GetDeclaredExecution", sourceObserveBody()))
	if err != nil {
		t.Fatal(err)
	}
	r := SourceProgramIdentity{ID: "prog_123", Name: SourceCollectorName, Digest: g.policy.CollectorDigest, RequestKey: p.Call().RequestKey, Subject: p.Subject(), Owner: p.Owner(), BindingID: g.policy.BindingID, BindingVersion: g.policy.BindingVersion, ScopeSHA256: g.digest}
	b.b.Grant.Targets = []string{p.Call().Target}
	if p.CheckProgram(context.Background(), r) != nil {
		t.Fatal("exact target refused")
	}
	r.RequestKey = "local-work-" + strings.Repeat("c", 64)
	if p.CheckProgram(context.Background(), r) == nil {
		t.Fatal("other admitted key released")
	}
}

func sourceCanonicalVerifier(purpose string, now time.Time) *authn.JWTVerifier {
	return authn.NewScenarioAuthenticatorProvider(authn.JWTConfig{Audience: "vrooli-local-work:" + purpose, ExclusiveAudience: true, Now: func() time.Time { return now }, JWKSURL: "https://fixture.invalid/jwks"})
}
func TestSourceGateRequiresExclusiveCanonicalVerifier(t *testing.T) {
	g, _, b := sourceFixture(t, SourceEvents)
	for name, verifier := range map[string]*authn.JWTVerifier{
		"permissive":    authn.NewScenarioAuthenticatorProvider(authn.JWTConfig{Audience: "vrooli-local-work:events"}),
		"migration":     authn.NewScenarioAuthenticatorProvider(authn.JWTConfig{Audience: "vrooli-local-work:events", ExclusiveAudience: true, AcceptedAudiences: []string{"scenario-authenticator:default"}}),
		"other purpose": sourceCanonicalVerifier(SourceCollector, g.clock()),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NewSourceGate(SourceGateConfig{Enabled: true, Policy: g.policy, Bindings: b, Verifier: verifier}); err == nil {
				t.Fatal("nonexclusive canonical verifier allowed")
			}
		})
	}
}
