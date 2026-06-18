package compat_test

import (
	"testing"

	"vrooli-bridge/internal/compat"

	"github.com/stretchr/testify/require"
)

// [REQ:BRG-P1-001] The evaluator classifies a node's reported protocol version
// across the four bands: unspecified (back-compat), ok (equal/newer),
// needs-update (older but supported), incompatible (below the floor).
func TestEvaluateAt_Bands(t *testing.T) {
	const current, min = uint32(3), uint32(2)

	require.Equal(t, compat.StatusUnspecified, compat.EvaluateAt(0, current, min), "absence is not a fault")
	require.Equal(t, compat.StatusOK, compat.EvaluateAt(3, current, min), "equal is ok")
	require.Equal(t, compat.StatusOK, compat.EvaluateAt(4, current, min), "newer is ok (DiscardUnknown)")
	require.Equal(t, compat.StatusNeedsUpdate, compat.EvaluateAt(2, current, min), "older-but-supported needs update")
	require.Equal(t, compat.StatusIncompatible, compat.EvaluateAt(1, current, min), "below the floor is incompatible")
}

// [REQ:BRG-P1-001] Only OK / Unspecified are dispatchable; a flagged node is
// excluded from work.
func TestStatus_Dispatchable(t *testing.T) {
	require.True(t, compat.StatusOK.Dispatchable())
	require.True(t, compat.StatusUnspecified.Dispatchable())
	require.False(t, compat.StatusNeedsUpdate.Dispatchable())
	require.False(t, compat.StatusIncompatible.Dispatchable())
}

// [REQ:BRG-P1-001] The live evaluator uses the package's current/min constants;
// a v1 node against a v1 control plane is OK (today's fleet is all v1).
func TestEvaluate_LiveConstants(t *testing.T) {
	require.Equal(t, compat.StatusOK, compat.Evaluate(compat.ProtocolVersion))
	require.Equal(t, compat.StatusUnspecified, compat.Evaluate(0))
}

func TestStatus_String(t *testing.T) {
	require.Equal(t, "ok", compat.StatusOK.String())
	require.Equal(t, "needs_update", compat.StatusNeedsUpdate.String())
	require.Equal(t, "incompatible", compat.StatusIncompatible.String())
	require.Equal(t, "unspecified", compat.StatusUnspecified.String())
}
