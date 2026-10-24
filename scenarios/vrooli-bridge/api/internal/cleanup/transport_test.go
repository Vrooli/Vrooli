package cleanup

import (
	"errors"
	"testing"
)

func TestSelectTransport(t *testing.T) {
	tests := []struct {
		name      string
		facts     TransportFacts
		transport Transport
		reason    string
		field     string
	}{
		{name: "paired agent", facts: TransportFacts{AgentOnline: true}, transport: TransportAgent, reason: "paired agent is online"},
		{name: "approved ssh fallback", facts: TransportFacts{TargetReachable: true, SSHManagement: true, SSHScopeApproved: true}, transport: TransportSSH, reason: "paired agent is offline; approved SSH management is available"},
		{name: "missing ssh capability", facts: TransportFacts{TargetReachable: true}, field: "ssh.management"},
		{name: "ssh scope not approved", facts: TransportFacts{TargetReachable: true, SSHManagement: true}, field: "ssh.management"},
		{name: "unreachable", facts: TransportFacts{TargetReachable: false}, field: "reachability"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			selection, err := SelectTransport(test.facts)
			if test.field != "" {
				if err == nil {
					t.Fatal("expected typed blocked result")
				}
				var blocked ErrBlocked
				if !errors.As(err, &blocked) || blocked.Field != test.field {
					t.Fatalf("error = %v, want field %q", err, test.field)
				}
				return
			}
			if err != nil || selection.Transport != test.transport || selection.Reason != test.reason {
				t.Fatalf("selection=%#v err=%v", selection, err)
			}
		})
	}
}
