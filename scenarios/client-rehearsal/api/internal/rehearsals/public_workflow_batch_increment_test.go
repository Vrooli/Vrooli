package rehearsals

import (
	"testing"
)

func TestPublicBatchIncrementGraphsImport(t *testing.T) {
	for _, id := range []string{"15238", "17831", "16904", "6338"} {
		t.Run(id, func(t *testing.T) {
			workflow, _, _, err := normalizeWorkflow(readFixture(t, "public-workflows/"+id+".json"))
			if err != nil {
				t.Fatalf("normalize public workflow %s: %v", id, err)
			}
			for _, node := range workflow.Nodes {
				if node.Request != nil {
					t.Logf("%s: %s", node.Name, node.Request.Operation)
				}
			}
		})
	}
}
