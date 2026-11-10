// Package discovery owns reduction detectors and their single ranking. Source
// discovery, parsing and graph freshness belong to Code Facts.
package discovery

import (
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	facts "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

type Location struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Symbol    string `json:"symbol"`
}

type Opportunity struct {
	ID               string      `json:"id"`
	DetectorID       string      `json:"detector_id"`
	Title            string      `json:"title"`
	Locations        []Location  `json:"locations"`
	RemovableLines   int         `json:"removable_lines"`
	ReplacementLines int         `json:"replacement_lines"`
	NetLines         int         `json:"net_lines"`
	ExpectedNetLines *float64    `json:"expected_net_lines"`
	Calibration      Calibration `json:"calibration"`
	Reason           string      `json:"reason"`
	Action           string      `json:"action"`
}

type Calibration struct {
	HistoricalCases    int      `json:"historical_cases"`
	Basis              string   `json:"basis"`
	RegressionFailures int      `json:"regression_failures"`
	RegressionCases    int      `json:"regression_cases"`
	TruePositive       int      `json:"true_positive"`
	FalsePositive      int      `json:"false_positive"`
	FalseNegative      int      `json:"false_negative"`
	TrueNegative       int      `json:"true_negative"`
	Precision          *float64 `json:"precision"`
	Recall             *float64 `json:"recall"`
	CorpusVersion      string   `json:"corpus_version"`
}

type Detector struct {
	ID          string      `json:"id"`
	Owner       string      `json:"owner"`
	Description string      `json:"description"`
	Input       string      `json:"input"`
	Calibration Calibration `json:"calibration"`
	detect      func([]symbol) []Opportunity
}

type Report struct {
	Scenario        string        `json:"scenario"`
	SourceHash      string        `json:"source_hash"`
	ConfigHash      string        `json:"config_hash"`
	Recommendations []Opportunity `json:"recommendations"`
	Detectors       []Detector    `json:"detectors"`
	Warnings        []string      `json:"warnings,omitempty"`
	Count           int           `json:"count"`
}

type symbol struct {
	id, name, path, pkg, receiver, signature, bodyHash, typeExpression string
	start, end, bodyLines, nameOccurrences, complexity                 int
	exported                                                           bool
}

// Registry has one authoritative catalog. New detectors enter through this
// constructor and the same labelled-corpus evaluation as existing detectors.
type Registry struct{ detectors []Detector }

func New() *Registry {
	registry := &Registry{detectors: []Detector{
		{ID: "migration.unbound-cli-handler", Owner: "tidiness-manager", Input: "code-facts:go syntax symbols", Description: "Unreferenced legacy RunContext method with a registered-operation replacement", detect: unboundCLIHandlers},
		{ID: "duplication.identical-function-body", Owner: "tidiness-manager", Input: "code-facts:go function body identities", Description: "Repeated nontrivial production function bodies", detect: identicalBodies},
	}}
	registry.calibrate()
	return registry
}

func (r *Registry) Describe() []Detector { return append([]Detector(nil), r.detectors...) }

func (r *Registry) Recommend(report *facts.CodeFactsReport, limit, minLines int) Report {
	result := Report{Scenario: report.GetTarget().GetRequested().GetScenario(), SourceHash: report.GetCache().GetSourceHash(), ConfigHash: report.GetCache().GetConfigHash(), Detectors: r.Describe(), Recommendations: []Opportunity{}}
	for _, warning := range report.GetWarnings() {
		result.Warnings = append(result.Warnings, warning.GetCode()+": "+warning.GetMessage())
	}
	symbols, warnings := graphSymbols(report, false)
	result.Warnings = append(result.Warnings, warnings...)
	for _, detector := range r.detectors {
		for _, opportunity := range detector.detect(symbols) {
			opportunity.DetectorID = detector.ID
			opportunity.ID = detector.ID + ":" + opportunity.Locations[0].Path + ":" + opportunity.Locations[0].Symbol
			opportunity.NetLines = opportunity.RemovableLines - opportunity.ReplacementLines
			opportunity.Calibration = detector.Calibration
			if opportunity.NetLines < minLines {
				continue
			}
			if detector.Calibration.Precision != nil {
				value := math.Round(*detector.Calibration.Precision*float64(opportunity.NetLines)*100) / 100
				opportunity.ExpectedNetLines = &value
			}
			result.Recommendations = append(result.Recommendations, opportunity)
		}
	}
	sort.Slice(result.Recommendations, func(i, j int) bool {
		a, b := result.Recommendations[i], result.Recommendations[j]
		if (a.ExpectedNetLines == nil) != (b.ExpectedNetLines == nil) {
			return a.ExpectedNetLines != nil
		}
		if a.ExpectedNetLines != nil && *a.ExpectedNetLines != *b.ExpectedNetLines {
			return *a.ExpectedNetLines > *b.ExpectedNetLines
		}
		if a.NetLines != b.NetLines {
			return a.NetLines > b.NetLines
		}
		return a.ID < b.ID
	})
	if limit > 0 && len(result.Recommendations) > limit {
		result.Recommendations = result.Recommendations[:limit]
	}
	result.Count = len(result.Recommendations)
	return result
}

func graphSymbols(report *facts.CodeFactsReport, includeTests bool) ([]symbol, []string) {
	var result []symbol
	var warnings []string
	seen := map[string]bool{}
	for _, envelope := range report.GetGraphs() {
		unit := envelope.GetParseUnit()
		if unit.GetLanguage() != "go" {
			warnings = append(warnings, "syntax reduction detectors do not yet cover "+unit.GetLanguage())
			continue
		}
		prefix, err := filepath.Rel(report.GetTarget().GetRootPath(), unit.GetRootPath())
		if err != nil || prefix == ".." || strings.HasPrefix(prefix, "../") {
			warnings = append(warnings, "parse unit outside target: "+unit.GetRootPath())
			continue
		}
		for _, node := range envelope.GetGraph().GetNodes() {
			attrs := node.GetAttributes()
			if attrs["signature"] == "" && attrs["type_expression"] == "" {
				continue
			}
			path := filepath.ToSlash(filepath.Join(prefix, node.GetPath()))
			if filepath.IsAbs(node.GetPath()) || path == ".." || strings.HasPrefix(path, "../") {
				warnings = append(warnings, "symbol outside target: "+node.GetPath())
				continue
			}
			if (!includeTests && strings.HasSuffix(path, "_test.go")) || strings.HasSuffix(path, ".pb.go") || strings.Contains("/"+path, "/generated/") || strings.Contains("/"+path, "/vendor/") {
				continue
			}
			number := func(key string) int { n, _ := strconv.Atoi(attrs[key]); return n }
			key := path + ":" + attrs["start_line"] + ":" + node.GetName()
			if number("start_line") <= 0 || number("end_line") < number("start_line") || seen[key] {
				continue
			}
			seen[key] = true
			result = append(result, symbol{id: unit.GetId() + ":" + node.GetId(), name: node.GetName(), path: path, pkg: unit.GetId() + ":" + attrs["package_id"], receiver: attrs["receiver"], signature: attrs["signature"], bodyHash: attrs["body_hash"], typeExpression: attrs["type_expression"], start: number("start_line"), end: number("end_line"), bodyLines: number("body_lines"), nameOccurrences: number("package_name_occurrences"), complexity: number("cyclomatic_complexity"), exported: attrs["exported"] == "true"})
		}
	}
	return result, warnings
}

func location(s symbol) Location {
	return Location{Path: s.path, StartLine: s.start, EndLine: s.end, Symbol: strings.TrimPrefix(s.receiver+"."+s.name, ".")}
}

func unboundCLIHandlers(symbols []symbol) []Opportunity {
	replacements := map[string]bool{}
	for _, s := range symbols {
		if strings.Contains(s.signature, "cliapp.OperationContext") {
			replacements[s.pkg+":"+s.receiver+":"+s.name] = true
		}
	}
	var result []Opportunity
	for _, s := range symbols {
		if s.exported || s.receiver == "" || s.nameOccurrences != 1 || !strings.Contains(s.signature, "cliapp.RunContext") || !replacements[s.pkg+":"+s.receiver+":"+s.name+"Call"] {
			continue
		}
		result = append(result, Opportunity{Title: "Delete unbound CLI handler " + s.name, Locations: []Location{location(s)}, RemovableLines: s.end - s.start + 1, Reason: "The private method name occurs only at its declaration; its OperationContext replacement exists in the same receiver package.", Action: "Verify the primitive binding, remove the old wrapper, migrate any tests of the old path and run the CLI suite."})
	}
	return result
}

func identicalBodies(symbols []symbol) []Opportunity {
	groups := map[string][]symbol{}
	for _, s := range symbols {
		if s.bodyHash != "" && s.bodyLines >= 5 && s.start > 0 {
			groups[s.bodyHash] = append(groups[s.bodyHash], s)
		}
	}
	var result []Opportunity
	for _, members := range groups {
		if len(members) < 2 {
			continue
		}
		sort.Slice(members, func(i, j int) bool {
			if members[i].path != members[j].path {
				return members[i].path < members[j].path
			}
			return members[i].start < members[j].start
		})
		opportunity := Opportunity{Title: fmt.Sprintf("Consolidate %d identical function bodies", len(members)), Reason: "The shared parser emitted the same syntax body hash for these production functions. Common ownership and behavioral equivalence still require review.", Action: "Choose the common owner, migrate callers and delete duplicate bodies; retain domain-specific wrappers only where required.", ReplacementLines: 3 * (len(members) - 1)}
		for i, s := range members {
			opportunity.Locations = append(opportunity.Locations, location(s))
			if i > 0 {
				opportunity.RemovableLines += s.bodyLines
			}
		}
		result = append(result, opportunity)
	}
	return result
}
