package scenarios

import (
	"testing"

	"github.com/stretchr/testify/require"
	scenariosv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/scenarios"
)

func TestLifecycleHuntCoveragePreservesCompletenessAndDegradation(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{name: "missing index", want: "unavailable (no index supplied)"},
		{name: "search envelope without coverage", data: []byte(`{"hits":[]}`), want: "unknown (index has no readable coverage metadata)"},
		{name: "complete source ledger export", data: []byte(`{"coverage":{"complete":true,"entry_count":110,"next_cursor":null}}`), want: "complete (110 entries)"},
		{name: "partial source ledger export", data: []byte(`{"coverage":{"complete":false,"degraded":["provider timeout"],"entry_count":40,"next_cursor":"next"}}`), want: "partial/degraded (40 entries, complete=false, next_cursor=true, degraded=[provider timeout])"},
		{name: "current top-level source ledger list", data: []byte(`{"source":"source-ledger.journal.list","scope":"ecosystem-hunt","complete":true,"degraded":false,"entry_count":110,"entries":[]}`), want: "complete (110 entries)"},
		{name: "top-level source ledger export", data: []byte(`{"source":"source-ledger.journal.export","scope":"ecosystem-hunt","complete":true,"degraded":false,"entry_count":110,"entries":[]}`), want: "complete (110 entries)"},
		{name: "partial top-level source ledger list", data: []byte(`{"source":"source-ledger.journal.list","scope":"ecosystem-hunt","complete":false,"degraded":["provider timeout"],"entry_count":40,"entries":[]}`), want: "partial/degraded (40 entries, complete=false, next_cursor=false, degraded=true)"},
		{name: "boolean degraded source ledger list", data: []byte(`{"source":"source-ledger.journal.list","scope":"ecosystem-hunt","complete":true,"degraded":true,"entry_count":40,"entries":[]}`), want: "partial/degraded (40 entries, complete=true, next_cursor=false, degraded=true)"},
		{name: "wrong source ledger scope", data: []byte(`{"source":"source-ledger.journal.list","scope":"other","complete":true,"degraded":false,"entry_count":110,"entries":[]}`), want: "unknown (index has no readable coverage metadata)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, lifecycleHuntCoverage(tt.data))
		})
	}
}

func TestCountPreselectionStatus(t *testing.T) {
	candidates := []*scenariosv1.LifecycleCandidate{
		{PreselectionStatus: "already_explicitly_modeled"},
		{PreselectionStatus: "prior_hunt_review_required"},
		nil,
	}
	require.Equal(t, 1, countPreselectionStatus(candidates, "already_explicitly_modeled"))
	require.Equal(t, 1, countPreselectionStatus(candidates, "prior_hunt_review_required"))
}
