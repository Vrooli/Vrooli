package clifreshness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

const baseSurface = `{"groups":[{"name":"agent","commands":[{"name":"launch","flags":[{"name":"claim"}]}]}]}`

func parseOrFail(t *testing.T, body string) surface {
	t.Helper()
	parsed, err := parseSurface([]byte(body))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return parsed
}

func TestSurfaceIgnoresDescriptionsAndOrdering(t *testing.T) {
	left := parseOrFail(t, `{"groups":[
		{"name":"agent","commands":[
			{"name":"launch","flags":[{"name":"claim"},{"name":"quiet"}]},
			{"name":"status"}
		]}
	]}`)
	right := parseOrFail(t, `{"groups":[
		{"name":"agent","description":"changed","commands":[
			{"name":"status","description":"changed"},
			{"name":"launch","description":"changed","flags":[{"name":"quiet","description":"changed"},{"name":"claim"}]}
		]}
	]}`)
	if !equalSurfaces(left, right) {
		t.Fatal("prose, command order or flag order changed the capability surface")
	}
}

func TestSurfaceTracksCommandsAndFlags(t *testing.T) {
	base := parseOrFail(t, baseSurface)
	for name, body := range map[string]string{
		"new command":  `{"groups":[{"name":"agent","commands":[{"name":"launch","flags":[{"name":"claim"}]},{"name":"status"}]}]}`,
		"new flag":     `{"groups":[{"name":"agent","commands":[{"name":"launch","flags":[{"name":"claim"},{"name":"quiet"}]}]}]}`,
		"removed flag": `{"groups":[{"name":"agent","commands":[{"name":"launch","flags":[]}]}]}`,
		"renamed":      `{"groups":[{"name":"agent","commands":[{"name":"start","flags":[{"name":"claim"}]}]}]}`,
		"new group":    `{"groups":[{"name":"agent","commands":[{"name":"launch","flags":[{"name":"claim"}]}]},{"name":"new","commands":[{"name":"thing"}]}]}`,
		"nested group": `{"groups":[{"name":"agent","groups":[{"name":"run","commands":[{"name":"launch","flags":[{"name":"claim"}]}]}]}]}`,
		"empty":        `{"groups":[]}`,
	} {
		t.Run(name, func(t *testing.T) {
			if equalSurfaces(base, parseOrFail(t, body)) {
				t.Fatalf("%s did not change the capability surface", name)
			}
		})
	}
}

func TestSurfaceHandlesNestedGroups(t *testing.T) {
	parsed := parseOrFail(t, `{"groups":[{"name":"a","groups":[{"name":"b","commands":[{"name":"c","flags":[{"name":"d"}]}]}]}]}`)
	if len(parsed.Commands) != 1 || parsed.Commands[0].Path != "a b c" {
		t.Fatalf("nested path = %+v", parsed.Commands)
	}
	if len(parsed.Commands[0].Flags) != 1 || parsed.Commands[0].Flags[0] != "d" {
		t.Fatalf("nested flags = %+v", parsed.Commands[0].Flags)
	}
}

func TestCommandSurfaceBehindComparesAgainstTheCheckout(t *testing.T) {
	repositoryRoot := filepath.Dir(filepath.Dir(mustAbs(t, ".")))
	source, err := os.ReadFile(filepath.Join(repositoryRoot, "cli", "manifest.json"))
	if err != nil {
		t.Skip("not running inside the repository checkout")
	}
	root := t.TempDir()
	writeManifest(t, root, string(source))
	behind, err := CommandSurfaceBehind(root)
	if err != nil {
		t.Fatal(err)
	}
	if behind {
		t.Fatal("the checkout's own manifest was reported as newer than the build")
	}

	// Adding one command makes the checkout newer than this build.
	var document map[string]any
	if err := json.Unmarshal(source, &document); err != nil {
		t.Fatal(err)
	}
	groups, _ := document["groups"].([]any)
	if len(groups) == 0 {
		t.Skip("manifest has no groups")
	}
	first, _ := groups[0].(map[string]any)
	commands, _ := first["commands"].([]any)
	first["commands"] = append(commands, map[string]any{"name": "brand-new-command"})
	modified, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	writeManifest(t, root, string(modified))
	if behind, err := CommandSurfaceBehind(root); err != nil || !behind {
		t.Fatalf("an added command was not detected: behind=%t err=%v", behind, err)
	}
}

func TestCommandSurfaceRefusesAnUnreadableManifest(t *testing.T) {
	root := t.TempDir()
	if _, err := CommandSurfaceBehind(root); err == nil {
		t.Fatal("a missing manifest was reported as an empty command surface")
	}
	writeManifest(t, root, `{"groups":`)
	if _, err := CommandSurfaceBehind(root); err == nil {
		t.Fatal("an invalid manifest was accepted")
	}
}

// The checked-in checkout compiles its own CLI, so the embedded command
// surface must already match the authored one. This is the invariant a refresh
// restores.
func TestThisCheckoutDoesNotNeedARefresh(t *testing.T) {
	root := filepath.Dir(filepath.Dir(mustAbs(t, ".")))
	if _, err := os.Stat(filepath.Join(root, "cli", "manifest.json")); err != nil {
		t.Skip("not running inside the repository checkout")
	}
	behind, err := CommandSurfaceBehind(root)
	if err != nil {
		t.Fatal(err)
	}
	if behind {
		t.Fatal("this test binary's embedded command surface is older than cli/manifest.json; rebuild the tests")
	}
}

func writeManifest(t *testing.T, root, body string) {
	t.Helper()
	path := filepath.Join(root, "cli", "manifest.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustAbs(t *testing.T, path string) string {
	t.Helper()
	absolute, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	return absolute
}
