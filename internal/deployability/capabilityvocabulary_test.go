package deployability

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

//go:generate go run ./cmd/capability-vocabulary --root ../..

// TestCapabilitySchemaEnumsFollowTheSingleVocabulary is the repository
// contract for capability names. Schemas are consumer artifacts; the JSON
// vocabulary is the only authored list. Keeping this as a live-repository
// test means an intentional vocabulary edit fails with the exact drifted
// schema named in the output.
func TestCapabilitySchemaEnumsFollowTheSingleVocabulary(t *testing.T) {
	root := repositoryRootForVocabularyTest(t)
	if err := CheckCapabilitySchemaEnums(root); err != nil {
		t.Fatal(err)
	}
}

func TestVocabularyPoliciesValidateAgainstSchema(t *testing.T) {
	root := repositoryRootForVocabularyTest(t)
	compiler := jsonschema.NewCompiler()
	schemaRaw, err := os.ReadFile(filepath.Join(root, ".vrooli", "schemas", "capability-vocabulary.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := compiler.AddResource("capability-vocabulary.schema.json", bytes.NewReader(schemaRaw)); err != nil {
		t.Fatal(err)
	}
	schema, err := compiler.Compile("capability-vocabulary.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	vocabularyRaw, err := os.ReadFile(filepath.Join(root, ".vrooli", "capability-vocabulary.json"))
	if err != nil {
		t.Fatal(err)
	}
	var document any
	if err := json.Unmarshal(vocabularyRaw, &document); err != nil {
		t.Fatal(err)
	}
	if err := schema.Validate(document); err != nil {
		t.Fatalf("capability vocabulary does not validate against its schema: %v", err)
	}
}

func TestGenerateCapabilitySchemaEnumsRewritesOnlyTheConsumerEnum(t *testing.T) {
	root := t.TempDir()
	writeJSONForVocabularyTest(t, filepath.Join(root, ".vrooli", "capability-vocabulary.json"), `{"capabilities":["zeta","alpha"]}`)
	const stale = "{\n  \"properties\": {\n    \"capability\": {\n      \"enum\": [\"stale\"]\n    },\n    \"other\": {\n      \"description\": \"unchanged\"\n    }\n  }\n}\n"
	for _, name := range []string{"safeguard.schema.json", "tool.schema.json"} {
		writeJSONForVocabularyTest(t, filepath.Join(root, ".vrooli", "schemas", name), stale)
	}
	if err := GenerateCapabilitySchemaEnums(root); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"safeguard.schema.json", "tool.schema.json"} {
		data, err := os.ReadFile(filepath.Join(root, ".vrooli", "schemas", name))
		if err != nil {
			t.Fatal(err)
		}
		want := "{\n  \"properties\": {\n    \"capability\": {\n      \"enum\": [\"alpha\",\"zeta\"]\n    },\n    \"other\": {\n      \"description\": \"unchanged\"\n    }\n  }\n}\n"
		if string(data) != want {
			t.Fatalf("generated %s = %q, want %q", name, data, want)
		}
	}
}

func repositoryRootForVocabularyTest(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtimeCallerForVocabularyTest()
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func runtimeCallerForVocabularyTest() (uintptr, string, int, bool) {
	return runtime.Caller(0)
}

func writeJSONForVocabularyTest(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
