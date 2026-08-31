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
	ID                string           `json:"id"`
	Ramp              string           `json:"ramp,omitempty"`
	Label             string           `json:"label"`
	Platform          string           `json:"platform"`
	OS                string           `json:"os"`
	Architecture      string           `json:"architecture"`
	DeviceKind        string           `json:"device_kind"`
	Revision          string           `json:"revision,omitempty"`
	LastSeenAt        time.Time        `json:"last_seen_at,omitempty"`
	SurvivesRestart   bool             `json:"survives_restart"`
	Transport         Transport        `json:"transport"`
	NodeID            string           `json:"node_id,omitempty"`
	Mode              string           `json:"mode,omitempty"`
	Capabilities      []string         `json:"capabilities,omitempty"`
	Scopes            []string         `json:"scopes,omitempty"`
	Available         bool             `json:"available"`
	Reason            string           `json:"reason,omitempty"`
	MissingCapability string           `json:"missing_capability,omitempty"`
	NextAction        string           `json:"next_action,omitempty"`
	Health            TargetHealth     `json:"health"`
	BridgeTrust       *BridgeTrust     `json:"bridge_trust,omitempty"`
	Revoked           bool             `json:"revoked,omitempty"`
	Readiness         []ReadinessCheck `json:"readiness,omitempty"`
}

func (t Target) Validate() error {
	if strings.TrimSpace(t.ID) == "" {
		return fmt.Errorf("target id is required")
	}
	if strings.TrimSpace(t.Platform) == "" {
		return fmt.Errorf("target %q platform is required", t.ID)
	}
	if t.Transport.Kind != TransportLocal && t.Transport.Kind != TransportBridge {
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

// CanHostSession is the single admission predicate shared by target
// catalogs, launch handlers, and readiness projections. It intentionally
// reports an operator-facing reason instead of allowing a later transport
// handshake to be the first place an unsupported target is discovered.
func CanHostSession(t Target) (bool, string) {
	if t.Transport.Kind == TransportLocal {
		return true, ""
	}
	if t.Transport.Kind != TransportBridge {
		return false, "this target does not provide a supported session transport"
	}
	if t.DeviceKind != "bridge-node" {
		return false, "this registered target does not host Bridge agent sessions"
	}
	if t.BridgeTrust == nil || !t.BridgeTrust.Registered {
		return false, "this node is not registered with Bridge"
	}
	if !t.BridgeTrust.Online || !t.Transport.Available {
		return false, "this node is offline or its Bridge channel is unavailable"
	}
	transportScope, ok := scopecatalog.TransportScope("interactive-session:write")
	if !ok || !scopecatalog.Resolve(t.Scopes, transportScope) {
		return false, "this node lacks the required interactive-session transport grant"
	}
	return true, ""
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
	for index, target := range i.Targets {
		if err := target.Validate(); err != nil {
			return fmt.Errorf("target %d: %w", index, err)
		}
	}
	return nil
}

// SelectionRequest describes the shared target constraints used by both
// bridge gates and delivery validation matrices.
type SelectionRequest struct {
	OS                   string          `json:"os"`
	RequiredCapabilities []string        `json:"required_capabilities,omitempty"`
	TransportKinds       []TransportKind `json:"transport_kinds,omitempty"`
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
