package graph

import (
	"testing"
)

func TestCLIDetector_ScenarioCLI(t *testing.T) {
	d := NewCLIDetector([]string{"prompt-manager", "app-monitor"})
	content := "Run `vrooli scenario start foo` then `prompt-manager skill read bar`"
	refs := d.Detect(content)

	var cliRefs []CodeReference
	for _, r := range refs {
		if r.Category == CodeScenarioCLI {
			cliRefs = append(cliRefs, r)
		}
	}

	if len(cliRefs) != 2 {
		t.Fatalf("expected 2 scenario-cli refs, got %d", len(cliRefs))
	}
}

func TestCLIDetector_APICall(t *testing.T) {
	d := NewCLIDetector(nil)
	content := "Fetch data: GET https://api.example.com/data"
	refs := d.Detect(content)

	var apiRefs []CodeReference
	for _, r := range refs {
		if r.Category == CodeAPICall {
			apiRefs = append(apiRefs, r)
		}
	}

	if len(apiRefs) != 1 {
		t.Fatalf("expected 1 api-call ref, got %d", len(apiRefs))
	}
}

func TestCLIDetector_CurlCommand(t *testing.T) {
	d := NewCLIDetector(nil)
	content := "curl -X POST https://api.example.com/endpoint"
	refs := d.Detect(content)

	var apiRefs []CodeReference
	for _, r := range refs {
		if r.Category == CodeAPICall {
			apiRefs = append(apiRefs, r)
		}
	}

	if len(apiRefs) < 1 {
		t.Fatalf("expected at least 1 api-call ref for curl, got %d", len(apiRefs))
	}
}

func TestCLIDetector_ScriptRef(t *testing.T) {
	d := NewCLIDetector(nil)
	content := "Run `scripts/deploy.sh --env prod` to deploy"
	refs := d.Detect(content)

	var scriptRefs []CodeReference
	for _, r := range refs {
		if r.Category == CodeScript {
			scriptRefs = append(scriptRefs, r)
		}
	}

	if len(scriptRefs) != 1 {
		t.Fatalf("expected 1 script ref, got %d", len(scriptRefs))
	}
	if scriptRefs[0].Value != "scripts/deploy.sh --env prod" {
		t.Errorf("expected 'scripts/deploy.sh --env prod', got %s", scriptRefs[0].Value)
	}
}

func TestCLIDetector_PlainTextScriptIgnored(t *testing.T) {
	d := NewCLIDetector(nil)
	// Plain text file references (no backticks) should NOT produce CodeScript refs.
	content := "See routes.ts and selectors.ts and deploy.sh for details"
	refs := d.Detect(content)

	for _, r := range refs {
		if r.Category == CodeScript {
			t.Errorf("unexpected script ref for plain text: %q", r.Value)
		}
	}
}

func TestCLIDetector_ShellScriptInBackticks(t *testing.T) {
	d := NewCLIDetector(nil)
	content := "Run `scripts/deploy.sh` and `lib/setup.bash`"
	refs := d.Detect(content)

	var scriptRefs []CodeReference
	for _, r := range refs {
		if r.Category == CodeScript {
			scriptRefs = append(scriptRefs, r)
		}
	}

	if len(scriptRefs) != 2 {
		t.Fatalf("expected 2 script refs, got %d: %v", len(scriptRefs), scriptRefs)
	}
}

func TestCLIDetector_NonBashScriptIgnored(t *testing.T) {
	d := NewCLIDetector(nil)
	// .ts, .py, .js in backticks should NOT be detected as scripts
	content := "Use `routes.ts` and `main.py` and `index.js`"
	refs := d.Detect(content)

	for _, r := range refs {
		if r.Category == CodeScript {
			t.Errorf("unexpected script ref for non-bash extension: %q", r.Value)
		}
	}
}

func TestCLIDetector_BashAndShCommands(t *testing.T) {
	d := NewCLIDetector(nil)
	content := "Run `bash -c 'echo hello'` and `sh -x script.sh`"
	refs := d.Detect(content)

	var extRefs []CodeReference
	for _, r := range refs {
		if r.Category == CodeExternalTool {
			extRefs = append(extRefs, r)
		}
	}

	if len(extRefs) != 2 {
		t.Fatalf("expected 2 external-tool refs for bash/sh, got %d: %v", len(extRefs), extRefs)
	}
}

func TestCLIDetector_Empty(t *testing.T) {
	d := NewCLIDetector(nil)
	refs := d.Detect("")

	if len(refs) != 0 {
		t.Fatalf("expected 0 refs for empty content, got %d", len(refs))
	}
}

func TestCLIDetector_VrooliAlwaysIncluded(t *testing.T) {
	d := NewCLIDetector(nil) // No scenario names passed
	content := "Run `vrooli help` for info"
	refs := d.Detect(content)

	var cliRefs []CodeReference
	for _, r := range refs {
		if r.Category == CodeScenarioCLI {
			cliRefs = append(cliRefs, r)
		}
	}

	if len(cliRefs) != 1 {
		t.Fatalf("expected 1 scenario-cli ref for vrooli, got %d", len(cliRefs))
	}
}

// ---------------------------------------------------------------------------
// Pipe splitting and external tool detection
// ---------------------------------------------------------------------------

func TestCLIDetector_PipedCommand(t *testing.T) {
	d := NewCLIDetector([]string{"prompt-manager"})
	content := "Run `vrooli scenario start foo | grep error`"
	refs := d.Detect(content)

	var cli, ext []CodeReference
	for _, r := range refs {
		switch r.Category {
		case CodeScenarioCLI:
			cli = append(cli, r)
		case CodeExternalTool:
			ext = append(ext, r)
		}
	}
	if len(cli) != 1 {
		t.Fatalf("expected 1 scenario-cli ref, got %d", len(cli))
	}
	if len(ext) != 1 {
		t.Fatalf("expected 1 external-tool ref, got %d", len(ext))
	}
	if ext[0].Value != "grep error" {
		t.Errorf("expected 'grep error', got %q", ext[0].Value)
	}
}

func TestCLIDetector_ChainedCommand(t *testing.T) {
	d := NewCLIDetector(nil)
	content := "Run `vrooli help && jq '.version'`"
	refs := d.Detect(content)

	var cli, ext []CodeReference
	for _, r := range refs {
		switch r.Category {
		case CodeScenarioCLI:
			cli = append(cli, r)
		case CodeExternalTool:
			ext = append(ext, r)
		}
	}
	if len(cli) != 1 {
		t.Fatalf("expected 1 scenario-cli ref, got %d", len(cli))
	}
	if len(ext) != 1 {
		t.Fatalf("expected 1 external-tool ref, got %d", len(ext))
	}
}

func TestCLIDetector_SemicolonChain(t *testing.T) {
	d := NewCLIDetector(nil)
	content := "Run `vrooli start; echo done`"
	refs := d.Detect(content)

	var cli, ext []CodeReference
	for _, r := range refs {
		switch r.Category {
		case CodeScenarioCLI:
			cli = append(cli, r)
		case CodeExternalTool:
			ext = append(ext, r)
		}
	}
	if len(cli) != 1 {
		t.Fatalf("expected 1 scenario-cli ref, got %d", len(cli))
	}
	if len(ext) != 1 {
		t.Fatalf("expected 1 external-tool ref, got %d", len(ext))
	}
}

func TestCLIDetector_ExternalToolOnly(t *testing.T) {
	d := NewCLIDetector(nil)
	content := "Run `grep -r pattern .` to search"
	refs := d.Detect(content)

	var ext []CodeReference
	for _, r := range refs {
		if r.Category == CodeExternalTool {
			ext = append(ext, r)
		}
	}
	if len(ext) != 1 {
		t.Fatalf("expected 1 external-tool ref, got %d", len(ext))
	}
	if ext[0].Value != "grep -r pattern ." {
		t.Errorf("expected 'grep -r pattern .', got %q", ext[0].Value)
	}
}

func TestCLIDetector_SingleWordBacktick(t *testing.T) {
	d := NewCLIDetector(nil)
	content := "Use `grep` for searching"
	refs := d.Detect(content)

	var ext []CodeReference
	for _, r := range refs {
		if r.Category == CodeExternalTool {
			ext = append(ext, r)
		}
	}
	if len(ext) != 1 {
		t.Fatalf("expected 1 external-tool ref, got %d", len(ext))
	}
}

func TestCLIDetector_CurlInBackticks(t *testing.T) {
	d := NewCLIDetector(nil)
	content := "Fetch: `curl -X POST https://api.example.com/endpoint`"
	refs := d.Detect(content)

	// curl in backticks → CodeExternalTool, NOT CodeAPICall
	var ext []CodeReference
	for _, r := range refs {
		if r.Category == CodeExternalTool {
			ext = append(ext, r)
		}
	}
	if len(ext) != 1 {
		t.Fatalf("expected 1 external-tool ref for curl, got %d", len(ext))
	}
}

func TestCLIDetector_OrPipe(t *testing.T) {
	d := NewCLIDetector(nil)
	content := "Run `vrooli start || echo failed`"
	refs := d.Detect(content)

	var cli, ext []CodeReference
	for _, r := range refs {
		switch r.Category {
		case CodeScenarioCLI:
			cli = append(cli, r)
		case CodeExternalTool:
			ext = append(ext, r)
		}
	}
	if len(cli) != 1 {
		t.Fatalf("expected 1 scenario-cli ref, got %d", len(cli))
	}
	if len(ext) != 1 {
		t.Fatalf("expected 1 external-tool ref, got %d", len(ext))
	}
}

// ---------------------------------------------------------------------------
// Unknown backtick content is skipped (not classified as external tool)
// ---------------------------------------------------------------------------

func TestCLIDetector_UnknownBacktickSkipped(t *testing.T) {
	d := NewCLIDetector(nil)
	// Inline code references that are NOT CLI commands
	content := "Use `useState` for state and `README.md` for docs and `--verbose` flag"
	refs := d.Detect(content)

	for _, r := range refs {
		if r.Category == CodeExternalTool || r.Category == CodeScenarioCLI {
			t.Errorf("unexpected CLI ref for inline code: %q (category=%s)", r.Value, r.Category)
		}
	}
}

func TestCLIDetector_MixedKnownAndUnknown(t *testing.T) {
	d := NewCLIDetector(nil)
	content := "Run `grep -r foo` then check `undefined` and `curl https://example.com`"
	refs := d.Detect(content)

	var ext []CodeReference
	for _, r := range refs {
		if r.Category == CodeExternalTool {
			ext = append(ext, r)
		}
	}
	// grep and curl are known external tools; undefined is not
	if len(ext) != 2 {
		t.Fatalf("expected 2 external-tool refs, got %d: %v", len(ext), ext)
	}
}

// ---------------------------------------------------------------------------
// Multi-line backtick and code fence tests
// ---------------------------------------------------------------------------

func TestCLIDetector_MultiLineBacktick(t *testing.T) {
	d := NewCLIDetector([]string{"visited-tracker"})
	content := "Run this:\n`visited-tracker status \\\n  --verbose`\nDone."
	refs := d.Detect(content)

	var cli []CodeReference
	for _, r := range refs {
		if r.Category == CodeScenarioCLI {
			cli = append(cli, r)
		}
	}
	if len(cli) != 1 {
		t.Fatalf("expected 1 scenario-cli ref for multi-line backtick, got %d", len(cli))
	}
	if cli[0].Value != "visited-tracker status \\\n  --verbose" {
		t.Errorf("unexpected value: %q", cli[0].Value)
	}
}

func TestCLIDetector_MultiLinePreservesLineNumber(t *testing.T) {
	d := NewCLIDetector(nil)
	// Backtick starts on line 3
	content := "line 1\nline 2\n`curl -X POST \\\n  https://example.com`\nline 5"
	refs := d.Detect(content)

	var ext []CodeReference
	for _, r := range refs {
		if r.Category == CodeExternalTool {
			ext = append(ext, r)
		}
	}
	if len(ext) != 1 {
		t.Fatalf("expected 1 external-tool ref, got %d", len(ext))
	}
	if ext[0].Line != 3 {
		t.Errorf("expected line 3 (opening backtick), got %d", ext[0].Line)
	}
}

func TestCLIDetector_CodeFenceStripped(t *testing.T) {
	d := NewCLIDetector([]string{"app-monitor"})
	content := "Some text\n```bash\napp-monitor start\n```\nMore text"
	refs := d.Detect(content)

	// Content inside code fences should NOT be detected as backtick commands.
	// The triple-backtick block is stripped before backtick matching.
	for _, r := range refs {
		if r.Category == CodeScenarioCLI && r.Value == "app-monitor start" {
			t.Fatal("code fence content should not be detected as a backtick command")
		}
	}
}

func TestCLIDetector_CodeFencePreservesLineNumbers(t *testing.T) {
	d := NewCLIDetector(nil)
	// Code fence spans lines 2-4, backtick command on line 5
	content := "line 1\n```\nignored\n```\n`grep -r foo`"
	refs := d.Detect(content)

	var ext []CodeReference
	for _, r := range refs {
		if r.Category == CodeExternalTool {
			ext = append(ext, r)
		}
	}
	if len(ext) != 1 {
		t.Fatalf("expected 1 external-tool ref, got %d", len(ext))
	}
	if ext[0].Line != 5 {
		t.Errorf("expected line 5, got %d", ext[0].Line)
	}
}
