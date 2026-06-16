package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"quality-health/internal/autofix"
	"quality-health/internal/commands"
	"quality-health/internal/contracts"
	"quality-health/internal/rules"
	"quality-health/internal/surfaces"
)

type Service struct {
	Discoverer surfaces.Discoverer
	Locator    surfaces.Locator
	Executor   commands.Executor
	Now        func() time.Time
}

type Request struct {
	Scenario                string
	Path                    string
	RuleIDs                 []string
	Surfaces                []string
	IncludeCommandExecution bool
	IncludeAutofixPreview   bool
	UseCache                bool
}

type Response struct {
	RunID             string
	Status            string
	Summary           string
	Inventory         surfaces.Inventory
	Contracts         []ContractEvaluation
	Findings          []Finding
	CommandResults    []commands.Result
	Maturity          Maturity
	NextSteps         []string
	AutofixCandidates []autofix.Candidate
}

type ContractEvaluation struct {
	ContractID string
	SurfaceID  string
	Status     string
	RuleIDs    []string
}

type Finding struct {
	ID               string
	Scenario         string
	TargetKind       string
	SurfaceID        string
	SurfaceKind      string
	Language         string
	Framework        string
	RuleID           string
	Category         string
	Severity         string
	FilePath         string
	Symbol           string
	Message          string
	Evidence         string
	Expected         string
	Observed         string
	WhyItMatters     string
	Remediation      string
	FixClass         string
	AutofixAvailable bool
	AutofixCommand   string
	SourceCommand    string
	CreatedAt        string
}

type Maturity struct {
	Rung      int
	Label     string
	Rationale string
}

func New(disc surfaces.Discoverer) *Service {
	return &Service{Discoverer: disc}
}

func (s *Service) Audit(ctx context.Context, req Request) (Response, error) {
	now := s.now()
	disc := s.Discoverer
	if disc == nil {
		disc = surfaces.CodeFactsClient{Locator: s.Locator}
	}
	inv, err := disc.Discover(ctx, req.Scenario, req.Path, req.UseCache)
	if err != nil {
		return Response{}, err
	}
	filtered := filterSurfaces(inv.Surfaces, req.Surfaces)
	inv.Surfaces = filtered
	res := Response{
		RunID:     "qh-" + now.UTC().Format("20060102-150405"),
		Inventory: inv,
	}
	var uncoveredSurfaces []string
	for _, surface := range filtered {
		allSurfaceRules := rules.SurfaceRules(surface)
		if len(allSurfaceRules) == 0 {
			res.Findings = append(res.Findings, coverageGapFinding(inv, surface, now))
			res.Contracts = append(res.Contracts, ContractEvaluation{
				ContractID: "",
				SurfaceID:  surface.ID,
				Status:     "uncovered",
				RuleIDs:    nil,
			})
			uncoveredSurfaces = append(uncoveredSurfaces, surface.ID)
			continue
		}
		surfaceRules := rules.Filter(allSurfaceRules, req.RuleIDs)
		before := len(res.Findings)
		res.Findings = append(res.Findings, s.evaluateRules(inv, surface, surfaceRules, now)...)
		res.Contracts = append(res.Contracts, ContractEvaluation{
			ContractID: contractIDForRules(allSurfaceRules),
			SurfaceID:  surface.ID,
			Status:     statusFromFindings(res.Findings[before:]),
			RuleIDs:    ruleIDs(allSurfaceRules),
		})
	}
	allScenarioRules := rules.ScenarioRules()
	scenarioRules := rules.Filter(allScenarioRules, req.RuleIDs)
	beforeScenario := len(res.Findings)
	res.Findings = append(res.Findings, s.evaluateRules(inv, surfaces.Surface{ID: "scenario", Kind: "scenario"}, scenarioRules, now)...)
	res.Contracts = append(res.Contracts, ContractEvaluation{
		ContractID: "scenario-quality-gates",
		SurfaceID:  "scenario",
		Status:     statusFromFindings(res.Findings[beforeScenario:]),
		RuleIDs:    ruleIDs(allScenarioRules),
	})
	if req.IncludeCommandExecution {
		res.CommandResults = commands.RunAll(ctx, s.Executor, inv)
	}
	if req.IncludeAutofixPreview {
		candidates, err := autofix.Preview(inv.RootPath, req.RuleIDs)
		if err == nil {
			res.AutofixCandidates = candidates
		}
	}
	sortFindings(res.Findings)
	res.Maturity = maturity(inv, res.Findings, res.CommandResults, uncoveredSurfaces)
	res.Status = auditStatus(inv, res.Findings)
	res.Summary = summary(res)
	res.NextSteps = nextSteps(res)
	return res, nil
}

func (s *Service) PreviewFix(ctx context.Context, scenario, path string, ruleIDs []string) (surfaces.Inventory, []autofix.Candidate, error) {
	inv, err := s.inventory(ctx, scenario, path)
	if err != nil {
		return surfaces.Inventory{}, nil, err
	}
	candidates, err := autofix.Preview(inv.RootPath, ruleIDs)
	return inv, candidates, err
}

func (s *Service) ApplyFix(ctx context.Context, scenario, path string, ruleIDs []string) (surfaces.Inventory, []autofix.Candidate, error) {
	inv, err := s.inventory(ctx, scenario, path)
	if err != nil {
		return surfaces.Inventory{}, nil, err
	}
	candidates, err := autofix.Apply(inv.RootPath, ruleIDs)
	return inv, candidates, err
}

func (s *Service) inventory(ctx context.Context, scenario, path string) (surfaces.Inventory, error) {
	disc := s.Discoverer
	if disc == nil {
		disc = surfaces.CodeFactsClient{Locator: s.Locator}
	}
	return disc.Discover(ctx, scenario, path, false)
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) evaluateRules(inv surfaces.Inventory, surface surfaces.Surface, ruleList []rules.Rule, now time.Time) []Finding {
	var out []Finding
	for _, rule := range ruleList {
		if rule.Evaluate == nil {
			continue
		}
		ctx := rules.EvalContext{Inventory: inv, Surface: surface, Now: now}
		for _, finding := range rule.Evaluate(ctx) {
			out = append(out, findingFromRule(inv, finding, now))
		}
	}
	return out
}

func findingFromRule(inv surfaces.Inventory, in rules.Finding, now time.Time) Finding {
	contract, _ := contracts.ByRule(in.RuleID)
	rule, _ := rules.ByID(in.RuleID)
	autofixAvailable := rule.FixClass == rules.FixClassAutofix && autofix.CanFix(inv.RootPath, in.RuleID, in.FilePath)
	f := Finding{
		Scenario:         inv.Scenario,
		TargetKind:       inv.TargetKind,
		SurfaceID:        in.Surface.ID,
		SurfaceKind:      in.Surface.Kind,
		Language:         in.Surface.Language,
		Framework:        in.Surface.Framework,
		RuleID:           in.RuleID,
		Category:         in.Category,
		Severity:         in.Severity,
		FilePath:         in.FilePath,
		Message:          in.Message,
		Evidence:         in.Evidence,
		Expected:         in.Expected,
		Observed:         in.Observed,
		WhyItMatters:     contract.WhyItMatters,
		Remediation:      contract.Remediation,
		FixClass:         rule.FixClass,
		AutofixAvailable: autofixAvailable,
		CreatedAt:        now.UTC().Format(time.RFC3339),
	}
	if autofixAvailable {
		f.AutofixCommand = fmt.Sprintf("quality-health fix-config run %s --rule %s --dry-run", inv.Scenario, in.RuleID)
	}
	f.ID = stableID(f)
	return f
}

func stableID(f Finding) string {
	h := sha256.Sum256([]byte(strings.Join([]string{f.Scenario, f.SurfaceID, f.RuleID, f.FilePath, f.Symbol, f.Expected, f.Observed}, "\x00")))
	return hex.EncodeToString(h[:])[:16]
}

func filterSurfaces(in []surfaces.Surface, ids []string) []surfaces.Surface {
	if len(ids) == 0 {
		return in
	}
	want := map[string]bool{}
	for _, id := range ids {
		want[strings.TrimSpace(id)] = true
	}
	var out []surfaces.Surface
	for _, s := range in {
		if want[s.ID] || want[s.Kind] {
			out = append(out, s)
		}
	}
	return out
}

func contractIDForRules(ruleList []rules.Rule) string {
	if len(ruleList) == 0 {
		return ""
	}
	return ruleList[0].ContractID
}

func ruleIDs(ruleList []rules.Rule) []string {
	out := make([]string, 0, len(ruleList))
	for _, rule := range ruleList {
		out = append(out, rule.ID)
	}
	return out
}

// coverageGapFinding produces the honest "not checked" signal for a discovered
// surface that no quality contract pack covers. It is info-only: it never gates
// run status, but it makes the gap visible and caps maturity.
func coverageGapFinding(inv surfaces.Inventory, surface surfaces.Surface, now time.Time) Finding {
	lang := surface.Language
	if lang == "" {
		lang = "unknown"
	}
	msg := fmt.Sprintf("surface %s (language=%s) discovered but no quality contract applies", surface.ID, lang)
	f := findingFromRule(inv, rules.Finding{
		Surface:  surface,
		RuleID:   contracts.RuleCoverageGap,
		Category: "coverage",
		Severity: "info",
		FilePath: surface.RootPath,
		Message:  msg,
		Evidence: msg,
		Expected: "a quality contract pack covering language=" + lang,
		Observed: "no applicable contract",
	}, now)
	f.WhyItMatters = "A discovered surface that receives zero evaluation must never report a clean pass; this gap keeps missing coverage visible instead of silently green."
	f.Remediation = "File a capability-gap so a quality contract pack is added for language=" + lang + "."
	return f
}

func statusFromFindings(findings []Finding) string {
	for _, f := range findings {
		if f.Severity == "error" {
			return "failed"
		}
	}
	if len(findings) > 0 {
		return "warning"
	}
	return "passed"
}

func sortFindings(findings []Finding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Severity != findings[j].Severity {
			return findings[i].Severity < findings[j].Severity
		}
		if findings[i].RuleID != findings[j].RuleID {
			return findings[i].RuleID < findings[j].RuleID
		}
		return findings[i].FilePath < findings[j].FilePath
	})
}

func auditStatus(inv surfaces.Inventory, findings []Finding) string {
	if inv.DegradedReason != "" {
		return "degraded"
	}
	for _, f := range findings {
		if f.Severity == "error" {
			return "failed"
		}
	}
	return "passed"
}

func maturity(inv surfaces.Inventory, findings []Finding, results []commands.Result, uncoveredSurfaces []string) Maturity {
	if inv.DegradedReason != "" || len(inv.Surfaces) == 0 {
		return Maturity{Rung: 0, Label: "L0", Rationale: "No reliable Code Facts-backed quality audit."}
	}
	if hasError(findings) {
		return Maturity{Rung: 2, Label: "L2", Rationale: "Surfaces discovered, but strict quality contracts are not yet satisfied."}
	}
	if len(uncoveredSurfaces) > 0 {
		return Maturity{Rung: 2, Label: "L2", Rationale: fmt.Sprintf("Coverage is incomplete: surface(s) %s discovered without an applicable quality contract; maturity is capped at L2 until a contract pack covers them.", strings.Join(uncoveredSurfaces, ", "))}
	}
	if len(results) == 0 {
		return Maturity{Rung: 3, Label: "L3", Rationale: "Strict quality contracts are satisfied; command execution was not requested."}
	}
	for _, r := range results {
		if r.Status != "passed" {
			return Maturity{Rung: 3, Label: "L3", Rationale: "Contracts are satisfied but one or more lint/type commands failed."}
		}
	}
	return Maturity{Rung: 4, Label: "L4", Rationale: "Contracts and lint/type commands passed."}
}

func hasError(findings []Finding) bool {
	for _, f := range findings {
		if f.Severity == "error" {
			return true
		}
	}
	return false
}

func summary(res Response) string {
	errors, warnings, infos := countFindings(res.Findings)
	base := fmt.Sprintf("%s: %d error(s), %d warning(s), %d info(s) across %d surface(s)", res.Inventory.Scenario, errors, warnings, infos, len(res.Inventory.Surfaces))
	if n := autofixableCount(res.Findings); n > 0 {
		base += fmt.Sprintf("; %d autofixable", n)
	}
	uncovered := 0
	for _, c := range res.Contracts {
		if c.Status == "uncovered" {
			uncovered++
		}
	}
	if uncovered > 0 {
		base += fmt.Sprintf("; %d surface(s) uncovered", uncovered)
	}
	return base
}

func nextSteps(res Response) []string {
	if n := autofixableCount(res.Findings); n > 0 {
		return []string{fmt.Sprintf("%d finding(s) autofixable — run `quality-health fix-config run %s --dry-run` to inspect safe config repairs.", n, res.Inventory.Scenario)}
	}
	if len(res.Findings) > 0 {
		return []string{fmt.Sprintf("Run `quality-health explain finding %s --scenario %s --rule %s` for remediation detail.", res.Findings[0].ID, res.Inventory.Scenario, res.Findings[0].RuleID)}
	}
	return []string{"No Quality Health remediation is required."}
}

func autofixableCount(findings []Finding) int {
	count := 0
	for _, f := range findings {
		if f.AutofixAvailable {
			count++
		}
	}
	return count
}

func countFindings(findings []Finding) (errors, warnings, infos int) {
	for _, f := range findings {
		switch f.Severity {
		case "error":
			errors++
		case "warning":
			warnings++
		default:
			infos++
		}
	}
	return errors, warnings, infos
}
