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
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteJSON(w, resp) }, func(w io.Writer) error {
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
	})
}

// RenderClaimView renders a single claim (heartbeat/activity/degrade/release).
func RenderClaimView(w io.Writer, format cliout.Format, resp capacityapp.ClaimView) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteJSON(w, resp) }, func(w io.Writer) error {
		return cliout.WriteSection(w, cliout.Section{Rows: [][]string{{
			resp.ClaimID, resp.OwnerKind + "/" + resp.OwnerID, resp.Status,
			"activity=" + resp.ActivityState, "idle=" + resp.IdleReclaimState,
			fmt.Sprintf("gen=%d", resp.Generation), humanBytes(resp.AmountBytes),
		}}})
	})
}

// RenderList renders the claim listing.
func RenderList(w io.Writer, format cliout.Format, resp capacityapp.ListOutput) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteJSON(w, resp) }, func(w io.Writer) error {
		rows := make([][]string, 0, len(resp.Claims))
		for _, c := range resp.Claims {
			rows = append(rows, []string{
				c.ClaimID, c.OwnerKind + "/" + c.OwnerID, c.ResourceKind, c.Status,
				"prio=" + c.PriorityTier, fmt.Sprintf("protected=%t", c.Protected), "idle=" + c.IdleReclaimState,
				"granted=" + humanBytes(c.AmountBytes), "observed=" + humanBytes(c.ObservedBytes), "peak=" + humanBytes(c.ObservedPeakBytes),
			})
		}
		return cliout.WriteSection(w, cliout.Section{Empty: cliout.EmptyLabel("capacity claims"), Rows: rows})
	})
}

// RenderReconcile renders reconciliation findings.
func RenderReconcile(w io.Writer, format cliout.Format, resp capacityapp.ReconcileOutput) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteJSON(w, resp) }, func(w io.Writer) error {
		rows := make([][]string, 0, len(resp.Findings))
		for _, f := range resp.Findings {
			marker := " "
			if f.Severity == "warn" {
				marker = "⚠"
			}
			rows = append(rows, []string{
				marker + " " + strings.ToUpper(f.Class), f.OwnerID, f.Message,
				fmt.Sprintf("pid=%d", f.PID), "observed=" + humanBytes(f.ObservedBytes),
			})
		}
		return cliout.WriteSection(w, cliout.Section{Empty: "no GPU consumers above tracking threshold", Rows: rows})
	})
}

// RenderSweep renders the presence-driven sweep result.
func RenderSweep(w io.Writer, format cliout.Format, resp capacityapp.SweepOutput) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteJSON(w, resp) }, func(w io.Writer) error {
		rows := make([][]string, 0, len(resp.Refreshed)+len(resp.Adopted)+len(resp.Expired)+len(resp.IdleUnloadCandidates))
		for _, c := range resp.Refreshed {
			rows = append(rows, []string{"refreshed", c.OwnerID, c.ClaimID})
		}
		for _, c := range resp.Adopted {
			rows = append(rows, []string{"adopted", c.OwnerID, c.ClaimID})
		}
		for _, c := range resp.Expired {
			rows = append(rows, []string{"expired", c.OwnerID, c.ClaimID})
		}
		for _, c := range resp.IdleUnloadCandidates {
			rows = append(rows, []string{"would-idle-unload (advisory)", c.OwnerID, c.ClaimID, "->" + c.Status})
		}
		return cliout.WriteSection(w, cliout.Section{Empty: "no resident claims refreshed, expired, or adopted", Rows: rows})
	})
}

// RenderGC renders the terminal-claim GC result.
func RenderGC(w io.Writer, format cliout.Format, resp capacityapp.GCOutput) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteJSON(w, resp) }, func(w io.Writer) error {
		if resp.Pruned == 0 {
			_, _ = fmt.Fprintf(w, "no terminal claims past retention %s to prune\n", resp.RetentionSpent)
			return nil
		}
		_, _ = fmt.Fprintf(w, "pruned %d terminal claim(s) (%s) past retention %s\n", resp.Pruned, humanBytes(resp.Bytes), resp.RetentionSpent)
		return nil
	})
}

// RenderRecommend renders advisory right-sizing suggestions.
func RenderRecommend(w io.Writer, format cliout.Format, resp capacityapp.RecommendOutput) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteJSON(w, resp) }, func(w io.Writer) error {
		rows := make([][]string, 0, len(resp.Recommendations))
		for _, r := range resp.Recommendations {
			rows = append(rows, []string{r.OwnerID, r.ClaimID, "reserve=" + humanBytes(r.PreferredBytes), "peak=" + humanBytes(r.ObservedPeakBytes), "suggest=" + humanBytes(r.SuggestedBytes), "saves=" + humanBytes(r.SavingsBytes)})
		}
		return cliout.WriteSection(w, cliout.Section{Empty: "no right-sizing recommendations (claims are right-sized, or lack observed-peak data)", Rows: rows})
	})
}

// RenderPolicy renders policy levers.
func RenderPolicy(w io.Writer, format cliout.Format, resp capacityapp.PolicyOutput) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteJSON(w, resp) }, func(w io.Writer) error {
		rows := make([][]string, 0, len(resp.Entries))
		for _, e := range resp.Entries {
			rows = append(rows, []string{e.Key, e.Value})
		}
		return cliout.WriteSection(w, cliout.Section{Rows: rows})
	})
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
