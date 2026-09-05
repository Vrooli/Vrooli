package machines

import (
	"context"
	"encoding/json"
	"reflect"
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
}

type PresenceSnapshot struct {
	Connected bool
}

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
		items = append(items, DriftItem{Kind: "profile", Name: policy.ProfileID, Reason: "profile has not been applied"})
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
