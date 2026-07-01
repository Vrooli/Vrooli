package repocontractcheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeRepoFile(t *testing.T, root, rel, body string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func TestOpenRouterPolicyFacts_FlagsViolations(t *testing.T) {
	cases := map[string]string{
		"scenarios/demo/api/provider.go":    "package ai\nvar m = \"anthropic/claude-3.5-sonnet\"\n",
		"scenarios/demo/api/img.go":         "package ai\nconst d = \"google/gemini-2.5-flash-image-preview\"\n",
		"scenarios/demo/api/env.go":         "package ai\nfunc f() string { return getenv(\"OPENROUTER_IMAGE_MODEL\") }\n",
		"scenarios/demo/api/prefixed.go":    "package ai\nvar e = getenv(\"WC_OPENROUTER_MODEL\")\n",
		"scenarios/demo/cli/run.sh":         "#!/usr/bin/env bash\nresource-openrouter generate --model \"openai/gpt-4o\" --prompt hi\n",
		"resources/opencode/config/seed.sh": "#!/usr/bin/env bash\nDEFAULT=\"deepseek/deepseek-v4-flash\"\n",
	}
	root := t.TempDir()
	for rel, body := range cases {
		writeRepoFile(t, root, rel, body)
	}
	err := checkOpenRouterPolicyFacts(nil, root, "")
	if err == nil {
		t.Fatal("expected violations, got nil")
	}
	msg := err.Error()
	for _, want := range []string{
		"scenarios/demo/api/provider.go",
		"scenarios/demo/api/img.go",
		"scenarios/demo/api/env.go",
		"scenarios/demo/api/prefixed.go",
		"scenarios/demo/cli/run.sh",
		"resources/opencode/config/seed.sh",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("missing expected violation for %s in: %s", want, msg)
		}
	}
}

func TestOpenRouterPolicyFacts_AllowsLegitimateCases(t *testing.T) {
	root := t.TempDir()
	// None of these may produce a violation.
	writeRepoFile(t, root, "scenarios/demo/api/imports.go", "package x\nimport (\n\t\"google/go-cmp\"\n)\n")
	writeRepoFile(t, root, "scenarios/demo/api/ocr.go", "package x\nvar m = \"mistral/ocr\"\n") // provider 'mistral' not in OpenRouter set
	writeRepoFile(t, root, "scenarios/demo/api/role.go", "package x\nvar r = getenv(\"WC_OPENROUTER_ROLE\")\nvar p = getenv(\"OPENROUTER_MODEL_POLICY_PATH\")\n")
	writeRepoFile(t, root, "scenarios/demo/api/resource.go", "package x\nvar res = \"resources/claude-code\"\n") // resource path, no provider prefix
	writeRepoFile(t, root, "scenarios/demo/api/provider_test.go", "package x\nvar m = \"anthropic/claude-3.5-sonnet\" // test fixture\n")
	writeRepoFile(t, root, "scenarios/demo/docs/example.md", "Example: anthropic/claude-3.5-sonnet\n") // docs not scanned
	writeRepoFile(t, root, "scenarios/agent-manager/api/internal/pricing/aliases.go", "package pricing\nvar a = \"openai/gpt-5.5\"\n")
	writeRepoFile(t, root, "scenarios/agent-inbox/api/integrations/openrouter_types.go", "package i\nvar c = \"anthropic/claude-3.5-sonnet\"\n")
	writeRepoFile(t, root, "scenarios/landing-page-business-suite/api/ai_gateway_service.go", "package g\nvar allowed = []string{\"openai/gpt-4o\", \"anthropic/claude-3-haiku\"}\n")
	// resources/openrouter is the authority — its own slugs are allowed.
	writeRepoFile(t, root, "resources/openrouter/model-policy.json", "{\"model\":\"bytedance-seed/seedream-4.5\"}\n")
	writeRepoFile(t, root, "resources/openrouter/cli/internal/policy/policy.go", "package policy\nvar x = \"openai/gpt-4o\"\n")

	if err := checkOpenRouterPolicyFacts(nil, root, ""); err != nil {
		t.Fatalf("expected no violations, got: %v", err)
	}
}

func TestOpenRouterPolicyFacts_OnlyScansRuntimeSurfaces(t *testing.T) {
	root := t.TempDir()
	// A concrete slug in a non-runtime scenario surface (requirements) is ignored.
	writeRepoFile(t, root, "scenarios/demo/requirements/req.go", "package r\nvar m = \"openai/gpt-4o\"\n")
	if err := checkOpenRouterPolicyFacts(nil, root, ""); err != nil {
		t.Fatalf("requirements surface should not be scanned, got: %v", err)
	}
}
