package cliutil

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

func TestEmbeddedFreshnessInputsMatchGoListDirectoryRules(t *testing.T) {
	for _, pattern := range []string{"assets", "all:assets", "assets/.*"} {
		t.Run(pattern, func(t *testing.T) {
			root := t.TempDir()
			mustWriteFile(t, filepath.Join(root, "go.mod"), "module example\n\ngo 1.25\n")
			mustWriteFile(t, filepath.Join(root, "main.go"), "package main\nimport \"embed\"\n//go:embed "+pattern+"\nvar assets embed.FS\n")
			for name, body := range map[string]string{"assets/visible.txt": "visible", "assets/.hidden": "hidden", "assets/_private": "private", "assets/.dir/item": "deep", "assets/nested/go.mod": "module nested\n", "assets/nested/file.txt": "nested"} {
				mustWriteFile(t, filepath.Join(root, name), body)
			}
			cmd := exec.Command("go", "list", "-json", ".")
			cmd.Dir = root
			cmd.Env = append(os.Environ(), "GOWORK=off")
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("go list: %v: %s", err, output)
			}
			var pkg struct{ EmbedFiles []string }
			if err := json.Unmarshal(output, &pkg); err != nil {
				t.Fatal(err)
			}
			spec, err := withEmbeddedFreshnessInputs(FreshnessSpec{SourceRoot: root, ContextRoot: root, Inputs: []string{"."}})
			if err != nil {
				t.Fatal(err)
			}
			slices.Sort(spec.literalInputs)
			slices.Sort(pkg.EmbedFiles)
			if !slices.Equal(spec.literalInputs, pkg.EmbedFiles) {
				t.Fatalf("resolved=%v; go list=%v", spec.literalInputs, pkg.EmbedFiles)
			}
		})
	}
}

func TestEmbedExclusionUsesGoBuildTarget(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "go.mod"), "module example\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(root, "main.go"), "package main\nfunc main() {}\n")
	mustWriteFile(t, filepath.Join(root, "asset_windows.go"), "package main\nimport _ \"embed\"\n//go:embed assets/tool\nvar data string\n")
	mustWriteFile(t, filepath.Join(root, "assets", "tool"), "asset")
	spec := FreshnessSpec{SourceRoot: root, ContextRoot: root, Inputs: []string{"."}, SkipFiles: []string{"tool"}}
	t.Setenv("GOOS", "linux")
	if _, err := ComputeFreshnessManifest(spec, "go", nil, 1); err != nil {
		t.Fatalf("inactive Windows embed: %v", err)
	}
	explicit := spec
	explicit.Inputs = []string{"main.go", "asset_windows.go"}
	if _, err := ComputeFreshnessManifest(explicit, "go", nil, 1); err == nil {
		t.Fatal("explicit compiler source must remain authoritative over host GOOS")
	}
	t.Setenv("GOOS", "windows")
	if _, err := ComputeFreshnessManifest(spec, "go", nil, 1); err == nil {
		t.Fatal("active Windows embed must fail on conflicting exclusion")
	}
}

func TestEmbedExclusionUsesGoBuildTags(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "go.mod"), "module example\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(root, "main.go"), "package main\nfunc main() {}\n")
	mustWriteFile(t, filepath.Join(root, "tagged.go"), "//go:build customtag\n\npackage main\nimport _ \"embed\"\n//go:embed assets/tool\nvar data string\n")
	mustWriteFile(t, filepath.Join(root, "assets", "tool"), "asset")
	spec := FreshnessSpec{SourceRoot: root, ContextRoot: root, Inputs: []string{"."}, SkipFiles: []string{"tool"}}
	t.Setenv("GOFLAGS", "")
	if _, err := ComputeFreshnessManifest(spec, "go", nil, 1); err != nil {
		t.Fatalf("inactive tagged embed: %v", err)
	}
	t.Setenv("GOFLAGS", "-tags=customtag")
	if _, err := ComputeFreshnessManifest(spec, "go", nil, 1); err == nil {
		t.Fatal("active tagged embed must fail on conflicting exclusion")
	}
}
