package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
)

func (h *handlers) replay(ctx cliapp.RunContext) error {
	options, err := replayOptionsFromContext(ctx)
	if err != nil {
		return err
	}
	cases, err := readReplayInput(options.path, options.profile, options.mode)
	if err != nil {
		return err
	}
	verifier, err := newAgentManagerSourceVerifier(h.sourceHTTP, options.managerURL)
	if err != nil {
		return err
	}
	replayCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	receipts, err := h.dispatchReplay(replayCtx, cases, options, verifier)
	if err != nil {
		return err
	}
	if err := writeReplayReceipt(ctx.Flag("receipt-file"), receipts); err != nil {
		return err
	}
	return renderReplay(ctx, receipts, options.mode, options.parallel)
}

type replayOptions struct {
	path, mode, profile, managerURL string
	parallel                        int
}

func replayOptionsFromContext(ctx cliapp.RunContext) (replayOptions, error) {
	opts := replayOptions{path: ctx.Flag("input-file"), mode: ctx.Flag("mode"), profile: ctx.Flag("profile"), managerURL: ctx.Flag("agent-manager-url"), parallel: 4}
	if opts.path == "" {
		return replayOptions{}, fmt.Errorf("--input-file is required")
	}
	if raw := ctx.Flag("parallel"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return replayOptions{}, fmt.Errorf("--parallel must be an integer: %w", err)
		}
		opts.parallel = parsed
	}
	if opts.mode == "" {
		opts.mode = "preview"
	}
	if opts.mode != "preview" && opts.mode != "execute" {
		return replayOptions{}, fmt.Errorf("--mode must be preview or execute")
	}
	if opts.managerURL == "" {
		opts.managerURL = os.Getenv("AGENT_MANAGER_API_URL")
	}
	return opts, nil
}

func readReplayInput(path, profile, mode string) ([]ReplayCase, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open replay input: %w", err)
	}
	defer file.Close()
	cases, err := ReadReplayCases(file)
	if err != nil {
		return nil, err
	}
	if err := overrideReplayProfile(cases, profile, mode); err != nil {
		return nil, err
	}
	return cases, nil
}

func (h *handlers) dispatchReplay(ctx context.Context, cases []ReplayCase, options replayOptions, verifier *agentManagerSourceVerifier) ([]ReplayReceipt, error) {
	if options.mode == "preview" {
		return PreviewCallerCases(ctx, cases, options.parallel, verifier.verify, func(callCtx context.Context, request *routingv1.PreviewRouteRequest) (*routingv1.PreviewRouteResponse, error) {
			return previewReplayCall(callCtx, h.client, request)
		})
	}
	return ReplayCallerCases(ctx, cases, options.parallel, verifier.verify, func(callCtx context.Context, request *routingv1.ExecuteRouteRequest) (*routingv1.ExecuteRouteResponse, error) {
		return executeReplayCall(callCtx, h.client, request)
	})
}

func renderReplay(ctx cliapp.RunContext, receipts []ReplayReceipt, mode string, parallel int) error {
	passed, failed, p50, p95 := replayStats(receipts)
	rows := make([]string, 0, len(receipts))
	for _, receipt := range receipts {
		if mode == "preview" {
			rows = append(rows, fmt.Sprintf("%s profile=%s role=%s status=%s provider=%s %s route_plan=%s route_preview_ms=%d source_run=%s source_event=%s", receipt.CaseID, receipt.Profile, receipt.Role, receipt.Status, receipt.SelectedProvider, replayEligibilitySummary(receipt.CandidateEligibility), receipt.RoutePlanID, receipt.LatencyMS, receipt.SourceRunID, receipt.SourceEventID))
		} else {
			rows = append(rows, fmt.Sprintf("%s profile=%s role=%s status=%s model=%s locality=%s gateway_execution_ms=%d source_run=%s source_event=%s", receipt.CaseID, receipt.Profile, receipt.Role, receipt.Status, receipt.SelectedModel, receipt.Locality, receipt.LatencyMS, receipt.SourceRunID, receipt.SourceEventID))
		}
	}
	summary := fmt.Sprintf("Replayed %d authenticated caller cases in %s mode, parallel=%d.", len(receipts), mode, parallel)
	if mode == "execute" {
		summary = fmt.Sprintf("Executed %d authenticated caller cases: succeeded=%d failed=%d p50_ms=%d p95_ms=%d parallel=%d.", len(receipts), passed, failed, p50, p95, parallel)
	}
	return ctx.RenderList(cliapp.ListReport{
		Summary:        []string{summary},
		ResultsHeading: "Caller replay receipts",
		Results:        rows,
		RetrievalHints: []string{"`ai-gateway routing evidence-list` — inspect recent gateway route evidence"},
	})
}

func replayEligibilitySummary(candidates []ReplayCandidateEligibility) string {
	local, remote, rejected := 0, 0, 0
	for _, candidate := range candidates {
		if candidate.RejectionCode != "" {
			rejected++
			continue
		}
		switch candidate.Locality {
		case "local":
			local++
		case "remote":
			remote++
		}
	}
	return fmt.Sprintf("eligible_local=%d eligible_remote=%d rejected=%d", local, remote, rejected)
}

func replayStats(receipts []ReplayReceipt) (passed, failed int, p50, p95 int64) {
	latencies := make([]int64, 0, len(receipts))
	for _, receipt := range receipts {
		if receipt.Status == "succeeded" {
			passed++
		} else {
			failed++
		}
		latencies = append(latencies, receipt.LatencyMS)
	}
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	if len(latencies) > 0 {
		p50 = latencies[(len(latencies)-1)*50/100]
		p95 = latencies[(len(latencies)-1)*95/100]
	}
	return passed, failed, p50, p95
}

func writeReplayReceipt(path string, receipts []ReplayReceipt) error {
	if path == "" {
		return nil
	}
	data, err := json.MarshalIndent(struct {
		SchemaVersion string          `json:"schema_version"`
		Receipts      []ReplayReceipt `json:"receipts"`
	}{SchemaVersion: "ai-gateway-replay/v1", Receipts: receipts}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write replay receipt: %w", err)
	}
	return nil
}
