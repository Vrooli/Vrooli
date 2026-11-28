package runtime

import (
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/tools"
)

func TestNodeToolchainManifestOwnsPinnedReleaseAndCIUsesIt(t *testing.T) {
	manifest := readToolManifest(t, "node")
	if !regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(manifest.Version) {
		t.Fatalf("Node version %q must be a pinned x.y.z release", manifest.Version)
	}
	if manifest.SourceType() != "url" || manifest.Acquisition == nil {
		t.Fatalf("Node source = %#v, want verified URL acquisition", manifest.Acquisition)
	}
	for _, platform := range []string{"linux/amd64", "linux/arm64", "darwin/amd64", "darwin/arm64", "windows/amd64", "windows/arm64"} {
		parts := strings.SplitN(platform, "/", 2)
		target, ok := hostreqkit.TargetFor(manifest.Acquisition, parts[0], parts[1])
		if !ok || target.Layout != "dir" || target.Archive == "" || target.BinPath == "" || len(target.SHA256) != 64 {
			t.Errorf("Node target %s = %#v; want a checksum-verified directory release", platform, target)
		}
	}
	assertWorkflowReadsToolVersion(t, manifest.Name, []string{".github/workflows/test.yml", ".github/workflows/codeql.yml"})
}

func TestPnpmToolchainManifestMatchesRootAndProjectPins(t *testing.T) {
	manifest := readToolManifest(t, "pnpm")
	if !regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(manifest.Version) {
		t.Fatalf("pnpm version %q must be a pinned x.y.z release", manifest.Version)
	}
	if manifest.SourceType() != "url" || manifest.Acquisition == nil {
		t.Fatalf("pnpm source = %#v, want verified URL acquisition", manifest.Acquisition)
	}
	for _, platform := range []string{"linux/amd64", "linux/arm64", "darwin/amd64", "darwin/arm64", "windows/amd64", "windows/arm64"} {
		parts := strings.SplitN(platform, "/", 2)
		target, ok := hostreqkit.TargetFor(manifest.Acquisition, parts[0], parts[1])
		if !ok || target.Layout != "file" || len(target.SHA256) != 64 || !strings.Contains(target.URL, "v"+manifest.Version+"/") {
			t.Errorf("pnpm target %s = %#v; want checksum-verified pnpm %s binary", platform, target, manifest.Version)
		}
	}

	repoRoot := repositoryRoot(t)
	workspace, err := os.ReadFile(filepath.Join(repoRoot, "pnpm-workspace.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	workspacePin := regexp.MustCompile(`(?m)^packageManager:\s*pnpm@([^\s]+)$`).FindSubmatch(workspace)
	if len(workspacePin) != 2 || string(workspacePin[1]) != manifest.Version {
		t.Fatalf("root pnpm-workspace.yaml pin = %q, want %s", workspacePin, manifest.Version)
	}

	managerPins := 0
	gitFiles := exec.Command("git", "ls-files", "-z", "--", "package.json", "**/package.json")
	gitFiles.Dir = repoRoot
	tracked, err := gitFiles.Output()
	if err != nil {
		t.Fatalf("list tracked package.json files: %v", err)
	}
	for _, rawPath := range strings.Split(string(tracked), "\x00") {
		if rawPath == "" {
			continue
		}
		path := filepath.Join(repoRoot, filepath.FromSlash(rawPath))
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", rawPath, err)
		}
		var pkg struct {
			PackageManager string `json:"packageManager"`
		}
		if err := json.Unmarshal(data, &pkg); err != nil {
			continue // Some checked-in package.json fixtures are deliberately partial.
		}
		if !strings.HasPrefix(pkg.PackageManager, "pnpm@") {
			continue
		}
		managerPins++
		versionAndIntegrity := strings.SplitN(strings.TrimPrefix(pkg.PackageManager, "pnpm@"), "+", 2)
		version := versionAndIntegrity[0]
		if version != manifest.Version {
			t.Errorf("%s pins pnpm %s, want the managed release %s", rawPath, version, manifest.Version)
		}
		if len(versionAndIntegrity) != 2 || !regexp.MustCompile(`^sha512\.[a-f0-9]{128}$`).MatchString(versionAndIntegrity[1]) {
			t.Errorf("%s packageManager pin must include SHA-512 hex integrity for Corepack", rawPath)
		}
	}
	if managerPins == 0 {
		t.Fatal("expected project packageManager pins")
	}
	assertWorkflowReadsToolVersion(t, manifest.Name, []string{".github/workflows/test.yml", ".github/workflows/codeql.yml"})
}

func readToolManifest(t *testing.T, name string) hostreqkit.ToolManifest {
	t.Helper()
	data, err := fs.ReadFile(tools.Manifests, name+"/tool.json")
	if err != nil {
		t.Fatalf("read %s manifest: %v", name, err)
	}
	var manifest hostreqkit.ToolManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse %s manifest: %v", name, err)
	}
	return manifest
}

// assertWorkflowReadsToolVersion requires every workflow that installs a tool to
// derive its version from the tool manifest instead of pinning a second copy.
// A workflow that never sets the tool up has nothing to derive, so it is not a
// contract violation; at least one workflow must still exercise the pin.
func assertWorkflowReadsToolVersion(t *testing.T, tool string, workflows []string) {
	t.Helper()
	repoRoot := repositoryRoot(t)
	checked := 0
	for _, workflow := range workflows {
		contents, err := os.ReadFile(filepath.Join(repoRoot, workflow))
		if err != nil {
			t.Errorf("read %s: %v", workflow, err)
			continue
		}
		text := string(contents)
		if !strings.Contains(text, "tools/"+tool+"/tool.json") && !strings.Contains(text, "setup-"+tool) {
			continue
		}
		checked++
		if !strings.Contains(text, "jq -r '.version' internal/tools/"+tool+"/tool.json") ||
			!strings.Contains(text, "steps."+tool+"-version.outputs.value") {
			t.Errorf("%s must derive %s version from its tool manifest", workflow, tool)
		}
	}
	if checked == 0 {
		t.Errorf("no workflow installs %s; the pinned manifest is unused", tool)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate JavaScript toolchain contract test")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "../.."))
}
