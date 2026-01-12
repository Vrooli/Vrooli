package vps

import (
	"context"
	"fmt"
	"strings"

	"scenario-to-cloud/ssh"
)

// TLSRenewResult captures the outcome of a Caddy TLS renewal attempt.
type TLSRenewResult struct {
	OK      bool
	Message string
	Output  string
}

// CaddyTLSRenewCommand builds the command used to renew/validate TLS via Caddy.
func CaddyTLSRenewCommand(domain string) string {
	domain = strings.TrimSpace(domain)
	return fmt.Sprintf(
		"caddy trust 2>/dev/null; "+
			"systemctl reload caddy && "+
			"sleep 3 && "+
			"curl -sf https://%s >/dev/null && echo 'Certificate valid'",
		domain,
	)
}

// RunCaddyTLSRenew executes a TLS renewal attempt over SSH.
func RunCaddyTLSRenew(ctx context.Context, sshRunner ssh.Runner, cfg ssh.Config, domain string) TLSRenewResult {
	cmd := CaddyTLSRenewCommand(domain)
	result, err := sshRunner.Run(ctx, cfg, cmd)
	if err != nil {
		return TLSRenewResult{
			OK:      false,
			Message: "Failed to renew TLS certificate",
			Output:  err.Error(),
		}
	}
	if result.ExitCode != 0 {
		output := result.Stderr
		if result.Stdout != "" {
			output = result.Stdout + "\n" + result.Stderr
		}
		return TLSRenewResult{
			OK:      false,
			Message: "Certificate renewal may have failed",
			Output:  output,
		}
	}
	return TLSRenewResult{
		OK:      true,
		Message: "TLS certificate renewed/validated successfully",
		Output:  result.Stdout,
	}
}
