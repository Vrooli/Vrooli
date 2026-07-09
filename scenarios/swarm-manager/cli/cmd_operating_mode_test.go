package main

import (
	"reflect"
	"testing"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

func TestHumanizeOperatingModeEnum(t *testing.T) {
	cases := map[string]string{
		"plan-manager-plan":   "Plan-manager plan",
		"plan-ref":            "Plan reference",
		"initiative":          "Initiative",
		"existing_item_flow":  "Existing item flow",
		"single_phase_run":    "Single phase run",
		"sequential_handoff":  "Sequential handoff",
		"operator_gated_loop": "Operator-gated loop",
		"unknown_value":       "unknown_value",
		"":                    "",
	}
	for in, want := range cases {
		if got := humanizeOperatingModeEnum(in); got != want {
			t.Errorf("humanizeOperatingModeEnum(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestContractChips(t *testing.T) {
	// all set, in declared order: structured, verdict, handoff, progress.
	all := &apipb.OperatingModePhaseOutputContractSummary{
		RequiresStructuredResult: true,
		RequiresProgress:         true,
		RequiresVerdict:          true,
		RequiresHandoff:          true,
	}
	if got := contractChips(all); !reflect.DeepEqual(got, []string{"structured", "verdict", "handoff", "progress"}) {
		t.Errorf("all chips = %v", got)
	}

	// none set -> empty (non-nil) slice.
	none := contractChips(&apipb.OperatingModePhaseOutputContractSummary{})
	if len(none) != 0 {
		t.Errorf("none chips = %v, want empty", none)
	}

	// subset.
	sub := contractChips(&apipb.OperatingModePhaseOutputContractSummary{RequiresVerdict: true})
	if !reflect.DeepEqual(sub, []string{"verdict"}) {
		t.Errorf("subset chips = %v", sub)
	}
}
