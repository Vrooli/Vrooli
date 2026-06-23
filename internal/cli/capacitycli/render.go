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
		_, _ = fmt.Fprintf(w, "%s\t%s/%s\t%s\t%s\tprio=%s\tprotected=%t\tgranted=%s\tobserved=%s\tpeak=%s\n",
			c.ClaimID, c.OwnerKind, c.OwnerID, c.ResourceKind, c.Status, c.PriorityTier, c.Protected,
			humanBytes(c.AmountBytes), humanBytes(c.ObservedBytes), humanBytes(c.ObservedPeakBytes))
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
	if len(resp.Refreshed) == 0 && len(resp.Expired) == 0 && len(resp.Adopted) == 0 && len(resp.IdleUnloadCandidates) == 0 {
		_, _ = fmt.Fprintln(w, "no resident claims refreshed, expired, or adopted")
		return nil
	}
	for _, c := range resp.Refreshed {
		_, _ = fmt.Fprintf(w, "refreshed\t%s\t%s\n", c.OwnerID, c.ClaimID)
	}
	for _, c := range resp.Adopted {
		_, _ = fmt.Fprintf(w, "adopted\t%s\t%s\n", c.OwnerID, c.ClaimID)
	}
	for _, c := range resp.Expired {
		_, _ = fmt.Fprintf(w, "expired\t%s\t%s\n", c.OwnerID, c.ClaimID)
	}
	for _, c := range resp.IdleUnloadCandidates {
		_, _ = fmt.Fprintf(w, "would-idle-unload (advisory)\t%s\t%s\t->%s\n", c.OwnerID, c.ClaimID, c.Status)
	}
	return nil
}

// RenderGC renders the terminal-claim GC result.
func RenderGC(w io.Writer, format cliout.Format, resp capacityapp.GCOutput) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, resp)
	}
	if resp.Pruned == 0 {
		_, _ = fmt.Fprintf(w, "no terminal claims past retention %s to prune\n", resp.RetentionSpent)
		return nil
	}
	_, _ = fmt.Fprintf(w, "pruned %d terminal claim(s) (%s) past retention %s\n", resp.Pruned, humanBytes(resp.Bytes), resp.RetentionSpent)
	return nil
}

// RenderRecommend renders advisory right-sizing suggestions.
func RenderRecommend(w io.Writer, format cliout.Format, resp capacityapp.RecommendOutput) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, resp)
	}
	if len(resp.Recommendations) == 0 {
		_, _ = fmt.Fprintln(w, "no right-sizing recommendations (claims are right-sized, or lack observed-peak data)")
		return nil
	}
	for _, r := range resp.Recommendations {
		_, _ = fmt.Fprintf(w, "%s\t%s\treserve=%s\tpeak=%s\tsuggest=%s\tsaves=%s\n",
			r.OwnerID, r.ClaimID, humanBytes(r.PreferredBytes), humanBytes(r.ObservedPeakBytes),
			humanBytes(r.SuggestedBytes), humanBytes(r.SavingsBytes))
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
