package adoptions_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"react-component-library/internal/adoptions"
	"react-component-library/internal/components"
)

func TestAdoptionVerdictBlocksUnmeasuredContractCoverage(t *testing.T) {
	verdict := adoptions.AdoptionVerdict{
		Version:   components.VersionStatusReleased,
		I18n:      "not-measured",
		Selectors: "pass",
	}
	require.True(t, verdict.Blocking())

	verdict.I18n = "pass"
	verdict.Selectors = "not-measured"
	require.True(t, verdict.Blocking())
}
