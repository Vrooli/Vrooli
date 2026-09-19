package vps

import (
	"context"
	"strings"
	"time"

	"scenario-to-cloud/reach"
)

// TLSRenewResult captures the outcome of a Caddy TLS renewal attempt.
type TLSRenewResult struct {
	OK      bool
	Message string
	Output  string
}

// RunCaddyTLSRenew asks the edge proxy to reload its configuration through
// the privilege broker's edge.caddy.reload action (Caddy renews certificates
// itself on reload) and then verifies the domain from the cloud side. No
// shell runs on the target.
func RunCaddyTLSRenew(ctx context.Context, rt Runtime, deploymentID, domain string, verify func(ctx context.Context, domain string) error) TLSRenewResult {
	domain = strings.TrimSpace(domain)
	subject, err := EncodeJSONArg(map[string]any{"caddy": map[string]any{"deployment_id": deploymentID}})
	if err != nil {
		return TLSRenewResult{OK: false, Message: "Failed to encode the reload request", Output: err.Error()}
	}
	res, err := rt.Reach.Exec(ctx, rt.Target, reach.Command{Verb: "cloud-target host repair", Args: []string{"--action", "edge.caddy.reload", "--subject", subject, "--json"}, RequiredScope: "vrooli:write", Effectful: true, Timeout: 2 * time.Minute})
	if err != nil {
		return TLSRenewResult{OK: false, Message: "Failed to reload the edge proxy", Output: err.Error()}
	}
	if res.ExitCode != 0 {
		return TLSRenewResult{OK: false, Message: "Edge proxy reload was refused or failed", Output: coalesce(strings.TrimSpace(res.Stdout), strings.TrimSpace(res.Stderr))}
	}
	if verify != nil && domain != "" {
		if err := verify(ctx, domain); err != nil {
			return TLSRenewResult{OK: false, Message: "Edge proxy reloaded but the certificate did not verify", Output: err.Error()}
		}
	}
	return TLSRenewResult{OK: true, Message: "TLS certificate renewed/validated successfully", Output: strings.TrimSpace(res.Stdout)}
}
