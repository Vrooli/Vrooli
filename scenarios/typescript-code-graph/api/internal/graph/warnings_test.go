package graph

import (
	"testing"

	"github.com/stretchr/testify/require"

	"typescript-code-graph/internal/sidecar"
)

func TestFromSidecarWarnings_ClassifiesKnownCodes(t *testing.T) {
	in := []sidecar.Warning{
		{Code: "parse_failure", Message: "bad token", Path: "a.ts"},
		{Code: "unresolved_import", Message: "x", Path: "b.ts"},
		{Code: "type_check_failure", Message: "tc", Path: "c.ts"},
	}
	out := fromSidecarWarnings(in)
	require.Len(t, out, 3)
	require.Equal(t, WarningKindParseError, out[0].Kind)
	require.Equal(t, "a.ts", out[0].File)
	require.Equal(t, WarningKindUnresolvedImport, out[1].Kind)
	require.Equal(t, WarningKindTypeCheckFailure, out[2].Kind)
}

func TestFromSidecarWarnings_UnknownDefaultsToTypeCheck(t *testing.T) {
	in := []sidecar.Warning{{Code: "future_code", Message: "x"}}
	out := fromSidecarWarnings(in)
	require.Equal(t, WarningKindTypeCheckFailure, out[0].Kind)
}

func TestFromSidecarWarnings_EmptyReturnsNil(t *testing.T) {
	require.Nil(t, fromSidecarWarnings(nil))
	require.Nil(t, fromSidecarWarnings([]sidecar.Warning{}))
}
