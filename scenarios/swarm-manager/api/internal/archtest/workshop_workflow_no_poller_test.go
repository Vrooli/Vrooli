package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkshopWorkflowHasNoSwarmPollingWorker(t *testing.T) {
	path := filepath.Join("..", "backlog", "workshop_workflow.go")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	for _, forbidden := range []string{"StartWorkshopWorkflowWorker", "time.NewTicker"} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("workshop workflow must not retain a Swarm polling loop (%s)", forbidden)
		}
	}
	if !strings.Contains(source, "ProcessWorkshopWorkflows") {
		t.Fatal("restart reconciliation must retain the bounded correlation scan")
	}
}
