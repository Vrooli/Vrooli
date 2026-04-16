package scenarioexec

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/scenario"
	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
	testscenario "github.com/vrooli/vrooli/packages/testkit-go/scenariofixture"
)

func TestWriterSupportsStreamingRejectsBuffer(t *testing.T) {
	if WriterSupportsStreaming(&bytes.Buffer{}) {
		t.Fatal("WriterSupportsStreaming() should not treat bytes.Buffer as streaming")
	}
}

func TestLocateTestGenieCLIUsesManifestDrivenResolver(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	testkitgo.WriteRepoContract(t, root, "scenarios")
	writeShellScenarioCLI(t, root, "test-genie")

	path, err := LocateTestGenieCLI(func(string) (string, error) { return "", os.ErrNotExist }, root, home)
	if err != nil {
		t.Fatalf("LocateTestGenieCLI: %v", err)
	}
	if want := filepath.Join(home, ".vrooli", "bin", "test-genie"); path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("installed binary missing: %v", err)
	}
}

func TestLocateScenarioCompletenessCLIUsesManifestDrivenResolver(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	testkitgo.WriteRepoContract(t, root, "scenarios")
	writeShellScenarioCLI(t, root, "scenario-completeness-scoring")

	path, err := LocateScenarioCompletenessCLI(func(string) (string, error) { return "", os.ErrNotExist }, root)
	if err != nil {
		t.Fatalf("LocateScenarioCompletenessCLI: %v", err)
	}
	if want := filepath.Join(home, ".vrooli", "bin", "scenario-completeness-scoring"); path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("installed binary missing: %v", err)
	}
}

func writeShellScenarioCLI(t *testing.T, root, name string) {
	t.Helper()
	testscenario.WriteScenarioService(t, root, name, testscenario.ScenarioServiceManifest(
		name,
		testscenario.WithCLI(&scenario.CLIConfig{
			Enabled: true,
			Command: name,
			Adapter: scenario.CLIAdapterConfig{
				Kind:          "shell_script",
				ScriptPath:    filepath.ToSlash(filepath.Join("cli", name)),
				InstallScript: "cli/install.sh",
			},
			Install: []scenario.CLIInstallStep{{Kind: "command", Run: "bash ./cli/install.sh"}},
			Freshness: &scenario.CLIFreshnessCheck{
				Inputs: []string{filepath.ToSlash(filepath.Join("cli", name)), "cli/install.sh"},
			},
		}),
	))
	installScript := "#!/usr/bin/env bash\nset -e\nmkdir -p \"$HOME/.vrooli/bin\"\ncp \"$(dirname \"$0\")/" + name + "\" \"$HOME/.vrooli/bin/" + name + "\"\nchmod +x \"$HOME/.vrooli/bin/" + name + "\"\n"
	testkitgo.WriteRelativeExecutable(t, root, filepath.Join("scenarios", name, "cli", "install.sh"), installScript)
	testkitgo.WriteRelativeExecutable(t, root, filepath.Join("scenarios", name, "cli", name), "#!/usr/bin/env bash\necho ok\n")
}
