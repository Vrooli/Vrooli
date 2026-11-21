package components_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"react-component-library/internal/components"
	"react-component-library/internal/components/mocks"
)

func TestExperienceClaimPresetsCoverEveryKind(t *testing.T) {
	want := map[string][]string{
		"control": {"tap-target-size", "content-not-clipped", "state-contrast", "padding", "keyboard-reachable", "size-parity"},
		"input":   {"accessible-name", "error-association", "state-contrast", "keyboard-reachable", "font-size"},
		"surface": {"no-document-horizontal-overflow", "spacing", "heading-hierarchy"},
		"overlay": {"focus-contained", "layered-dismissal", "focus-restored", "reading-order"},
		"shell":   {"chrome-pinned", "viewport-fill", "safe-area-tap-targets", "no-document-horizontal-overflow"},
	}
	for kind, claims := range want {
		require.Equal(t, claims, components.ExperienceClaimPreset(kind), "preset for %s", kind)
	}
}

func TestInitializeComponentScaffoldsKindClaims(t *testing.T) {
	root := t.TempDir()
	svc := components.NewServiceWithContent(mocks.NewFakeRepository(), components.NewFSContentStore(root))
	for _, kind := range []string{"control", "input", "surface", "overlay", "shell"} {
		libraryID := "react-component-library:Scaffold" + kind
		created, err := svc.InitializeComponent(context.Background(), components.InitializeComponentInput{
			LibraryID: libraryID, Slug: "Scaffold" + kind, DisplayName: "Scaffold " + kind,
			Kind: kind, InitialVersion: "1.0.0", InitialSource: "export function Example() { return <div />; }",
			ScaffoldExamples: true,
		})
		require.NoError(t, err)
		contractPath := filepath.Join(root, filepath.FromSlash(created.SourcePath))
		contractPath = filepath.Join(filepath.Dir(contractPath), "experience-contract.json")
		raw, err := os.ReadFile(contractPath)
		require.NoError(t, err)
		var document struct {
			Contract struct {
				Provenance string `json:"provenance"`
			} `json:"contract"`
			Claims []struct {
				Type string `json:"type"`
			} `json:"claims"`
		}
		require.NoError(t, json.Unmarshal(raw, &document))
		got := make([]string, 0, len(document.Claims))
		for _, claim := range document.Claims {
			got = append(got, claim.Type)
		}
		require.Equal(t, components.ExperienceClaimPreset(kind), got)
		require.Equal(t, libraryID+"@1.0.0", document.Contract.Provenance)
	}
}
