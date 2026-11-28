package rehearsals

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
)

// Qualification attempt only: an unsupported generic Code/Postgres behavior must
// remain visible as zero-credit evidence rather than be hidden by graph rewriting.
func TestPublic20541IdempotentLeadIntakeQualificationAttempt(t *testing.T) {
	raw := readFixture(t, "public-workflows/20541.json")
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(raw))); got != "94b0d3a178940af0c8ff435b3b7bad8ae89934999fc1394a431ed52866704d8d" {
		t.Fatalf("sanitized fixture changed without provenance update: %s", got)
	}
	if strings.Contains(raw, "Oneclick AI Squad") || strings.Contains(raw, "oneclick-ai") || strings.Contains(raw, `"path": "idempotent-lead"`) || strings.Contains(raw, "httpbin.org") {
		t.Fatal("fixture retained source identity or source webhook path")
	}
	workflow, _, _, err := normalizeWorkflow(raw)
	if err != nil {
		t.Logf("no run credit: generic importer cannot normalize unchanged idempotent lead graph: %v", err)
		return
	}
	if len(workflow.Nodes) != 15 || len(workflow.EntryNodeIDs) != 1 {
		t.Fatalf("unexpected sanitized graph shape: nodes=%d entries=%v", len(workflow.Nodes), workflow.EntryNodeIDs)
	}
	t.Log("import succeeded; bounded profile-driven HTTP and Postgres simulator run remains required before run credit")
}
