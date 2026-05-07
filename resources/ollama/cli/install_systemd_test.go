package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInstallShellHasCgroupProtections is a regression guard for the host-stability
// fix on 2026-05-07. The active /etc/systemd/system/ollama.service on the dev
// workstation had no cgroup limits, so an embeddings burst from agents bypassing
// the resource-ollama wrapper could exhaust host memory and contributed to a
// kernel hard-hang investigation. Both code paths in lib/install.sh that render
// the unit (install_service for fresh installs, update_service_config for
// reconfigure) must include the same cgroup directives.
func TestInstallShellHasCgroupProtections(t *testing.T) {
	t.Parallel()

	// Locate lib/install.sh relative to this test (cli/ is a sibling of lib/).
	// Walk up from CWD until we find a resources/ollama/lib/install.sh.
	path := findInstallShell(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	content := string(data)

	// Required directives that must appear in the rendered unit. We assert on
	// the full reference (e.g. "MemoryMax=$OLLAMA_MEMORY_MAX") so a refactor
	// that drops a directive while keeping the variable name would still trip
	// the test.
	required := []string{
		"MemoryHigh=$OLLAMA_MEMORY_HIGH",
		"MemoryMax=$OLLAMA_MEMORY_MAX",
		"TasksMax=$OLLAMA_TASKS_MAX",
		"OOMScoreAdjust=$OLLAMA_OOM_SCORE_ADJUST",
	}
	for _, want := range required {
		if !strings.Contains(content, want) {
			t.Errorf("install.sh rendered unit missing %q", want)
		}
	}

	// The shared helper that renders the resource-control block must exist
	// and be referenced from both heredocs.
	if !strings.Contains(content, "ollama::_render_resource_controls()") {
		t.Error("install.sh missing ollama::_render_resource_controls helper")
	}
	usages := strings.Count(content, "$(ollama::_render_resource_controls)")
	if usages < 2 {
		t.Errorf("ollama::_render_resource_controls used %d times; want >=2 (install_service + update_service_config)", usages)
	}

	// needs_parallel_config_update must trigger when cgroup protections are
	// missing — otherwise existing stock-installer units never get re-rendered.
	if !strings.Contains(content, "MemoryMax|MemoryHigh|OOMScoreAdjust") {
		t.Error("needs_parallel_config_update does not detect missing cgroup protections")
	}
}

func findInstallShell(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for dir := cwd; dir != "/" && dir != "."; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, "lib", "install.sh")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		// When tests run from cli/, ../lib/install.sh is the answer.
		candidate = filepath.Join(dir, "..", "lib", "install.sh")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	t.Fatal("could not locate resources/ollama/lib/install.sh from cwd")
	return ""
}
