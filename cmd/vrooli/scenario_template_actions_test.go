package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/cliout"
)

func TestParseScenarioGenerateRequestRequiresTemplateName(t *testing.T) {
	if _, err := scenariocli.ParseGenerateRequest(nil, &bytes.Buffer{}, func(name string) (scenariocli.TemplateInfo, error) {
		return scenariocli.TemplateInfo{}, nil
	}, scenariocli.ParseGenerateArgs); err == nil {
		t.Fatal("expected missing template name error")
	}
}

func TestRenderScenarioGenerateResponseDryRun(t *testing.T) {
	var stdout bytes.Buffer
	err := scenariocli.RenderGenerateResponse(&stdout, cliout.FormatHuman, scenariocli.GenerateResult{
		TemplateName: "demo",
		Destination:  "/tmp/alpha",
		Values:       map[string]string{"SCENARIO_ID": "alpha"},
		DryRun:       true,
	})
	if err != nil {
		t.Fatalf("renderScenarioGenerateResponse: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "[DRY-RUN] Would generate template demo at /tmp/alpha") {
		t.Fatalf("stdout = %q", output)
	}
}
