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
			{Name: "upsert", Description: "Preview or apply one dependency governance record from JSON", Run: func(args []string) error { return runUpsert(core, args) }},
			{Name: "approve", Description: "Preview or apply an approved dependency governance decision", Run: func(args []string) error { return runApprove(core, args) }},
			{Name: "deny", Description: "Preview or apply a denied dependency governance decision", Run: func(args []string) error { return runDeny(core, args) }},
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
	var all bool
	var policyMode string
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	fs.BoolVar(&all, "all", false, "Validate every discovered scenario")
	fs.StringVar(&policyMode, "policy-mode", "", "Override registry policy mode: advisory, strict, or review_gate")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	positionals := fs.Args()
	if all {
		if len(positionals) != 0 {
			return fmt.Errorf("usage: %s deps approved validate --all [--policy-mode advisory|strict|review_gate] [--json]", support.AppName)
		}
		resp, err := governanceClient(core).ValidateFleetApprovedDependencies(context.Background(), connect.NewRequest(&governancev1.ValidateFleetApprovedDependenciesRequest{
			PolicyMode: policyMode,
		}))
		if err != nil {
			return cliapp.WrapAPIError("validate approved dependencies across fleet", err, nil)
		}
		if jsonOutput {
			return printProto(resp.Msg)
		}
		return printFleetValidation(resp.Msg)
	}
	if len(positionals) != 1 {
		return fmt.Errorf("usage: %s deps approved validate <scenario> [--policy-mode advisory|strict|review_gate] [--json]", support.AppName)
	}
	resp, err := governanceClient(core).ValidateApprovedDependencies(context.Background(), connect.NewRequest(&governancev1.ValidateApprovedDependenciesRequest{
		Scenario:   positionals[0],
		PolicyMode: policyMode,
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

func runUpsert(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("deps approved upsert")
	var filePath string
	var apply bool
	var jsonOutput bool
	fs.StringVar(&filePath, "file", "", "Path to an ApprovedDependencyRecord JSON file")
	fs.BoolVar(&apply, "apply", false, "Apply the change; default is dry-run preview")
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 || strings.TrimSpace(filePath) == "" {
		return fmt.Errorf("usage: %s deps approved upsert --file <record.json> [--apply] [--json]", support.AppName)
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read approved dependency record file: %w", err)
	}
	var record governancev1.ApprovedDependencyRecord
	if err := protojson.Unmarshal(data, &record); err != nil {
		return fmt.Errorf("parse approved dependency record JSON: %w", err)
	}
	return runMutation(core, &record, !apply, jsonOutput)
}

func runApprove(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("deps approved approve")
	var versionRange, rationale, approvedBy, reviewExpires string
	var surfaces, groups, scenarios, useCases string
	var apply bool
	var jsonOutput bool
	fs.StringVar(&versionRange, "range", "*", "Approved version or version range")
	fs.StringVar(&rationale, "rationale", "", "Approval rationale")
	fs.StringVar(&approvedBy, "approved-by", "", "Reviewer or approving group")
	fs.StringVar(&reviewExpires, "review-expires", "", "Review expiry date in YYYY-MM-DD format")
	fs.StringVar(&surfaces, "surfaces", "", "Comma-separated allowed surfaces")
	fs.StringVar(&groups, "groups", "", "Comma-separated allowed dependency groups")
	fs.StringVar(&scenarios, "scenarios", "", "Comma-separated allowed scenarios")
	fs.StringVar(&useCases, "use-cases", "", "Comma-separated use-case tags")
	fs.BoolVar(&apply, "apply", false, "Apply the change; default is dry-run preview")
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	positionals := fs.Args()
	if len(positionals) != 1 || strings.TrimSpace(rationale) == "" {
		return fmt.Errorf("usage: %s deps approved approve <ecosystem>/<package> --rationale <text> [--range <range>] [--apply] [--json]", support.AppName)
	}
	ecosystem, packageName, err := parseDependencyID(positionals[0])
	if err != nil {
		return err
	}
	record := &governancev1.ApprovedDependencyRecord{
		Ecosystem:               ecosystem,
		PackageName:             packageName,
		VersionRange:            versionRange,
		State:                   "approved",
		AllowedSurfaces:         splitCSV(surfaces),
		UseCases:                splitCSV(useCases),
		Rationale:               rationale,
		ApprovedBy:              approvedBy,
		ReviewExpires:           reviewExpires,
		AllowedScenarios:        splitCSV(scenarios),
		AllowedDependencyGroups: splitCSV(groups),
	}
	return runMutation(core, record, !apply, jsonOutput)
}

func runDeny(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("deps approved deny")
	var versionRange, rationale, replacement string
	var scenarios string
	var apply bool
	var jsonOutput bool
	fs.StringVar(&versionRange, "range", "*", "Denied version or version range")
	fs.StringVar(&rationale, "reason", "", "Denial rationale")
	fs.StringVar(&replacement, "replacement", "", "Replacement or remediation guidance")
	fs.StringVar(&scenarios, "scenarios", "", "Comma-separated scenarios denied by this decision")
	fs.BoolVar(&apply, "apply", false, "Apply the change; default is dry-run preview")
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	positionals := fs.Args()
	if len(positionals) != 1 || strings.TrimSpace(rationale) == "" {
		return fmt.Errorf("usage: %s deps approved deny <ecosystem>/<package> --reason <text> [--replacement <text>] [--apply] [--json]", support.AppName)
	}
	ecosystem, packageName, err := parseDependencyID(positionals[0])
	if err != nil {
		return err
	}
	record := &governancev1.ApprovedDependencyRecord{
		Ecosystem:        ecosystem,
		PackageName:      packageName,
		VersionRange:     versionRange,
		State:            "denied",
		Rationale:        rationale,
		Replacement:      replacement,
		DeniedScenarios:  splitCSV(scenarios),
		AllowedScenarios: nil,
	}
	return runMutation(core, record, !apply, jsonOutput)
}

func runMutation(core *cliapp.ScenarioApp, record *governancev1.ApprovedDependencyRecord, dryRun bool, jsonOutput bool) error {
	resp, err := governanceClient(core).UpsertApprovedDependency(context.Background(), connect.NewRequest(&governancev1.UpsertApprovedDependencyRequest{
		Record: record,
		DryRun: dryRun,
	}))
	if err != nil {
		return cliapp.WrapAPIError("upsert approved dependency", err, nil)
	}
	if jsonOutput {
		return printProto(resp.Msg)
	}
	report := cliapp.ListReport{
		Summary: []string{
			resp.Msg.GetMessage(),
			fmt.Sprintf("Dry run: %t", resp.Msg.GetDryRun()),
			fmt.Sprintf("Changed: %t", resp.Msg.GetChanged()),
			fmt.Sprintf("Records: %d", governanceRecordCount(resp.Msg.GetSummary())),
			resp.Msg.GetGuidance(),
		},
		RetrievalHints: []string{
			fmt.Sprintf("%s deps approved explain %s/%s --json", support.AppName, resp.Msg.GetRecord().GetEcosystem(), resp.Msg.GetRecord().GetPackageName()),
			fmt.Sprintf("%s deps approved validate --all --json", support.AppName),
		},
	}
	return support.PrintList(false, report, nil)
}

func printFleetValidation(resp *governancev1.FleetApprovedDependencyValidationResponse) error {
	results := make([]string, 0, len(resp.GetScenarios()))
	for _, scenario := range resp.GetScenarios() {
		results = append(results, fmt.Sprintf("%s: passed=%t observed=%d findings=%d", scenario.GetScenario(), scenario.GetPassed(), scenario.GetSummary().GetObserved(), len(scenario.GetFindings())))
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Passed: %t", resp.GetPassed()),
			fmt.Sprintf("Scenarios: %d", resp.GetSummary().GetScenarioCount()),
			fmt.Sprintf("Dependencies: %d", resp.GetSummary().GetDependencyCount()),
			fmt.Sprintf("Findings: %d", resp.GetSummary().GetFindingCount()),
			fmt.Sprintf("Policy mode: %s", resp.GetSummary().GetPolicyMode()),
			resp.GetGuidance(),
		},
		ResultsHeading: "Scenario Governance Results",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s deps approved validate --all --json", support.AppName),
			fmt.Sprintf("%s deps approved validate test-genie --json", support.AppName),
		},
	}
	return support.PrintList(false, report, nil)
}

func parseDependencyID(value string) (string, string, error) {
	ecosystem, packageName, ok := strings.Cut(strings.TrimSpace(value), "/")
	if !ok || strings.TrimSpace(ecosystem) == "" || strings.TrimSpace(packageName) == "" {
		return "", "", fmt.Errorf("approved dependency must be formatted as <ecosystem>/<package>")
	}
	return ecosystem, packageName, nil
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func governanceRecordCount(summary *governancev1.DependencyGovernanceSummary) int32 {
	if summary == nil {
		return 0
	}
	return summary.GetApproved() + summary.GetApprovedWithConstraints() + summary.GetNeedsReview() + summary.GetDenied() + summary.GetDeprecated()
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
