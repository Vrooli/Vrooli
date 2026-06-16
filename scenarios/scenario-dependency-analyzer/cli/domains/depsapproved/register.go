package depsapproved

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"scenario-dependency-analyzer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
	governanceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance/dependency_governance_v1connect"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "deps-approved",
		Description: "Inspect approved third-party dependency governance records",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Description: "List approved dependency records", Run: func(args []string) error { return runList(core, args) }},
			{Name: "search", Description: "Search approved dependency records", Run: func(args []string) error { return runSearch(core, args) }},
			{Name: "explain", Description: "Explain one approved dependency record", Run: func(args []string) error { return runExplain(core, args) }},
			{Name: "validate", Description: "Validate scenario dependencies against governance records", Run: func(args []string) error { return runValidate(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("deps approved list")
	var ecosystem, state, surface, useCase string
	var jsonOutput bool
	fs.StringVar(&ecosystem, "ecosystem", "", "Filter by ecosystem")
	fs.StringVar(&state, "state", "", "Filter by governance state")
	fs.StringVar(&surface, "surface", "", "Filter by allowed surface")
	fs.StringVar(&useCase, "use-case", "", "Filter by use case")
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("usage: %s deps approved list [--ecosystem npm|go] [--state approved] [--json]", support.AppName)
	}
	resp, err := governanceClient(core).ListApprovedDependencies(context.Background(), connect.NewRequest(&governancev1.ListApprovedDependenciesRequest{
		Ecosystem: ecosystem,
		State:     state,
		Surface:   surface,
		UseCase:   useCase,
	}))
	if err != nil {
		return cliapp.WrapAPIError("list approved dependencies", err, nil)
	}
	if jsonOutput {
		return printProto(resp.Msg)
	}
	return printRecords("Approved Dependencies", resp.Msg.GetRecords(), resp.Msg.GetGuidance())
}

func runSearch(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("deps approved search")
	var ecosystem string
	var limit int
	var jsonOutput bool
	fs.StringVar(&ecosystem, "ecosystem", "", "Filter by ecosystem")
	fs.IntVar(&limit, "limit", 20, "Maximum records to return")
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	query := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if query == "" {
		return fmt.Errorf("usage: %s deps approved search <query> [--ecosystem npm|go] [--json]", support.AppName)
	}
	resp, err := governanceClient(core).SearchApprovedDependencies(context.Background(), connect.NewRequest(&governancev1.SearchApprovedDependenciesRequest{
		Query:     query,
		Ecosystem: ecosystem,
		Limit:     int32(limit),
	}))
	if err != nil {
		return cliapp.WrapAPIError("search approved dependencies", err, nil)
	}
	if jsonOutput {
		return printProto(resp.Msg)
	}
	return printRecords("Approved Dependency Search", resp.Msg.GetRecords(), resp.Msg.GetGuidance())
}

func runExplain(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("deps approved explain")
	var jsonOutput bool
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	positionals := fs.Args()
	if len(positionals) != 1 {
		return fmt.Errorf("usage: %s deps approved explain <ecosystem>/<package> [--json]", support.AppName)
	}
	ecosystem, packageName, ok := strings.Cut(positionals[0], "/")
	if !ok || strings.TrimSpace(ecosystem) == "" || strings.TrimSpace(packageName) == "" {
		return fmt.Errorf("approved dependency must be formatted as <ecosystem>/<package>")
	}
	resp, err := governanceClient(core).ExplainApprovedDependency(context.Background(), connect.NewRequest(&governancev1.ExplainApprovedDependencyRequest{
		Ecosystem:   ecosystem,
		PackageName: packageName,
	}))
	if err != nil {
		return cliapp.WrapAPIError("explain approved dependency", err, nil)
	}
	if jsonOutput {
		return printProto(resp.Msg)
	}
	if !resp.Msg.GetFound() {
		report := cliapp.ListReport{
			Summary: []string{fmt.Sprintf("Dependency: %s/%s", ecosystem, packageName), "Found: false", resp.Msg.GetGuidance()},
		}
		return support.PrintList(false, report, nil)
	}
	return printRecords("Approved Dependency", []*governancev1.ApprovedDependencyRecord{resp.Msg.GetRecord()}, resp.Msg.GetGuidance())
}

func runValidate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("deps approved validate")
	var jsonOutput bool
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	positionals := fs.Args()
	if len(positionals) != 1 {
		return fmt.Errorf("usage: %s deps approved validate <scenario> [--json]", support.AppName)
	}
	resp, err := governanceClient(core).ValidateApprovedDependencies(context.Background(), connect.NewRequest(&governancev1.ValidateApprovedDependenciesRequest{
		Scenario: positionals[0],
	}))
	if err != nil {
		return cliapp.WrapAPIError("validate approved dependencies", err, nil)
	}
	if jsonOutput {
		return printProto(resp.Msg)
	}
	results := make([]string, 0, len(resp.Msg.GetFindings()))
	for _, finding := range resp.Msg.GetFindings() {
		results = append(results, fmt.Sprintf("%s: %s/%s - %s", finding.GetSeverity(), finding.GetEcosystem(), finding.GetPackageName(), finding.GetTitle()))
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Scenario: %s", resp.Msg.GetScenario()),
			fmt.Sprintf("Passed: %t", resp.Msg.GetPassed()),
			fmt.Sprintf("Observed dependencies: %d", resp.Msg.GetSummary().GetObserved()),
			resp.Msg.GetGuidance(),
		},
		ResultsHeading: "Governance Findings",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s deps approved validate %s --json", support.AppName, positionals[0]),
			fmt.Sprintf("%s deps approved list --json", support.AppName),
		},
	}
	return support.PrintList(false, report, nil)
}

func governanceClient(core *cliapp.ScenarioApp) governanceconnect.DependencyGovernanceServiceClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, 45*time.Second)
	return governanceconnect.NewDependencyGovernanceServiceClient(httpClient, baseURL)
}

func printProto(msg proto.Message) error {
	body, err := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, string(body))
	return err
}

func printRecords(title string, records []*governancev1.ApprovedDependencyRecord, guidance string) error {
	results := make([]string, 0, len(records))
	for _, record := range records {
		results = append(results, fmt.Sprintf("%s/%s %s [%s]", record.GetEcosystem(), record.GetPackageName(), record.GetVersionRange(), record.GetState()))
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Records: %d", len(records)),
			guidance,
		},
		ResultsHeading: title,
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s deps approved search \"React graph library\" --json", support.AppName),
			fmt.Sprintf("%s deps approved validate scenario-dependency-analyzer --json", support.AppName),
		},
	}
	return support.PrintList(false, report, nil)
}
