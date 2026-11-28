package measures

import (
	"testing"

	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/measures"
)

func TestRegisterProvidesOutageAggregateCommand(t *testing.T) {
	group := Register(nil)
	want := []string{"uptime", "outages", "restarts", "heal-outcomes", "critical"}
	if group.Name != "measure" || !group.NeedsAPI {
		t.Fatalf("measure group = %+v, want API-backed measure group", group)
	}
	if len(group.Subcommands) != len(want) {
		t.Fatalf("subcommand count = %d, want %d", len(group.Subcommands), len(want))
	}
	for i, name := range want {
		if group.Subcommands[i].Name != name {
			t.Fatalf("subcommand[%d] = %q, want %q", i, group.Subcommands[i].Name, name)
		}
	}
}

func TestHealOutcomeSummaryKeepsSkipsOutOfAttempts(t *testing.T) {
	got := healOutcomeSummary([]*measuresv1.HealOutcomeCount{
		{Outcome: "SUCCEEDED", Count: 31},
		{Outcome: "FAILED", Count: 19},
		{Outcome: "TIMED_OUT", Count: 0},
		{Outcome: "SKIPPED_UPDATE_FENCE", Count: 5},
		{Outcome: "SKIPPED_NOT_ENABLED", Count: 22},
	})
	want := "heal attempts: succeeded 31, failed 19, timed-out 0; skipped (not run): update-fence 5, not-enabled 22"
	if got != want {
		t.Fatalf("summary = %q, want %q", got, want)
	}
}
