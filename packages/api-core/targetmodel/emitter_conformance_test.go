package targetmodel

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEveryReadinessIdentityHasAnEmitter(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../../.."))
	identities := map[string]string{
		"ReadinessRegistry": ReadinessRegistry, "ReadinessHeartbeat": ReadinessHeartbeat,
		"ReadinessChannel": ReadinessChannel, "ReadinessProtocol": ReadinessProtocol,
		"ReadinessDispatch": ReadinessDispatch, "ReadinessBridgeScope": ReadinessBridgeScope,
		"ReadinessSessionSupport": ReadinessSessionSupport,
	}
	seen := make(map[string]bool, len(identities))
	// Emitters live in scenario consumers. Restrict the walk to Go source under
	// scenarios so this check stays fast and does not parse unrelated generated
	// or vendored modules in the repository.
	err := filepath.WalkDir(filepath.Join(root, "scenarios"), func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "model.go") || strings.HasSuffix(entry.Name(), "emitter_conformance_test.go") {
			return walkErr
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for name := range identities {
			if strings.Contains(string(raw), "targetmodel."+name) {
				seen[name] = true
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for name, identity := range identities {
		if !seen[name] {
			t.Errorf("readiness identity %q (%s) has no emitter", name, identity)
		}
	}
}
