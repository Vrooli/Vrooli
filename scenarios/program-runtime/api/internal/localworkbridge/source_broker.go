package localworkbridge

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"slices"
	"time"
)

const SourceBrokerSchema = "flywheel.local-work-source-broker.v1"
const sourceBrokerResponseLimit = 1 << 20

// SourceBrokerFence belongs to a trusted lifecycle owner, not to the worker or
// its JSON. Check must verify the current joined launch generation, revocation
// and the actual private-channel/no-network confinement before returning nil.
// No production implementation is provided. Metadata or a self-attestation from
// the worker cannot implement this boundary. Methods must honor cancellation.
type SourceBrokerScope struct {
	Subject, Owner, SourceInstance, SourceEpoch, LaunchGeneration string
	RequestKey, AdmissionDeadline                                 string
	EventsScopeSHA256, CollectorScopeSHA256                       string
}

type SourceBrokerFence interface {
	Check(context.Context, SourceBrokerScope) error
}

// SourceBrokerCredentials returns a fresh caller-owned buffer for one source
// purpose. The broker clears that buffer after verification and never returns
// credentials to the worker or the owner adapter. The authority owns custody.
// Implementations must cooperate with cancellation; no ambient fallback exists.
type SourceBrokerCredentials interface {
	Credential(context.Context, string) ([]byte, error)
}

// SourceBrokerOwners is a finite trusted owner adapter, not an HTTP proxy. It
// MUST use canonical owner records and SourcePermit.Recheck at the final effect
// transaction/read boundary, serialized with session invalidation as well as
// source authority changes. Rechecking and then committing without that shared
// serialization leaves a race; a returned error cannot undo an effect. Collector
// ownership must already be stored in the
// same durable admission row; inventing it here is forbidden. An adapter around
// a generic anonymous API without these guarantees does not meet this contract.
// Returned buffers/values transfer ownership to the broker and may not be mutated
// afterward. All calls must honor ctx; the broker does not abandon owner calls.
type SourceBrokerOwners interface {
	ReadEvent(context.Context, *SourcePermit) (json.RawMessage, error)
	AdmitCollector(context.Context, *SourcePermit) (SourceBrokerProgram, error)
	ObserveCollector(context.Context, *SourcePermit) (SourceBrokerProgram, error)
	WaitCollector(context.Context, *SourcePermit) (SourceBrokerProgram, error)
}

// SourceBrokerProgram deliberately excludes raw source, stderr, credentials,
// caller attribution and extra owner fields. Identity must be loaded from the
// owner's durable row, not copied from the permit or caller-supplied fields.
type SourceBrokerProgram struct {
	Found          bool
	Identity       SourceProgramIdentity
	Status, Stdout string
}

// SourceBrokerSelection comes from the trusted authenticated receipt owner,
// never worker input. That owner must persist the key and fixed deadline with
// the exact selected event set across restart. This is not a second admission
// ledger. New selections need fresh owner authorization and a new broker binding.
type SourceBrokerSelection struct {
	RequestKey, AdmissionDeadline string
	// Only NativeSelection can populate exact authenticated identity metadata.
	// A static trusted config with no identities is not native provenance proof.
	nativeEvents  []SourceEventIdentity
	nativeBinding *nativeSelectionBinding
}

type nativeSelectionBinding struct{ requestKey, admissionDeadline, eventsDigest, collectorDigest string }

type SourceBrokerConfig struct {
	Enabled           bool
	LaunchGeneration  string
	Selection         SourceBrokerSelection
	Events, Collector *SourceGate
	Credentials       SourceBrokerCredentials
	Fence             SourceBrokerFence
	Owners            SourceBrokerOwners
}

// SourceBroker is an unregistered, inert-by-default operation core. It opens no
// listener, socket, process or storage, changes no security settings and creates
// no credentials. Enabled construction is NOT evidence of kernel confinement.
// The existing ProcessRunner/credential pipe is NOT connected to this core.
type SourceBroker struct {
	enabled           bool
	events, collector *SourceGate
	credentials       SourceBrokerCredentials
	fence             SourceBrokerFence
	owners            SourceBrokerOwners
	busy              chan struct{}
	scope             SourceBrokerScope
	nativeEvents      map[string]SourceEventIdentity
}

func NewSourceBroker(c SourceBrokerConfig) (*SourceBroker, error) {
	b := &SourceBroker{enabled: c.Enabled}
	if !c.Enabled {
		return b, nil
	}
	if !sourceKey.MatchString(c.Selection.RequestKey) || !sourceAtom.MatchString(c.LaunchGeneration) || c.Events == nil || c.Collector == nil || !c.Events.enabled || !c.Collector.enabled || c.Events.policy.Purpose != SourceEvents || c.Collector.policy.Purpose != SourceCollector || c.Credentials == nil || c.Fence == nil || c.Owners == nil {
		return nil, ErrSourceAuthority
	}
	if _, err := time.Parse(time.RFC3339Nano, c.Selection.AdmissionDeadline); err != nil {
		return nil, ErrSourceAuthority
	}
	e, p := c.Events.policy, c.Collector.policy
	if e.Subject != p.Subject || e.Owner != p.Owner || e.SourceInstance != p.SourceInstance || e.SourceEpoch != p.SourceEpoch || e.BindingID == p.BindingID {
		return nil, ErrSourceAuthority
	}
	b.events, b.collector, b.credentials, b.fence, b.owners = c.Events, c.Collector, c.Credentials, c.Fence, c.Owners
	b.scope = SourceBrokerScope{Subject: e.Subject, Owner: e.Owner, SourceInstance: e.SourceInstance, SourceEpoch: e.SourceEpoch, LaunchGeneration: c.LaunchGeneration, RequestKey: c.Selection.RequestKey, AdmissionDeadline: c.Selection.AdmissionDeadline, EventsScopeSHA256: c.Events.digest, CollectorScopeSHA256: c.Collector.digest}
	if c.Selection.nativeBinding != nil || len(c.Selection.nativeEvents) != 0 {
		binding := c.Selection.nativeBinding
		if binding == nil || binding.requestKey != c.Selection.RequestKey || binding.admissionDeadline != c.Selection.AdmissionDeadline || binding.eventsDigest != c.Events.digest || binding.collectorDigest != c.Collector.digest {
			return nil, ErrSourceAuthority
		}
		if len(c.Selection.nativeEvents) != len(e.EventIDs) {
			return nil, ErrSourceAuthority
		}
		b.nativeEvents = make(map[string]SourceEventIdentity, len(e.EventIDs))
		for _, identity := range c.Selection.nativeEvents {
			if !slices.Contains(e.EventIDs, identity.ID) || identity.Scenario != "agent-manager" || !slices.Contains(nativeEventTypes, identity.Type) {
				return nil, ErrSourceAuthority
			}
			if _, duplicate := b.nativeEvents[identity.ID]; duplicate {
				return nil, ErrSourceAuthority
			}
			b.nativeEvents[identity.ID] = identity
		}
	}
	b.busy = make(chan struct{}, 1)
	return b, nil
}

// Execute accepts one bounded finite request. It never forwards request headers,
// URLs, arbitrary inputs or bytes as owner authority. No goroutine or retry is
// created. A lost admission response must be observed with the same durable key,
// never implicitly re-admitted with a fresh key. Callers own joining this method.
func (b *SourceBroker) Execute(ctx context.Context, raw []byte) ([]byte, error) {
	return b.execute(ctx, raw, nil)
}

// execute retains the admitted permit until a trusted private transport has
// completed release. The callback is never supplied by worker JSON.
func (b *SourceBroker) execute(ctx context.Context, raw []byte, release func(context.Context, *SourcePermit, []byte) error) ([]byte, error) {
	if b == nil || !b.enabled {
		return nil, ErrDisabled
	}
	if ctx == nil || ctx.Err() != nil || len(raw) > 4096 {
		return nil, ErrSourceAuthority
	}
	select {
	case b.busy <- struct{}{}:
		defer func() { <-b.busy }()
	default:
		return nil, ErrSourceAuthority
	}
	gate, r, operation, err := b.request(raw)
	if err != nil {
		return nil, ErrSourceAuthority
	}
	// Bound each cooperative operation, including authority lookups and waits.
	ctx, cancel := context.WithTimeout(ctx, 100*time.Second)
	defer cancel()
	if b.fence.Check(ctx, b.scope) != nil || ctx.Err() != nil {
		return nil, ErrSourceAuthority
	}
	purpose := "events_reader"
	if gate == b.collector {
		purpose = "runtime_collector"
	}
	credential, err := b.credentials.Credential(ctx, purpose)
	if err != nil || ctx.Err() != nil || len(credential) == 0 || len(credential) > 8192 {
		clear(credential)
		return nil, ErrSourceAuthority
	}
	r.Header.Set("Authorization", string(credential))
	permit, err := gate.Admit(ctx, r)
	clear(credential)
	r.Header.Del("Authorization")
	if err != nil {
		return nil, ErrSourceAuthority
	}
	// Carry the launch fence into the exact permit the owner must recheck at
	// its final serialized effect boundary, not just the broker pre/post path.
	permit.finalFence = func(ctx context.Context) error { return b.fence.Check(ctx, b.scope) }
	if gate == b.events && len(b.nativeEvents) != 0 {
		identity, ok := b.nativeEvents[permit.call.EventID]
		if !ok {
			return nil, ErrSourceAuthority
		}
		permit.selectedEvent = &identity
	}
	if gate == b.collector {
		permit.call.RequestKey = b.scope.RequestKey
	}
	if permit.Recheck(ctx) != nil {
		return nil, ErrSourceAuthority
	}
	// Clip owner work to the token's remaining real-time window, and to an
	// admission's fixed deadline. Fresh authority is checked again on release.
	until := permit.principal.ExpiresAt
	if permit.call.AdmissionDeadline != "" {
		deadline, _ := time.Parse(time.RFC3339Nano, permit.call.AdmissionDeadline)
		if deadline.Before(until) {
			until = deadline
		}
	}
	budget := until.Sub(gate.clock())
	if permit.call.TimeoutMillis > 0 && time.Duration(permit.call.TimeoutMillis)*time.Millisecond < budget {
		budget = time.Duration(permit.call.TimeoutMillis) * time.Millisecond
	}
	ownerCtx, stop := context.WithTimeout(ctx, budget)
	defer stop()
	if ownerCtx.Err() != nil {
		return nil, ErrSourceAuthority
	}
	var result []byte
	if operation == "read_event" {
		envelope, e := b.owners.ReadEvent(ownerCtx, permit)
		if e != nil {
			return nil, ErrSourceAuthority
		}
		result, err = brokerEventResult(ownerCtx, permit, envelope)
	} else {
		var program SourceBrokerProgram
		switch operation {
		case "admit_collector":
			program, err = b.owners.AdmitCollector(ownerCtx, permit)
		case "observe_collector":
			program, err = b.owners.ObserveCollector(ownerCtx, permit)
		case "wait_collector":
			program, err = b.owners.WaitCollector(ownerCtx, permit)
		}
		if err == nil {
			result, err = brokerProgramResult(ownerCtx, permit, operation, program)
		}
	}
	if err != nil || ownerCtx.Err() != nil || permit.Recheck(ownerCtx) != nil || len(result) > sourceBrokerResponseLimit {
		return nil, ErrSourceAuthority
	}
	if release != nil {
		defer clear(result)
		if release(ownerCtx, permit, result) != nil {
			return nil, ErrSourceAuthority
		}
		return nil, nil
	}
	return result, nil
}

func (b *SourceBroker) request(raw []byte) (*SourceGate, *http.Request, string, error) {
	fail := func() (*SourceGate, *http.Request, string, error) { return nil, nil, "", ErrSourceAuthority }
	v, err := sourceObject(raw)
	if err != nil || v["schema"] != SourceBrokerSchema {
		return fail()
	}
	op, _ := v["operation"].(string)
	str := func(key string) string { s, _ := v[key].(string); return s }
	gate := b.collector
	method, path, query := "POST", "", ""
	var body map[string]any
	switch op {
	case "read_event":
		if !sourceFields(v, "schema", "operation", "event_id") || !sourceAtom.MatchString(str("event_id")) {
			return fail()
		}
		gate = b.events
		method, path = "GET", "/api/v1/events"
		query = url.Values{"event_id": {str("event_id")}, "limit": {"2"}}.Encode()
	case "observe_collector":
		if !sourceFields(v, "schema", "operation") {
			return fail()
		}
		path = sourceLibrary + "GetDeclaredExecution"
		body = map[string]any{"name": SourceCollectorName, "idempotencyKey": b.scope.RequestKey}
	case "admit_collector":
		if !sourceFields(v, "schema", "operation") {
			return fail()
		}
		path = sourceLibrary + "RunDeclaredProgram"
		body = map[string]any{"name": SourceCollectorName, "expectedDigest": gate.policy.CollectorDigest, "inputs": json.RawMessage(bytes.Clone(gate.policy.CollectorInputs)), "provenance": "PROVENANCE_AGENT", "async": true, "idempotencyKey": b.scope.RequestKey, "admissionDeadline": b.scope.AdmissionDeadline}
	case "wait_collector":
		if !sourceFields(v, "schema", "operation", "program_id", "timeout_ms") {
			return fail()
		}
		path = sourcePrograms + "WaitForProgram"
		body = map[string]any{"id": str("program_id"), "timeoutMillis": v["timeout_ms"]}
	default:
		return fail()
	}
	r := &http.Request{Method: method, URL: &url.URL{Path: path, RawQuery: query}, Header: make(http.Header)}
	if body != nil {
		encoded, e := json.Marshal(body)
		if e != nil {
			return fail()
		}
		r.Body = io.NopCloser(bytes.NewReader(encoded))
		r.ContentLength = int64(len(encoded))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Connect-Protocol-Version", "1")
		r.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(encoded)), nil }
	}
	// Parse before touching a credential. Reset our own bounded in-memory body.
	if _, err = gate.parse(r); err != nil {
		return fail()
	}
	if r.GetBody != nil {
		r.Body, _ = r.GetBody()
	}
	return gate, r, op, nil
}

func brokerEventResult(ctx context.Context, p *SourcePermit, raw []byte) ([]byte, error) {
	if len(raw) == 0 {
		if err := p.Recheck(ctx); err != nil {
			return nil, err
		}
		return []byte("[]"), nil
	}
	if len(raw) > sourceBrokerResponseLimit {
		return nil, ErrSourceAuthority
	}
	raw = bytes.Clone(raw)
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	value, err := sourceValue(d, 0)
	if err != nil {
		return nil, ErrSourceAuthority
	}
	if _, err = d.Token(); err != io.EOF {
		return nil, ErrSourceAuthority
	}
	event, ok := value.(map[string]any)
	if !ok {
		return nil, ErrSourceAuthority
	}
	id, _ := event["eventId"].(string)
	typ, _ := event["eventType"].(string)
	source, ok := event["source"].(map[string]any)
	if !ok {
		return nil, ErrSourceAuthority
	}
	scenario, _ := source["scenario"].(string)
	if err = p.CheckEvent(ctx, SourceEventIdentity{ID: id, Type: typ, Scenario: scenario}); err != nil {
		return nil, err
	}
	// Copy normalized bytes, never return an owner-owned alias.
	return json.Marshal([]any{event})
}

func brokerProgramResult(ctx context.Context, p *SourcePermit, operation string, r SourceBrokerProgram) ([]byte, error) {
	if !r.Found {
		if operation != "observe_collector" || r.Identity != (SourceProgramIdentity{}) || r.Status != "" || r.Stdout != "" {
			return nil, ErrSourceAuthority
		}
		if err := p.Recheck(ctx); err != nil {
			return nil, err
		}
		return []byte(`{"admissionContractVersion":2,"found":false}`), nil
	}
	if err := p.CheckProgram(ctx, r.Identity); err != nil {
		return nil, err
	}
	terminal := slices.Contains([]string{"PROGRAM_STATUS_SUCCEEDED", "PROGRAM_STATUS_FAILED", "PROGRAM_STATUS_CANCELLED"}, r.Status)
	if !terminal && !slices.Contains([]string{"PROGRAM_STATUS_ACCEPTED", "PROGRAM_STATUS_RUNNING"}, r.Status) {
		return nil, ErrSourceAuthority
	}
	if len(r.Stdout) > 65536 {
		return nil, ErrSourceAuthority
	}
	// Only successful snapshot output may be released, never partial output
	// from a still-running or failed collector.
	if r.Status != "PROGRAM_STATUS_SUCCEEDED" && r.Stdout != "" {
		return nil, ErrSourceAuthority
	}
	program := map[string]any{"id": r.Identity.ID, "programName": r.Identity.Name, "programDigest": r.Identity.Digest, "status": r.Status}
	if r.Status == "PROGRAM_STATUS_SUCCEEDED" {
		program["stdout"] = r.Stdout
	}
	result := map[string]any{"program": program}
	if operation == "observe_collector" {
		result["admissionContractVersion"] = 2
		result["found"] = true
	}
	if operation == "wait_collector" {
		result["terminal"] = terminal
	}
	return json.Marshal(result)
}
