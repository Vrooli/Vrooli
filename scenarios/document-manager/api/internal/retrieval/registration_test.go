package retrieval

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"testing"
)

func TestRegisterAcceptsScenarioDescriptorWithoutProviderClient(t *testing.T) { // [REQ:DOC-P0-018]
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".vrooli", "search.json")
	if err := os.WriteFile(path, []byte(`{"name":"document-manager"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	Register(context.Background(), path, log.New(os.Stderr, "", 0))
}
