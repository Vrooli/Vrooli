package gates

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateSpecifierShapeFindsBareAndExactPins(t *testing.T) {
	root := specifierShapeFixture(t)
	writeSpecifierShapeFile(t, root, "library/components/button/component.json", `{"catalogId":"catalog.button"}`)
	writeSpecifierShapeFile(t, root, "library/components/button/versions/1/index.tsx", `
import Button from "@vrooli/react-component-library/button";
import { Icon } from "@vrooli/react-component-library/icon/2.1.0";
import { Card } from "@vrooli/react-component-library/card/3";
`)

	result, err := ValidateSpecifierShape(Scope{Root: root})
	require.NoError(t, err)
	require.Equal(t, 1, result.Inspected)
	require.Len(t, result.Findings, 2)
	require.Equal(t, []string{"catalog.specifier_bare", "catalog.specifier_exact_pin"}, []string{
		result.Findings[0].Code,
		result.Findings[1].Code,
	})
	require.Equal(t, "catalog.button", result.Findings[0].AssetID)
	require.Equal(t, "scenarios/react-component-library/library/components/button/versions/1/index.tsx", result.Findings[0].File)
	require.Contains(t, result.Findings[0].Remediation, "major line")
	require.Contains(t, result.Findings[1].Message, "icon/2.1.0")
	require.Contains(t, result.Findings[1].Remediation, "icon/<major>")
}

func TestValidateSpecifierShapeAllowsMajorSelectorsAndSkipsPrereleaseVersions(t *testing.T) {
	root := specifierShapeFixture(t)
	writeSpecifierShapeFile(t, root, "library/components/button/component.json", `{"catalogId":"catalog.button"}`)
	writeSpecifierShapeFile(t, root, "library/components/button/versions/1/index.tsx", `import Button from "@vrooli/react-component-library/button/1";`)
	writeSpecifierShapeFile(t, root, "library/components/button/versions/2-beta/index.tsx", `import Legacy from "@vrooli/react-component-library/button/2.0.0";`)

	result, err := ValidateSpecifierShape(Scope{Root: root})
	require.NoError(t, err)
	require.Equal(t, 1, result.Inspected, "pre-release implementation versions are outside this release gate")
	require.Empty(t, result.Findings)
}

func TestValidateSpecifierShapeHonorsBackfilledProvenance(t *testing.T) {
	root := specifierShapeFixture(t)
	writeSpecifierShapeFile(t, root, "library/components/button/component.json", `{"catalogId":"catalog.button"}`)
	writeSpecifierShapeFile(t, root, "library/components/button/versions/1/index.tsx", `import Button from "@vrooli/react-component-library/button/2.0.0";`)
	writeSpecifierShapeFile(t, root, "library/release-provenance.json", `{"entries":[{"libraryId":"react-component-library:button","version":"1","backfilled":true}]}`)

	result, err := ValidateSpecifierShape(Scope{Root: root})
	require.NoError(t, err)
	require.Equal(t, 1, result.Inspected)
	require.Empty(t, result.Findings, "historical backfilled releases are exempt from the new selector policy")
}

func TestValidateSpecifierShapeReturnsSourceReadErrors(t *testing.T) {
	root := specifierShapeFixture(t)
	path := filepath.Join(root, "scenarios/react-component-library/library/components/button/versions/1/index.tsx")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.Symlink(filepath.Join(filepath.Dir(path), "missing.tsx"), path))

	_, err := ValidateSpecifierShape(Scope{Root: root})
	require.Error(t, err)
	require.Contains(t, err.Error(), path)
}

func specifierShapeFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "scenarios/react-component-library/library"), 0o755))
	return root
}

func writeSpecifierShapeFile(t *testing.T, root, path, contents string) {
	t.Helper()
	fullPath := filepath.Join(root, "scenarios/react-component-library", filepath.FromSlash(path))
	require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o755))
	require.NoError(t, os.WriteFile(fullPath, []byte(contents), 0o644))
}
