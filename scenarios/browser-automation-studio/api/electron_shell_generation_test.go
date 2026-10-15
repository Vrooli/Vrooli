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
	templateGenerator := filepath.Join(root, "..", "scenario-to-desktop", "templates", "build-tools", "dist", "template-generator.js")
	if _, err := os.Stat(templateGenerator); err != nil {
		t.Fatalf("scenario-to-desktop owner generator is unavailable: %v", err)
	}

	output := filepath.Join(t.TempDir(), "electron")
	config := map[string]any{
		"app_name":           "browser-automation-studio",
		"app_display_name":   "Vrooli Ascension",
		"app_description":    "Visual browser automation studio with AI-powered workflow builder and real-time debugging",
		"version":            "1.0.0",
		"author":             "Vrooli Team",
		"author_email":       "noreply@vrooli.com",
		"homepage":           "https://vrooli.com",
		"icon":               filepath.Join(root, "platforms", "electron", "assets", "icon.png"),
		"license":            "MIT",
		"app_id":             "com.vrooli.browser-automation-studio",
		"app_url":            "",
		"server_type":        "external",
		"server_port":        20000,
		"server_path":        "http://localhost:20000",
		"api_endpoint":       "http://localhost:15000",
		"scenario_dist_path": filepath.Join(root, "ui", "dist"),
		"deployment_mode":    "proxy",
		"scenario_name":      "browser-automation-studio",
		"auto_manage_vrooli": false,
		"vrooli_binary_path": "vrooli",
		"framework":          "electron",
		"strategy_id":        "electron-sidecar",
		"template_type":      "basic",
		"features": map[string]any{
			"splash": true, "systemTray": false, "autoUpdater": true,
			"singleInstance": true, "devTools": true, "menu": true,
		},
		"window": map[string]any{
			"width": 1200, "height": 800, "background": "#f5f5f5", "title": "Vrooli Ascension",
		},
		"platforms": []string{"linux"},
		"ports": map[string]any{
			"api":                       map[string]any{"env_var": "API_PORT", "port": 15000},
			"ui":                        map[string]any{"env_var": "UI_PORT", "port": 20000},
			"playwright_driver":         map[string]any{"env_var": "PLAYWRIGHT_DRIVER_PORT", "port": 24400},
			"playwright_driver_frames":  map[string]any{"env_var": "PLAYWRIGHT_DRIVER_FRAMES_PORT", "port": 24400},
			"playwright_driver_metrics": map[string]any{"env_var": "PLAYWRIGHT_DRIVER_METRICS_PORT", "port": 24400},
		},
		"output_path": output,
	}
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
