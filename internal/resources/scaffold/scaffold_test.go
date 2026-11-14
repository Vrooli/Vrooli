package scaffold

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	resources "github.com/vrooli/vrooli/internal/resources"
)

func TestScaffoldProducesAllSupportedArchetypes(t *testing.T) {
	for _, driver := range []string{"managed-service", "external-cli", "cloud-api", "native-cli"} {
		t.Run(driver, func(t *testing.T) {
			root := t.TempDir()
			if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
				t.Fatal(err)
			}
			contract := map[string]any{"dependencies": map[string]any{"resources": map[string]any{}}}
			data, err := json.Marshal(contract)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, ".vrooli", "service.json"), data, 0o644); err != nil {
				t.Fatal(err)
			}

			name := "fixture-" + driver
			if err := resources.NewController(root, t.TempDir()).Scaffold(name, driver); err != nil {
				t.Fatal(err)
			}
			resourceRoot := filepath.Join(root, "resources", name)
			for _, path := range []string{"resource.json", "README.md", "Makefile", "cli/main.go", "cli/go.mod", "docs/README.md"} {
				if _, err := os.Stat(filepath.Join(resourceRoot, path)); err != nil {
					t.Errorf("missing generated %s: %v", path, err)
				}
			}
		})
	}
}
