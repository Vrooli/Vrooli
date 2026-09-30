package discovery

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiscoveryEvalGateRequiresMeasuredMetStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		floorMet bool
		passes   bool
	}{
		{name: "met above floor", status: "met", floorMet: true, passes: true},
		{name: "met below floor", status: "met", floorMet: false, passes: false},
		{name: "partial", status: "partial", floorMet: true, passes: false},
		{name: "unavailable", status: "unavailable", floorMet: false, passes: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.passes, discoveryEvalPasses(test.status, test.floorMet))
		})
	}
}
