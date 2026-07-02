package validation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEngineFindsWorkflowMaturityGates(t *testing.T) {
	root := makeValidationFixture(t)

	report, err := NewEngine().ValidateScenario(t.Context(), "", root)
	require.NoError(t, err)
	require.Equal(t, "sample-scenario", report.Scenario)

	codes := findingCodes(report.Findings)
	require.Contains(t, codes, CodeRegistryStale)
	require.Contains(t, codes, CodeMetadataIncomplete)
	require.Contains(t, codes, CodeRequirementUnlinked)
	require.Contains(t, codes, CodeSelectorUnregistered)
	require.Contains(t, codes, CodeSubflowUnresolved)
	require.Contains(t, codes, CodeExecutionModeInvalid)
	require.Contains(t, codes, CodeResetLegacy)
	require.Contains(t, codes, CodeMutatingSafety)
	require.Contains(t, codes, CodeSeedMissing)
	require.NotContains(t, codes, CodeSurfaceAbsent)

	for _, f := range report.Findings {
		require.NotEmpty(t, f.Title)
		require.NotEmpty(t, f.Description)
		require.NotEmpty(t, f.Remediation)
		require.NotEmpty(t, f.Severity)
	}
}

func TestBuildMaturityAssessmentUsesWorkflowSpec(t *testing.T) {
	root := makeValidationFixture(t)
	report, err := NewEngine().ValidateScenario(t.Context(), "", root)
	require.NoError(t, err)

	spec, err := LoadSpec(filepath.Join(repoRootFromTest(t), "scenarios", "workflow-health"))
	require.NoError(t, err)
	assessment, err := BuildMaturityAssessment(report.Scenario, report.Findings, *spec)
	require.NoError(t, err)

	require.Equal(t, "workflow-health", assessment.Provider)
	require.Equal(t, "workflow", assessment.Phase)
	require.NotEmpty(t, assessment.Findings)
	require.NotEmpty(t, assessment.Local.GetCurrentLevel())
}

func TestFixRegistryPreviewsAndAppliesMechanicalWorkflowFixes(t *testing.T) {
	root := makeValidationFixture(t)
	registry := NewFixRegistry()

	candidates, err := registry.Preview(root, []string{CodeRegistryStale, CodeMetadataIncomplete, CodeExecutionModeInvalid, CodeResetLegacy})
	require.NoError(t, err)
	require.NotEmpty(t, candidates)
	require.True(t, registry.CanFix(root, CodeResetLegacy, "bas/cases/02-admin/delete-project.json"))

	byRule := map[string]int{}
	for _, candidate := range candidates {
		byRule[candidate.RuleID]++
		require.NotEmpty(t, candidate.FilePath)
		require.NotEqual(t, candidate.Before, candidate.After)
	}
	require.Positive(t, byRule[CodeRegistryStale])
	require.Positive(t, byRule[CodeMetadataIncomplete])
	require.Positive(t, byRule[CodeExecutionModeInvalid])
	require.Positive(t, byRule[CodeResetLegacy])

	applied, err := registry.Apply(root, []string{CodeResetLegacy})
	require.NoError(t, err)
	require.Len(t, applied, 1)
	require.True(t, applied[0].Applied)

	repreview, err := registry.Preview(root, []string{CodeResetLegacy})
	require.NoError(t, err)
	require.Empty(t, repreview)
}

func TestEngineReportsSurfaceAbsentForEmptyScenario(t *testing.T) {
	root := filepath.Join(t.TempDir(), "empty")
	require.NoError(t, os.MkdirAll(root, 0o755))

	report, err := NewEngine().ValidateScenario(t.Context(), "", root)
	require.NoError(t, err)
	require.Equal(t, []string{CodeSurfaceAbsent}, findingCodes(report.Findings))
}

func makeValidationFixture(t *testing.T) string {
	t.Helper()

	root := filepath.Join(t.TempDir(), "sample-scenario")
	writeJSON(t, filepath.Join(root, "bas", "registry.json"), map[string]any{
		"scenario":     "sample-scenario",
		"generated_at": "2026-07-02T12:00:00Z",
		"metadata":     map[string]any{"execution_mode": "observer"},
		"playbooks": []map[string]any{
			{
				"file":         "bas/cases/01-foundation/create-project.json",
				"description":  "registry description",
				"order":        "01.01",
				"requirements": []string{"REQ-CREATE"},
				"fixtures":     []string{},
				"reset":        "none",
			},
			{
				"file":         "bas/cases/stale/missing.json",
				"description":  "stale",
				"order":        "99.01",
				"requirements": []string{"REQ-STALE"},
				"fixtures":     []string{},
				"reset":        "none",
			},
		},
	})
	writeJSON(t, filepath.Join(root, "requirements", "index.json"), map[string]any{
		"imports": []string{"module.json"},
	})
	writeJSON(t, filepath.Join(root, "requirements", "module.json"), map[string]any{
		"requirements": []map[string]any{
			{
				"id": "REQ-CREATE",
				"validation": []map[string]any{
					{"ref": "bas/cases/01-foundation/create-project.json"},
				},
			},
		},
	})
	writeJSON(t, filepath.Join(root, "bas", "cases", "01-foundation", "create-project.json"), map[string]any{
		"metadata": map[string]any{
			"name":           "Create project",
			"description":    "Creates a project through the UI.",
			"execution_mode": "observer",
			"labels":         map[string]any{"reset": "none"},
		},
		"nodes": []map[string]any{
			{
				"id": "direct-selector",
				"action": map[string]any{
					"assert": map[string]any{"selector": "[data-testid='project-list']"},
				},
			},
		},
	})
	writeJSON(t, filepath.Join(root, "bas", "cases", "02-admin", "delete-project.json"), map[string]any{
		"metadata": map[string]any{
			"name":           "Delete project",
			"execution_mode": "mutating",
			"labels":         map[string]any{"reset": "database"},
		},
		"nodes": []map[string]any{
			{
				"id": "missing-subflow",
				"action": map[string]any{
					"subflow": map[string]any{"workflow_path": "actions/missing-seed.json"},
				},
			},
		},
	})
	writeJSON(t, filepath.Join(root, "bas", "flows", "01-user", "broken-mode.json"), map[string]any{
		"metadata": map[string]any{
			"execution_mode": "danger",
			"labels":         map[string]any{"intent": "broken mode"},
		},
		"nodes": []map[string]any{},
	})
	writeJSON(t, filepath.Join(root, "bas", "actions", "open-demo-project.json"), map[string]any{
		"metadata": map[string]any{"name": "Open demo project", "description": "Open demo project.", "execution_mode": "observer"},
		"nodes":    []map[string]any{},
	})
	return root
}

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	data, err := json.MarshalIndent(value, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, append(data, '\n'), 0o644))
}

func findingCodes(findings []Finding) []string {
	out := make([]string, 0, len(findings))
	for _, finding := range findings {
		out = append(out, finding.Code)
	}
	return out
}

func repoRootFromTest(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			parent := filepath.Dir(filepath.Dir(filepath.Dir(dir)))
			if _, err := os.Stat(filepath.Join(parent, "scenarios", "workflow-health")); err == nil {
				return parent
			}
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, dir, parent, "repo root not found")
		dir = parent
	}
}
