package graph

import (
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestModeForProfilePreservesFullCompatibilityMode(t *testing.T) {
	if got, want := modeForProfile(ExtractionProfileFull), loadMode; got != want {
		t.Fatalf("full mode = %v, want %v", got, want)
	}
	if got, want := modeForProfile(""), loadMode; got != want {
		t.Fatalf("empty profile mode = %v, want full mode %v", got, want)
	}
}

func TestModeForProfileStructuralOmitsTypeChecking(t *testing.T) {
	mode := modeForProfile(ExtractionProfileStructural)
	for _, omitted := range []packages.LoadMode{
		packages.NeedTypes,
		packages.NeedTypesInfo,
		packages.NeedDeps,
	} {
		if mode&omitted != 0 {
			t.Fatalf("structural mode %v unexpectedly includes %v", mode, omitted)
		}
	}
	for _, required := range []packages.LoadMode{
		packages.NeedFiles,
		packages.NeedImports,
		packages.NeedSyntax,
		packages.NeedName,
	} {
		if mode&required == 0 {
			t.Fatalf("structural mode %v omits required bit %v", mode, required)
		}
	}
}

func TestModeForProfileSemanticRetainsTypeCheckingWithoutTests(t *testing.T) {
	if got, want := modeForProfile(ExtractionProfileSemantic), loadMode; got != want {
		t.Fatalf("semantic mode = %v, want %v", got, want)
	}
	if opts := (LoadOptions{Profile: ExtractionProfileSemantic}); opts.Profile.normalized() == ExtractionProfileFull {
		t.Fatal("semantic profile normalized to full")
	}
}
