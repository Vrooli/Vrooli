package machines

import "context"

// NodeSnapshot and PresenceSnapshot are read models supplied by their owning
// Registry and Presence domains. They intentionally contain no Machine fields.
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
