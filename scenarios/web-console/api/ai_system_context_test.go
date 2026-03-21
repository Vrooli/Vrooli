package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeLookPath returns a LookPathFunc that "finds" only the tools in the found set.
func fakeLookPath(found map[string]bool) LookPathFunc {
	return func(file string) (string, error) {
		if found[file] {
			return "/usr/bin/" + file, nil
		}
		return "", fmt.Errorf("not found: %s", file)
	}
}

func TestDiscoverSystemContext_FindsTools(t *testing.T) {
	found := map[string]bool{"rg": true, "git": true, "docker": true, "jq": true}
	ctx := DiscoverSystemContext(fakeLookPath(found))

	if ctx.OS == "" {
		t.Error("OS should not be empty")
	}
	if ctx.Arch == "" {
		t.Error("Arch should not be empty")
	}

	// Verify discovered tools land in correct categories.
	assertToolInCategory(t, ctx, "Search", "rg")
	assertToolInCategory(t, ctx, "Version control", "git")
	assertToolInCategory(t, ctx, "Containers", "docker")
	assertToolInCategory(t, ctx, "Data processing", "jq")

	// Verify tools that weren't found are absent.
	if tools, ok := ctx.Tools["Editors"]; ok {
		t.Errorf("Editors category should be absent, got %v", tools)
	}
}

func TestDiscoverSystemContext_NoToolsFound(t *testing.T) {
	ctx := DiscoverSystemContext(fakeLookPath(nil))

	if ctx.OS == "" {
		t.Error("OS should always be populated")
	}
	if ctx.Arch == "" {
		t.Error("Arch should always be populated")
	}
	if len(ctx.Tools) != 0 {
		t.Errorf("expected no tools, got %d categories", len(ctx.Tools))
	}
}

func TestDiscoverSystemContext_ExtraToolsEnvVar(t *testing.T) {
	t.Setenv("WC_EXTRA_TOOLS", "mytool, othertool")

	found := map[string]bool{"mytool": true, "othertool": true}
	ctx := DiscoverSystemContext(fakeLookPath(found))

	custom, ok := ctx.Tools["Custom"]
	if !ok {
		t.Fatal("expected Custom category from WC_EXTRA_TOOLS")
	}
	if len(custom) != 2 {
		t.Errorf("expected 2 custom tools, got %d: %v", len(custom), custom)
	}
}

func TestDiscoverSystemContext_ExtraToolsEmpty(t *testing.T) {
	t.Setenv("WC_EXTRA_TOOLS", "")
	ctx := DiscoverSystemContext(fakeLookPath(nil))

	if _, ok := ctx.Tools["Custom"]; ok {
		t.Error("Custom category should not exist for empty WC_EXTRA_TOOLS")
	}
}

func TestBuildCommandSystemPrompt_IncludesContext(t *testing.T) {
	ctx := &SystemContext{
		OS:     "linux",
		Arch:   "amd64",
		Distro: "Ubuntu 24.04",
		Shell:  "bash",
		Tools: map[string][]string{
			"Search":          {"rg", "fd"},
			"Version control": {"git"},
		},
	}

	prompt := buildCommandSystemPrompt(ctx)

	for _, want := range []string{
		"linux/amd64",
		"Ubuntu 24.04",
		"bash",
		"rg, fd",
		"git",
		"Prefer modern alternatives",
		"output ONLY the shell command",
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q\n\nfull prompt:\n%s", want, prompt)
		}
	}
}

func TestBuildCommandSystemPrompt_NilFallback(t *testing.T) {
	prompt := buildCommandSystemPrompt(nil)
	if prompt != commandSystemPrompt {
		t.Errorf("nil context should return hardcoded prompt, got:\n%s", prompt)
	}
}

func TestBuildSuggestSystemPrompt_IncludesContext(t *testing.T) {
	ctx := &SystemContext{
		OS:    "linux",
		Arch:  "amd64",
		Shell: "zsh",
		Tools: map[string][]string{
			"Search": {"rg"},
		},
	}

	prompt := buildSuggestSystemPrompt(ctx)

	for _, want := range []string{
		"linux/amd64",
		"zsh",
		"rg",
		"1 to 3 shell commands",
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q\n\nfull prompt:\n%s", want, prompt)
		}
	}
}

func TestBuildSuggestSystemPrompt_NilFallback(t *testing.T) {
	prompt := buildSuggestSystemPrompt(nil)
	if prompt != suggestSystemPrompt {
		t.Errorf("nil context should return hardcoded prompt, got:\n%s", prompt)
	}
}

func TestBuildPrompt_NoTools(t *testing.T) {
	ctx := &SystemContext{
		OS:   "linux",
		Arch: "amd64",
	}

	prompt := buildCommandSystemPrompt(ctx)

	if strings.Contains(prompt, "Available tools") {
		t.Errorf("prompt should not mention tools when none found:\n%s", prompt)
	}
	if !strings.Contains(prompt, "linux/amd64") {
		t.Errorf("prompt should still have OS info:\n%s", prompt)
	}
}

func TestBuildPrompt_CategoryOrder(t *testing.T) {
	ctx := &SystemContext{
		OS:   "linux",
		Arch: "amd64",
		Tools: map[string][]string{
			"Version control": {"git"},
			"Search":          {"rg"},
			"Custom":          {"mytool"},
		},
	}

	prompt := buildCommandSystemPrompt(ctx)

	searchIdx := strings.Index(prompt, "Search:")
	vcsIdx := strings.Index(prompt, "Version control:")
	customIdx := strings.Index(prompt, "Custom:")

	if searchIdx < 0 || vcsIdx < 0 || customIdx < 0 {
		t.Fatalf("missing categories in prompt:\n%s", prompt)
	}
	if searchIdx > vcsIdx {
		t.Errorf("Search should appear before Version control (canonical order)")
	}
	if customIdx < vcsIdx {
		t.Errorf("Custom should appear after canonical categories")
	}
}

func TestEnrichedPromptReachesProvider_Generate(t *testing.T) {
	var capturedSystem string
	provider := &contextCapturingProvider{
		name:   "ollama",
		result: "rg debug --type md",
		capture: func(sp, _ string) {
			capturedSystem = sp
		},
	}
	srv := newTestServerWithAI(provider)
	srv.systemContext = &SystemContext{
		OS:     "linux",
		Arch:   "amd64",
		Distro: "Ubuntu 24.04",
		Shell:  "bash",
		Tools: map[string][]string{
			"Search": {"rg", "fd"},
		},
	}

	body := strings.NewReader(`{"prompt":"find debugging docs"}`)
	req := httptest.NewRequest("POST", "/api/v1/ai/generate", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleAIGenerate(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	for _, want := range []string{"linux/amd64", "rg, fd", "bash"} {
		if !strings.Contains(capturedSystem, want) {
			t.Errorf("system prompt missing %q\n\ngot:\n%s", want, capturedSystem)
		}
	}
}

func TestEnrichedPromptReachesProvider_Suggest(t *testing.T) {
	var capturedSystem string
	provider := &contextCapturingProvider{
		name:   "ollama",
		result: "rg debug --type md\nfind . -name '*debug*'",
		capture: func(sp, _ string) {
			capturedSystem = sp
		},
	}
	srv := newTestServerWithAI(provider)
	srv.systemContext = &SystemContext{
		OS:    "linux",
		Arch:  "amd64",
		Shell: "bash",
		Tools: map[string][]string{
			"Search": {"rg"},
		},
	}

	body := strings.NewReader(`{"prompt":"find debugging docs"}`)
	req := httptest.NewRequest("POST", "/api/v1/ai/suggest", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleAISuggest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp AISuggestResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Commands) == 0 {
		t.Fatal("expected at least one command")
	}
	if !strings.Contains(capturedSystem, "rg") {
		t.Errorf("system prompt missing tool info\n\ngot:\n%s", capturedSystem)
	}
}

func TestShellBasename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/bin/bash", "bash"},
		{"/usr/bin/zsh", "zsh"},
		{"bash", "bash"},
		{"", ""},
	}
	for _, tt := range tests {
		got := shellBasename(tt.input)
		if got != tt.want {
			t.Errorf("shellBasename(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCountFoundTools(t *testing.T) {
	tools := map[string][]string{
		"Search":          {"rg", "fd"},
		"Version control": {"git"},
	}
	if n := countFoundTools(tools); n != 3 {
		t.Errorf("countFoundTools = %d, want 3", n)
	}
	if n := countFoundTools(nil); n != 0 {
		t.Errorf("countFoundTools(nil) = %d, want 0", n)
	}
}

// assertToolInCategory verifies a tool appears in the given category.
func assertToolInCategory(t *testing.T, ctx *SystemContext, category, tool string) {
	t.Helper()
	tools, ok := ctx.Tools[category]
	if !ok {
		t.Errorf("category %q not found in discovered tools", category)
		return
	}
	for _, found := range tools {
		if found == tool {
			return
		}
	}
	t.Errorf("tool %q not found in category %q (got %v)", tool, category, tools)
}
