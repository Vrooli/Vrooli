package cleanup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

// [REQ:CLN-P0-005]
func TestRegisterBuildsCleanupCommandsFromManifest(t *testing.T) {
	t.Parallel()

	group, err := Register(&cliapp.ScenarioApp{}, readManifest(t))
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if group.Name != GroupName {
		t.Fatalf("group name = %q, want %q", group.Name, GroupName)
	}
	if got := len(group.Subcommands); got != 7 {
		t.Fatalf("cleanup subcommand count = %d, want 7", got)
	}

	names := map[string]bool{}
	for _, cmd := range group.Subcommands {
		names[cmd.Name] = true
	}
	for _, want := range []string{"providers", "policy", "set-profile", "plan", "apply", "report-pressure", "audit"} {
		if !names[want] {
			t.Fatalf("cleanup command %q missing from %#v", want, names)
		}
	}
}

func readManifest(t *testing.T) []byte {
	t.Helper()
	bytes, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	return bytes
}
