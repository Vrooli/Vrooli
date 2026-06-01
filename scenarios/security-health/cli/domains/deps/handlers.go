package deps

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	dependenciesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/security-health/v1/dependencies"
	dependenciesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/security-health/v1/dependencies/dependencies_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client dependenciesconnect.DependencyServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: dependenciesconnect.NewDependencyServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) search(ctx cliapp.RunContext) error {
	limit := 0
	if raw := strings.TrimSpace(ctx.Flag("limit")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			limit = v
		}
	}
	req := &dependenciesv1.SearchRequest{
		Query:          ctx.Positional("query"),
		Limit:          int32(limit),
		Mode:           parseMode(ctx.Flag("mode")),
		Ecosystem:      parseEcosystem(ctx.Flag("ecosystem")),
		VulnerableOnly: ctx.BoolFlag("vulnerable-only"),
		NameGlob:       ctx.Flag("name-glob"),
	}
	resp, err := h.client.Search(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("deps search", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no search response")
	}
	results := make([]string, 0, len(resp.Msg.Results))
	for _, r := range resp.Msg.Results {
		rec := r.GetRecord()
		line := fmt.Sprintf("%-8s %s@%s  [%s]", ecosystemLabel(rec.GetEcosystem()), rec.GetName(), rec.GetVersion(), rec.GetScenario())
		if len(rec.GetVulnIds()) > 0 {
			line += fmt.Sprintf("  ⚠ %s: %s", rec.GetMaxSeverity(), strings.Join(rec.GetVulnIds(), ","))
		}
		results = append(results, line)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d result(s) (mode=%s)", len(resp.Msg.Results), modeLabel(resp.Msg.GetModeUsed()))},
		ResultsHeading: "Dependencies",
		Results:        results,
	})
}

func (h *handlers) status(ctx cliapp.RunContext) error {
	resp, err := h.client.Status(context.Background(), connect.NewRequest(&dependenciesv1.StatusRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("deps status", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no status response")
	}
	m := resp.Msg
	return cliapp.RenderProtoList(ctx, m, cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("available=%v ollama=%v qdrant=%v", m.GetAvailable(), m.GetOllama(), m.GetQdrant()),
			fmt.Sprintf("indexed=%d vulnerable=%d", m.GetIndexedCount(), m.GetVulnerableCount()),
			fmt.Sprintf("last reconcile: %s — %s", nonEmpty(m.GetLastReconcileAt(), "never"), nonEmpty(m.GetLastReconcileOutcome(), "—")),
		},
		ResultsHeading: "Status",
		Results:        []string{},
	})
}

func parseMode(raw string) dependenciesv1.Mode {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "ai":
		return dependenciesv1.Mode_MODE_AI
	case "text":
		return dependenciesv1.Mode_MODE_TEXT
	default:
		return dependenciesv1.Mode_MODE_UNSPECIFIED
	}
}

func parseEcosystem(raw string) dependenciesv1.Ecosystem {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "go":
		return dependenciesv1.Ecosystem_ECOSYSTEM_GO
	case "npm":
		return dependenciesv1.Ecosystem_ECOSYSTEM_NPM
	default:
		return dependenciesv1.Ecosystem_ECOSYSTEM_UNSPECIFIED
	}
}

func ecosystemLabel(e dependenciesv1.Ecosystem) string {
	switch e {
	case dependenciesv1.Ecosystem_ECOSYSTEM_GO:
		return "go"
	case dependenciesv1.Ecosystem_ECOSYSTEM_NPM:
		return "npm"
	default:
		return "?"
	}
}

func modeLabel(m dependenciesv1.Mode) string {
	switch m {
	case dependenciesv1.Mode_MODE_AI:
		return "ai"
	case dependenciesv1.Mode_MODE_TEXT:
		return "text"
	default:
		return "unspecified"
	}
}

func nonEmpty(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}
