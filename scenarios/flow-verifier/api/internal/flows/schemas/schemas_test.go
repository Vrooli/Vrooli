package schemas

import (
	"encoding/json"
	"testing"
)

func TestEmbeddedSchemasParseAsJSON(t *testing.T) {
	for name, blob := range map[string][]byte{
		"Temporal":                 Temporal,
		"Navigation":               Navigation,
		"FormalArtifact":           FormalArtifact,
		"NavigationMinimalExample": NavigationMinimalExample,
		"NavigationFullExample":    NavigationFullExample,
	} {
		if len(blob) == 0 {
			t.Fatalf("%s schema is empty — go:embed failed?", name)
		}
		var sink any
		if err := json.Unmarshal(blob, &sink); err != nil {
			t.Fatalf("%s schema is not valid JSON: %v", name, err)
		}
	}
}
