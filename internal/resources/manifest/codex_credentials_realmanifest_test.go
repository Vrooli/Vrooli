package manifest_test

import (
	"path/filepath"
	"testing"

	manifest "github.com/vrooli/vrooli/internal/resources/manifest"
)

// TestAgentRunnerCredentialsAreOptional ensures an Agent Manager consumer can
// start even when no runner is ready. Authentication and executable readiness
// belong to runner probes and run selection, not process-environment assembly.
func TestAgentRunnerCredentialsAreOptional(t *testing.T) {
	repoRoot := findRepoRoot(t)
	for _, name := range []string{"claude-code", "codex", "grok", "opencode"} {
		t.Run(name, func(t *testing.T) {
			resource, err := manifest.Load(filepath.Join(repoRoot, "resources", name, "resource.json"))
			if err != nil {
				t.Fatalf("load %s resource manifest: %v", name, err)
			}
			for _, credential := range resource.Credentials.All() {
				if credential.Required {
					t.Fatalf("%s credential %s must be optional: runner readiness is assessed after scenario startup", name, credential.Env)
				}
			}
		})
	}
}
