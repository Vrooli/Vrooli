package gates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func TestValidateStoryCoverageUsesActiveImplementationAndContract(t *testing.T) {
	root := t.TempDir()
	assetDir := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Card")
	versionDir := filepath.Join(assetDir, "versions", "1.0.0")
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"catalogId":"components.card","assetKind":"component","latest":"1.0.0"}`
	if err := os.WriteFile(filepath.Join(assetDir, "component.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	source := `type CardProps = { label?: string; disabled?: boolean; tone?: "quiet" | "loud"; }; export function Card(props: CardProps) { return props.label; }`
	if err := os.WriteFile(filepath.Join(versionDir, "Card.tsx"), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	story := `{"schemaVersion":5,"kind":"component","args":{"fields":[]},"environment":{"fixtures":[]},"stories":[{"id":"default","name":"Default","role":"anatomy","args":{}}]}`
	if err := os.WriteFile(filepath.Join(versionDir, "story.json"), []byte(story), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := ValidateStoryCoverage(Scope{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected != 1 {
		t.Fatalf("inspected = %d, want the active renderable asset", result.Inspected)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected missing axis and boundary stories to be reported")
	}
	for _, finding := range result.Findings {
		if finding.Code != "catalog.story_coverage_missing" {
			t.Fatalf("unexpected finding: %+v", finding)
		}
	}
}

func TestValidateStoryCoverageSkipsOutOfScopeAndUnrenderableAssets(t *testing.T) {
	root := t.TempDir()
	for name, assetKind := range map[string]string{"Card": "component", "Token": "resource"} {
		assetDir := filepath.Join(root, "scenarios", "react-component-library", "library", "components", name)
		versionDir := filepath.Join(assetDir, "versions", "1.0.0")
		if err := os.MkdirAll(versionDir, 0o755); err != nil {
			t.Fatal(err)
		}
		manifest := fmt.Sprintf(`{"catalogId":"components.%s","assetKind":%q,"latest":"1.0.0"}`, strings.ToLower(name), assetKind)
		if err := os.WriteFile(filepath.Join(assetDir, "component.json"), []byte(manifest), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	result, err := ValidateStoryCoverage(Scope{Root: root, Assets: []string{"components.unknown"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected != 0 || len(result.Findings) != 1 || result.Findings[0].Code != "catalog.gate_inspected_nothing" {
		t.Fatalf("out-of-scope assets should report an empty measurement: %+v", result)
	}
}

func TestValidateStoryCoverageAcceptsCompleteCapabilityStories(t *testing.T) {
	root := t.TempDir()
	assetDir := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Badge")
	versionDir := filepath.Join(assetDir, "versions", "1.0.0")
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assetDir, "component.json"), []byte(`{"catalogId":"components.badge","assetKind":"component","latest":"1.0.0"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	source := `type BadgeProps = { label?: string; disabled?: boolean; tone?: "quiet" | "loud"; }; export function Badge(props: BadgeProps) { return props.label; }`
	if err := os.WriteFile(filepath.Join(versionDir, "Badge.tsx"), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	story := `{"schemaVersion":5,"kind":"component","args":{"fields":[]},"environment":{"fixtures":[]},"stories":[{"id":"anatomy","name":"Anatomy","role":"anatomy","args":{}},{"id":"tone","name":"Tone","role":"axis","axis":"tone","args":{},"covers":{"tone":["quiet","loud"]}},{"id":"empty","name":"Empty","role":"boundary","args":{}},{"id":"overflow","name":"Overflow","role":"boundary","args":{}},{"id":"disabled","name":"Disabled","role":"boundary","args":{}}]}`
	if err := os.WriteFile(filepath.Join(versionDir, "story.json"), []byte(story), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := ValidateStoryCoverage(Scope{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected != 1 || len(result.Findings) != 0 {
		t.Fatalf("complete story capability coverage should pass: %+v", result)
	}
}

func TestValidateStoryCoverageSkipsMissingAndMalformedContracts(t *testing.T) {
	root := t.TempDir()
	for name, story := range map[string]string{"Missing": "", "Malformed": "{"} {
		assetDir := filepath.Join(root, "scenarios", "react-component-library", "library", "components", name)
		versionDir := filepath.Join(assetDir, "versions", "1.0.0")
		if err := os.MkdirAll(versionDir, 0o755); err != nil {
			t.Fatal(err)
		}
		manifest := fmt.Sprintf(`{"catalogId":"components.%s","assetKind":"component","latest":"1.0.0"}`, strings.ToLower(name))
		if err := os.WriteFile(filepath.Join(assetDir, "component.json"), []byte(manifest), 0o600); err != nil {
			t.Fatal(err)
		}
		if story != "" {
			if err := os.WriteFile(filepath.Join(versionDir, "story.json"), []byte(story), 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}

	result, err := ValidateStoryCoverage(Scope{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected != 0 || len(result.Findings) != 1 || result.Findings[0].Code != "catalog.gate_inspected_nothing" {
		t.Fatalf("missing and malformed contracts should not be measured: %+v", result)
	}
}
