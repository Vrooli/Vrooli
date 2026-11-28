package deps

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSourceDeclarationsCollectsRuntimeImportsFromSourceHeaders(t *testing.T) {
	declarations, err := ParseSourceDeclarations(`/**
 * @deps {"react":"^18","clsx":"^2.1.0"}
 */
import { clsx } from "clsx";

// @deps {"tailwind-merge":"^2.2.0"}
`)
	require.NoError(t, err)
	byName := map[string]string{}
	for _, declaration := range declarations {
		byName[declaration.DepName] = declaration.VersionRange
	}
	require.Equal(t, "^18", byName["react"])
	require.Equal(t, "^2.1.0", byName["clsx"])
	require.Equal(t, "^2.2.0", byName["tailwind-merge"])
}

func TestObjectDependencyHeadersHaveCanonicalOrder(t *testing.T) {
	cases := []string{
		`{"tailwind-merge":"^2.2.0","clsx":"^2.1.0","react":"^18"}`,
		`{"react":{"range":"^18","kind":"peer"},"tailwind-merge":{"range":"^2.2.0"},"clsx":{"range":"^2.1.0"}}`,
	}
	for _, raw := range cases {
		for attempt := 0; attempt < 32; attempt++ {
			got, err := ParseHeaderField(raw)
			require.NoError(t, err)
			names := make([]string, len(got))
			for i, d := range got {
				names[i] = d.DepName
			}
			require.Equal(t, []string{"clsx", "react", "tailwind-merge"}, names, "JSON object member order must not alter runtime dependency identity")
		}
	}
}
