// Package redeploy provides the convenience redeploy workflow command.
package redeploy

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"

	"scenario-to-cloud/cli/deployment"
)

// Run executes the redeploy workflow: create/update → execute → report.
func Run(client *deployment.Client, args []string) error {
	fs := flag.NewFlagSet("redeploy", flag.ContinueOnError)
	name := fs.String("name", "", "Optional deployment name")
	host := fs.String("host", "", "VPS host selector (existing deployment mode)")
	scenarioID := fs.String("scenario", "", "Scenario ID selector (existing deployment mode)")
	domain := fs.String("domain", "", "Domain selector (existing deployment mode)")
	target := fs.String("target", "", "Convenience selector (domain or host, existing deployment mode)")
	preflight := fs.Bool("preflight", false, "Run VPS preflight checks")
	forceBuild := fs.Bool("force-bundle", false, "Force rebuild of bundle")
	forceRun := fs.Bool("force-run", false, "Selector mode only: execute existing deployment immediately (requires explicit opt-in)")
	ifNeeded := fs.Bool("if-needed", false, "Deploy only when missing, unhealthy, or outdated")
	wait := fs.Bool("wait", false, "Wait for completion and print stage durations")
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	selectorMode := selectorFlagsSet(*host, *scenarioID, *domain, *target)
	if fs.NArg() > 1 || (fs.NArg() == 1 && selectorMode) || (fs.NArg() == 0 && !selectorMode) {
		return printUsage()
	}

	if selectorMode {
		if *ifNeeded && *forceRun {
			return fmt.Errorf("--if-needed and --force-run cannot be combined")
		}
		if !*ifNeeded && !*forceRun {
			return fmt.Errorf("selector mode requires --if-needed or --force-run to avoid unsafe ad-hoc execution")
		}
		selector, err := toSelector(*host, *scenarioID, *domain, *target)
		if err != nil {
			return err
		}
		if *forceRun {
			return runSelectorForce(client, selector, *preflight, *forceBuild, *wait, *jsonOutput)
		}
		return runSelectorIfNeeded(client, selector, *preflight, *forceBuild, *wait, *jsonOutput)
	}

	manifestPath := fs.Arg(0)
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}
	if *ifNeeded {
		return runIfNeeded(client, manifestPath, manifestBytes, *name, *preflight, *forceBuild, *wait, *jsonOutput)
	}

	return createAndExecute(client, manifestBytes, *name, *preflight, *forceBuild, *wait, *jsonOutput, map[string]interface{}{
		"mode":      "manifest_direct",
		"action":    "create_execute",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func runSelectorIfNeeded(client *deployment.Client, selector deployment.ManifestSelector, preflight bool, forceBuild bool, wait bool, jsonOutput bool) error {
	existing, err := deployment.ResolveLatestBySelector(client, selector)
	if err != nil {
		return fmt.Errorf("resolve deployment: %w", err)
	}
	if existing == nil {
		return fmt.Errorf(
			"no deployment found for selector host=%s scenario=%s domain=%s target=%s\n\nNext steps:\n  1) Create a manifest:\n     scenario-to-cloud manifest init --scenario <scenario-id> --host <host> --domain <domain> --out scenarios/<scenario-id>/.vrooli/cloud/manifest.prod.json\n  2) Validate it:\n     scenario-to-cloud manifest validate scenarios/<scenario-id>/.vrooli/cloud/manifest.prod.json\n  3) Deploy:\n     scenario-to-cloud redeploy scenarios/<scenario-id>/.vrooli/cloud/manifest.prod.json --if-needed --preflight --wait",
			displayOrNA(selector.Host),
			displayOrNA(selector.ScenarioID),
			displayOrNA(selector.Domain),
			displayOrNA(selector.Target),
		)
	}

	if !jsonOutput {
		fmt.Printf("Found existing deployment by selector: %s (%s)\n", existing.ID, existing.Status)
	}
	_, health, err := client.Health(existing.ID)
	if err != nil {
		return fmt.Errorf("deployment health: %w", err)
	}

	healthState := strings.ToLower(strings.TrimSpace(health.Health))
	freshnessState := ""
	if health.Freshness != nil {
		freshnessState = strings.ToLower(strings.TrimSpace(health.Freshness.Status))
	}

	if healthState == "healthy" && freshnessState == "current" && !forceBuild {
		if jsonOutput {
			return printJSON(map[string]interface{}{
				"mode":                "selector_if_needed",
				"selector":            selector,
				"resolved_deployment": existing,
				"health":              health,
				"action":              "noop",
				"timestamp":           time.Now().UTC().Format(time.RFC3339),
			})
		}
		fmt.Println("Deployment is healthy and current. No redeploy needed.")
		fmt.Printf("  Verify: scenario-to-cloud deployment health %s\n", existing.ID)
		return nil
	}

	if healthState == "stopped" && freshnessState == "current" && !forceBuild {
		if !jsonOutput {
			fmt.Println("Deployment is stopped but current. Starting deployment...")
		}
		_, startResp, err := client.Start(existing.ID, deployment.ExecuteRequest{})
		if err != nil {
			return fmt.Errorf("start deployment: %w", err)
		}
		if !jsonOutput {
			fmt.Printf("Start initiated (run_id: %s)\n", startResp.RunID)
		}
		if wait && jsonOutput {
			waitResult, waitErr := waitForCompletionQuiet(client, existing.ID)
			out := map[string]interface{}{
				"mode":                "selector_if_needed",
				"selector":            selector,
				"resolved_deployment": existing,
				"health":              health,
				"action":              "start",
				"start":               startResp,
				"wait":                waitResult,
				"timestamp":           time.Now().UTC().Format(time.RFC3339),
			}
			if waitErr != nil {
				_ = printJSON(out)
				return waitErr
			}
			return printJSON(out)
		}
		if wait {
			return deployment.WaitForDeploymentCompletion(client, existing.ID)
		}
		if jsonOutput {
			return printJSON(map[string]interface{}{
				"mode":                "selector_if_needed",
				"selector":            selector,
				"resolved_deployment": existing,
				"health":              health,
				"action":              "start",
				"start":               startResp,
				"timestamp":           time.Now().UTC().Format(time.RFC3339),
			})
		}
		fmt.Printf("  Check status:  scenario-to-cloud deployment get %s\n", existing.ID)
		return nil
	}

	derivedForceBuild := forceBuild || freshnessState == "outdated"
	if freshnessState == "outdated" && !forceBuild && !jsonOutput {
		fmt.Println("Deployment is outdated relative to local scenario state; forcing bundle rebuild.")
	}

	if !jsonOutput {
		fmt.Println("Redeploy required; executing existing deployment...")
	}
	return executeExisting(client, existing.ID, preflight, derivedForceBuild, wait, jsonOutput, map[string]interface{}{
		"mode":                "selector_if_needed",
		"selector":            selector,
		"resolved_deployment": existing,
		"health":              health,
		"action":              "execute",
	})
}

func runSelectorForce(client *deployment.Client, selector deployment.ManifestSelector, preflight bool, forceBuild bool, wait bool, jsonOutput bool) error {
	existing, err := deployment.ResolveLatestBySelector(client, selector)
	if err != nil {
		return fmt.Errorf("resolve deployment: %w", err)
	}
	if existing == nil {
		return fmt.Errorf(
			"no deployment found for selector host=%s scenario=%s domain=%s target=%s\n\nNext steps:\n  1) Create a manifest:\n     scenario-to-cloud manifest init --scenario <scenario-id> --host <host> --domain <domain> --out scenarios/<scenario-id>/.vrooli/cloud/manifest.prod.json\n  2) Validate it:\n     scenario-to-cloud manifest validate scenarios/<scenario-id>/.vrooli/cloud/manifest.prod.json\n  3) Deploy:\n     scenario-to-cloud redeploy scenarios/<scenario-id>/.vrooli/cloud/manifest.prod.json --if-needed --preflight --wait",
			displayOrNA(selector.Host),
			displayOrNA(selector.ScenarioID),
			displayOrNA(selector.Domain),
			displayOrNA(selector.Target),
		)
	}

	if !jsonOutput {
		fmt.Printf("Found existing deployment by selector: %s (%s)\n", existing.ID, existing.Status)
		fmt.Println("Force run requested; executing existing deployment now.")
	}
	return executeExisting(client, existing.ID, preflight, forceBuild, wait, jsonOutput, map[string]interface{}{
		"mode":                "selector_force",
		"selector":            selector,
		"resolved_deployment": existing,
		"action":              "execute",
	})
}

func runIfNeeded(client *deployment.Client, manifestPath string, manifestBytes []byte, name string, preflight bool, forceBuild bool, wait bool, jsonOutput bool) error {
	selector, err := deployment.ReadSelectorFromManifest(manifestPath)
	if err != nil {
		return err
	}

	existing, err := deployment.ResolveLatestBySelector(client, selector)
	if err != nil {
		return fmt.Errorf("resolve deployment: %w", err)
	}

	if existing == nil {
		if !jsonOutput {
			fmt.Println("No matching deployment found for manifest target; creating a new deployment.")
		}
		return createAndExecute(client, manifestBytes, name, preflight, forceBuild, wait, jsonOutput, map[string]interface{}{
			"mode":      "manifest_if_needed",
			"selector":  selector,
			"action":    "create_execute",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	}

	if !jsonOutput {
		fmt.Printf("Found existing deployment: %s (%s)\n", existing.ID, existing.Status)
	}
	_, health, err := client.Health(existing.ID)
	if err != nil {
		return fmt.Errorf("deployment health: %w", err)
	}

	healthState := strings.ToLower(strings.TrimSpace(health.Health))
	freshnessState := ""
	if health.Freshness != nil {
		freshnessState = strings.ToLower(strings.TrimSpace(health.Freshness.Status))
	}

	if healthState == "healthy" && freshnessState == "current" && !forceBuild {
		if jsonOutput {
			return printJSON(map[string]interface{}{
				"mode":                "manifest_if_needed",
				"selector":            selector,
				"resolved_deployment": existing,
				"health":              health,
				"action":              "noop",
				"timestamp":           time.Now().UTC().Format(time.RFC3339),
			})
		}
		fmt.Println("Deployment is healthy and current. No redeploy needed.")
		fmt.Printf("  Verify: scenario-to-cloud deployment health %s\n", existing.ID)
		return nil
	}

	if healthState == "stopped" && freshnessState == "current" && !forceBuild {
		if !jsonOutput {
			fmt.Println("Deployment is stopped but current. Starting deployment...")
		}
		_, startResp, err := client.Start(existing.ID, deployment.ExecuteRequest{})
		if err != nil {
			return fmt.Errorf("start deployment: %w", err)
		}
		if !jsonOutput {
			fmt.Printf("Start initiated (run_id: %s)\n", startResp.RunID)
		}
		if wait && jsonOutput {
			waitResult, waitErr := waitForCompletionQuiet(client, existing.ID)
			out := map[string]interface{}{
				"mode":                "manifest_if_needed",
				"selector":            selector,
				"resolved_deployment": existing,
				"health":              health,
				"action":              "start",
				"start":               startResp,
				"wait":                waitResult,
				"timestamp":           time.Now().UTC().Format(time.RFC3339),
			}
			if waitErr != nil {
				_ = printJSON(out)
				return waitErr
			}
			return printJSON(out)
		}
		if wait {
			return deployment.WaitForDeploymentCompletion(client, existing.ID)
		}
		if jsonOutput {
			return printJSON(map[string]interface{}{
				"mode":                "manifest_if_needed",
				"selector":            selector,
				"resolved_deployment": existing,
				"health":              health,
				"action":              "start",
				"start":               startResp,
				"timestamp":           time.Now().UTC().Format(time.RFC3339),
			})
		}
		fmt.Printf("  Check status:  scenario-to-cloud deployment get %s\n", existing.ID)
		return nil
	}

	derivedForceBuild := forceBuild || freshnessState == "outdated"
	if freshnessState == "outdated" && !forceBuild && !jsonOutput {
		fmt.Println("Deployment is outdated relative to local scenario state; forcing bundle rebuild.")
	}

	if !jsonOutput {
		fmt.Println("Redeploy required; updating deployment record and executing...")
	}
	return createAndExecute(client, manifestBytes, name, preflight, derivedForceBuild, wait, jsonOutput, map[string]interface{}{
		"mode":                "manifest_if_needed",
		"selector":            selector,
		"resolved_deployment": existing,
		"health":              health,
		"action":              "create_execute",
		"timestamp":           time.Now().UTC().Format(time.RFC3339),
	})
}

func executeExisting(client *deployment.Client, deploymentID string, preflight bool, forceBuild bool, wait bool, jsonOutput bool, envelope map[string]interface{}) error {
	execReq := deployment.ExecuteRequest{
		RunPreflight:     preflight,
		ForceBundleBuild: forceBuild,
	}
	execBody, execResp, err := client.Execute(deploymentID, execReq)
	if err != nil {
		return fmt.Errorf("execute deployment: %w", err)
	}
	if !jsonOutput {
		fmt.Printf("Execution started (run_id: %s)\n", execResp.RunID)
	}
	if wait && jsonOutput {
		waitResult, waitErr := waitForCompletionQuiet(client, deploymentID)
		out := envelopeWithCommon(envelope, map[string]interface{}{
			"execute":   json.RawMessage(execBody),
			"run_id":    execResp.RunID,
			"wait":      waitResult,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		if waitErr != nil {
			_ = printJSON(out)
			return waitErr
		}
		return printJSON(out)
	}
	if wait {
		return deployment.WaitForDeploymentCompletion(client, deploymentID)
	}
	if jsonOutput {
		return printJSON(envelopeWithCommon(envelope, map[string]interface{}{
			"execute":   json.RawMessage(execBody),
			"run_id":    execResp.RunID,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}))
	}
	fmt.Println()
	fmt.Println("Deployment is running in the background.")
	fmt.Printf("  Check status:  scenario-to-cloud deployment get %s\n", deploymentID)
	fmt.Printf("  View history:  scenario-to-cloud deployment history %s\n", deploymentID)
	fmt.Printf("  Stop:          scenario-to-cloud deployment stop %s\n", deploymentID)
	return nil
}

func createAndExecute(client *deployment.Client, manifestBytes []byte, name string, preflight bool, forceBuild bool, wait bool, jsonOutput bool, envelope map[string]interface{}) error {
	// Step 1: Create or update the deployment
	if !jsonOutput {
		fmt.Println("Creating/updating deployment...")
	}

	createReq := deployment.CreateRequest{
		Name:     name,
		Manifest: json.RawMessage(manifestBytes),
	}

	createBody, createResp, err := client.Create(createReq)
	if err != nil {
		return fmt.Errorf("create deployment: %w", err)
	}

	dep := createResp.Deployment
	if !jsonOutput {
		action := "Created"
		if createResp.Updated {
			action = "Updated"
		}
		fmt.Printf("%s deployment: %s (%s)\n", action, dep.ID, dep.Name)
	}

	// Step 2: Execute the deployment
	if !jsonOutput {
		fmt.Println("Starting deployment execution...")
	}

	execReq := deployment.ExecuteRequest{
		RunPreflight:     preflight,
		ForceBundleBuild: forceBuild,
	}

	execBody, execResp, err := client.Execute(dep.ID, execReq)
	if err != nil {
		return fmt.Errorf("execute deployment: %w", err)
	}

	if !jsonOutput {
		fmt.Printf("Execution started (run_id: %s)\n", execResp.RunID)
	}

	// If JSON output, return a combined response
	if jsonOutput {
		combined := envelopeWithCommon(envelope, map[string]interface{}{
			"create":    json.RawMessage(createBody),
			"execute":   json.RawMessage(execBody),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		if wait {
			waitResult, waitErr := waitForCompletionQuiet(client, dep.ID)
			combined["wait"] = waitResult
			if err := printJSON(combined); err != nil {
				return err
			}
			return waitErr
		}
		return printJSON(combined)
	}

	if wait {
		return deployment.WaitForDeploymentCompletion(client, dep.ID)
	}

	// Print helpful next steps
	fmt.Println()
	fmt.Println("Deployment is running in the background.")
	fmt.Printf("  Check status:  scenario-to-cloud deployment get %s\n", dep.ID)
	fmt.Printf("  View history:  scenario-to-cloud deployment history %s\n", dep.ID)
	fmt.Printf("  Stop:          scenario-to-cloud deployment stop %s\n", dep.ID)

	return nil
}

func selectorFlagsSet(host string, scenarioID string, domain string, target string) bool {
	return strings.TrimSpace(host) != "" ||
		strings.TrimSpace(scenarioID) != "" ||
		strings.TrimSpace(domain) != "" ||
		strings.TrimSpace(target) != ""
}

func toSelector(host string, scenarioID string, domain string, target string) (deployment.ManifestSelector, error) {
	host = strings.TrimSpace(host)
	scenarioID = strings.TrimSpace(scenarioID)
	domain = strings.TrimSpace(domain)
	target = strings.TrimSpace(target)
	if target != "" && (host != "" || domain != "") {
		return deployment.ManifestSelector{}, fmt.Errorf("--target cannot be combined with --host or --domain")
	}
	if host == "" && domain == "" && target == "" {
		return deployment.ManifestSelector{}, fmt.Errorf("at least one selector is required: --host, --domain, or --target")
	}
	return deployment.ManifestSelector{
		Host:       host,
		ScenarioID: scenarioID,
		Domain:     domain,
		Target:     target,
	}, nil
}

func displayOrNA(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "n/a"
	}
	return v
}

func printUsage() error {
	fmt.Println(`Usage:
  scenario-to-cloud redeploy <manifest.json> [options]
  scenario-to-cloud redeploy --domain <domain> --scenario <id> --if-needed [options]
  scenario-to-cloud redeploy --host <host> [--scenario <id>] --if-needed [options]
  scenario-to-cloud redeploy --target <domain-or-host> [--scenario <id>] --if-needed [options]

Convenience command that creates/updates a deployment and executes it.
Selector mode operates on an existing deployment and does not require a local manifest.

	Options:
	  --name <name>       Optional deployment name
	  --host <host>       VPS host selector (existing deployment mode)
	  --scenario <id>     Scenario ID selector (existing deployment mode)
	  --domain <domain>   Domain selector (existing deployment mode)
	  --target <value>    Convenience selector (domain or host, existing deployment mode)
	  --preflight         Run VPS preflight checks before deployment
	  --force-bundle      Force rebuild of bundle even if one exists
	  --if-needed         Selector mode: only execute when deployment is missing, unhealthy, stopped, or outdated
	  --force-run         Selector mode: execute existing deployment immediately (explicit operator override)
	  --wait              Wait for completion and print stage durations
	  --json              Output raw JSON

	Examples:
	  scenario-to-cloud redeploy cloud-manifest.json
	  scenario-to-cloud redeploy cloud-manifest.json --preflight
	  scenario-to-cloud redeploy cloud-manifest.json --if-needed --preflight --wait
	  scenario-to-cloud redeploy --domain vrooli.com --scenario landing-page-business-suite --if-needed --preflight --wait
	  scenario-to-cloud redeploy --domain vrooli.com --scenario landing-page-business-suite --force-run --preflight --wait
	  scenario-to-cloud redeploy cloud-manifest.json --preflight --wait
	  scenario-to-cloud redeploy cloud-manifest.json --name "Production Deploy"`)
	return nil
}

type waitStageDuration struct {
	Stage      string `json:"stage"`
	DurationMs int64  `json:"duration_ms"`
}

type waitSummary struct {
	DeploymentID    string                 `json:"deployment_id"`
	FinalStatus     string                 `json:"final_status"`
	FinalStage      string                 `json:"final_stage"`
	TotalDurationMs int64                  `json:"total_duration_ms"`
	StageDurations  []waitStageDuration    `json:"stage_durations,omitempty"`
	Error           string                 `json:"error,omitempty"`
	Deployment      *deployment.Deployment `json:"deployment,omitempty"`
}

func waitForCompletionQuiet(client *deployment.Client, deploymentID string) (waitSummary, error) {
	startedAt := time.Now()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	currentStage := ""
	stageStartedAt := time.Time{}
	stageInitialized := false
	stageDurations := make([]waitStageDuration, 0, 4)

	for {
		_, getResp, err := client.Get(deploymentID)
		if err != nil {
			return waitSummary{}, fmt.Errorf("failed to poll deployment status: %w", err)
		}
		d := getResp.Deployment
		now := time.Now()
		stage := stageFromDeployment(d)

		if !stageInitialized {
			stageInitialized = true
			currentStage = stage
			stageStartedAt = now
		} else if stage != currentStage {
			stageDurations = append(stageDurations, waitStageDuration{
				Stage:      currentStage,
				DurationMs: now.Sub(stageStartedAt).Milliseconds(),
			})
			currentStage = stage
			stageStartedAt = now
		}

		if isTerminalDeploymentStatus(d.Status) {
			if stageInitialized {
				stageDurations = append(stageDurations, waitStageDuration{
					Stage:      currentStage,
					DurationMs: now.Sub(stageStartedAt).Milliseconds(),
				})
			}
			summary := waitSummary{
				DeploymentID:    deploymentID,
				FinalStatus:     string(d.Status),
				FinalStage:      currentStage,
				TotalDurationMs: now.Sub(startedAt).Milliseconds(),
				StageDurations:  stageDurations,
				Deployment:      d,
			}
			if d.ErrorMessage != nil {
				summary.Error = strings.TrimSpace(*d.ErrorMessage)
			}
			if d.Status == deployment.StatusDeployed {
				return summary, nil
			}
			return summary, fmt.Errorf("deployment ended with status %s", d.Status)
		}

		<-ticker.C
	}
}

func stageFromDeployment(d *deployment.Deployment) string {
	if d != nil && d.ProgressStep != nil && strings.TrimSpace(*d.ProgressStep) != "" {
		return strings.TrimSpace(*d.ProgressStep)
	}
	if d == nil {
		return "unknown"
	}
	return string(d.Status)
}

func isTerminalDeploymentStatus(status deployment.DeploymentStatus) bool {
	switch status {
	case deployment.StatusDeployed, deployment.StatusFailed, deployment.StatusStopped, deployment.StatusSetupComplete:
		return true
	default:
		return false
	}
}

func envelopeWithCommon(base map[string]interface{}, extra map[string]interface{}) map[string]interface{} {
	out := map[string]interface{}{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

func printJSON(v interface{}) error {
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json output: %w", err)
	}
	cliutil.PrintJSON(out)
	return nil
}
