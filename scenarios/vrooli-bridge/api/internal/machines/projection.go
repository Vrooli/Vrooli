package machines

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/vrooli/vrooli/packages/proto/privilegedops"
)

// NodeSnapshot and PresenceSnapshot are read models supplied by their owning
// Registry and Presence domains. They intentionally contain no Machine fields.
// Capabilities are node observations only. They never grant an operation: the
// separately stored ApprovedScopes are the authorization ceiling. Keeping both
// fields in the projection prevents a newly reported capability from silently
// becoming permission to use it.
type NodeSnapshot struct {
	ID             string
	Name           string
	Capabilities   []string
	ApprovedScopes []string
	// ConfigurationState and ConfigurationUnmet are the outcome of the last
	// configuration apply Bridge recorded for this node. They explain an
	// unapplied profile; they never count as applying it.
	ConfigurationState string
	ConfigurationUnmet []string
}

type PresenceSnapshot struct {
	Connected bool
}

// ErrNodeMissing is returned by a NodeReader when the node no longer exists.
var ErrNodeMissing = errors.New("node no longer exists")

type NodeReader interface {
	GetNode(context.Context, string) (NodeSnapshot, error)
}

type PresenceReader interface {
	GetPresence(context.Context, string) (PresenceSnapshot, error)
}

type Projection struct {
	Machine  Machine
	Node     NodeSnapshot
	Presence PresenceSnapshot
	HasNode  bool
}

// Readiness is a derived explanation, never a persisted Machine status. It
// names the missing owned facts so an operator can distinguish a disconnected
// node from an unreviewed host key or an unapproved suggested capability.
type Readiness struct {
	Ready   bool
	Reasons []string
}

// DriftItem is a computed difference between desired machine intent and the
// last applied profile/current node facts. It is deliberately structured so
// callers can render or act on each difference without parsing prose.
type DriftItem struct {
	Kind   string
	Name   string
	Reason string
}

// ComputeDrift compares durable desired intent with the applied profile and
// live node observations. A missing applied record is itself drift; an absent
// node is reported separately so a disconnected machine is never mistaken
// for a configured one.
func ComputeDrift(machine Machine, policy PolicySnapshot, projection Projection) []DriftItem {
	items := make([]DriftItem, 0)
	if machine.AppliedProfileID == "" || machine.AppliedProfileVersion == "" {
		if !connectionProfileObserved(policy, projection) {
			items = append(items, DriftItem{Kind: "profile", Name: policy.ProfileID, Reason: unappliedProfileReason(projection)})
		}
	} else if machine.AppliedProfileID != policy.ProfileID || machine.AppliedProfileVersion != policy.ProfileVersion {
		items = append(items, DriftItem{Kind: "profile", Name: policy.ProfileID, Reason: "desired profile differs from the applied profile"})
	}
	desired := machine.DesiredSelectionJSON
	if desired == "" {
		desired = policy.SelectionJSON
	}
	if machine.AppliedSelectionJSON != "" && desired != "" {
		items = append(items, selectionDrift(desired, machine.AppliedSelectionJSON)...)
	}
	if !projection.HasNode {
		return items
	}
	have := make(map[string]struct{}, len(projection.Node.Capabilities))
	for _, capability := range projection.Node.Capabilities {
		have[capability] = struct{}{}
	}
	for _, capability := range policy.RequiredCapabilities {
		if _, ok := have[capability]; !ok {
			items = append(items, DriftItem{Kind: "capability", Name: capability, Reason: "required capability is not reported by the node"})
		}
	}
	return items
}

// connectionProfileObserved reports whether a connection-only profile is
// visibly in effect even though no apply was recorded. Such a profile asks for
// nothing but a connected Bridge agent and capabilities Bridge can observe, so
// a connected node that has them conforms; calling that drift sent operators
// to re-apply a machine that was already what its profile asks for. Profiles
// that place other scenarios still need a recorded apply, because Bridge
// cannot observe those scenarios from here.
func connectionProfileObserved(policy PolicySnapshot, projection Projection) bool {
	if !projection.HasNode || !projection.Presence.Connected || policy.ProfileID == "" {
		return false
	}
	for _, scenario := range policy.Scenarios {
		if scenario != "vrooli-bridge" {
			return false
		}
	}
	return len(missingRequirements("", policy.RequiredCapabilities, projection.Node.Capabilities)) == 0
}

// unappliedProfileReason names why a profile is unapplied when Bridge knows.
func unappliedProfileReason(projection Projection) string {
	const base = "profile has not been applied"
	state := strings.TrimSpace(projection.Node.ConfigurationState)
	if !projection.HasNode || state == "" {
		return base
	}
	reason := base + "; the last configuration apply ended " + state
	if unmet := projection.Node.ConfigurationUnmet; len(unmet) > 0 {
		shown := unmet
		if len(shown) > 3 {
			shown = shown[:3]
		}
		reason += fmt.Sprintf(" with %d unmet item(s): %s", len(unmet), strings.Join(shown, ", "))
		if len(unmet) > len(shown) {
			reason += ", …"
		}
	}
	return reason
}

// WithControlPlaneCapabilities adds capabilities Bridge establishes itself
// rather than hears from the node. SSH management is Bridge holding a verified
// host key, a trusted connection, its own client key, and a login for the
// machine; a node cannot report that, so requiring the node to report it made
// the drift permanent on every correctly onboarded machine.
func WithControlPlaneCapabilities(projection Projection, trust TrustRecord) Projection {
	if !projection.HasNode || !trust.SSHManagementEstablished() {
		return projection
	}
	for _, capability := range projection.Node.Capabilities {
		if capability == privilegedops.CapabilitySSHManagement {
			return projection
		}
	}
	projection.Node.Capabilities = append(append([]string(nil), projection.Node.Capabilities...), privilegedops.CapabilitySSHManagement)
	return projection
}

func selectionDrift(desiredJSON, appliedJSON string) []DriftItem {
	var desired, applied map[string]json.RawMessage
	if json.Unmarshal([]byte(desiredJSON), &desired) != nil || json.Unmarshal([]byte(appliedJSON), &applied) != nil {
		return []DriftItem{{Kind: "selection", Name: "document", Reason: "desired and applied selection documents are not both valid"}}
	}
	keys := make(map[string]struct{}, len(desired)+len(applied))
	for key := range desired {
		keys[key] = struct{}{}
	}
	for key := range applied {
		keys[key] = struct{}{}
	}
	items := make([]DriftItem, 0)
	for key := range keys {
		var left, right any
		if json.Unmarshal(desired[key], &left) != nil || json.Unmarshal(applied[key], &right) != nil || !reflect.DeepEqual(left, right) {
			items = append(items, DriftItem{Kind: "selection", Name: key, Reason: "desired selection field differs from applied selection field"})
		}
	}
	return items
}

// EvaluateReadiness joins independent Machine, trust, policy, Registry, and
// Presence facts. Suggested scopes remain suggestions: requiring their
// approval here never grants authorization, it only explains why a selected
// profile is not yet ready for its proposed use.
func EvaluateReadiness(machine Machine, trust TrustRecord, policy PolicySnapshot, projection Projection) Readiness {
	reasons := baseReadinessReasons(machine, trust)
	reasons = append(reasons, nodeReadinessReasons(policy, projection)...)
	return Readiness{Ready: len(reasons) == 0, Reasons: reasons}
}

func baseReadinessReasons(machine Machine, trust TrustRecord) []string {
	reasons := make([]string, 0, 2)
	if machine.Lifecycle != LifecycleActive {
		reasons = append(reasons, "machine_not_active")
	}
	if trust.HostKeyState != HostKeyVerified {
		reasons = append(reasons, "host_key_not_verified")
	}
	return reasons
}

func nodeReadinessReasons(policy PolicySnapshot, projection Projection) []string {
	if !projection.HasNode {
		return []string{"no_current_node"}
	}
	reasons := missingRequirements("missing_capability:", policy.RequiredCapabilities, projection.Node.Capabilities)
	reasons = append(reasons, missingRequirements("scope_not_approved:", policy.SuggestedScopes, projection.Node.ApprovedScopes)...)
	if !projection.Presence.Connected {
		reasons = append([]string{"node_offline"}, reasons...)
	}
	return reasons
}

func missingRequirements(prefix string, required, actual []string) []string {
	have := make(map[string]bool, len(actual))
	for _, value := range actual {
		have[value] = true
	}
	missing := make([]string, 0)
	for _, value := range required {
		if !have[value] {
			missing = append(missing, prefix+value)
		}
	}
	return missing
}

// Compose joins live views at read time. No Registry or Presence field is
// copied into Machine storage, so a stale connection cannot become durable
// enrollment truth.
func Compose(ctx context.Context, machine Machine, nodes NodeReader, presence PresenceReader) (Projection, error) {
	projection := Projection{Machine: machine}
	for _, lineage := range machine.Lineage {
		if !lineage.Current {
			continue
		}
		node, err := nodes.GetNode(ctx, lineage.NodeID)
		if errors.Is(err, ErrNodeMissing) {
			// The lineage still names a node the registry has deleted. That is
			// a machine with no current node, which drift and readiness already
			// explain; failing the whole read made the machine unreadable.
			return projection, nil
		}
		if err != nil {
			return Projection{}, err
		}
		live, err := presence.GetPresence(ctx, lineage.NodeID)
		if err != nil {
			return Projection{}, err
		}
		projection.Node, projection.Presence, projection.HasNode = node, live, true
		return projection, nil
	}
	return projection, nil
}
