package scenarios

import (
	"testing"

	"github.com/stretchr/testify/require"
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, lifecycleHuntCoverage(tt.data))
		})
	}
}
