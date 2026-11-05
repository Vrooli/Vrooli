package agentharness

import (
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func TestDecideMatrix(t *testing.T) {
	mutate := CommandSpec{Name: "permissions deny", Mutating: true}
	read := CommandSpec{Name: "permissions list", Mutating: false}
	override := CallerOverrideFlags{AuthorizedByUser: true}
	noOverride := CallerOverrideFlags{}

	cases := []struct {
		name   string
		kind   cliutil.CallerKind
		cmd    CommandSpec
		flags  CallerOverrideFlags
		policy Policy
		want   GateDecision
	}{
		{"read verbs always allowed (human)", cliutil.CallerKindHuman, read, noOverride, Policy{AgentAccess: AgentAccessDeny}, DecisionAllow},
		{"read verbs always allowed (agent)", cliutil.CallerKindExternalAgent, read, noOverride, Policy{AgentAccess: AgentAccessDeny}, DecisionAllow},
		{"human + mutate + deny", cliutil.CallerKindHuman, mutate, noOverride, Policy{AgentAccess: AgentAccessDeny}, DecisionAllow},
		{"unknown + mutate + deny", cliutil.CallerKindUnknown, mutate, noOverride, Policy{AgentAccess: AgentAccessDeny}, DecisionAllow},
		{"external agent + mutate + allow", cliutil.CallerKindExternalAgent, mutate, noOverride, Policy{AgentAccess: AgentAccessAllow}, DecisionAllow},
		{"external agent + mutate + warn", cliutil.CallerKindExternalAgent, mutate, noOverride, Policy{AgentAccess: AgentAccessWarn}, DecisionWarn},
		{"external agent + mutate + confirm + no override", cliutil.CallerKindExternalAgent, mutate, noOverride, Policy{AgentAccess: AgentAccessConfirm}, DecisionDeny},
		{"external agent + mutate + confirm + override", cliutil.CallerKindExternalAgent, mutate, override, Policy{AgentAccess: AgentAccessConfirm}, DecisionAllow},
		{"external agent + mutate + deny + no override", cliutil.CallerKindExternalAgent, mutate, noOverride, Policy{AgentAccess: AgentAccessDeny}, DecisionDeny},
		{"external agent + mutate + deny + override", cliutil.CallerKindExternalAgent, mutate, override, Policy{AgentAccess: AgentAccessDeny}, DecisionAllow},
		{"vrooli agent + mutate + deny", cliutil.CallerKindVrooliAgent, mutate, noOverride, Policy{AgentAccess: AgentAccessDeny}, DecisionDeny},
		{"override agent + mutate + deny", cliutil.CallerKindOverride, mutate, noOverride, Policy{AgentAccess: AgentAccessDeny}, DecisionDeny},
		{"unknown policy fails closed", cliutil.CallerKindExternalAgent, mutate, noOverride, Policy{AgentAccess: AgentAccess("weird")}, DecisionDeny},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := Decide(tc.kind, tc.cmd, tc.flags, tc.policy)
			if got != tc.want {
				t.Errorf("Decide(%v, %v, %v, %v) = %v, want %v", tc.kind, tc.cmd, tc.flags, tc.policy, got, tc.want)
			}
		})
	}
}

func TestRenderDenyMessageIncludesContext(t *testing.T) {
	dctx := DenyContext{ResourceLabel: "resource-claude-code", ConfigPath: "~/.claude/settings.json"}
	msg := RenderDenyMessage(dctx, CommandSpec{Name: "permissions deny"}, DefaultPolicy())
	for _, want := range []string{OverrideFlag, "permissions deny", "resource-claude-code", "~/.claude/settings.json"} {
		if !strings.Contains(msg, want) {
			t.Errorf("deny message missing %q: %s", want, msg)
		}
	}
}

func TestDefaultIsDeny(t *testing.T) {
	if DefaultPolicy().AgentAccess != AgentAccessDeny {
		t.Errorf("default must be deny for permissions verbs: %v", DefaultPolicy())
	}
}

func TestDecisionString(t *testing.T) {
	for d, want := range map[GateDecision]string{
		DecisionAllow:    "allow",
		DecisionWarn:     "warn",
		DecisionDeny:     "deny",
		GateDecision(99): "unknown",
	} {
		if got := d.String(); got != want {
			t.Errorf("Decision(%d).String() = %q, want %q", d, got, want)
		}
	}
}
