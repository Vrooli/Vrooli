package machines

import (
	"testing"

	machinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/machines"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/shared"

	"github.com/vrooli/cli-core/cliapp"
)

// Reachability must state age rather than imply it: a row that omits when a
// machine last answered reads exactly like one that answered a moment ago.
func TestReachabilityAlwaysStatesAge(t *testing.T) {
	reachable := &machinesv1.Machine{
		Target:              &sharedv1.Target{Dispatchable: true},
		HeartbeatAgeSeconds: 8,
	}
	if got := reachability(reachable); got != "reachable (8s ago)" {
		t.Errorf("reachability(reachable) = %q", got)
	}
	stale := &machinesv1.Machine{
		Target:              &sharedv1.Target{},
		HeartbeatAgeSeconds: 7 * 24 * 3600,
	}
	if got := reachability(stale); got != "not responding (last 7d ago)" {
		t.Errorf("reachability(stale) = %q", got)
	}
	never := &machinesv1.Machine{Target: &sharedv1.Target{}}
	if got := reachability(never); got != "never responded" {
		t.Errorf("reachability(never) = %q, want the never-seen wording rather than an age of zero", got)
	}
}

// A wildcard reaches apps that do not exist yet, so no count can describe it.
func TestAppBreadthNeverCountsAWildcard(t *testing.T) {
	if got := appBreadth(0, true); got != "every app" {
		t.Errorf("appBreadth(wildcard) = %q", got)
	}
	if got := appBreadth(79, true); got != "every app" {
		t.Errorf("a wildcard with a count still reaches every app, got %q", got)
	}
	if got := appBreadth(1, false); got != "1 app" {
		t.Errorf("appBreadth(1) = %q", got)
	}
	if got := appBreadth(79, false); got != "79 apps" {
		t.Errorf("appBreadth(79) = %q", got)
	}
}

func TestHumanAgeChangesUnitWithMagnitude(t *testing.T) {
	for seconds, want := range map[int64]string{
		0: "just now", 8: "8s ago", 90: "1m ago", 7200: "2h ago", 7 * 24 * 3600: "7d ago",
	} {
		if got := humanAge(seconds); got != want {
			t.Errorf("humanAge(%d) = %q, want %q", seconds, got, want)
		}
	}
}

// Approving a machine without confirming what it is showing is the one thing
// this command must refuse, because the words are the only field the joining
// machine could not choose for itself.
func TestDecideRefusesApprovalWithoutConfirmationWords(t *testing.T) {
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cliapp.ArgSchema{
			Positionals: []cliapp.Positional{{Name: "request-id"}},
			Flags:       []cliapp.Flag{{Name: "approve", Bool: true}, {Name: "words"}, {Name: "preset"}},
		},
		Positionals: map[string]string{"request-id": "req-1"},
		BoolFlags:   map[string]bool{"approve": true},
	})
	if err := (&handlers{}).decide(ctx); err == nil {
		t.Fatal("approving with no confirmation words succeeded")
	}
}

func TestSplitListAcceptsSpacesAndCommas(t *testing.T) {
	for input, want := range map[string]int{
		"amber dolphin quartz":   3,
		"amber, dolphin, quartz": 3,
		"amber":                  1,
		"  ":                     0,
		"":                       0,
	} {
		if got := splitList(input); len(got) != want {
			t.Errorf("splitList(%q) = %v, want %d elements", input, got, want)
		}
	}
}

func TestMachineIDIsRequired(t *testing.T) {
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "machine-id"}}},
	})
	if _, err := machineID(ctx, "forget"); err == nil {
		t.Fatal("a missing machine id succeeded")
	}
}
