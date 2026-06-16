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
	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_health"
	healthconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_health/dependency_health_v1connect"
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
		},
	}
}

func run(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("health")
	var useCache bool
	var jsonOutput bool
	fs.BoolVar(&useCache, "use-cache", true, "Allow cached upstream facts")
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
	client := healthconnect.NewDependencyHealthServiceClient(httpClient, baseURL)
	resp, err := client.ValidateDependencyHealth(context.Background(), connect.NewRequest(&healthv1.ValidateDependencyHealthRequest{
		Scenario: scenario,
		UseCache: useCache,
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

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Scenario: %s", resp.Msg.GetScenario()),
			fmt.Sprintf("Passed: %t", resp.Msg.GetPassed()),
			fmt.Sprintf("Findings: %d", resp.Msg.GetSummary().GetFindings()),
			fmt.Sprintf("Degraded integrations: %d", resp.Msg.GetSummary().GetDegradedDependencies()),
		},
		ResultsHeading: "Dependency Health Sections",
		Results:        sectionLines(resp.Msg.GetSections()),
		RetrievalHints: []string{
			fmt.Sprintf("%s health %s --json", support.AppName, strings.TrimSpace(scenario)),
			fmt.Sprintf("%s drift %s --json", support.AppName, strings.TrimSpace(scenario)),
		},
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
