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
