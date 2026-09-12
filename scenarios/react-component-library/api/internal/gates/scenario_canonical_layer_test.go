package gates

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateScenarioCanonicalLayer(t *testing.T) {
	root := t.TempDir()
	fixtures := map[string]string{
		"mounted": `import { Button } from "@vrooli/react-component-library/Button/1"; import { BaseStyles } from "@vrooli/react-component-library/BaseStyles/1"; export const Root = () => <><BaseStyles /><Button /></>;`,
		"missing": `import { Button } from "@vrooli/react-component-library/Button/1"; export const Root = () => <Button />;`,
	}
	for scenario, source := range fixtures {
		path := filepath.Join(root, "scenarios", scenario, "ui", "src", "main.tsx")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	result, err := ValidateScenarioCanonicalLayer(Scope{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected != 2 || len(result.Findings) != 1 || result.Findings[0].File != "scenarios/missing/ui/src/main.tsx" {
		t.Fatalf("unexpected result: inspected=%d findings=%#v", result.Inspected, result.Findings)
	}
}
