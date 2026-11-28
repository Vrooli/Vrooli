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
	maxFlowDepth          = 3
	maxFlowResults        = 5
	maxFlowCallsPerNode   = 8
	maxFlowPathsPerStart  = 8
	maxFlowStartFunctions = 20
)

type FlowSearchHit struct {
	ID              string     `json:"id"`
	Title           string     `json:"title"`
	Text            string     `json:"text"`
	Score           float64    `json:"score"`
	Path            string     `json:"path"`
	Paths           []string   `json:"paths"`
	GraphHash       string     `json:"graph_hash"`
	Attestation     string     `json:"attestation"`
	DepthLimited    bool       `json:"depth_limited"`
	BranchTruncated bool       `json:"branch_truncated"`
	Steps           []FlowStep `json:"steps"`
}

type FlowStep struct {
	CallSiteID  string `json:"call_site_id"`
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
	id                         string
	caller, callee, name, path string
	line                       int
	graphHash                  string
}

type flowPathCandidate struct {
	steps           []FlowStep
	depthLimited    bool
	branchTruncated bool
	score           float64
	stableKey       string
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
			calls = append(calls, flowCall{id: node.GetId(), caller: attrs["enclosing_symbol"], callee: attrs["callee_symbol"], name: attrs["callee"], path: node.GetPath(), line: line, graphHash: languageGraph.GetGraphHash()})
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
	if len(starts) > maxFlowStartFunctions {
		starts = starts[:maxFlowStartFunctions]
		response.Degradations = append(response.Degradations, FlowDegradation{Code: "flow_start_limit", Message: fmt.Sprintf("flow analysis is capped at %d matching start functions", maxFlowStartFunctions)})
	}
	var candidates []struct {
		start flowSymbol
		path  flowPathCandidate
	}
	primaryBranch := map[string]string{}
	for _, start := range starts {
		paths := enumerateFlowPaths(start.symbol, byCaller, symbols, map[string]bool{start.symbol.id: true}, 0, terms)
		for _, path := range paths {
			if len(path.steps) == 0 {
				continue
			}
			if current, ok := primaryBranch[start.symbol.id]; !ok || path.stableKey < current {
				primaryBranch[start.symbol.id] = path.stableKey
			}
			candidates = append(candidates, struct {
				start flowSymbol
				path  flowPathCandidate
			}{start.symbol, path})
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].path.score != candidates[j].path.score {
			return candidates[i].path.score > candidates[j].path.score
		}
		if len(candidates[i].path.steps) != len(candidates[j].path.steps) {
			return len(candidates[i].path.steps) > len(candidates[j].path.steps)
		}
		if candidates[i].start.id != candidates[j].start.id {
			return candidates[i].start.id < candidates[j].start.id
		}
		return candidates[i].path.stableKey < candidates[j].path.stableKey
	})
	for i, candidate := range candidates {
		hit := makeFlowHit(candidate.start, candidate.path.score, candidate.path.steps, flowGraphHash(graphs), candidate.path.depthLimited, candidate.path.branchTruncated)
		if baseIDCounts[hit.ID] > 1 {
			hit.ID += ":" + filepath.ToSlash(candidate.start.path) + ":" + candidate.start.id
		}
		if candidate.path.stableKey != primaryBranch[candidate.start.id] {
			hit.ID += ":branch:" + candidate.path.stableKey
		}
		response.Results = append(response.Results, hit)
		if len(response.Results) >= limit {
			if i+1 < len(candidates) {
				omitted := len(candidates) - i - 1
				response.Degradations = append(response.Degradations, FlowDegradation{Code: "flow_branch_result_limit", Message: fmt.Sprintf("%d additional generated ranked branch candidates were omitted at the %d-result response limit; bounded branch expansion may omit more", omitted, limit)})
			}
			break
		}
	}
	if len(response.Results) == 0 {
		response.Degradations = append(response.Degradations, FlowDegradation{Code: "no_resolved_source_path", Message: "Matching calls were present, but their callee symbols could not be joined to functions in this bounded graph."})
	}
	return response
}

func enumerateFlowPaths(start flowSymbol, byCaller map[string][]flowCall, symbols map[string]flowSymbol, visited map[string]bool, depth int, terms []string) []flowPathCandidate {
	if depth >= maxFlowDepth {
		return []flowPathCandidate{{depthLimited: flowHasContinuation(start, byCaller, symbols, visited)}}
	}
	calls := append([]flowCall(nil), byCaller[start.id]...)
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
	branchTruncated := len(calls) > maxFlowCallsPerNode
	if branchTruncated {
		calls = calls[:maxFlowCallsPerNode]
	}
	var candidates []flowPathCandidate
	for _, call := range calls {
		target, resolved := symbols[call.callee]
		step := FlowStep{CallSiteID: call.id, From: start.name, Call: call.name, Path: call.path, Line: call.line, GraphHash: call.graphHash, Resolution: "callee_symbol_unmatched"}
		if resolved {
			step.To = target.name
			step.ToPath = target.path
			step.ToGraphHash = target.graphHash
			step.Resolution = "source_symbol_resolved"
		}
		candidate := flowPathCandidate{steps: []FlowStep{step}, branchTruncated: branchTruncated}
		if resolved && !visited[target.id] {
			nextVisited := make(map[string]bool, len(visited)+1)
			for id := range visited {
				nextVisited[id] = true
			}
			nextVisited[target.id] = true
			tails := enumerateFlowPaths(target, byCaller, symbols, nextVisited, depth+1, terms)
			if len(tails) > 0 {
				candidatesForCall := make([]flowPathCandidate, 0, len(tails))
				for _, tail := range tails {
					steps := append(append([]FlowStep(nil), candidate.steps...), tail.steps...)
					candidatesForCall = append(candidatesForCall, flowPathCandidate{steps: steps, depthLimited: tail.depthLimited, branchTruncated: branchTruncated || tail.branchTruncated})
				}
				candidates = append(candidates, candidatesForCall...)
				continue
			}
		}
		candidates = append(candidates, candidate)
	}
	for i := range candidates {
		candidates[i].score = flowScore(terms, flowPathText(start, candidates[i].steps))
		candidates[i].stableKey = flowPathKey(candidates[i].steps)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		if len(candidates[i].steps) != len(candidates[j].steps) {
			return len(candidates[i].steps) > len(candidates[j].steps)
		}
		return candidates[i].stableKey < candidates[j].stableKey
	})
	if len(candidates) > maxFlowPathsPerStart {
		candidates = candidates[:maxFlowPathsPerStart]
		for i := range candidates {
			candidates[i].branchTruncated = true
		}
	}
	return candidates
}

func flowHasContinuation(start flowSymbol, byCaller map[string][]flowCall, symbols map[string]flowSymbol, visited map[string]bool) bool {
	for _, call := range byCaller[start.id] {
		target, resolved := symbols[call.callee]
		if !resolved || !visited[target.id] {
			return true
		}
	}
	return false
}

func makeFlowHit(start flowSymbol, score float64, steps []FlowStep, graphHash string, depthLimited, branchTruncated bool) FlowSearchHit {
	paths := []string{start.path}
	parts := []string{start.name}
	for _, step := range steps {
		parts = append(parts, fmt.Sprintf("%s -[%s:%d %s]-> %s", step.From, step.Path, step.Line, step.Resolution, step.To))
		if step.ToPath != "" && (len(paths) == 0 || paths[len(paths)-1] != step.ToPath) {
			paths = append(paths, step.ToPath)
		}
	}
	text := strings.Join(parts, "; ") + "; DERIVED/PARTIAL: source call edges do not prove runtime reachability"
	if depthLimited {
		text += "; depth limit reached: additional source calls are omitted"
	}
	if branchTruncated {
		text += "; branch expansion reached a configured bound: additional source branches are omitted"
	}
	return FlowSearchHit{ID: flowBaseID(start), Title: start.name + " source call path", Text: text, Score: score, Path: start.path, Paths: paths, GraphHash: graphHash, Attestation: "DERIVED/PARTIAL", DepthLimited: depthLimited, BranchTruncated: branchTruncated, Steps: steps}
}

func flowPathText(start flowSymbol, steps []FlowStep) string {
	parts := []string{start.name + " " + start.path}
	for _, step := range steps {
		parts = append(parts, step.Call, step.Path, step.To, step.ToPath)
	}
	return strings.Join(parts, " ")
}

func flowPathKey(steps []FlowStep) string {
	parts := make([]string, 0, len(steps))
	for _, step := range steps {
		parts = append(parts, fmt.Sprintf("%s:%d:%s:%s:%s", filepath.ToSlash(step.Path), step.Line, step.CallSiteID, step.ToPath, step.To))
	}
	return strings.Join(parts, "/")
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
