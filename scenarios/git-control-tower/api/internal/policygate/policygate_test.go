package policygate

import (
	"strings"
	"testing"

	"git-control-tower/internal/config"

	"github.com/vrooli/cli-core/cliutil"
)

func TestDecide_ReadOnlyAlwaysAllow(t *testing.T) {
	for _, eff := range []string{"read", "", "info"} {
		cmd := CommandSpec{Name: "x", Effect: eff}
		for _, kind := range []cliutil.CallerKind{cliutil.CallerKindHuman, cliutil.CallerKindVrooliAgent, cliutil.CallerKindExternalAgent} {
			got := Decide(kind, cmd, CallerOverrideFlags{}, denyPolicy())
			if got != DecisionAllow {
				t.Errorf("effect=%q kind=%v: got %v, want Allow", eff, kind, got)
			}
		}
	}
}

func TestDecide_HumanAlwaysAllow(t *testing.T) {
	cmd := CommandSpec{Name: "repo commit", Effect: "write"}
	for _, kind := range []cliutil.CallerKind{cliutil.CallerKindHuman, cliutil.CallerKindUnknown} {
		for _, p := range []config.AgentAccess{config.AgentAccessAllow, config.AgentAccessWarn, config.AgentAccessConfirm, config.AgentAccessDeny} {
			policy := config.PolicyConfig{AgentAccess: p, AgentOverrideFlag: "--x"}
			if got := Decide(kind, cmd, CallerOverrideFlags{}, policy); got != DecisionAllow {
				t.Errorf("kind=%v policy=%q: got %v, want Allow", kind, p, got)
			}
		}
	}
}

func TestDecide_AgentMatrix(t *testing.T) {
	cmd := CommandSpec{Name: "WorktreeService.CreateWorktree", Effect: "write"}
	cases := []struct {
		name     string
		kind     cliutil.CallerKind
		policy   config.AgentAccess
		override bool
		want     Decision
	}{
		{"vrooli+allow", cliutil.CallerKindVrooliAgent, config.AgentAccessAllow, false, DecisionAllow},
		{"vrooli+warn", cliutil.CallerKindVrooliAgent, config.AgentAccessWarn, false, DecisionWarn},
		{"vrooli+deny", cliutil.CallerKindVrooliAgent, config.AgentAccessDeny, false, DecisionDeny},
		{"vrooli+confirm-no-override", cliutil.CallerKindVrooliAgent, config.AgentAccessConfirm, false, DecisionDeny},
		{"vrooli+confirm-with-override", cliutil.CallerKindVrooliAgent, config.AgentAccessConfirm, true, DecisionAllow},
		{"external+allow", cliutil.CallerKindExternalAgent, config.AgentAccessAllow, false, DecisionAllow},
		{"external+warn", cliutil.CallerKindExternalAgent, config.AgentAccessWarn, false, DecisionWarn},
		{"external+deny", cliutil.CallerKindExternalAgent, config.AgentAccessDeny, false, DecisionDeny},
		{"external+confirm-no-override", cliutil.CallerKindExternalAgent, config.AgentAccessConfirm, false, DecisionDeny},
		{"external+confirm-with-override", cliutil.CallerKindExternalAgent, config.AgentAccessConfirm, true, DecisionAllow},
		{"override-agent+confirm-no-override", cliutil.CallerKindOverride, config.AgentAccessConfirm, false, DecisionDeny},
		{"override-agent+confirm-with-override", cliutil.CallerKindOverride, config.AgentAccessConfirm, true, DecisionAllow},
		{"override-agent+deny", cliutil.CallerKindOverride, config.AgentAccessDeny, false, DecisionDeny},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			policy := config.PolicyConfig{AgentAccess: tc.policy, AgentOverrideFlag: "--ok"}
			got := Decide(tc.kind, cmd, CallerOverrideFlags{AuthorizedByUser: tc.override}, policy)
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDecide_DestructiveTreatedAsMutating(t *testing.T) {
	cmd := CommandSpec{Name: "worktree remove", Effect: "destructive"}
	policy := config.PolicyConfig{AgentAccess: config.AgentAccessConfirm, AgentOverrideFlag: "--ok"}
	if got := Decide(cliutil.CallerKindExternalAgent, cmd, CallerOverrideFlags{}, policy); got != DecisionDeny {
		t.Errorf("destructive+confirm w/o override: got %v, want Deny", got)
	}
}

func TestDecide_UnknownPolicyFailsClosed(t *testing.T) {
	cmd := CommandSpec{Name: "x", Effect: "write"}
	policy := config.PolicyConfig{AgentAccess: config.AgentAccess("nonsense"), AgentOverrideFlag: "--ok"}
	if got := Decide(cliutil.CallerKindVrooliAgent, cmd, CallerOverrideFlags{AuthorizedByUser: true}, policy); got != DecisionDeny {
		t.Errorf("unknown policy: got %v, want Deny (fail closed)", got)
	}
}

func TestRenderDenyMessage_SubstitutesPlaceholders(t *testing.T) {
	cmd := CommandSpec{Name: "repo commit", Effect: "write"}
	policy := config.PolicyConfig{
		AgentAccess:              config.AgentAccessConfirm,
		AgentOverrideFlag:        "--ok-i-meant-it",
		AgentDenyMessageTemplate: "cmd={command} policy={policy} flag={override_flag}",
	}
	got := RenderDenyMessage(cmd, policy)
	want := "cmd=repo commit policy=confirm flag=--ok-i-meant-it"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderDenyMessage_DefaultIncludesContext(t *testing.T) {
	cmd := CommandSpec{Name: "repo commit", Effect: "write"}
	policy := config.PolicyConfig{
		AgentAccess:       config.AgentAccessConfirm,
		AgentOverrideFlag: "--ok-i-meant-it",
	}
	got := RenderDenyMessage(cmd, policy)
	if !strings.Contains(got, "repo commit") {
		t.Errorf("default template should mention command name; got %q", got)
	}
	if !strings.Contains(got, "--ok-i-meant-it") {
		t.Errorf("default template should mention override flag; got %q", got)
	}
	if !strings.Contains(got, "confirm") {
		t.Errorf("default template should mention policy; got %q", got)
	}
}

func TestDecisionString(t *testing.T) {
	cases := map[Decision]string{
		DecisionAllow: "allow",
		DecisionWarn:  "warn",
		DecisionDeny:  "deny",
	}
	for d, s := range cases {
		if d.String() != s {
			t.Errorf("Decision(%d).String() = %q, want %q", d, d.String(), s)
		}
	}
}

func denyPolicy() config.PolicyConfig {
	return config.PolicyConfig{AgentAccess: config.AgentAccessDeny, AgentOverrideFlag: "--ok"}
}
