package versions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildReachabilityIndexResolvesBareMajorExactAndKeepsNewestMajor(t *testing.T) {
	root := t.TempDir()
	writeReachabilityFile(t, root, "scenarios/react-component-library/library/components/Button/versions/1.0.0/Button.tsx", "export const Button = 1;\n")
	writeReachabilityFile(t, root, "scenarios/react-component-library/library/components/Button/versions/1.1.0/Button.tsx", "export const Button = 11;\n")
	writeReachabilityFile(t, root, "scenarios/react-component-library/library/components/Button/versions/2.0.0/Button.tsx", "export const Button = 2;\n")
	writeReachabilityFile(t, root, "scenarios/react-component-library/library/hooks/useThing/versions/1.0.0/useThing.ts", "export const useThing = 1;\n")
	writeReachabilityFile(t, root, "scenarios/react-component-library/library/hooks/useThing/versions/1.1.0/useThing.ts", "export const useThing = 11;\n")
	writeReachabilityFile(t, root, "scenarios/react-component-library/library/release-provenance.json", `{"schemaVersion":1,"entries":[]}`)
	writeReachabilityFile(t, root, "consumer/src.ts", `import "@vrooli/react-component-library/Button";\nimport "@vrooli/react-component-library/useThing/1";\nimport "@vrooli/react-component-library/Button/1.0.0";\n`)

	index, err := buildReachabilityIndex(root)
	require.NoError(t, err)
	require.Len(t, index.Report.OnDisk, 5)
	require.Contains(t, versionsOf(index.Report.Reachable), "components/Button@1.0.0")
	require.Contains(t, versionsOf(index.Report.Reachable), "components/Button@1.1.0")
	require.Contains(t, versionsOf(index.Report.Reachable), "components/Button@2.0.0")
	require.Contains(t, versionsOf(index.Report.Reachable), "hooks/useThing@1.1.0")
	require.Contains(t, versionsOf(index.Report.Unreachable), "hooks/useThing@1.0.0")
	require.Empty(t, index.Report.AssetsWithNoImporter)
}

func TestReleaseProvenanceEntryCountUsesLedgerEntries(t *testing.T) {
	root := t.TempDir()
	libraryRoot := filepath.Join(root, "library")
	require.NoError(t, os.MkdirAll(libraryRoot, 0o755))
	body, err := json.Marshal(map[string]any{"schemaVersion": 1, "entries": []any{map[string]any{"libraryId": "rcl:Button", "version": "1.0.0"}}})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(libraryRoot, "release-provenance.json"), body, 0o644))
	count, err := releaseProvenanceEntryCount(libraryRoot)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestBuildReachabilityIndexProtectsRelativeVersionImports(t *testing.T) {
	root := t.TempDir()
	writeReachabilityFile(t, root, "scenarios/react-component-library/library/components/Panel/versions/1.0.0/Panel.tsx", `import { useAnnounce } from "../../../../hooks/useAnnounce/versions/1.0.0/useAnnounce";\nexport const Panel = useAnnounce;\n`)
	writeReachabilityFile(t, root, "scenarios/react-component-library/library/hooks/useAnnounce/versions/1.0.0/useAnnounce.ts", "export const useAnnounce = () => undefined;\n")
	writeReachabilityFile(t, root, "scenarios/react-component-library/library/hooks/useAnnounce/versions/1.1.0/useAnnounce.ts", "export const useAnnounce = () => undefined;\n")
	writeReachabilityFile(t, root, "scenarios/react-component-library/library/release-provenance.json", `{"schemaVersion":1,"entries":[]}`)

	index, err := buildReachabilityIndex(root)
	require.NoError(t, err)
	require.Contains(t, versionsOf(index.Report.Reachable), "hooks/useAnnounce@1.0.0")
	require.Contains(t, versionsOf(index.Report.Reachable), "hooks/useAnnounce@1.1.0")
	require.Empty(t, index.Report.Unreachable)
	require.Len(t, index.Report.AssetsWithNoImporter, 1)
	require.Equal(t, "Panel", index.Report.AssetsWithNoImporter[0].Asset)
}

func writeReachabilityFile(t *testing.T, root, relative, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
}

func versionsOf(items []reachabilityVersion) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.Kind+"/"+item.Asset+"@"+item.Version)
	}
	return result
}
