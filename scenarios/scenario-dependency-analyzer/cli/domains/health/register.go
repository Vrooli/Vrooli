package health

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"

	"scenario-dependency-analyzer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	maturityreport "github.com/vrooli/maturity-go/report"
	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_health"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Dependency Health",
		Commands: []cliapp.Command{
			{
				Name:        "health",
				Description: "Validate dependency health for one scenario",
				NeedsAPI:    true,
				Run: func(args []string) error {
					return run(core, args)
				},
			},
			{
				Name:        "freshness",
				Description: "Report fleet/touched package freshness for Go scenario surfaces",
				Run: func(args []string) error {
					return runFreshness(args)
				},
			},
		},
	}
}

func run(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("health")
	var jsonOutput bool
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	positionals := fs.Args()
	if len(positionals) != 1 {
		return fmt.Errorf("usage: %s health <scenario> [--json]", support.AppName)
	}
	scenario := positionals[0]

	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, 90*time.Second)
	client := scenariovalidationconnect.NewScenarioValidationServiceClient(httpClient, baseURL)
	resp, err := client.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario: scenario,
	}))
	if err != nil {
		return cliapp.WrapAPIError("validate dependency health", err, nil)
	}
	if jsonOutput {
		body, err := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(resp.Msg)
		if err != nil {
			return fmt.Errorf("render dependency health JSON: %w", err)
		}
		fmt.Fprintln(os.Stdout, string(body))
		return nil
	}

	native := &healthv1.DependencyHealthResponse{}
	if detail := resp.Msg.GetNativeDetail(); detail != nil {
		if err := detail.UnmarshalTo(native); err != nil {
			return fmt.Errorf("unpack dependency health detail: %w", err)
		}
	}
	if native.GetScenario() == "" {
		native.Scenario = resp.Msg.GetScenario()
	}
	if native.GetAssessment() == nil {
		native.Assessment = resp.Msg.GetAssessment()
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Scenario: %s", native.GetScenario()),
			fmt.Sprintf("Status: %s", resp.Msg.GetStatus().String()),
			fmt.Sprintf("Passed: %t", native.GetPassed()),
			fmt.Sprintf("Findings: %d", native.GetSummary().GetFindings()),
			fmt.Sprintf("Degraded integrations: %d", native.GetSummary().GetDegradedDependencies()),
		},
		ResultsHeading: "Dependency Health Sections",
		Results:        sectionLines(native.GetSections()),
		RetrievalHints: []string{
			fmt.Sprintf("%s health %s --json", support.AppName, strings.TrimSpace(scenario)),
			fmt.Sprintf("%s drift %s --json", support.AppName, strings.TrimSpace(scenario)),
		},
	}
	if maturity := maturityreport.BuildMaturityListReport(native.GetAssessment()); maturity.Summary != nil {
		report.Summary = append(report.Summary, maturity.Summary...)
		report.RetrievalHints = append(report.RetrievalHints, maturity.RetrievalHints...)
	}
	return support.PrintList(false, report, nil)
}

func sectionLines(sections []*healthv1.DependencyHealthSection) []string {
	out := make([]string, 0, len(sections))
	for _, section := range sections {
		out = append(out, fmt.Sprintf("%s: %s - %s", section.GetStatus(), section.GetTitle(), section.GetSummary()))
	}
	return out
}
