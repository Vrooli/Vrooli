package facts

import (
	"context"
	"fmt"
	"strconv"
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

func (h *handlers) describe(ctx cliapp.RunContext) error {
	target := parseTarget(ctx.Positional("target"))
	include, err := parseFamilies(ctx.Flag("include"))
	if err != nil {
		return err
	}
	resp, err := h.client.DescribeCodeFacts(context.Background(), connect.NewRequest(&factsv1.DescribeCodeFactsRequest{
		Target:   target,
		Include:  include,
		UseCache: !ctx.BoolFlag("no-cache"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("describe code facts", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no code facts report")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Resolved %s with %d surface(s), %d parse unit(s), %d fact(s).",
				resp.Msg.GetTarget().GetRootPath(), len(resp.Msg.GetSurfaces()), len(resp.Msg.GetParseUnits()), len(resp.Msg.GetFacts())),
		},
		ResultsHeading: "Evidence",
		Results:        evidenceLines(resp.Msg.GetEvidence()),
		RetrievalHints: []string{
			"`facts surfaces <target>` — inspect scenario/generic surfaces",
			"`facts proto-adoption <target>` — request proto adoption proof evidence",
		},
	})
}

func (h *handlers) describeCall(ctx cliapp.OperationContext) (*factsv1.CodeFactsReport, error) {
	target := parseTarget(ctx.Positional("target"))
	include, err := parseFamilies(ctx.Flag("include"))
	if err != nil {
		return nil, err
	}
	resp, err := h.client.DescribeCodeFacts(context.Background(), connect.NewRequest(&factsv1.DescribeCodeFactsRequest{Target: target, Include: include, UseCache: !ctx.BoolFlag("no-cache")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("describe code facts", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no code facts report")
	}
	return resp.Msg, nil
}

func (h *handlers) describeReport(_ cliapp.OperationContext, msg *factsv1.CodeFactsReport) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Resolved %s with %d surface(s), %d parse unit(s), %d fact(s).", msg.GetTarget().GetRootPath(), len(msg.GetSurfaces()), len(msg.GetParseUnits()), len(msg.GetFacts()))}, ResultsHeading: "Evidence", Results: evidenceLines(msg.GetEvidence()), RetrievalHints: []string{"`facts surfaces <target>` — inspect scenario/generic surfaces", "`facts proto-adoption <target>` — request proto adoption proof evidence"}}
}

func (h *handlers) search(ctx cliapp.RunContext) error {
	query := strings.TrimSpace(ctx.Positional("query"))
	if query == "" {
		return fmt.Errorf("a search query is required")
	}
	var target *factsv1.CodeTarget
	if raw := strings.TrimSpace(ctx.Positional("target")); raw != "" {
		target = parseTarget(raw)
	}
	families, err := parseFamilies(ctx.Flag("include"))
	if err != nil {
		return err
	}
	resp, err := h.client.Search(context.Background(), connect.NewRequest(&factsv1.SearchRequest{
		Query: query, Limit: int32(parseSearchLimit(ctx.Flag("limit"))), Target: target, Families: families,
	}))
	if err != nil {
		return cliapp.WrapAPIError("search code facts", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no code-facts search response")
	}
	results := make([]string, 0, len(resp.Msg.GetResults()))
	for _, hit := range resp.Msg.GetResults() {
		results = append(results, fmt.Sprintf("%s — %s (%s, %.2f) %s", hit.GetTitle(), hit.GetPath(), hit.GetFactKind(), hit.GetScore(), hit.GetText()))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d code-evidence hit(s).", len(results))}, ResultsHeading: "Code evidence", Results: results})
}

func (h *handlers) searchCall(ctx cliapp.OperationContext) (*factsv1.SearchResponse, error) {
	query := strings.TrimSpace(ctx.Positional("query"))
	if query == "" {
		return nil, fmt.Errorf("a search query is required")
	}
	var target *factsv1.CodeTarget
	if raw := strings.TrimSpace(ctx.Positional("target")); raw != "" {
		target = parseTarget(raw)
	}
	families, err := parseFamilies(ctx.Flag("include"))
	if err != nil {
		return nil, err
	}
	resp, err := h.client.Search(context.Background(), connect.NewRequest(&factsv1.SearchRequest{
		Query: query, Limit: int32(parseSearchLimit(ctx.Flag("limit"))), Target: target, Families: families,
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("search code facts", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) searchReport(_ cliapp.OperationContext, resp *factsv1.SearchResponse) cliapp.ListReport {
	results := make([]string, 0, len(resp.GetResults()))
	for _, hit := range resp.GetResults() {
		results = append(results, fmt.Sprintf("%s — %s (%s, %.2f) %s", hit.GetTitle(), hit.GetPath(), hit.GetFactKind(), hit.GetScore(), hit.GetText()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d code-evidence hit(s).", len(results))}, ResultsHeading: "Code evidence", Results: results}
}

func parseSearchLimit(raw string) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return 10
	}
	return value
}

func (h *handlers) surfaces(ctx cliapp.RunContext) error {
	target := parseTarget(ctx.Positional("target"))
	resp, err := h.client.ListSurfaces(context.Background(), connect.NewRequest(&factsv1.ListSurfacesRequest{
		Target:   target,
		UseCache: !ctx.BoolFlag("no-cache"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list surfaces", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no surfaces response")
	}
	results := make([]string, 0, len(resp.Msg.GetSurfaces()))
	for _, surface := range resp.Msg.GetSurfaces() {
		results = append(results, fmt.Sprintf("%s — %s (%s)", surface.GetId(), surface.GetKind(), surface.GetStatus()))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d surface(s).", len(resp.Msg.GetSurfaces()))},
		ResultsHeading: "Surfaces",
		Results:        results,
	})
}

func (h *handlers) surfacesCall(ctx cliapp.OperationContext) (*factsv1.ListSurfacesResponse, error) {
	resp, err := h.client.ListSurfaces(context.Background(), connect.NewRequest(&factsv1.ListSurfacesRequest{Target: parseTarget(ctx.Positional("target")), UseCache: !ctx.BoolFlag("no-cache")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list surfaces", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no surfaces response")
	}
	return resp.Msg, nil
}

func (h *handlers) surfacesReport(_ cliapp.OperationContext, msg *factsv1.ListSurfacesResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.GetSurfaces()))
	for _, surface := range msg.GetSurfaces() {
		results = append(results, fmt.Sprintf("%s — %s (%s)", surface.GetId(), surface.GetKind(), surface.GetStatus()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d surface(s).", len(msg.GetSurfaces()))}, ResultsHeading: "Surfaces", Results: results}
}

func (h *handlers) protoAdoption(ctx cliapp.RunContext) error {
	target := parseTarget(ctx.Positional("target"))
	resp, err := h.client.CheckProtoAdoption(context.Background(), connect.NewRequest(&factsv1.CheckProtoAdoptionRequest{
		Target:   target,
		Surfaces: splitCSVValues(ctx.FlagValues("surface")),
		UseCache: !ctx.BoolFlag("no-cache"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("check proto adoption", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no proto adoption response")
	}
	return renderProof(ctx, resp.Msg)
}

func (h *handlers) protoAdoptionCall(ctx cliapp.OperationContext) (*factsv1.ProofReport, error) {
	resp, err := h.client.CheckProtoAdoption(context.Background(), connect.NewRequest(&factsv1.CheckProtoAdoptionRequest{Target: parseTarget(ctx.Positional("target")), Surfaces: splitCSVValues(ctx.FlagValues("surface")), UseCache: !ctx.BoolFlag("no-cache")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("check proto adoption", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no proto adoption response")
	}
	return resp.Msg, nil
}

func (h *handlers) endpointProof(ctx cliapp.RunContext) error {
	target := parseTarget(ctx.Positional("target"))
	resp, err := h.client.CheckEndpointProof(context.Background(), connect.NewRequest(&factsv1.CheckEndpointProofRequest{
		Target:      target,
		EndpointIds: splitCSVValues(ctx.FlagValues("endpoint")),
		UseCache:    !ctx.BoolFlag("no-cache"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("check endpoint proof", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no endpoint proof response")
	}
	return renderProof(ctx, resp.Msg)
}

func (h *handlers) endpointProofCall(ctx cliapp.OperationContext) (*factsv1.ProofReport, error) {
	resp, err := h.client.CheckEndpointProof(context.Background(), connect.NewRequest(&factsv1.CheckEndpointProofRequest{Target: parseTarget(ctx.Positional("target")), EndpointIds: splitCSVValues(ctx.FlagValues("endpoint")), UseCache: !ctx.BoolFlag("no-cache")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("check endpoint proof", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no endpoint proof response")
	}
	return resp.Msg, nil
}

func (h *handlers) proofReport(_ cliapp.OperationContext, msg *factsv1.ProofReport) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%s returned %d evidence item(s).", msg.GetFamily(), len(msg.GetEvidence()))}, ResultsHeading: "Evidence", Results: evidenceLines(msg.GetEvidence())}
}

func renderProof(ctx cliapp.RunContext, msg *factsv1.ProofReport) error {
	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%s returned %d evidence item(s).", msg.GetFamily(), len(msg.GetEvidence()))},
		ResultsHeading: "Evidence",
		Results:        evidenceLines(msg.GetEvidence()),
	})
}

func parseTarget(raw string) *factsv1.CodeTarget {
	raw = strings.TrimSpace(raw)
	switch {
	case strings.HasPrefix(raw, "scenario:"):
		return &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: strings.TrimPrefix(raw, "scenario:")}
	case strings.HasPrefix(raw, "project:"):
		return &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_PROJECT, RepoRoot: strings.TrimPrefix(raw, "project:")}
	case strings.HasPrefix(raw, "repo:"):
		return &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_REPO, RepoRoot: strings.TrimPrefix(raw, "repo:")}
	case strings.HasPrefix(raw, "package:"):
		return &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_PACKAGE, PackageName: strings.TrimPrefix(raw, "package:")}
	case strings.HasPrefix(raw, "control-plane:"):
		return &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_CONTROL_PLANE, RepoRoot: strings.TrimPrefix(raw, "control-plane:")}
	default:
		return &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_PATH, Path: raw}
	}
}

func parseFamilies(raw string) ([]factsv1.FactFamily, error) {
	parts := splitCSV(raw)
	if len(parts) == 0 {
		return nil, nil
	}
	out := make([]factsv1.FactFamily, 0, len(parts))
	for _, part := range parts {
		key := "FACT_FAMILY_" + strings.ToUpper(strings.ReplaceAll(part, "-", "_"))
		key = strings.ReplaceAll(key, " ", "_")
		value, ok := factsv1.FactFamily_value[key]
		if !ok {
			return nil, fmt.Errorf("unknown fact family %q", part)
		}
		out = append(out, factsv1.FactFamily(value))
	}
	return out, nil
}

func splitCSV(raw string) []string {
	return splitCSVValues([]string{raw})
}

func splitCSVValues(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, raw := range values {
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" || seen[part] {
				continue
			}
			seen[part] = true
			out = append(out, part)
		}
	}
	return out
}

func evidenceLines(evidence []*factsv1.Evidence) []string {
	out := make([]string, 0, len(evidence))
	for _, ev := range evidence {
		out = append(out, fmt.Sprintf("%s — %s", ev.GetStatus(), ev.GetMessage()))
	}
	return out
}
