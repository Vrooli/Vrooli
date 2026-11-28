// Package bridge owns the deliberately dormant local-work wake projection.
// Events are observations, not grants. No title, body, path or raw payload is sent.
package bridge

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"time"

	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/subscription"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	WakeSchema        = "flywheel.local-work-wake.v2"
	RunEventType      = "agent_manager.run.notification.v1"
	TerminalEventType = "agent_manager.run.terminal.v1"
	// ReservedTarget is not a URL. Generic webhook delivery must never handle it.
	ReservedTarget           = "flywheel:local-work:v2"
	subscriptionName         = "flywheel-local-work-v2"
	terminalSubscriptionName = "flywheel-local-work-terminal-v1"
	maxWakeBytes             = 8192
)

var (
	ErrWakeRejected = errors.New("local-work wake rejected")
	ErrWakeDisabled = errors.New("local-work wake disabled")
	opaque          = regexp.MustCompile(`^[A-Za-z0-9_.:/-]{1,64}$`)
	eventIdentity   = regexp.MustCompile(`^[A-Za-z0-9_.:/-]{1,256}$`)
	digest          = regexp.MustCompile(`^[0-9a-f]{64}$`)
	runIdentity     = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

type Scope struct {
	Epoch      string `json:"epoch"`
	Digest     string `json:"digest"`
	Checkpoint string `json:"checkpoint"`
}

// Policy comes from approved owner configuration, NEVER from an event or HTTP
// request. An epoch changes when source selection/consent changes. The digest
// identifies that exact selection. Unknown coverage remains an explicit hold.
type Policy struct {
	Enabled        bool
	SourceInstance string
	Scope          Scope
	PermissionHold bool
}

// WakeBoundary serializes policy revocation/replacement with every callback.
// A production implementation must keep its policy current and the boundary
// held through the send, and honor cancellation. Returning cached authority
// without sharing the revocation boundary is not a production implementation.
// No production authority provider is installed by this change.
type WakeBoundary interface {
	WithinWake(context.Context, func(context.Context, Policy) error) error
}

type RunLocator struct {
	Schema                    string `json:"schema"`
	OwnerID                   string `json:"owner_id"`
	RunID                     string `json:"run_id"`
	SourceInstance            string `json:"source_instance,omitempty"`
	SourceEpoch               string `json:"source_epoch,omitempty"`
	LifecycleVersion          string `json:"lifecycle_version,omitempty"`
	TerminalObservationSHA256 string `json:"terminal_observation_sha256,omitempty"`
}

type WakeEvent struct {
	EventID             string     `json:"event_id"`
	EventType           string     `json:"event_type"`
	Source              string     `json:"source"`
	OccurredAt          string     `json:"occurred_at"`
	SourceContentDigest string     `json:"source_content_digest"`
	Locator             RunLocator `json:"locator"`
}

type Wake struct {
	Schema          string      `json:"schema"`
	SourceInstance  string      `json:"source_instance"`
	Events          []WakeEvent `json:"events"`
	ObservedAt      string      `json:"observed_at"`
	Scope           Scope       `json:"scope"`
	PermissionHold  bool        `json:"permission_hold"`
	CoverageGap     *string     `json:"coverage_gap"`
	Purpose         string      `json:"purpose"`
	Activation      string      `json:"activation"`
	EffectAuthority string      `json:"effect_authority"`
	BatchID         string      `json:"batch_id,omitempty"`
}

// SubscriptionSpec returns an inert specification only. It does not create or
// enable a subscription. Registration is an explicit activation step.
func SubscriptionSpec() subscription.Subscription {
	return subscription.Subscription{
		Name: subscriptionName, OwnerScenario: "vrooli-events",
		EventPattern: RunEventType, SourceFilter: "agent-manager", DeliveryType: subscription.DeliveryWebhook,
		DeliveryTarget: ReservedTarget, Enabled: false,
	}
}

// TerminalSubscriptionSpec is a separate, inert opt-in. It does not widen the
// legacy notification selection or register/enable a subscription.
func TerminalSubscriptionSpec() subscription.Subscription {
	spec := SubscriptionSpec()
	spec.Name = terminalSubscriptionName
	spec.EventPattern = TerminalEventType
	return spec
}

func Reserved(sub subscription.Subscription) bool {
	return sub.DeliveryTarget == ReservedTarget || sub.Name == subscriptionName || sub.Name == terminalSubscriptionName
}

func ValidSubscription(sub subscription.Subscription) bool {
	want := SubscriptionSpec()
	if sub.Name == terminalSubscriptionName {
		want = TerminalSubscriptionSpec()
	}
	return sub.Enabled && sub.ID > 0 && sub.Name == want.Name && sub.OwnerScenario == want.OwnerScenario &&
		sub.EventPattern == want.EventPattern && sub.SourceFilter == want.SourceFilter &&
		sub.DeliveryType == want.DeliveryType && sub.DeliveryTarget == want.DeliveryTarget
}

// IsWakeBody also quarantines a previously reserved row after its subscription
// is edited. Its contents must never fall back to generic webhook transport.
func IsWakeBody(body string) bool {
	var head struct {
		Schema string `json:"schema"`
	}
	return json.Unmarshal([]byte(body), &head) == nil && head.Schema == WakeSchema
}

func validPolicy(p Policy) bool {
	return p.Enabled && opaque.MatchString(p.SourceInstance) && opaque.MatchString(p.Scope.Epoch) &&
		digest.MatchString(p.Scope.Digest) && (p.Scope.Checkpoint == "known" || p.Scope.Checkpoint == "unknown")
}

func (f *Forwarder) within(ctx context.Context, fn func(context.Context, Policy) error) error {
	if f == nil || f.boundary == nil || ctx == nil || ctx.Err() != nil {
		return ErrWakeDisabled
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	called := false
	err := f.boundary.WithinWake(ctx, func(inner context.Context, p Policy) error {
		if called || inner == nil || inner.Err() != nil || ctx.Err() != nil || !validPolicy(p) {
			return ErrWakeDisabled
		}
		called = true
		deadline, _ := ctx.Deadline()
		bounded, cancelInner := context.WithDeadline(inner, deadline)
		defer cancelInner()
		stop := context.AfterFunc(ctx, cancelInner)
		defer stop()
		return fn(bounded, p)
	})
	if err != nil || !called || ctx.Err() != nil {
		return ErrWakeRejected
	}
	return nil
}

// Project reads canonical protobuf directly (no occurrence fallback), checks
// its indexes (the durable occurrence index is millisecond precision), and
// takes only the explicit run locator from the selected type.
// The inner payload can contain arbitrary untrusted text; none is forwarded.
func (f *Forwarder) Project(ctx context.Context, e store.Event) (body string, selected bool, err error) {
	if !supportedRunEvent(e.EventType) || e.SourceScenario != "agent-manager" {
		return "", false, nil
	}
	if f == nil {
		return "", false, nil
	}
	err = f.within(ctx, func(_ context.Context, p Policy) error {
		if len(e.Payload) == 0 || len(e.Payload) > 1<<20 {
			return ErrWakeRejected
		}
		var env domain.EventEnvelope
		if proto.Unmarshal(e.Payload, &env) != nil || !eventIdentity.MatchString(env.EventId) || env.EventId != e.EventID ||
			env.EventType != e.EventType || env.GetSource().GetScenario() != e.SourceScenario ||
			env.OccurredAt == nil || env.OccurredAt.CheckValid() != nil || !env.OccurredAt.AsTime().Truncate(time.Millisecond).Equal(e.CreatedAt.Truncate(time.Millisecond)) {
			return ErrWakeRejected
		}
		var facts structpb.Struct
		if env.Data == nil || env.Data.UnmarshalTo(&facts) != nil {
			return ErrWakeRejected
		}
		locator, _, _, err := projectRunFacts(env.EventType, &facts)
		if err != nil {
			return ErrWakeRejected
		}
		now := f.now().UTC()
		occurred := env.OccurredAt.AsTime().UTC()
		if occurred.After(now.Add(time.Minute)) {
			return ErrWakeRejected
		}
		sum := sha256.Sum256(e.Payload)
		wake := Wake{
			Schema: WakeSchema, SourceInstance: p.SourceInstance,
			Events: []WakeEvent{{
				EventID: e.EventID, EventType: e.EventType, Source: "agent-manager", OccurredAt: canonicalOccurrence(occurred),
				SourceContentDigest: hex.EncodeToString(sum[:]), Locator: locator,
			}},
			ObservedAt: now.Format(time.RFC3339Nano), Scope: p.Scope, PermissionHold: p.PermissionHold,
			Purpose: "request_read_only_reconciliation", Activation: "not_approved", EffectAuthority: "none",
		}
		unsigned, encodeErr := canonical(wake)
		if encodeErr != nil {
			return ErrWakeRejected
		}
		hash := sha256.Sum256(unsigned)
		wake.BatchID = hex.EncodeToString(hash[:])
		encoded, encodeErr := canonical(wake)
		if encodeErr != nil || len(encoded) > maxWakeBytes {
			return ErrWakeRejected
		}
		body, selected = string(encoded), true
		return nil
	})
	if err != nil {
		return "", false, err
	}
	return body, selected, nil
}

func canonical(v any) ([]byte, error) {
	encoded, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var object map[string]any
	if err = json.Unmarshal(encoded, &object); err != nil {
		return nil, err
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err = enc.Encode(object); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(b.Bytes(), []byte("\n")), nil
}

func decodeWake(body string, p Policy, eventID string) (Wake, error) {
	var w Wake
	if len(body) == 0 || len(body) > maxWakeBytes {
		return w, ErrWakeRejected
	}
	d := json.NewDecoder(bytes.NewBufferString(body))
	d.DisallowUnknownFields()
	if d.Decode(&w) != nil || w.Schema != WakeSchema || len(w.Events) != 1 || w.SourceInstance != p.SourceInstance ||
		w.Scope != p.Scope || w.PermissionHold != p.PermissionHold || w.CoverageGap != nil ||
		w.Purpose != "request_read_only_reconciliation" || w.Activation != "not_approved" || w.EffectAuthority != "none" {
		return w, ErrWakeRejected
	}
	e := w.Events[0]
	if e.EventID != eventID || !eventIdentity.MatchString(e.EventID) || !supportedRunEvent(e.EventType) || e.Source != "agent-manager" ||
		!digest.MatchString(e.SourceContentDigest) || !validRunLocator(e.EventType, e.Locator) ||
		!runIdentity.MatchString(e.Locator.RunID) || e.Locator.RunID == "00000000-0000-0000-0000-000000000000" {
		return w, ErrWakeRejected
	}
	occurred, err := time.Parse(time.RFC3339Nano, e.OccurredAt)
	if err != nil {
		return w, ErrWakeRejected
	}
	observed, err := time.Parse(time.RFC3339Nano, w.ObservedAt)
	if err != nil || occurred.After(observed.Add(time.Minute)) {
		return w, ErrWakeRejected
	}
	reencoded, err := canonical(w)
	if err != nil || string(reencoded) != body {
		return w, ErrWakeRejected
	}
	id := w.BatchID
	w.BatchID = ""
	unsigned, err := canonical(w)
	if err != nil {
		return w, ErrWakeRejected
	}
	hash := sha256.Sum256(unsigned)
	if id != hex.EncodeToString(hash[:]) {
		return w, ErrWakeRejected
	}
	w.BatchID = id
	return w, nil
}

// Stored canonical protobuf timestamps have seconds/nanos, not a textual
// original lexeme. Match protojson's 0/3/6/9-digit UTC representation so the
// selected canonical reader can compare evidence without JS millisecond loss.
func canonicalOccurrence(t time.Time) string {
	t = t.UTC()
	switch {
	case t.Nanosecond() == 0:
		return t.Format("2006-01-02T15:04:05Z")
	case t.Nanosecond()%1000000 == 0:
		return t.Format("2006-01-02T15:04:05.000Z")
	case t.Nanosecond()%1000 == 0:
		return t.Format("2006-01-02T15:04:05.000000Z")
	default:
		return t.Format("2006-01-02T15:04:05.000000000Z")
	}
}

// Both variants are observations only. Terminal selectors bind the original
// immutable receipt, not the latest run state. Legacy wire output is unchanged.
func supportedRunEvent(typ string) bool { return typ == RunEventType || typ == TerminalEventType }

var terminalScopeToken = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$`)

func validRunLocator(typ string, l RunLocator) bool {
	if l.OwnerID != "agent-manager" || !runIdentity.MatchString(l.RunID) || l.RunID == "00000000-0000-0000-0000-000000000000" {
		return false
	}
	if typ == RunEventType {
		return l.Schema == "agent-manager.run-event.v1" && l.SourceInstance == "" && l.SourceEpoch == "" && l.LifecycleVersion == "" && l.TerminalObservationSHA256 == ""
	}
	version, err := strconv.ParseInt(l.LifecycleVersion, 10, 64)
	return typ == TerminalEventType && l.Schema == "agent-manager.run-terminal.v1" && terminalScopeToken.MatchString(l.SourceInstance) && terminalScopeToken.MatchString(l.SourceEpoch) && digest.MatchString(l.TerminalObservationSHA256) && err == nil && version > 0 && strconv.FormatInt(version, 10) == l.LifecycleVersion
}
func projectRunFacts(typ string, facts *structpb.Struct) (RunLocator, string, string, error) {
	fields := facts.GetFields()
	value := func(key string) string { return fields[key].GetStringValue() }
	l := RunLocator{Schema: "agent-manager.run-event.v1", OwnerID: "agent-manager"}
	category, status := "", ""
	if typ == RunEventType {
		l.RunID = fields["correlation"].GetStructValue().GetFields()["run_id"].GetStringValue()
		category = value("category")
		if category != "run-completed" && category != "run-failed" {
			return l, "", "", ErrWakeRejected
		}
	} else if typ == TerminalEventType {
		if len(fields) != 6 {
			return l, "", "", ErrWakeRejected
		}
		l.Schema = "agent-manager.run-terminal.v1"
		l.RunID, l.SourceInstance, l.SourceEpoch = value("run_id"), value("source_instance"), value("source_epoch")
		l.LifecycleVersion, l.TerminalObservationSHA256 = value("lifecycle_version"), value("terminal_observation_sha256")
		status = value("status")
		switch status {
		case "complete", "failed", "cancelled", "unknown":
		default:
			return l, "", "", ErrWakeRejected
		}
	}
	if !validRunLocator(typ, l) {
		return l, "", "", ErrWakeRejected
	}
	return l, category, status, nil
}
