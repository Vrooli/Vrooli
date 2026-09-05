package preview

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	"react-component-library/internal/components"
)

func TestScanImportedSourceDeclarationsFollowsRelativeImports(t *testing.T) {
	root := t.TempDir()
	entryDir := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Root", "versions", "1.0.0")
	dependencyPath := filepath.Join(root, "scenarios", "react-component-library", "library", "foundations", "Runtime", "versions", "1.0.0", "Runtime.tsx")
	require.NoError(t, os.MkdirAll(entryDir, 0o755))
	require.NoError(t, os.MkdirAll(filepath.Dir(dependencyPath), 0o755))
	require.NoError(t, os.WriteFile(dependencyPath, []byte("/**\n * @deps {\"clsx\": \"^2.1.0\"}\n */\nexport const runtime = true;\n"), 0o644))

	svc := &service{repoRoot: root}
	fields, err := svc.scanImportedSourceDeclarations([]components.ComponentVersionFile{{
		Path:    "Root.tsx",
		Content: "import { runtime } from \"../../../../foundations/Runtime/versions/1.0.0/Runtime\";\nexport { runtime };",
	}}, "components/Root/versions/1.0.0/Root.tsx")
	require.NoError(t, err)
	require.Len(t, fields, 1)
	require.Equal(t, "clsx", fields[0].DepName)
	require.Equal(t, "^2.1.0", fields[0].VersionRange)
}

func TestScanImportedSourceDeclarationsFollowsVersionPinnedCatalogImports(t *testing.T) {
	root := t.TempDir()
	entryDir := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Root", "versions", "1.0.0")
	dependencyPath := filepath.Join(root, "scenarios", "react-component-library", "library", "foundations", "ClassMerge", "versions", "1.0.2", "ClassMerge.tsx")
	require.NoError(t, os.MkdirAll(entryDir, 0o755))
	require.NoError(t, os.MkdirAll(filepath.Dir(dependencyPath), 0o755))
	require.NoError(t, os.WriteFile(dependencyPath, []byte("/**\n * @deps {\"clsx\": \"^2.1.0\", \"tailwind-merge\": \"^2.2.0\"}\n */\nexport const cn = true;\n"), 0o644))

	svc := &service{repoRoot: root}
	fields, err := svc.scanImportedSourceDeclarations([]components.ComponentVersionFile{{
		Path:    "Root.tsx",
		Content: "import { cn } from \"@vrooli/react-component-library/ClassMerge/1.0.2\";\nexport { cn };",
	}}, "components/Root/versions/1.0.0/Root.tsx")
	require.NoError(t, err)
	require.Len(t, fields, 2)
	sort.Slice(fields, func(i, j int) bool { return fields[i].DepName < fields[j].DepName })
	require.Equal(t, "clsx", fields[0].DepName)
	require.Equal(t, "tailwind-merge", fields[1].DepName)
}
