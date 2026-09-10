// Package targetmodel owns the provider-neutral target and transport model
// shared by local probes, bridge inventory, and cross-OS selectors.
package targetmodel

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/api-core/scopecatalog"
)

// TransportKind identifies how a caller reaches a target.
type TransportKind string

const (
	TransportLocal  TransportKind = "local"
	TransportBridge TransportKind = "bridge"
	// TransportSSH reaches a target over an operator-supplied SSH binding
	// without Bridge enrollment. It never hosts agent sessions and never
	// falls back to or from Bridge.
	TransportSSH TransportKind = "ssh"
)

// Transport carries reachability metadata. Endpoint is intentionally omitted
// from JSON because credentials and concrete endpoints are not evidence.
type Transport struct {
	Kind      TransportKind `json:"kind"`
	ID        string        `json:"id"`
	Trust     string        `json:"trust,omitempty"`
	Endpoint  string        `json:"-"`
	Available bool          `json:"available"`
	Reason    string        `json:"reason,omitempty"`
}

type TargetHealth struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

type BridgeTrust struct {
	Registered         bool   `json:"registered"`
	Online             bool   `json:"online"`
	DispatchAuthorized bool   `json:"dispatch_authorized"`
	Reason             string `json:"reason,omitempty"`
}

// ReadinessCheck is the provider-neutral shape used by product surfaces to
// explain why a target can or cannot be selected. Identity is stable for
// machine logic; label, detail, and recovery action are presentation-safe.
type ReadinessCheck struct {
	Identity string `json:"identity"`
	Label    string `json:"label"`
	Passed   bool   `json:"passed"`
	// State is the richer capability vocabulary. Transport checks retain their
	// boolean Passed value; capability checks use all four states below.
	State          ReadinessState `json:"state,omitempty"`
	Version        string         `json:"version,omitempty"`
	Detail         string         `json:"detail,omitempty"`
	RecoveryAction string         `json:"recovery_action,omitempty"`
}

// ReadinessState distinguishes an observed absence from an unsupported
// platform and from an observation that is not available yet.
type ReadinessState string

const (
	ReadinessReady         ReadinessState = "ready"
	ReadinessMissing       ReadinessState = "missing"
	ReadinessNotApplicable ReadinessState = "not_applicable"
	ReadinessUnknown       ReadinessState = "unknown"
)

const (
	ReadinessRegistry         = "registry_record"
	ReadinessHeartbeat        = "heartbeat_fresh"
	ReadinessChannel          = "channel_held"
	ReadinessProtocol         = "protocol_compatible"
	ReadinessDispatch         = "dispatchable"
	ReadinessBridgeScope      = "bridge_scope"
	ReadinessSessionSupport   = "session_support"
	ReadinessCapabilityPrefix = "capability:"
)

// Operation names are deliberately closed: readiness is evaluated for a
// requested operation, not as one broad "target is ready" bit. A headless
// operation therefore does not inherit visual-session prerequisites.
const (
	OperationHeadlessExecution = "headless_execution"
	OperationVisualValidation  = "visual_validation"
	OperationProvisioning      = "provisioning"
	OperationDeviceOperation   = "device_operation"

	DefaultReadinessStaleAfter = 45 * time.Second
)

// OperationReadiness is the safe, target-scoped admission result shared by
// API, Doctor, and selection surfaces. ObservedAt/FreshUntil describe the
// freshness envelope only; they are never authorization or credential data.
type OperationReadiness struct {
	Operation      string         `json:"operation"`
	Ready          bool           `json:"ready"`
	State          ReadinessState `json:"state"`
	ReasonCode     string         `json:"reason_code,omitempty"`
	Detail         string         `json:"detail,omitempty"`
	RecoveryAction string         `json:"recovery_action,omitempty"`
	ObservedAt     time.Time      `json:"observed_at,omitempty"`
	FreshUntil     time.Time      `json:"fresh_until,omitempty"`
	Source         string         `json:"source,omitempty"`
}

// ReadinessCheckFor resolves the stable identity to the common operator
// wording. Unknown identities remain explicit instead of disappearing from a
// product surface.
func ReadinessCheckFor(identity string, passed bool, detail string) ReadinessCheck {
	identity = strings.TrimSpace(identity)
	labels := map[string]string{
		ReadinessRegistry:       "Registered",
		ReadinessHeartbeat:      "Heartbeat fresh",
		ReadinessChannel:        "Live channel",
		ReadinessProtocol:       "Protocol compatible",
		ReadinessDispatch:       "Dispatchable",
		ReadinessBridgeScope:    "Bridge scope",
		ReadinessSessionSupport: "Session support",
	}
	label := labels[identity]
	if label == "" {
		label = identity
	}
	state := ReadinessMissing
	if passed {
		state = ReadinessReady
	}
	return ReadinessCheck{Identity: identity, Label: label, Passed: passed, State: state, Detail: detail, RecoveryAction: recoveryAction(identity, passed)}
}

// CapabilityReadinessCheck creates the operator-facing fact for one named
// capability. Unlike transport readiness, missing capability facts do not
// make the target itself undispatchable.
//
// label is the capability's human name ("Claude Code"), which every producer
// already carries: the host probe defines it and the Bridge heartbeat forwards
// it intact. Before this parameter existed the constructor set Label to the
// slug, so the name was discarded at the last hop and every consumer rendered
// "claude". A blank label still falls back to the slug, because a fact with no
// name at all is worse than one named after its id.
func CapabilityReadinessCheck(capability, label string, state ReadinessState, detail, recovery string) ReadinessCheck {
	capability = strings.TrimSpace(capability)
	label = strings.TrimSpace(label)
	if label == "" {
		label = capability
	}
	if state != ReadinessReady && state != ReadinessMissing && state != ReadinessNotApplicable && state != ReadinessUnknown {
		state = ReadinessUnknown
	}
	return ReadinessCheck{
		Identity:       ReadinessCapabilityPrefix + capability,
		Label:          label,
		Passed:         state == ReadinessReady,
		State:          state,
		Detail:         detail,
		RecoveryAction: recovery,
	}
}

func recoveryAction(identity string, passed bool) string {
	if passed {
		return ""
	}
	switch identity {
	case ReadinessHeartbeat, ReadinessChannel:
		return "Reconnect the Bridge agent, then refresh"
	case ReadinessProtocol:
		return "Update or provision the Bridge agent, then refresh"
	case ReadinessBridgeScope:
		return "Grant the required Bridge scope in the node settings"
	case ReadinessSessionSupport:
		return "Enable session support on the node"
	default:
		return "Refresh the target and inspect its readiness"
	}
}

// HeartbeatFresh computes freshness from the timestamp itself. A missing
// timestamp is never fresh, even when a transport channel is still present.
func HeartbeatFresh(lastSeen, now time.Time, staleAfter time.Duration) (bool, time.Duration) {
	if lastSeen.IsZero() || staleAfter <= 0 {
		return false, 0
	}
	age := now.UTC().Sub(lastSeen.UTC())
	return age >= 0 && age <= staleAfter, age
}

// Target is one observable execution destination. It contains identity,
// platform, capability, transport, trust, health, and explicit recovery
// information so an unavailable selection is actionable rather than guessed.
type Target struct {
	ID                 string               `json:"id"`
	Ramp               string               `json:"ramp,omitempty"`
	Label              string               `json:"label"`
	Platform           string               `json:"platform"`
	OS                 string               `json:"os"`
	Architecture       string               `json:"architecture"`
	DeviceKind         string               `json:"device_kind"`
	Revision           string               `json:"revision,omitempty"`
	LastSeenAt         time.Time            `json:"last_seen_at,omitempty"`
	SurvivesRestart    bool                 `json:"survives_restart"`
	Transport          Transport            `json:"transport"`
	NodeID             string               `json:"node_id,omitempty"`
	Mode               string               `json:"mode,omitempty"`
	Capabilities       []string             `json:"capabilities,omitempty"`
	Scopes             []string             `json:"scopes,omitempty"`
	Available          bool                 `json:"available"`
	Reason             string               `json:"reason,omitempty"`
	MissingCapability  string               `json:"missing_capability,omitempty"`
	NextAction         string               `json:"next_action,omitempty"`
	Health             TargetHealth         `json:"health"`
	BridgeTrust        *BridgeTrust         `json:"bridge_trust,omitempty"`
	Revoked            bool                 `json:"revoked,omitempty"`
	Readiness          []ReadinessCheck     `json:"readiness,omitempty"`
	OperationReadiness []OperationReadiness `json:"operation_readiness,omitempty"`
}

func (t Target) Validate() error {
	if strings.TrimSpace(t.ID) == "" {
		return fmt.Errorf("target id is required")
	}
	if strings.TrimSpace(t.Platform) == "" {
		return fmt.Errorf("target %q platform is required", t.ID)
	}
	if t.Transport.Kind != TransportLocal && t.Transport.Kind != TransportBridge && t.Transport.Kind != TransportSSH {
		return fmt.Errorf("target %q has invalid transport %q", t.ID, t.Transport.Kind)
	}
	if !t.Available {
		if strings.TrimSpace(t.MissingCapability) == "" {
			return fmt.Errorf("target %q is unavailable without a missing capability", t.ID)
		}
		if strings.TrimSpace(t.NextAction) == "" {
			return fmt.Errorf("target %q is unavailable without a next action", t.ID)
		}
	}
	return nil
}

func (t Target) Supports(capability string) bool {
	wanted := strings.TrimSpace(capability)
	if wanted == "" {
		return false
	}
	for _, observed := range t.Capabilities {
		if strings.EqualFold(strings.TrimSpace(observed), wanted) {
			return true
		}
	}
	return false
}

// CanHostSession is the compatibility wrapper for headless execution. The
// operation-specific predicate is shared by target catalogs, launch handlers,
// and readiness projections; this legacy API keeps its boolean result stable.
func CanHostSession(t Target) (bool, string) {
	decision := EvaluateOperationReadiness(t, OperationHeadlessExecution, time.Now().UTC())
	if decision.Ready {
		return true, ""
	}
	return false, decision.Detail
}

// EvaluateOperations computes every supported operation from the same target
// snapshot. Callers may render all decisions, but must use the decision for
// the operation they are about to perform.
func EvaluateOperations(t Target, now time.Time) []OperationReadiness {
	return []OperationReadiness{
		EvaluateOperationReadiness(t, OperationHeadlessExecution, now),
		EvaluateOperationReadiness(t, OperationVisualValidation, now),
		EvaluateOperationReadiness(t, OperationProvisioning, now),
		EvaluateOperationReadiness(t, OperationDeviceOperation, now),
	}
}

// EvaluateOperationReadiness is the canonical freshness-qualified predicate
// for target admission. It is intentionally conservative: an absent or
// unknown observation is never promoted to ready, while headless execution
// remains independent from visual-session prerequisites.
func EvaluateOperationReadiness(t Target, operation string, now time.Time) OperationReadiness {
	decision := OperationReadiness{Operation: operation, State: ReadinessUnknown, Source: "target_projection"}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if !t.LastSeenAt.IsZero() {
		decision.ObservedAt = t.LastSeenAt.UTC()
		decision.FreshUntil = decision.ObservedAt.Add(DefaultReadinessStaleAfter)
	}

	switch operation {
	case OperationHeadlessExecution, OperationVisualValidation, OperationProvisioning, OperationDeviceOperation:
	default:
		return operationFailure(decision, ReadinessUnknown, "operation_unknown", "target operation is not supported", "choose a supported operation")
	}
	if t.Revoked {
		return operationFailure(decision, ReadinessMissing, "grant_revoked", "target authority has been revoked", "restore the target grant and refresh")
	}
	if operation == OperationDeviceOperation && t.DeviceKind != "attached" {
		return operationFailure(decision, ReadinessNotApplicable, "device_not_applicable", "device operations apply only to attached targets", "select an attached target")
	}
	if t.Transport.Kind == TransportLocal {
		if operation == OperationVisualValidation && !hasReadyCapability(t, "gui_session", "desktop_session", "gui") {
			return operationFailure(decision, ReadinessMissing, "gui_session_missing", "visual validation requires an active GUI session", "log in to a desktop session, then refresh")
		}
		return operationReady(decision)
	}
	if t.Transport.Kind != TransportBridge {
		return operationFailure(decision, ReadinessNotApplicable, "transport_unsupported", "this target does not provide a supported Bridge operation transport", "select a Bridge-backed target")
	}
	if t.DeviceKind != "bridge-node" {
		return operationFailure(decision, ReadinessNotApplicable, "operation_not_supported", "this registered target does not host Bridge agent operations", "select a Bridge agent target")
	}
	if t.BridgeTrust == nil || !t.BridgeTrust.Registered {
		return operationFailure(decision, ReadinessMissing, "target_unregistered", "this node is not registered with Bridge", "register the node, then refresh")
	}
	if !t.BridgeTrust.Online || !t.Transport.Available {
		return operationFailure(decision, ReadinessMissing, "channel_unavailable", "this node is offline or its Bridge channel is unavailable", "reconnect the Bridge agent, then refresh")
	}
	if fresh, _ := HeartbeatFresh(t.LastSeenAt, now, DefaultReadinessStaleAfter); !t.LastSeenAt.IsZero() && !fresh {
		return operationFailure(decision, ReadinessUnknown, "heartbeat_stale", "host facts are stale; readiness was not assumed", "refresh the target heartbeat before admission")
	}
	for _, required := range []struct {
		identity string
		code     string
		detail   string
		recovery string
	}{
		{ReadinessRegistry, "target_unregistered", "Bridge registry record is not present", "register the node, then refresh"},
		{ReadinessHeartbeat, "heartbeat_stale", "host heartbeat is not fresh", "refresh the target heartbeat before admission"},
		{ReadinessChannel, "channel_unavailable", "Bridge does not hold a live channel", "reconnect the Bridge agent, then refresh"},
		{ReadinessProtocol, "protocol_incompatible", "Bridge protocol is not compatible", "update the Bridge agent, then refresh"},
		{ReadinessDispatch, "dispatch_unavailable", "Bridge dispatch is not available", "restore dispatchability, then refresh"},
		{ReadinessBridgeScope, "grant_revoked", "the required Bridge operation grant is not approved", "restore the target grant and refresh"},
	} {
		if fact, ok := readinessFact(t, required.identity); ok {
			state := fact.State
			if state == "" {
				if fact.Passed {
					state = ReadinessReady
				} else {
					state = ReadinessMissing
				}
			}
			if state != ReadinessReady || !fact.Passed {
				if state == ReadinessUnknown {
					return operationFailure(decision, ReadinessUnknown, "readiness_unknown", fact.Detail, required.recovery)
				}
				return operationFailure(decision, ReadinessMissing, required.code, nonEmpty(fact.Detail, required.detail), nonEmpty(fact.RecoveryAction, required.recovery))
			}
		}
	}
	if !hasInteractiveGrant(t.Scopes) {
		return operationFailure(decision, ReadinessMissing, "grant_revoked", "the required Bridge operation grant is not approved", "restore the target grant and refresh")
	}
	if operation == OperationVisualValidation && !hasReadyCapability(t, "gui_session", "desktop_session", "gui") {
		return operationFailure(decision, ReadinessMissing, "gui_session_missing", "visual validation requires an active GUI session", "log in to a desktop session, then refresh")
	}
	if operation == OperationDeviceOperation && !hasReadyCapability(t, "device_adapter", "attached_device") {
		return operationFailure(decision, ReadinessUnknown, "device_adapter_unknown", "attached-device adapter readiness has not been observed", "refresh the attached-device inventory before admission")
	}
	return operationReady(decision)
}

func operationReady(decision OperationReadiness) OperationReadiness {
	decision.Ready = true
	decision.State = ReadinessReady
	decision.Detail = "all prerequisites for this operation are fresh and authorized"
	return decision
}

func operationFailure(decision OperationReadiness, state ReadinessState, code, detail, recovery string) OperationReadiness {
	decision.State, decision.ReasonCode, decision.Detail, decision.RecoveryAction = state, code, detail, recovery
	return decision
}

func readinessFact(target Target, identity string) (ReadinessCheck, bool) {
	for _, fact := range target.Readiness {
		if fact.Identity == identity {
			return fact, true
		}
	}
	return ReadinessCheck{}, false
}

func hasReadyCapability(target Target, capabilities ...string) bool {
	for _, capability := range capabilities {
		identity := ReadinessCapabilityPrefix + capability
		if fact, ok := readinessFact(target, identity); ok {
			state := fact.State
			if state == "" && fact.Passed {
				state = ReadinessReady
			}
			if state == ReadinessReady && fact.Passed {
				return true
			}
			continue
		}
		if target.Supports(capability) {
			return true
		}
	}
	return false
}

func hasInteractiveGrant(scopes []string) bool {
	transportScope, ok := scopecatalog.TransportScope("interactive-session:write")
	return ok && scopecatalog.Resolve(scopes, transportScope)
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

// Inventory is a point-in-time target snapshot.
type Inventory struct {
	Targets  []Target  `json:"targets"`
	Observed time.Time `json:"observed_at"`
}

func (i Inventory) Validate() error {
	if len(i.Targets) == 0 {
		return fmt.Errorf("inventory contains no targets")
	}
	seen := make(map[string]struct{}, len(i.Targets))
	for index, target := range i.Targets {
		if err := target.Validate(); err != nil {
			return fmt.Errorf("target %d: %w", index, err)
		}
		if _, exists := seen[target.ID]; exists {
			return fmt.Errorf("inventory contains duplicate target identity %q", target.ID)
		}
		seen[target.ID] = struct{}{}
	}
	return nil
}

// SelectionRequest describes the shared target constraints used by both
// bridge gates and delivery validation matrices.
type SelectionRequest struct {
	OS                   string          `json:"os"`
	RequiredCapabilities []string        `json:"required_capabilities,omitempty"`
	TransportKinds       []TransportKind `json:"transport_kinds,omitempty"`
	Operation            string          `json:"operation,omitempty"`
}

// Selection is an explicit result, including the reason and next action when
// the requested target cannot currently be used.
type Selection struct {
	Target     Target `json:"target"`
	Found      bool   `json:"found"`
	Available  bool   `json:"available"`
	Reason     string `json:"reason"`
	NextAction string `json:"next_action"`
}

// Select deterministically chooses the lowest-ID target matching the request.
// It considers every matching target before declaring the request unavailable,
// so a degraded target does not mask another healthy target on the same OS.
func Select(inventory Inventory, request SelectionRequest) Selection {
	os := strings.ToLower(strings.TrimSpace(request.OS))
	if os == "" {
		return Selection{Reason: "target OS is required", NextAction: "provide a target OS and select again"}
	}

	candidates := make([]Target, 0, len(inventory.Targets))
	for _, target := range inventory.Targets {
		if strings.ToLower(strings.TrimSpace(target.OS)) != os || !transportAllowed(target.Transport.Kind, request.TransportKinds) {
			continue
		}
		candidates = append(candidates, target)
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })
	if len(candidates) == 0 {
		return Selection{Reason: fmt.Sprintf("no target matches OS %q", os), NextAction: "register or probe a target for this OS"}
	}

	var unavailable *Selection
	for _, target := range candidates {
		if !target.Available {
			if unavailable == nil {
				candidate := Selection{Target: target, Found: true, Reason: target.Reason, NextAction: target.NextAction}
				if candidate.Reason == "" {
					candidate.Reason = fmt.Sprintf("target %q is unavailable", target.ID)
				}
				if candidate.NextAction == "" {
					candidate.NextAction = "restore target availability and select again"
				}
				unavailable = &candidate
			}
			continue
		}
		if request.Operation != "" {
			decision := EvaluateOperationReadiness(target, request.Operation, time.Now().UTC())
			if !decision.Ready {
				if unavailable == nil {
					unavailable = &Selection{Target: target, Found: true, Reason: decision.Detail, NextAction: decision.RecoveryAction}
				}
				continue
			}
		}
		missing := missingCapabilities(target, request.RequiredCapabilities)
		if len(missing) > 0 {
			if unavailable == nil {
				unavailable = &Selection{
					Target:     target,
					Found:      true,
					Reason:     fmt.Sprintf("target %q is missing capabilities: %s", target.ID, strings.Join(missing, ", ")),
					NextAction: "provide the missing capabilities and select again",
				}
			}
			continue
		}
		return Selection{Target: target, Found: true, Available: true, Reason: target.Reason, NextAction: target.NextAction}
	}
	return *unavailable
}

// SelectByID resolves one target by its durable identity. Labels are never a
// selector because two targets can present the same label. Duplicate IDs fail
// closed as an ambiguous inventory instead of silently choosing one owner.
func SelectByID(inventory Inventory, id string) Selection {
	id = strings.TrimSpace(id)
	if id == "" {
		return Selection{Reason: "target identity is required", NextAction: "provide a durable target identity and select again"}
	}
	var match *Target
	for index := range inventory.Targets {
		candidate := &inventory.Targets[index]
		if candidate.ID != id {
			continue
		}
		if match != nil {
			return Selection{Found: true, Reason: fmt.Sprintf("target identity %q is ambiguous", id), NextAction: "refresh the owner inventory and select by a unique target identity"}
		}
		match = candidate
	}
	if match == nil {
		return Selection{Reason: fmt.Sprintf("target %q was not found", id), NextAction: "refresh the owner inventory and select an available target"}
	}
	selection := Selection{Target: *match, Found: true, Available: match.Available, Reason: match.Reason, NextAction: match.NextAction}
	if !selection.Available {
		if selection.Reason == "" {
			selection.Reason = fmt.Sprintf("target %q is unavailable", id)
		}
		if selection.NextAction == "" {
			selection.NextAction = "restore target availability and select again"
		}
	}
	return selection
}

// SelectByOS applies the same deterministic selector to each requested OS.
func SelectByOS(inventory Inventory, oses []string, request SelectionRequest) map[string]Selection {
	result := make(map[string]Selection, len(oses))
	for _, raw := range oses {
		os := strings.ToLower(strings.TrimSpace(raw))
		if os == "" {
			continue
		}
		request.OS = os
		if _, exists := result[os]; !exists {
			result[os] = Select(inventory, request)
		}
	}
	return result
}

func transportAllowed(kind TransportKind, allowed []TransportKind) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, candidate := range allowed {
		if kind == candidate {
			return true
		}
	}
	return false
}

func missingCapabilities(target Target, required []string) []string {
	missing := make([]string, 0, len(required))
	for _, capability := range required {
		if !target.Supports(capability) {
			missing = append(missing, strings.TrimSpace(capability))
		}
	}
	return missing
}

func UnavailableTarget(reason, missingCapability string) Target {
	return Target{
		ID: "bridge:unavailable", Label: "Bridge fleet", Platform: "desktop", DeviceKind: "desktop",
		Mode: "remote", Transport: Transport{Kind: TransportBridge, ID: "bridge", Available: false, Reason: reason},
		Available: false, Reason: reason, MissingCapability: missingCapability,
		NextAction:  "restore bridge inventory and probe again",
		Health:      TargetHealth{Status: "unavailable", Reason: reason},
		BridgeTrust: &BridgeTrust{Reason: "bridge node identity was not verified"},
	}
}
