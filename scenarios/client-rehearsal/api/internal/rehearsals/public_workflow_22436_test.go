package rehearsals

import (
	"strings"
	"testing"
)

func TestPublic22436KPIFileWriterIsRetainedWithoutSimulatorCredit(t *testing.T) {
	workflowJSON := readFixture(t, "public-workflows/22436.json")
	if !strings.Contains(workflowJSON, `"active": false`) {
		t.Fatal("public file-writer candidate must remain inactive")
	}
	for _, marker := range []string{"fs.readFileSync", "fs.writeFileSync", "fs.appendFileSync", "fs.renameSync"} {
		if !strings.Contains(workflowJSON, marker) {
			t.Fatalf("sanitized public source no longer demonstrates expected file behavior %q", marker)
		}
	}
	if _, _, _, err := normalizeWorkflow(workflowJSON); err == nil {
		t.Fatal("public file-writer workflow unexpectedly imported into the no-file-effect simulator")
	} else {
		t.Logf("candidate receives no run credit: bounded importer fails closed: %v", err)
	}
}
