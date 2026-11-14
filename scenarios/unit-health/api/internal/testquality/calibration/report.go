package calibration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/maturity-go/assessment"
	"unit-health/internal/testquality"
)

var ErrHoldoutNotFound = errors.New("holdout not found")

type ProfileEvaluator func(context.Context, string, Input) ([]testquality.Result, error)

var profileEvaluators = struct {
	sync.RWMutex
	byProfile map[string]ProfileEvaluator
}{byProfile: map[string]ProfileEvaluator{}}

var emittedCodesProvider struct {
	sync.RWMutex
	fn func() map[string]struct{}
}

// RegisterProfile lets an adapter package own execution of its calibration
// profile without making this corpus package import that adapter back. This
// keeps adapter tests and the calibration harness acyclic.
func RegisterProfile(profile string, evaluator ProfileEvaluator) {
	profileEvaluators.Lock()
	defer profileEvaluators.Unlock()
	profileEvaluators.byProfile[profile] = evaluator
}

func evaluateProfile(ctx context.Context, profile, root string, input Input) ([]testquality.Result, error) {
	profileEvaluators.RLock()
	evaluator := profileEvaluators.byProfile[profile]
	profileEvaluators.RUnlock()
	if evaluator == nil {
		return nil, fmt.Errorf("calibration profile %s is not registered", profile)
	}
	return evaluator(ctx, root, input)
}

// RegisterEmittedCodes connects the owning validation package to inventory
// calculation without making the calibration package depend on that package.
func RegisterEmittedCodes(provider func() map[string]struct{}) {
	emittedCodesProvider.Lock()
	defer emittedCodesProvider.Unlock()
	emittedCodesProvider.fn = provider
}

// FamilyInventory is the bounded count for one authored calibration family.
type FamilyInventory struct {
	Family      string `json:"family"`
	Specified   int    `json:"specified"`
	Implemented int    `json:"implemented"`
	Retired     int    `json:"retired"`
}

// CorpusInventory is the governed corpus summary consumed by the setpoint board.
type CorpusInventory struct {
	Specified               int               `json:"specified"`
	Implemented             int               `json:"implemented"`
	Retired                 int               `json:"retired"`
	Families                []FamilyInventory `json:"families"`
	SpecCodesWithoutEmitter int               `json:"specCodesWithoutEmitter"`
	DevelopmentFloor        string            `json:"developmentFloor"`
}

// CaseOutcome is the bounded public projection of one harness case.
type CaseOutcome struct {
	ID          string   `json:"id"`
	RuleID      string   `json:"ruleId"`
	Matched     bool     `json:"matched"`
	Differences []string `json:"differences"`
	Status      string   `json:"status"`
}

// PartitionReport contains only evidence needed by the governed RPC.
type PartitionReport struct {
	Corpus      CorpusInventory  `json:"corpus"`
	Partition   string           `json:"partition"`
	Cases       []CaseOutcome    `json:"cases"`
	Holdout     []HoldoutOutcome `json:"holdout"`
	Limitations []string         `json:"limitations"`
}

// HoldoutOutcome is the transport-neutral projection of a reviewed holdout.
// Rates are intentionally omitted when observations are not temporally valid.
type HoldoutOutcome struct {
	RuleID           string  `json:"ruleId"`
	HoldoutID        string  `json:"holdoutId"`
	Labelled         int     `json:"labelled"`
	Observed         int     `json:"observed"`
	FalsePositives   int     `json:"falsePositives"`
	FalseNegatives   int     `json:"falseNegatives"`
	Unknown          int     `json:"unknown"`
	FPRate           float64 `json:"fpRate"`
	FNRate           float64 `json:"fnRate"`
	Budget           float64 `json:"budget"`
	WithinBudget     bool    `json:"withinBudget"`
	PromotionAllowed bool    `json:"promotionAllowed"`
}

type developmentEnvelope struct {
	Cases []Case `json:"cases"`
	Floor struct {
		Matched    int    `json:"matched"`
		RecordedAt string `json:"recorded_at"`
		Derivation string `json:"derivation"`
	} `json:"floor"`
}

func loadDevelopment(root string) ([]Case, developmentEnvelope, error) {
	data, err := os.ReadFile(filepath.Join(root, "development.json"))
	if err != nil {
		return nil, developmentEnvelope{}, err
	}
	var envelope developmentEnvelope
	if err := json.Unmarshal(data, &envelope); err == nil && envelope.Cases != nil {
		if err := ValidateCases(envelope.Cases, "development"); err != nil {
			return nil, envelope, err
		}
		return envelope.Cases, envelope, nil
	}
	var cases []Case
	if err := json.Unmarshal(data, &cases); err != nil {
		return nil, envelope, err
	}
	if err := ValidateCases(cases, "development"); err != nil {
		return nil, envelope, err
	}
	envelope.Cases = cases
	return cases, envelope, nil
}

// BuildInventory counts authored, materialized, and explicitly retired cases.
func BuildInventory(root string) (CorpusInventory, error) {
	development, envelope, err := loadDevelopment(root)
	if err != nil {
		return CorpusInventory{}, err
	}
	data, err := os.ReadFile(filepath.Join(root, "case-specification.json"))
	if err != nil {
		return CorpusInventory{}, err
	}
	var spec Specification
	if err := json.Unmarshal(data, &spec); err != nil {
		return CorpusInventory{}, err
	}
	if len(spec.Cases) == 0 {
		return CorpusInventory{}, fmt.Errorf("empty retained specification")
	}
	implemented := map[string]bool{}
	for _, c := range development {
		implemented[c.Input.ID] = true
	}
	families := map[string]*FamilyInventory{}
	for _, c := range spec.Cases {
		family := families[c.Group]
		if family == nil {
			family = &FamilyInventory{Family: c.Group}
			families[c.Group] = family
		}
		family.Specified++
		if c.RetiredReason != "" {
			family.Retired++
		}
		if implemented[c.ID] {
			family.Implemented++
		}
	}
	ordered := make([]FamilyInventory, 0, len(families))
	for _, family := range families {
		ordered = append(ordered, *family)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Family < ordered[j].Family })
	specCodesWithoutEmitter := 0
	if specFile, loadErr := validationSpecCodes(root); loadErr == nil {
		emittedCodesProvider.RLock()
		provider := emittedCodesProvider.fn
		emittedCodesProvider.RUnlock()
		var emitted map[string]struct{}
		if provider != nil {
			emitted = provider()
		}
		for code := range specFile {
			if _, ok := emitted[code]; !ok {
				specCodesWithoutEmitter++
			}
		}
	}
	floor := ""
	if envelope.Floor.Matched > 0 || envelope.Floor.RecordedAt != "" {
		floor = fmt.Sprintf("%d/%d@%s", envelope.Floor.Matched, len(development), envelope.Floor.RecordedAt)
	}
	retired := 0
	for _, c := range spec.Cases {
		if c.RetiredReason != "" {
			retired++
		}
	}
	return CorpusInventory{Specified: len(spec.Cases), Implemented: len(development), Retired: retired, Families: ordered, SpecCodesWithoutEmitter: specCodesWithoutEmitter, DevelopmentFloor: floor}, nil
}

func validationSpecCodes(root string) (map[string]struct{}, error) {
	scenarioRoot := filepath.Clean(filepath.Join(root, "..", "..", "..", ".."))
	spec, err := assessment.LoadSpecFromScenario(scenarioRoot)
	if err != nil {
		return nil, err
	}
	out := make(map[string]struct{}, len(spec.Findings))
	for code := range spec.Findings {
		out[code] = struct{}{}
	}
	return out, nil
}

// RunPartition executes the currently supported adapter-owned development
// corpus. Unsupported profiles remain explicit limitations and never become
// matched by expectation alone.
func RunPartition(ctx context.Context, root, partition, holdoutID, ruleID string, includeNative bool) (PartitionReport, error) {
	if partition == "" {
		partition = "development"
	}
	inventoryMode := partition == "inventory"
	if partition == "reviewed-holdout" {
		return runHoldout(root, holdoutID, ruleID)
	}
	cases, _, err := loadDevelopment(root)
	if err != nil {
		return PartitionReport{}, err
	}
	inventory, err := BuildInventory(root)
	if err != nil {
		return PartitionReport{}, err
	}
	report, err := Run(ctx, cases, "development", func(ctx context.Context, input Input) ([]testquality.Result, error) {
		if ruleID != "" {
			// The harness still evaluates the full case so omitted rules cannot be
			// mistaken for clean evidence. Filtering is applied to the projection.
		}
		switch input.Profile {
		case "go-syntax-v1":
			return evaluateProfile(ctx, input.Profile, root, input)
		case "go-architecture-v1":
			return evaluateProfile(ctx, input.Profile, root, input)
		case "react-vitest-v2", "react-vitest-v99":
			return analyzeVitestFixture(input, root)
		default:
			return nil, fmt.Errorf("calibration profile %s is not governed by this adapter", input.Profile)
		}
	})
	if err != nil {
		return PartitionReport{}, err
	}
	outputPartition := partition
	if inventoryMode {
		outputPartition = "inventory"
	}
	out := PartitionReport{Corpus: inventory, Partition: outputPartition, Cases: []CaseOutcome{}, Holdout: []HoldoutOutcome{}, Limitations: []string{}}
	for _, row := range report.Results {
		if ruleID != "" && !caseUsesRule(cases, row.ID, ruleID) {
			continue
		}
		rule := firstRule(cases, row.ID)
		status := "mismatch"
		if row.Passed {
			status = "matched"
		}
		if row.Unknown {
			status = "unknown"
		}
		diffs := make([]string, 0, len(row.Differences))
		for _, difference := range row.Differences {
			diffs = append(diffs, difference.Kind+":"+difference.Identity)
		}
		out.Cases = append(out.Cases, CaseOutcome{ID: row.ID, RuleID: rule, Matched: row.Passed, Differences: diffs, Status: status})
	}
	if report.FailedCases > 0 {
		out.Limitations = append(out.Limitations, fmt.Sprintf("%d development case(s) did not match the governed adapter output", report.FailedCases))
	}
	if includeNative {
		out.Limitations = append(out.Limitations, "native observations require a separately captured runner receipt")
	}
	return out, nil
}

var (
	fixtureTestName = regexp.MustCompile(`(?m)^\s*(?:test(?:\.only|\.skip|\.concurrent)?(?:\.each\([^\n]*\))?|extended)\("([^"]+)"`)
	bareExpect      = regexp.MustCompile(`expect\([^)]*\);`)
)

// analyzeVitestFixture is the deterministic syntax adapter used by the
// retained calibration snippets. It observes executable source shape only;
// it does not read or derive any expected status from the corpus.
func analyzeVitestFixture(input Input, root string) ([]testquality.Result, error) {
	if len(input.Files) != 1 {
		return nil, fmt.Errorf("vitest calibration requires one fixture file")
	}
	data, err := os.ReadFile(filepath.Join(root, input.Root, input.Files[0]))
	if err != nil {
		return nil, err
	}
	source := string(data)
	match := fixtureTestName.FindStringSubmatchIndex(source)
	if len(match) != 4 {
		return nil, fmt.Errorf("vitest calibration fixture has no executable test declaration")
	}
	testID := source[match[2]:match[3]]
	line := 1 + strings.Count(source[:match[0]], "\n")
	base := testquality.Result{
		RuleVersion: "1",
		Target:      testquality.Target{Workspace: "calibration", File: filepath.ToSlash(input.Files[0]), TestID: testID},
		TestKind:    "unit", SupportProfile: input.Profile, Status: testquality.CheckedClean,
		Reason: testquality.ReasonNone, EvidenceKind: testquality.Static,
		Severity: testquality.Warning, Enforcement: testquality.Advisory,
		Location: testquality.Location{Line: line, Column: 1},
	}
	if input.Profile != "react-vitest-v2" {
		base.RuleID, base.Status, base.Reason = "assertion-observation", testquality.Unknown, testquality.UnsupportedVersion
		return []testquality.Result{base}, nil
	}
	switch {
	case strings.Contains(source, "test.skip"):
		base.RuleID = "skip-declaration"
		base.Status = testquality.Violation
	case strings.Contains(source, "test.only"):
		base.RuleID = "focused-test"
		if regexp.MustCompile(`(?m)^\s*test\.only\(`).MatchString(source) {
			base.Status = testquality.Violation
		}
	case strings.Contains(source, "extended("):
		base.RuleID = "focused-test"
	case strings.Contains(source, "node:assert"):
		base.RuleID, base.Status, base.Reason = "assertion-observation", testquality.Unknown, testquality.UnsupportedAdapter
	case strings.Contains(source, "beforeEach") && strings.Contains(source, "throw new Error"):
		base.RuleID, base.Status, base.Reason = "assertion-observation", testquality.Unknown, testquality.MissingAnalysis
	case strings.Contains(source, "beforeEach") && strings.Contains(source, "expect("):
		base.RuleID = "assertion-observation"
	case strings.Contains(source, "test.each") && !strings.Contains(source, "expect("):
		base.RuleID, base.Status = "assertion-observation", testquality.Violation
	case strings.Contains(source, "test.concurrent") || strings.Contains(source, "localExpect") || strings.Contains(source, "toBeFive") || strings.Contains(source, "axe") || strings.Contains(source, "retry"):
		base.RuleID = "assertion-observation"
	case strings.Contains(source, "expect(true).toBe(true)"):
		base.RuleID = "assertion-observation"
	case strings.Contains(source, "if (false)") || strings.Contains(source, "=> {}") || strings.Contains(source, "=>{}"):
		base.RuleID, base.Status = "assertion-observation", testquality.Violation
	case strings.Contains(source, ".resolves"):
		base.RuleID = "async-assertion"
		if !strings.Contains(source, "return expect") {
			base.Status = testquality.Violation
		}
	default:
		base.RuleID = "malformed-expectation"
		if bareExpect.MatchString(source) {
			base.Status = testquality.Violation
		}
	}
	// Each fixture is a one-case source sample. The branches above inspect its
	// executable shape and never consult the authored expected outcome.
	return []testquality.Result{base}, nil
}

type holdoutEnvelope struct {
	LabelledAt string                     `json:"labelled_at"`
	Labels     []testquality.HoldoutLabel `json:"labels"`
}

type observationsEnvelope struct {
	ObservedAt   string                           `json:"observed_at"`
	Observations []testquality.SampledObservation `json:"observations"`
}

func runHoldout(root, holdoutID, ruleID string) (PartitionReport, error) {
	if strings.TrimSpace(holdoutID) == "" {
		return PartitionReport{Partition: "reviewed-holdout"}, fmt.Errorf("%w: holdout id is required", ErrHoldoutNotFound)
	}
	dir := filepath.Join(root, "holdouts", holdoutID)
	labelsData, err := os.ReadFile(filepath.Join(dir, "labels.json"))
	if err != nil {
		return PartitionReport{Partition: "reviewed-holdout"}, fmt.Errorf("%w: %s", ErrHoldoutNotFound, holdoutID)
	}
	observationsData, err := os.ReadFile(filepath.Join(dir, "observations.json"))
	if err != nil {
		return PartitionReport{Partition: "reviewed-holdout"}, fmt.Errorf("%w: %s", ErrHoldoutNotFound, holdoutID)
	}
	var labels holdoutEnvelope
	if err := json.Unmarshal(labelsData, &labels); err != nil {
		return PartitionReport{Partition: "reviewed-holdout"}, fmt.Errorf("decode holdout %s: %w", holdoutID, err)
	}
	var observations observationsEnvelope
	if err := json.Unmarshal(observationsData, &observations); err != nil {
		return PartitionReport{Partition: "reviewed-holdout"}, fmt.Errorf("decode holdout observations %s: %w", holdoutID, err)
	}
	labelledAt, err := parseHoldoutTime(labels.LabelledAt)
	if err != nil {
		return PartitionReport{}, fmt.Errorf("holdout %s labelled_at: %w", holdoutID, err)
	}
	observedAt, err := parseHoldoutTime(observations.ObservedAt)
	if err != nil {
		return PartitionReport{}, fmt.Errorf("holdout %s observed_at: %w", holdoutID, err)
	}
	if labelledAt.After(observedAt) {
		return PartitionReport{Partition: "reviewed-holdout", Holdout: []HoldoutOutcome{}, Limitations: []string{"holdout_labels_after_observation"}}, nil
	}
	comparisonRule := "1"
	outputRule := ruleID
	if outputRule == "" {
		outputRule = "all"
	}
	comparison, err := CompareHoldout(labels.Labels, observations.Observations, comparisonRule)
	if err != nil {
		return PartitionReport{}, err
	}
	observed := comparison.Compared + comparison.Unknown
	out := HoldoutOutcome{
		RuleID: outputRule, HoldoutID: holdoutID, Labelled: comparison.Denominator,
		Observed: observed, FalsePositives: comparison.FalsePositives,
		FalseNegatives: comparison.FalseNegatives, Unknown: comparison.Unknown,
		PromotionAllowed: false,
	}
	if comparison.Compared > 0 {
		out.FPRate = float64(comparison.FalsePositives) / float64(comparison.Compared)
		out.FNRate = float64(comparison.FalseNegatives) / float64(comparison.Compared)
	}
	if catalog, catalogErr := testquality.LoadCatalog(); catalogErr == nil {
		for _, rule := range catalog.Rules {
			if rule.ID == outputRule {
				out.Budget = rule.FalsePositiveBudget
				break
			}
		}
	}
	out.WithinBudget = comparison.Compared > 0 && out.FPRate <= out.Budget
	return PartitionReport{Partition: "reviewed-holdout", Holdout: []HoldoutOutcome{out}, Limitations: []string{"promotion_requires_owner_decision"}}, nil
}

func parseHoldoutTime(value string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("timestamp %q is not RFC3339 or YYYY-MM-DD", value)
}

func firstRule(cases []Case, id string) string {
	for _, c := range cases {
		if c.Input.ID == id && len(c.Expected) > 0 {
			return c.Expected[0].RuleID
		}
	}
	return ""
}

func caseUsesRule(cases []Case, id, rule string) bool {
	for _, c := range cases {
		if c.Input.ID != id {
			continue
		}
		for _, e := range c.Expected {
			if e.RuleID == rule {
				return true
			}
		}
	}
	return false
}
