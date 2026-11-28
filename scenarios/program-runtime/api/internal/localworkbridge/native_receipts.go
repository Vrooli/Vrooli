package localworkbridge

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"slices"
	"time"
)

// NativeReceiptScope is trusted commissioning configuration. It is never read
// from the worker, webhook payload, hostname, or environment.
type NativeReceiptScope struct {
	Owner, Subject, SourceInstance, SourceEpoch string
	AllowedTypes                                []string
}

// NativeReceiptGuard must serialize effect with all changes to the native
// authority (including shutdown and revocation). A check before an unrelated
// transaction is insufficient. No production guard is installed here. The
// callback context must preserve cancellation and be bounded by authority/key
// expiry. Check expiry again at the commit boundary. The callback must run once,
// synchronously; the owner must join it before replacing
// this scope. This interface creates no authority or OS confinement.
type NativeReceiptGuard interface {
	WithCurrent(context.Context, NativeReceiptScope, func(context.Context) error) error
}

// NativeReceiptCredentials supplies only the dedicated native verification key
// for this exact commissioned scope, in a fresh caller-owned buffer that will
// be cleared. It must prove key provenance/rotation; a
// fleet/shared or prior-epoch key cannot authenticate source/epoch ownership.
// The existing webhook has no signed epoch claim, so key isolation is required.
// No production credential adapter is provided or called by main.
type NativeReceiptCredentials interface {
	VerificationKey(context.Context, NativeReceiptScope) ([]byte, error)
}

// NativeCollectorReconciler reads the actual retained terminal program under the
// collector authority/session boundary and holds that boundary through effect.
// It must call effect exactly once, synchronously, with a cancellation/expiry
// bounded context. It must not trust worker results or reacquire a lock already
// held by the native guard. Deployment must establish a consistent lock order.
// This contract is not itself a production joined authority implementation.
type NativeCollectorReconciler interface {
	WithTerminal(context.Context, *SourcePermit, func(context.Context, SourceBrokerProgram) error) error
}

type NativeReceiptConfig struct {
	Enabled             bool
	Scope               NativeReceiptScope
	Credentials         NativeReceiptCredentials
	Guard               NativeReceiptGuard
	Store               *NativeReceiptStore
	CollectorReconciler NativeCollectorReconciler
}

// NativeReceiptOwner authenticates native deliveries outside the untrusted
// worker. Construction performs no I/O. Only a supplied trusted DB is used;
// this code opens no file, route, listener, process or credential backend.
// The DB must be inaccessible to the worker, including its WAL/journal files.
// It must not be the old Python outbox. Main does not register this owner.
type NativeReceiptOwner struct {
	enabled             bool
	scope               NativeReceiptScope
	digest              string
	credentials         NativeReceiptCredentials
	guard               NativeReceiptGuard
	store               *NativeReceiptStore
	clock               func() time.Time
	collectorReconciler NativeCollectorReconciler
}

var ErrNativeReceiptConflict = errors.New("native receipt identity conflict")
var ErrNativeReceiptCapacity = errors.New("native receipt capacity exhausted")

var nativeEventTypes = []string{"agent_manager.run.decision_requested.v1", "agent_manager.run.decision_withdrawn.v1", "agent_manager.run.notification.v1"}

func NewNativeReceiptOwner(c NativeReceiptConfig) (*NativeReceiptOwner, error) {
	o := &NativeReceiptOwner{enabled: c.Enabled, clock: time.Now}
	if !c.Enabled {
		return o, nil
	}
	s := c.Scope
	s.AllowedTypes = slices.Clone(s.AllowedTypes)
	slices.Sort(s.AllowedTypes)
	if c.Store == nil || c.Store.db == nil || c.Credentials == nil || c.Guard == nil || s.SourceInstance != "swarminator" || s.SourceEpoch == "uncommissioned" || len(s.AllowedTypes) == 0 || len(s.AllowedTypes) > 3 {
		return nil, ErrSourceAuthority
	}
	for _, v := range []string{s.Owner, s.Subject, s.SourceEpoch} {
		if !sourceAtom.MatchString(v) {
			return nil, ErrSourceAuthority
		}
	}
	for i, t := range s.AllowedTypes {
		if !slices.Contains(nativeEventTypes, t) || i > 0 && t == s.AllowedTypes[i-1] {
			return nil, ErrSourceAuthority
		}
	}
	raw, _ := json.Marshal(s)
	sum := sha256.Sum256(raw)
	o.scope, o.digest, o.credentials, o.guard, o.store = s, hex.EncodeToString(sum[:]), c.Credentials, c.Guard, c.Store
	o.collectorReconciler = c.CollectorReconciler
	return o, nil
}

func (o *NativeReceiptOwner) scopeCopy() NativeReceiptScope {
	s := o.scope
	s.AllowedTypes = slices.Clone(s.AllowedTypes)
	return s
}

// Accept returns true only after authenticated durable acceptance or an exact
// duplicate. Authenticated out-of-scope deliveries return false without DB I/O.
// A returned error never acknowledges a delivery. No payload bytes are stored.
func (o *NativeReceiptOwner) Accept(ctx context.Context, r Request) (bool, error) {
	if o == nil || !o.enabled {
		return false, ErrDisabled
	}
	if ctx == nil || ctx.Err() != nil || r.Operation != "accept_native" || len(r.Body) == 0 || len(r.Body) > int(MaxBodyBytes) || len(r.Signature) != 71 || !sourceAtom.MatchString(r.EventID) || r.EventID != r.IdempotencyKey {
		return false, ErrSourceAuthority
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	body := bytes.Clone(r.Body)
	accepted := false
	err := o.guard.WithCurrent(ctx, o.scopeCopy(), func(ctx context.Context) error {
		key, err := o.credentials.VerificationKey(ctx, o.scopeCopy())
		defer clear(key)
		if err != nil || len(key) < 32 || len(key) > 8192 || ctx.Err() != nil {
			return ErrSourceAuthority
		}
		mac := hmac.New(sha256.New, key)
		_, _ = mac.Write(body)
		expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(expected), []byte(r.Signature)) {
			return ErrSourceAuthority
		}
		d := json.NewDecoder(bytes.NewReader(body))
		d.UseNumber()
		value, err := sourceValue(d, 0)
		if err != nil {
			return ErrSourceAuthority
		}
		if _, err = d.Token(); err != io.EOF {
			return ErrSourceAuthority
		}
		v, ok := value.(map[string]any)
		if !ok || !(sourceFields(v, "event_id", "event_type", "source_scenario", "delivered_at") || sourceFields(v, "event_id", "event_type", "source_scenario", "delivered_at", "payload")) {
			return ErrSourceAuthority
		}
		id, okID := v["event_id"].(string)
		typ, okType := v["event_type"].(string)
		source, okSource := v["source_scenario"].(string)
		delivered, okTime := v["delivered_at"].(string)
		if !okID || !okType || !okSource || !okTime || id != r.EventID || !sourceAtom.MatchString(typ) || !sourceAtom.MatchString(source) {
			return ErrSourceAuthority
		}
		if _, err = time.Parse(time.RFC3339Nano, delivered); err != nil {
			return ErrSourceAuthority
		}
		if source != "agent-manager" || !slices.Contains(o.scope.AllowedTypes, typ) {
			return nil
		}
		sum := sha256.Sum256(body)
		if ctx.Err() != nil {
			return ErrSourceAuthority
		}
		err = o.store.accept(ctx, o.digest, nativeReceipt{ID: id, Type: typ, Scenario: source, BodySHA256: hex.EncodeToString(sum[:]), ReceivedAt: o.clock().UTC().Format(time.RFC3339Nano)})
		accepted = err == nil
		return err
	})
	if err != nil || ctx.Err() != nil {
		return false, ErrSourceAuthority
	}
	return accepted, nil
}

// NativeSelection is an opaque owner result, never deserialized from worker
// input. It remains active until source-backed collector reconciliation.
// Neither timeout nor restart mints another attempt. Reconciliation retains
// receipt identities permanently and does not attest cloud handler delivery.
type NativeSelection struct {
	scope         NativeReceiptScope
	digest        string
	key, deadline string
	receipts      []nativeReceipt
}

func (s *NativeSelection) EventIDs() []string {
	if s == nil {
		return nil
	}
	ids := make([]string, len(s.receipts))
	for i, r := range s.receipts {
		ids[i] = r.ID
	}
	return ids
}

// BrokerSelection binds both approved gates to this exact authenticated source
// and selected ID set. It does not mint gates, credentials, grants or bindings.
// This pure projection is not durable commissioning proof. Use owner.BindSelection
// before broker use when later source-backed retirement is required.
func (s *NativeSelection) BrokerSelection(events, collector *SourceGate) (SourceBrokerSelection, error) {
	if s == nil || events == nil || collector == nil || !events.enabled || !collector.enabled || events.policy.Purpose != SourceEvents || collector.policy.Purpose != SourceCollector {
		return SourceBrokerSelection{}, ErrSourceAuthority
	}
	for _, g := range []*SourceGate{events, collector} {
		p := g.policy
		if p.Owner != s.scope.Owner || p.Subject != s.scope.Subject || p.SourceInstance != s.scope.SourceInstance || p.SourceEpoch != s.scope.SourceEpoch {
			return SourceBrokerSelection{}, ErrSourceAuthority
		}
	}
	ids := s.EventIDs()
	slices.Sort(ids)
	if !slices.Equal(ids, events.policy.EventIDs) {
		return SourceBrokerSelection{}, ErrSourceAuthority
	}
	identities := make([]SourceEventIdentity, len(s.receipts))
	for i, r := range s.receipts {
		identities[i] = SourceEventIdentity{ID: r.ID, Type: r.Type, Scenario: r.Scenario}
	}
	return SourceBrokerSelection{RequestKey: s.key, AdmissionDeadline: s.deadline, nativeEvents: identities, nativeBinding: &nativeSelectionBinding{requestKey: s.key, admissionDeadline: s.deadline, eventsDigest: events.digest, collectorDigest: collector.digest}}, nil
}

// Select resumes the same committed selection after loss/restart/expiry, or
// atomically selects at most 64 authenticated unselected receipts. Empty is nil.
// It never accepts event IDs, request keys or deadlines from the worker.
func (o *NativeReceiptOwner) Select(ctx context.Context) (*NativeSelection, error) {
	if o == nil || !o.enabled {
		return nil, ErrDisabled
	}
	if ctx == nil || ctx.Err() != nil {
		return nil, ErrSourceAuthority
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	var selected *NativeSelection
	err := o.guard.WithCurrent(ctx, o.scopeCopy(), func(ctx context.Context) error {
		var err error
		selected, err = o.store.selectReceipts(ctx, o.scopeCopy(), o.digest, o.clock().UTC())
		return err
	})
	if err != nil || ctx.Err() != nil {
		return nil, ErrSourceAuthority
	}
	return selected, nil
}

// ReconcileCollector retires only the exact selected collector attempt after a
// trusted owner reads its immutable terminal result. It accepts no worker result.
// Retirement is not cloud delivery or handler reconciliation: all receipts and
// terminal tombstones remain retained. No API here deletes or retries them.
// A still-current observe permit may reconcile after the admission deadline;
// that deadline is never renewed. A lost-response retry checks the same retained
// result without touching a later active selection.
func (o *NativeReceiptOwner) ReconcileCollector(ctx context.Context, selected *NativeSelection, p *SourcePermit) error {
	if o == nil || !o.enabled {
		return ErrDisabled
	}
	if ctx == nil || ctx.Err() != nil || o.collectorReconciler == nil || selected == nil || selected.digest != o.digest || !validNativeSelection(selected) || p == nil || p.gate == nil || p.call.Operation != "local-work.collector.observe" || p.call.RequestKey != selected.key {
		return ErrSourceAuthority
	}
	policy := p.gate.policy
	if policy.Owner != o.scope.Owner || policy.Subject != o.scope.Subject || policy.SourceInstance != o.scope.SourceInstance || policy.SourceEpoch != o.scope.SourceEpoch {
		return ErrSourceAuthority
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	invoked := false
	err := o.guard.WithCurrent(ctx, o.scopeCopy(), func(ctx context.Context) error {
		if p.Recheck(ctx) != nil {
			return ErrSourceAuthority
		}
		return o.collectorReconciler.WithTerminal(ctx, p, func(ctx context.Context, result SourceBrokerProgram) error {
			if invoked {
				return ErrSourceAuthority
			}
			invoked = true
			if !result.Found || !slices.Contains([]string{"PROGRAM_STATUS_SUCCEEDED", "PROGRAM_STATUS_FAILED", "PROGRAM_STATUS_CANCELLED"}, result.Status) || len(result.Stdout) > 65536 || result.Status != "PROGRAM_STATUS_SUCCEEDED" && result.Stdout != "" || p.CheckProgram(ctx, result.Identity) != nil {
				return ErrSourceAuthority
			}
			return o.store.completeSelection(ctx, selected, result, func() error { return p.Recheck(ctx) })
		})
	})
	if err != nil || !invoked || ctx.Err() != nil {
		return ErrSourceAuthority
	}
	return nil
}

// BindSelection durably pins the trusted approved gate digests before broker use.
// It never replaces an existing binding, even after restart. The pure selection
// projection alone cannot authorize retirement. This method creates no source
// authority and does not attest deployment or confinement.
func (o *NativeReceiptOwner) BindSelection(ctx context.Context, selected *NativeSelection, events, collector *SourceGate) (SourceBrokerSelection, error) {
	if o == nil || !o.enabled {
		return SourceBrokerSelection{}, ErrDisabled
	}
	if ctx == nil || ctx.Err() != nil || selected == nil || selected.digest != o.digest || !validNativeSelection(selected) {
		return SourceBrokerSelection{}, ErrSourceAuthority
	}
	projection, err := selected.BrokerSelection(events, collector)
	if err != nil {
		return SourceBrokerSelection{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	err = o.guard.WithCurrent(ctx, o.scopeCopy(), func(ctx context.Context) error {
		return o.store.bindSelection(ctx, selected, events.digest, collector.digest)
	})
	if err != nil || ctx.Err() != nil {
		return SourceBrokerSelection{}, ErrSourceAuthority
	}
	return projection, nil
}
