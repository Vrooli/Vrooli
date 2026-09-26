package retirement

import (
	"encoding/json"
	"os"
	"path/filepath"
	"react-component-library/internal/catalogcoverage"
	"react-component-library/internal/components"
	"testing"
)

func TestCatalogPreflightPreservesRequiredAndSuggestedConsumers(t *testing.T) {
	assets := []catalogcoverage.Asset{{ID: "retired", Kind: "component", Maturity: "deprecated", DeclarationPath: "retired.json"}, {ID: "required", Kind: "component", Requires: []string{"retired"}}, {ID: "suggested", Kind: "component", Suggests: []string{"retired"}}}
	result, err := checkCatalog(assets, "retired")
	if err != nil {
		t.Fatal(err)
	}
	if result.Ready() || len(result.RequiredBy) != 1 || result.RequiredBy[0] != "required" || len(result.SuggestedBy) != 1 || result.SuggestedBy[0] != "suggested" {
		t.Fatalf("missing blocker: %+v", result)
	}
	result, err = checkCatalog(assets[:1], "retired")
	if err != nil || len(result.RequiredBy) != 0 || len(result.SuggestedBy) != 0 {
		t.Fatalf("unreferenced retired asset: %+v %v", result, err)
	}
	assets[0].Maturity = "production-ready"
	if _, err = checkCatalog(assets[:1], "retired"); err == nil {
		t.Fatal("active asset accepted")
	}
	if _, err = checkCatalog(assets, "unknown"); err == nil {
		t.Fatal("unknown identity accepted")
	}
}

func TestIncompletePreflightCannotBeReady(t *testing.T) {
	if (PreflightResult{}).Ready() {
		t.Fatal("zero result claims readiness")
	}
	if (PreflightResult{CatalogID: "known"}).Ready() {
		t.Fatal("catalog-only result claims readiness")
	}
	if !(PreflightResult{Completed: true}).Ready() {
		t.Fatal("completed blocker-free result rejected")
	}
}

func TestCurrentPlaceholderRetirement(t *testing.T) {
	root := os.Getenv("RCL_RETIREMENT_REPO")
	if root == "" {
		t.Skip("opt-in current retirement evidence")
	}
	for _, name := range []string{"TopBar", "PageFrame", "DashboardPage", "DetailPage", "CollectionPage"} {
		t.Run(name, func(t *testing.T) {
			c := components.Component{LibraryID: "react-component-library:" + name}
			archive := archivePaths(root, c)
			if _, err := os.Stat(filepath.Join(libraryRoot(root), "components", name)); !os.IsNotExist(err) {
				t.Fatalf("active source remains: %v", err)
			}
			data, err := os.ReadFile(archive.SourceArchivePath + ".receipt.json")
			if err != nil {
				t.Fatal(err)
			}
			var receipt Result
			if err := json.Unmarshal(data, &receipt); err != nil {
				t.Fatal(err)
			}
			if !receipt.Retired || receipt.LibraryID != c.LibraryID {
				t.Fatalf("invalid receipt: %+v", receipt)
			}
			for _, path := range []string{receipt.Archive.SnapshotPath, receipt.CatalogArchivePath, filepath.Join(archive.SourceArchivePath, "component.json")} {
				if _, err := os.Stat(path); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
