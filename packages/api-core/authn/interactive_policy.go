package authn

import (
	"errors"
	"net/url"
	"os"
	"slices"
	"strings"
)

// Policy is the reviewed, finite operation set available to an interactive
// grant. It cannot include retained human decisions or arbitrary capabilities.
type Policy struct{ Scopes, Operations, Domains []string }

// interactivePolicies is the one catalogue of reviewed interactive grant
// policies. It answers two questions from the same rows: what a named policy
// covers (InteractivePolicy) and which policy an agent asks for to perform a
// refused operation (PoliciesForOperation, and so every refusal's request
// command). docs/concepts/AUTHORIZATION.md lists the same rows; a test fails
// when the two disagree.
var interactivePolicies = []struct {
	name   string
	policy Policy
}{
	{"swarm-bookkeeping", Policy{[]string{"swarm-manager:read", "swarm-manager:write", "swarm-manager:destructive"}, []string{"swarm.bookkeeping"}, []string{"swarm-manager"}}},
	// run-creation is a standing, operator-approved authority for a trusted
	// service (or an unattended agent) to create agent-manager runs within
	// declared bounds. run.create is the sole operation; the concurrency/budget
	// limits and the accountable owner live on the grant record, not here.
	{"run-creation", Policy{[]string{"agent-manager:read", "agent-manager:write"}, []string{"run.create"}, []string{"agent-manager"}}},
	{"agent-investigation", Policy{[]string{"agent-manager:read", "agent-manager:write"}, []string{"agent.investigate"}, []string{"agent-manager"}}},
	{"agent-management", Policy{[]string{"agent-manager:read", "agent-manager:write"}, []string{"agent.investigate", "agent.manage"}, []string{"agent-manager"}}},
	// run-deletion permanently deletes named agent-manager runs
	// (agent-manager:run/<id>, or a bounded prefix). It is separate from
	// agent-management because deletion is irreversible; it never covers run review.
	{"run-deletion", Policy{[]string{"agent-manager:read", "agent-manager:write"}, []string{"run.delete"}, []string{"agent-manager"}}},
	// team-preparation covers disabled, undispatched draft revisions only; no
	// start, grants or activation.
	{"team-preparation", Policy{[]string{"prompt-manager:read", "prompt-manager:write"}, []string{"team.prepare"}, []string{"prompt-manager"}}},
	// team-management covers existing owner-checked team.manage mutations only.
	// Targets still must be explicit and are stored on the approved grant;
	// team-preparation remains the separate disabled-draft policy.
	{"team-management", Policy{[]string{"prompt-manager:read", "prompt-manager:write"}, []string{"team.manage"}, []string{"prompt-manager"}}},
	// prompt-agent-creation creates one explicitly named Prompt Manager catalog
	// entity. It does not authorize updates, deletion, membership, or agent-file changes.
	{"prompt-agent-creation", Policy{[]string{"prompt-manager:read", "prompt-manager:write"}, []string{"agent.create"}, []string{"prompt-manager"}}},
	// effort-metadata covers owner-facing effort metadata only (goal home,
	// display name, revisions). Agent Manager refuses any change to authority,
	// subjects or withdrawal.
	{"effort-metadata", Policy{[]string{"agent-manager:read", "agent-manager:write"}, []string{"effort.reconcile"}, []string{"agent-manager"}}},
	// delivery-management is commission control on an existing effort only. The
	// domain preserves enrollment authority and owns clock, revision and
	// retained-run checks.
	{"delivery-management", Policy{[]string{"agent-manager:read", "agent-manager:write"}, []string{"effort.delivery"}, []string{"agent-manager"}}},
	{"deployment-operations", Policy{[]string{"deployment-manager:read", "deployment-manager:write", "deployment-manager:destructive", "scenario-to-cloud:read", "scenario-to-cloud:write", "scenario-to-cloud:destructive"}, []string{"deployment.manage"}, []string{"deployment-manager", "scenario-to-cloud"}}},
	{"nutrition-management", Policy{[]string{"nutrition-planner:read", "nutrition-planner:write", "nutrition-planner:destructive"}, []string{"nutrition.manage"}, []string{"nutrition-planner"}}},
	// Treasury still evaluates its signed mandate, exact amount, recipient,
	// funded headroom and kill switch. A grant must name the exact authorized
	// spend target; it cannot change policy or human decisions.
	{"treasury-spending", Policy{[]string{"treasury:write"}, []string{"treasury.spend"}, []string{"treasury"}}},
	// Agents need a separate read rule for the derived wallet while their
	// spend rule remains bound to one authorization ID.
	{"treasury-wallet-reading", Policy{[]string{"treasury:read"}, []string{"treasury.read"}, []string{"treasury"}}},
}

// InteractivePolicy returns the named reviewed policy. The result is a copy.
func InteractivePolicy(name string) (Policy, bool) {
	for _, entry := range interactivePolicies {
		if entry.name == name {
			return Policy{slices.Clone(entry.policy.Scopes), slices.Clone(entry.policy.Operations), slices.Clone(entry.policy.Domains)}, true
		}
	}
	return Policy{}, false
}

// InteractivePolicyNames lists every reviewed policy in catalogue order.
func InteractivePolicyNames() []string {
	names := make([]string, 0, len(interactivePolicies))
	for _, entry := range interactivePolicies {
		names = append(names, entry.name)
	}
	return names
}

// PoliciesForOperation lists the reviewed policies whose operations include the
// operation, narrowest first (fewest operations), so the first entry is the
// least authority an agent can ask for. It is empty for an operation no
// reviewed policy covers.
func PoliciesForOperation(operation string) []string {
	var names []string
	for _, entry := range interactivePolicies {
		if slices.Contains(entry.policy.Operations, operation) {
			names = append(names, entry.name)
		}
	}
	size := func(name string) int { p, _ := InteractivePolicy(name); return len(p.Operations) }
	slices.SortStableFunc(names, func(a, b string) int { return size(a) - size(b) })
	return names
}

// operatorDecisions are the operations that record a human decision. No agent
// grant covers them: the operator answers where the entry says.
var operatorDecisions = map[string]string{
	"run.review":          "Agent Manager (the run's review page) or the decision in the operator's inbox (agent_manager.run.decision_requested.v1)",
	"investigation.apply": "Agent Manager or the decision in the operator's inbox (agent_manager.investigation.decision_requested.v1)",
	"sandbox.sync":        "workspace-sandbox's review result, which is a callback the operator's review records, not an agent command",
}

// OperatorDecision reports where the operator answers an operation that records
// a human decision and so has no agent grant.
func OperatorDecision(operation string) (where string, ok bool) {
	where, ok = operatorDecisions[operation]
	return where, ok
}

// OperationRefusal is the refusal RequireOperation returns to an agent that
// holds no grant covering an operation on a target. It still satisfies
// errors.Is(err, ErrOperation); what it adds is the exact request the agent
// can run so the operator can approve that specific action.
type OperationRefusal struct {
	Operation string
	Target    string
	// Policies are the reviewed policies that grant Operation, narrowest first.
	Policies []string
	// OperatorDecisionWhere is set instead of Policies when the operation
	// records a human decision: no grant exists, and this says where it is answered.
	OperatorDecisionWhere string
}

// NewOperationRefusal builds the refusal for an operation and target.
func NewOperationRefusal(operation, target string) *OperationRefusal {
	refusal := &OperationRefusal{Operation: operation, Target: target}
	if where, ok := OperatorDecision(operation); ok {
		refusal.OperatorDecisionWhere = where
		return refusal
	}
	refusal.Policies = PoliciesForOperation(operation)
	return refusal
}

// Unwrap keeps errors.Is(err, ErrOperation) true for every existing caller.
func (e *OperationRefusal) Unwrap() error { return ErrOperation }

// Command is the ready-to-run request, or "" when no grant can be requested.
func (e *OperationRefusal) Command() string {
	if len(e.Policies) == 0 || strings.TrimSpace(e.Target) == "" {
		return ""
	}
	return AuthorizationRequestCommand(e.Policies[0], e.Target)
}

func (e *OperationRefusal) Error() string {
	base := ErrOperation.Error() + ": "
	switch {
	case e.OperatorDecisionWhere != "":
		return base + e.Operation + " records an operator decision, so no agent grant covers it; the operator answers it in " + e.OperatorDecisionWhere
	case len(e.Policies) == 0:
		return base + e.Operation + " on " + e.Target + " has no reviewed agent grant policy; it is an operator action"
	case strings.TrimSpace(e.Target) == "":
		return base + e.Operation + " needs an approved grant from the reviewed policy " + e.Policies[0] + "; request it with `vrooli agent authorize \"$VROOLI_AGENT_SESSION\" --policy " + e.Policies[0] + " --target <domain:object>` naming the object"
	}
	return base + e.Operation + " on " + e.Target + " needs an approved grant. Ask the operator to approve exactly this by running: " + e.Command()
}

// RequestAuthority returns the request command carried by a refusal anywhere in
// err's chain, or "" when err is not an OperationRefusal with a command.
func RequestAuthority(err error) string {
	var refusal *OperationRefusal
	if errors.As(err, &refusal) {
		return refusal.Command()
	}
	return ""
}

// AuthorizationRequestCommand is the exact command an agent runs to ask the
// operator for a policy on a target (`vrooli agent authorize`: the operator
// approves it in the browser, and the session uses it on its next request).
// The approval origin is included when this process is configured with it;
// otherwise `vrooli agent authorize` resolves it itself.
func AuthorizationRequestCommand(policy, target string) string {
	command := `vrooli agent authorize "$VROOLI_AGENT_SESSION" --policy ` + shellWord(policy) + " --target " + shellWord(target)
	if origin := configuredApprovalOrigin(); origin != "" {
		command += " --approval-base " + shellWord(origin)
	}
	return command + ` --reason "<why>"`
}

// configuredApprovalOrigin is the Agent Manager browser origin this process is
// configured with (AGENT_MANAGER_AUTHORIZATION_URL), or "" when it has none or
// it is not a valid origin.
func configuredApprovalOrigin() string {
	configured := strings.TrimSpace(os.Getenv("AGENT_MANAGER_AUTHORIZATION_URL"))
	u, err := url.Parse(configured)
	if configured == "" || err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" ||
		(u.Scheme != "https" && u.Scheme != "http") || (u.Path != "" && u.Path != "/") {
		return ""
	}
	return strings.TrimRight(u.String(), "/")
}

// shellWord quotes a value only when a shell would otherwise change it (for
// example the `*` in a `domain:*` target).
func shellWord(value string) string {
	for _, r := range value {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune(":/._@+=-", r)) {
			return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
		}
	}
	return value
}
