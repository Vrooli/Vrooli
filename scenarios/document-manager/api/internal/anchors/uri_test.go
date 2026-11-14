package anchors

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCanonicalAnchorKindsRoundTrip(t *testing.T) { // [REQ:DOC-P0-019] [REQ:DOC-P0-028]
	cases := []struct {
		name string
		uri  URI
	}{
		{"geometric", URI{DocumentHash: "sha256-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", Derivation: 3, Kind: KindGeometric, Coordinates: Coordinates{Page: 7, Box: [4]float64{.118, .342, .56, .411}}}},
		{"tabular", URI{DocumentHash: "sha256-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", Derivation: 3, Kind: KindTabular, Coordinates: Coordinates{Sheet: 2, StartCell: "B4", EndCell: "D9"}}},
		{"logical", URI{DocumentHash: "sha256-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", Derivation: 3, Kind: KindLogical, Coordinates: Coordinates{StablePath: "7", Path: "2/1", Start: 120, End: 260}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := tc.uri.String()
			require.NoError(t, err)
			parsed, err := Parse(raw)
			require.NoError(t, err)
			canonical, err := parsed.String()
			require.NoError(t, err)
			require.Equal(t, raw, canonical)
		})
	}
}

func TestNonCanonicalInputIsRejected(t *testing.T) { // [REQ:DOC-P0-028]
	raw := "vrooli-anchor:1/sha256-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef/1/geometric/p1@0.5,0.342000,0.560000,0.411000"
	_, err := Parse(raw)
	require.Error(t, err)
}

func TestResolutionHasSixOutcomesAndNoGuess(t *testing.T) { // [REQ:DOC-P0-009] [REQ:DOC-P0-028]
	u := URI{Kind: KindLogical, Coordinates: Coordinates{StablePath: "7", Path: "2/1", Start: 1, End: 3}}
	require.Equal(t, ResolvedDegraded, Resolve(u, true, true, true, "", "slide-7").Outcome)
	require.Equal(t, Unresolved, Resolve(u, true, true, true, "", "").Outcome)
	require.Equal(t, Forbidden, Resolve(u, true, true, false, "", "").Outcome)
	require.Equal(t, UnknownVersion, Resolve(u, true, false, true, "", "").Outcome)
	require.Equal(t, UnknownDocument, Resolve(u, false, true, true, "", "").Outcome)
	require.Equal(t, Resolved, Resolve(u, true, true, true, "exact", "").Outcome)
}
