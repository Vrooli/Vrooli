package deployability

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"testing"
)

// TestCapabilitySchemaEnumsFollowTheSingleVocabulary is the repository
// contract for capability names. Schemas are consumer artifacts; the JSON
// vocabulary is the only authored list. Keeping this as a live-repository
// test means an intentional vocabulary edit fails with the exact drifted
// schema named in the output.
func TestCapabilitySchemaEnumsFollowTheSingleVocabulary(t *testing.T) {
	root := repositoryRootForVocabularyTest(t)
	var vocabulary struct {
		Capabilities []string `json:"capabilities"`
	}
	readJSONForVocabularyTest(t, filepath.Join(root, ".vrooli", "capability-vocabulary.json"), &vocabulary)
	want := append([]string(nil), vocabulary.Capabilities...)
	sort.Strings(want)
	for _, schemaName := range []string{"safeguard.schema.json", "tool.schema.json"} {
		var schema struct {
			Properties map[string]struct {
				Enum []string `json:"enum"`
			} `json:"properties"`
		}
		readJSONForVocabularyTest(t, filepath.Join(root, ".vrooli", "schemas", schemaName), &schema)
		got := append([]string(nil), schema.Properties["capability"].Enum...)
		sort.Strings(got)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("%s capability enum drifted from .vrooli/capability-vocabulary.json: got %v want %v", schemaName, got, want)
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

func readJSONForVocabularyTest(t *testing.T, path string, value any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, value); err != nil {
		t.Fatal(fmt.Errorf("parse %s: %w", path, err))
	}
}
