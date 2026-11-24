package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateVersionShapeChecksLiveVersionsAndRespectsRetirementAndScope(t *testing.T) {
	root := t.TempDir()
	library := filepath.Join(root, "scenarios", "react-component-library", "library")
	policyPath := filepath.Join(root, "scenarios", "react-component-library", "catalog", "version-shape.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(policyPath), 0o700))
	require.NoError(t, os.WriteFile(policyPath, []byte(`{"enforcement":"full"}`), 0o600))
	writeVersion := func(kind, name, catalogID, manifest string, files map[string]string) string {
		t.Helper()
		assetDir := filepath.Join(library, kind, name)
		versionDir := filepath.Join(assetDir, "versions", "1.0.0")
		require.NoError(t, os.MkdirAll(versionDir, 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(assetDir, "component.json"), []byte(manifest), 0o600))
		for file, body := range files {
			path := filepath.Join(versionDir, file)
			require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
			require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
		}
		if catalogID != "" {
			assetPath := filepath.Join(root, "scenarios", "react-component-library", "catalog", "assets", "components", name+".json")
			require.NoError(t, os.MkdirAll(filepath.Dir(assetPath), 0o700))
			require.NoError(t, os.WriteFile(assetPath, []byte(`{"kind":"catalog-asset","asset":{"id":"`+catalogID+`"}}`), 0o600))
		}
		return versionDir
	}
	writeVersion("components", "Healthy", "components.healthy", `{"catalogId":"components.healthy"}`, map[string]string{
		"Healthy.tsx": "export const Healthy = () => null;", "story.tsx": "export default {};", "story.json": "{}", "dependencies.json": "{}",
	})
	writeVersion("components", "Broken", "components.broken", `{"catalogId":"components.broken"}`, map[string]string{
		"Unexpected.md": "not part of a published version", "nested/extra.ts": "forbidden nested file",
	})
	writeVersion("primitives", "Uncataloged", "", `{"catalogId":"react-component-library:Uncataloged"}`, map[string]string{"Uncataloged.tsx": "export {};"})
	writeVersion("hooks", "Retired", "hooks.retired", `{"catalogId":"hooks.retired","deprecatedVersions":["1.0.0"]}`, map[string]string{"Retired.tsx": "export {};"})
	retiredFolder := filepath.Join(library, "components", "Broken", "versions", "1.0.0.retired")
	require.NoError(t, os.MkdirAll(retiredFolder, 0o700))

	result, err := ValidateVersionShape(Scope{Root: root})
	require.NoError(t, err)
	require.Equal(t, 3, result.Inspected, "retired versions and explicit retired folders are excluded")
	require.NotEmpty(t, result.Findings)
	var broken, corpus bool
	for _, finding := range result.Findings {
		if finding.AssetID == "components.broken" {
			broken = true
			require.True(t, strings.Contains(finding.Message, "required") || strings.Contains(finding.Message, "not part of the canonical version shape") || strings.Contains(finding.Message, "subdirectories are forbidden"))
		}
		if finding.AssetID == "__corpus__.version-shape" {
			corpus = true
		}
	}
	require.True(t, broken, "catalog-backed shape findings retain their asset identity")
	require.True(t, corpus, "uncataloged implementation findings are reported at corpus scope")

	scoped, err := ValidateVersionShape(Scope{Root: root, Assets: []string{"components.broken"}})
	require.NoError(t, err)
	require.Equal(t, 1, scoped.Inspected)
	require.NotEmpty(t, scoped.Findings)
}
