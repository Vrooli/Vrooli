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
	PolicyRuleID     string      `json:"policy_rule_id,omitempty"`
	PolicySeverity   string      `json:"policy_severity,omitempty"`
	PolicyStandard   string      `json:"policy_standard,omitempty"`
	PolicySource     string      `json:"policy_source,omitempty"`
	PolicyFindingID  string      `json:"policy_finding_id,omitempty"`
	PolicyObservedAt string      `json:"policy_observed_at,omitempty"`
}

type Calibration struct {
	HistoricalCases      int                `json:"historical_cases"`
	Basis                string             `json:"basis"`
	ScopeFrameID         string             `json:"scope_frame_id,omitempty"`
	ScopeCases           int                `json:"scope_cases,omitempty"`
	SamplingFrame        string             `json:"sampling_frame,omitempty"`
	SamplingSelection    string             `json:"sampling_selection,omitempty"`
	SamplingPopulation   string             `json:"sampling_population,omitempty"`
	SamplingUnit         string             `json:"sampling_unit,omitempty"`
	SamplingMethod       string             `json:"sampling_method,omitempty"`
	SamplingSeed         string             `json:"sampling_seed,omitempty"`
	AdjudicationProtocol string             `json:"adjudication_protocol,omitempty"`
	AdjudicationComplete bool               `json:"adjudication_complete"`
	RepresentativeSample bool               `json:"representative_sample"`
	RegressionFailures   int                `json:"regression_failures"`
	RegressionCases      int                `json:"regression_cases"`
	TargetNegativeCases  int                `json:"target_negative_cases"`
	TruePositive         int                `json:"true_positive"`
	FalsePositive        int                `json:"false_positive"`
	FalseNegative        int                `json:"false_negative"`
	TrueNegative         int                `json:"true_negative"`
	Precision            *float64           `json:"precision"`
	Recall               *float64           `json:"recall"`
	CorpusVersion        string             `json:"corpus_version"`
	EvidenceFrames       []CalibrationFrame `json:"evidence_frames,omitempty"`
}

// CalibrationFrame reports exact measurements for a declared bounded
// population. It does not replace the detector-wide rates unless the main
// corpus itself has a representative fleet sampling design.
type CalibrationFrame struct {
	FrameID              string              `json:"frame_id"`
	PopulationID         string              `json:"population_id"`
	Population           string              `json:"population"`
	CompleteCensus       bool                `json:"complete_census"`
	SamplingUnit         string              `json:"sampling_unit"`
	SelectionMethod      string              `json:"selection_method"`
	RandomSeed           string              `json:"random_seed"`
	AdjudicationProtocol string              `json:"adjudication_protocol"`
	AdjudicationComplete bool                `json:"adjudication_complete"`
	MeasurementMethod    string              `json:"measurement_method,omitempty"`
	ResponseComplete     bool                `json:"response_complete,omitempty"`
	GeneralizationScope  string              `json:"generalization_scope"`
	Targets              []CalibrationTarget `json:"targets"`
	Cases                int                 `json:"cases"`
	TruePositive         int                 `json:"true_positive"`
	FalsePositive        int                 `json:"false_positive"`
	FalseNegative        int                 `json:"false_negative"`
	TrueNegative         int                 `json:"true_negative"`
	Precision            *float64            `json:"precision"`
	Recall               *float64            `json:"recall"`
}

type CalibrationTarget struct {
	Scenario              string  `json:"scenario"`
	Applicability         string  `json:"applicability"`
	SourceHash            string  `json:"source_hash"`
	ConfigHash            string  `json:"config_hash"`
	SourceManifestSHA256  string  `json:"source_manifest_sha256"`
	Stratum               int     `json:"stratum,omitempty"`
	InclusionProbability  float64 `json:"inclusion_probability,omitempty"`
	SourceFileCount       int     `json:"source_file_count"`
	EligibleFunctions     int     `json:"eligible_functions"`
	SourceGroups          int     `json:"source_groups"`
	ServedGroups          int     `json:"served_groups"`
	ServedRecommendations int     `json:"served_recommendations"`
	ResponseComplete      bool    `json:"response_complete,omitempty"`
}

type Detector struct {
	ID          string      `json:"id"`
	Owner       string      `json:"owner"`
	Description string      `json:"description"`
	Input       string      `json:"input"`
	Calibration Calibration `json:"calibration"`
	detect      func([]symbol) []Opportunity
}

// AuditorFinding is the standards-policy evidence published by Scenario
// Auditor. These findings are recommendations to remediate a rule violation,
// not estimates of removable code, so they retain their policy metadata and
// remain uncalibrated for line-reduction ranking.
type AuditorFinding struct {
	ID             string `json:"id,omitempty"`
	RuleID         string `json:"rule_id"`
	Type           string `json:"type,omitempty"`
	Severity       string `json:"severity"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	FilePath       string `json:"file_path"`
	LineNumber     int    `json:"line_number"`
	Recommendation string `json:"recommendation"`
	Standard       string `json:"standard"`
	Source         string `json:"source,omitempty"`
	DiscoveredAt   string `json:"discovered_at,omitempty"`
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
	id, language, name, path, pkg, receiver, signature, bodyHash, typeExpression string
	start, end, bodyLines, nameOccurrences, complexity                           int
	exported                                                                     bool
}

// Registry has one authoritative catalog. New detectors enter through this
// constructor and the same labelled-corpus evaluation as existing detectors.
type Registry struct{ detectors []Detector }

func New() *Registry {
	registry := &Registry{detectors: []Detector{
		{ID: "standards.scenario-auditor", Owner: "scenario-auditor", Input: "scenario-auditor: cached standards violations", Description: "Scenario Auditor standards findings with original severity, standard, rule, source and remediation guidance"},
		{ID: "migration.unbound-cli-handler", Owner: "tidiness-manager", Input: "code-facts:go syntax symbols", Description: "Unreferenced legacy RunContext method with a registered-operation replacement", detect: unboundCLIHandlers},
		{ID: "duplication.identical-function-body", Owner: "tidiness-manager", Input: "code-facts:go function body identities", Description: "Repeated nontrivial Go production function bodies", detect: identicalBodies},
		{ID: "duplication.identical-function-body.typescript", Owner: "tidiness-manager", Input: "code-facts:typescript function body identities", Description: "Repeated nontrivial TypeScript production function bodies", detect: identicalTypeScriptBodies},
		{ID: "duplication.identical-function-body.javascript", Owner: "tidiness-manager", Input: "code-facts:javascript function body identities", Description: "Repeated nontrivial JavaScript production function bodies", detect: identicalJavaScriptBodies},
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
		if detector.detect == nil {
			continue // external evidence detectors are populated by their adapters
		}
		for _, opportunity := range detector.detect(symbols) {
			opportunity.DetectorID = detector.ID
			opportunity.ID = detector.ID + ":" + opportunity.Locations[0].Path + ":" + opportunity.Locations[0].Symbol
			opportunity.NetLines = opportunity.RemovableLines - opportunity.ReplacementLines
			opportunity.Calibration = calibrationForTarget(detector.Calibration, result.Scenario, result.SourceHash, result.ConfigHash)
			if opportunity.NetLines < minLines {
				continue
			}
			if opportunity.Calibration.Precision != nil {
				value := math.Round(*opportunity.Calibration.Precision*float64(opportunity.NetLines)*100) / 100
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

// calibrationForTarget applies a measured precision only when the current
// scenario, source snapshot and detector config exactly match a complete
// target census. It never promotes a target result into a fleet estimate.
func calibrationForTarget(calibration Calibration, scenario, sourceHash, configHash string) Calibration {
	if calibration.Precision != nil || scenario == "" || sourceHash == "" || configHash == "" {
		return calibration
	}
	var match *CalibrationFrame
	for i := range calibration.EvidenceFrames {
		frame := &calibration.EvidenceFrames[i]
		if !frame.CompleteCensus || !frame.AdjudicationComplete || !frame.ResponseComplete || frame.Precision == nil {
			continue
		}
		for _, target := range frame.Targets {
			if target.Scenario != scenario || target.SourceHash != sourceHash || target.ConfigHash != configHash || !target.ResponseComplete {
				continue
			}
			if match != nil && (match.Precision == nil || *match.Precision != *frame.Precision ||
				(match.Recall == nil) != (frame.Recall == nil) ||
				(match.Recall != nil && *match.Recall != *frame.Recall)) {
				return calibration
			}
			match = frame
		}
	}
	if match == nil {
		return calibration
	}
	result := calibration
	result.Precision = match.Precision
	result.Recall = match.Recall
	result.ScopeFrameID = match.FrameID
	result.ScopeCases = match.Cases
	result.Basis = "complete exact-source target census; no fleet generalization"
	result.SamplingFrame = match.FrameID
	result.SamplingPopulation = match.PopulationID
	result.SamplingUnit = match.SamplingUnit
	result.SamplingMethod = match.SelectionMethod
	result.SamplingSeed = match.RandomSeed
	result.AdjudicationProtocol = match.AdjudicationProtocol
	result.AdjudicationComplete = match.AdjudicationComplete
	result.RepresentativeSample = false
	return result
}

// RecommendWithAuditorFindings places standards findings in the same
// recommendation registry while preserving their policy meaning. Findings
// have no measured line reduction and are never assigned synthetic precision.
func (r *Registry) RecommendWithAuditorFindings(report *facts.CodeFactsReport, findings []AuditorFinding, limit, minLines int) Report {
	result := r.Recommend(report, 0, minLines)
	var auditorCalibration Calibration
	for _, detector := range r.detectors {
		if detector.ID == "standards.scenario-auditor" {
			auditorCalibration = detector.Calibration
			break
		}
	}
	for _, finding := range findings {
		if strings.TrimSpace(finding.RuleID) == "" || strings.TrimSpace(finding.FilePath) == "" {
			continue
		}
		path, ok := auditorLocationPath(report.GetTarget().GetRootPath(), result.Scenario, finding.FilePath)
		if !ok {
			result.Warnings = append(result.Warnings, "ignored Scenario Auditor finding outside target: "+finding.FilePath)
			continue
		}
		title := finding.Title
		if title == "" {
			title = finding.RuleID
		}
		reason := finding.Description
		if reason == "" {
			reason = "Scenario Auditor reported a standards-policy violation."
		}
		id := fmt.Sprintf("standards.scenario-auditor:%s:%s:%d", finding.RuleID, path, finding.LineNumber)
		result.Recommendations = append(result.Recommendations, Opportunity{
			ID: id, DetectorID: "standards.scenario-auditor", Title: title,
			Locations: []Location{{Path: path, StartLine: finding.LineNumber, EndLine: finding.LineNumber}},
			Reason:    reason, Action: finding.Recommendation,
			Calibration:  auditorCalibration,
			PolicyRuleID: finding.RuleID, PolicySeverity: finding.Severity, PolicyStandard: finding.Standard, PolicySource: finding.Source,
			PolicyFindingID: finding.ID, PolicyObservedAt: finding.DiscoveredAt,
		})
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

func auditorLocationPath(targetRoot, scenario, findingPath string) (string, bool) {
	path := filepath.Clean(findingPath)
	if filepath.IsAbs(path) {
		relative, err := filepath.Rel(targetRoot, path)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return "", false
		}
		path = relative
	} else {
		path = strings.TrimPrefix(filepath.ToSlash(path), "./")
		prefix := "scenarios/" + scenario + "/"
		path = strings.TrimPrefix(path, prefix)
	}
	path = filepath.ToSlash(path)
	if path == "" || path == "." || path == ".." || strings.HasPrefix(path, "../") || filepath.IsAbs(path) {
		return "", false
	}
	return path, true
}

func graphSymbols(report *facts.CodeFactsReport, includeTests bool) ([]symbol, []string) {
	var result []symbol
	var warnings []string
	seen := map[string]bool{}
	for _, envelope := range report.GetGraphs() {
		unit := envelope.GetParseUnit()
		if language := unit.GetLanguage(); language != "go" && language != "typescript" && language != "javascript" {
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
			if attrs["signature"] == "" && attrs["type_expression"] == "" && attrs["body_hash"] == "" {
				continue
			}
			path := filepath.ToSlash(filepath.Join(prefix, node.GetPath()))
			if filepath.IsAbs(node.GetPath()) || path == ".." || strings.HasPrefix(path, "../") {
				warnings = append(warnings, "symbol outside target: "+node.GetPath())
				continue
			}
			if (!includeTests && (attrs["is_test"] == "true" || strings.HasSuffix(path, "_test.go"))) || strings.HasSuffix(path, ".pb.go") || strings.Contains("/"+path, "/generated/") || strings.Contains("/"+path, "/vendor/") {
				continue
			}
			number := func(key string) int { n, _ := strconv.Atoi(attrs[key]); return n }
			key := path + ":" + attrs["start_line"] + ":" + node.GetName()
			if number("start_line") <= 0 || number("end_line") < number("start_line") || seen[key] {
				continue
			}
			seen[key] = true
			language := attrs["language"]
			if language == "" {
				language = unit.GetLanguage()
			}
			result = append(result, symbol{id: unit.GetId() + ":" + node.GetId(), language: language, name: node.GetName(), path: path, pkg: unit.GetId() + ":" + attrs["package_id"], receiver: attrs["receiver"], signature: attrs["signature"], bodyHash: attrs["body_hash"], typeExpression: attrs["type_expression"], start: number("start_line"), end: number("end_line"), bodyLines: number("body_lines"), nameOccurrences: number("package_name_occurrences"), complexity: number("cyclomatic_complexity"), exported: attrs["exported"] == "true"})
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
	return identicalBodiesForLanguage(symbols, "go")
}

func identicalTypeScriptBodies(symbols []symbol) []Opportunity {
	return identicalBodiesForLanguage(symbols, "typescript")
}

func identicalJavaScriptBodies(symbols []symbol) []Opportunity {
	return identicalBodiesForLanguage(symbols, "javascript")
}

func identicalBodiesForLanguage(symbols []symbol, language string) []Opportunity {
	var result []Opportunity
	for _, members := range identicalBodyGroups(symbols, language) {
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

func identicalBodyGroups(symbols []symbol, language string) [][]symbol {
	byHash := map[string][]symbol{}
	for _, s := range symbols {
		if s.bodyHash != "" && s.bodyLines >= 5 && s.start > 0 && (language == "" || s.language == language) {
			byHash[s.language+":"+s.bodyHash] = append(byHash[s.language+":"+s.bodyHash], s)
		}
	}
	groups := make([][]symbol, 0, len(byHash))
	for _, members := range byHash {
		if len(members) < 2 {
			continue
		}
		sort.Slice(members, func(i, j int) bool {
			if members[i].path != members[j].path {
				return members[i].path < members[j].path
			}
			return members[i].start < members[j].start
		})
		groups = append(groups, members)
	}
	sort.Slice(groups, func(i, j int) bool {
		if groups[i][0].path != groups[j][0].path {
			return groups[i][0].path < groups[j][0].path
		}
		return groups[i][0].start < groups[j][0].start
	})
	return groups
}

// DuplicateFunctionBodies projects exact body groups from Code Facts into the
// scanner's legacy duplicate-block response shape. Parsing and clone identity
// remain owned by the shared graph providers.
type DuplicateBodyGroup struct {
	Locations []Location
	Lines     int
}

func DuplicateFunctionBodies(report *facts.CodeFactsReport, minLines int) ([]DuplicateBodyGroup, []string) {
	return DuplicateFunctionBodiesForLanguage(report, minLines, "")
}

func DuplicateFunctionBodiesForLanguage(report *facts.CodeFactsReport, minLines int, language string) ([]DuplicateBodyGroup, []string) {
	symbols, warnings := graphSymbols(report, false)
	var result []DuplicateBodyGroup
	for _, members := range identicalBodyGroups(symbols, language) {
		if members[0].bodyLines < minLines {
			continue
		}
		group := DuplicateBodyGroup{Lines: members[0].bodyLines}
		for _, member := range members {
			group.Locations = append(group.Locations, location(member))
		}
		result = append(result, group)
	}
	return result, warnings
}
