package gates

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"react-component-library/internal/components"
)

func TestRequiredStorySetDerivesCapabilitiesFromProps(t *testing.T) {
	source := `type CardProps = {
		status?: "ready" | "error";
		children?: React.ReactNode;
		disabled?: boolean;
		items?: string[];
		loading?: boolean;
	}
	function Card({ items }: CardProps) { return items.map((item) => item.id); }`
	fields := components.DeriveStoryFields(source)
	required := requiredStorySet("components.card", source, fields)

	for _, want := range []string{
		"anatomy:anatomy",
		"axis:status",
		"boundary:empty",
		"boundary:overflow",
		"boundary:loading",
		"boundary:error",
		"boundary:disabled",
		"boundary:one",
		"boundary:many",
	} {
		found := false
		for _, story := range required {
			if storyKey(story.Role, story.Name) == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("required story set does not contain %q: %#v", want, required)
		}
	}
}

func TestRenderableStoryAssetKindMatchesGateDeclaration(t *testing.T) {
	for _, assetKind := range []string{"primitive", "component", "pattern", "page-template", "navigation"} {
		if !renderableStoryAssetKind(assetKind) {
			t.Errorf("asset kind %q should be included", assetKind)
		}
	}
	for _, assetKind := range []string{"foundation", "runtime-hook", "runtime-service", "resource"} {
		if renderableStoryAssetKind(assetKind) {
			t.Errorf("asset kind %q should be excluded", assetKind)
		}
	}
}

func TestBoundaryAxisFindingsRejectEnumOnlyBoundary(t *testing.T) {
	contract := &components.StoryContract{
		Stories: []components.StoryDefinition{
			{ID: "tone", Role: "axis", Covers: map[string][]json.RawMessage{
				"tone": {json.RawMessage(`"success"`), json.RawMessage(`"error"`)},
			}},
			{ID: "error", Role: "boundary", Args: json.RawMessage(`{"tone":"error"}`)},
		},
	}
	findings := boundaryAxisFindings("/repo", "/repo/story.json", "feedback.card", contract, []components.StoryField{
		{Path: "tone", Kind: components.StoryFieldEnum},
	})
	if len(findings) != 1 || findings[0].Code != "catalog.story_boundary_is_axis_value" {
		t.Fatalf("findings = %#v, want one axis-value finding", findings)
	}
}

func TestStoryGrammarAssetScopeUsesCatalogIdentity(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	result, err := ValidateStoryGrammar(Scope{Root: root, Assets: []string{"react-component-library:Badge"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected != 1 {
		t.Fatalf("asset-scoped story grammar inspected %d files, want the active Badge version", result.Inspected)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("asset-scoped story grammar findings = %#v", result.Findings)
	}
}
