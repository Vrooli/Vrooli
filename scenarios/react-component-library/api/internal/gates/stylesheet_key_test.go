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

func TestValidateStylesheetKeyUniquenessRejectsDifferentCSSUnderOneKey(t *testing.T) {
	root := t.TempDir()
	source := func(version, css string) string {
		return `/** @libraryId react-component-library:MessageList */
/** @version ` + version + ` */
const styles = ` + "`" + css + "`" + `;
export const MessageList = () => <StyleSheet libraryId="react-component-library:MessageList" version="1.0.0" css={styles} />`
	}
	// 1.0.1 was copied from 1.0.0 and kept its key while its CSS moved on.
	writeStylesheetFixture(t, root, "1.0.0", source("1.0.0", ".list{gap:1px}"))
	writeStylesheetFixture(t, root, "1.0.1", source("1.0.1", ".list{gap:2px}"))

	result, err := ValidateStylesheetKeyUniqueness(Scope{Context: context.Background(), Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) != 2 {
		t.Fatalf("expected both colliding directories to be reported, got %#v", result.Findings)
	}
	for _, finding := range result.Findings {
		if finding.Code != "catalog.stylesheet-key-duplicate" || !strings.Contains(finding.Message, "messagelist-1.0.0") {
			t.Fatalf("unexpected duplicate finding: %#v", finding)
		}
	}
}

// A key shared by versions that inject the same CSS breaks nothing; it is
// recorded, not failed.
func TestValidateStylesheetKeyUniquenessAcceptsIdenticalCSSUnderOneKey(t *testing.T) {
	root := t.TempDir()
	const source = `/** @libraryId react-component-library:MessageList */
/** @version 1.0.0 */
export const styles = ` + "`.list{gap:1px}`" + `;
export const MessageList = () => <StyleSheet libraryId="react-component-library:MessageList" version="1.0.0" css={styles} />`
	writeStylesheetFixture(t, root, "1.0.0", source)
	writeStylesheetFixture(t, root, "1.0.1", strings.Replace(source, "@version 1.0.0", "@version 1.0.1", 1))

	result, err := ValidateStylesheetKeyUniqueness(Scope{Context: context.Background(), Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("identical CSS under one key must not fail, got %#v", result.Findings)
	}
	if len(result.InformationalFindings) != 2 || result.InformationalFindings[0].Code != "catalog.stylesheet-key-shared" {
		t.Fatalf("expected the shared key recorded as information, got %#v", result.InformationalFindings)
	}
}

// Released history keeps whatever key it shipped with; only the mutable draft
// is judged.
func TestValidateStylesheetKeyReportsReleasedLegacyKeysAsInformation(t *testing.T) {
	root := t.TempDir()
	writeStylesheetFixture(t, root, "1.0.0", `/** @libraryId react-component-library:MessageList */
/** @version 1.0.0 */
export const MessageList = () => <StyleSheet name="message-list-1" css=".old{}" />`)
	manifest := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "MessageList", "component.json")
	if err := os.WriteFile(manifest, []byte(`{"catalogId":"react-component-library:MessageList","libraryId":"react-component-library:MessageList","latest":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := ValidateStylesheetKey(Scope{Context: context.Background(), Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) != 0 || len(result.InformationalFindings) != 1 {
		t.Fatalf("expected the released legacy key as information only, got findings %#v informational %#v", result.Findings, result.InformationalFindings)
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

// The active draft must key its stylesheet by its own version; released history
// keeps whatever it shipped with.
func TestValidateStylesheetKeyRejectsADraftThatKeepsItsReleaseVersion(t *testing.T) {
	root := t.TempDir()
	writeStylesheetFixture(t, root, "1.4.3", `/** @libraryId react-component-library:MessageList */
/** @version 1.4.3 */
export const MessageList = () => { useLibraryStyleSheet("react-component-library:MessageList", "1.4.2", css); return null; };`)
	writeStylesheetFixture(t, root, "1.4.4-draft.1", `/** @libraryId react-component-library:MessageList */
/** @version 1.4.4-draft.1 */
export const MessageList = () => { useLibraryStyleSheet("react-component-library:MessageList", "1.4.3", css); return null; };`)
	manifest := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "MessageList", "component.json")
	if err := os.WriteFile(manifest, []byte(`{"catalogId":"react-component-library:MessageList","libraryId":"react-component-library:MessageList","latest":"1.4.3","draft":"1.4.4-draft.1"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := ValidateStylesheetKey(Scope{Context: context.Background(), Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) != 1 || result.Findings[0].Code != "catalog.stylesheet-key-version" || !strings.Contains(result.Findings[0].File, "1.4.4-draft.1") {
		t.Fatalf("expected one version finding on the draft only, got %#v", result.Findings)
	}
}

// Once the latest keys by its own version, a collision left between older
// releases is history: recorded, not failed.
func TestValidateStylesheetKeyUniquenessRecordsCollisionsConfinedToHistory(t *testing.T) {
	root := t.TempDir()
	source := func(version, keyVersion, css string) string {
		return `/** @libraryId react-component-library:MessageList */
/** @version ` + version + ` */
const styles = ` + "`" + css + "`" + `;
export const MessageList = () => <StyleSheet libraryId="react-component-library:MessageList" version="` + keyVersion + `" css={styles} />`
	}
	writeStylesheetFixture(t, root, "1.0.0", source("1.0.0", "1.0.0", ".list{gap:1px}"))
	writeStylesheetFixture(t, root, "1.0.1", source("1.0.1", "1.0.0", ".list{gap:2px}"))
	writeStylesheetFixture(t, root, "1.0.2", source("1.0.2", "1.0.2", ".list{gap:3px}"))
	manifest := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "MessageList", "component.json")
	if err := os.WriteFile(manifest, []byte(`{"catalogId":"react-component-library:MessageList","libraryId":"react-component-library:MessageList","latest":"1.0.2"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := ValidateStylesheetKeyUniqueness(Scope{Context: context.Background(), Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("a collision between older releases must not fail, got %#v", result.Findings)
	}
	if len(result.InformationalFindings) != 2 || result.InformationalFindings[0].Code != "catalog.stylesheet-key-duplicate-history" {
		t.Fatalf("expected the historical collision recorded, got %#v", result.InformationalFindings)
	}
}
