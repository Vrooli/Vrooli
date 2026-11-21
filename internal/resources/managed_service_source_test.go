package resources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/vrooli/internal/operatorstate"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

func TestManagedServiceSourceAdoptionIsExplicitAndPreservesHostTool(t *testing.T) {
	root := t.TempDir()
	manifest := managedServiceTestManifest(strings.Repeat("a", 64))
	manifest.ManagedService.Acquisition = &binaryfetch.Acquisition{Kind: "url", Targets: []binaryfetch.AcquisitionTarget{{URL: "https://example.invalid/openrgb", SHA256: strings.Repeat("a", 64), Layout: "file"}}}
	manifest.ManagedService.HostToolExecutable = "openrgb"
	manifest.ManagedService.HostToolVersion = "1.1"
	manifest.ManagedService.SourceSelectionRequired = true
	manifest.ManagedService.ProviderPolicy.AllowedModes = []resourcedeployment.ProviderMode{resourcedeployment.ProviderManagedPrivate}
	manifest.ManagedService.ProviderPolicy.ExternalAccessCapabilities = nil
	writeResourceManifest(t, root, manifest.Name, manifest)
	copyOperatorStateSchemaForTest(t, root)
	stateRoot := filepath.Join(root, ".vrooli", "state")
	c := NewController(root, filepath.Join(root, "home"))
	c.operatorState = operatorstate.New(operatorstate.Config{RepoRoot: root, StateRoot: stateRoot})

	bin := t.TempDir()
	executable := filepath.Join(bin, "openrgb")
	argvMarker := filepath.Join(root, "host-tool-argv")
	if err := os.WriteFile(executable, []byte("#!/bin/sh\nprintf '%s\\n' \"$*\" >> \""+argvMarker+"\"\nprintf 'OpenRGB 1.1\\n'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	choice, err := c.SetManagedServiceSource(context.Background(), manifest.Name, "host")
	if err != nil {
		t.Fatal(err)
	}
	if choice.Mode != "host" || choice.Version != "1.1" || choice.Executable != executable || len(choice.SHA256) != 64 {
		t.Fatalf("choice = %+v", choice)
	}

	selectedPath, err := managedServiceArtifactPath(c, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if selectedPath != executable {
		t.Fatalf("selected path = %q, want adopted executable %q", selectedPath, executable)
	}
	if err := ensureManagedServiceArtifact(context.Background(), c, manifest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(executable); err != nil {
		t.Fatalf("adopted host executable was changed or removed: %v", err)
	}
	argv, err := os.ReadFile(argvMarker)
	if err != nil || strings.TrimSpace(string(argv)) != "--version\n--version" {
		t.Fatalf("compatibility checks invoked host executable with %q err=%v", argv, err)
	}

	artifact, err := managedServiceArtifactForLaunch(context.Background(), c, manifest, executable)
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Verification != "host-tool" || artifact.SHA256 != choice.SHA256 {
		t.Fatalf("launch artifact = %+v", artifact)
	}
	if _, err := c.StageManagedServiceArtifact(context.Background(), manifest.Name, manifest.ManagedService.Artifact.Version); err == nil || !strings.Contains(err.Error(), "host package manager") {
		t.Fatalf("managed update staging for adopted tool = %v", err)
	}
}

func TestCompatibleHostToolVersionRequiresExactOpenRGBRelease(t *testing.T) {
	for _, test := range []struct {
		output string
		want   bool
	}{
		{"OpenRGB 1.1\n", true},
		{"OpenRGB 0.9+ (1.1)\n", false},
		{"Other 1.1\n", false},
		{"OpenRGB 1.10\n", false},
	} {
		if got := compatibleHostToolVersion(test.output, "1.1"); got != test.want {
			t.Errorf("compatibleHostToolVersion(%q)=%t want %t", test.output, got, test.want)
		}
	}
}

func TestSourceSelectionRequiredDoesNotSilentlySelectManaged(t *testing.T) {
	root := t.TempDir()
	manifest := managedServiceTestManifest(strings.Repeat("a", 64))
	requests := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { requests++; _, _ = w.Write([]byte("unused")) }))
	defer upstream.Close()
	manifest.ManagedService.Acquisition = &binaryfetch.Acquisition{Kind: "url", Targets: []binaryfetch.AcquisitionTarget{{URL: upstream.URL, SHA256: strings.Repeat("a", 64), Layout: "file"}}}
	manifest.ManagedService.HostToolExecutable = "openrgb"
	manifest.ManagedService.HostToolVersion = "1.1"
	manifest.ManagedService.SourceSelectionRequired = true
	writeResourceManifest(t, root, manifest.Name, manifest)
	copyOperatorStateSchemaForTest(t, root)
	c := NewController(root, filepath.Join(root, "home"))
	c.operatorState = operatorstate.New(operatorstate.Config{RepoRoot: root, StateRoot: filepath.Join(root, ".vrooli", "state")})
	marker := filepath.Join(root, "host-tool-invoked")
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "openrgb"), []byte("#!/bin/sh\nprintf invoked > \""+marker+"\"\nprintf 'OpenRGB 1.1\\n'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	if _, err := managedServiceArtifactPath(c, manifest); err == nil || !strings.Contains(err.Error(), "requires an explicit source choice") {
		t.Fatalf("implicit source selection error = %v", err)
	}
	for _, action := range []string{"install", "status"} {
		err := (managedServiceDriver{}).Run(context.Background(), c, Resource{Name: manifest.Name}, manifest, action, nil, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "requires an explicit source choice") {
			t.Fatalf("default %s error = %v", action, err)
		}
	}
	if requests != 0 {
		t.Fatalf("default install made %d acquisition requests", requests)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("default setup invoked host OpenRGB, marker stat err=%v", err)
	}
	if _, running, err := func() (ManagedServiceState, bool, error) {
		supervisor, _, err := managedServiceSupervisorFor(manifest.Name)
		if err != nil {
			return ManagedServiceState{}, false, err
		}
		state, running, err := supervisor.Status()
		return state, running, err
	}(); err != nil || running {
		t.Fatalf("default setup process state: running=%t err=%v", running, err)
	}
}

func TestManagedServiceSourceRejectsIncompatibleHostBeforePersisting(t *testing.T) {
	root := t.TempDir()
	manifest := managedServiceTestManifest(strings.Repeat("a", 64))
	manifest.ManagedService.Acquisition = &binaryfetch.Acquisition{Kind: "url", Targets: []binaryfetch.AcquisitionTarget{{URL: "https://example.invalid/openrgb", SHA256: strings.Repeat("a", 64), Layout: "file"}}}
	manifest.ManagedService.HostToolExecutable = "openrgb"
	manifest.ManagedService.HostToolVersion = "1.1"
	manifest.ManagedService.SourceSelectionRequired = true
	manifest.ManagedService.ProviderPolicy.AllowedModes = []resourcedeployment.ProviderMode{resourcedeployment.ProviderManagedPrivate}
	manifest.ManagedService.ProviderPolicy.ExternalAccessCapabilities = nil
	writeResourceManifest(t, root, manifest.Name, manifest)
	copyOperatorStateSchemaForTest(t, root)
	c := NewController(root, filepath.Join(root, "home"))
	c.operatorState = operatorstate.New(operatorstate.Config{RepoRoot: root, StateRoot: filepath.Join(root, ".vrooli", "state")})
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "openrgb"), []byte("#!/bin/sh\nprintf 'OpenRGB 0.9\\n'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	if _, err := c.SetManagedServiceSource(context.Background(), manifest.Name, "host"); err == nil || !strings.Contains(err.Error(), "incompatible") {
		t.Fatalf("incompatible selection error = %v", err)
	}
	if _, selected, err := c.ManagedServiceSource(context.Background(), manifest.Name); err != nil || selected {
		t.Fatalf("incompatible host persisted: selected=%t err=%v", selected, err)
	}
}

func copyOperatorStateSchemaForTest(t *testing.T, root string) {
	t.Helper()
	for _, name := range []string{operatorstate.SchemaPath, ".vrooli/schemas/common.schema.json"} {
		source, err := os.ReadFile(filepath.Join(projectRoot(t), name))
		if err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, source, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
