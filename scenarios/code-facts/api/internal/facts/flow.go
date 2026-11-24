package facts

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

const (
	maxFlowDepth   = 3
	maxFlowResults = 5
)

type FlowSearchHit struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Text        string     `json:"text"`
	Score       float64    `json:"score"`
	Path        string     `json:"path"`
	Paths       []string   `json:"paths"`
	GraphHash   string     `json:"graph_hash"`
	Attestation string     `json:"attestation"`
	Steps       []FlowStep `json:"steps"`
}

type FlowStep struct {
	From        string `json:"from"`
	To          string `json:"to,omitempty"`
	Call        string `json:"call"`
	Path        string `json:"path"`
	ToPath      string `json:"to_path,omitempty"`
	Line        int    `json:"line"`
	GraphHash   string `json:"graph_hash,omitempty"`
	ToGraphHash string `json:"to_graph_hash,omitempty"`
	Resolution  string `json:"resolution"`
}

type FlowSearchResponse struct {
	Query        string            `json:"query"`
	Scenario     string            `json:"scenario"`
	Results      []FlowSearchHit   `json:"results"`
	Degradations []FlowDegradation `json:"degradations,omitempty"`
}

type FlowDegradation struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type flowSymbol struct{ id, name, path, graphHash string }
type flowCall struct {
	caller, callee, name, path string
	line                       int
	graphHash                  string
}

// DeriveFeatureFlows converts Code Facts' existing resolved Go call nodes into
// bounded candidate paths. It follows only explicit callee_symbol links and
// labels the result PARTIAL because the graph does not prove runtime entry,
// dynamic dispatch, reflection, or execution conditions.
func DeriveFeatureFlows(graphs []*factsv1.LanguageGraph, query, scenario string, limit int) *FlowSearchResponse {
	response := &FlowSearchResponse{Query: strings.TrimSpace(query), Scenario: scenario, Results: []FlowSearchHit{}}
	if limit <= 0 || limit > maxFlowResults {
		limit = maxFlowResults
	}
	terms := flowTokens(query)
	symbols := map[string]flowSymbol{}
	var calls []flowCall
	for _, languageGraph := range graphs {
		if languageGraph == nil || languageGraph.GetGraph() == nil {
			continue
		}
		for _, node := range languageGraph.GetGraph().GetNodes() {
			attrs := node.GetAttributes()
			kind := attrs["kind"]
			if kind == "go_func" || kind == "go_method" {
				symbols[node.GetId()] = flowSymbol{id: node.GetId(), name: node.GetName(), path: node.GetPath(), graphHash: languageGraph.GetGraphHash()}
			}
			if kind != "go_call" {
				continue
			}
			line, _ := strconv.Atoi(attrs["start_line"])
			calls = append(calls, flowCall{caller: attrs["enclosing_symbol"], callee: attrs["callee_symbol"], name: attrs["callee"], path: node.GetPath(), line: line, graphHash: languageGraph.GetGraphHash()})
		}
	}
	if len(symbols) == 0 || len(calls) == 0 {
		response.Degradations = append(response.Degradations, FlowDegradation{Code: "call_graph_unavailable", Message: "The selected source graph has no function and call nodes; no flow path can be derived."})
		return response
	}
	byLocation := map[string][]string{}
	byCaller := map[string][]flowCall{}
	locationKey := func(name, path string) string { return path + "\x00" + name }
	for id, symbol := range symbols {
		key := locationKey(symbol.name, symbol.path)
		byLocation[key] = append(byLocation[key], id)
	}
	for _, call := range calls {
		for _, id := range byLocation[locationKey(call.caller, call.path)] {
			byCaller[id] = append(byCaller[id], call)
		}
	}
	type scoredStart struct {
		symbol flowSymbol
		score  float64
	}
	var starts []scoredStart
	for _, symbol := range symbols {
		score := flowScore(terms, symbol.name+" "+symbol.path)
		if score > 0 && len(byCaller[symbol.id]) > 0 {
			starts = append(starts, scoredStart{symbol, score})
		}
	}
	sort.Slice(starts, func(i, j int) bool {
		if starts[i].score != starts[j].score {
			return starts[i].score > starts[j].score
		}
		return starts[i].symbol.id < starts[j].symbol.id
	})
	baseIDCounts := make(map[string]int, len(starts))
	for _, start := range starts {
		baseIDCounts[flowBaseID(start.symbol)]++
	}
	if len(starts) == 0 {
		response.Degradations = append(response.Degradations, FlowDegradation{Code: "no_matching_flow_entry", Message: "No source function matching the query has a resolved call-site node in this graph."})
		return response
	}
	for _, start := range starts {
		steps := bestFlowPath(start.symbol, byCaller, symbols, map[string]bool{start.symbol.id: true}, 0)
		if len(steps) > 0 {
			hit := makeFlowHit(start.symbol, start.score, steps, flowGraphHash(graphs))
			if baseIDCounts[hit.ID] > 1 {
				hit.ID += ":" + filepath.ToSlash(start.symbol.path) + ":" + start.symbol.id
			}
			response.Results = append(response.Results, hit)
		}
		if len(response.Results) >= limit {
			break
		}
	}
	if len(response.Results) == 0 {
		response.Degradations = append(response.Degradations, FlowDegradation{Code: "no_resolved_source_path", Message: "Matching calls were present, but their callee symbols could not be joined to functions in this bounded graph."})
	}
	return response
}

func bestFlowPath(start flowSymbol, byCaller map[string][]flowCall, symbols map[string]flowSymbol, visited map[string]bool, depth int) []FlowStep {
	if depth >= maxFlowDepth {
		return nil
	}
	calls := byCaller[start.id]
	sort.Slice(calls, func(i, j int) bool {
		_, left := symbols[calls[i].callee]
		_, right := symbols[calls[j].callee]
		if left != right {
			return left
		}
		if calls[i].line != calls[j].line {
			return calls[i].line < calls[j].line
		}
		return calls[i].name < calls[j].name
	})
	var best []FlowStep
	for _, call := range calls {
		target, resolved := symbols[call.callee]
		step := FlowStep{From: start.name, Call: call.name, Path: call.path, Line: call.line, GraphHash: call.graphHash, Resolution: "callee_symbol_unmatched"}
		if resolved {
			step.To = target.name
			step.ToPath = target.path
			step.ToGraphHash = target.graphHash
			step.Resolution = "source_symbol_resolved"
		}
		candidate := []FlowStep{step}
		if resolved && !visited[target.id] {
			nextVisited := make(map[string]bool, len(visited)+1)
			for id := range visited {
				nextVisited[id] = true
			}
			nextVisited[target.id] = true
			candidate = append(candidate, bestFlowPath(target, byCaller, symbols, nextVisited, depth+1)...)
		}
		if len(candidate) > len(best) {
			best = candidate
		}
		if len(best) == maxFlowDepth {
			return best
		}
	}
	return best
}

func makeFlowHit(start flowSymbol, score float64, steps []FlowStep, graphHash string) FlowSearchHit {
	paths := []string{start.path}
	parts := []string{start.name}
	for _, step := range steps {
		parts = append(parts, fmt.Sprintf("%s -[%s:%d %s]-> %s", step.From, step.Path, step.Line, step.Resolution, step.To))
		if step.ToPath != "" && (len(paths) == 0 || paths[len(paths)-1] != step.ToPath) {
			paths = append(paths, step.ToPath)
		}
	}
	return FlowSearchHit{ID: flowBaseID(start), Title: start.name + " source call path", Text: strings.Join(parts, "; ") + "; DERIVED/PARTIAL: source call edges do not prove runtime reachability", Score: score, Path: start.path, Paths: paths, GraphHash: graphHash, Attestation: "DERIVED/PARTIAL", Steps: steps}
}

func flowBaseID(start flowSymbol) string {
	return "code-facts:flow:" + start.name + ":" + filepath.Base(start.path)
}

func flowGraphHash(graphs []*factsv1.LanguageGraph) string {
	for _, graph := range graphs {
		if graph != nil && graph.GetGraphHash() != "" {
			return graph.GetGraphHash()
		}
	}
	return ""
}

func flowTokens(value string) []string {
	return compositionTokens(value)
}

func flowScore(terms []string, text string) float64 {
	if len(terms) == 0 {
		return 0
	}
	fields := map[string]bool{}
	for _, token := range flowTokens(text) {
		fields[token] = true
	}
	matched := 0
	for _, term := range terms {
		if fields[term] {
			matched++
		}
	}
	return float64(matched) / float64(len(terms))
}
