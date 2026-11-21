package clipolicy_test

import (
	"reflect"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/cli/clipolicy"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/projectcli"
)

func TestMaintenancePolicyKeepsPositionalAndFlagGrammar(t *testing.T) {
	for _, test := range []struct {
		name            string
		args            []string
		allowed, parses bool
	}{
		{"cleanup trims positional target", []string{"CLEANUP", "\u2003orphans\u00a0"}, false, true},
		{"lock cleanup trims positional target", []string{"CLEANUP", " locks "}, true, true},
		{"cleanup target case remains significant", []string{"cleanup", "ORPHANS"}, true, false},
		{"orphan action case remains significant", []string{"ORPHANS", "KILL"}, true, false},
		{"orphan action whitespace remains significant", []string{"orphans", " kill "}, true, false},
		{"lock action case remains significant", []string{"LOCKS", "CLEAN"}, true, false},
		{"lock action whitespace remains significant", []string{"locks", " clean "}, true, false},
		{"orphan dry run", []string{"ORPHANS", "kill", "--dry-run"}, true, true},
		{"dry-run flag case remains significant", []string{"cleanup", "orphans", "--DRY-RUN"}, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			var err error
			switch commandtree.NormalizeName(test.args[0]) {
			case "cleanup":
				_, err = projectcli.ParseCleanupRequest(test.args[1:])
			case "orphans":
				_, err = projectcli.ParseOrphansRequest(test.args[1:])
			case "locks":
				_, err = projectcli.ParseLocksRequest(test.args[1:])
			}
			if (err == nil) != test.parses {
				t.Fatalf("parser grammar changed: %v", err)
			}
			argv := append([]string{"vrooli"}, test.args...)
			before := append([]string(nil), argv...)
			got := clipolicy.ClassifyAgentCommand(argv, []string{cliutil.AgentSessionEnv + "=fixture"})
			if got.Allowed != test.allowed {
				t.Fatalf("policy changed argument semantics: %+v", got)
			}
			if !reflect.DeepEqual(before, argv) {
				t.Fatal("classification mutated caller argv")
			}
		})
	}
	for _, argv := range [][]string{
		{"vrooli", " ScEnArIo ", "RESTART", "stop-all", "--path", "/Tmp/Mixed Case/STOP-ALL"},
		{"vrooli", " ReSoUrCe ", "STATUS", "STOP-ALL"},
	} {
		before := append([]string(nil), argv...)
		if got := clipolicy.ClassifyAgentCommand(argv, []string{cliutil.AgentSessionEnv + "=fixture"}); !got.Allowed {
			t.Fatalf("argument treated as destructive operation: %q %+v", argv, got)
		}
		if !reflect.DeepEqual(before, argv) {
			t.Fatal("classification mutated target or path")
		}
	}
}
