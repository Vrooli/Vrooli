package validation

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCodeSeverityMatchesMaturitySpec(t *testing.T) {
	spec, err := LoadSpec(filepath.Join(repoRootFromTest(t), "scenarios", "workflow-health"))
	require.NoError(t, err)
	require.Equal(t, "workflow-health", spec.Provider)
	require.Equal(t, "workflow", spec.Phase)
	require.Equal(t, "2.0.0", spec.Version)
	require.Len(t, spec.Levels, 6)

	for code, mapping := range spec.Findings {
		require.NotEmptyf(t, mapping.CleanRequirement, "code %s", code)
		require.NotEmptyf(t, mapping.LocalLevelImpact, "code %s", code)
		want := specSeverityToLocal(mapping.SeverityDefault)
		require.NotEmptyf(t, want, "code %s severity %s", code, mapping.SeverityDefault)
		require.Equalf(t, want, codeSeverity[code], "code %s", code)
	}
	for code := range codeSeverity {
		require.Contains(t, spec.Findings, code)
	}
}

func specSeverityToLocal(value string) Severity {
	switch value {
	case "SEVERITY_ERROR", "SEVERITY_BLOCKER":
		return SeverityError
	case "SEVERITY_WARNING":
		return SeverityWarning
	case "SEVERITY_INFO":
		return SeverityInfo
	default:
		return ""
	}
}
