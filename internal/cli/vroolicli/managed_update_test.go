package vroolicli

import (
	"bytes"
	"strings"
	"testing"

	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"
)

func TestManagedUpdateInternalCommandsRequireAnAdmittedRequest(t *testing.T) {
	for _, args := range [][]string{{"managed-update-candidate", "cloudflared", "not-admitted"}, {"managed-update-exec", "cloudflared", "not-admitted", "candidate"}} {
		t.Run(args[0], func(t *testing.T) {
			root := t.TempDir()
			testkitgo.WriteRepoContract(t, root, "scenarios")
			t.Setenv("XDG_STATE_HOME", t.TempDir())
			app := newRuntimeTestApp(t, root)
			home := t.TempDir()
			app.HomeDirFn = func() (string, error) { return home, nil }
			var stdout, stderr bytes.Buffer
			if code := app.Run(args, &stdout, &stderr); code == 0 {
				t.Fatal("internal selectors admitted a new resource operation")
			}
			if !strings.Contains(stderr.String(), "managed update request is not recorded") {
				t.Fatalf("request was not checked by its durable owner: %s", stderr.String())
			}
		})
	}
}
