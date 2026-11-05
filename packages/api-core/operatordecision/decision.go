// Package operatordecision is the domain-neutral substrate for asking a human a
// yes/no (or small-menu) question and acting on the answer. It collapses the
// four bespoke "operator approval" paths that grew up inside agent-manager
// (effort decisions, run approvals, "ask, don't block" authorization grants, and
// interactive authorization pushes) into one shared library every scenario can
// reuse.
//
// The contract is data-driven and rides the existing Vrooli Events + notification
// -hub rails, so NO new proto message types are required. A producer publishes a
// generic `<source>.<kind>.decision_requested.v1` domain event whose payload is a
// facts map (subject, question, options, recommended, reversible, …). The hub
// turns that into an operator Ask (phone/desktop/email) with no per-scenario code.
// The operator's answer returns as `notification_hub.ask.resolved.v1`, which this
// package decodes into a DecisionResolution and fans out to DomainEffects.
//
// The package is storage-free and scenario-free: it imports only api-core's
// eventbus and schedule. A scenario supplies its own durable store (implementing
// PendingStore), its own Publisher (api-core eventbus.Client satisfies it), its
// own Clock, and — where it wants a context link — resolves the context_url
// before building Facts. The SQL/durability lives per-scenario, never here.
package operatordecision

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/vrooli/api-core/eventbus"
)

// AskResolvedEventType is notification-hub's resolution event type. A scenario's
// resolution webhook gates on this before fanning a DecisionResolution to its
// DomainEffects. It mirrors supervision.AskResolvedEventType in agent-manager;
// phase 2 reseats agent-manager onto this constant.
const AskResolvedEventType = "notification_hub.ask.resolved.v1"

// DecisionRequestedSuffix is the event-type suffix every producer appends after
// its `<source>.<kind>` prefix. notification-hub subscribes to the wildcard
// `**.decision_requested.v1`, so emitting `<source>.<kind>.decision_requested.v1`
// needs no hub change for a new scenario.
const DecisionRequestedSuffix = ".decision_requested.v1"

// DecisionRequestedEventType composes the full producer event type from a source
// scenario and a decision kind (e.g. ("deployment-manager","release") →
// "deployment-manager.release.decision_requested.v1"). Both parts are required.
func DecisionRequestedEventType(source, kind string) string {
	return strings.TrimSpace(source) + "." + strings.TrimSpace(kind) + DecisionRequestedSuffix
}

// EventID is the id eventbus.Client assigns to a published event: the SHA-256 of
// its JSON encoding. Because the encoding is deterministic, a producer that
// rebuilds the same event (same Source, EventType, Occurred, Payload) gets the
// same id — so a retry after a lost acknowledgement is deduplicated by Vrooli
// Events and the hub. It matches effortdecision.EventID byte-for-byte.
func EventID(event eventbus.DomainEvent) string {
	seed, _ := json.Marshal(event)
	sum := sha256.Sum256(seed)
	return hex.EncodeToString(sum[:])
}

// CorrelationID is the stable idempotency key for an ask: the same logical ask
// (the same ordered set of parts) always hashes to the same id, so a repeated
// raise reuses the one pending row and the one operator notification. Parts are
// NUL-joined (after trimming) and hashed; the first 16 bytes are rendered hex
// behind the given prefix. It generalizes agent-manager's
// authorizationCorrelationID (which was CorrelationID("auth", subject, operation,
// target)).
func CorrelationID(prefix string, parts ...string) string {
	trimmed := make([]string, len(parts))
	for i, p := range parts {
		trimmed[i] = strings.TrimSpace(p)
	}
	sum := sha256.Sum256([]byte(strings.Join(trimmed, "\x00")))
	id := hex.EncodeToString(sum[:16])
	if prefix = strings.TrimSpace(prefix); prefix != "" {
		return prefix + "-" + id
	}
	return id
}
