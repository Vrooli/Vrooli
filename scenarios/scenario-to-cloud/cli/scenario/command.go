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
	var scenarioID string
	jsonOutput := false
	impactOutput := false
	verbose := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud scenario deps <scenario-id> [flags]

Flags:
  --json      Output raw JSON from /scenarios/:id/dependencies
  --impact    Show resource impact summary and per-dependency requirements
  --verbose   Alias for --impact`)
			return nil
		case "--json":
			jsonOutput = true
		case "--impact":
			impactOutput = true
		case "--verbose":
			impactOutput = true
			verbose = true
		default:
			if !strings.HasPrefix(args[i], "-") && scenarioID == "" {
				scenarioID = args[i]
			}
		}
	}

	if scenarioID == "" {
		return fmt.Errorf("usage: scenario-to-cloud scenario deps <scenario-id>")
	}

	body, resp, err := client.Dependencies(scenarioID)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var (
		report    DeploymentReportResponse
		hasImpact bool
	)
	if impactOutput {
		_, r, err := client.DeploymentReport(context.Background(), scenarioID)
		if err == nil {
			report = r
			hasImpact = true
		}
	}

	// Pretty print
	fmt.Printf("Dependencies for: %s\n", resp.ScenarioID)
	fmt.Println(strings.Repeat("-", 70))
	fmt.Printf("Source: %s (analyzer_available=%t)\n", resp.Source, resp.AnalyzerAvailable)
	fmt.Println(strings.Repeat("-", 70))

	if len(resp.Resources) == 0 && len(resp.Scenarios) == 0 {
		fmt.Println("No dependencies.")
		return nil
	}

	directSet := make(map[string]struct{}, len(resp.Resources)+len(resp.Scenarios))
	for _, name := range resp.Resources {
		directSet[keyForDep("resource", name)] = struct{}{}
	}
	for _, name := range resp.Scenarios {
		directSet[keyForDep("scenario", name)] = struct{}{}
	}

	type depRow struct {
		Scope string
		Type  string
		Name  string
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

	transitiveRows := make([]depRow, 0)
	if hasImpact {
		seen := make(map[string]struct{})
		var walk func(nodes []DeploymentDependencyNode, depth int)
		walk = func(nodes []DeploymentDependencyNode, depth int) {
			for _, n := range nodes {
				if depth > 0 {
					key := keyForDep(n.Type, n.Name)
					if _, isDirect := directSet[key]; !isDirect {
						if _, exists := seen[key]; !exists {
							seen[key] = struct{}{}
							transitiveRows = append(transitiveRows, depRow{Scope: "transitive", Type: n.Type, Name: n.Name})
						}
					}
				}
				walk(n.Children, depth+1)
			}
		}
		walk(report.Dependencies, 0)
		sort.Slice(transitiveRows, func(i, j int) bool {
			if transitiveRows[i].Type == transitiveRows[j].Type {
				return transitiveRows[i].Name < transitiveRows[j].Name
			}
			return transitiveRows[i].Type < transitiveRows[j].Type
		})
	}

	fmt.Printf("%-12s %-12s %s\n", "SCOPE", "TYPE", "NAME")
	for _, row := range directRows {
		fmt.Printf("%-12s %-12s %s\n", row.Scope, row.Type, row.Name)
	}
	for _, row := range transitiveRows {
		fmt.Printf("%-12s %-12s %s\n", row.Scope, row.Type, row.Name)
	}
	if !impactOutput {
		fmt.Println(strings.Repeat("-", 70))
		fmt.Println("Tip: add --impact for RAM/Disk/CPU summary, or --impact --verbose for per-dependency footprint details.")
	}

	if impactOutput {
		fmt.Println(strings.Repeat("-", 70))
		if !hasImpact {
			fmt.Println("Impact summary unavailable: unable to fetch analyzer deployment report.")
			return nil
		}
		tier, aggregate, ok := chooseImpactTier(report.Aggregates)
		if !ok {
			fmt.Println("Impact summary unavailable: analyzer report has no aggregate tier requirements.")
			return nil
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
			fmt.Println(strings.Repeat("-", 70))
			fmt.Printf("%-12s %-12s %-28s %-8s %-8s %-8s\n", "SCOPE", "TYPE", "NAME", "RAM_MB", "DISK_MB", "CPU")
			nodeByKey := flattenNodeMap(report.Dependencies)
			for _, row := range append(directRows, transitiveRows...) {
				node, ok := nodeByKey[keyForDep(row.Type, row.Name)]
				if !ok {
					fmt.Printf("%-12s %-12s %-28s %-8s %-8s %-8s\n", row.Scope, row.Type, truncate(row.Name, 28), "-", "-", "-")
					continue
				}
				req := resolveRequirementForTier(node, tier)
				if req == nil {
					fmt.Printf("%-12s %-12s %-28s %-8s %-8s %-8s\n", row.Scope, row.Type, truncate(row.Name, 28), "-", "-", "-")
					continue
				}
				fmt.Printf("%-12s %-12s %-28s %-8s %-8s %-8s\n",
					row.Scope, row.Type, truncate(row.Name, 28),
					formatMaybe(req.RAMMB), formatMaybe(req.DiskMB), formatMaybe(req.CPUCores))
			}
		}
	}

	return nil
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
