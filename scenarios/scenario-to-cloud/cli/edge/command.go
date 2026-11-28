package edge

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"

	"scenario-to-cloud/cli/internal/apierr"
	"scenario-to-cloud/cli/internal/protoout"
	"scenario-to-cloud/cli/internal/selector"
)

// Run executes edge subcommands.
func Run(client *Client, args []string) error {
	if len(args) == 0 {
		return printUsage()
	}

	switch args[0] {
	case "status":
		return runStatus(client, args[1:])
	case "dns-check":
		return runDNSCheck(client, args[1:])
	case "dns-records":
		return runDNSRecords(client, args[1:])
	case "caddy":
		return runCaddy(client, args[1:])
	case "tls":
		return runTLS(client, args[1:])
	case "tls-renew":
		return runTLSRenew(client, args[1:])
	case "help", "-h", "--help":
		return printUsage()
	default:
		return fmt.Errorf("unknown subcommand: %s\n\nRun 'scenario-to-cloud edge help' for usage", args[0])
	}
}

func printUsage() error {
	fmt.Println(`Usage: scenario-to-cloud edge <command> [arguments]

Commands:
  status <selector>                 Typed edge observation: routes, private listeners, DNS, TLS, readiness
                                    ` + selector.Usage + `
  dns-check <deployment-id>         Check DNS configuration
  dns-records <deployment-id>       List DNS records
  caddy <deployment-id> <action>    Caddy control (reload, restart, status, validate)
  tls <deployment-id>               Show TLS certificate information
  tls-renew <deployment-id>         Renew TLS certificates

Run 'scenario-to-cloud edge <command> -h' for command-specific options.`)
	return nil
}

func runDNSCheck(client *Client, args []string) error {
	var deploymentID string
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud edge dns-check <deployment-id> [flags]

Flags:
  --json    Output raw JSON`)
			return nil
		case "--json":
			jsonOutput = true
		default:
			if !strings.HasPrefix(args[i], "-") && deploymentID == "" {
				deploymentID = args[i]
			}
		}
	}

	if deploymentID == "" {
		return fmt.Errorf("usage: scenario-to-cloud edge dns-check <deployment-id>")
	}

	body, resp, err := client.DNSCheck(deploymentID)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	// Pretty print
	fmt.Printf("DNS Check for: %s\n", resp.Domain)
	fmt.Println(strings.Repeat("-", 60))

	healthStatus := "UNHEALTHY"
	if resp.Healthy {
		healthStatus = "HEALTHY"
	}
	fmt.Printf("Status: %s\n\n", healthStatus)

	if len(resp.Records) > 0 {
		fmt.Printf("%-8s %-25s %-8s %s\n", "TYPE", "NAME", "STATUS", "VALUE")
		for _, r := range resp.Records {
			status := "FAIL"
			if r.OK {
				status = "OK"
			}
			value := r.Actual
			if !r.OK && r.Expected != "" {
				value = fmt.Sprintf("%s (expected: %s)", r.Actual, r.Expected)
			}
			fmt.Printf("%-8s %-25s %-8s %s\n", r.Type, truncate(r.Name, 25), status, value)
		}
	}

	if len(resp.Issues) > 0 {
		fmt.Println("\nIssues:")
		for _, issue := range resp.Issues {
			fmt.Printf("  - %s\n", issue)
		}
	}

	return nil
}

func runDNSRecords(client *Client, args []string) error {
	var deploymentID string
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud edge dns-records <deployment-id> [flags]

Flags:
  --json    Output raw JSON`)
			return nil
		case "--json":
			jsonOutput = true
		default:
			if !strings.HasPrefix(args[i], "-") && deploymentID == "" {
				deploymentID = args[i]
			}
		}
	}

	if deploymentID == "" {
		return fmt.Errorf("usage: scenario-to-cloud edge dns-records <deployment-id>")
	}

	body, resp, err := client.DNSRecords(deploymentID)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	// Pretty print
	fmt.Printf("DNS Records for: %s\n", resp.Domain)
	fmt.Println(strings.Repeat("-", 80))

	if len(resp.Records) == 0 {
		fmt.Println("No DNS records found.")
		return nil
	}

	fmt.Printf("%-8s %-30s %-8s %s\n", "TYPE", "NAME", "TTL", "VALUE")
	for _, r := range resp.Records {
		value := r.Value
		if r.Priority > 0 {
			value = fmt.Sprintf("%d %s", r.Priority, r.Value)
		}
		fmt.Printf("%-8s %-30s %-8d %s\n", r.Type, truncate(r.Name, 30), r.TTL, value)
	}

	return nil
}

func runCaddy(client *Client, args []string) error {
	var deploymentID, action string
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud edge caddy <deployment-id> <action> [flags]

Actions:
  reload     Reload Caddy configuration
  restart    Restart Caddy server
  status     Show Caddy status
  validate   Validate Caddy configuration

Flags:
  --json     Output raw JSON`)
			return nil
		case "--json":
			jsonOutput = true
		default:
			if !strings.HasPrefix(args[i], "-") {
				if deploymentID == "" {
					deploymentID = args[i]
				} else if action == "" {
					action = args[i]
				}
			}
		}
	}

	if deploymentID == "" || action == "" {
		return fmt.Errorf("usage: scenario-to-cloud edge caddy <deployment-id> <action>")
	}

	validActions := map[string]bool{"reload": true, "restart": true, "status": true, "validate": true}
	if !validActions[action] {
		return fmt.Errorf("action must be 'reload', 'restart', 'status', or 'validate', got: %s", action)
	}

	req := CaddyRequest{Action: action}
	body, resp, err := client.Caddy(deploymentID, req)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if resp.Success {
		fmt.Printf("Caddy '%s': SUCCESS\n", resp.Action)
	} else {
		fmt.Printf("Caddy '%s': FAILED\n", resp.Action)
	}

	if resp.Message != "" {
		fmt.Printf("Message: %s\n", resp.Message)
	}

	if resp.Config != "" {
		fmt.Println("\nConfiguration:")
		fmt.Println(resp.Config)
	}

	return nil
}

func runTLS(client *Client, args []string) error {
	var deploymentID string
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud edge tls <deployment-id> [flags]

Flags:
  --json    Output raw JSON`)
			return nil
		case "--json":
			jsonOutput = true
		default:
			if !strings.HasPrefix(args[i], "-") && deploymentID == "" {
				deploymentID = args[i]
			}
		}
	}

	if deploymentID == "" {
		return fmt.Errorf("usage: scenario-to-cloud edge tls <deployment-id>")
	}

	body, resp, err := client.TLSInfo(deploymentID)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	// Pretty print
	fmt.Printf("TLS Certificates for Deployment: %s\n", resp.DeploymentID)
	fmt.Println(strings.Repeat("-", 80))

	if len(resp.Certificates) == 0 {
		fmt.Println("No TLS certificates found.")
		return nil
	}

	for i, cert := range resp.Certificates {
		if i > 0 {
			fmt.Println()
		}
		status := "VALID"
		if cert.Expired {
			status = "EXPIRED"
		} else if cert.DaysLeft < 30 {
			status = "EXPIRING SOON"
		}

		fmt.Printf("Domain: %s\n", cert.Domain)
		fmt.Printf("  Status:     %s (%d days left)\n", status, cert.DaysLeft)
		fmt.Printf("  Issuer:     %s\n", cert.Issuer)
		fmt.Printf("  Valid:      %s to %s\n", cert.ValidFrom, cert.ValidUntil)
		fmt.Printf("  Auto-Renew: %v\n", cert.AutoRenew)
		if len(cert.SANs) > 0 {
			fmt.Printf("  SANs:       %s\n", strings.Join(cert.SANs, ", "))
		}
	}

	return nil
}

func runTLSRenew(client *Client, args []string) error {
	var deploymentID, domain string
	force := false
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud edge tls-renew <deployment-id> [flags]

Flags:
  --domain <name>   Renew specific domain (default: all)
  --force           Force renewal even if not expiring
  --json            Output raw JSON`)
			return nil
		case "--domain":
			if i+1 < len(args) {
				i++
				domain = args[i]
			}
		case "--force":
			force = true
		case "--json":
			jsonOutput = true
		default:
			if !strings.HasPrefix(args[i], "-") && deploymentID == "" {
				deploymentID = args[i]
			}
		}
	}

	if deploymentID == "" {
		return fmt.Errorf("usage: scenario-to-cloud edge tls-renew <deployment-id>")
	}

	body, resp, err := client.TLSRenew(deploymentID, domain, force)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if resp.Success {
		fmt.Println("TLS Renewal: SUCCESS")
	} else {
		fmt.Println("TLS Renewal: FAILED")
	}

	if resp.Message != "" {
		fmt.Printf("Message: %s\n", resp.Message)
	}

	if len(resp.Results) > 0 {
		fmt.Println("\nResults:")
		for _, r := range resp.Results {
			status := "OK"
			if !r.Success {
				status = "FAILED"
			}
			msg := ""
			if r.Message != "" {
				msg = fmt.Sprintf(" - %s", r.Message)
			}
			fmt.Printf("  [%s] %s%s\n", status, r.Domain, msg)
		}
	}

	return nil
}

// truncate shortens a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// runStatus prints the typed edge observation. Readiness is the producer's
// verdict: anything but an externally passed readiness exits 1.
func runStatus(client *Client, args []string) error {
	fs := flag.NewFlagSet("edge status", flag.ContinueOnError)
	sel := selector.Register(fs)
	jsonOutput := fs.Bool("json", false, "Output proto JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	chosen, err := sel.Selector(fs.Args())
	if err != nil {
		return err
	}
	ref, err := selector.Resolve(context.Background(), client.Deployments, chosen)
	if err != nil {
		return err
	}
	resp, err := client.Observation(context.Background(), ref.GetId())
	if err != nil {
		return err
	}
	if *jsonOutput {
		return protoout.Print(resp)
	}
	obs := resp.GetObservation()
	fmt.Println(selector.Identity(ref))
	fmt.Printf("domain: %s  acme: %s  spec digest: %s  observed: %s\n", obs.GetDomain(), obs.GetAcmeEnvironment(), obs.GetSpecDigest(), obs.GetObservedAt().AsTime().UTC().Format("2006-01-02T15:04:05Z"))
	for _, r := range obs.GetRoutes() {
		fmt.Printf("  route %s -> %d (%s)\n", r.GetHost(), r.GetUpstreamPort(), r.GetListenerId())
	}
	for _, l := range obs.GetPrivateListeners() {
		fmt.Printf("  private %s %s/%s:%d (%s)\n", l.GetId(), l.GetOwner(), l.GetPortName(), l.GetPort(), l.GetReason())
	}
	for _, d := range obs.GetDns() {
		fmt.Printf("  dns %s ipv4=%s ipv6=%s match=%t %s\n", d.GetHost(), d.GetIpv4().GetState(), d.GetIpv6().GetState(), d.GetMatch(), d.GetReasonCode())
	}
	for _, t := range obs.GetTls() {
		fmt.Printf("  tls %s issuer=%s not_after=%s days_left=%d renewal=%s %s\n", t.GetHost(), t.GetIssuer(), t.GetNotAfter(), t.GetDaysLeft(), t.GetRenewalState(), t.GetDetail())
	}
	ready := obs.GetReadiness()
	fmt.Printf("readiness: local=%s external=%s", ready.GetLocal().GetStatus(), ready.GetExternal().GetStatus())
	if ready.GetExternal().GetReasonCode() != "" {
		fmt.Printf(" (%s)", ready.GetExternal().GetReasonCode())
	}
	fmt.Println()
	for _, na := range obs.GetNextActions() {
		fmt.Printf("  next: %s", na.GetLabel())
		if na.GetReference() != "" {
			fmt.Printf(" (%s)", na.GetReference())
		}
		fmt.Println()
	}
	if ready.GetExternal().GetStatus() != "passed" {
		return apierr.Failed("edge for deployment %s is not externally ready (%s)", ref.GetId(), strings.TrimSpace(ready.GetExternal().GetStatus()+" "+ready.GetExternal().GetReasonCode()))
	}
	return nil
}
