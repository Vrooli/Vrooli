package agentapp

import (
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	agentinstall "github.com/vrooli/vrooli/packages/resource-agent-install"
)

func testBuild(t *testing.T, spec Spec) *cliapp.ResourceApp {
	t.Helper()
	spec.Build = Build{Fingerprint: "unknown", Timestamp: "unknown"}
	if spec.Groups == nil {
		spec.Groups = func(agentinstall.Spec) ([]cliapp.SubcommandGroup, error) { return nil, nil }
	}
	app, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	return app
}

func TestNewTakesTheBinaryFromTheHarnessDescriptor(t *testing.T) {
	t.Setenv("HOME", "/home/test")
	var install agentinstall.Spec
	testBuild(t, Spec{Name: "antigravity", DataDir: []string{".gemini"}, UpstreamSourceKind: "npm", UpstreamSourceID: "pkg", Groups: func(got agentinstall.Spec) ([]cliapp.SubcommandGroup, error) {
		install = got
		return nil, nil
	}})
	if install.ResourceName != "antigravity" || install.Binary != "agy" || install.BinDir != "/home/test/.local/bin" || install.DataDir != "/home/test/.gemini" ||
		install.UpstreamSourceKind != "npm" || install.UpstreamSourceID != "pkg" {
		t.Fatalf("install = %+v", install)
	}
}

func TestNewRefusesAResourceWithNoHarness(t *testing.T) {
	_, err := New(Spec{Name: "not-a-harness", Groups: func(agentinstall.Spec) ([]cliapp.SubcommandGroup, error) { return nil, nil }})
	if err == nil || !strings.Contains(err.Error(), "declares no harness") {
		t.Fatalf("err = %v", err)
	}
}

func TestNewReturnsAGroupsError(t *testing.T) {
	_, err := New(Spec{Name: "codex", Groups: func(agentinstall.Spec) ([]cliapp.SubcommandGroup, error) { return nil, errBoom{} }})
	if _, ok := err.(errBoom); !ok {
		t.Fatalf("err = %v", err)
	}
}

type errBoom struct{}

func (errBoom) Error() string { return "boom" }

func TestNewAppliesEveryResourceDifference(t *testing.T) {
	wrapped := false
	app := testBuild(t, Spec{
		Name: "grok", NoModelDiscovery: true,
		WrapInstall:     func(c cliapp.Command) cliapp.Command { wrapped = true; return c },
		InstallCommands: []cliapp.Command{{Name: "extra-install", Description: "x"}},
		CommandGroups:   []cliapp.CommandGroup{{Title: "Execution", Commands: []cliapp.Command{{Name: "run", Description: "r"}}}},
		Groups: func(agentinstall.Spec) ([]cliapp.SubcommandGroup, error) {
			return []cliapp.SubcommandGroup{{Name: "own", Description: "o"}}, nil
		},
	})
	if !wrapped {
		t.Fatal("WrapInstall was not applied")
	}
	var help strings.Builder
	if err := app.CLI.RunWithWriters([]string{"--help"}, &help, &help); err != nil {
		t.Fatal(err)
	}
	text := help.String()
	for _, want := range []string{"extra-install", "Execution", "own", "policy"} {
		if !strings.Contains(text, want) {
			t.Fatalf("help lacks %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "models  ") {
		t.Fatalf("NoModelDiscovery still lists models:\n%s", text)
	}
	if strings.Index(text, "install-direct") > strings.Index(text, "extra-install") {
		t.Fatalf("install commands out of order:\n%s", text)
	}
}
