package usagemodes

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/domain"

	"github.com/vrooli/api-core/operatordecision"
)

const proposalPolicy = `{"runner":"codex","model_aliases":{"gpt-6.1-terra":{"canonical_model":"codex/gpt-6.1-terra"},"gpt-6-luna":{"canonical_model":"codex/gpt-6-luna"}},
"roles":{"code.economy.delivery":{"model":"gpt-6.1-terra","canonical_model":"codex/gpt-6.1-terra","models":[{"model":"gpt-6.1-terra","effort":"medium"}]}},
"usage_tiers":{"model_map":{"gpt-6.1-terra":"gpt-6-luna"},"session_model":"gpt-6-luna"}}`

type policyFiles map[domain.RunnerType]string

func (p policyFiles) PolicyFile(_ context.Context, runner domain.RunnerType) ([]byte, error) {
	return []byte(p[runner]), nil
}

func proposal(change PolicyChange) RoleDefaultProposal {
	return RoleDefaultProposal{ModeID: "um-l", Runner: "codex", Role: "code.economy.delivery", From: "gpt-6.1-terra", To: "gpt-6-luna",
		Verdict: "cheaper looks better", Evidence: "30% cheaper per accepted outcome (range 0.55 to 0.80).", Change: change}
}

func TestRoleDefaultChangeNamesTheExactEditAndWritesNothing(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "model-policy.json")
	if err := os.WriteFile(file, []byte(proposalPolicy), 0o600); err != nil {
		t.Fatal(err)
	}
	s := newService(t)
	s.Tiers = ResourceTiers{Files: policyFiles{domain.RunnerType("codex"): proposalPolicy}}
	change := s.RoleDefaultChange(context.Background(), "codex", "code.economy.delivery", "gpt-6.1-terra", "gpt-6-luna")
	want := []string{
		`/roles/code.economy.delivery/model`, `/roles/code.economy.delivery/canonical_model`, `/roles/code.economy.delivery/models/0/model`,
	}
	if change.File != "resources/codex/model-policy.json" || len(change.Ops) != len(want) {
		t.Fatalf("change = %+v", change)
	}
	for i, op := range change.Ops {
		if op.Path != want[i] || op.Op != "replace" || !strings.Contains(op.Value, "gpt-6-luna") {
			t.Fatalf("op %d = %+v", i, op)
		}
	}
	if raw, _ := os.ReadFile(file); string(raw) != proposalPolicy {
		t.Fatal("proposing a change must not touch the policy file")
	}
	missing := s.RoleDefaultChange(context.Background(), "codex", "code.unknown", "a", "b")
	if len(missing.Ops) != 0 || !strings.Contains(missing.Note, "role-policy catalog") {
		t.Fatalf("a role with no resource entry must say where its default comes from: %+v", missing)
	}
}

func TestRoleDefaultProposalNeedsAVerifiedHumanAndOnlyRecordsApproval(t *testing.T) {
	n, rec, _ := newNotifier(t)
	change := PolicyChange{File: "resources/codex/model-policy.json", Ops: []PolicyOp{{Op: "replace", Path: "/roles/code.economy.delivery/model", From: "gpt-6.1-terra", Value: "gpt-6-luna"}}}
	id, err := n.ProposeRoleDefault(context.Background(), proposal(change))
	if err != nil {
		t.Fatal(err)
	}
	if again, _ := n.ProposeRoleDefault(context.Background(), proposal(change)); again != id || len(rec.decisions()) != 1 {
		t.Fatalf("one decision per proposal: %s %s %d", id, again, len(rec.decisions()))
	}
	payload := rec.decisions()[0].Payload
	if payload["default_answer"] != nil || payload["recommended"] != AnswerNotNow {
		t.Fatalf("nothing may be approved by default: %+v", payload)
	}
	for name, res := range map[string]operatordecision.DecisionResolution{
		"no identity":   answer(rec.decisions()[0], AnswerApprove, "", "answered", false),
		"by default":    answer(rec.decisions()[0], AnswerApprove, "operator@example.com", "answered", true),
		"agent default": answer(rec.decisions()[0], AnswerApprove, "default", "answered", false),
	} {
		n2, rec2, _ := newNotifier(t)
		if _, err := n2.ProposeRoleDefault(context.Background(), proposal(change)); err != nil {
			t.Fatal(err)
		}
		res.Correlation = answer(rec2.decisions()[0], AnswerApprove, "x", "answered", false).Correlation
		if _, err := n2.Apply(context.Background(), res); err != nil {
			t.Fatal(err)
		}
		state, note, _ := n2.AskState(id)
		if state != operatordecision.PendingStateRejected || !strings.Contains(note, "no verified human") {
			t.Errorf("%s: state %s note %q", name, state, note)
		}
	}
	if _, err := n.Apply(context.Background(), answer(rec.decisions()[0], AnswerApprove, "operator@example.com", "answered", false)); err != nil {
		t.Fatal(err)
	}
	state, note, _ := n.AskState(id)
	if state != operatordecision.PendingStateApproved || !strings.Contains(note, "Not applied") || !strings.Contains(note, "/roles/code.economy.delivery/model") {
		t.Fatalf("state %s note %q: approval must record the verified human, say nothing was applied and keep the exact change", state, note)
	}
}
