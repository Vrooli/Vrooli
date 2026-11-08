package stepback

import (
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
)

var t0 = time.Date(2026, 11, 6, 20, 0, 0, 0, time.UTC)

func at(minutes int) time.Time { return t0.Add(time.Duration(minutes) * time.Minute) }

func turn(start, end int, ended bool) Turn {
	return Turn{ID: "turn", Start: at(start), End: at(end), Ended: ended}
}

func compaction(minute int, goal string) Compaction {
	return Compaction{At: at(minute), GoalObjective: goal}
}

func settled(minutes ...int) []Descendant {
	out := make([]Descendant, 0, len(minutes))
	for _, m := range minutes {
		out = append(out, Descendant{Duration: time.Duration(m) * time.Minute})
	}
	return out
}

func ids(in Input) string { return IDs(Assess(in)) }

func TestObjectiveLostNeedsAnOmittedCompactionFollowedByWork(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   Input
		want string
	}{
		{"omitted then 40 min of the same turn", Input{Transcript: Transcript{
			Turns: []Turn{turn(0, 70, true)}, Compactions: []Compaction{compaction(30, domain.CompactionGoalOmitted)},
		}}, ObjectiveLost},
		{"open turn still working 45 min later", Input{Transcript: Transcript{
			Turns: []Turn{turn(0, 75, false)}, Compactions: []Compaction{compaction(30, domain.CompactionGoalOmitted)},
		}}, ObjectiveLost},
		{"turn ended soon after the omission", Input{Transcript: Transcript{
			Turns: []Turn{turn(0, 50, true), turn(55, 200, true)}, Compactions: []Compaction{compaction(30, domain.CompactionGoalOmitted)},
		}}, ""},
		{"objective retained", Input{Transcript: Transcript{
			Turns: []Turn{turn(0, 200, true)}, Compactions: []Compaction{compaction(30, domain.CompactionGoalRetained)},
		}}, ""},
		{"retention unknown", Input{Transcript: Transcript{
			Turns: []Turn{turn(0, 200, true)}, Compactions: []Compaction{compaction(30, "")},
		}}, ""},
		{"no turn evidence", Input{Transcript: Transcript{
			Compactions: []Compaction{compaction(30, domain.CompactionGoalOmitted)},
		}}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := ids(tc.in); got != tc.want {
				t.Fatalf("triggers = %q, want %q", got, tc.want)
			}
		})
	}
}

// The span starts at the first omission in a turn, whatever order the
// compactions arrive in.
func TestObjectiveLostMeasuresFromTheFirstOmissionInTheTurn(t *testing.T) {
	in := Input{Transcript: Transcript{
		Turns:       []Turn{turn(0, 100, true)},
		Compactions: []Compaction{compaction(90, domain.CompactionGoalOmitted), compaction(20, domain.CompactionGoalOmitted)},
	}}
	triggers := Assess(in)
	if len(triggers) != 1 || !strings.Contains(triggers[0].Detail, "worst 80 min") {
		t.Fatalf("triggers = %+v, want one objective-lost measured from minute 20", triggers)
	}
}

func TestUnitPerTurnNeedsALongTurnAcrossTwoCompactions(t *testing.T) {
	long := turn(0, 200, false)
	for _, tc := range []struct {
		name string
		in   Input
		want string
	}{
		{"3h20 turn, two compactions", Input{Transcript: Transcript{
			Turns:       []Turn{long},
			Compactions: []Compaction{compaction(60, ""), compaction(150, "")},
		}}, UnitPerTurn},
		{"one compaction", Input{Transcript: Transcript{Turns: []Turn{long}, Compactions: []Compaction{compaction(60, "")}}}, ""},
		{"compactions in other turns", Input{Transcript: Transcript{
			Turns:       []Turn{long, turn(300, 320, true)},
			Compactions: []Compaction{compaction(305, ""), compaction(310, "")},
		}}, ""},
		{"short turns", Input{Transcript: Transcript{
			Turns:       []Turn{turn(0, 170, true)},
			Compactions: []Compaction{compaction(60, ""), compaction(150, "")},
		}}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := ids(tc.in); got != tc.want {
				t.Fatalf("triggers = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestShortWorkersCountsOnlySettledDescendants(t *testing.T) {
	for _, tc := range []struct {
		name        string
		descendants []Descendant
		want        string
	}{
		{"half of four ran under 6 min", settled(2, 5, 30, 240), ShortWorkers},
		{"one of four", settled(2, 30, 60, 240), ""},
		{"three is too few to judge", settled(1, 2, 3), ""},
		{"live workers are not short yet", append(settled(1, 2, 120), Descendant{Live: true}, Descendant{Live: true}), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := ids(Input{Descendants: tc.descendants}); got != tc.want {
				t.Fatalf("triggers = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestInheritedContextNeedsMostSpawnsToForkHistory(t *testing.T) {
	for _, tc := range []struct {
		spawns, inherited int
		want              string
	}{
		{4, 2, InheritedContext},
		{10, 4, ""},
		{3, 3, ""},
	} {
		in := Input{Transcript: Transcript{Spawns: tc.spawns, SpawnsWithHistory: tc.inherited}}
		if got := ids(in); got != tc.want {
			t.Fatalf("%d/%d spawns: triggers = %q, want %q", tc.inherited, tc.spawns, got, tc.want)
		}
	}
}

func TestTurnStartReadsTheUUIDv7TurnID(t *testing.T) {
	first := time.Date(2026, 11, 6, 20, 4, 14, 0, time.UTC)
	got := TurnStart("01a1b275-d1d6-7960-9492-31d87173da33", first)
	if want := time.Date(2026, 11, 6, 20, 3, 48, 310_000_000, time.UTC); !got.Equal(want) {
		t.Fatalf("TurnStart(v7) = %v, want %v", got, want)
	}
	if got := TurnStart("a682d12b-5f7b-57db-8d1d-336baf8928cb", first); !got.Equal(first) {
		t.Fatalf("TurnStart(non-v7) = %v, want the first event %v", got, first)
	}
	if got := TurnStart("not-a-uuid", first); !got.Equal(first) {
		t.Fatalf("TurnStart(garbage) = %v, want the first event", got)
	}
}

func TestStanceNotRetainedCountsCompactionsThatKeptNoOperatorMessage(t *testing.T) {
	kept := func(n int) *int { return &n }
	in := Input{Transcript: Transcript{Compactions: []Compaction{
		{At: at(10), OperatorMessagesKept: kept(1)},
		{At: at(20), OperatorMessagesKept: kept(0)},
		{At: at(30)}, // unknown: older record or a native sub-agent
	}}}
	triggers := Assess(in)
	if len(triggers) != 1 || triggers[0].ID != StanceNotRetained || !strings.Contains(triggers[0].Detail, "1 compaction(s)") {
		t.Fatalf("triggers = %+v, want one stance-not-retained for the one compaction that kept nothing", triggers)
	}
	in.Transcript.Compactions[1].OperatorMessagesKept = kept(2)
	if got := ids(in); got != "" {
		t.Fatalf("triggers = %q, want none while every known compaction kept the stance", got)
	}
}
