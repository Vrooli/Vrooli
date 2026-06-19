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

func TestMaturitySpecKeepsEveryLocalRungReachable(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", ".vrooli", "maturity.json"))
	require.NoError(t, err)
	spec, err := assessment.ParseSpec(raw)
	require.NoError(t, err)

	requiredCodeByLevel := map[string]string{}
	for code, mapping := range spec.Findings {
		if mapping.CleanRequirement == string(assessment.CleanRequirementRequired) {
			requiredCodeByLevel[mapping.LocalLevelImpact] = code
		}
	}

	for _, test := range []struct {
		name       string
		codeLevel  string
		wantLevel  string
		wantNext   string
		wantClean  bool
		wantBlocks bool
	}{
		{name: "L0", codeLevel: "L1", wantLevel: "L0", wantNext: "L1", wantBlocks: true},
		{name: "L1", codeLevel: "L2", wantLevel: "L1", wantNext: "L2", wantBlocks: true},
		{name: "L2", codeLevel: "L3", wantLevel: "L2", wantNext: "L3", wantBlocks: true},
		{name: "L3", codeLevel: "L4", wantLevel: "L3", wantNext: "L4", wantBlocks: true},
		{name: "L4", codeLevel: "L5", wantLevel: "L4", wantNext: "L5", wantBlocks: true},
		{name: "L5", wantLevel: "L5", wantClean: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			findings := []assessment.Finding{}
			if test.codeLevel != "" {
				code := requiredCodeByLevel[test.codeLevel]
				require.NotEmptyf(t, code, "no REQUIRED finding maps to %s", test.codeLevel)
				findings = append(findings, assessment.Finding{Code: code})
			}

			got := assessment.LocalMaturity(*spec, findings)
			require.Equal(t, test.wantLevel, got.CurrentLevel)
			require.Equal(t, test.wantNext, got.NextLevel)
			require.Equal(t, test.wantClean, got.Clean)
			if test.wantBlocks {
				require.NotEmpty(t, got.BlockingFindingCodes)
			} else {
				require.Empty(t, got.BlockingFindingCodes)
			}
		})
	}
}
