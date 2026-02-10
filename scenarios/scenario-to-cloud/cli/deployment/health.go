package deployment

import (
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"

	"scenario-to-cloud/cli/internal/flagutil"
)

func runHealth(client *Client, args []string) error {
	fs := flag.NewFlagSet("deployment health", flag.ContinueOnError)
	selectorArgs := registerSelectorFlags(fs)
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	if err := flagutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	selectorFlagsUsed := selectorArgs.anySet()
	if fs.NArg() > 1 || (fs.NArg() == 1 && selectorFlagsUsed) || (fs.NArg() == 0 && !selectorFlagsUsed) {
		return fmt.Errorf("usage: scenario-to-cloud deployment health <id> OR scenario-to-cloud deployment health [--host <host> | --domain <domain> | --target <domain-or-host>] [--scenario <id>]")
	}

	id := ""
	if fs.NArg() == 1 {
		id = strings.TrimSpace(fs.Arg(0))
	} else {
		selector, err := selectorArgs.toSelector()
		if err != nil {
			return err
		}
		resolved, err := ResolveLatestBySelector(client, selector)
		if err != nil {
			return err
		}
		if resolved == nil {
			hostForManifest := displayOrNA(selector.Host)
			if hostForManifest == "n/a" {
				hostForManifest = displayOrNA(selector.Target)
			}
			return fmt.Errorf(
				"no deployment found for selector host=%s scenario=%s domain=%s\n\nNext steps:\n  1) Create a manifest:\n     scenario-to-cloud manifest init --scenario <scenario-id> --host %s --domain <domain> --out scenarios/<scenario-id>/.vrooli/cloud/manifest.prod.json\n  2) Validate it:\n     scenario-to-cloud manifest validate scenarios/<scenario-id>/.vrooli/cloud/manifest.prod.json\n  3) Deploy:\n     scenario-to-cloud redeploy scenarios/<scenario-id>/.vrooli/cloud/manifest.prod.json --if-needed --preflight --wait",
				displayOrNA(selector.Host),
				displayOrNA(selector.ScenarioID),
				displayOrNA(selector.Domain),
				hostForManifest,
			)
		}
		id = resolved.ID
	}

	body, resp, healthErr := client.Health(id)
	if healthErr != nil {
		return healthErr
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
	fmt.Printf("Deployment Health: %s\n", resp.DeploymentName)
	fmt.Printf("Deployment ID: %s\n", resp.DeploymentID)
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
		if strings.EqualFold(strings.TrimSpace(resp.Freshness.Status), "outdated") {
			if command := freshnessConvergenceCommand(resp); command != "" {
				fmt.Printf("  Next step: %s\n", command)
			}
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

func freshnessConvergenceCommand(resp HealthResponse) string {
	for _, recommendation := range resp.Recommendations {
		if strings.TrimSpace(recommendation.Command) == "" {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(recommendation.Category), "freshness") {
			return strings.TrimSpace(recommendation.Command)
		}
	}

	scenarioID := strings.TrimSpace(resp.ScenarioID)
	if scenarioID == "" {
		return ""
	}
	if domain := strings.TrimSpace(resp.Domain); domain != "" {
		return fmt.Sprintf("scenario-to-cloud redeploy --domain %s --scenario %s --if-needed --preflight --wait", domain, scenarioID)
	}
	if host := strings.TrimSpace(resp.Host); host != "" {
		return fmt.Sprintf("scenario-to-cloud redeploy --host %s --scenario %s --if-needed --preflight --wait", host, scenarioID)
	}
	return ""
}
