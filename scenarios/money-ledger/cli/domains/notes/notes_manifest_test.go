package notes

import (
	"os"
	"path/filepath"
	"testing"

	notesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/notes"

	"github.com/vrooli/cli-core/cliapp"
)

// TestNotesManifestCoversNotesService asserts that every RPC declared on
// NotesService either has a manifest command binding or is documented in
// the manifest's `omitted` array with a reason. Catches drift between
// proto and CLI: adding a new RPC without binding/omitting it fails here.
//
// Run by `go test ./cli/domains/notes/...` after scenario instantiation;
// the template itself does not build because of money-ledger placeholders.
func TestNotesManifestCoversNotesService(t *testing.T) {
	manifest := readNotesManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, notesv1.File_money_ledger_v1_notes_notes_proto, "NotesService")
}

func readNotesManifest(t *testing.T) []byte {
	t.Helper()
	// This test file lives at cli/domains/notes/; the manifest lives at cli/.
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
