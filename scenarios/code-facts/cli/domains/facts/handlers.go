package facts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliutil"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	factsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts/facts_v1connect"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	client      factsconnect.CodeFactsServiceClient
	flowHTTP    connect.HTTPClient
	flowBaseURL string
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: factsconnect.NewCodeFactsServiceClient(httpClient, baseURL), flowHTTP: httpClient, flowBaseURL: strings.TrimRight(baseURL, "/")}
}

func (h *handlers) describeCall(ctx cliapp.OperationContext) (*factsv1.CodeFactsReport, error) {
	target := parseTarget(ctx.Positional("target"))
	include, err := parseFamilies(ctx.Flag("include"))
	if err != nil {
		return nil, err
	}
	resp, err := h.client.DescribeCodeFacts(context.Background(), connect.NewRequest(&factsv1.DescribeCodeFactsRequest{Target: target, Include: include, UseCache: !ctx.BoolFlag("no-cache"), IncludeGraphs: ctx.BoolFlag("graphs")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("describe code facts", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no code facts report")
	}
	return resp.Msg, nil
}

func (h *handlers) compositionCall(ctx cliapp.OperationContext) (*factsv1.CompositionReport, error) {
	resp, err := h.client.DescribeComposition(context.Background(), connect.NewRequest(&factsv1.DescribeCompositionRequest{Target: parseTarget(ctx.Positional("target"))}))
	if err != nil {
		return nil, cliapp.WrapAPIError("describe composition", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no composition report")
	}
	return resp.Msg, nil
}

func (h *handlers) compositionReport(_ cliapp.OperationContext, report *factsv1.CompositionReport) cliapp.ListReport {
	results := make([]string, 0, len(report.GetBindings()))
	for _, binding := range report.GetBindings() {
		results = append(results, fmt.Sprintf("%s %s.%s (%s): %s %s %s %s:%d sha256:%s [%s]", binding.GetEnvironment(), binding.GetConsumer(), binding.GetDependency(), binding.GetInterfaceType(), binding.GetStatus(), binding.GetProvider(), binding.GetProviderExpression(), binding.GetSourcePath(), binding.GetLine(), binding.GetSourceHash(), binding.GetProvenance()))
	}
	summary := fmt.Sprintf("Found %d production/test composition evidence row(s). Scope: %s", len(results), report.GetAnalysisScope())
	if strings.Contains(report.GetAnalysisScope(), "supported nested command specs are joined to explicit source registrations") {
		summary = fmt.Sprintf("Found %d control-plane CLI source composition row(s). Scope: %s", len(results), report.GetAnalysisScope())
	}
	return cliapp.ListReport{Summary: []string{summary}, ResultsHeading: "Composition bindings", Results: results}
}

func (h *handlers) compositionSearchCall(ctx cliapp.OperationContext) (*factsv1.SearchResponse, error) {
	resp, err := h.client.SearchComposition(context.Background(), connect.NewRequest(&factsv1.SearchCompositionRequest{Query: strings.TrimSpace(ctx.Positional("query")), Scenario: strings.TrimSpace(ctx.Positional("scenario")), Limit: int32(parseSearchLimit(ctx.Flag("limit")))}))
	if err != nil {
		return nil, cliapp.WrapAPIError("search composition evidence", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no composition search response")
	}
	return resp.Msg, nil
}

func (h *handlers) compositionSearchReport(_ cliapp.OperationContext, response *factsv1.SearchResponse) cliapp.ListReport {
	results := make([]string, 0, len(response.GetResults()))
	for _, hit := range response.GetResults() {
		results = append(results, fmt.Sprintf("%.3f %s — %s (%s:%d sha256:%s)", hit.GetScore(), hit.GetTitle(), hit.GetText(), hit.GetPath(), hit.GetStartLine(), hit.GetSourceHash()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d ranked composition binding hit(s).", len(results))}, ResultsHeading: "Composition evidence", Results: results}
}

func (h *handlers) flowSearchCall(ctx cliapp.OperationContext) (*structpb.Struct, error) {
	query := strings.TrimSpace(ctx.Positional("query"))
	scenario := strings.TrimSpace(ctx.Positional("scenario"))
	path := strings.TrimSpace(ctx.Positional("path"))
	if query == "" || scenario == "" || path == "" {
		return nil, fmt.Errorf("query, scenario, and relative source path are required")
	}
	limit := parseSearchLimit(ctx.Flag("limit"))
	if limit > 5 {
		limit = 5
	}
	body, err := json.Marshal(map[string]any{"query": query, "scenario": scenario, "path": path, "limit": limit})
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, h.flowBaseURL+"/api/v1/flows/search", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := h.flowHTTP.Do(request)
	if err != nil {
		return nil, cliapp.WrapAPIError("search feature flow", err, nil)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("search feature flow returned HTTP %d", response.StatusCode)
	}
	var result map[string]any
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode feature flow response: %w", err)
	}
	return structpb.NewStruct(result)
}

func (h *handlers) ambientSeamsCall(ctx cliapp.OperationContext) (*structpb.Struct, error) {
	query, scenario := strings.TrimSpace(ctx.Positional("query")), strings.TrimSpace(ctx.Positional("scenario"))
	if query == "" || scenario == "" {
		return nil, fmt.Errorf("query and scenario are required")
	}
	limit := parseSearchLimit(ctx.Flag("limit"))
	if limit > 12 {
		limit = 12
	}
	body, err := json.Marshal(map[string]any{"query": query, "scenario": scenario, "limit": limit})
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, h.flowBaseURL+"/api/v1/ambient-seams/search", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := h.flowHTTP.Do(request)
	if err != nil {
		return nil, cliapp.WrapAPIError("search ambient seam evidence", err, nil)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("search ambient seams returned HTTP %d", response.StatusCode)
	}
	var result map[string]any
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode ambient seam response: %w", err)
	}
	return structpb.NewStruct(result)
}

func (h *handlers) persistencePatternsCall(ctx cliapp.OperationContext) (*structpb.Struct, error) {
	query, scenario := strings.TrimSpace(ctx.Positional("query")), strings.TrimSpace(ctx.Positional("scenario"))
	if query == "" || scenario == "" {
		return nil, fmt.Errorf("query and scenario are required")
	}
	limit := parseSearchLimit(ctx.Flag("limit"))
	if limit > 5 {
		limit = 5
	}
	body, err := json.Marshal(map[string]any{"query": query, "scenario": scenario, "limit": limit})
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, h.flowBaseURL+"/api/v1/persistence-patterns/search", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := h.flowHTTP.Do(request)
	if err != nil {
		return nil, cliapp.WrapAPIError("search persistence patterns", err, nil)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("search persistence patterns returned HTTP %d", response.StatusCode)
	}
	var result map[string]any
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode persistence pattern response: %w", err)
	}
	return structpb.NewStruct(result)
}

func (h *handlers) persistencePatternsReport(_ cliapp.OperationContext, response *structpb.Struct) cliapp.MutationReport {
	body := response.AsMap()
	raw, _ := body["results"].([]any)
	lines := make([]string, 0, len(raw)+1)
	for _, item := range raw {
		hit, _ := item.(map[string]any)
		lines = append(lines, fmt.Sprintf("%v · %v (%v:%v sha256:%v)", hit["classification"], hit["entity"], hit["source_path"], hit["line"], hit["source_sha256"]))
	}
	for _, item := range body["degradations"].([]any) {
		lines = append(lines, fmt.Sprintf("degraded: %v", item))
	}
	unknowns, _ := body["unknowns"].([]any)
	for _, item := range unknowns {
		unknown, _ := item.(map[string]any)
		schema, _ := unknown["schema"].(map[string]any)
		lines = append(lines, fmt.Sprintf("UNKNOWN lineage %v: %v (schema %v:%v sha256:%v)", unknown["entity"], unknown["reason"], schema["path"], schema["line"], schema["sha256"]))
	}
	return cliapp.MutationReport{Result: append([]string{fmt.Sprintf("Found %d joined static persistence source pattern(s), %d UNKNOWN lineage candidate(s). No live migration/database state is inferred. Scope: %v", len(raw), len(unknowns), body["scope"])}, lines...)}
}

func (h *handlers) endpointAnatomyCall(ctx cliapp.OperationContext) (*structpb.Struct, error) {
	query, scenario := strings.TrimSpace(ctx.Positional("query")), strings.TrimSpace(ctx.Positional("scenario"))
	if query == "" || scenario == "" {
		return nil, fmt.Errorf("query and scenario are required")
	}
	limit := parseSearchLimit(ctx.Flag("limit"))
	if limit > 10 {
		limit = 10
	}
	body, err := json.Marshal(map[string]any{"query": query, "scenario": scenario, "limit": limit})
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, h.flowBaseURL+"/api/v1/endpoint-anatomy/search", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := h.flowHTTP.Do(request)
	if err != nil {
		return nil, cliapp.WrapAPIError("search endpoint anatomy", err, nil)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("search endpoint anatomy returned HTTP %d", response.StatusCode)
	}
	var result map[string]any
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode endpoint anatomy response: %w", err)
	}
	return structpb.NewStruct(result)
}

func (h *handlers) endpointAnatomyReport(_ cliapp.OperationContext, response *structpb.Struct) cliapp.MutationReport {
	body := response.AsMap()
	rows, _ := body["results"].([]any)
	lines := make([]string, 0, len(rows)+1)
	for _, item := range rows {
		row, _ := item.(map[string]any)
		if row["transport"] == "connect" {
			chain, _ := row["connect_chain"].([]any)
			links := make([]string, 0, len(chain))
			for _, source := range chain {
				links = append(links, fmt.Sprintf("%v:%v sha256:%v", sourceField(source, "path"), sourceField(source, "line"), sourceField(source, "sha256")))
			}
			lines = append(lines, fmt.Sprintf("Connect %v.%v %v [%v]; source chain=%s; runtime mount remains unverified", row["connect_service"], row["connect_method"], row["route_path"], row["state"], strings.Join(links, " → ")))
		} else if row["state"] == "ROUTE_AMBIGUOUS" {
			candidates, _ := row["route_candidates"].([]any)
			branches := make([]string, 0, len(candidates))
			for _, raw := range candidates {
				candidate, _ := raw.(map[string]any)
				registration, handler := candidate["registration"], candidate["handler_source"]
				branches = append(branches, fmt.Sprintf("%v at %v:%v sha256:%v; definition=%v:%v sha256:%v", candidate["handler"], sourceField(registration, "path"), sourceField(registration, "line"), sourceField(registration, "sha256"), sourceField(handler, "path"), sourceField(handler, "line"), sourceField(handler, "sha256")))
			}
			lines = append(lines, fmt.Sprintf("%v %v [ROUTE_AMBIGUOUS; %v candidate(s), %v omitted]; candidates=%s; no handler selected; runtime dispatch precedence unknown", row["method"], row["route_path"], row["route_candidate_total"], row["route_candidates_omitted"], strings.Join(branches, " | ")))
		} else {
			lines = append(lines, fmt.Sprintf("%v %v → %v [%v]; declaration=%v:%v; route=%v:%v; handler=%v:%v", row["method"], row["route_path"], row["handler"], row["state"], sourceField(row["declaration"], "path"), sourceField(row["declaration"], "line"), sourceField(row["registration"], "path"), sourceField(row["registration"], "line"), sourceField(row["handler_source"], "path"), sourceField(row["handler_source"], "line")))
		}
	}
	for _, item := range body["degradations"].([]any) {
		lines = append(lines, fmt.Sprintf("degraded: %v", item))
	}
	return cliapp.MutationReport{Result: append([]string{fmt.Sprintf("Found %d bounded endpoint anatomy row(s). Source linkage does not prove runtime mounting. Scope: %v", len(rows), body["scope"])}, lines...)}
}

func sourceField(value any, key string) any {
	m, _ := value.(map[string]any)
	if m == nil {
		return "unknown"
	}
	v, ok := m[key]
	if !ok {
		return "unknown"
	}
	return v
}

func (h *handlers) ambientSeamsReport(_ cliapp.OperationContext, response *structpb.Struct) cliapp.MutationReport {
	body := response.AsMap()
	raw, _ := body["results"].([]any)
	lines := make([]string, 0, len(raw)+1)
	for _, item := range raw {
		hit, _ := item.(map[string]any)
		lines = append(lines, fmt.Sprintf("%v %v (%v:%v sha256:%v; match=%v; layer=%v; context=%v; lexical matches=%v in=%v; exact evidence=%s; AST-argument-context=%v %v:%v; basis=%v; %v)", hit["kind"], hit["expression"], hit["source_path"], hit["line"], hit["source_sha256"], sourceField(hit, "match_status"), hit["source_layer"], hit["context_symbol"], hit["matched_terms"], hit["matched_fields"], ambientMatchEvidenceText(hit["match_evidence"]), hit["argument_context_status"], hit["argument_callee"], hit["argument_line"], hit["basis"], hit["limit"]))
	}
	degradations, _ := body["degradations"].([]any)
	for _, item := range degradations {
		degradation, _ := item.(map[string]any)
		lines = append(lines, fmt.Sprintf("degraded %v: %v", degradation["code"], degradation["message"]))
	}
	return cliapp.MutationReport{Result: append([]string{fmt.Sprintf("Found %d bounded ambient seam source candidate(s); lexical match status=%v. Ties and unsupported syntax are UNKNOWN. Static file/call context does not establish production wiring, test substitution or runtime behavior. Scope: %v", len(raw), sourceField(body, "match_status"), body["scope"])}, lines...)}
}

func ambientMatchEvidenceText(raw any) string {
	evidence, _ := raw.([]any)
	parts := make([]string, 0, len(evidence))
	for _, item := range evidence {
		match, _ := item.(map[string]any)
		parts = append(parts, fmt.Sprintf("%v→%v=%q (%v:%v sha256:%v)", match["term"], match["field"], match["value"], match["source_path"], match["line"], match["source_sha256"]))
	}
	if len(parts) == 0 {
		return "unknown"
	}
	return strings.Join(parts, "; ")
}

func (h *handlers) flowSearchReport(_ cliapp.OperationContext, response *structpb.Struct) cliapp.MutationReport {
	body := response.AsMap()
	results, _ := body["results"].([]any)
	lines := make([]string, 0, len(results)+1)
	for _, raw := range results {
		hit, _ := raw.(map[string]any)
		lines = append(lines, fmt.Sprintf("%v — %v (%v; graph:%v; %v)", hit["title"], hit["text"], hit["path"], hit["graph_hash"], hit["attestation"]))
	}
	degradations, _ := body["degradations"].([]any)
	for _, raw := range degradations {
		degradation, _ := raw.(map[string]any)
		lines = append(lines, fmt.Sprintf("degraded %v: %v", degradation["code"], degradation["message"]))
	}
	return cliapp.MutationReport{Result: append([]string{fmt.Sprintf("Found %d bounded source flow path(s); source reachability remains partial.", len(results))}, lines...)}
}

func (h *handlers) invariantCandidatesCall(ctx cliapp.OperationContext) (*factsv1.SearchInvariantCandidatesResponse, error) {
	query, scenario, path := strings.TrimSpace(ctx.Positional("query")), strings.TrimSpace(ctx.Positional("scenario")), strings.TrimSpace(ctx.Positional("path"))
	if query == "" || scenario == "" || path == "" {
		return nil, fmt.Errorf("query, scenario, and relative source path are required")
	}
	limit := parseSearchLimit(ctx.Flag("limit"))
	if limit > 10 {
		limit = 10
	}
	response, err := h.client.SearchInvariantCandidates(context.Background(), connect.NewRequest(&factsv1.SearchInvariantCandidatesRequest{Query: query, Scenario: scenario, Path: path, Limit: int32(limit)}))
	if err != nil {
		return nil, cliapp.WrapAPIError("search invariant candidate evidence", err, nil)
	}
	if response == nil || response.Msg == nil {
		return nil, fmt.Errorf("server returned no invariant candidate response")
	}
	return response.Msg, nil
}

func (h *handlers) invariantCandidatesReport(_ cliapp.OperationContext, response *factsv1.SearchInvariantCandidatesResponse) cliapp.ListReport {
	lines := make([]string, 0, len(response.GetResults())+len(response.GetDegradations()))
	for _, hit := range response.GetResults() {
		e := hit.GetEnforcement()
		lines = append(lines, fmt.Sprintf("%s — %s (%s:%d sha256:%s; %s; type=%s db=%s test=%s runtime=%s)", hit.GetTitle(), hit.GetText(), hit.GetPath(), hit.GetLine(), hit.GetSha256(), hit.GetAttestation(), e.GetType(), e.GetDatabase(), e.GetTest(), e.GetRuntime()))
	}
	for _, d := range response.GetDegradations() {
		lines = append(lines, fmt.Sprintf("degraded %s: %s", d.GetCode(), d.GetMessage()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d candidate source location(s); enforcement is not established. Scope: %s", len(response.GetResults()), response.GetScope())}, ResultsHeading: "Invariant candidates", Results: lines}
}

func (h *handlers) describeReport(_ cliapp.OperationContext, msg *factsv1.CodeFactsReport) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Resolved %s with %d surface(s), %d parse unit(s), %d fact(s), %d language graph(s).", msg.GetTarget().GetRootPath(), len(msg.GetSurfaces()), len(msg.GetParseUnits()), len(msg.GetFacts()), len(msg.GetGraphs()))}, ResultsHeading: "Evidence", Results: evidenceLines(msg.GetEvidence()), RetrievalHints: []string{"`facts surfaces <target>` — inspect scenario/generic surfaces", "`facts proto-adoption <target>` — request proto adoption proof evidence"}}
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
	if err := validateImportSearchTarget(target, families); err != nil {
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
		line := fmt.Sprintf("%s — %s (%s, %.2f) %s", hit.GetTitle(), hit.GetPath(), hit.GetFactKind(), hit.GetScore(), hit.GetText())
		if hit.GetFactKind() == "imports" || hit.GetFactKind() == "imports_partial" {
			line += fmt.Sprintf(" [source %s hash:%s generation:%s lines:%d-%d status:%s]", hit.GetPath(), hit.GetSourceHash(), hit.GetGeneration(), hit.GetStartLine(), hit.GetEndLine(), hit.GetEvidenceStatus())
		}
		results = append(results, line)
	}
	summary := []string{fmt.Sprintf("Found %d code-evidence hit(s).", len(results))}
	if degraded := resp.GetDegradedStages(); len(degraded) > 0 {
		summary = append(summary, "Partial evidence: "+strings.Join(degraded, ", "))
	}
	return cliapp.ListReport{Summary: summary, ResultsHeading: "Code evidence", Results: results}
}

func validateImportSearchTarget(target *factsv1.CodeTarget, families []factsv1.FactFamily) error {
	importsRequested := false
	for _, family := range families {
		if family == factsv1.FactFamily_FACT_FAMILY_IMPORTS {
			importsRequested = true
			break
		}
	}
	if !importsRequested {
		return nil
	}
	if target == nil || target.GetKind() != factsv1.TargetKind_TARGET_KIND_SCENARIO || !validCLIImportScenarioSlug(target.GetScenario()) {
		return fmt.Errorf("include=imports requires target scenario:<slug>; project, path, and default targets are not supported")
	}
	return nil
}

func validCLIImportScenarioSlug(value string) bool {
	if value == "" || len(value) > 64 {
		return false
	}
	first := value[0]
	if (first < 'a' || first > 'z') && (first < '0' || first > '9') {
		return false
	}
	for _, char := range value {
		if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '-' {
			return false
		}
	}
	return true
}

func parseSearchLimit(raw string) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return 10
	}
	return value
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

func (h *handlers) protoAdoptionCall(ctx cliapp.OperationContext) (*factsv1.ProofReport, error) {
	resp, err := h.client.CheckProtoAdoption(context.Background(), connect.NewRequest(&factsv1.CheckProtoAdoptionRequest{Target: parseTarget(ctx.Positional("target")), Surfaces: cliutil.ParseUniqueCSVValues(ctx.FlagValues("surface")), UseCache: !ctx.BoolFlag("no-cache")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("check proto adoption", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no proto adoption response")
	}
	return resp.Msg, nil
}

func (h *handlers) endpointProofCall(ctx cliapp.OperationContext) (*factsv1.ProofReport, error) {
	resp, err := h.client.CheckEndpointProof(context.Background(), connect.NewRequest(&factsv1.CheckEndpointProofRequest{Target: parseTarget(ctx.Positional("target")), EndpointIds: cliutil.ParseUniqueCSVValues(ctx.FlagValues("endpoint")), UseCache: !ctx.BoolFlag("no-cache")}))
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
	parts := cliutil.ParseUniqueCSVValues([]string{raw})
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

func evidenceLines(evidence []*factsv1.Evidence) []string {
	out := make([]string, 0, len(evidence))
	for _, ev := range evidence {
		out = append(out, fmt.Sprintf("%s — %s", ev.GetStatus(), ev.GetMessage()))
	}
	return out
}
