package deployment

import (
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func runHealth(client *Client, args []string) error {
	jsonOutput, id, err := parseHealthArgs(args)
	if err != nil {
		return err
	}
	if id == "" {
		return fmt.Errorf("usage: scenario-to-cloud deployment health <id>")
	}

	body, resp, healthErr := client.Health(id)
	if healthErr != nil {
		return healthErr
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	printHealthReport(resp)
	return nil
}

func parseHealthArgs(args []string) (bool, string, error) {
	var (
		jsonOutput  bool
		positionals []string
	)
	for _, arg := range args {
		switch arg {
		case "--json":
			jsonOutput = true
		case "-h", "--help":
			return false, "", fmt.Errorf("usage: scenario-to-cloud deployment health <id>")
		default:
			if strings.HasPrefix(arg, "-") {
				return false, "", fmt.Errorf("unknown flag: %s", arg)
			}
			positionals = append(positionals, arg)
		}
	}
	if len(positionals) != 1 {
		return false, "", nil
	}
	return jsonOutput, positionals[0], nil
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
	if resp.Freshness != nil {
		fmt.Printf("Freshness: %s", strings.ToUpper(resp.Freshness.Status))
		if resp.Freshness.Summary != "" {
			fmt.Printf("%s%s", strings.Repeat(" ", 23-len(resp.Freshness.Status)), resp.Freshness.Summary)
		}
		fmt.Println()
		if resp.Freshness.LocalVersion != "" || resp.Freshness.DeployedVersion != "" {
			fmt.Printf("  Version: local=%s deployed=%s (%s)\n",
				displayOrNA(resp.Freshness.LocalVersion),
				displayOrNA(resp.Freshness.DeployedVersion),
				displayOrNA(resp.Freshness.VersionStatus),
			)
		}
		if resp.Freshness.LocalBundleSHA256 != "" || resp.Freshness.DeployedBundleSHA256 != "" {
			fmt.Printf("  Fingerprint: local=%s deployed=%s (%s)\n",
				shortHashOrNA(resp.Freshness.LocalBundleSHA256),
				shortHashOrNA(resp.Freshness.DeployedBundleSHA256),
				displayOrNA(resp.Freshness.FingerprintStatus),
			)
		}
		for _, note := range resp.Freshness.Notes {
			fmt.Printf("  Note: %s\n", note)
		}
	}
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

func displayOrNA(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "n/a"
	}
	return v
}

func shortHashOrNA(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "n/a"
	}
	if len(v) <= 12 {
		return v
	}
	return v[:12]
}
