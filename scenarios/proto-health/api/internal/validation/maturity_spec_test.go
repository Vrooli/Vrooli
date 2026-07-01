package validation

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
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
	require.Equal(t, "2.0.0", spec.Version)
	require.NotEmpty(t, spec.Capabilities)
	require.NotEmpty(t, spec.Fallback.CapabilityID)
	catalog, err := NewFindingCatalog(spec)
	require.NoError(t, err)
	for _, code := range AllFindingCodes() {
		_, ok := spec.Findings[code]
		require.Truef(t, ok, "missing maturity mapping for %s", code)
		require.NotEmptyf(t, spec.Findings[code].CapabilityID, "missing capability_id for %s", code)
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

	gatingCodeByCapabilityLevel := map[string]string{}
	for code, mapping := range spec.Findings {
		if mapping.CleanRequirement == string(assessment.CleanRequirementRequired) &&
			strings.Contains(mapping.SeverityDefault, "ERROR") &&
			mapping.GlobalImpact != assessment.ImpactAdvisory {
			gatingCodeByCapabilityLevel[mapping.CapabilityID+"."+mapping.LocalLevelImpact] = code
		}
	}

	for _, capability := range spec.Capabilities {
		t.Run(capability.ID, func(t *testing.T) {
			gatingCodes := 0
			for idx, level := range capability.Levels {
				code := gatingCodeByCapabilityLevel[capability.ID+"."+level.ID]
				if code == "" {
					continue
				}
				gatingCodes++
				got := assessment.CapabilityMaturity(*spec, []assessment.Finding{{Code: code}})
				var local assessment.LocalResult
				for _, item := range got {
					if item.CapabilityID == capability.ID {
						local = item
						break
					}
				}
				require.Equal(t, capability.ID, local.CapabilityID)
				if idx == 0 {
					require.Empty(t, local.CurrentLevel)
					require.Equal(t, level.ID, local.NextLevel)
				} else {
					require.Equal(t, capability.Levels[idx-1].ID, local.CurrentLevel)
					require.Equal(t, level.ID, local.NextLevel)
				}
				require.False(t, local.Clean)
				require.NotEmpty(t, local.BlockingFindingCodes)
			}
			require.NotZerof(t, gatingCodes, "capability %s has no enforced gate", capability.ID)
		})
	}

	got := assessment.LocalMaturity(*spec, nil)
	require.True(t, got.Clean)
	require.Empty(t, got.BlockingFindingCodes)
}
