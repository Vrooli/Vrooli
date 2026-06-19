package validation

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/maturity-go/assessment"
)

func TestMaturitySpecCoversProtoHealthFindings(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", ".vrooli", "maturity.json"))
	require.NoError(t, err)
	spec, err := assessment.ParseSpec(raw)
	require.NoError(t, err)

	require.Equal(t, "proto-health", spec.Provider)
	require.Equal(t, "proto", spec.Phase)
	catalog, err := NewFindingCatalog(spec)
	require.NoError(t, err)
	for _, code := range AllFindingCodes() {
		_, ok := spec.Findings[code]
		require.Truef(t, ok, "missing maturity mapping for %s", code)
		require.NotEmptyf(t, spec.Findings[code].CleanRequirement, "missing clean_requirement for %s", code)
		require.Truef(t, assessment.IsValidCleanRequirement(assessment.CleanRequirement(spec.Findings[code].CleanRequirement)), "invalid clean_requirement for %s", code)
		severity, err := catalog.ResolveSeverity(code)
		require.NoError(t, err)
		require.NotEmpty(t, severity)
	}
	want := append([]string{}, AllFindingCodes()...)
	got := make([]string, 0, len(spec.Findings))
	for code := range spec.Findings {
		got = append(got, code)
	}
	sort.Strings(want)
	sort.Strings(got)
	require.Equal(t, want, got)
}
