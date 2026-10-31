package usagemodes

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"agent-manager/internal/domain"

	"github.com/vrooli/api-core/operatordecision"
)

// KindRoleDefault is the decision kind for a role-default change proposal, and
// AnswerApprove the answer that records the operator's approval.
const (
	KindRoleDefault = "role-default"
	AnswerApprove   = "approve"
)

// PolicyOp is one RFC 6902 style edit of a resource's model-policy.json.
type PolicyOp struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	From  string `json:"from_value,omitempty"`
	Value string `json:"value"`
}

// PolicyChange is the exact edit that makes a role's default the challenger.
// Agent Manager never applies it: resource files belong to the resource, and
// there is no governed apply path from this API. The operator applies it.
type PolicyChange struct {
	File string
	Ops  []PolicyOp
	// Note says what could not be derived, if anything.
	Note string
}

// Text renders the change for a decision body or a terminal.
func (c PolicyChange) Text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "File: %s\n", c.File)
	for _, op := range c.Ops {
		raw, _ := json.Marshal(op)
		b.Write(raw)
		b.WriteByte('\n')
	}
	if c.Note != "" {
		fmt.Fprintf(&b, "Note: %s\n", c.Note)
	}
	return strings.TrimRight(b.String(), "\n")
}

func pointerKey(key string) string {
	return strings.NewReplacer("~", "~0", "/", "~1").Replace(key)
}

// RoleDefaultChange derives the policy edit for moving role's default model
// from one model to another in a runner's resource model-policy.json. It reads
// the file through the same source as the usage tiers and writes nothing.
func (s *Service) RoleDefaultChange(ctx context.Context, runner, role, from, to string) PolicyChange {
	runner = resourceRunner(runner)
	change := PolicyChange{File: "resources/" + strings.TrimSuffix(runner, "-code") + "/model-policy.json"}
	if runner == "claude-code" {
		change.File = "resources/claude-code/model-policy.json"
	}
	tiers, ok := s.Tiers.(ResourceTiers)
	if !ok || tiers.Files == nil {
		change.Note = "the resource policy file is not readable from here; set roles." + role + ".model to " + to + " in " + change.File
		return change
	}
	raw, err := tiers.Files.PolicyFile(ctx, domain.RunnerType(runner))
	if err != nil {
		change.Note = "the resource policy file could not be read (" + err.Error() + "); set roles." + role + ".model to " + to + " in " + change.File
		return change
	}
	var doc struct {
		Roles   map[string]json.RawMessage `json:"roles"`
		Aliases map[string]struct {
			Canonical string `json:"canonical_model"`
		} `json:"model_aliases"`
	}
	if json.Unmarshal(raw, &doc) != nil || doc.Roles[role] == nil {
		change.Note = "role " + role + " has no entry under roles in " + change.File + ", so its default comes from the role-policy catalog; change the model that catalog maps it to instead"
		return change
	}
	var entry struct {
		Model     string `json:"model"`
		Canonical string `json:"canonical_model"`
		Models    []struct {
			Model string `json:"model"`
		} `json:"models"`
	}
	_ = json.Unmarshal(doc.Roles[role], &entry)
	base := "/roles/" + pointerKey(role)
	change.Ops = append(change.Ops, PolicyOp{Op: "replace", Path: base + "/model", From: entry.Model, Value: to})
	if alias, ok := doc.Aliases[to]; ok && alias.Canonical != "" && entry.Canonical != "" {
		change.Ops = append(change.Ops, PolicyOp{Op: "replace", Path: base + "/canonical_model", From: entry.Canonical, Value: alias.Canonical})
	}
	for i, m := range entry.Models {
		if m.Model == from || m.Model == entry.Model {
			change.Ops = append(change.Ops, PolicyOp{Op: "replace", Path: base + "/models/" + strconv.Itoa(i) + "/model", From: m.Model, Value: to})
		}
	}
	return change
}

// RoleDefaultProposal asks the operator to change one role's default.
type RoleDefaultProposal struct {
	ModeID   string
	Runner   string
	Role     string
	From, To string
	// Verdict and Evidence are what the comparison showed, in words.
	Verdict  string
	Evidence string
	Change   PolicyChange
}

// ProposeRoleDefault raises one decision to change a role's default model. It
// is an ask only: approving records the operator's approval and prints the
// exact change, and nothing is written to any resource file. One per mode.
func (n *Notifier) ProposeRoleDefault(ctx context.Context, p RoleDefaultProposal) (string, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	id := operatordecision.CorrelationID("umr", p.ModeID, p.Role, p.To)
	body := fmt.Sprintf("The comparison for %s on %s reads %q. %s\n\nApproving does not change anything by itself: Agent Manager does not edit resource files. It records your approval and keeps this exact change for you to apply:\n%s",
		p.Role, runnerTitle(usageRunnerName(p.Runner)), p.Verdict, p.Evidence, p.Change.Text())
	facts := operatordecision.Facts{
		Subject:     fmt.Sprintf("Make %s the default for %s?", p.To, p.Role),
		Question:    fmt.Sprintf("Make %s the default model for %s instead of %s? The evidence is a range, not proof.", p.To, p.Role, p.From),
		BigPicture:  body,
		Options:     []operatordecision.Option{{Key: AnswerApprove, Label: "Approve the change"}, {Key: AnswerNotNow, Label: "Not now"}},
		Recommended: AnswerNotNow, Reversible: true, Urgency: "low",
		ContextURL: n.link(ctx, "?tab=evidence"),
		Correlation: map[string]string{
			"kind": KindRoleDefault, "decision_id": id, "mode_id": p.ModeID, "role": p.Role, "from": p.From, "to": p.To,
		},
	}
	st := n.load()
	if ask := st.Asks[id]; ask != nil && ask.State == operatordecision.PendingStatePending {
		return id, nil
	}
	if _, err := n.raiser().Raise(ctx, pendingAskFor(id, subjectFor(KindRoleDefault, ""), facts.Correlation, map[string]string{"policy_change": p.Change.Text()}), facts); err != nil {
		return "", err
	}
	return id, n.save()
}

func usageRunnerName(runner string) string {
	if strings.HasPrefix(runner, "claude") {
		return "claude"
	}
	return runner
}
