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
	path := ctx.Flag("input-file")
	if path == "" {
		return fmt.Errorf("--input-file is required")
	}
	parallel := 4
	if raw := ctx.Flag("parallel"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return fmt.Errorf("--parallel must be an integer: %w", err)
		}
		parallel = parsed
	}
	mode := ctx.Flag("mode")
	if mode == "" {
		mode = "preview"
	}
	if mode != "preview" && mode != "execute" {
		return fmt.Errorf("--mode must be preview or execute")
	}
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open replay input: %w", err)
	}
	defer file.Close()
	cases, err := ReadReplayCases(file)
	if err != nil {
		return err
	}
	managerURL := ctx.Flag("agent-manager-url")
	if managerURL == "" {
		managerURL = os.Getenv("AGENT_MANAGER_API_URL")
	}
	verifier, err := newAgentManagerSourceVerifier(h.sourceHTTP, managerURL)
	if err != nil {
		return err
	}
	replayCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	var receipts []ReplayReceipt
	if mode == "preview" {
		receipts, err = PreviewCallerCases(replayCtx, cases, parallel, verifier.verify, func(callCtx context.Context, request *routingv1.PreviewRouteRequest) (*routingv1.PreviewRouteResponse, error) {
			return previewReplayCall(callCtx, h.client, request)
		})
	} else {
		receipts, err = ReplayCallerCases(replayCtx, cases, parallel, verifier.verify, func(callCtx context.Context, request *routingv1.ExecuteRouteRequest) (*routingv1.ExecuteRouteResponse, error) {
			return executeReplayCall(callCtx, h.client, request)
		})
	}
	if err != nil {
		return err
	}
	if err := writeReplayReceipt(ctx.Flag("receipt-file"), receipts); err != nil {
		return err
	}
	passed, failed, p50, p95 := replayStats(receipts)
	rows := make([]string, 0, len(receipts))
	for _, receipt := range receipts {
		if mode == "preview" {
			rows = append(rows, fmt.Sprintf("%s role=%s status=%s provider=%s route_plan=%s route_preview_ms=%d source_run=%s source_event=%s", receipt.CaseID, receipt.Role, receipt.Status, receipt.SelectedProvider, receipt.RoutePlanID, receipt.LatencyMS, receipt.SourceRunID, receipt.SourceEventID))
		} else {
			rows = append(rows, fmt.Sprintf("%s role=%s status=%s model=%s locality=%s gateway_execution_ms=%d source_run=%s source_event=%s", receipt.CaseID, receipt.Role, receipt.Status, receipt.SelectedModel, receipt.Locality, receipt.LatencyMS, receipt.SourceRunID, receipt.SourceEventID))
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
