package main

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cliout"
)

func TestParseScenarioStartRequestRequiresSingleNameWhenPathProvided(t *testing.T) {
	_, err := parseScenarioStartRequest(globalOptions{}, []string{"alpha", "beta", "--path", "/tmp/custom"})
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

	err := executeCommandAction(configuredApp(), ctx, nil, commandAction[struct{}, struct{}]{
		parse: func(ctx *commandContext, args []string) (struct{}, error) {
			return struct{}{}, commandHelpOnly("Usage: vrooli scenario fake")
		},
		run: func(app *App, ctx *commandContext, req struct{}) (cliout.Format, struct{}, error) {
			t.Fatal("run should not be called for help-only command")
			return "", struct{}{}, nil
		},
		render: func(w io.Writer, format cliout.Format, resp struct{}) error {
			t.Fatal("render should not be called for help-only command")
			return nil
		},
	})
	if err != nil {
		t.Fatalf("executeCommandAction: %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "Usage: vrooli scenario fake") {
		t.Fatalf("help output missing usage text: %q", got)
	}
}

func TestRenderScenarioListResponseHumanIncludesPortsWhenPresent(t *testing.T) {
	var stdout bytes.Buffer
	err := renderScenarioListResponse(&stdout, cliout.FormatHuman, scenarioListResponse{
		Items: []scenarioListItemOutput{{
			Name:        "alpha",
			Description: "demo",
			Ports: []scenarioListPortOutput{{
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
	req, err := parseScenarioOpenRequest(globalOptions{}, []string{"alpha", "--print-url", "--json"})
	if err != nil {
		t.Fatalf("parseScenarioOpenRequest: %v", err)
	}
	if req.ScenarioName != "alpha" || !req.PrintURL || !req.JSON {
		t.Fatalf("request = %+v", req)
	}
}

func TestRenderScenarioPortResponseHumanSingleMatchesLegacyContract(t *testing.T) {
	var stdout bytes.Buffer
	err := renderScenarioPortResponse(&stdout, cliout.FormatHuman, scenarioPortResponse{
		Single: &scenarioPortSingleOutput{
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
