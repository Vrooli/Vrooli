package capacitycli

import (
	"fmt"
	"io"
	"strings"

	capacityapp "github.com/vrooli/vrooli/internal/app/capacity"
	"github.com/vrooli/vrooli/internal/cliout"
)

// RenderClaim renders a claim admission result.
func RenderClaim(w io.Writer, format cliout.Format, resp capacityapp.ClaimOutput) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, resp)
	}
	v := resp.Verdict
	_, _ = fmt.Fprintf(w, "claim %s\n", resp.Claim.ClaimID)
	_, _ = fmt.Fprintf(w, "  verdict: %s", v.Kind)
	if v.Step != "" {
		_, _ = fmt.Fprintf(w, " (step %s)", v.Step)
	}
	_, _ = fmt.Fprintf(w, "  granted: %s\n", humanBytes(v.GrantedBytes))
	_, _ = fmt.Fprintf(w, "  owner: %s/%s  priority: %s  enforce: %s\n", resp.Claim.OwnerKind, resp.Claim.OwnerID, resp.Claim.PriorityTier, resp.Enforce)
	if v.Reason != "" {
		_, _ = fmt.Fprintf(w, "  reason: %s\n", v.Reason)
	}
	for _, warn := range v.Warnings {
		_, _ = fmt.Fprintf(w, "  ⚠ %s\n", warn)
	}
	return nil
}

// RenderClaimView renders a single claim (heartbeat/activity/degrade/release).
func RenderClaimView(w io.Writer, format cliout.Format, resp capacityapp.ClaimView) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, resp)
	}
	_, _ = fmt.Fprintf(w, "%s\t%s/%s\t%s\tactivity=%s\tgen=%d\t%s\n",
		resp.ClaimID, resp.OwnerKind, resp.OwnerID, resp.Status, resp.ActivityState, resp.Generation, humanBytes(resp.AmountBytes))
	return nil
}

// RenderList renders the claim listing.
func RenderList(w io.Writer, format cliout.Format, resp capacityapp.ListOutput) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, resp)
	}
	if len(resp.Claims) == 0 {
		_, _ = fmt.Fprintln(w, "no capacity claims")
		return nil
	}
	for _, c := range resp.Claims {
		_, _ = fmt.Fprintf(w, "%s\t%s/%s\t%s\t%s\tprio=%s\tprotected=%t\t%s\n",
			c.ClaimID, c.OwnerKind, c.OwnerID, c.ResourceKind, c.Status, c.PriorityTier, c.Protected, humanBytes(c.AmountBytes))
	}
	return nil
}

// RenderReconcile renders reconciliation findings.
func RenderReconcile(w io.Writer, format cliout.Format, resp capacityapp.ReconcileOutput) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, resp)
	}
	if len(resp.Findings) == 0 {
		_, _ = fmt.Fprintln(w, "no GPU consumers above tracking threshold")
		return nil
	}
	for _, f := range resp.Findings {
		marker := " "
		if f.Severity == "warn" {
			marker = "⚠"
		}
		_, _ = fmt.Fprintf(w, "%s %s\t%s\t%s\tpid=%d\tobserved=%s\n",
			marker, strings.ToUpper(f.Class), f.OwnerID, f.Message, f.PID, humanBytes(f.ObservedBytes))
	}
	return nil
}

// RenderSweep renders the presence-driven sweep result.
func RenderSweep(w io.Writer, format cliout.Format, resp capacityapp.SweepOutput) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, resp)
	}
	if len(resp.Refreshed) == 0 && len(resp.Expired) == 0 {
		_, _ = fmt.Fprintln(w, "no resident claims refreshed or expired")
		return nil
	}
	for _, c := range resp.Refreshed {
		_, _ = fmt.Fprintf(w, "refreshed\t%s\t%s\n", c.OwnerID, c.ClaimID)
	}
	for _, c := range resp.Expired {
		_, _ = fmt.Fprintf(w, "expired\t%s\t%s\n", c.OwnerID, c.ClaimID)
	}
	return nil
}

// RenderPolicy renders policy levers.
func RenderPolicy(w io.Writer, format cliout.Format, resp capacityapp.PolicyOutput) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, resp)
	}
	for _, e := range resp.Entries {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", e.Key, e.Value)
	}
	return nil
}

func humanBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
