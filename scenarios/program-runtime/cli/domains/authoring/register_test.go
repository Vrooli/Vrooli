package authoring

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuthoringEvalGateRequiresMeasuredStatusAndFloor(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		floorMet bool
		passes   bool
	}{
		{name: "measured above floor", status: "measured", floorMet: true, passes: true},
		{name: "measured below floor", status: "measured", floorMet: false, passes: false},
		{name: "partial", status: "partial", floorMet: true, passes: false},
		{name: "unavailable", status: "unavailable", floorMet: false, passes: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.passes, authoringEvalPasses(test.status, test.floorMet))
		})
	}
}
