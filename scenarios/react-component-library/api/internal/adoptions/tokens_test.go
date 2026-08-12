package adoptions_test

import (
	"context"
	"strings"
	"testing"

	"react-component-library/internal/adoptions"
)

func TestTranslateDesignTokensUsesConsumerVocabulary(t *testing.T) {
	mapping := adoptions.TokenMapping{Namespace: "wc", Roles: map[string]adoptions.TokenRoleMapping{
		"app-primary":          {Target: "wc-accent-active", CSSVariable: "--wc-accent-active"},
		"app-muted-foreground": {Target: "wc-text-muted", CSSVariable: "--wc-text-muted"},
		"app-border":           {Target: "wc-default", CSSVariable: "--wc-border-default"},
	}}
	body, translations, err := adoptions.TranslateDesignTokens("bg-app-primary text-app-muted-foreground border-app-border", "wc", mapping)
	if err != nil {
		t.Fatal(err)
	}
	if body != "bg-wc-accent-active text-wc-text-muted border-wc-default" {
		t.Fatalf("translated body = %q", body)
	}
	if len(translations) != 3 {
		t.Fatalf("translations = %#v", translations)
	}
}

func TestTranslateDesignTokensFailsForUnknownNamespace(t *testing.T) {
	_, _, err := adoptions.TranslateDesignTokens("bg-app-primary", "unknown", adoptions.TokenMapping{Namespace: "unknown", Roles: map[string]adoptions.TokenRoleMapping{
		"app-primary": {Target: "unknown-primary", CSSVariable: "--unknown-primary"},
	}})
	if err == nil || !strings.Contains(err.Error(), "not governed") {
		t.Fatalf("err = %v", err)
	}
}

func TestValidateTokenMappingRejectsMissingRole(t *testing.T) {
	err := adoptions.ValidateTokenMappingInjective(adoptions.TokenMapping{
		Namespace: "wc",
		Roles: map[string]adoptions.TokenRoleMapping{
			"app-primary": {Target: "wc-accent-active", CSSVariable: "--wc-accent-active"},
		},
	}, []string{"app-primary", "app-danger"})
	if err == nil || !strings.Contains(err.Error(), "not governed") {
		t.Fatalf("err = %v", err)
	}
}

func TestValidateTokenMappingRejectsLowContrast(t *testing.T) {
	err := adoptions.ValidateTokenMappingInjective(adoptions.TokenMapping{
		Namespace: "wc",
		Roles: map[string]adoptions.TokenRoleMapping{
			"app-primary": {Target: "wc-accent-active", CSSVariable: "--wc-accent-active"},
			"app-danger":  {Target: "wc-error-surface", CSSVariable: "--wc-error-surface"},
			"app-info":    {Target: "wc-accent", CSSVariable: "--wc-accent"},
			"app-warning": {Target: "wc-accent-border", CSSVariable: "--wc-accent-border"},
		},
		ContrastFloor: 4.5,
		ContrastPairs: []adoptions.TokenContrastPair{{Foreground: "app-primary", Background: "app-danger", Ratio: 2.1}},
	}, []string{"app-primary", "app-danger", "app-info", "app-warning"})
	if err == nil || !strings.Contains(err.Error(), "below floor") {
		t.Fatalf("err = %v", err)
	}
}

func TestFSScenarioFileReaderLoadsScenarioOwnedMapping(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "web-console/ui/token-map.json", `{
  "namespace": "wc",
  "roles": {
    "app-danger": {"target": "wc-error-surface", "css_variable": "--wc-error-surface"},
    "app-info": {"target": "wc-accent", "css_variable": "--wc-accent"},
    "app-primary": {"target": "wc-accent-active", "css_variable": "--wc-accent-active"},
    "app-warning": {"target": "wc-accent-border", "css_variable": "--wc-accent-border"}
  },
  "contrast_floor": 4.5,
  "contrast_pairs": [{"foreground": "app-primary", "background": "app-danger", "ratio": 4.5}]
}`)

	mapping, err := adoptions.NewFSScenarioFileReader(root).TokenMapping(context.Background(), "web-console")
	if err != nil {
		t.Fatal(err)
	}
	if mapping.Namespace != "wc" || mapping.Roles["app-primary"].CSSVariable != "--wc-accent-active" {
		t.Fatalf("mapping = %#v", mapping)
	}
}

func TestFSScenarioFileReaderRejectsMissingMapping(t *testing.T) {
	_, err := adoptions.NewFSScenarioFileReader(t.TempDir()).TokenMapping(context.Background(), "web-console")
	if err == nil || !strings.Contains(err.Error(), "mapping missing") {
		t.Fatalf("err = %v", err)
	}
}

func TestCrossBoundaryImportRequiresProvenanceAndLibraryPath(t *testing.T) {
	if !adoptions.CrossBoundaryImport("@vrooliComponentSource react-component-library:Button\nexport {Button} from '../../../../react-component-library/library/components/Button';") {
		t.Fatal("expected counterfeit adoption to be rejected")
	}
	if adoptions.CrossBoundaryImport("export {Button} from '../../../../react-component-library/library/components/Button';") {
		t.Fatal("unprovenanced source should not be classified as an adoption")
	}
}
