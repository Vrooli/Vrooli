package adoptions_test

import (
	"strings"
	"testing"

	"react-component-library/internal/adoptions"
)

func TestTranslateDesignTokensUsesConsumerVocabulary(t *testing.T) {
	body, translations, err := adoptions.TranslateDesignTokens("bg-app-primary text-app-muted-foreground border-app-border", "wc")
	if err != nil {
		t.Fatal(err)
	}
	if body != "bg-wc-accent text-wc-text-muted border-wc-default" {
		t.Fatalf("translated body = %q", body)
	}
	if len(translations) != 3 {
		t.Fatalf("translations = %#v", translations)
	}
}

func TestTranslateDesignTokensFailsForUnknownNamespace(t *testing.T) {
	_, _, err := adoptions.TranslateDesignTokens("bg-app-primary", "unknown")
	if err == nil || !strings.Contains(err.Error(), "not governed") {
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
