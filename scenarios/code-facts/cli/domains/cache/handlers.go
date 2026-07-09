package cache

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	factsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts/facts_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	client factsconnect.CodeFactsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: factsconnect.NewCodeFactsServiceClient(httpClient, baseURL)}
}

func (h *handlers) status(ctx cliapp.RunContext) error {
	target := parseTarget(ctx.Positional("target"))
	resp, err := h.client.GetCacheStatus(context.Background(), connect.NewRequest(&factsv1.GetCacheStatusRequest{
		Target: target,
	}))
	if err != nil {
		return cliapp.WrapAPIError("get cache status", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no cache status")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Cache %s has %d entrie(s).", resp.Msg.GetCacheKey(), resp.Msg.GetEntries()),
			fmt.Sprintf("Stored %d bytes across %d row(s); budget %d bytes; utilization %.2f%%.",
				resp.Msg.GetTotalPayloadBytes(), resp.Msg.GetTotalRows(), resp.Msg.GetBudgetBytes(), resp.Msg.GetUtilization()*100),
		},
		ResultsHeading: "Entries",
		Results:        cacheEntryLines(resp.Msg.GetEntriesMetadata()),
	})
}

func (h *handlers) inspect(ctx cliapp.RunContext) error {
	target := parseTarget(ctx.Positional("target"))
	resp, err := h.client.InspectCache(context.Background(), connect.NewRequest(&factsv1.InspectCacheRequest{
		Target:   target,
		CacheKey: ctx.Flag("cache-key"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("inspect cache", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no cache status")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Cache %s has %d entrie(s).", resp.Msg.GetCacheKey(), resp.Msg.GetEntries()),
			fmt.Sprintf("Stored %d bytes across %d row(s); budget %d bytes; utilization %.2f%%.",
				resp.Msg.GetTotalPayloadBytes(), resp.Msg.GetTotalRows(), resp.Msg.GetBudgetBytes(), resp.Msg.GetUtilization()*100),
		},
		ResultsHeading: "Entries",
		Results:        cacheEntryLines(resp.Msg.GetEntriesMetadata()),
	})
}

func (h *handlers) clear(ctx cliapp.RunContext) error {
	all := ctx.BoolFlag("all")
	targetArg := strings.TrimSpace(ctx.Positional("target"))
	if all && targetArg != "" {
		return fmt.Errorf("--all cannot be combined with a target")
	}
	if !all && targetArg == "" {
		return fmt.Errorf("target is required unless --all is set")
	}
	resp, err := h.client.ClearCache(context.Background(), connect.NewRequest(&factsv1.ClearCacheRequest{
		Target: parseTarget(targetArg),
		DryRun: ctx.BoolFlag("dry-run"),
		All:    all,
	}))
	if err != nil {
		return cliapp.WrapAPIError("clear cache", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no cache clear response")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Matched %d entrie(s), cleared %d.", resp.Msg.GetMatchedEntries(), resp.Msg.GetClearedEntries())},
	})
}

func parseTarget(raw string) *factsv1.CodeTarget {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if scenario, ok := strings.CutPrefix(raw, "scenario:"); ok {
		return &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: scenario}
	}
	return &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_PATH, Path: raw}
}

func cacheEntryLines(entries []*factsv1.CacheMetadata) []string {
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		out = append(out, fmt.Sprintf("%s %s — %s source=%s config=%s hits=%d",
			entry.GetScope(), entry.GetCacheKey(), entry.GetState(), entry.GetSourceHash(), entry.GetConfigHash(), entry.GetHitCount()))
	}
	return out
}
