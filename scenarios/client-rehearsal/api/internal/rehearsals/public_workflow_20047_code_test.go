package rehearsals

import (
	"encoding/json"
	"testing"
)

func TestPublic20047CodeNodesFitBoundedGenericInterpreter(t *testing.T) {
	var exported struct {
		Nodes []struct {
			Type       string         `json:"type"`
			Parameters map[string]any `json:"parameters"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal([]byte(readFixture(t, "public-workflows/20047.json")), &exported); err != nil {
		t.Fatalf("decode sanitized public export: %v", err)
	}
	codeNodes := 0
	for _, node := range exported.Nodes {
		if node.Type != "n8n-nodes-base.code" {
			continue
		}
		codeNodes++
		source, _ := node.Parameters["jsCode"].(string)
		if source == "" {
			t.Fatal("public Code node has no source")
		}
		if _, err := parseSafeCodeMode(source, false, false); err != nil {
			t.Fatalf("public Code node outside generic bounded subset: %v", err)
		}
	}
	if codeNodes != 6 {
		t.Fatalf("sanitized public workflow contains %d Code nodes, want 6", codeNodes)
	}
}
