package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCaptureClassificationUsesSharedTransitionRunner(t *testing.T) {
	path := filepath.Join("..", "captures", "classify.go")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	for _, forbidden := range []string{"SpawnBacklog(", "CreateRun(", "ContinueRun(", "ReadSkill("} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("capture classification must not use raw agent integration %q", forbidden)
		}
	}
	for _, forbidden := range []string{"StartWorkflow(", "CollectWorkflow(", "validateClassificationInput("} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("capture classification must delegate workflow mechanics to transitionrunner, found %q", forbidden)
		}
	}
	for _, required := range []string{"transitionRunner.Start(", "transitionRunner.Apply("} {
		if !strings.Contains(source, required) {
			t.Fatalf("capture classification must use shared transition runner %q", required)
		}
	}
}
