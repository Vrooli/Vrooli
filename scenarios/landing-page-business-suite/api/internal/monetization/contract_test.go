package monetization

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// TestPaidFeaturesExampleMatchesSchema keeps the operator-facing contract
// example executable: a schema change that the example does not understand
// must fail this test before the two surfaces drift.
func TestPaidFeaturesExampleMatchesSchema(t *testing.T) {
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("resolve monetization test workspace: %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(workingDir, "../../../../../"))
	schemaBytes, err := os.ReadFile(filepath.Join(repoRoot, ".vrooli", "schemas", "monetization.schema.json"))
	if err != nil {
		t.Fatalf("read monetization schema: %v", err)
	}
	schema, err := jsonschema.CompileString("monetization.schema.json", string(schemaBytes))
	if err != nil {
		t.Fatalf("compile monetization schema: %v", err)
	}
	docBytes, err := os.ReadFile(filepath.Join(repoRoot, "docs", "concepts", "PAID_FEATURES.md"))
	if err != nil {
		t.Fatalf("read paid-features contract: %v", err)
	}
	blocks := regexp.MustCompile("(?s)```jsonc?\\s*(.*?)```").FindAllSubmatch(docBytes, -1)
	if len(blocks) == 0 {
		t.Fatal("paid-features contract has no JSON example")
	}
	var example any
	if err := json.Unmarshal(blocks[0][1], &example); err != nil {
		t.Fatalf("decode paid-features JSON example: %v", err)
	}
	if err := schema.Validate(example); err != nil {
		t.Fatalf("paid-features JSON example does not match monetization schema: %v", err)
	}
	for _, scenario := range []string{"browser-automation-studio", "landing-page-business-suite", "web-console"} {
		manifestBytes, err := os.ReadFile(filepath.Join(repoRoot, "scenarios", scenario, ".vrooli", "monetization.json"))
		if err != nil {
			t.Fatalf("read %s monetization manifest: %v", scenario, err)
		}
		var manifest any
		if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
			t.Fatalf("decode %s monetization manifest: %v", scenario, err)
		}
		if err := schema.Validate(manifest); err != nil {
			t.Fatalf("%s monetization manifest does not match schema: %v", scenario, err)
		}
	}
}
