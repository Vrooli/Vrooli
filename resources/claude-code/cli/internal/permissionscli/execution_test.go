package permissionscli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func TestUnsupportedExecutionDocumentCannotWrite(t *testing.T) {
	h, _, _ := newTestHandlers(t, cliutil.CallerKindHuman)
	path := filepath.Join(t.TempDir(), "execution.json")
	if err := os.WriteFile(path, []byte(`{"schema_version":"v2","rules":[],"execution":{"filesystem":{"workspace":"write"},"network":{"enabled":false},"approval":{"policy":"on-request","reviewer":"user"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, run := range []func([]string) error{h.Plan, h.Reconcile} {
		err := run([]string{"--document", path, "--json"})
		if err == nil || !strings.Contains(err.Error(), "unsupported") {
			t.Fatalf("unsupported execution intent was not rejected: %v", err)
		}
	}
}
