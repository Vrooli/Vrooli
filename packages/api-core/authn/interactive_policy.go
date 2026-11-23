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
	// Session reclamation is one exact scope plus the digest of its current
	// owner/process-incarnation proof. A grant cannot name a wildcard target.
	{"agent-session-reaping", Policy{[]string{"agent-session:reap"}, []string{"agent-session.reap"}, []string{"agent-session"}}},
	// Research commission effects stay bounded to named commission targets;
	// this never authorizes answers, verdicts or grant issuance.
	{"research-management", Policy{[]string{"research-desk:read", "research-desk:write"}, []string{"research.launch", "research.control"}, []string{"research-desk"}}},
	// research-decisions lets an agent record an answer to a needs-you item
	// for the operator who approved it (research-desk:decision/<owner>/<item>,
	// or research-desk:decision/* for every item). The desk still refuses
	// items that are not answerable (secrets, ended commissions) and marks the
	// entry as delegated. It never covers verdicts, steering or grant issuance.
	{"research-decisions", Policy{[]string{"research-desk:read", "research-desk:write"}, []string{"research.answer"}, []string{"research-desk"}}},
	{"deployment-operations", Policy{[]string{"deployment-manager:read", "deployment-manager:write", "deployment-manager:destructive", "scenario-to-cloud:read", "scenario-to-cloud:write", "scenario-to-cloud:destructive"}, []string{"deployment.manage"}, []string{"deployment-manager", "scenario-to-cloud"}}},
	{"nutrition-management", Policy{[]string{"nutrition-planner:read", "nutrition-planner:write"}, []string{"nutrition.manage"}, []string{"nutrition-planner"}}},
	// Treasury still evaluates its signed mandate, exact amount, recipient,
	// funded headroom and kill switch. A grant must name the exact authorized
	// spend target; it cannot change policy or human decisions.
	{"treasury-spending", Policy{[]string{"treasury:write"}, []string{"treasury.spend"}, []string{"treasury"}}},
	// Agents need a separate read rule for the derived wallet while their
	// spend rule remains bound to one authorization ID.
	{"treasury-wallet-reading", Policy{[]string{"treasury:read"}, []string{"treasury.read"}, []string{"treasury"}}},
	// root-state-transition is a standing, operator-approved authority for a
	// recoverable root control-plane transition: the candidate image replaces
	// the serving one and the prior image stays installed for rollback. It
	// never covers irreversible or externally visible effects, which remain
	// operator decisions, and the runtime still refuses a transition whose
	// compensating controls (state snapshot, retention, receipt) are absent.
	{"root-state-transition", Policy{[]string{"root-state:transition"}, []string{"root-state.transition"}, []string{"root-state"}}},
	// agent-guard-override lets one agent session past one agent guard (money,
	// credentials, outside messages, hidden commands) that stopped a legitimate
	// action. The target is the guard's check ID, e.g. agent-guard:secret; the
	// policy runtime asks the credential broker for it when the guard stops.
	{"agent-guard-override", Policy{[]string{"agent-guard:override"}, []string{"agent-guard.override"}, []string{"agent-guard"}}},
}

// ValidateInteractiveTarget enforces policy-specific target shapes at the
// authorization request boundary. Most policies rely on the reviewed domain
// list; exact agent-session reclamation also binds a scope to one proof digest.
func ValidateInteractiveTarget(policyName, target string) error {
	policy, ok := InteractivePolicy(policyName)
	if !ok {
		return errors.New("unknown reviewed policy")
	}
	domain, object, ok := strings.Cut(target, ":")
	if !ok || object == "" || !slices.Contains(policy.Domains, domain) {
		return errors.New("target is outside the reviewed policy domain")
	}
	if policyName != "agent-session-reaping" {
		return nil
	}
	parts := strings.Split(object, "/")
	if len(parts) != 4 || parts[0] != "reap" || parts[2] != "proof" || !validExactScope(parts[1]) || !validProofDigest(parts[3]) {
		return errors.New("agent-session-reaping requires agent-session:reap/<exact-scope>/proof/<sha256>")
	}
	return nil
}

func validExactScope(scope string) bool {
	if !strings.HasPrefix(scope, "vrooli-agent-") || !strings.HasSuffix(scope, ".scope") {
		return false
	}
	body := strings.TrimSuffix(strings.TrimPrefix(scope, "vrooli-agent-"), ".scope")
	if body == "" {
		return false
	}
	for _, r := range body {
		if !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') && r != '_' && r != '-' && r != '.' {
			return false
		}
	}
	return true
}

func validProofDigest(digest string) bool {
	if len(digest) != 64 {
		return false
	}
	for _, r := range digest {
		if !(r >= '0' && r <= '9') && !(r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
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
//
// The agent session is written as a literal when this process knows it. The
// operator copies this command into their own shell, where VROOLI_AGENT_SESSION
// is not exported, so an unexpanded variable would produce a command that
// cannot run.
func AuthorizationRequestCommand(policy, target string) string {
	selector := `"$VROOLI_AGENT_SESSION"`
	if session := strings.TrimSpace(os.Getenv("VROOLI_AGENT_SESSION")); session != "" {
		selector = shellWord(session)
	}
	command := `vrooli agent authorize ` + selector + ` --policy ` + shellWord(policy) + " --target " + shellWord(target)
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
