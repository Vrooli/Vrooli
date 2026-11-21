package scenario

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// Run executes scenario subcommands.
func Run(client *Client, args []string) error {
	if len(args) == 0 {
		return printUsage()
	}

	switch args[0] {
	case "list":
		return runList(client, args[1:])
	case "ports":
		return runPorts(client, args[1:])
	case "deps", "dependencies":
		return runDeps(client, args[1:])
	case "help", "-h", "--help":
		return printUsage()
	default:
		return fmt.Errorf("unknown subcommand: %s\n\nRun 'scenario-to-cloud scenario help' for usage", args[0])
	}
}

func printUsage() error {
	fmt.Println(`Usage: scenario-to-cloud scenario <command> [arguments]

Commands:
  list                    List all available scenarios
  ports <scenario-id>     Show port allocations for a scenario
  deps <scenario-id>      Show dependencies for a scenario

Run 'scenario-to-cloud scenario <command> -h' for command-specific options.`)
	return nil
}

func runList(client *Client, args []string) error {
	jsonOutput := false

	for _, arg := range args {
		switch arg {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud scenario list [flags]

Flags:
  --json    Output raw JSON`)
			return nil
		case "--json":
			jsonOutput = true
		}
	}

	body, resp, err := client.List()
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	// Pretty print
	if len(resp.Scenarios) == 0 {
		fmt.Println("No scenarios found.")
		return nil
	}

	fmt.Printf("Available Scenarios: %d\n", len(resp.Scenarios))
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("%-25s %-10s %-8s %s\n", "ID", "VERSION", "PARTS", "DESCRIPTION")

	for _, s := range resp.Scenarios {
		parts := []string{}
		if s.HasAPI {
			parts = append(parts, "API")
		}
		if s.HasUI {
			parts = append(parts, "UI")
		}
		if s.CLIInstalled {
			parts = append(parts, "CLI")
		}
		partsStr := strings.Join(parts, ",")
		if partsStr == "" {
			partsStr = "-"
		}
		desc := s.Description
		if len(desc) > 35 {
			desc = desc[:32] + "..."
		}
		fmt.Printf("%-25s %-10s %-8s %s\n", truncate(s.ID, 25), s.Version, partsStr, desc)
	}

	return nil
}

func runPorts(client *Client, args []string) error {
	var scenarioID string
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud scenario ports <scenario-id> [flags]

Flags:
  --json    Output raw JSON`)
			return nil
		case "--json":
			jsonOutput = true
		default:
			if !strings.HasPrefix(args[i], "-") && scenarioID == "" {
				scenarioID = args[i]
			}
		}
	}

	if scenarioID == "" {
		return fmt.Errorf("usage: scenario-to-cloud scenario ports <scenario-id>")
	}

	body, resp, err := client.Ports(scenarioID)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	// Pretty print
	fmt.Printf("Port Allocations for: %s\n", resp.ScenarioID)
	fmt.Println(strings.Repeat("-", 60))

	if len(resp.Ports) == 0 {
		fmt.Println("No ports allocated.")
		return nil
	}

	fmt.Printf("%-20s %-8s %-10s %-8s %s\n", "SERVICE", "PORT", "PROTOCOL", "PUBLIC", "PATH")
	for _, p := range resp.Ports {
		protocol := p.Protocol
		if protocol == "" {
			protocol = "tcp"
		}
		public := "no"
		if p.Public {
			public = "yes"
		}
		path := p.Path
		if path == "" {
			path = "-"
		}
		fmt.Printf("%-20s %-8d %-10s %-8s %s\n", truncate(p.Service, 20), p.Port, protocol, public, path)
	}

	return nil
}

func runDeps(client *Client, args []string) error {
	options, helpRequested := parseDepsOptions(args)
	if helpRequested {
		printDepsUsage()
		return nil
	}
	if options.scenarioID == "" {
		return fmt.Errorf("usage: scenario-to-cloud scenario deps <scenario-id>")
	}

	body, resp, err := client.Dependencies(options.scenarioID)
	if err != nil {
		return err
	}
	if options.jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	report, hasImpact := fetchDepsImpact(client, options)
	printDepsHeader(resp)
	if len(resp.Resources) == 0 && len(resp.Scenarios) == 0 {
		fmt.Println("No dependencies.")
		return nil
	}
	directRows, transitiveRows := dependencyRows(resp, report, hasImpact)
	printDependencyRows(directRows, transitiveRows)
	if !options.impactOutput {
		fmt.Println(strings.Repeat("-", 70))
		fmt.Println("Tip: add --impact for RAM/Disk/CPU summary, or --impact --verbose for per-dependency footprint details.")
		return nil
	}
	printDepsImpact(report, hasImpact, options.verbose, directRows, transitiveRows)
	return nil
}

type depsOptions struct {
	scenarioID   string
	jsonOutput   bool
	impactOutput bool
	verbose      bool
}

type depRow struct {
	Scope string
	Type  string
	Name  string
}

func parseDepsOptions(args []string) (depsOptions, bool) {
	var options depsOptions
	for _, arg := range args {
		switch arg {
		case "-h", "--help":
			return options, true
		case "--json":
			options.jsonOutput = true
		case "--impact":
			options.impactOutput = true
		case "--verbose":
			options.impactOutput = true
			options.verbose = true
		default:
			if !strings.HasPrefix(arg, "-") && options.scenarioID == "" {
				options.scenarioID = arg
			}
		}
	}
	return options, false
}

func printDepsUsage() {
	fmt.Println(`Usage: scenario-to-cloud scenario deps <scenario-id> [flags]

Flags:
  --json      Output raw JSON from /scenarios/:id/dependencies
  --impact    Show resource impact summary and per-dependency requirements
  --verbose   Alias for --impact`)
}

func fetchDepsImpact(client *Client, options depsOptions) (DeploymentReportResponse, bool) {
	if !options.impactOutput {
		return DeploymentReportResponse{}, false
	}
	_, report, err := client.DeploymentReport(context.Background(), options.scenarioID)
	return report, err == nil
}

func printDepsHeader(resp DepsResponse) {
	fmt.Printf("Dependencies for: %s\n", resp.ScenarioID)
	fmt.Println(strings.Repeat("-", 70))
	fmt.Printf("Source: %s (analyzer_available=%t)\n", resp.Source, resp.AnalyzerAvailable)
	fmt.Println(strings.Repeat("-", 70))
}

func dependencyRows(resp DepsResponse, report DeploymentReportResponse, hasImpact bool) ([]depRow, []depRow) {
	directSet := make(map[string]struct{}, len(resp.Resources)+len(resp.Scenarios))
	for _, name := range resp.Resources {
		directSet[keyForDep("resource", name)] = struct{}{}
	}
	for _, name := range resp.Scenarios {
		directSet[keyForDep("scenario", name)] = struct{}{}
	}

	directRows := make([]depRow, 0, len(resp.Resources)+len(resp.Scenarios))
	for _, name := range resp.Resources {
		directRows = append(directRows, depRow{Scope: "direct", Type: "resource", Name: name})
	}
	for _, name := range resp.Scenarios {
		directRows = append(directRows, depRow{Scope: "direct", Type: "scenario", Name: name})
	}
	sort.Slice(directRows, func(i, j int) bool {
		if directRows[i].Type == directRows[j].Type {
			return directRows[i].Name < directRows[j].Name
		}
		return directRows[i].Type < directRows[j].Type
	})

	transitiveRows := []depRow{}
	if hasImpact {
		transitiveRows = transitiveDependencyRows(report.Dependencies, directSet)
	}
	return directRows, transitiveRows
}

func transitiveDependencyRows(nodes []DeploymentDependencyNode, directSet map[string]struct{}) []depRow {
	rows := make([]depRow, 0)
	seen := make(map[string]struct{})
	var walk func(nodes []DeploymentDependencyNode, depth int)
	walk = func(nodes []DeploymentDependencyNode, depth int) {
		for _, node := range nodes {
			if depth > 0 {
				key := keyForDep(node.Type, node.Name)
				_, isDirect := directSet[key]
				_, exists := seen[key]
				if !isDirect && !exists {
					seen[key] = struct{}{}
					rows = append(rows, depRow{Scope: "transitive", Type: node.Type, Name: node.Name})
				}
			}
			walk(node.Children, depth+1)
		}
	}
	walk(nodes, 0)
	sortDepRows(rows)
	return rows
}

func sortDepRows(rows []depRow) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Type == rows[j].Type {
			return rows[i].Name < rows[j].Name
		}
		return rows[i].Type < rows[j].Type
	})
}

func printDependencyRows(directRows, transitiveRows []depRow) {
	fmt.Printf("%-12s %-12s %s\n", "SCOPE", "TYPE", "NAME")
	for _, row := range directRows {
		fmt.Printf("%-12s %-12s %s\n", row.Scope, row.Type, row.Name)
	}
	for _, row := range transitiveRows {
		fmt.Printf("%-12s %-12s %s\n", row.Scope, row.Type, row.Name)
	}
}

func printDepsImpact(report DeploymentReportResponse, hasImpact, verbose bool, directRows, transitiveRows []depRow) {
	fmt.Println(strings.Repeat("-", 70))
	if !hasImpact {
		fmt.Println("Impact summary unavailable: unable to fetch analyzer deployment report.")
		return
	}
	tier, aggregate, ok := chooseImpactTier(report.Aggregates)
	if !ok {
		fmt.Println("Impact summary unavailable: analyzer report has no aggregate tier requirements.")
		return
	}
	fmt.Printf("Impact Summary (%s): RAM %.0f MB, Disk %.0f MB, CPU %.2f cores\n",
		tier, aggregate.EstimatedRequirements.RAMMB, aggregate.EstimatedRequirements.DiskMB, aggregate.EstimatedRequirements.CPUCores)

	total, withReq := requirementCoverage(report.Dependencies)
	confidence := "unknown"
	if report.MetadataGaps != nil {
		if report.MetadataGaps.TotalGaps == 0 {
			confidence = "medium"
		} else {
			confidence = "low"
		}
	}
	fmt.Printf("Coverage: %d/%d dependencies have requirement footprints (confidence: %s)\n", withReq, total, confidence)

	if verbose {
		printVerboseDeps(report, tier, directRows, transitiveRows)
	}
}

func printVerboseDeps(report DeploymentReportResponse, tier string, directRows, transitiveRows []depRow) {
	fmt.Println(strings.Repeat("-", 70))
	fmt.Printf("%-12s %-12s %-28s %-8s %-8s %-8s\n", "SCOPE", "TYPE", "NAME", "RAM_MB", "DISK_MB", "CPU")
	nodeByKey := flattenNodeMap(report.Dependencies)
	for _, row := range append(directRows, transitiveRows...) {
		node, ok := nodeByKey[keyForDep(row.Type, row.Name)]
		if !ok {
			printVerboseDepRow(row, nil)
			continue
		}
		printVerboseDepRow(row, resolveRequirementForTier(node, tier))
	}
}

func printVerboseDepRow(row depRow, req *DeploymentRequirements) {
	if req == nil {
		fmt.Printf("%-12s %-12s %-28s %-8s %-8s %-8s\n", row.Scope, row.Type, truncate(row.Name, 28), "-", "-", "-")
		return
	}
	fmt.Printf("%-12s %-12s %-28s %-8s %-8s %-8s\n",
		row.Scope, row.Type, truncate(row.Name, 28),
		formatMaybe(req.RAMMB), formatMaybe(req.DiskMB), formatMaybe(req.CPUCores))
}

func keyForDep(depType, name string) string {
	return depType + ":" + name
}

func chooseImpactTier(aggregates map[string]DeploymentAggregate) (string, DeploymentAggregate, bool) {
	if len(aggregates) == 0 {
		return "", DeploymentAggregate{}, false
	}
	preferred := []string{"tier-4-saas", "saas", "server", "tier-1-local", "local", "desktop", "tier-2-desktop"}
	for _, tier := range preferred {
		agg, ok := aggregates[tier]
		if !ok {
			continue
		}
		return tier, agg, true
	}
	for tier, agg := range aggregates {
		return tier, agg, true
	}
	return "", DeploymentAggregate{}, false
}

func requirementCoverage(nodes []DeploymentDependencyNode) (total int, withRequirements int) {
	var walk func([]DeploymentDependencyNode)
	walk = func(items []DeploymentDependencyNode) {
		for _, n := range items {
			if !isDeployIntentNode(n) {
				walk(n.Children)
				continue
			}
			total++
			if n.Requirements != nil && (n.Requirements.RAMMB > 0 || n.Requirements.DiskMB > 0 || n.Requirements.CPUCores > 0) {
				withRequirements++
			}
			walk(n.Children)
		}
	}
	walk(nodes)
	return total, withRequirements
}

func isDeployIntentNode(n DeploymentDependencyNode) bool {
	if n.Required != nil && *n.Required {
		return true
	}
	if n.Enabled != nil && *n.Enabled {
		return true
	}
	if n.Required != nil || n.Enabled != nil {
		return false
	}
	return true
}

func flattenNodeMap(nodes []DeploymentDependencyNode) map[string]DeploymentDependencyNode {
	result := map[string]DeploymentDependencyNode{}
	var walk func([]DeploymentDependencyNode)
	walk = func(items []DeploymentDependencyNode) {
		for _, n := range items {
			key := keyForDep(n.Type, n.Name)
			if _, exists := result[key]; !exists {
				result[key] = n
			}
			walk(n.Children)
		}
	}
	walk(nodes)
	return result
}

func resolveRequirementForTier(node DeploymentDependencyNode, tier string) *DeploymentRequirements {
	if support, ok := node.TierSupport[tier]; ok && support.Requirements != nil {
		req := support.Requirements
		if req.RAMMB > 0 || req.DiskMB > 0 || req.CPUCores > 0 {
			return req
		}
	}
	if node.Requirements != nil && (node.Requirements.RAMMB > 0 || node.Requirements.DiskMB > 0 || node.Requirements.CPUCores > 0) {
		return node.Requirements
	}
	return nil
}

func formatMaybe(value float64) string {
	if value <= 0 {
		return "-"
	}
	if value == float64(int64(value)) {
		return strconv.FormatInt(int64(value), 10)
	}
	return strconv.FormatFloat(value, 'f', 2, 64)
}

// truncate shortens a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
