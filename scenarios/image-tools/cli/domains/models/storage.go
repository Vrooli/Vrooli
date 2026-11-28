package models

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"

	"github.com/vrooli/cli-core/cliapp"
)

// storage renders GetModelStorage: headroom, the adaptive idle threshold, the
// ranked offload candidates and recent receipts.
func (h *handlers) storage(ctx cliapp.RunContext) error {
	limit := int32(0)
	if raw := strings.TrimSpace(ctx.Flag("receipts")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			return fmt.Errorf("--receipts must be a non-negative integer")
		}
		limit = int32(n)
	}
	resp, err := h.client.GetModelStorage(context.Background(), connect.NewRequest(&modelsv1.GetModelStorageRequest{ReceiptLimit: limit}))
	if err != nil {
		return cliapp.WrapAPIError("read model storage", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Report == nil {
		return fmt.Errorf("server returned no model storage report")
	}
	results := formatStorageReport(resp.Msg.Report)
	for _, r := range resp.Msg.Receipts {
		results = append(results, formatReceipt(r))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        storageSummary(resp.Msg.Report, "Model storage"),
		ResultsHeading: "Candidates (offload order) and recent receipts",
		Results:        results,
		RetrievalHints: []string{
			"`models offload` — dry-run what a pass would remove now; add --apply to remove",
			"`models keep <id>` / `models unkeep <id>` — pin or unpin weights against offload",
		},
	})
}

// offload renders OffloadModels: a dry-run by default, --apply to remove.
func (h *handlers) offload(ctx cliapp.RunContext) error {
	req := &modelsv1.OffloadModelsRequest{Apply: ctx.BoolFlag("apply"), ModelIds: splitModelIDs(ctx.FlagValues("model"))}
	if raw := strings.TrimSpace(ctx.Flag("need-bytes")); raw != "" {
		n, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return fmt.Errorf("--need-bytes must be a byte count")
		}
		req.NeedBytes = n
	}
	resp, err := h.client.OffloadModels(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("offload models", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Report == nil {
		return fmt.Errorf("server returned no offload report")
	}
	report := resp.Msg.Report
	if !resp.Msg.Applied {
		results := []string{}
		for _, c := range report.Candidates {
			if c.Selected {
				results = append(results, formatCandidate(c))
			}
		}
		if len(results) == 0 {
			results = append(results, "nothing would be offloaded now")
		}
		return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
			Summary:        storageSummary(report, "Dry-run (nothing removed)"),
			ResultsHeading: "Would offload",
			Results:        results,
			RetrievalHints: []string{"`models offload --apply` — remove these weights and write receipts", "`models storage` — every candidate with its protections"},
		})
	}
	changes := []string{}
	removed := uint64(0)
	for _, r := range resp.Msg.Receipts {
		changes = append(changes, formatReceipt(r))
		if r.Outcome == "removed" {
			removed += r.Bytes
		}
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Offloaded %s across %d attempt(s).", humanBytes(int64(removed)), len(resp.Msg.Receipts))},
		Changes:     changes,
		NextCommand: []string{"`models storage` — confirm headroom and receipts"},
	})
}

// keep pins (or with --unkeep unpins) a model against offload.
func (h *handlers) keep(ctx cliapp.RunContext) error {
	return h.setKeep(ctx, !ctx.BoolFlag("unkeep"), ctx.Flag("reason"))
}

// unkeep is the explicit spelling of `keep --unkeep`.
func (h *handlers) unkeep(ctx cliapp.RunContext) error { return h.setKeep(ctx, false, "") }

func (h *handlers) setKeep(ctx cliapp.RunContext, keep bool, reason string) error {
	id := ctx.Positional("id")
	resp, err := h.client.SetModelKeep(context.Background(), connect.NewRequest(&modelsv1.SetModelKeepRequest{Id: id, Keep: keep, Reason: reason}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("set keep pin for %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no keep response")
	}
	result := fmt.Sprintf("Kept %s: offload will never remove its weights.", id)
	if !keep {
		result = fmt.Sprintf("Unkept %s: the adaptive policy may offload it when idle.", id)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{result},
		NextCommand: []string{"`models storage` — see the candidate's protections"},
	})
}

func splitModelIDs(values []string) []string {
	var out []string
	for _, v := range values {
		for _, id := range strings.Split(v, ",") {
			if id = strings.TrimSpace(id); id != "" {
				out = append(out, id)
			}
		}
	}
	return out
}

func storageSummary(r *modelsv1.ModelStorageReport, title string) []string {
	p := r.GetPolicy()
	threshold := "none (headroom is ample; nothing idles out)"
	if r.GetIdleThresholdSeconds() >= 0 {
		threshold = humanDuration(r.GetIdleThresholdSeconds())
	}
	lines := []string{
		fmt.Sprintf("%s: mode %s, %s free, %s headroom above the %s floor (reserve %s, ample %s).",
			title, r.GetMode(), humanBytes(int64(r.GetFreeBytes())), humanBytes(r.GetHeadroomBytes()), humanBytes(int64(p.GetFloorBytes())),
			humanBytes(int64(p.GetReserveBytes())), humanBytes(int64(p.GetAmpleBytes()))),
		fmt.Sprintf("Idle threshold: %s. Installed weights: %s across %d model(s).", threshold, humanBytes(int64(r.GetInstalledBytes())), len(r.GetCandidates())),
		fmt.Sprintf("Selected: %s; projected headroom %s (target %s).", humanBytes(int64(r.GetSelectedBytes())), humanBytes(r.GetProjectedHeadroomBytes()), humanBytes(r.GetTargetHeadroomBytes())),
		fmt.Sprintf("Auto-acquire %s; auto-offload %s (every %s).", onOff(p.GetAutoAcquire()), onOff(p.GetAutoOffload()), humanDuration(p.GetIntervalSeconds())),
		fmt.Sprintf("Staging: %s held by acquisitions; abandoned staging %s across %d dir(s), reclaimed after %s without writes.",
			humanBytes(int64(r.GetStagingBytes())), humanBytes(int64(r.GetAbandonedStagingBytes())), r.GetAbandonedStagingCount(), humanDuration(p.GetStagingGraceSeconds())),
	}
	if r.GetShortfallBytes() > 0 {
		lines = append(lines, fmt.Sprintf("Shortfall: %s cannot be freed without removing protected weights.", humanBytes(int64(r.GetShortfallBytes()))))
	}
	return lines
}

func formatStorageReport(r *modelsv1.ModelStorageReport) []string {
	out := make([]string, 0, len(r.GetCandidates()))
	for _, c := range r.GetCandidates() {
		out = append(out, formatCandidate(c))
	}
	return out
}

func formatCandidate(c *modelsv1.ModelStorageCandidate) string {
	state := "eligible"
	if len(c.GetProtected()) > 0 {
		state = "protected: " + strings.Join(c.GetProtected(), ",")
	}
	if c.GetSelected() {
		state = "SELECTED (" + c.GetSelectionReason() + ")"
	}
	return fmt.Sprintf("%-28s %9s  idle %-9s (%s)  reacquire ~%s  priority %.3g  %s",
		c.GetModelId(), humanBytes(int64(c.GetBytes())), humanDuration(c.GetIdleSeconds()), c.GetLastUseBasis(),
		humanDuration(int64(c.GetReacquireSeconds())), c.GetPriority(), state)
}

func formatReceipt(r *modelsv1.ModelOffloadReceipt) string {
	line := fmt.Sprintf("receipt %s %s %s %s (%s, trigger %s, idle %s)", r.GetCreatedAt(), r.GetModelId(), r.GetOutcome(), humanBytes(int64(r.GetBytes())), r.GetReason(), r.GetTrigger(), humanDuration(r.GetIdleSeconds()))
	if r.GetError() != "" {
		line += ": " + r.GetError()
	}
	return line
}

func humanBytes(n int64) string {
	sign := ""
	if n < 0 {
		sign, n = "-", -n
	}
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%s%.1f GiB", sign, float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%s%.0f MiB", sign, float64(n)/(1<<20))
	default:
		return fmt.Sprintf("%s%d B", sign, n)
	}
}

func humanDuration(seconds int64) string {
	d := time.Duration(seconds) * time.Second
	switch {
	case d >= 48*time.Hour:
		return fmt.Sprintf("%.1fd", d.Hours()/24)
	case d >= time.Hour:
		return fmt.Sprintf("%.1fh", d.Hours())
	default:
		return d.Round(time.Second).String()
	}
}

func onOff(v bool) string {
	if v {
		return "on"
	}
	return "off"
}
