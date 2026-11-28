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
			indexLine(m.GetIndexedVectors(), m.GetExpectedVectors(), m.GetIndexReady()),
			fmt.Sprintf("last reconcile: %s — %s", nonEmpty(m.GetLastReconcileAt(), "never"), nonEmpty(m.GetLastReconcileOutcome(), "—")),
		},
		ResultsHeading: "Status",
		Results:        []string{},
	})
}

func (h *handlers) vulnerabilities(ctx cliapp.RunContext) error {
	req := &dependenciesv1.ListVulnerabilitiesRequest{
		Ecosystem:         parseEcosystem(ctx.Flag("ecosystem")),
		PackageName:       ctx.Positional("package"),
		Scenario:          ctx.Flag("scenario"),
		VulnerabilityId:   ctx.Flag("vulnerability"),
		MinimumConfidence: parseConfidence(ctx.Flag("minimum-confidence")),
	}
	if raw := strings.TrimSpace(ctx.Flag("limit")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			req.Limit = int32(v)
		}
	}
	resp, err := h.client.ListVulnerabilities(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("deps vulnerabilities", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no vulnerabilities response")
	}
	lines := make([]string, 0, len(resp.Msg.GetVulnerabilities()))
	for _, v := range resp.Msg.GetVulnerabilities() {
		lines = append(lines, vulnerabilityLine(v))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d vulnerability record(s)", resp.Msg.GetTotal())},
		ResultsHeading: "Vulnerabilities",
		Results:        lines,
	})
}

func (h *handlers) explain(ctx cliapp.RunContext) error {
	req := &dependenciesv1.ExplainVulnerabilityRequest{
		VulnerabilityId: ctx.Positional("vulnerability"),
		Ecosystem:       parseEcosystem(ctx.Flag("ecosystem")),
		PackageName:     ctx.Flag("package"),
	}
	resp, err := h.client.ExplainVulnerability(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("deps explain", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no vulnerability explanation")
	}
	var lines []string
	if resp.Msg.GetFound() {
		lines = []string{vulnerabilityLine(resp.Msg.GetVulnerability())}
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("found=%v", resp.Msg.GetFound())},
		ResultsHeading: "Vulnerability",
		Results:        lines,
	})
}

// indexLine renders the vector-index coverage: "index: 4123/4390 (94%) —
// building" while the backfill populates, "ready" once AI mode is served.
func indexLine(indexed, expected int32, ready bool) string {
	state := "building"
	if ready {
		state = "ready"
	}
	pct := 0
	if expected > 0 {
		pct = int(float64(indexed) / float64(expected) * 100)
	}
	return fmt.Sprintf("index: %d/%d (%d%%) — %s", indexed, expected, pct, state)
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
	case "yarn":
		return dependenciesv1.Ecosystem_ECOSYSTEM_YARN
	case "bun":
		return dependenciesv1.Ecosystem_ECOSYSTEM_BUN
	case "python", "pip":
		return dependenciesv1.Ecosystem_ECOSYSTEM_PYTHON
	case "rust", "cargo":
		return dependenciesv1.Ecosystem_ECOSYSTEM_RUST
	case "c":
		return dependenciesv1.Ecosystem_ECOSYSTEM_C
	case "cpp", "c++":
		return dependenciesv1.Ecosystem_ECOSYSTEM_CPP
	default:
		return dependenciesv1.Ecosystem_ECOSYSTEM_UNSPECIFIED
	}
}

func parseConfidence(raw string) dependenciesv1.EvidenceConfidence {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "degraded":
		return dependenciesv1.EvidenceConfidence_EVIDENCE_CONFIDENCE_DEGRADED
	case "advisory":
		return dependenciesv1.EvidenceConfidence_EVIDENCE_CONFIDENCE_ADVISORY
	case "gating":
		return dependenciesv1.EvidenceConfidence_EVIDENCE_CONFIDENCE_GATING
	default:
		return dependenciesv1.EvidenceConfidence_EVIDENCE_CONFIDENCE_UNSPECIFIED
	}
}

func ecosystemLabel(e dependenciesv1.Ecosystem) string {
	switch e {
	case dependenciesv1.Ecosystem_ECOSYSTEM_GO:
		return "go"
	case dependenciesv1.Ecosystem_ECOSYSTEM_NPM:
		return "npm"
	case dependenciesv1.Ecosystem_ECOSYSTEM_YARN:
		return "yarn"
	case dependenciesv1.Ecosystem_ECOSYSTEM_BUN:
		return "bun"
	case dependenciesv1.Ecosystem_ECOSYSTEM_PYTHON:
		return "python"
	case dependenciesv1.Ecosystem_ECOSYSTEM_RUST:
		return "rust"
	case dependenciesv1.Ecosystem_ECOSYSTEM_C:
		return "c"
	case dependenciesv1.Ecosystem_ECOSYSTEM_CPP:
		return "cpp"
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

func vulnerabilityLine(v *dependenciesv1.VulnerabilityRecord) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%-8s %s@%s  %s [%s/%s] scenarios=%s fixed=%s",
		ecosystemLabel(v.GetEcosystem()),
		v.GetName(),
		v.GetVersion(),
		v.GetVulnerabilityId(),
		nonEmpty(v.GetNormalizedSeverity(), "unknown"),
		confidenceLabel(v.GetConfidence()),
		strings.Join(v.GetScenarios(), ","),
		firstFixed(v.GetFixedRanges()),
	)
}

func confidenceLabel(c dependenciesv1.EvidenceConfidence) string {
	switch c {
	case dependenciesv1.EvidenceConfidence_EVIDENCE_CONFIDENCE_DEGRADED:
		return "degraded"
	case dependenciesv1.EvidenceConfidence_EVIDENCE_CONFIDENCE_ADVISORY:
		return "advisory"
	case dependenciesv1.EvidenceConfidence_EVIDENCE_CONFIDENCE_GATING:
		return "gating"
	default:
		return "unspecified"
	}
}

func firstFixed(ranges []*dependenciesv1.FixedVersionRange) string {
	for _, r := range ranges {
		if strings.TrimSpace(r.GetRange()) != "" {
			return r.GetRange()
		}
	}
	return "unknown"
}
