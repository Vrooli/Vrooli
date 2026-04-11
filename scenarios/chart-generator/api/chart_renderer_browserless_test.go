package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChartRendererCapturePNGWithBrowserlessUsesResourceCLI(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "browserless.log")
	binDir := filepath.Join(tempDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("mkdir bin dir: %v", err)
	}

	fakeResource := filepath.Join(binDir, "resource-browserless")
	script := `#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" > "$FAKE_BROWSERLESS_LOG"
output=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --output)
      output="$2"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done
printf 'png' > "$output"
`
	if err := os.WriteFile(fakeResource, []byte(script), 0755); err != nil {
		t.Fatalf("write fake resource-browserless: %v", err)
	}

	oldPath := os.Getenv("PATH")
	oldLog := os.Getenv("FAKE_BROWSERLESS_LOG")
	t.Cleanup(func() {
		_ = os.Setenv("PATH", oldPath)
		if oldLog == "" {
			_ = os.Unsetenv("FAKE_BROWSERLESS_LOG")
		} else {
			_ = os.Setenv("FAKE_BROWSERLESS_LOG", oldLog)
		}
	})
	if err := os.Setenv("PATH", binDir+string(os.PathListSeparator)+oldPath); err != nil {
		t.Fatalf("set PATH: %v", err)
	}
	if err := os.Setenv("FAKE_BROWSERLESS_LOG", logPath); err != nil {
		t.Fatalf("set FAKE_BROWSERLESS_LOG: %v", err)
	}

	htmlPath := filepath.Join(tempDir, "chart.html")
	if err := os.WriteFile(htmlPath, []byte("<html><body>chart</body></html>"), 0644); err != nil {
		t.Fatalf("write html: %v", err)
	}
	outputPath := filepath.Join(tempDir, "chart.png")

	renderer := NewChartRenderer(tempDir)
	req := ChartGenerationProcessorRequest{Width: 640, Height: 480}
	if err := renderer.capturePNGWithBrowserless(outputPath, htmlPath, req); err != nil {
		t.Fatalf("capturePNGWithBrowserless: %v", err)
	}

	got, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read command log: %v", err)
	}
	args := string(got)
	for _, want := range []string{
		"screenshot",
		"--html-file " + htmlPath,
		"--output " + outputPath,
		"--viewport 640x480",
		"--wait-ms 2000",
	} {
		if !strings.Contains(args, want) {
			t.Fatalf("command args %q missing %q", args, want)
		}
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if string(data) != "png" {
		t.Fatalf("unexpected output contents: %q", string(data))
	}
}
