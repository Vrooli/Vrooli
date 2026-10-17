package installgateway

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/tools/hyperframes"
)

func hostToolFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join(root, "internal", "tools", "hyperframes")
	if err := os.MkdirAll(source, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"package.json", "package-lock.json"} {
		b, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "internal", "tools", "hyperframes", name))
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(source, name), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestHostToolResolvesOnlyDeclaredFrozenClosure(t *testing.T) {
	root := hostToolFixture(t)
	r, err := Resolve(root, "hyperframes", "host-tool", "npm", "hyperframes", hyperframes.Version)
	if err != nil {
		t.Fatal(err)
	}
	if !r.HostTool || !r.Profile.FrozenLockfile || !r.Profile.ScriptsDisabled || r.Profile.Governance != "scenario-dependency-analyzer" {
		t.Fatalf("unsafe profile %+v", r)
	}
	if r.Command() != "npm ci --ignore-scripts --no-audit --no-fund" {
		t.Fatalf("command %s", r.Command())
	}
	if err = ApplyExactVersion(&r, hyperframes.Version); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(r.Command(), "save-exact") {
		t.Fatal("mutating exact-save flag appended to frozen reproduction")
	}
	for _, change := range []struct{ owner, pkg, version string }{{"../escape", "hyperframes", hyperframes.Version}, {"hyperframes", "other", hyperframes.Version}, {"hyperframes", "hyperframes", "0.8.45"}} {
		if _, err = Resolve(root, change.owner, "host-tool", "npm", change.pkg, change.version); err == nil {
			t.Fatalf("accepted %+v", change)
		}
	}
	if err = os.WriteFile(filepath.Join(root, "internal", "tools", "hyperframes", "package-lock.json"), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = Resolve(root, "hyperframes", "host-tool", "npm", "hyperframes", hyperframes.Version); err == nil {
		t.Fatal("unrecognized lock accepted")
	}
}
