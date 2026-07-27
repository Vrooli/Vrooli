package depsapproved

import (
	"context"
	"fmt"
	"os"
	"sort"
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
			{Name: "triage", Description: "Show grouped approved-dependency governance decisions", Run: func(args []string) error { return runTriage(core, args) }},
			{Name: "findings", Description: "List fleet approved-dependency governance findings", Run: func(args []string) error { return runFindings(core, args) }},
			{Name: "usage", Description: "Show fleet usage for one dependency", Run: func(args []string) error { return runUsage(core, args) }},
			{Name: "upsert", Description: "Preview or apply one dependency governance record from JSON", Run: func(args []string) error { return runUpsert(core, args) }},
			{Name: "propose-records", Description: "Propose draft governance records from fleet findings", Run: func(args []string) error { return runProposeRecords(core, args) }},
			{Name: "upsert-batch", Description: "Preview or apply a batch of dependency governance records", Run: func(args []string) error { return runUpsertBatch(core, args) }},
			{Name: "security-gaps", Description: "List vulnerable dependency exposures not yet represented in governance", Run: func(args []string) error { return runSecurityGaps(core, args) }},
			{Name: "approve-observed", Description: "Approve a dependency from observed fleet usage", Run: func(args []string) error { return runApproveObserved(core, args) }},
			{Name: "widen-range", Description: "Widen an existing approval to the observed major line", Run: func(args []string) error { return runWidenRange(core, args) }},
			{Name: "approve", Description: "Preview or apply an approved dependency governance decision", Run: func(args []string) error { return runApprove(core, args) }},
			{Name: "deny", Description: "Preview or apply a denied dependency governance decision", Run: func(args []string) error { return runDeny(core, args) }},
			{Name: "deny-vulnerable", Description: "Preview or apply a security-derived denied dependency decision", Run: func(args []string) error { return runDenyVulnerable(core, args) }},
			{Name: "remediate", Description: "Show remediation guidance for a vulnerable dependency", Run: func(args []string) error { return runRemediate(core, args) }},
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
	var ecosystem, framework, surface, state string
	var limit int
	var jsonOutput bool
	fs.StringVar(&ecosystem, "ecosystem", "", "Filter by ecosystem (npm|go|pip)")
	fs.StringVar(&framework, "framework", "", "Filter by framework/keyword (e.g. react)")
	fs.StringVar(&surface, "surface", "", "Filter by allowed surface (ui|api|cli|playwright-driver|tools/<package>)")
	fs.StringVar(&state, "state", "", "Filter by governance state (approved|denied|needs_review|deprecated)")
	fs.IntVar(&limit, "limit", 20, "Maximum records to return")
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	query := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if query == "" && framework == "" && ecosystem == "" && surface == "" && state == "" {
		return fmt.Errorf("usage: %s deps approved search <query> [--ecosystem npm|go|pip] [--framework react] [--surface ui|api|cli|playwright-driver|tools/<package>] [--state approved|denied] [--json]", support.AppName)
	}
	resp, err := governanceClient(core).SearchApprovedDependencies(context.Background(), connect.NewRequest(&governancev1.SearchApprovedDependenciesRequest{
		Query:     query,
		Ecosystem: ecosystem,
		Framework: framework,
		Surface:   surface,
		State:     state,
		Limit:     int32(limit),
	}))
	if err != nil {
		return cliapp.WrapAPIError("search approved dependencies", err, nil)
	}
	if jsonOutput {
		return printProto(resp.Msg)
	}
	return printSearchResults(resp.Msg.GetRecords(), resp.Msg.GetGuidance(), query)
}

// printSearchResults renders search hits with a governed footer: AI search over
// the governance corpus is NOT an exhaustive allowlist, so it always tells the
// agent how to propose a new package or install an approved one — steering every
// dependency action back through SDA rather than a raw package manager or a
// hand-edited JSON.
func printSearchResults(records []*governancev1.ApprovedDependencyRecord, guidance, query string) error {
	results := make([]string, 0, len(records))
	for _, record := range records {
		line := fmt.Sprintf("%s/%s %s [%s%s]", record.GetEcosystem(), record.GetPackageName(), record.GetVersionRange(), record.GetState(), rangePolicySuffix(record))
		if notes := strings.TrimSpace(record.GetSecurityNotes()); notes != "" {
			line += "\n    security: " + notes
		}
		if scenarios := record.GetExampleScenarios(); len(scenarios) > 0 {
			line += "\n    used by: " + strings.Join(scenarios, ", ")
		}
		results = append(results, line)
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Records: %d", len(records)),
			guidance,
			"This is not an exhaustive allowlist — a better package may exist.",
		},
		ResultsHeading: "Dependency Search",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("Install an approved package: %s deps install <ecosystem>/<package> --scenario <name> --surface <ui|api|cli|playwright-driver|tools/<package>>", support.AppName),
			fmt.Sprintf("Propose a new package from observed usage: %s deps approved approve-observed <ecosystem>/<package> --from-findings --apply", support.AppName),
			fmt.Sprintf("Inspect one record: %s deps approved explain <ecosystem>/<package>", support.AppName),
		},
	}
	return support.PrintList(false, report, nil)
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

func runFindings(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("deps approved findings")
	var jsonOutput bool
	var policyMode, scenario, ecosystem, packageName, severity, findingClass string
	var limit int
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	fs.StringVar(&policyMode, "policy-mode", "", "Override registry policy mode: advisory, strict, or review_gate")
	fs.StringVar(&scenario, "scenario", "", "Filter by scenario")
	fs.StringVar(&ecosystem, "ecosystem", "", "Filter by ecosystem")
	fs.StringVar(&packageName, "package", "", "Filter by package name")
	fs.StringVar(&severity, "severity", "", "Filter by severity")
	fs.StringVar(&findingClass, "class", "", "Filter by finding class")
	fs.IntVar(&limit, "limit", 40, "Maximum grouped findings in human output")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("usage: %s deps approved findings [--scenario <name>] [--ecosystem npm|go] [--package <name>] [--severity WARNING|ERROR] [--class <finding-class>] [--json]", support.AppName)
	}
	resp, err := governanceClient(core).ListApprovedDependencyFindings(context.Background(), connect.NewRequest(&governancev1.ListApprovedDependencyFindingsRequest{
		PolicyMode:   policyMode,
		Scenario:     scenario,
		Ecosystem:    ecosystem,
		PackageName:  packageName,
		Severity:     severity,
		FindingClass: findingClass,
	}))
	if err != nil {
		return cliapp.WrapAPIError("list approved dependency findings", err, nil)
	}
	if jsonOutput {
		return printProto(resp.Msg)
	}
	results := groupedFindingResults(resp.Msg.GetFindings(), limit)
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Findings: %d", resp.Msg.GetSummary().GetFindingCount()),
			fmt.Sprintf("Scenarios: %d", resp.Msg.GetSummary().GetScenarioCount()),
			fmt.Sprintf("Dependencies: %d", resp.Msg.GetSummary().GetDependencyCount()),
			fmt.Sprintf("Policy mode: %s", resp.Msg.GetSummary().GetPolicyMode()),
			resp.Msg.GetGuidance(),
		},
		ResultsHeading: "Governance Findings",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s deps approved findings --json", support.AppName),
			fmt.Sprintf("%s deps approved validate --all --json", support.AppName),
		},
	}
	return support.PrintList(false, report, nil)
}

type findingGroup struct {
	ecosystem   string
	name        string
	class       string
	severity    string
	count       int
	scenarios   map[string]struct{}
	versions    map[string]struct{}
	title       string
	remediation string
}

func groupedFindingResults(findings []*governancev1.ApprovedDependencyFinding, limit int) []string {
	groups := map[string]*findingGroup{}
	for _, finding := range findings {
		key := strings.Join([]string{finding.GetEcosystem(), finding.GetPackageName(), finding.GetFindingClass()}, "\x00")
		group := groups[key]
		if group == nil {
			group = &findingGroup{
				ecosystem:   finding.GetEcosystem(),
				name:        finding.GetPackageName(),
				class:       finding.GetFindingClass(),
				severity:    "INFO",
				scenarios:   map[string]struct{}{},
				versions:    map[string]struct{}{},
				title:       finding.GetTitle(),
				remediation: finding.GetRemediation(),
			}
			groups[key] = group
		}
		group.count++
		group.severity = cliMaxSeverity(group.severity, finding.GetSeverity())
		if finding.GetScenario() != "" {
			group.scenarios[finding.GetScenario()] = struct{}{}
		}
		if finding.GetObserved() != "" {
			group.versions[finding.GetObserved()] = struct{}{}
		}
	}
	outGroups := make([]*findingGroup, 0, len(groups))
	for _, group := range groups {
		outGroups = append(outGroups, group)
	}
	sort.Slice(outGroups, func(i, j int) bool {
		left := outGroups[i]
		right := outGroups[j]
		if cliSeverityRank(left.severity) != cliSeverityRank(right.severity) {
			return cliSeverityRank(left.severity) > cliSeverityRank(right.severity)
		}
		if left.count != right.count {
			return left.count > right.count
		}
		return left.ecosystem+"/"+left.name+"/"+left.class < right.ecosystem+"/"+right.name+"/"+right.class
	})
	results := make([]string, 0, len(outGroups))
	shown := len(outGroups)
	if limit > 0 && shown > limit {
		shown = limit
	}
	for _, group := range outGroups[:shown] {
		results = append(results, fmt.Sprintf(
			"%s %s: %s/%s findings=%d scenarios=%d versions=%s next=%s",
			group.severity,
			group.class,
			group.ecosystem,
			group.name,
			group.count,
			len(group.scenarios),
			sampleStrings(group.versions, 3),
			groupedFindingCommand(group),
		))
	}
	if hidden := len(outGroups) - shown; hidden > 0 {
		results = append(results, fmt.Sprintf("%d more grouped finding(s) hidden; rerun with --limit %d or --json", hidden, len(outGroups)))
	}
	return results
}

func groupedFindingCommand(group *findingGroup) string {
	if group == nil {
		return fmt.Sprintf("%s deps approved triage", support.AppName)
	}
	switch group.class {
	case "UNRECORDED_DIRECT":
		return fmt.Sprintf("%s deps approved usage %s/%s --json", support.AppName, group.ecosystem, group.name)
	case "VERSION_OUT_OF_RANGE":
		return fmt.Sprintf("%s deps approved approve %s/%s --range <reviewed-range> --rationale <text> --json", support.AppName, group.ecosystem, group.name)
	default:
		return fmt.Sprintf("%s deps approved triage --ecosystem %s --package %s --json", support.AppName, group.ecosystem, group.name)
	}
}

func cliMaxSeverity(left, right string) string {
	if cliSeverityRank(right) > cliSeverityRank(left) {
		return strings.ToUpper(right)
	}
	return strings.ToUpper(left)
}

func cliSeverityRank(value string) int {
	rank := map[string]int{"": 0, "INFO": 1, "WARNING": 2, "ERROR": 3, "BLOCKER": 4}
	return rank[strings.ToUpper(value)]
}

func sampleStrings(values map[string]struct{}, limit int) string {
	items := make([]string, 0, len(values))
	for value := range values {
		items = append(items, value)
	}
	sort.Strings(items)
	if len(items) == 0 {
		return "unknown"
	}
	if limit > 0 && len(items) > limit {
		items = append(items[:limit], fmt.Sprintf("+%d more", len(items)-limit))
	}
	return strings.Join(items, ",")
}

func runTriage(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("deps approved triage")
	var jsonOutput bool
	var policyMode, section, ecosystem, packageName string
	var limit int
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	fs.StringVar(&policyMode, "policy-mode", "", "Override registry policy mode: advisory, strict, or review_gate")
	fs.StringVar(&section, "section", "", "Filter by section: security, seeding, ranges, hotspots, or expired")
	fs.StringVar(&ecosystem, "ecosystem", "", "Filter by ecosystem")
	fs.StringVar(&packageName, "package", "", "Filter by package name")
	fs.IntVar(&limit, "limit", 10, "Maximum groups per section in human output")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("usage: %s deps approved triage [--section security|seeding|ranges|hotspots|expired] [--ecosystem npm|go] [--package <name>] [--limit 20] [--json]", support.AppName)
	}
	requestLimit := int32(limit)
	if jsonOutput {
		requestLimit = 0
	}
	resp, err := governanceClient(core).GetApprovedDependencyTriage(context.Background(), connect.NewRequest(&governancev1.GetApprovedDependencyTriageRequest{
		PolicyMode:  policyMode,
		Section:     section,
		Ecosystem:   ecosystem,
		PackageName: packageName,
		Limit:       requestLimit,
	}))
	if err != nil {
		return cliapp.WrapAPIError("show approved dependency triage", err, nil)
	}
	if jsonOutput {
		return printProto(resp.Msg)
	}
	return printTriage(resp.Msg, limit)
}

func runUsage(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("deps approved usage")
	var jsonOutput bool
	var policyMode string
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	fs.StringVar(&policyMode, "policy-mode", "", "Override registry policy mode: advisory, strict, or review_gate")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	positionals := fs.Args()
	if len(positionals) != 1 {
		return fmt.Errorf("usage: %s deps approved usage <ecosystem>/<package> [--policy-mode advisory|strict|review_gate] [--json]", support.AppName)
	}
	ecosystem, packageName, err := parseDependencyID(positionals[0])
	if err != nil {
		return err
	}
	resp, err := governanceClient(core).GetApprovedDependencyUsage(context.Background(), connect.NewRequest(&governancev1.GetApprovedDependencyUsageRequest{
		Ecosystem:   ecosystem,
		PackageName: packageName,
		PolicyMode:  policyMode,
	}))
	if err != nil {
		return cliapp.WrapAPIError("show approved dependency usage", err, nil)
	}
	if jsonOutput {
		return printProto(resp.Msg)
	}
	if !resp.Msg.GetFound() {
		report := cliapp.ListReport{
			Summary: []string{fmt.Sprintf("Dependency: %s/%s", ecosystem, packageName), "Found: false", resp.Msg.GetGuidance()},
			RetrievalHints: []string{
				fmt.Sprintf("%s deps approved findings --ecosystem %s --package %s --json", support.AppName, ecosystem, packageName),
				fmt.Sprintf("%s deps approved validate --all --json", support.AppName),
			},
		}
		return support.PrintList(false, report, nil)
	}
	group := resp.Msg.GetUsageGroup()
	results := make([]string, 0, len(group.GetObservedDependencies())+len(resp.Msg.GetFindings()))
	for _, dep := range group.GetObservedDependencies() {
		results = append(results, fmt.Sprintf("usage: %s %s %s (%s)", dep.GetFilePath(), dep.GetDependencyGroup(), dep.GetVersion(), dep.GetSurfaceId()))
	}
	for _, finding := range resp.Msg.GetFindings() {
		results = append(results, fmt.Sprintf("finding: %s %s in %s - %s", finding.GetSeverity(), finding.GetFindingClass(), finding.GetScenario(), finding.GetTitle()))
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Dependency: %s/%s", group.GetEcosystem(), group.GetPackageName()),
			fmt.Sprintf("Scenarios: %d", group.GetScenarioCount()),
			fmt.Sprintf("Usages: %d", group.GetUsageCount()),
			fmt.Sprintf("Findings: %d", group.GetFindingCount()),
			resp.Msg.GetGuidance(),
		},
		ResultsHeading: "Dependency Usage",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s deps approved usage %s/%s --json", support.AppName, ecosystem, packageName),
			fmt.Sprintf("%s deps approved approve %s/%s --rationale <text> --json", support.AppName, ecosystem, packageName),
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

func runProposeRecords(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("deps approved propose-records")
	var jsonOutput bool
	var policyMode, ecosystem, packageName, scenario, state, rangeStrategy string
	var topUnrecorded, minimumScenarioCount int
	var includeDev, includeRuntime bool
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	fs.StringVar(&policyMode, "policy-mode", "", "Override registry policy mode: advisory, strict, or review_gate")
	fs.IntVar(&topUnrecorded, "top-unrecorded", 25, "Maximum unrecorded dependency groups to propose")
	fs.StringVar(&ecosystem, "ecosystem", "", "Filter by ecosystem")
	fs.StringVar(&packageName, "package", "", "Filter by package name")
	fs.StringVar(&scenario, "scenario", "", "Filter by scenario")
	fs.BoolVar(&includeDev, "include-dev", false, "Include direct dev dependencies; default includes dev and runtime")
	fs.BoolVar(&includeRuntime, "include-runtime", false, "Include direct runtime dependencies; default includes dev and runtime")
	fs.IntVar(&minimumScenarioCount, "minimum-scenario-count", 0, "Require use across at least this many scenarios")
	fs.StringVar(&state, "state", "needs_review", "Proposed record state")
	fs.StringVar(&rangeStrategy, "range-strategy", "", "Version range strategy: observed, exact, or wildcard")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("usage: %s deps approved propose-records [--top-unrecorded 25] [--ecosystem npm|go] [--package <name>] [--json]", support.AppName)
	}
	resp, err := governanceClient(core).ProposeApprovedDependencyRecords(context.Background(), connect.NewRequest(&governancev1.ProposeApprovedDependencyRecordsRequest{
		PolicyMode:           policyMode,
		TopUnrecorded:        int32(topUnrecorded),
		Ecosystem:            ecosystem,
		PackageName:          packageName,
		Scenario:             scenario,
		IncludeDev:           includeDev,
		IncludeRuntime:       includeRuntime,
		MinimumScenarioCount: int32(minimumScenarioCount),
		State:                state,
		RangeStrategy:        rangeStrategy,
	}))
	if err != nil {
		return cliapp.WrapAPIError("propose approved dependency records", err, nil)
	}
	if jsonOutput {
		return printProto(resp.Msg)
	}
	return printProposals(resp.Msg)
}

func runUpsertBatch(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("deps approved upsert-batch")
	var filePath string
	var apply bool
	var dryRunFlag bool
	var jsonOutput bool
	fs.StringVar(&filePath, "file", "", "Path to proposal or BatchUpsertApprovedDependenciesRequest JSON")
	fs.BoolVar(&apply, "apply", false, "Apply the changes; default is dry-run preview")
	fs.BoolVar(&dryRunFlag, "dry-run", false, "Preview the changes without writing")
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 || strings.TrimSpace(filePath) == "" {
		return fmt.Errorf("usage: %s deps approved upsert-batch --file <proposals.json> [--apply|--dry-run] [--json]", support.AppName)
	}
	if apply && dryRunFlag {
		return fmt.Errorf("--apply and --dry-run cannot be used together")
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read approved dependency proposal file: %w", err)
	}
	var batch governancev1.BatchUpsertApprovedDependenciesRequest
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, &batch); err != nil {
		return fmt.Errorf("parse approved dependency batch JSON: %w", err)
	}
	if len(batch.GetRecords()) == 0 {
		return fmt.Errorf("approved dependency batch contains no records")
	}
	batch.DryRun = !apply
	resp, err := governanceClient(core).BatchUpsertApprovedDependencies(context.Background(), connect.NewRequest(&batch))
	if err != nil {
		return cliapp.WrapAPIError("batch upsert approved dependencies", err, nil)
	}
	if jsonOutput {
		return printProto(resp.Msg)
	}
	return printBatchUpsert(resp.Msg)
}

func runSecurityGaps(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("deps approved security-gaps")
	var jsonOutput bool
	var ecosystem, packageName, scenario, vulnerabilityID, minimumSeverity string
	var limit int
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	fs.StringVar(&ecosystem, "ecosystem", "", "Filter by ecosystem")
	fs.StringVar(&packageName, "package", "", "Filter by package name")
	fs.StringVar(&scenario, "scenario", "", "Filter by scenario")
	fs.StringVar(&vulnerabilityID, "vulnerability", "", "Filter by vulnerability id")
	fs.StringVar(&minimumSeverity, "minimum-severity", "", "Filter by minimum normalized severity")
	fs.IntVar(&limit, "limit", 25, "Maximum gaps to show")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("usage: %s deps approved security-gaps [--ecosystem npm|go] [--package <name>] [--scenario <name>] [--vulnerability <id>] [--minimum-severity high] [--limit 25] [--json]", support.AppName)
	}
	requestLimit := int32(limit)
	if jsonOutput {
		requestLimit = 0
	}
	resp, err := governanceClient(core).ListSecurityGovernanceGaps(context.Background(), connect.NewRequest(&governancev1.ListSecurityGovernanceGapsRequest{
		Ecosystem:       ecosystem,
		PackageName:     packageName,
		Scenario:        scenario,
		VulnerabilityId: vulnerabilityID,
		MinimumSeverity: minimumSeverity,
		Limit:           requestLimit,
	}))
	if err != nil {
		return cliapp.WrapAPIError("list security governance gaps", err, nil)
	}
	if jsonOutput {
		return printProto(resp.Msg)
	}
	return printSecurityGaps(resp.Msg, limit)
}

func runApproveObserved(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("deps approved approve-observed")
	var jsonOutput bool
	var apply bool
	var fromFindings bool
	var policyMode, rangeStrategy, rangePolicy, rationale, approvedBy string
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	fs.BoolVar(&apply, "apply", false, "Apply the change; default is dry-run preview")
	fs.BoolVar(&fromFindings, "from-findings", false, "Build evidence from fleet governance findings and observed usage")
	fs.StringVar(&policyMode, "policy-mode", "", "Override registry policy mode: advisory, strict, or review_gate")
	fs.StringVar(&rangeStrategy, "range-strategy", "", "Version range strategy: observed, exact, major_line, minimum, or wildcard")
	fs.StringVar(&rangePolicy, "range-policy", "", "Range policy override: exact, major_line, minimum, or dev_tooling")
	fs.StringVar(&rationale, "rationale", "", "Optional approval rationale")
	fs.StringVar(&approvedBy, "approved-by", "", "Reviewer or approving group")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	positionals := fs.Args()
	if len(positionals) != 1 {
		return fmt.Errorf("usage: %s deps approved approve-observed <ecosystem>/<package> [--from-findings] [--range-strategy observed|major_line|wildcard] [--apply] [--json]", support.AppName)
	}
	ecosystem, packageName, err := parseDependencyID(positionals[0])
	if err != nil {
		return err
	}
	resp, err := governanceClient(core).ApproveObservedDependency(context.Background(), connect.NewRequest(&governancev1.ApproveObservedDependencyRequest{
		Ecosystem:     ecosystem,
		PackageName:   packageName,
		PolicyMode:    policyMode,
		RangeStrategy: rangeStrategy,
		RangePolicy:   rangePolicy,
		Rationale:     rationale,
		ApprovedBy:    approvedBy,
		DryRun:        !apply,
		FromFindings:  fromFindings,
	}))
	if err != nil {
		return cliapp.WrapAPIError("approve observed dependency", err, nil)
	}
	if jsonOutput {
		return printProto(resp.Msg)
	}
	return printDecision("Approve Observed Dependency", resp.Msg)
}

func runWidenRange(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("deps approved widen-range")
	var jsonOutput bool
	var apply bool
	var toMajorLine bool
	var policyMode, rationale, approvedBy string
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	fs.BoolVar(&apply, "apply", false, "Apply the change; default is dry-run preview")
	fs.BoolVar(&toMajorLine, "to-major-line", false, "Widen the record to the observed major line")
	fs.StringVar(&policyMode, "policy-mode", "", "Override registry policy mode: advisory, strict, or review_gate")
	fs.StringVar(&rationale, "rationale", "", "Optional rationale for the range widening")
	fs.StringVar(&approvedBy, "approved-by", "", "Reviewer or approving group")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	positionals := fs.Args()
	if len(positionals) != 1 {
		return fmt.Errorf("usage: %s deps approved widen-range <ecosystem>/<package> --to-major-line [--apply] [--json]", support.AppName)
	}
	if !toMajorLine {
		return fmt.Errorf("--to-major-line is required")
	}
	ecosystem, packageName, err := parseDependencyID(positionals[0])
	if err != nil {
		return err
	}
	resp, err := governanceClient(core).WidenApprovedDependencyRange(context.Background(), connect.NewRequest(&governancev1.WidenApprovedDependencyRangeRequest{
		Ecosystem:    ecosystem,
		PackageName:  packageName,
		PolicyMode:   policyMode,
		TargetPolicy: "major_line",
		Rationale:    rationale,
		ApprovedBy:   approvedBy,
		DryRun:       !apply,
	}))
	if err != nil {
		return cliapp.WrapAPIError("widen approved dependency range", err, nil)
	}
	if jsonOutput {
		return printProto(resp.Msg)
	}
	return printDecision("Widen Approved Dependency Range", resp.Msg)
}

func runApprove(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("deps approved approve")
	var versionRange, rangePolicy, rationale, approvedBy, reviewExpires string
	var scenarios, useCases string
	var apply bool
	var jsonOutput bool
	fs.StringVar(&versionRange, "range", "*", "Approved version or version range")
	fs.StringVar(&rangePolicy, "range-policy", "", "Range policy: exact, major_line, minimum, or dev_tooling")
	fs.StringVar(&rationale, "rationale", "", "Approval rationale")
	fs.StringVar(&approvedBy, "approved-by", "", "Reviewer or approving group")
	fs.StringVar(&reviewExpires, "review-expires", "", "Review expiry date in YYYY-MM-DD format")
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
		Ecosystem:        ecosystem,
		PackageName:      packageName,
		VersionRange:     versionRange,
		RangePolicy:      rangePolicy,
		State:            "approved",
		UseCases:         splitCSV(useCases),
		Rationale:        rationale,
		ApprovedBy:       approvedBy,
		ReviewExpires:    reviewExpires,
		AllowedScenarios: splitCSV(scenarios),
	}
	return runMutation(core, record, !apply, jsonOutput)
}

func runDeny(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("deps approved deny")
	var versionRange, rangePolicy, rationale, replacement string
	var scenarios string
	var apply bool
	var jsonOutput bool
	fs.StringVar(&versionRange, "range", "*", "Denied version or version range")
	fs.StringVar(&rangePolicy, "range-policy", "", "Range policy: exact, major_line, minimum, dev_tooling, or security_denied")
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
		RangePolicy:      rangePolicy,
		State:            "denied",
		Rationale:        rationale,
		Replacement:      replacement,
		DeniedScenarios:  splitCSV(scenarios),
		AllowedScenarios: nil,
	}
	return runMutation(core, record, !apply, jsonOutput)
}

func runRemediate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("deps approved remediate")
	var vulnerabilityID string
	var jsonOutput bool
	fs.StringVar(&vulnerabilityID, "vulnerability", "", "Security vulnerability id to explain")
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	positionals := fs.Args()
	if len(positionals) != 1 || strings.TrimSpace(vulnerabilityID) == "" {
		return fmt.Errorf("usage: %s deps approved remediate <ecosystem>/<package> --vulnerability <id> [--json]", support.AppName)
	}
	ecosystem, packageName, err := parseDependencyID(positionals[0])
	if err != nil {
		return err
	}
	resp, err := governanceClient(core).PreviewVulnerabilityRemediation(context.Background(), connect.NewRequest(&governancev1.PreviewVulnerabilityRemediationRequest{
		Ecosystem:       ecosystem,
		PackageName:     packageName,
		VulnerabilityId: vulnerabilityID,
	}))
	if err != nil {
		return cliapp.WrapAPIError("preview vulnerable dependency remediation", err, nil)
	}
	if jsonOutput {
		return printProto(resp.Msg)
	}
	return printVulnerabilityRemediation(resp.Msg, false)
}

func runDenyVulnerable(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("deps approved deny-vulnerable")
	var vulnerabilityID, affectedRange, fixedRange, rationale, approvedBy string
	var apply bool
	var jsonOutput bool
	fs.StringVar(&vulnerabilityID, "vulnerability", "", "Security vulnerability id driving the denial")
	fs.StringVar(&affectedRange, "affected-range", "", "Affected version range to deny; defaults to Security Health evidence")
	fs.StringVar(&fixedRange, "fixed-range", "", "Fixed version range; defaults to Security Health evidence")
	fs.StringVar(&rationale, "reason", "", "Optional denial rationale; defaults to Security Health evidence")
	fs.StringVar(&approvedBy, "approved-by", "", "Reviewer or approving group")
	fs.BoolVar(&apply, "apply", false, "Apply the change; default is dry-run preview")
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	positionals := fs.Args()
	if len(positionals) != 1 || strings.TrimSpace(vulnerabilityID) == "" {
		return fmt.Errorf("usage: %s deps approved deny-vulnerable <ecosystem>/<package> --vulnerability <id> [--affected-range <range>] [--fixed-range <range>] [--apply] [--json]", support.AppName)
	}
	ecosystem, packageName, err := parseDependencyID(positionals[0])
	if err != nil {
		return err
	}
	resp, err := governanceClient(core).DenyVulnerableDependency(context.Background(), connect.NewRequest(&governancev1.DenyVulnerableDependencyRequest{
		Ecosystem:       ecosystem,
		PackageName:     packageName,
		VulnerabilityId: vulnerabilityID,
		AffectedRange:   affectedRange,
		FixedRange:      fixedRange,
		Rationale:       rationale,
		ApprovedBy:      approvedBy,
		DryRun:          !apply,
	}))
	if err != nil {
		return cliapp.WrapAPIError("deny vulnerable dependency", err, nil)
	}
	if jsonOutput {
		return printProto(resp.Msg)
	}
	return printVulnerabilityRemediation(resp.Msg, true)
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

func printVulnerabilityRemediation(resp *governancev1.VulnerabilityRemediationResponse, includeMutation bool) error {
	if !resp.GetFound() {
		report := cliapp.ListReport{
			Summary: []string{"Found: false", "Security Health did not return matching vulnerability evidence.", resp.GetGuidance()},
		}
		return support.PrintList(false, report, nil)
	}
	vuln := resp.GetVulnerability()
	summary := []string{
		fmt.Sprintf("Dependency: %s/%s", vuln.GetEcosystem(), vuln.GetPackageName()),
		fmt.Sprintf("Vulnerability: %s", vuln.GetVulnerabilityId()),
		fmt.Sprintf("Evidence: %s confidence, %s reachability, source=%s", nonEmpty(vuln.GetConfidence(), "unknown"), nonEmpty(vuln.GetReachability(), "unknown"), nonEmpty(vuln.GetSource(), "unknown")),
		fmt.Sprintf("Affected scenarios: %d", len(resp.GetAffectedScenarios())),
		resp.GetRemediation(),
	}
	if includeMutation && resp.GetMutation() != nil {
		summary = append(summary,
			resp.GetMutation().GetMessage(),
			fmt.Sprintf("Dry run: %t", resp.GetMutation().GetDryRun()),
			fmt.Sprintf("Changed: %t", resp.GetMutation().GetChanged()),
		)
	}
	results := make([]string, 0, len(resp.GetAffectedScenarios())+1)
	for _, scenario := range resp.GetAffectedScenarios() {
		results = append(results, "scenario: "+scenario)
	}
	if record := resp.GetSuggestedRecord(); record != nil {
		results = append(results, fmt.Sprintf("suggested decision: %s/%s %s [%s%s]", record.GetEcosystem(), record.GetPackageName(), record.GetVersionRange(), record.GetState(), rangePolicySuffix(record)))
	}
	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Security Governance Preview",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s deps approved deny-vulnerable %s/%s --vulnerability %s --json", support.AppName, vuln.GetEcosystem(), vuln.GetPackageName(), vuln.GetVulnerabilityId()),
			fmt.Sprintf("%s deps approved validate --all --json", support.AppName),
		},
	}
	return support.PrintList(false, report, nil)
}

func printProposals(resp *governancev1.ApprovedDependencyProposalResponse) error {
	results := make([]string, 0, len(resp.GetRecords())+len(resp.GetWarnings()))
	for _, record := range resp.GetRecords() {
		results = append(results, fmt.Sprintf("proposal: %s/%s %s [%s%s]", record.GetEcosystem(), record.GetPackageName(), record.GetVersionRange(), record.GetState(), rangePolicySuffix(record)))
	}
	for _, warning := range resp.GetWarnings() {
		results = append(results, "warning: "+warning)
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Proposed records: %d", len(resp.GetRecords())),
			fmt.Sprintf("Evidence groups: %d", len(resp.GetEvidenceGroups())),
			resp.GetGuidance(),
		},
		ResultsHeading: "Dependency Governance Proposals",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s deps approved propose-records --top-unrecorded 25 --json > proposals.json", support.AppName),
			fmt.Sprintf("%s deps approved upsert-batch --file proposals.json --dry-run --json", support.AppName),
		},
	}
	return support.PrintList(false, report, nil)
}

func printBatchUpsert(resp *governancev1.BatchUpsertApprovedDependenciesResponse) error {
	results := make([]string, 0, len(resp.GetMutations())+len(resp.GetWarnings()))
	for _, mutation := range resp.GetMutations() {
		record := mutation.GetRecord()
		results = append(results, fmt.Sprintf("mutation: %s/%s changed=%t %s", record.GetEcosystem(), record.GetPackageName(), mutation.GetChanged(), mutation.GetMessage()))
	}
	for _, warning := range resp.GetWarnings() {
		results = append(results, "warning: "+warning)
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Dry run: %t", resp.GetDryRun()),
			fmt.Sprintf("Changed: %t", resp.GetChanged()),
			fmt.Sprintf("Records touched: %d", len(resp.GetMutations())),
			fmt.Sprintf("Registry records: %d", governanceRecordCount(resp.GetSummary())),
			resp.GetGuidance(),
		},
		ResultsHeading: "Batch Upsert Results",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s deps approved validate --all --json", support.AppName),
			fmt.Sprintf("%s deps approved triage", support.AppName),
		},
	}
	return support.PrintList(false, report, nil)
}

func printDecision(title string, resp *governancev1.DependencyGovernanceDecisionResponse) error {
	record := resp.GetRecord()
	mutation := resp.GetMutation()
	group := resp.GetEvidenceGroup()
	results := make([]string, 0, len(resp.GetWarnings())+len(group.GetScenarios())+1)
	if record != nil {
		results = append(results, fmt.Sprintf("decision: %s/%s %s [%s%s]", record.GetEcosystem(), record.GetPackageName(), record.GetVersionRange(), record.GetState(), rangePolicySuffix(record)))
	}
	if group != nil {
		for _, scenario := range group.GetScenarios() {
			results = append(results, "scenario: "+scenario)
		}
	}
	for _, warning := range resp.GetWarnings() {
		results = append(results, "warning: "+warning)
	}
	summary := []string{title}
	if mutation != nil {
		summary = append(summary,
			mutation.GetMessage(),
			fmt.Sprintf("Dry run: %t", mutation.GetDryRun()),
			fmt.Sprintf("Changed: %t", mutation.GetChanged()),
		)
	}
	if group != nil {
		summary = append(summary,
			fmt.Sprintf("Observed scenarios: %d", group.GetScenarioCount()),
			fmt.Sprintf("Observed usages: %d", group.GetUsageCount()),
		)
	}
	summary = append(summary, resp.GetGuidance())
	hints := []string{fmt.Sprintf("%s deps approved validate --all --json", support.AppName)}
	if record != nil {
		hints = append([]string{fmt.Sprintf("%s deps approved explain %s/%s --json", support.AppName, record.GetEcosystem(), record.GetPackageName())}, hints...)
	}
	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Decision Evidence",
		Results:        results,
		RetrievalHints: hints,
	}
	return support.PrintList(false, report, nil)
}

func printSecurityGaps(resp *governancev1.SecurityGovernanceGapsResponse, limit int) error {
	results := make([]string, 0, len(resp.GetGaps())+len(resp.GetWarnings()))
	for _, gap := range resp.GetGaps() {
		coverage := "uncovered"
		if gap.GetDeniedRecordCovers() {
			coverage = "denied-covered"
		}
		overlap := ""
		if gap.GetApprovedRecordOverlaps() {
			overlap = " approved-overlap"
		}
		results = append(results, fmt.Sprintf(
			"%s/%s@%s %s [%s] scenarios=%d signal=%s %s%s next=%s",
			gap.GetEcosystem(),
			gap.GetPackageName(),
			nonEmpty(gap.GetObservedVersion(), "unknown"),
			strings.Join(gap.GetVulnerabilityIds(), ","),
			nonEmpty(gap.GetNormalizedSeverity(), "unknown"),
			len(gap.GetScenarios()),
			nonEmpty(gap.GetSignalCategory(), "unknown"),
			coverage,
			overlap,
			gap.GetSuggestedCommand(),
		))
	}
	if hidden := int(resp.GetTotal()) - len(resp.GetGaps()); limit > 0 && hidden > 0 {
		results = append(results, fmt.Sprintf("%d more security gap candidate(s) hidden; rerun with --limit %d or --json", hidden, int(resp.GetTotal())))
	}
	for _, warning := range resp.GetWarnings() {
		results = append(results, "warning: "+warning)
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Total vulnerability records: %d", resp.GetTotal()),
			fmt.Sprintf("Uncovered by denied governance: %d", resp.GetUncoveredCount()),
			fmt.Sprintf("Denied-covered: %d", resp.GetDeniedCoveredCount()),
			fmt.Sprintf("Approved overlaps: %d", resp.GetApprovedOverlapCount()),
			resp.GetGuidance(),
		},
		ResultsHeading: "Security Governance Gaps",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s deps approved security-gaps --json", support.AppName),
			fmt.Sprintf("%s deps approved deny-vulnerable npm/<package> --vulnerability <id> --json", support.AppName),
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

func printTriage(resp *governancev1.ApprovedDependencyTriageResponse, limit int) error {
	results := []string{}
	results = appendTriageResults(results, "Security Actions", resp.GetSecurityActions(), limit)
	results = appendTriageResults(results, "Registry Seeding", resp.GetRegistrySeeding(), limit)
	results = appendTriageResults(results, "Range Policy", resp.GetRangePolicy(), limit)
	results = appendTriageResults(results, "Scenario Hotspots", resp.GetScenarioHotspots(), limit)
	results = appendTriageResults(results, "Stale Or Expired Reviews", resp.GetStaleOrExpiredReviews(), limit)
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Status: %s", resp.GetSummary().GetStatus()),
			fmt.Sprintf("Findings: %d", resp.GetSummary().GetFindingCount()),
			fmt.Sprintf("Warnings: %d", resp.GetSummary().GetWarningCount()),
			fmt.Sprintf("Errors: %d", resp.GetSummary().GetErrorCount()),
			fmt.Sprintf("Policy mode: %s", resp.GetSummary().GetPolicyMode()),
			resp.GetGuidance(),
		},
		ResultsHeading: "Governance Triage",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s deps approved triage --json", support.AppName),
			fmt.Sprintf("%s deps approved findings --json", support.AppName),
		},
	}
	return support.PrintList(false, report, nil)
}

func appendTriageResults(results []string, heading string, groups []*governancev1.DependencyGovernanceTriageGroup, limit int) []string {
	if len(groups) == 0 {
		return results
	}
	results = append(results, heading)
	shown := len(groups)
	if limit > 0 && shown > limit {
		shown = limit
	}
	for _, group := range groups[:shown] {
		results = append(results, fmt.Sprintf(
			"%s/%s: %s, findings=%d, scenarios=%d, action=%s, next=%s",
			group.GetEcosystem(),
			group.GetPackageName(),
			group.GetHighestSeverity(),
			group.GetFindingCount(),
			group.GetScenarioCount(),
			group.GetActionType(),
			group.GetRecommendedCommand(),
		))
	}
	if hidden := len(groups) - shown; hidden > 0 {
		results = append(results, fmt.Sprintf("%s: %d more group(s) hidden; rerun with --limit %d or --json", heading, hidden, len(groups)))
	}
	return results
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

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
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
		results = append(results, fmt.Sprintf("%s/%s %s [%s%s]", record.GetEcosystem(), record.GetPackageName(), record.GetVersionRange(), record.GetState(), rangePolicySuffix(record)))
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

func rangePolicySuffix(record *governancev1.ApprovedDependencyRecord) string {
	if record == nil || strings.TrimSpace(record.GetRangePolicy()) == "" {
		return ""
	}
	return ", policy=" + record.GetRangePolicy()
}
