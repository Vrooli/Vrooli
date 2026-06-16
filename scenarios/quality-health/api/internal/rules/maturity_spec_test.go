package rules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/maturity-go/assessment"
)

func TestMaturitySpecCoversQualityHealthFindings(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", ".vrooli", "maturity.json"))
	require.NoError(t, err)
	spec, err := assessment.ParseSpec(raw)
	require.NoError(t, err)

	for _, rule := range Registry() {
		if _, ok := spec.Findings[rule.ID]; !ok {
			t.Fatalf("maturity spec missing rule %s", rule.ID)
		}
	}
	if _, ok := spec.Findings[RuleCoverageGap]; !ok {
		t.Fatalf("maturity spec missing rule %s", RuleCoverageGap)
	}
}
