package credentials

import (
	"testing"

	"github.com/vrooli/vrooli/internal/cliout"
)

// The CLI declares `--format text` as the default for every credentials verb
// (cli/manifest.json), so "text" must reach cliout's human form rather than a
// usage error. `vrooli credentials store status` refused its own default with
// "store format must be text or json" on minimouse and on this host
// (2026-09-15), leaving the credential store unreadable from the bare command.
func TestNormalizeOutputFormatAcceptsTheCLIsDefaultText(t *testing.T) {
	for _, format := range []string{"", "text", "  text  ", "human"} {
		got, ok := normalizeOutputFormat(format)
		if !ok {
			t.Fatalf("normalizeOutputFormat(%q) refused a renderable format", format)
		}
		if got != string(cliout.FormatHuman) {
			t.Fatalf("normalizeOutputFormat(%q) = %q, want %q", format, got, cliout.FormatHuman)
		}
	}

	got, ok := normalizeOutputFormat("json")
	if !ok || got != string(cliout.FormatJSON) {
		t.Fatalf("normalizeOutputFormat(\"json\") = %q, %v", got, ok)
	}

	if _, ok := normalizeOutputFormat("xml"); ok {
		t.Fatal("an unsupported format must still be refused")
	}
}

// Each credentials verb reimplemented the same check, and every one of them
// promised "text" in its error message while rejecting it. They are pinned
// together so a future verb cannot regress only one of them.
func TestEveryCredentialsFormatValidatorAcceptsText(t *testing.T) {
	for _, format := range []string{"", "text", "human", "json"} {
		if _, err := storeFormat(format); err != nil {
			t.Fatalf("storeFormat(%q) = %v", format, err)
		}
		if _, _, err := normalizeKeyringOptions(KeyringOptions{Format: format}); err != nil {
			t.Fatalf("normalizeKeyringOptions(%q) = %v", format, err)
		}
	}

	if _, err := storeFormat("xml"); err == nil {
		t.Fatal("storeFormat must still refuse an unsupported format")
	}
	if _, _, err := normalizeKeyringOptions(KeyringOptions{Format: "xml"}); err == nil {
		t.Fatal("normalizeKeyringOptions must still refuse an unsupported format")
	}
}

// The keyring path is trimmed independently of the format, so a normalized
// format must not disturb it.
func TestNormalizeKeyringOptionsKeepsTheTrimmedPath(t *testing.T) {
	path, format, err := normalizeKeyringOptions(KeyringOptions{Path: "  /tmp/store.json  ", Format: "text"})
	if err != nil {
		t.Fatalf("normalizeKeyringOptions = %v", err)
	}
	if path != "/tmp/store.json" {
		t.Fatalf("path = %q", path)
	}
	if format != string(cliout.FormatHuman) {
		t.Fatalf("format = %q", format)
	}
}
