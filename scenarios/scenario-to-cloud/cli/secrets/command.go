package secrets

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"scenario-to-cloud/cli/deployment"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Run(client *Client, deploymentClient *deployment.Client, args []string) error {
	if len(args) == 0 {
		return printUsage()
	}

	switch args[0] {
	case "set":
		return runSet(client, deploymentClient, args[1:])
	case "get":
		return runGet(client, deploymentClient, args[1:])
	case "verify":
		return runVerify(client, deploymentClient, args[1:])
	case "delete":
		return runDelete(client, deploymentClient, args[1:])
	case "plan-get":
		return runLegacyGet(client, args[1:])
	case "help", "-h", "--help":
		return printUsage()
	default:
		return fmt.Errorf("unknown subcommand: %s\n\nRun 'scenario-to-cloud secrets help' for usage", args[0])
	}
}

func printUsage() error {
	fmt.Println(`Usage: scenario-to-cloud secrets <command> [arguments]

Commands:
  set <KEY>            Set a secret in one or more targets (workspace/scenario/deployment)
  get <KEY>            Read a secret from one or more targets
  verify <KEY>         Verify secret value consistency across selected targets
  delete <KEY>         Delete a secret from one or more targets
  plan-get <scenario>  Legacy: get secret plan for a scenario

Common selector flags (for deployment target):
  --deployment-id <id>
  --domain <domain> --scenario <id>
  --host <host> --scenario <id>
  --target <domain-or-host> [--scenario <id>]
  --all-deployments --scenario <id>

Examples:
  scenario-to-cloud secrets set LPBS_SERVICE_SECRET --scenario landing-page-business-suite --generate hex:64 --targets scenario,deployment --domain <domain> --restart
  scenario-to-cloud secrets get LPBS_SERVICE_SECRET --scenario landing-page-business-suite --targets scenario,deployment --domain <domain>
  scenario-to-cloud secrets verify LPBS_SERVICE_SECRET --scenario landing-page-business-suite --targets scenario,deployment --domain <domain>
  scenario-to-cloud secrets delete LPBS_SERVICE_SECRET --scenario landing-page-business-suite --targets scenario,deployment --domain <domain> --restart`)
	return nil
}

type selectorFlags struct {
	deploymentID string
	scenarioID   string
	host         string
	domain       string
	target       string
	all          bool
}

func runSet(client *Client, deploymentClient *deployment.Client, args []string) error {
	fs := flag.NewFlagSet("secrets set", flag.ContinueOnError)
	value := fs.String("value", "", "Secret value")
	generate := fs.String("generate", "", "Generate value (hex:<n>, base64:<n>, alnum:<n>, uuid)")
	targetsRaw := fs.String("targets", "scenario", "Comma-separated targets: workspace,scenario,deployment")
	restart := fs.Bool("restart", false, "When targeting deployment, restart scenario after secret update")
	jsonOutput := fs.Bool("json", false, "Output raw JSON")

	sel := selectorFlags{}
	fs.StringVar(&sel.deploymentID, "deployment-id", "", "Deployment ID")
	fs.StringVar(&sel.scenarioID, "scenario", "", "Scenario ID (required for scenario target; recommended for deployment selector)")
	fs.StringVar(&sel.host, "host", "", "Deployment host selector")
	fs.StringVar(&sel.domain, "domain", "", "Deployment domain selector")
	fs.StringVar(&sel.target, "target", "", "Deployment convenience selector (domain or host)")
	fs.BoolVar(&sel.all, "all-deployments", false, "Apply to all deployments for --scenario")

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: scenario-to-cloud secrets set <KEY> [flags]")
	}
	key := strings.TrimSpace(fs.Arg(0))
	if strings.TrimSpace(*value) == "" && strings.TrimSpace(*generate) == "" {
		return fmt.Errorf("either --value or --generate is required")
	}
	if strings.TrimSpace(*value) != "" && strings.TrimSpace(*generate) != "" {
		return fmt.Errorf("--value and --generate are mutually exclusive")
	}

	targets, err := parseTargets(*targetsRaw)
	if err != nil {
		return err
	}

	deploymentIDs, err := resolveDeploymentTargets(deploymentClient, targets["deployment"], sel)
	if err != nil {
		return err
	}

	results := make([]map[string]interface{}, 0)
	resolvedValue := *value
	if strings.TrimSpace(*generate) != "" {
		generatedValue, genErr := generateSecretFromSpec(*generate)
		if genErr != nil {
			return genErr
		}
		resolvedValue = generatedValue
	}
	localReq := LocalSecretSetRequest{Value: resolvedValue}

	if targets["workspace"] {
		_, resp, err := client.SetLocal("workspace", key, "", localReq)
		if err != nil {
			return err
		}
		results = append(results, map[string]interface{}{
			"target": "workspace",
			"result": resp,
		})
	}

	if targets["scenario"] {
		if strings.TrimSpace(sel.scenarioID) == "" {
			return fmt.Errorf("--scenario is required when targets include scenario")
		}
		_, resp, err := client.SetLocal("scenario", key, sel.scenarioID, localReq)
		if err != nil {
			return err
		}
		results = append(results, map[string]interface{}{
			"target": "scenario",
			"result": resp,
		})
	}

	if targets["deployment"] {
		for _, deploymentID := range deploymentIDs {
			createReq := DeploymentSecretCreateRequest{
				Key:             key,
				Value:           resolvedValue,
				RestartScenario: *restart,
			}
			_, resp, createErr := client.CreateDeploymentSecret(deploymentID, createReq)
			if createErr != nil {
				if !looksLikeHTTPStatus(createErr, 409) {
					return createErr
				}
				_, resp, updateErr := client.UpdateDeploymentSecret(deploymentID, key, DeploymentSecretUpdateRequest{
					Value:           createReq.Value,
					RestartScenario: *restart,
				})
				if updateErr != nil {
					return updateErr
				}
				results = append(results, map[string]interface{}{
					"target":        "deployment",
					"deployment_id": deploymentID,
					"result":        resp,
				})
				continue
			}
			results = append(results, map[string]interface{}{
				"target":        "deployment",
				"deployment_id": deploymentID,
				"result":        resp,
			})
		}
	}

	if *jsonOutput {
		cliutil.PrintJSON(mustJSON(map[string]interface{}{
			"action":    "set",
			"key":       key,
			"targets":   keys(targets),
			"results":   results,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}))
		return nil
	}

	for _, entry := range results {
		target := fmt.Sprintf("%v", entry["target"])
		if depID, ok := entry["deployment_id"].(string); ok && depID != "" {
			fmt.Printf("[%s:%s] %s\n", target, depID, key)
			continue
		}
		fmt.Printf("[%s] %s\n", target, key)
	}
	return nil
}

func runGet(client *Client, deploymentClient *deployment.Client, args []string) error {
	fs := flag.NewFlagSet("secrets get", flag.ContinueOnError)
	targetsRaw := fs.String("targets", "scenario", "Comma-separated targets: workspace,scenario,deployment")
	reveal := fs.Bool("reveal", false, "Reveal secret value")
	jsonOutput := fs.Bool("json", false, "Output raw JSON")

	sel := selectorFlags{}
	fs.StringVar(&sel.deploymentID, "deployment-id", "", "Deployment ID")
	fs.StringVar(&sel.scenarioID, "scenario", "", "Scenario ID (required for scenario target; recommended for deployment selector)")
	fs.StringVar(&sel.host, "host", "", "Deployment host selector")
	fs.StringVar(&sel.domain, "domain", "", "Deployment domain selector")
	fs.StringVar(&sel.target, "target", "", "Deployment convenience selector (domain or host)")
	fs.BoolVar(&sel.all, "all-deployments", false, "Read from all deployments for --scenario")

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: scenario-to-cloud secrets get <KEY> [flags]")
	}
	key := strings.TrimSpace(fs.Arg(0))

	targets, err := parseTargets(*targetsRaw)
	if err != nil {
		return err
	}

	deploymentIDs, err := resolveDeploymentTargets(deploymentClient, targets["deployment"], sel)
	if err != nil {
		return err
	}

	results := make([]map[string]interface{}, 0)

	if targets["workspace"] {
		_, resp, err := client.GetLocal("workspace", key, "", *reveal)
		if err != nil {
			return err
		}
		results = append(results, map[string]interface{}{"target": "workspace", "result": resp})
	}
	if targets["scenario"] {
		if strings.TrimSpace(sel.scenarioID) == "" {
			return fmt.Errorf("--scenario is required when targets include scenario")
		}
		_, resp, err := client.GetLocal("scenario", key, sel.scenarioID, *reveal)
		if err != nil {
			return err
		}
		results = append(results, map[string]interface{}{"target": "scenario", "result": resp})
	}
	if targets["deployment"] {
		for _, deploymentID := range deploymentIDs {
			_, resp, err := client.GetDeploymentSecret(deploymentID, key, *reveal)
			if err != nil {
				return err
			}
			results = append(results, map[string]interface{}{
				"target":        "deployment",
				"deployment_id": deploymentID,
				"result":        resp,
			})
		}
	}

	if *jsonOutput {
		cliutil.PrintJSON(mustJSON(map[string]interface{}{
			"action":    "get",
			"key":       key,
			"results":   results,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}))
		return nil
	}

	for _, entry := range results {
		target := fmt.Sprintf("%v", entry["target"])
		if depID, ok := entry["deployment_id"].(string); ok && depID != "" {
			fmt.Printf("[%s:%s] %s\n", target, depID, key)
			continue
		}
		fmt.Printf("[%s] %s\n", target, key)
	}
	return nil
}

type verifyTargetResult struct {
	Target       string `json:"target"`
	DeploymentID string `json:"deployment_id,omitempty"`
	Present      bool   `json:"present"`
	Fingerprint  string `json:"fingerprint,omitempty"`
}

func runVerify(client *Client, deploymentClient *deployment.Client, args []string) error {
	fs := flag.NewFlagSet("secrets verify", flag.ContinueOnError)
	targetsRaw := fs.String("targets", "scenario,deployment", "Comma-separated targets: workspace,scenario,deployment")
	jsonOutput := fs.Bool("json", false, "Output raw JSON")

	sel := selectorFlags{}
	fs.StringVar(&sel.deploymentID, "deployment-id", "", "Deployment ID")
	fs.StringVar(&sel.scenarioID, "scenario", "", "Scenario ID (required for scenario target; recommended for deployment selector)")
	fs.StringVar(&sel.host, "host", "", "Deployment host selector")
	fs.StringVar(&sel.domain, "domain", "", "Deployment domain selector")
	fs.StringVar(&sel.target, "target", "", "Deployment convenience selector (domain or host)")
	fs.BoolVar(&sel.all, "all-deployments", false, "Read from all deployments for --scenario")

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: scenario-to-cloud secrets verify <KEY> [flags]")
	}
	key := strings.TrimSpace(fs.Arg(0))

	targets, err := parseTargets(*targetsRaw)
	if err != nil {
		return err
	}

	deploymentIDs, err := resolveDeploymentTargets(deploymentClient, targets["deployment"], sel)
	if err != nil {
		return err
	}

	results := make([]verifyTargetResult, 0, 2+len(deploymentIDs))
	fingerprints := make([]string, 0, 2+len(deploymentIDs))
	missing := make([]string, 0)

	if targets["workspace"] {
		_, resp, err := client.GetLocal("workspace", key, "", true)
		if err != nil {
			if !looksLikeHTTPStatus(err, 404) {
				return err
			}
		}
		result := verifyTargetResult{Target: "workspace", Present: strings.TrimSpace(resp.Value) != ""}
		if result.Present {
			result.Fingerprint = fingerprintSecret(resp.Value)
			fingerprints = append(fingerprints, result.Fingerprint)
		} else {
			missing = append(missing, "workspace")
		}
		results = append(results, result)
	}

	if targets["scenario"] {
		if strings.TrimSpace(sel.scenarioID) == "" {
			return fmt.Errorf("--scenario is required when targets include scenario")
		}
		_, resp, err := client.GetLocal("scenario", key, sel.scenarioID, true)
		if err != nil {
			if !looksLikeHTTPStatus(err, 404) {
				return err
			}
		}
		result := verifyTargetResult{Target: "scenario", Present: strings.TrimSpace(resp.Value) != ""}
		if result.Present {
			result.Fingerprint = fingerprintSecret(resp.Value)
			fingerprints = append(fingerprints, result.Fingerprint)
		} else {
			missing = append(missing, "scenario")
		}
		results = append(results, result)
	}

	if targets["deployment"] {
		for _, deploymentID := range deploymentIDs {
			_, resp, err := client.GetDeploymentSecret(deploymentID, key, true)
			if err != nil {
				if !looksLikeHTTPStatus(err, 404) {
					return err
				}
			}
			value := strings.TrimSpace(resp.Secret.Value)
			result := verifyTargetResult{
				Target:       "deployment",
				DeploymentID: deploymentID,
				Present:      value != "",
			}
			if result.Present {
				result.Fingerprint = fingerprintSecret(value)
				fingerprints = append(fingerprints, result.Fingerprint)
			} else {
				missing = append(missing, fmt.Sprintf("deployment:%s", deploymentID))
			}
			results = append(results, result)
		}
	}

	consistent := true
	if len(fingerprints) > 1 {
		base := fingerprints[0]
		for _, current := range fingerprints[1:] {
			if current != base {
				consistent = false
				break
			}
		}
	}
	verified := consistent && len(missing) == 0 && len(fingerprints) > 0

	domainArg := "<domain>"
	if strings.TrimSpace(sel.domain) != "" {
		domainArg = strings.TrimSpace(sel.domain)
	}
	nextSteps := []string{}
	if !verified {
		if strings.TrimSpace(sel.scenarioID) != "" {
			nextSteps = append(nextSteps,
				fmt.Sprintf("scenario-to-cloud secrets set %s --scenario %s --generate hex:64 --targets scenario,deployment --domain %s --restart", key, sel.scenarioID, domainArg),
				fmt.Sprintf("scenario-to-cloud secrets verify %s --scenario %s --targets scenario,deployment --domain %s", key, sel.scenarioID, domainArg),
			)
		} else {
			nextSteps = append(nextSteps, "Re-run with --scenario <id> and a deployment selector to sync/verify across targets.")
		}
	}

	payload := map[string]interface{}{
		"action":     "verify",
		"key":        key,
		"verified":   verified,
		"consistent": consistent,
		"missing":    missing,
		"results":    results,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"next_steps": nextSteps,
	}

	if *jsonOutput {
		cliutil.PrintJSON(mustJSON(payload))
		if !verified {
			return fmt.Errorf("secret verification failed for key %s", key)
		}
		return nil
	}

	status := "PASS"
	if !verified {
		status = "FAIL"
	}
	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Verification: %s", status),
			fmt.Sprintf("Key: %s", key),
		},
	}
	targetItems := make([]string, 0, len(results))
	for _, result := range results {
		label := result.Target
		if result.DeploymentID != "" {
			label = fmt.Sprintf("%s:%s", result.Target, result.DeploymentID)
		}
		state := "present"
		if !result.Present {
			state = "missing"
		}
		fp := ""
		if result.Fingerprint != "" {
			fp = " fingerprint=" + result.Fingerprint
		}
		targetItems = append(targetItems, fmt.Sprintf("%s: %s%s", label, state, fp))
	}
	if len(targetItems) > 0 {
		report.Triage = append(report.Triage, cliapp.TriageGroup{Heading: "Targets", Items: targetItems})
	}
	if len(missing) > 0 {
		report.Triage = append(report.Triage, cliapp.TriageGroup{
			Heading: "Missing targets",
			Items:   []string{strings.Join(missing, ", ")},
		})
	}
	if !consistent {
		report.Triage = append(report.Triage, cliapp.TriageGroup{
			Heading: "Consistency",
			Items:   []string{"Fingerprint mismatch detected across targets."},
		})
	}
	report.NextSteps = nextSteps
	if err := cliapp.RenderOperationalReport(os.Stdout, report); err != nil {
		return err
	}
	if !verified {
		return fmt.Errorf("secret verification failed for key %s", key)
	}
	return nil
}

func runDelete(client *Client, deploymentClient *deployment.Client, args []string) error {
	fs := flag.NewFlagSet("secrets delete", flag.ContinueOnError)
	targetsRaw := fs.String("targets", "scenario", "Comma-separated targets: workspace,scenario,deployment")
	restart := fs.Bool("restart", false, "When targeting deployment, restart scenario after delete")
	jsonOutput := fs.Bool("json", false, "Output raw JSON")

	sel := selectorFlags{}
	fs.StringVar(&sel.deploymentID, "deployment-id", "", "Deployment ID")
	fs.StringVar(&sel.scenarioID, "scenario", "", "Scenario ID (required for scenario target; recommended for deployment selector)")
	fs.StringVar(&sel.host, "host", "", "Deployment host selector")
	fs.StringVar(&sel.domain, "domain", "", "Deployment domain selector")
	fs.StringVar(&sel.target, "target", "", "Deployment convenience selector (domain or host)")
	fs.BoolVar(&sel.all, "all-deployments", false, "Delete from all deployments for --scenario")

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: scenario-to-cloud secrets delete <KEY> [flags]")
	}
	key := strings.TrimSpace(fs.Arg(0))

	targets, err := parseTargets(*targetsRaw)
	if err != nil {
		return err
	}

	deploymentIDs, err := resolveDeploymentTargets(deploymentClient, targets["deployment"], sel)
	if err != nil {
		return err
	}

	results := make([]map[string]interface{}, 0)

	if targets["workspace"] {
		_, resp, err := client.DeleteLocal("workspace", key, "")
		if err != nil {
			return err
		}
		results = append(results, map[string]interface{}{"target": "workspace", "result": resp})
	}
	if targets["scenario"] {
		if strings.TrimSpace(sel.scenarioID) == "" {
			return fmt.Errorf("--scenario is required when targets include scenario")
		}
		_, resp, err := client.DeleteLocal("scenario", key, sel.scenarioID)
		if err != nil {
			return err
		}
		results = append(results, map[string]interface{}{"target": "scenario", "result": resp})
	}
	if targets["deployment"] {
		for _, deploymentID := range deploymentIDs {
			_, resp, err := client.DeleteDeploymentSecret(deploymentID, key, *restart)
			if err != nil {
				return err
			}
			results = append(results, map[string]interface{}{
				"target":        "deployment",
				"deployment_id": deploymentID,
				"result":        resp,
			})
		}
	}

	if *jsonOutput {
		cliutil.PrintJSON(mustJSON(map[string]interface{}{
			"action":    "delete",
			"key":       key,
			"results":   results,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}))
		return nil
	}

	for _, entry := range results {
		target := fmt.Sprintf("%v", entry["target"])
		if depID, ok := entry["deployment_id"].(string); ok && depID != "" {
			fmt.Printf("[%s:%s] deleted %s\n", target, depID, key)
			continue
		}
		fmt.Printf("[%s] deleted %s\n", target, key)
	}
	return nil
}

func runLegacyGet(client *Client, args []string) error {
	var scenarioID string
	reveal := false
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud secrets plan-get <scenario-id> [flags]

Flags:
  --reveal  Show secret values (default: masked)
  --json    Output raw JSON`)
			return nil
		case "--reveal":
			reveal = true
		case "--json":
			jsonOutput = true
		default:
			if !strings.HasPrefix(args[i], "-") && scenarioID == "" {
				scenarioID = args[i]
			}
		}
	}
	if scenarioID == "" {
		return fmt.Errorf("usage: scenario-to-cloud secrets plan-get <scenario-id>")
	}

	body, _, err := client.LegacyGet(scenarioID, reveal)
	if err != nil {
		return err
	}
	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Printf("Secret plan loaded for scenario: %s\n", scenarioID)
	return nil
}

func parseTargets(raw string) (map[string]bool, error) {
	out := map[string]bool{
		"workspace":  false,
		"scenario":   false,
		"deployment": false,
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		out["scenario"] = true
		return out, nil
	}
	for _, token := range strings.Split(raw, ",") {
		t := strings.ToLower(strings.TrimSpace(token))
		if t == "" {
			continue
		}
		if _, ok := out[t]; !ok {
			return nil, fmt.Errorf("unknown target %q (valid: workspace,scenario,deployment)", t)
		}
		out[t] = true
	}
	if !out["workspace"] && !out["scenario"] && !out["deployment"] {
		return nil, fmt.Errorf("no valid targets provided")
	}
	return out, nil
}

func resolveDeploymentTargets(client *deployment.Client, needed bool, sel selectorFlags) ([]string, error) {
	if !needed {
		return nil, nil
	}
	if client == nil {
		return nil, fmt.Errorf("deployment client is required for deployment target operations")
	}
	if strings.TrimSpace(sel.deploymentID) != "" {
		return []string{strings.TrimSpace(sel.deploymentID)}, nil
	}

	if sel.all {
		if strings.TrimSpace(sel.scenarioID) == "" {
			return nil, fmt.Errorf("--all-deployments requires --scenario")
		}
		_, resp, err := client.List(deployment.ListOptions{ScenarioID: strings.TrimSpace(sel.scenarioID)})
		if err != nil {
			return nil, err
		}
		ids := make([]string, 0)
		for _, d := range resp.Deployments {
			if strings.TrimSpace(sel.domain) != "" && !strings.EqualFold(strings.TrimSpace(d.Domain), strings.TrimSpace(sel.domain)) {
				continue
			}
			if strings.TrimSpace(sel.host) != "" && !strings.EqualFold(strings.TrimSpace(d.Host), strings.TrimSpace(sel.host)) {
				continue
			}
			if strings.TrimSpace(sel.target) != "" {
				t := strings.TrimSpace(sel.target)
				if !strings.EqualFold(strings.TrimSpace(d.Domain), t) && !strings.EqualFold(strings.TrimSpace(d.Host), t) {
					continue
				}
			}
			ids = append(ids, d.ID)
		}
		if len(ids) == 0 {
			return nil, fmt.Errorf("no deployments matched --all-deployments selector")
		}
		return ids, nil
	}

	selector := deployment.ManifestSelector{
		ScenarioID: strings.TrimSpace(sel.scenarioID),
		Host:       strings.TrimSpace(sel.host),
		Domain:     strings.TrimSpace(sel.domain),
		Target:     strings.TrimSpace(sel.target),
	}
	resolved, err := deployment.ResolveLatestBySelector(client, selector)
	if err != nil {
		return nil, err
	}
	if resolved == nil {
		return nil, fmt.Errorf("no deployment resolved; provide --deployment-id or selector flags")
	}
	return []string{resolved.ID}, nil
}

func looksLikeHTTPStatus(err error, status int) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), fmt.Sprintf("(%d)", status)) ||
		strings.Contains(err.Error(), fmt.Sprintf(" %d ", status)) ||
		strings.Contains(err.Error(), fmt.Sprintf(":%d", status))
}

func keys(flags map[string]bool) []string {
	out := make([]string, 0, len(flags))
	for k, v := range flags {
		if v {
			out = append(out, k)
		}
	}
	return out
}

func fingerprintSecret(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])[:12]
}

func mustJSON(v interface{}) []byte {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return []byte(`{"error":"json_marshal_failed"}`)
	}
	return b
}

func generateSecretFromSpec(spec string) (string, error) {
	spec = strings.TrimSpace(spec)
	if strings.EqualFold(spec, "uuid") {
		return generateUUID(), nil
	}
	parts := strings.Split(spec, ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("unsupported generate spec %q (expected hex:<n>, base64:<n>, alnum:<n>, or uuid)", spec)
	}
	method := strings.ToLower(strings.TrimSpace(parts[0]))
	n, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || n <= 0 || n > 4096 {
		return "", fmt.Errorf("invalid generate length %q", parts[1])
	}
	switch method {
	case "hex":
		buf := make([]byte, (n+1)/2)
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("generate hex secret: %w", err)
		}
		out := hex.EncodeToString(buf)
		if len(out) > n {
			out = out[:n]
		}
		return out, nil
	case "base64":
		buf := make([]byte, n)
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("generate base64 secret: %w", err)
		}
		out := base64.RawStdEncoding.EncodeToString(buf)
		if len(out) > n {
			out = out[:n]
		}
		return out, nil
	case "alnum":
		const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
		buf := make([]byte, n)
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("generate alnum secret: %w", err)
		}
		out := make([]byte, n)
		for i := range buf {
			out[i] = alphabet[int(buf[i])%len(alphabet)]
		}
		return string(out), nil
	default:
		return "", fmt.Errorf("unsupported generate method %q", method)
	}
}

func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
