package graph

import (
	"testing"

	"github.com/stretchr/testify/require"

	"typescript-code-graph/internal/sidecar"
)

func TestFromSidecarWarnings_ClassifiesKnownKinds(t *testing.T) {
	in := []sidecar.Warning{
		{Kind: 1, Message: "bad token", File: "a.ts"},
		{Kind: 2, Message: "x", File: "b.ts"},
		{Kind: 3, Message: "tc", File: "c.ts"},
	}
	out := fromSidecarWarnings(in)
	require.Len(t, out, 3)
	require.Equal(t, WarningKindParseError, out[0].Kind)
	require.Equal(t, "a.ts", out[0].File)
	require.Equal(t, WarningKindUnresolvedImport, out[1].Kind)
	require.Equal(t, "b.ts", out[1].File)
	require.Equal(t, WarningKindTypeCheckFailure, out[2].Kind)
}

func TestFromSidecarWarnings_UnknownDefaultsToTypeCheck(t *testing.T) {
	in := []sidecar.Warning{{Kind: 99, Message: "x"}}
	out := fromSidecarWarnings(in)
	require.Equal(t, WarningKindTypeCheckFailure, out[0].Kind)
}

func TestFromSidecarWarnings_EmptyReturnsNil(t *testing.T) {
	require.Nil(t, fromSidecarWarnings(nil))
	require.Nil(t, fromSidecarWarnings([]sidecar.Warning{}))
}
