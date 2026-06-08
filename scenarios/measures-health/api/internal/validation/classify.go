// Package validation grades how well a target scenario has adopted the Measures
// capability. The pure heart is Classify: given a scenario's derived stateful
// domains, its harvested measures, and its waivers/overrides, it produces the
// coverage Report with per-domain status, per-measure tier, normalized producer
// findings, and a pass/fail verdict. The filesystem/network gathering that feeds
// Classify lives behind seams in service.go (see docs/internal/SEAMS.md).
//
// Expectation model (copied from cli-health + security-health):
//   - EXPECTED  = the scenario's stateful domains (a domain proto == a persisted
//     entity type == stateful by construction), minus a stateless/utility filter
//     and any measures.domains[] override.
//   - COVERED   = a stateful domain with >=1 manifest measure block.
//   - WAIVED    = a stateful domain listed in measures.omitted[] with a reason.
//   - uncovered stateful domain -> ERROR; stale waiver -> WARNING.
package validation

import (
	"sort"
	"strings"

	measures "github.com/vrooli/measures-go"
	"measures-health/internal/measurescan"
)

// Severity classifies a finding. ERROR drives passed=false and is the only level
// that feeds the ecosystem-manager `measures` ladder dimension as a gap.
type Severity int

const (
	SeverityInfo Severity = iota
	SeverityWarning
	SeverityError
)

func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "error"
	case SeverityWarning:
		return "warning"
	default:
		return "info"
	}
}

// DomainStatus is a domain's coverage classification.
type DomainStatus string

const (
	StatusCovered     DomainStatus = "covered"
	StatusUncovered   DomainStatus = "uncovered"
	StatusWaived      DomainStatus = "waived"
	StatusNotExpected DomainStatus = "not_expected"
)

// Finding is one normalized producer result (mirrors the proto Finding shape so
// the test-genie `measures` phase parses it identically to security-health).
type Finding struct {
	RuleID      string
	Severity    Severity
	Title       string
	Description string
	Remediation string
	FilePath    string
	Scanner     string // "coverage" | "probe" | "tier"
}

// MeasureSummary is one declared measure's projection in the report.
type MeasureSummary struct {
	Name          string
	Intent        string
	Tier          measurescan.Tier
	Effect        string
	QuestionCount int
	TierNote      string
	ProbePassed   bool
	ProbeDetail   string
}

// DomainCoverage is one domain's classification plus the measures covering it.
type DomainCoverage struct {
	Domain       string
	Status       DomainStatus
	MeasureCount int
	Tier         measurescan.Tier // worst tier among the domain's measures
	WaiverReason string
	Note         string
	Measures     []MeasureSummary
}

// Report is the full coverage verdict for one scenario.
type Report struct {
	Scenario        string
	Passed          bool
	Domains         []DomainCoverage
	Findings        []Finding
	SkippedScanners []string
}

// Summary rolls up findings by severity.
func (r Report) Summary() (errors, warnings, infos int) {
	for _, f := range r.Findings {
		switch f.Severity {
		case SeverityError:
			errors++
		case SeverityWarning:
			warnings++
		default:
			infos++
		}
	}
	return
}

// DerivedDomain is a domain derived from the target scenario's proto domain/
// folder, tagged with whether it is stateful (a measure is expected) after the
// stateless/utility filter.
type DerivedDomain struct {
	Name     string
	Stateful bool
	Note     string // why not stateful, when Stateful == false
}

// HarvestedMeasure is one assembled measure plus its graded tier. AssembleErr is
// non-empty when the manifest measure block failed to assemble against its proto
// request (drift / malformed declaration) — a hard ERROR.
type HarvestedMeasure struct {
	Name          string
	Domain        string
	Intent        string
	Effect        string
	QuestionCount int
	Tier          measurescan.Tier
	TierNote      string
	Decl          measures.MeasureDeclaration
	AssembleErr   string
}

// Inputs is everything Classify needs, already gathered by the service shell.
type Inputs struct {
	Scenario  string
	Domains   []DerivedDomain
	Measures  []HarvestedMeasure
	Omitted   []measurescan.Omission
	Overrides []measurescan.DomainOverride
}

// manifestPath is the conventional location a coverage finding points at.
const manifestPath = "cli/manifest.json"

// Classify is the pure heart: it turns gathered inputs into the coverage report
// + normalized findings. It performs no I/O and is fully table-testable.
func Classify(in Inputs) Report {
	rep := Report{Scenario: in.Scenario}

	// 1. Resolve statefulness per domain: start from the derived set, then let
	//    measures.domains[] overrides force the flag (and introduce domains the
	//    proto scan did not surface).
	stateful := map[string]bool{}
	note := map[string]string{}
	known := map[string]bool{} // every domain we have any signal about
	for _, d := range in.Domains {
		stateful[d.Name] = d.Stateful
		known[d.Name] = true
		if !d.Stateful && d.Note != "" {
			note[d.Name] = d.Note
		}
	}
	for _, ov := range in.Overrides {
		stateful[ov.Domain] = ov.Stateful
		known[ov.Domain] = true
		if !ov.Stateful {
			if r := strings.TrimSpace(ov.Reason); r != "" {
				note[ov.Domain] = r
			} else if note[ov.Domain] == "" {
				note[ov.Domain] = "marked non-stateful via measures.domains[] override"
			}
		}
	}

	// 2. Bucket measures by domain (best-first within a domain is irrelevant; we
	//    aggregate). Track assembly drift as a hard finding.
	byDomain := map[string][]MeasureSummary{}
	for _, m := range in.Measures {
		known[m.Domain] = true
		if m.AssembleErr != "" {
			rep.Findings = append(rep.Findings, Finding{
				RuleID:      "measures.malformed-declaration",
				Severity:    SeverityError,
				Title:       "Malformed measure declaration: " + m.Name,
				Description: "The measure block did not assemble against its bound proto request: " + m.AssembleErr,
				Remediation: "Fix the manifest measure block or its binding so it assembles against the proto request message (a manifest param must name a real proto field; result.value_field is mandatory).",
				FilePath:    manifestPath,
				Scanner:     "coverage",
			})
			continue
		}
		ms := MeasureSummary{
			Name:          m.Name,
			Intent:        m.Intent,
			Tier:          m.Tier,
			Effect:        m.Effect,
			QuestionCount: m.QuestionCount,
			TierNote:      m.TierNote,
		}
		byDomain[m.Domain] = append(byDomain[m.Domain], ms)
	}

	// 3. Waivers keyed by domain. A waiver also makes its domain "known" so a
	//    waiver pointing at a non-stateful/nonexistent domain surfaces as a stale
	//    WARNING rather than vanishing.
	waived := map[string]string{}
	for _, o := range in.Omitted {
		waived[o.Domain] = o.Reason
		known[o.Domain] = true
	}

	// 4. Classify every known domain in stable order.
	names := make([]string, 0, len(known))
	for n := range known {
		names = append(names, n)
	}
	sort.Strings(names)

	for _, name := range names {
		isStateful := stateful[name]
		ms := byDomain[name]
		dc := DomainCoverage{Domain: name, Measures: ms, MeasureCount: len(ms)}

		switch {
		case len(ms) > 0:
			// Covered (whether or not it was "expected" — extra measures on a
			// non-stateful domain are over-delivery, never a gap).
			dc.Status = StatusCovered
			dc.Tier = worstTier(ms)
			rep.Findings = append(rep.Findings, tierFindings(name, ms)...)
		case isStateful && waived[name] != "":
			dc.Status = StatusWaived
			dc.WaiverReason = waived[name]
		case isStateful:
			dc.Status = StatusUncovered
			rep.Findings = append(rep.Findings, Finding{
				RuleID:      "measures.uncovered-domain",
				Severity:    SeverityError,
				Title:       "Stateful domain has no measure: " + name,
				Description: "Domain `" + name + "` owns countable persisted state but declares no measure and is not waived.",
				Remediation: "Declare a `measure` block on a CLI command bound to the `" + name + "` domain, or waive it in measures.omitted[] with a reason if it genuinely has no historical value.",
				FilePath:    manifestPath,
				Scanner:     "coverage",
			})
		case waived[name] != "":
			// Waiver pointing at a non-stateful / nonexistent domain -> stale.
			dc.Status = StatusNotExpected
			dc.Note = "waiver targets a non-stateful domain"
			rep.Findings = append(rep.Findings, Finding{
				RuleID:      "measures.stale-waiver",
				Severity:    SeverityWarning,
				Title:       "Stale waiver: " + name,
				Description: "measures.omitted[] waives `" + name + "`, but that domain is not stateful (or does not exist), so the waiver is unnecessary.",
				Remediation: "Remove the `" + name + "` entry from measures.omitted[]; only stateful domains need a waiver.",
				FilePath:    manifestPath,
				Scanner:     "coverage",
			})
		default:
			// Known stateless domain (from the substrate filter): informational.
			dc.Status = StatusNotExpected
			dc.Note = note[name]
			if dc.Note == "" {
				dc.Note = "not stateful"
			}
		}
		rep.Domains = append(rep.Domains, dc)
	}

	errs, _, _ := rep.Summary()
	rep.Passed = errs == 0
	return rep
}

// tierFindings emits advisory findings for weak extraction tiers: a fallback
// (all best-effort) measure is a WARNING; a partial measure is INFO. Neither
// blocks the verdict — tier is a maturity signal, not a correctness gate — but
// both surface so a scenario knows what to harden.
func tierFindings(domain string, ms []MeasureSummary) []Finding {
	var out []Finding
	for _, m := range ms {
		switch m.Tier {
		case measurescan.TierFallback:
			out = append(out, Finding{
				RuleID:      "measures.tier-fallback",
				Severity:    SeverityWarning,
				Title:       "Measure has no canonical params: " + m.Name,
				Description: "Every parameter of `" + m.Name + "` is best-effort extracted (no time_window / enum / bounded type), so answers may be wrong rather than abstained.",
				Remediation: "Adopt the canonical time_window type for date params and proto enums/bounds for the rest so extraction is deterministic or constrained.",
				FilePath:    manifestPath,
				Scanner:     "tier",
			})
		case measurescan.TierPartial:
			detail := "some parameters are best-effort extracted"
			if m.TierNote != "" {
				detail = m.TierNote
			}
			out = append(out, Finding{
				RuleID:      "measures.tier-partial",
				Severity:    SeverityInfo,
				Title:       "Measure partially canonical: " + m.Name,
				Description: "`" + m.Name + "`: " + detail + ".",
				Remediation: "Promote the remaining bare params to canonical/constrained types for full-tier extraction.",
				FilePath:    manifestPath,
				Scanner:     "tier",
			})
		}
	}
	return out
}

// worstTier returns the weakest tier among a domain's measures (the domain is
// only as strong as its weakest measure). Empty when there are no measures.
func worstTier(ms []MeasureSummary) measurescan.Tier {
	rank := map[measurescan.Tier]int{
		measurescan.TierFull:     0,
		measurescan.TierPartial:  1,
		measurescan.TierFallback: 2,
	}
	worst := measurescan.TierFull
	for _, m := range ms {
		if rank[m.Tier] > rank[worst] {
			worst = m.Tier
		}
	}
	return worst
}
