package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestElectronShellMatchesProcessedCanonicalTemplate exercises the same
// scenario-to-desktop generator used for BAS output, then compares every
// generated source file with the current shell. It intentionally keeps no
// shell fixture: the canonical template is the expected source of bytes.
func TestElectronShellMatchesProcessedCanonicalTemplate(t *testing.T) {
	apiDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Dir(apiDir)
	repoRoot := filepath.Dir(filepath.Dir(root))
	ownerBuildTools := filepath.Join(root, "..", "scenario-to-desktop", "templates", "build-tools")
	freshness := exec.Command("node", filepath.Join(ownerBuildTools, "write-build-stamp.js"), "--check")
	if outputBytes, err := freshness.CombinedOutput(); err != nil {
		t.Fatalf("scenario-to-desktop owner generator is not fresh: %v\n%s", err, outputBytes)
	}
	templateGenerator := filepath.Join(ownerBuildTools, "dist", "template-generator.js")

	output := filepath.Join(t.TempDir(), "electron")
	config := electronShellGenerationConfig(t, root, output)
	configBytes, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(t.TempDir(), "desktop-config.json")
	if err := os.WriteFile(configPath, configBytes, 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("node", templateGenerator, configPath)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "SKIP_DESKTOP_DEPENDENCY_INSTALL=1", "VROOLI_ROOT="+repoRoot)
	if outputBytes, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("scenario-to-desktop generator failed: %v\n%s", err, outputBytes)
	}

	generatedRoot := filepath.Join(output, "src")
	currentRoot := filepath.Join(root, "platforms", "electron", "src")
	generatedFiles := sourceFiles(t, generatedRoot)
	currentFiles := sourceFiles(t, currentRoot)
	if fmt.Sprint(generatedFiles) != fmt.Sprint(currentFiles) {
		t.Fatalf("generated shell file set differs from canonical output\ncurrent: %v\ngenerated: %v", currentFiles, generatedFiles)
	}
	for _, relative := range generatedFiles {
		want, err := os.ReadFile(filepath.Join(generatedRoot, relative))
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(currentRoot, relative))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			limit := len(got)
			if len(want) < limit {
				limit = len(want)
			}
			offset := 0
			for offset < limit && got[offset] == want[offset] {
				offset++
			}
			start := offset - 80
			if start < 0 {
				start = 0
			}
			endGot, endWant := offset+160, offset+160
			if endGot > len(got) {
				endGot = len(got)
			}
			if endWant > len(want) {
				endWant = len(want)
			}
			t.Errorf("generated shell differs from processed canonical source: %s at byte %d\ncurrent: %q\ngenerated: %q", relative, offset, got[start:endGot], want[start:endWant])
		}
	}
	for _, relative := range []string{
		".gitignore", "README.md", "package-lock.json", "package.json", "tsconfig.json", "scripts/notarize.js",
	} {
		want, err := os.ReadFile(filepath.Join(output, relative))
		if err != nil {
			t.Fatalf("read generated output %s: %v", relative, err)
		}
		got, err := os.ReadFile(filepath.Join(root, "platforms", "electron", relative))
		if err != nil {
			t.Fatalf("read current output %s: %v", relative, err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("generated Electron output differs from processed canonical source: %s", relative)
		}
	}

	mainProcess, err := os.ReadFile(filepath.Join(generatedRoot, "main.ts"))
	if err != nil {
		t.Fatal(err)
	}
	for _, invariant := range []string{
		"nodeIntegration: false, contextIsolation: true, sandbox: true",
		"setWindowOpenHandler(({ url })",
	} {
		if !strings.Contains(string(mainProcess), invariant) {
			t.Errorf("generated Electron main process lost security invariant %q", invariant)
		}
	}
	preload, err := os.ReadFile(filepath.Join(generatedRoot, "preload.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(preload), "contextBridge.exposeInMainWorld(") {
		t.Error("generated Electron preload no longer exposes its narrow context bridge")
	}
}

func sourceFiles(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	return files
}

// Generation and comparison consume the same reviewed source-proxy input. The
// saved bundled desktop-config.json is a distinct deployment profile.
func electronShellGenerationConfig(t *testing.T, root, output string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, ".vrooli", "electron-shell-config.json"))
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	config["output_path"] = output
	return config
}

func TestElectronShellGenerationInputKeepsPolicyInSharedConfig(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, ".vrooli", "electron-shell-config.json")
	// The only supported override is the temporary comparison destination;
	// policy changes in the shared input must flow through without defaults.
	before := []byte(`{"deployment_mode":"proxy","window":{"width":1437},"features":{"systemTray":true},"output_path":"platforms/electron"}`)
	if err := os.WriteFile(configPath, before, 0o644); err != nil {
		t.Fatal(err)
	}
	config := electronShellGenerationConfig(t, root, "temporary-output")
	if config["output_path"] != "temporary-output" {
		t.Fatal("comparison did not isolate output")
	}
	if config["deployment_mode"] != "proxy" || config["window"].(map[string]any)["width"] != float64(1437) || config["features"].(map[string]any)["systemTray"] != true {
		t.Fatal("comparison reconstructed policy instead of reading the shared input")
	}
	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("comparison mutated authoritative input")
	}
}
