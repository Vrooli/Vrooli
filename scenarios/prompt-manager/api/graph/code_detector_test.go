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
	content := "Run scripts/deploy.sh to deploy"
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
	if scriptRefs[0].Value != "scripts/deploy.sh" {
		t.Errorf("expected scripts/deploy.sh, got %s", scriptRefs[0].Value)
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
