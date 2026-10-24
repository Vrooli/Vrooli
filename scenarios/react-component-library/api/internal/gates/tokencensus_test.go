package gates

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTokenCensusClassifiesEveryCompatibilityVerdict(t *testing.T) {
	root := t.TempDir()
	writeCensusFixture(t, root, "templates/design/_base/tokens.css", ":root {\n  /* @tier Expression */\n  --shared: red;\n}\n")
	writeCensusFixture(t, root, "templates/design/kit-a/metadata.json", `{}`)
	writeCensusFixture(t, root, "templates/design/kit-a/adapters/react-vite-tailwind/tokens.css", ":root {\n  --a: red;\n}\n")
	writeCensusFixture(t, root, "templates/design/kit-b/metadata.json", `{}`)
	writeCensusFixture(t, root, "templates/design/kit-b/adapters/react-vite-tailwind/tokens.css", ":root {\n  --b: blue;\n}\n")
	writeAssetFixture(t, root, "foundations", "BaseStyles", "rcl:BaseStyles", `:root { --base: 1px; }`, nil)
	writeAssetFixture(t, root, "components", "Universal", "rcl:Universal", `.x { color: var(--shared); }`, []string{"kit-a", "kit-b"})
	writeAssetFixture(t, root, "components", "Restricted", "rcl:Restricted", `.x { color: var(--a); }`, []string{"kit-a", "kit-b"})
	writeAssetFixture(t, root, "components", "Unsatisfiable", "rcl:Unsatisfiable", `.x { color: var(--a); background: var(--b); }`, nil)
	writeAssetFixture(t, root, "components", "Undefined", "rcl:Undefined", `.x { color: var(--missing); }`, []string{"kit-a"})

	census, err := TokenCensus(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, verdict := range []CompatibilityVerdict{
		CompatibilityUniversal,
		CompatibilityRestricted,
		CompatibilityUnsatisfiable,
		CompatibilityUndefinedVocabulary,
	} {
		if census.VerdictCounts[string(verdict)] != 1 {
			t.Fatalf("verdict %s count = %d, want 1", verdict, census.VerdictCounts[string(verdict)])
		}
	}
	if len(census.UndefinedProperties) != 1 || census.UndefinedProperties[0] != "--missing" {
		t.Fatalf("undefined properties = %#v, want [--missing]", census.UndefinedProperties)
	}
	if len(census.AffinityOverclaims) != 1 || census.AffinityOverclaims[0].LibraryID != "rcl:Restricted" {
		t.Fatalf("affinity overclaims = %#v, want Restricted only", census.AffinityOverclaims)
	}
}

func writeAssetFixture(t *testing.T, root, kind, name, libraryID, source string, styles []string) {
	t.Helper()
	manifest := `{"libraryId":"` + libraryID + `","latest":"1.0.0","designStyles":[`
	for index, style := range styles {
		if index > 0 {
			manifest += `,`
		}
		manifest += `{"styleId":"` + style + `"}`
	}
	manifest += `]}`
	base := filepath.Join("scenarios", "react-component-library", "library", kind, name)
	writeCensusFixture(t, root, filepath.Join(base, "component.json"), manifest)
	writeCensusFixture(t, root, filepath.Join(base, "versions", "1.0.0", name+".tsx"), source)
}

func writeCensusFixture(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
