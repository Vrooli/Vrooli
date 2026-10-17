package vector

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRoundTripUsesCompactRepresentation(t *testing.T) {
	original := []float64{0.125, -0.25, 0.3333333}
	raw := Encode(original)
	require.Less(t, len(raw), len("[0.125,-0.25,0.3333333]"))
	decoded, err := Decode(raw)
	require.NoError(t, err)
	require.InDeltaSlice(t, original, decoded, 0.00001)
}
