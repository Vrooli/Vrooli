package graph

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"architecture-cartographer/internal/domains"
	intgraph "architecture-cartographer/internal/graph"
	intslice "architecture-cartographer/internal/slice"
	"architecture-cartographer/internal/zones"
	"google.golang.org/protobuf/proto"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/graph"
)

const (
	maxSliceSearchBodyBytes = 64 << 10
	maxSliceSearchResults   = 5
	maxSliceEvidenceItems   = 4
)

var sliceSearchTokens = regexp.MustCompile(`[a-zA-Z0-9]+`)
var scenarioSlugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

type sliceSearchRequest struct {
	Query      string `json:"query"`
	Scope      string `json:"scope"`
	Limit      int    `json:"limit"`
	SnapshotID string `json:"snapshot_id,omitempty"`
}

type sliceSearchResponse struct {
	Query        string             `json:"query"`
	Scope        string             `json:"scope"`
	Results      []sliceSearchHit   `json:"results"`
	Degradations []sliceDegradation `json:"degradations,omitempty"`
}

type sliceSearchHit struct {
	ID         string               `json:"id"`
	Title      string               `json:"title"`
	Score      float64              `json:"score"`
	Text       string               `json:"text"`
	Path       string               `json:"path,omitempty"`
	Paths      []string             `json:"paths,omitempty"`
	Scenario   string               `json:"scenario"`
	Domain     string               `json:"domain"`
	SnapshotID string               `json:"snapshot_id"`
	Slice      *graphv1.DomainSlice `json:"slice"`
}

type sliceDegradation struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type rankedDomain struct {
	domain domains.DerivedDomain
	score  float64
}

// SearchSlicesHTTP returns ranked slice anatomy for exactly one scenario. It
// uses only an existing graph snapshot; this read path never initiates a graph
// extraction or scans the scenario fleet.
func (h *Handler) SearchSlicesHTTP() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		decoder := json.NewDecoder(io.LimitReader(r.Body, maxSliceSearchBodyBytes))
		decoder.DisallowUnknownFields()
		var req sliceSearchRequest
		if err := decoder.Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			http.Error(w, "request body must contain one JSON object", http.StatusBadRequest)
			return
		}
		scenario, err := scenarioFromScope(req.Scope)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		query := strings.TrimSpace(req.Query)
		if query == "" {
			http.Error(w, "query is required", http.StatusBadRequest)
			return
		}
		limit := req.Limit
		if limit == 0 {
			limit = maxSliceSearchResults
		}
		if limit < 1 || limit > maxSliceSearchResults {
			http.Error(w, fmt.Sprintf("limit must be between 1 and %d", maxSliceSearchResults), http.StatusBadRequest)
			return
		}
		if h.domainsSvc == nil {
			http.Error(w, "domain map service unavailable", http.StatusServiceUnavailable)
			return
		}

		snapshot, err := h.cachedSnapshot(r.Context(), scenario, strings.TrimSpace(req.SnapshotID))
		if err != nil {
			if errors.Is(err, errSliceSnapshotUnavailable) {
				writeSliceSearchJSON(w, http.StatusOK, sliceSearchResponse{
					Query: query, Scope: req.Scope,
					Degradations: []sliceDegradation{{Code: "graph_snapshot_unavailable", Message: "No cached graph snapshot exists for this scenario; use GraphService.GetSlice or extract a graph first."}},
				})
				return
			}
			if errors.Is(err, errSliceSnapshotScopeMismatch) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			http.Error(w, "graph snapshot unavailable", http.StatusServiceUnavailable)
			return
		}
		domainMap, err := h.domainsSvc.GetDomainMap(r.Context(), scenario)
		if err != nil {
			http.Error(w, "domain map unavailable", http.StatusServiceUnavailable)
			return
		}
		results := rankedDomainsForSlice(query, domainMap, limit)
		response := sliceSearchResponse{Query: query, Scope: req.Scope, Results: make([]sliceSearchHit, 0, len(results))}
		for _, result := range results {
			derived := intslice.Build(
				sliceSnapshot(snapshot),
				sliceDomainMap(domainMap),
				sliceClassifier{cfg: zones.LoadForScenarioName(scenario), domains: domainMap},
				result.domain.Name,
			)
			slice := sliceToProto(derived)
			response.Results = append(response.Results, projectSliceHit(domainMap, slice, result.score))
		}
		writeSliceSearchJSON(w, http.StatusOK, response)
	}
}

var errSliceSnapshotUnavailable = errors.New("cached graph snapshot unavailable")
var errSliceSnapshotScopeMismatch = errors.New("snapshot is outside the requested scenario scope")

func (h *Handler) cachedSnapshot(ctx context.Context, scenario, snapshotID string) (intgraph.GraphSnapshot, error) {
	if snapshotID != "" {
		snapshot, err := h.svc.GetSnapshot(ctx, snapshotID)
		if err == nil && snapshot.Scenario != scenario {
			return intgraph.GraphSnapshot{}, fmt.Errorf("%w: snapshot %q belongs to scenario %q", errSliceSnapshotScopeMismatch, snapshotID, snapshot.Scenario)
		}
		return snapshot, err
	}
	page, err := h.svc.ListSnapshots(ctx, intgraph.ListSnapshotsFilter{Scenario: scenario, PageSize: 1})
	if err != nil {
		return intgraph.GraphSnapshot{}, err
	}
	if len(page.Snapshots) == 0 {
		return intgraph.GraphSnapshot{}, errSliceSnapshotUnavailable
	}
	return page.Snapshots[0], nil
}

func scenarioFromScope(scope string) (string, error) {
	scope = strings.TrimSpace(scope)
	if !strings.HasPrefix(scope, "scenario:") {
		return "", errors.New("scope must be scenario:<slug>")
	}
	scenario := strings.TrimPrefix(scope, "scenario:")
	if !scenarioSlugPattern.MatchString(scenario) {
		return "", errors.New("scope must contain exactly one scenario slug")
	}
	return scenario, nil
}

func rankedDomainsForSlice(query string, domainMap domains.DerivedDomainMap, limit int) []rankedDomain {
	tokens := uniqueSliceTokens(query)
	if len(tokens) == 0 {
		return nil
	}
	ranked := make([]rankedDomain, 0, len(domainMap.Domains))
	for _, domain := range domainMap.Domains {
		score := scoreSliceDomain(tokens, domain)
		if score > 0 {
			ranked = append(ranked, rankedDomain{domain: domain, score: score})
		}
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return ranked[i].domain.Name < ranked[j].domain.Name
	})
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	return ranked
}

func scoreSliceDomain(queryTokens []string, domain domains.DerivedDomain) float64 {
	fields := []struct {
		text   string
		weight float64
	}{
		{domain.Name, 3},
		{strings.Join(domain.Glossary, " "), 2},
		{domain.Responsibility, 1.5},
		{domain.Purpose, 1.5},
		{domain.OwnsData, 1},
		{strings.Join(domain.Paths, " "), 1},
	}
	matched := map[string]float64{}
	for _, field := range fields {
		for _, token := range uniqueSliceTokens(field.text) {
			if _, exists := matched[token]; !exists {
				matched[token] = field.weight
			}
		}
	}
	var earned, possible float64
	for _, token := range queryTokens {
		possible += 3
		if weight, ok := matched[token]; ok {
			earned += weight
		}
	}
	if possible == 0 {
		return 0
	}
	return earned / possible
}

func projectSliceHit(domainMap domains.DerivedDomainMap, slice *graphv1.DomainSlice, score float64) sliceSearchHit {
	var domain *domains.DerivedDomain
	for i := range domainMap.Domains {
		if domainMap.Domains[i].Name == slice.GetDomain() {
			domain = &domainMap.Domains[i]
			break
		}
	}
	paths := make([]string, 0, maxSliceEvidenceItems)
	parts := []string{fmt.Sprintf("%s/%s implementation slice", slice.GetScenario(), slice.GetDomain())}
	if domain != nil {
		if domain.Responsibility != "" {
			parts = append(parts, "Responsibility: "+strings.TrimSpace(domain.Responsibility))
		}
		if domain.Purpose != "" {
			parts = append(parts, "Purpose: "+strings.TrimSpace(domain.Purpose))
		}
	}
	present := 0
	for _, rung := range slice.GetRungs() {
		state := "missing"
		if rung.GetPresent() {
			state = "present"
			present++
		}
		parts = append(parts, fmt.Sprintf("%s rung %s (%d file(s), %d symbol(s))", rung.GetName(), state, len(rung.GetFiles()), len(rung.GetSymbols())))
		for _, file := range rung.GetFiles() {
			if len(paths) < maxSliceEvidenceItems {
				paths = append(paths, file.GetPath())
			}
		}
		for _, evidence := range rung.GetEvidence() {
			if len(paths) < maxSliceEvidenceItems {
				paths = append(paths, evidence.GetValue())
			}
		}
	}
	parts = append(parts, fmt.Sprintf("Attestation DERIVED/PARTIAL (%d/5 rungs present; symbol-level links remain outside this slice)", present))
	for _, edge := range slice.GetLayerEdges() {
		parts = append(parts, "Layer edge "+edge.GetFromRung()+"->"+edge.GetToRung())
	}
	id := slice.GetScenario() + "/" + slice.GetDomain()
	path := ""
	if len(paths) > 0 {
		path = paths[0]
	}
	return sliceSearchHit{
		ID: id, Title: id + " implementation slice", Score: score,
		Text: strings.Join(parts, "; "), Path: path, Paths: paths,
		Scenario: slice.GetScenario(), Domain: slice.GetDomain(), SnapshotID: slice.GetSnapshotId(), Slice: boundedSlice(slice),
	}
}

// boundedSlice keeps each rung's anatomy useful while limiting payload growth
// when a domain contains many files or symbols.
func boundedSlice(in *graphv1.DomainSlice) *graphv1.DomainSlice {
	out := proto.Clone(in).(*graphv1.DomainSlice)
	for _, rung := range out.GetRungs() {
		if len(rung.GetFiles()) > maxSliceEvidenceItems {
			rung.Files = rung.GetFiles()[:maxSliceEvidenceItems]
		}
		if len(rung.GetSymbols()) > maxSliceEvidenceItems {
			rung.Symbols = rung.GetSymbols()[:maxSliceEvidenceItems]
		}
		if len(rung.GetEvidence()) > maxSliceEvidenceItems {
			rung.Evidence = rung.GetEvidence()[:maxSliceEvidenceItems]
		}
	}
	if len(out.GetLayerEdges()) > maxSliceEvidenceItems {
		out.LayerEdges = out.GetLayerEdges()[:maxSliceEvidenceItems]
	}
	return out
}

func uniqueSliceTokens(value string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, raw := range sliceSearchTokens.FindAllString(strings.ToLower(value), -1) {
		if len(raw) < 2 || raw == "the" || raw == "and" || raw == "how" || raw == "what" || raw == "does" || raw == "which" || raw == "where" || raw == "with" || raw == "from" || raw == "into" {
			continue
		}
		if _, exists := seen[raw]; exists {
			continue
		}
		seen[raw] = struct{}{}
		out = append(out, raw)
	}
	return out
}

func writeSliceSearchJSON(w http.ResponseWriter, status int, body sliceSearchResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
