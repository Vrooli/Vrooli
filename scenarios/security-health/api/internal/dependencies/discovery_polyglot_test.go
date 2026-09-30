package dependencies

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverScenarioPolyglotLockfiles(t *testing.T) {
	root := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("api/go.mod", "module example.com/demo\n\nrequire example.com/dep v1.2.3\n")
	write("web/package-lock.json", `{"packages":{"":{"name":"demo"},"node_modules/left-pad":{"name":"left-pad","version":"1.3.0"}}}`)
	write("tools/yarn.lock", "\"kleur@^4.1.5\":\n  version \"4.1.5\"\n")
	write("tools/bun.lock", `{"lockfileVersion":1,"workspaces":{"":{"dependencies":{"nanoid":["5.0.7",""]}}}}`)
	write("scripts/requirements.txt", "requests==2.32.0\n")
	write("native/Cargo.lock", "[[package]]\nname = \"serde\"\nversion = \"1.0.0\"\n")

	records, err := DiscoverScenario(root, "demo")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]Ecosystem{
		"example.com/dep": EcosystemGo,
		"left-pad":        EcosystemNPM,
		"kleur":           EcosystemYarn,
		"nanoid":          EcosystemBun,
		"requests":        EcosystemPython,
		"serde":           EcosystemRust,
	}
	got := map[string]Ecosystem{}
	for _, record := range records {
		got[record.Name] = record.Ecosystem
	}
	for name, ecosystem := range want {
		if got[name] != ecosystem {
			t.Errorf("%s ecosystem = %q, want %q (records=%+v)", name, got[name], ecosystem, records)
		}
	}
}
