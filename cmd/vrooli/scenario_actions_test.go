package main

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/cli/scenariohandlers"
	"github.com/vrooli/vrooli/internal/cliout"
)

func TestParseScenarioStartRequestRequiresSingleNameWhenPathProvided(t *testing.T) {
	_, err := scenariocli.ParseStartRequest(false, []string{"alpha", "beta", "--path", "/tmp/custom"})
	if err == nil {
		t.Fatal("expected path + multiple names to fail")
	}
	if !strings.Contains(err.Error(), "accepts exactly one scenario name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecuteScenarioCommandRendersHelpOnlyErrors(t *testing.T) {
	var stdout bytes.Buffer
	ctx := &commandContext{Stdout: &stdout}

	err := rootcli.BindGlobalCommand(commandStdout,
		func(ctx *commandContext, args []string) (struct{}, error) {
			return struct{}{}, rootcli.CommandHelpOnly(scenariocli.RequirementsHelpText())
		},
		func(ctx *commandContext, req struct{}) (cliout.Format, struct{}, error) {
			t.Fatal("run should not be called for help-only command")
			return "", struct{}{}, nil
		},
		func(w io.Writer, format cliout.Format, resp struct{}) error {
			t.Fatal("render should not be called for help-only command")
			return nil
		},
	)(ctx, nil)
	if err != nil {
		t.Fatalf("executeCommandAction: %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, scenariocli.RequirementsHelpText()) {
		t.Fatalf("help output missing usage text: %q", got)
	}
}

func TestRenderScenarioListResponseHumanIncludesPortsWhenPresent(t *testing.T) {
	var stdout bytes.Buffer
	err := scenariocli.RenderListResponse(&stdout, cliout.FormatHuman, scenariocli.ListResponse{
		Items: []scenariocli.ListItemOutput{{
			Name:        "alpha",
			Description: "demo",
			Ports: []scenariocli.ListPortOutput{{
				Key:  "API_PORT",
				Port: 18080,
			}},
		}},
		RunningCount: 1,
	})
	if err != nil {
		t.Fatalf("renderScenarioListResponse: %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "API_PORT=18080") {
		t.Fatalf("ports missing from human output: %q", got)
	}
}

func TestParseScenarioOpenRequestAcceptsJSONAndPrintURL(t *testing.T) {
	req, err := scenariocli.ParseOpenRequest(false, []string{"alpha", "--print-url", "--json"})
	if err != nil {
		t.Fatalf("parseScenarioOpenRequest: %v", err)
	}
	if req.ScenarioName != "alpha" || !req.PrintURL || !req.JSON {
		t.Fatalf("request = %+v", req)
	}
}

func TestRenderScenarioPortResponseHumanSingleMatchesLegacyContract(t *testing.T) {
	var stdout bytes.Buffer
	err := scenariocli.RenderPortResponse(&stdout, cliout.FormatHuman, scenariocli.PortResponse{
		Single: &scenariocli.PortSingleOutput{
			Success:  true,
			Scenario: "alpha",
			PortName: "UI_PORT",
			Port:     38080,
		},
	})
	if err != nil {
		t.Fatalf("renderScenarioPortResponse: %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != "38080" {
		t.Fatalf("stdout = %q", got)
	}
}

func TestParseScenarioValidateEnvRequestAcceptsJSON(t *testing.T) {
	req, err := scenariocli.ParseValidateEnvRequest(false, []string{"alpha", "--json"})
	if err != nil {
		t.Fatalf("parseScenarioValidateEnvRequest: %v", err)
	}
	if req.Name != "alpha" || !req.JSON {
		t.Fatalf("request = %+v", req)
	}
}

func TestScenarioHandlerHelpersCompile(t *testing.T) {
	if scenariohandlers.FormatTemplateRequiredFlags(scenariocli.TemplateManifest{}) == "" {
		t.Fatal("expected template flags helper to return a non-empty string")
	}
}
