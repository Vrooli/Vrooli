package archtest

import (
	"os"
	"path/filepath"
	"testing"
)

// TestReviewLifecycleHasNoPrivateWorker keeps transitionrunner as the only
// lifecycle owner for review workflows. Review may build snapshots and project
// results, but it must not regain a poller or a domain-owned sweeper.
func TestReviewLifecycleHasNoPrivateWorker(t *testing.T) {
	for _, name := range []string{"polling.go", "sweeper.go"} {
		if _, err := os.Stat(filepath.Join("..", "review", name)); err == nil {
			t.Fatalf("review lifecycle worker %s must not exist; transitionrunner owns workflow progress", name)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat review worker %s: %v", name, err)
		}
	}
}
