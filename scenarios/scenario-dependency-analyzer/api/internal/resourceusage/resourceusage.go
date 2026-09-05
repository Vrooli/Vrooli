// Package resourceusage builds and serves SDA's resource-usage federated leaf:
// which scenarios declare (use) a given local resource. The corpus is derived
// from a fleet scan of every scenario's .vrooli/service.json, inverting
// scenario->resources into resource->consuming-scenarios. It owns the
// ResourceUsageService Connect handler and the aisearch.ResourceUsageProvider
// fleet-scan seam.
package resourceusage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"

	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/aisearch"
	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/config"

	resourceusagev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/resource_usage"
	resourceusageconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/resource_usage/resource_usage_v1connect"
)

// UsageSearcher is the read seam the SearchResourceUsage handler consumes: the
// multi-corpus aisearch service, queried against the .resources corpus.
type UsageSearcher interface {
	SearchCorpus(ctx context.Context, corpus aisearch.CorpusID, query string, limit int) ([]aisearch.CorpusResult, error)
}

// RegisterConnectRoutes mounts the ResourceUsageService Connect contract. A nil
// searcher degrades SearchResourceUsage to an honest unavailable.
func RegisterConnectRoutes(router *gin.Engine, searcher UsageSearcher) {
	connectPath, connectHandler := resourceusageconnect.NewResourceUsageServiceHandler(&handler{searcher: searcher})
	router.Any(connectPath+"*path", gin.WrapH(connectHandler))
}

type handler struct {
	searcher UsageSearcher
}

// SearchResourceUsage is the federated AI-search leaf over resource usage.
func (h *handler) SearchResourceUsage(ctx context.Context, req *connect.Request[resourceusagev1.SearchResourceUsageRequest]) (*connect.Response[resourceusagev1.SearchResourceUsageResponse], error) {
	if h == nil || h.searcher == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("resource usage search is not configured"))
	}
	msg := req.Msg
	if msg == nil {
		msg = &resourceusagev1.SearchResourceUsageRequest{}
	}
	results, err := h.searcher.SearchCorpus(ctx, aisearch.CorpusResources, msg.GetQuery(), int(msg.GetLimit()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &resourceusagev1.SearchResourceUsageResponse{
		Results: make([]*resourceusagev1.ResourceUsageHit, 0, len(results)),
	}
	for _, r := range results {
		usedBy := payloadStrings(r.Payload, "used_by")
		out.Results = append(out.Results, &resourceusagev1.ResourceUsageHit{
			Resource:       payloadString(r.Payload, "resource", r.SourceID),
			Type:           payloadString(r.Payload, "type", ""),
			UsedBy:         usedBy,
			Summary:        usageSummary(usedBy),
			RelevanceScore: r.Score,
		})
	}
	return connect.NewResponse(out), nil
}

var _ resourceusageconnect.ResourceUsageServiceHandler = (*handler)(nil)

func usageSummary(usedBy []string) string {
	if len(usedBy) == 0 {
		return "Used by scenarios: (none)."
	}
	return "Used by scenarios: " + strings.Join(usedBy, ", ") + "."
}

func payloadString(payload map[string]any, key, fallback string) string {
	if payload != nil {
		if v, ok := payload[key].(string); ok && strings.TrimSpace(v) != "" {
			return v
		}
	}
	return fallback
}

func payloadStrings(payload map[string]any, key string) []string {
	if payload == nil {
		return nil
	}
	switch v := payload[key].(type) {
	case []string:
		return append([]string(nil), v...)
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// UsageProvider implements aisearch.ResourceUsageProvider by scanning every
// scenario's service.json and inverting scenario->resources.
type UsageProvider struct {
	scenariosDir func() string
}

// NewUsageProvider builds the fleet-scan provider. scenariosDir resolves the
// directory holding all scenario folders.
func NewUsageProvider(scenariosDir func() string) *UsageProvider {
	return &UsageProvider{scenariosDir: scenariosDir}
}

// ResourceUsages scans the fleet and returns one record per declared resource,
// each carrying the sorted set of consuming scenarios.
func (p *UsageProvider) ResourceUsages(_ context.Context) ([]aisearch.ResourceUsage, error) {
	dir := ""
	if p.scenariosDir != nil {
		dir = strings.TrimSpace(p.scenariosDir())
	}
	if dir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	// resource name -> {type, consuming scenarios}
	consumers := map[string]map[string]struct{}{}
	types := map[string]string{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		scenario := entry.Name()
		scenarioPath := filepath.Join(dir, scenario)
		cfg, err := config.LoadServiceConfig(scenarioPath)
		if err != nil {
			continue // missing/invalid service.json — skip silently
		}
		for name, res := range config.ResolvedResourceMap(cfg) {
			resName := strings.TrimSpace(name)
			if resName == "" {
				continue
			}
			if consumers[resName] == nil {
				consumers[resName] = map[string]struct{}{}
			}
			consumers[resName][scenario] = struct{}{}
			if t := strings.TrimSpace(res.Type); t != "" && types[resName] == "" {
				types[resName] = t
			}
		}
	}
	names := make([]string, 0, len(consumers))
	for name := range consumers {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]aisearch.ResourceUsage, 0, len(names))
	for _, name := range names {
		used := make([]string, 0, len(consumers[name]))
		for s := range consumers[name] {
			used = append(used, s)
		}
		sort.Strings(used)
		out = append(out, aisearch.ResourceUsage{
			Resource: name,
			Type:     types[name],
			UsedBy:   used,
		})
	}
	return out, nil
}
