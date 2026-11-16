package hub

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func shapeCodes(warnings []PacketWarning) []string {
	var codes []string
	for _, warning := range warnings {
		if warning.Code == WarnExplainerMissing {
			codes = append(codes, warning.Code)
		}
	}
	return codes
}

func TestEveryDecisionWithoutAPageIsToldToAddOne(t *testing.T) {
	plain := coverSpec()
	warnings := CheckPacket(plain, nil, nil)
	require.Equal(t, []string{WarnExplainerMissing}, shapeCodes(warnings))
	for _, warning := range warnings {
		if warning.Code == WarnExplainerMissing {
			require.Equal(t, WarningAdvice, warning.Severity, "a missing page never blocks an ask")
			require.Contains(t, warning.Message, "--explainer")
			require.Contains(t, warning.Message, "examples/")
		}
	}
}
