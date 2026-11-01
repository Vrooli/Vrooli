package configcli

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/vrooli/vrooli/resources/opencode/cli/internal/config"
)

func TestProviderModelsReadsChatModelsFromTheOwningResource(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX shell fixture")
	}
	dir := t.TempDir()
	script := `#!/bin/sh
cat <<'JSON'
{"models":[
  {"model":"chat-a","display_name":"Chat A","endpoints":["chat"],"capabilities":["chat"]},
  {"model":"embed","endpoints":["embeddings"],"capabilities":["embeddings"]},
  {"model":"chat-b","capabilities":["chat"]},
  {"model":"chat-a","endpoints":["chat"]},
  {"model":" "}
]}
JSON
`
	path := filepath.Join(dir, "resource-testprovider")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	models, err := providerModels(context.Background(), "testprovider")
	if err != nil {
		t.Fatal(err)
	}
	if want := []config.ProviderModel{{ID: "chat-a", Name: "Chat A"}, {ID: "chat-b"}}; !slices.Equal(models, want) {
		t.Fatalf("models = %v, want %v", models, want)
	}
}

func TestProviderModelsReportsAMissingCatalog(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if _, err := providerModels(context.Background(), "missingprovider"); err == nil {
		t.Fatal("expected a missing provider CLI to be reported")
	}
}
