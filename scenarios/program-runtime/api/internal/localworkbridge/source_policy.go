package localworkbridge

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
)

// This is an inert source-owner policy boundary, not a credential issuer or a
// live route. Production must install it at EVERY ingress accepting its audience
// and bind its decisions to the owner's final transaction. Never put a broad
// credential behind this policy and describe the credential itself as scoped.
const (
	SourceEvents        = "events"
	SourceCollector     = "collector"
	SourceCollectorName = "program-runtime.local-work-snapshot"
	sourceLibrary       = "/vrooli.program_runtime.v1.library.LibraryService/"
	sourcePrograms      = "/vrooli.program_runtime.v1.programs.ProgramService/"
)

var ErrSourceAuthority = errors.New("local work source authority unavailable")
var sourceAtom = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
var sourceDigest = regexp.MustCompile(`^[a-f0-9]{64}$`)
var sourceKey = regexp.MustCompile(`^local-work-[a-f0-9]{64}$`)

// SourceBinding is read from a trusted owner authority on EVERY check. It is
// not supplied by request fields, token claims, a scope label, or a local cache.
// Version and ScopeSHA256 bind that record to the reviewed immutable policy.
// Revocation/unavailability must fail closed. The authority owns durable grant
// storage; this package does not create a parallel store or approvals.
type SourceBinding struct {
	ID, Version, Subject, Owner, Audience, ScopeSHA256 string
	NotBefore, ExpiresAt                               time.Time
	Revoked                                            bool
	Grant                                              *identity.OperationGrant
}
type SourceBindingReader interface {
	ReadSourceBinding(context.Context, string) (SourceBinding, error)
}

// SourcePolicy is approved non-secret configuration. Each instance admits one
// dedicated service identity and one source purpose. EventIDs is a finite,
// trusted selection, never an event-ID set supplied by the bearer caller.
// Maintaining later selections requires an owned authenticated receipt ledger.
type SourcePolicy struct {
	Purpose, BindingID, BindingVersion, Subject, Owner string
	SourceInstance, SourceEpoch                        string
	EventIDs                                           []string
	CollectorDigest                                    string
	CollectorInputs                                    json.RawMessage
}
type SourceGateConfig struct {
	Enabled  bool
	Policy   SourcePolicy
	Verifier *authn.JWTVerifier
	Bindings SourceBindingReader
}
type SourceGate struct {
	enabled  bool
	policy   SourcePolicy
	digest   string
	verifier authn.Provider
	bindings SourceBindingReader
	clock    func() time.Time
}

// SourceCall is the finite parsed request. It is a value snapshot; no HTTP
// header, unknown JSON field, caller attribution, grants, URL or raw source can
// be forwarded as owner authority. Use Call() for a fresh defensive copy.
type SourceCall struct {
	Operation, Target, EventID, ProgramID, RequestKey, AdmissionDeadline string
	Inputs                                                               json.RawMessage
	TimeoutMillis                                                        int64
}
type SourcePermit struct {
	gate      *SourceGate
	principal identity.Principal
	call      SourceCall
	// Installed only by the trusted broker, never supplied in an HTTP request.
	// The owner serializes final rechecks/commit with session invalidation.
	finalFence    func(context.Context) error
	selectedEvent *SourceEventIdentity
}

func (p *SourcePermit) Call() SourceCall {
	c := p.call
	c.Inputs = bytes.Clone(c.Inputs)
	return c
}
func (p *SourcePermit) Subject() string     { return p.gate.policy.Subject }
func (p *SourcePermit) Owner() string       { return p.gate.policy.Owner }
func (p *SourcePermit) ScopeSHA256() string { return p.gate.digest }

// CollectorSpec returns the exact reviewed declaration and inputs from the
// verified policy, not request metadata. It does not resolve or admit source.
func (p *SourcePermit) CollectorSpec() (name, digest string, inputs json.RawMessage, err error) {
	if p == nil || p.gate == nil || p.gate.policy.Purpose != SourceCollector ||
		!slices.Contains([]string{"local-work.collector.admit", "local-work.collector.observe", "local-work.collector.wait"}, p.call.Operation) {
		return "", "", nil, ErrSourceAuthority
	}
	return SourceCollectorName, p.gate.policy.CollectorDigest, bytes.Clone(p.gate.policy.CollectorInputs), nil
}

// Recheck must be invoked by the owner at the final read/commit boundary, with
// current owner policy and under the same admission serialization as the effect.
// HTTP admission alone cannot close the authorization-to-commit race. A blocked
// owner call must be context-bounded; expiry never grants an extra retry window.
func (p *SourcePermit) Recheck(ctx context.Context) error {
	if p == nil || p.gate == nil {
		return ErrSourceAuthority
	}
	if p.finalFence != nil && p.finalFence(ctx) != nil {
		return ErrSourceAuthority
	}
	// A fence may block. Read current authority and expiry AFTER it returns.
	if err := p.gate.authorize(ctx, p.principal, p.call); err != nil {
		return err
	}
	if ctx.Err() != nil {
		return ErrSourceAuthority
	}
	return nil
}

func NewSourceGate(cfg SourceGateConfig) (*SourceGate, error) {
	g := &SourceGate{enabled: cfg.Enabled, clock: time.Now}
	if !cfg.Enabled {
		return g, nil
	}
	if cfg.Verifier == nil || cfg.Bindings == nil || cfg.Verifier.Source() != identity.SourceScenarioAuthenticator {
		return nil, ErrSourceAuthority
	}
	p := cfg.Policy
	if !cfg.Verifier.UsesExclusiveAudience("vrooli-local-work:" + p.Purpose) {
		return nil, ErrSourceAuthority
	}
	if p.Purpose != SourceEvents && p.Purpose != SourceCollector {
		return nil, ErrSourceAuthority
	}
	for _, v := range []string{p.BindingID, p.BindingVersion, p.Subject, p.Owner, p.SourceEpoch} {
		if !sourceAtom.MatchString(v) {
			return nil, ErrSourceAuthority
		}
	}
	if p.SourceInstance != "swarminator" || p.SourceEpoch == "uncommissioned" {
		return nil, ErrSourceAuthority
	}
	p.EventIDs = slices.Clone(p.EventIDs)
	if p.Purpose == SourceEvents {
		if len(p.EventIDs) == 0 || len(p.EventIDs) > 64 || p.CollectorDigest != "" || len(p.CollectorInputs) != 0 {
			return nil, ErrSourceAuthority
		}
		slices.Sort(p.EventIDs)
		for i, id := range p.EventIDs {
			if !sourceAtom.MatchString(id) || (i > 0 && id == p.EventIDs[i-1]) {
				return nil, ErrSourceAuthority
			}
		}
	} else {
		if len(p.EventIDs) != 0 || !sourceDigest.MatchString(p.CollectorDigest) {
			return nil, ErrSourceAuthority
		}
		value, err := sourceObject(p.CollectorInputs)
		if err != nil {
			return nil, ErrSourceAuthority
		}
		allowed := []string{"effort_page_token", "research_root", "row_limit", "run_offsets", "run_statuses", "sample_offsets", "team_ids", "team_task_offsets", "usage_window_days"}
		for k := range value {
			if !slices.Contains(allowed, k) {
				return nil, ErrSourceAuthority
			}
		}
		p.CollectorInputs, _ = json.Marshal(value)
	}
	raw, _ := json.Marshal(p)
	g.policy = p
	g.digest = fmt.Sprintf("%x", sha256.Sum256(raw))
	g.verifier = cfg.Verifier
	g.bindings = cfg.Bindings
	return g, nil
}
func (g *SourceGate) ScopeSHA256() string { return g.digest }
func (g *SourceGate) audience() string    { return "vrooli-local-work:" + g.policy.Purpose }
func (g *SourceGate) scope() string       { return "local-work:" + g.policy.Purpose }

// Admit rejects every method/path/shape except the three pinned collector RPCs
// or an exact selected event query. Verification uses the canonical provider;
// grants use OperationGrant.Allows, including its operation-target rule pairing.
// No installed issuer/verifier, live grant store, route or credential is created.
func (g *SourceGate) Admit(ctx context.Context, r *http.Request) (*SourcePermit, error) {
	if g == nil || !g.enabled {
		return nil, ErrDisabled
	}
	if ctx.Err() != nil || r == nil || r.URL == nil {
		return nil, ErrSourceAuthority
	}
	// No alternate auth carriers, proxy override, or test-routing context may
	// alter owner or data partition behind a capability. Only the Bearer token
	// reaches the verifier, on a new request with no caller-controlled context.
	seenHeaders := map[string]bool{}
	headerBytes := 0
	for k, vs := range r.Header {
		lower := strings.ToLower(k)
		if seenHeaders[lower] {
			return nil, ErrSourceAuthority
		}
		seenHeaders[lower] = true
		headerBytes += len(k)
		for _, value := range vs {
			headerBytes += len(value)
			if strings.ContainsAny(value, "\r\n") {
				return nil, ErrSourceAuthority
			}
		}
		if headerBytes > 8192 || len(vs) != 1 || (lower != "authorization" && lower != "content-type" && lower != "connect-protocol-version" && lower != "accept" && lower != "user-agent" && lower != "content-length") {
			return nil, ErrSourceAuthority
		}
	}
	if len(r.Header.Values("Authorization")) != 1 || !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") || strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")) == "" {
		return nil, ErrSourceAuthority
	}
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if len(token) > 8192 || strings.ContainsAny(token, " \t\r\n,") {
		return nil, ErrSourceAuthority
	}
	call, err := g.parse(r)
	if err != nil {
		return nil, ErrSourceAuthority
	}
	verifiedRequest := &http.Request{Header: http.Header{"Authorization": []string{"Bearer " + token}}}
	principal, err := g.verifier.VerifyRequest(ctx, verifiedRequest)
	if err != nil {
		return nil, ErrSourceAuthority
	}
	if err = g.authorize(ctx, principal, call); err != nil {
		return nil, err
	}
	// Copy all slices and discard token-borne grants. No request proof is exported
	// to generic RequireCapability or installed as a context principal.
	principal.Scopes = slices.Clone(principal.Scopes)
	principal.Sources = slices.Clone(principal.Sources)
	principal.Roles = slices.Clone(principal.Roles)
	principal.Grant = nil
	return &SourcePermit{gate: g, principal: principal, call: call}, nil
}
func (g *SourceGate) authorize(ctx context.Context, p identity.Principal, c SourceCall) error {
	if ctx.Err() != nil {
		return ErrSourceAuthority
	}
	now := g.clock()
	if !p.IsVerified() || p.Kind != identity.ActorService || p.Source != identity.SourceScenarioAuthenticator || p.Issuer != "scenario-authenticator" || p.Audience != g.audience() || p.Subject != g.policy.Subject || len(p.Scopes) != 1 || p.Scopes[0] != g.scope() || len(p.Roles) != 0 || p.Grant != nil || p.ExpiresAt.IsZero() || !now.Before(p.ExpiresAt) {
		return ErrSourceAuthority
	}
	b, err := g.bindings.ReadSourceBinding(ctx, g.policy.BindingID)
	now = g.clock()
	if err != nil || ctx.Err() != nil || b.Revoked || b.ID != g.policy.BindingID || b.Version != g.policy.BindingVersion || b.Subject != g.policy.Subject || b.Owner != g.policy.Owner || b.Audience != g.audience() || b.ScopeSHA256 != g.digest || b.NotBefore.IsZero() || now.Before(b.NotBefore) || !now.Before(b.ExpiresAt) || !now.Before(p.ExpiresAt) || b.ExpiresAt.Sub(b.NotBefore) > 7*24*time.Hour || !b.ExpiresAt.After(b.NotBefore) || p.ExpiresAt.After(b.ExpiresAt) || b.Grant == nil || b.Grant.Approver != b.Owner || b.Grant.ExpiresAt.After(b.ExpiresAt) || !b.Grant.Allows(c.Operation, c.Target, now) {
		return ErrSourceAuthority
	}
	if c.AdmissionDeadline != "" {
		deadline, err := time.Parse(time.RFC3339Nano, c.AdmissionDeadline)
		if err != nil || !now.Before(deadline) || deadline.After(now.Add(10*time.Minute)) || deadline.After(b.ExpiresAt) || deadline.After(p.ExpiresAt) {
			return ErrSourceAuthority
		}
	}
	return nil
}
func (g *SourceGate) parse(r *http.Request) (SourceCall, error) {
	bad := func() (SourceCall, error) { return SourceCall{}, ErrSourceAuthority }
	if r.URL.RawPath != "" || r.URL.Fragment != "" || r.URL.User != nil || r.URL.Scheme != "" || r.URL.Host != "" || r.URL.ForceQuery || len(r.TransferEncoding) > 0 || r.ContentLength > 16384 {
		return bad()
	}
	if g.policy.Purpose == SourceEvents {
		if r.Method != "GET" || r.URL.Path != "/api/v1/events" || r.ContentLength > 0 {
			return bad()
		}
		if r.Body != nil {
			data, err := io.ReadAll(io.LimitReader(r.Body, 1))
			if err != nil || len(data) > 0 {
				return bad()
			}
		}
		q, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil || len(q) != 2 || len(q["event_id"]) != 1 || len(q["limit"]) != 1 || q.Get("limit") != "2" || !slices.Contains(g.policy.EventIDs, q.Get("event_id")) {
			return bad()
		}
		id := q.Get("event_id")
		return SourceCall{Operation: "local-work.event.read", Target: "vrooli-events:local-work/" + g.digest + "/event/" + id, EventID: id}, nil
	}
	if r.Method != "POST" || r.URL.RawQuery != "" || r.Body == nil || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Connect-Protocol-Version") != "1" {
		return bad()
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 16385))
	if err != nil || len(raw) > 16384 {
		return bad()
	}
	v, err := sourceObject(raw)
	if err != nil {
		return bad()
	}
	stringField := func(k string) string { s, _ := v[k].(string); return s }
	name, key := stringField("name"), stringField("idempotencyKey")
	base := "program-runtime:local-work/" + g.digest
	switch r.URL.Path {
	case sourceLibrary + "GetDeclaredExecution":
		if !sourceFields(v, "name", "idempotencyKey") || name != SourceCollectorName || !sourceKey.MatchString(key) {
			return bad()
		}
		return SourceCall{Operation: "local-work.collector.observe", Target: base + "/admission/" + key, RequestKey: key}, nil
	case sourceLibrary + "RunDeclaredProgram":
		if !sourceFields(v, "name", "inputs", "expectedDigest", "provenance", "async", "idempotencyKey", "admissionDeadline") || name != SourceCollectorName || stringField("expectedDigest") != g.policy.CollectorDigest || stringField("provenance") != "PROVENANCE_AGENT" || v["async"] != true || !sourceKey.MatchString(key) {
			return bad()
		}
		if _, ok := v["inputs"].(map[string]any); !ok {
			return bad()
		}
		inputs, _ := json.Marshal(v["inputs"])
		if !bytes.Equal(inputs, g.policy.CollectorInputs) || stringField("admissionDeadline") == "" {
			return bad()
		}
		return SourceCall{Operation: "local-work.collector.admit", Target: base + "/admission/" + key, RequestKey: key, AdmissionDeadline: stringField("admissionDeadline"), Inputs: bytes.Clone(inputs)}, nil
	case sourcePrograms + "WaitForProgram":
		if !sourceFields(v, "id", "timeoutMillis") || !sourceAtom.MatchString(stringField("id")) {
			return bad()
		}
		n, ok := v["timeoutMillis"].(json.Number)
		if !ok {
			return bad()
		}
		timeout, err := n.Int64()
		if err != nil || timeout < 1 || timeout > 90000 {
			return bad()
		}
		id := stringField("id")
		return SourceCall{Operation: "local-work.collector.wait", Target: base + "/program/" + id, ProgramID: id, TimeoutMillis: timeout}, nil
	default:
		return bad()
	}
}
func sourceFields(v map[string]any, keys ...string) bool {
	if len(v) != len(keys) {
		return false
	}
	for _, k := range keys {
		if _, ok := v[k]; !ok {
			return false
		}
	}
	return true
}

// Decode with duplicate-key detection at every depth. encoding/json's default
// last-wins behavior would allow a different policy/owner parser to see a
// different request. Unknown fields and proto snake_case aliases are rejected.
func sourceObject(raw []byte) (map[string]any, error) {
	if len(raw) == 0 || len(raw) > 16384 {
		return nil, ErrSourceAuthority
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	v, err := sourceValue(d, 0)
	if err != nil {
		return nil, err
	}
	if _, err = d.Token(); err != io.EOF {
		return nil, ErrSourceAuthority
	}
	obj, ok := v.(map[string]any)
	if !ok {
		return nil, ErrSourceAuthority
	}
	return obj, nil
}
func sourceValue(d *json.Decoder, depth int) (any, error) {
	if depth > 16 {
		return nil, ErrSourceAuthority
	}
	t, err := d.Token()
	if err != nil {
		return nil, ErrSourceAuthority
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return t, nil
	}
	switch delim {
	case '{':
		m := map[string]any{}
		for d.More() {
			k, err := d.Token()
			if err != nil {
				return nil, ErrSourceAuthority
			}
			key, ok := k.(string)
			if !ok {
				return nil, ErrSourceAuthority
			}
			if _, ok = m[key]; ok {
				return nil, ErrSourceAuthority
			}
			v, err := sourceValue(d, depth+1)
			if err != nil {
				return nil, err
			}
			m[key] = v
		}
		end, err := d.Token()
		if err != nil || end != json.Delim('}') {
			return nil, ErrSourceAuthority
		}
		return m, nil
	case '[':
		a := []any{}
		for d.More() {
			v, err := sourceValue(d, depth+1)
			if err != nil {
				return nil, err
			}
			a = append(a, v)
		}
		end, err := d.Token()
		if err != nil || end != json.Delim(']') {
			return nil, ErrSourceAuthority
		}
		return a, nil
	default:
		return nil, ErrSourceAuthority
	}
}

// SourceEventIdentity must be derived by the Events owner from its canonical
// row, not from caller-supplied receipt fields. Validate before releasing bytes.
type SourceEventIdentity struct{ ID, Type, Scenario string }

func (p *SourcePermit) CheckEvent(ctx context.Context, event SourceEventIdentity) error {
	if p != nil && p.selectedEvent != nil && *p.selectedEvent != event {
		return ErrSourceAuthority
	}
	if p == nil || p.gate == nil || p.call.Operation != "local-work.event.read" || event.ID != p.call.EventID || event.Scenario != "agent-manager" || !slices.Contains([]string{"agent_manager.run.notification.v1", "agent_manager.run.decision_requested.v1", "agent_manager.run.decision_withdrawn.v1"}, event.Type) {
		return ErrSourceAuthority
	}
	return p.Recheck(ctx)
}

// SourceProgramIdentity is owner-retained authority metadata. It MUST be
// persisted atomically with the existing admission row and loaded by the owner;
// Program.caller_run_id and other caller-supplied fields cannot substitute.
// A source policy decision does not populate this record or migrate ownership.
type SourceProgramIdentity struct {
	ID, Name, Digest, RequestKey, Subject, Owner, BindingID, BindingVersion, ScopeSHA256 string
}

func (p *SourcePermit) CheckProgram(ctx context.Context, record SourceProgramIdentity) error {
	if p == nil || p.gate == nil || p.gate.policy.Purpose != SourceCollector || record.Name != SourceCollectorName || record.Digest != p.gate.policy.CollectorDigest || record.Subject != p.Subject() || record.Owner != p.Owner() || record.BindingID != p.gate.policy.BindingID || record.BindingVersion != p.gate.policy.BindingVersion || record.ScopeSHA256 != p.gate.digest || !sourceAtom.MatchString(record.ID) || !sourceKey.MatchString(record.RequestKey) {
		return ErrSourceAuthority
	}
	if (p.call.ProgramID != "" && record.ID != p.call.ProgramID) || (p.call.RequestKey != "" && record.RequestKey != p.call.RequestKey) {
		return ErrSourceAuthority
	}
	return p.Recheck(ctx)
}

// AdmissionIdentity derives immutable ownership only for a verified finite
// collector admission. The owner persists it in the same insert as admission;
// this accessor neither creates an execution nor substitutes for final Recheck.
func (p *SourcePermit) AdmissionIdentity(id string) (SourceProgramIdentity, error) {
	if p == nil || p.gate == nil || p.gate.policy.Purpose != SourceCollector || p.call.Operation != "local-work.collector.admit" || !sourceAtom.MatchString(id) || !sourceKey.MatchString(p.call.RequestKey) {
		return SourceProgramIdentity{}, ErrSourceAuthority
	}
	return SourceProgramIdentity{ID: id, Name: SourceCollectorName, Digest: p.gate.policy.CollectorDigest, RequestKey: p.call.RequestKey, Subject: p.Subject(), Owner: p.Owner(), BindingID: p.gate.policy.BindingID, BindingVersion: p.gate.policy.BindingVersion, ScopeSHA256: p.gate.digest}, nil
}
