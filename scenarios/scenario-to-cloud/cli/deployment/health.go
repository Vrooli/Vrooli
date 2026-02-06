package deployment

import (
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func runHealth(client *Client, args []string) error {
	fs := flag.NewFlagSet("deployment health", flag.ContinueOnError)
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() != 1 {
		return fmt.Errorf("usage: scenario-to-cloud deployment health <id>")
	}

	body, resp, err := client.Health(fs.Arg(0))
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	printHealthReport(resp)
	return nil
}

func printHealthReport(resp HealthResponse) {
	// Header
	fmt.Printf("Deployment Health: %s (%s)\n", resp.DeploymentName, truncate(resp.DeploymentID, 8))
	meta := fmt.Sprintf("Scenario: %s", resp.ScenarioID)
	if resp.Domain != "" {
		meta += fmt.Sprintf("  |  Domain: %s", resp.Domain)
	}
	if resp.Host != "" {
		meta += fmt.Sprintf("  |  Host: %s", resp.Host)
	}
	fmt.Println(meta)

	// Top divider
	divider := strings.Repeat("=", 75)
	thinDivider := strings.Repeat("-", 75)
	fmt.Println(divider)

	// Overall status
	fmt.Printf("Overall: %s", strings.ToUpper(string(resp.Health)))
	fmt.Printf("%s%s\n", strings.Repeat(" ", 25-len(resp.Health)), resp.Summary)
	fmt.Println(thinDivider)

	// Sections
	for _, sec := range resp.Sections {
		fmt.Println()
		statusLabel := sectionStatusLabel(sec.Status)
		fmt.Printf("[%s] %s\n", statusLabel, sec.Title)

		for _, check := range sec.Checks {
			printCheck(check)
		}
	}

	// Recommendations
	if len(resp.Recommendations) > 0 {
		fmt.Println()
		fmt.Println(thinDivider)
		fmt.Println("Recommendations:")
		for i, rec := range resp.Recommendations {
			priority := recommendationPriority(rec.Priority)
			fmt.Printf("  %d. [%s] %s\n", i+1, priority, rec.Summary)
			if rec.Command != "" {
				fmt.Printf("     -> %s\n", rec.Command)
			}
		}
	}

	// Footer
	fmt.Println(thinDivider)
	fmt.Printf("Checked in %.1fs\n", float64(resp.DurationMs)/1000)
}

func printCheck(check HealthCheck) {
	fmt.Printf("  %s\n", check.Message)
}

func sectionStatusLabel(status string) string {
	switch status {
	case "pass":
		return "PASS"
	case "warn":
		return "WARN"
	case "fail":
		return "FAIL"
	case "skip":
		return "SKIP"
	case "error":
		return "ERROR"
	default:
		return strings.ToUpper(status)
	}
}

func recommendationPriority(p int) string {
	switch p {
	case 1:
		return "critical"
	case 2:
		return "important"
	default:
		return "suggestion"
	}
}
