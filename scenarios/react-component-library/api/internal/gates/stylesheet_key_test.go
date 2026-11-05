package gates

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeStylesheetFixture(t *testing.T, root, version, source string) {
	t.Helper()
	asset := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "MessageList")
	versionDir := filepath.Join(asset, "versions", version)
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(asset, "component.json"), []byte(`{"catalogId":"react-component-library:MessageList","libraryId":"react-component-library:MessageList"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(versionDir, "MessageList.tsx"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestValidateStylesheetKeyRejectsLegacyName(t *testing.T) {
	root := t.TempDir()
	writeStylesheetFixture(t, root, "1.0.0", `/** @libraryId react-component-library:MessageList */
/** @version 1.0.0 */
export const MessageList = () => <StyleSheet name="message-list-1" css=".old{}" />`)

	result, err := ValidateStylesheetKey(Scope{Context: context.Background(), Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) != 1 || result.Findings[0].Code != "catalog.stylesheet-key" {
		t.Fatalf("expected literal stylesheet finding, got %#v", result.Findings)
	}
}

func TestValidateStylesheetKeyUniquenessRejectsDuplicateIdentity(t *testing.T) {
	root := t.TempDir()
	const source = `/** @libraryId react-component-library:MessageList */
/** @version 1.0.0 */
export const MessageList = () => <StyleSheet libraryId="react-component-library:MessageList" version="1.0.0" css=".ok{}" />`
	writeStylesheetFixture(t, root, "1.0.0", source)
	writeStylesheetFixture(t, root, "1.0.0-copy", source)

	result, err := ValidateStylesheetKeyUniqueness(Scope{Context: context.Background(), Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) != 2 {
		t.Fatalf("expected both duplicate directories to be reported, got %#v", result.Findings)
	}
	for _, finding := range result.Findings {
		if finding.Code != "catalog.stylesheet-key-duplicate" || !strings.Contains(finding.Message, "messagelist") {
			t.Fatalf("unexpected duplicate finding: %#v", finding)
		}
	}
}

func TestValidateStylesheetKeyAcceptsGovernedDraftIdentity(t *testing.T) {
	root := t.TempDir()
	writeStylesheetFixture(t, root, "1.3.1-draft.1", `/** @libraryId react-component-library:MessageList */
/** @version 1.3.1-draft.1 */
useLibraryStyleSheet("react-component-library:MessageList", "1.3.1-draft.1", styles);`)
	result, err := ValidateStylesheetKey(Scope{Context: context.Background(), Root: root})
	if err != nil || len(result.Findings) != 0 {
		t.Fatalf("valid draft misclassified: %+v %v", result, err)
	}
}
