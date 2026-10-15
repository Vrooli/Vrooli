package teams

import (
	"encoding/json"
	"strings"
	"testing"

	"prompt-manager/internal/teamconfig"
)

func TestFinitePreparationCLISendsOnlyExactAmendment(t *testing.T) { // [REQ:PM-TEAM-PREPARATION]
	current := finiteCLIConfig()
	current.FiniteLeaderState = json.RawMessage(`{"status":"idle"}`)
	next := *current.FiniteLeader
	next.AcceptedRevision = "accepted:r4"
	next.SourceRefs = []string{"owner:new-source"}
	want := current
	want.FiniteLeader = &next
	want.PreparationAmendments = preparationCLIReceipt(current, next)
	b, _ := json.Marshal(finitePreparationInput{ExpectedRevision: "accepted:r3", Binding: &next, EvidenceRef: "owner:evidence"})
	ctx := &fakeContext{getResponse: current, response: want}
	if err := route(ctx, []string{"heartbeat-amend-effort", "new-team", "leader", "--request-file", finiteCLIRequest(t, string(b)), "--json"}); err != nil {
		t.Fatal(err)
	}
	ctx.assertMethodPath(t, "PUT", "/teams/new-team/heartbeats/leader")
	var sent map[string]json.RawMessage
	if err := json.Unmarshal(ctx.gotPayload, &sent); err != nil {
		t.Fatal(err)
	}
	if len(sent) != 1 || sent["finiteEffortTransition"] == nil {
		t.Fatal("amendment carried configuration or activation", string(ctx.gotPayload))
	}
	var transition FiniteEffortTransition
	json.Unmarshal(sent["finiteEffortTransition"], &transition)
	if transition.Operation != "amend" || transition.ExpectedRevision != "accepted:r3" || transition.Binding.AcceptedRevision != "accepted:r4" || transition.EvidenceRef != "owner:evidence" {
		t.Fatal("amendment lost exact CAS/evidence")
	}
}

func TestFinitePreparationCLIRejectsUnsafeInputAndFalseConfirmation(t *testing.T) { // [REQ:PM-TEAM-PREPARATION]
	current := finiteCLIConfig()
	current.FiniteLeaderState = json.RawMessage(`{"status":"idle"}`)
	next := *current.FiniteLeader
	next.AcceptedRevision = "accepted:r4"
	input, _ := json.Marshal(finitePreparationInput{ExpectedRevision: "accepted:r3", Binding: &next, EvidenceRef: "owner:evidence"})
	for _, bad := range []string{strings.TrimSuffix(string(input), "}") + `,"enabled":true}`, string(input) + ` {}`, strings.Repeat(" ", 16385), `{"expectedRevision":"accepted:r3"}`} {
		ctx := &fakeContext{getResponse: current, response: current}
		if err := cmdHeartbeatAmendEffort(ctx, []string{"new-team", "leader", "--request-file", finiteCLIRequest(t, bad)}); err == nil {
			t.Fatal("unsafe input accepted", bad)
		}
		if ctx.gotMethod == "PUT" {
			t.Fatal("invalid input wrote")
		}
	}
	for _, mutation := range []string{"enabled", "team-enabled", "wrong-revision", "wrong-profile"} {
		t.Run(mutation, func(t *testing.T) {
			want := current
			want.FiniteLeader = &next
			want.PreparationAmendments = preparationCLIReceipt(current, next)
			switch mutation {
			case "enabled":
				want.Enabled = true
			case "team-enabled":
				want.TeamEnabled = true
			case "wrong-revision":
				want.FiniteLeader = current.FiniteLeader
			case "wrong-profile":
				want.ProfileKey = "different"
			}
			ctx := &fakeContext{getResponse: current, response: want}
			if err := cmdHeartbeatAmendEffort(ctx, []string{"new-team", "leader", "--request-file", finiteCLIRequest(t, string(input))}); err == nil {
				t.Fatal("false owner confirmation accepted")
			}
		})
	}
}

func preparationCLIReceipt(current HeartbeatConfig, next teamconfig.FiniteLeader) json.RawMessage {
	b, _ := json.Marshal([]any{map[string]any{"previous": current.FiniteLeader, "replacement": next, "evidenceRef": "owner:evidence", "actor": map[string]any{"subject": "creator-run"}}})
	return b
}
