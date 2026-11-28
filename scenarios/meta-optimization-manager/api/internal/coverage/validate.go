package coverage

import (
	"context"
	"fmt"
	"strings"

	"github.com/vrooli/api-core/spacedoc"
)

// ValidateBaseDocs checks the space docs themselves — the self-honesty gate
// (OT-P0-004). It surfaces base-document-integrity issues and reports ok=false
// when any ERROR-severity issue exists (which flips the CLI exit non-zero). The
// checks are deliberately conservative so the gate never false-fails on
// legitimate authoring:
//
//   - guide_row_no_skill (ERROR): a Guide row claims coverage (now/in_reach) but
//     names no guiding skill — a direct contradiction of the denominator.
//   - guide_row_not_one_skill (INFO): a covered Guide row maps to ≠1 skill. The
//     trials gate prefers one primary skill per row; this surfaces the fuzziness
//     without gating (multi-skill rows are legitimate).
//   - missing_provider (WARN): an Answer/Validate cell authored NOW whose declared
//     provider is not in the live registry — drift between the doc and reality.
//   - ungraduated_pointer (WARN): a concern whose Validate phase is live (NOW) but
//     whose Guide row is absent (MISSING) — the prose→programmatic loop built the
//     validator but left no pointer-skill behind (Guide→Validate→Answer gradient,
//     COVERAGE-MODEL.md). Cross-document, deterministic: no live registry needed.
//   - graduation_ref_unresolved (WARN): a graduation cross-walk entry points at a
//     Guide/Validate cell id that no longer exists — a stale map vs renumbered doc.
//   - denominator_unavailable (WARN): the owner's space verb was unreachable.
func (s *service) ValidateBaseDocs(ctx context.Context, projection Projection) (BaseDocReport, error) {
	targets := AllProjections
	if projection != "" {
		if OwnerFor(projection) == "" {
			return BaseDocReport{}, fmt.Errorf("coverage: unknown projection %q", projection)
		}
		targets = []Projection{projection}
	}

	report := BaseDocReport{OK: true}
	defs := map[Projection]*spacedoc.SpaceDefinition{}
	for _, p := range targets {
		def, err := s.reader.Read(ctx, p)
		if err != nil {
			severity := SeverityWarn
			code := "denominator_unavailable"
			if strings.Contains(strings.ToLower(err.Error()), "unrecognized status token") {
				severity = SeverityError
				code = "denominator_parse_error"
			}
			report.add(BaseDocIssue{
				Projection: p,
				Code:       code,
				Message:    fmt.Sprintf("%s space verb unreachable: %v", OwnerFor(p), err),
				Location:   OwnerFor(p),
				Severity:   severity,
			})
			continue
		}
		defs[p] = def
		if p == ProjectionGuide {
			validateGuideRows(def, &report)
		}
		validateLiveDrift(ctx, s, p, def, &report)
	}

	// Cross-projection Guide↔Validate graduation consistency. Needs both
	// denominators; when Validate is not among the scoped targets (e.g.
	// --projection guide) it is read on demand. Pure doc-vs-doc — the Validate
	// covered set is AUTHORITATIVE, so no live test-genie call is required.
	if guideDef, ok := defs[ProjectionGuide]; ok {
		validateDef, haveValidate := defs[ProjectionValidate]
		onDemand := false
		if !haveValidate {
			onDemand = true
			if vd, err := s.reader.Read(ctx, ProjectionValidate); err == nil {
				validateDef, haveValidate = vd, true
			}
		}
		switch {
		case haveValidate:
			for _, is := range graduationFindings(graduationLinks, guideDef, validateDef) {
				report.add(is)
			}
		case onDemand:
			// Only surface a dedicated skip note when Validate was not already a
			// scoped target (its unavailability is otherwise reported above).
			report.add(BaseDocIssue{
				Projection: ProjectionGuide,
				Code:       "graduation_check_skipped",
				Message:    fmt.Sprintf("Guide→Validate graduation cross-check skipped: %s denominator unavailable", OwnerFor(ProjectionValidate)),
				Location:   OwnerFor(ProjectionValidate),
				Severity:   SeverityWarn,
			})
		}
	}
	return report, nil
}

// graduationLink ties a Guide SWE-task row to the Validate phase it graduates
// into along the Guide→Validate→Answer maturation gradient (COVERAGE-MODEL.md).
type graduationLink struct {
	GuideID    string // guide-space cell id (G…)
	ValidateID string // validate-space cell id (V…)
	Concern    string // human label for findings
}

// graduationLinks is MoM's curated cross-projection cross-walk. It lives here, in
// the aggregator, because a link spans two owners' denominators (prompt-manager's
// Guide row and test-genie's Validate phase) — it is aggregation glue, not a
// denominator field, so neither owner's space doc owns it (DECISIONS: MoM never
// owns a denominator, but it does own the cross-walk between them). Keyed on the
// stable cell ids both space docs publish; every entry is validated each run
// (a dangling id raises graduation_ref_unresolved).
var graduationLinks = []graduationLink{
	{"G2", "V4", "Architecture"},
	{"G10", "V14", "Proto / RPC"},
	{"G11", "V15", "UI"},
	{"G12", "V9", "Storage"},
	{"G14", "V8", "Unit tests"},
	{"G16", "V6", "Quality / lint / types"},
	{"G20", "V16", "Performance"},
	{"G21", "V12", "Security"},
	{"G23", "V7", "Documentation"},
	{"G26", "V5", "Dependencies"},
	{"G31", "V20", "Concurrency / replay safety"},
	{"G32", "V18", "Observability / telemetry"},
}

// graduationFindings is the deterministic Guide↔Validate consistency check. For
// each declared link it reports graduation_ref_unresolved when an id is dangling,
// and ungraduated_pointer when the Validate phase is live (NOW) but the Guide row
// is absent (MISSING). It fires on MISSING only: a PARTIAL/in_reach Guide row
// already has *some* guidance (the prose→programmatic backlog, not an absent
// pointer), so flagging it here would be noise.
func graduationFindings(links []graduationLink, guideDef, validateDef *spacedoc.SpaceDefinition) []BaseDocIssue {
	guideByID := indexCells(guideDef)
	validateByID := indexCells(validateDef)
	var out []BaseDocIssue
	for _, link := range links {
		g, gok := guideByID[link.GuideID]
		v, vok := validateByID[link.ValidateID]
		if !gok || !vok {
			loc := fmt.Sprintf("%s#%s", validateDef.Source, link.ValidateID)
			if !gok {
				loc = fmt.Sprintf("%s#%s", guideDef.Source, link.GuideID)
			}
			out = append(out, BaseDocIssue{
				Projection: ProjectionGuide,
				Code:       "graduation_ref_unresolved",
				Message:    fmt.Sprintf("graduation link %s→%s (%s) references a missing cell (guide_found=%t, validate_found=%t)", link.GuideID, link.ValidateID, link.Concern, gok, vok),
				Location:   loc,
				Severity:   SeverityWarn,
			})
			continue
		}
		if v.Status == spacedoc.StatusNow && g.Status == spacedoc.StatusMissing {
			out = append(out, BaseDocIssue{
				Projection: ProjectionGuide,
				Code:       "ungraduated_pointer",
				Message: fmt.Sprintf("Guide row %s (%s) is MISSING but its Validate phase %s (%s) is live — guidance should exist as a graduated pointer, not be absent",
					g.ID, link.Concern, v.ID, v.Question),
				Location: fmt.Sprintf("%s#%s", guideDef.Source, g.ID),
				Severity: SeverityWarn,
			})
		}
	}
	return out
}

// indexCells indexes a denominator's cells by their stable id.
func indexCells(def *spacedoc.SpaceDefinition) map[string]spacedoc.Cell {
	out := make(map[string]spacedoc.Cell, len(def.Cells))
	for _, c := range def.Cells {
		out[c.ID] = c
	}
	return out
}

// validateGuideRows enforces the Guide-row → skill rules.
func validateGuideRows(def *spacedoc.SpaceDefinition, report *BaseDocReport) {
	for _, c := range def.Cells {
		if c.Status == spacedoc.StatusMissing {
			continue // a MISSING row legitimately has no skill
		}
		skills := skillCount(c.Owner)
		loc := fmt.Sprintf("%s#%s", def.Source, c.ID)
		if skills == 0 {
			report.add(BaseDocIssue{
				Projection: ProjectionGuide,
				Code:       "guide_row_no_skill",
				Message:    fmt.Sprintf("Guide row %s is %s but names no guiding skill", c.ID, c.Status),
				Location:   loc,
				Severity:   SeverityError,
			})
			continue
		}
		if c.Status == spacedoc.StatusNow && skills != 1 {
			report.add(BaseDocIssue{
				Projection: ProjectionGuide,
				Code:       "guide_row_not_one_skill",
				Message:    fmt.Sprintf("Guide row %s maps to %d skills; the trials gate prefers one primary skill", c.ID, skills),
				Location:   loc,
				Severity:   SeverityInfo,
			})
		}
	}
}

// validateLiveDrift flags Answer/Validate cells authored NOW whose declared
// provider is not live (drift), using the numerator join. Skipped when the live
// registry is unreachable (that is reported once as denominator/registry
// unavailability, not per-cell).
func validateLiveDrift(ctx context.Context, s *service, p Projection, def *spacedoc.SpaceDefinition, report *BaseDocReport) {
	if p == ProjectionGuide {
		return
	}
	join := s.joiner.Join(ctx, p, def.Cells)
	if p == ProjectionAnswer && join.Available {
		for _, cell := range def.Cells {
			if len(providerTokens(cell.Owner)) == 0 || join.OwnerResolved[cell.ID] {
				continue
			}
			report.add(BaseDocIssue{
				Projection: p,
				Code:       "owner_leaf_unresolved",
				Message:    fmt.Sprintf("cell %s owner %q does not resolve to a registered provider leaf id", cell.ID, cell.Owner),
				Location:   fmt.Sprintf("%s#%s", def.Source, cell.ID),
				Severity:   SeverityError,
			})
		}
	}
	if !join.Available || join.Statuses == nil {
		return
	}
	for _, c := range def.Cells {
		if c.Status != spacedoc.StatusNow {
			continue
		}
		if eff, ok := join.Statuses[c.ID]; ok && eff != spacedoc.StatusNow {
			if p == ProjectionAnswer {
				issue := answerDriftIssue(c, join.Evidence[c.ID])
				if issue != nil {
					report.add(*issue)
					continue
				}
			}
			report.add(BaseDocIssue{
				Projection: p,
				Code:       "missing_provider",
				Message:    fmt.Sprintf("cell %s is authored NOW but its provider %q is not live", c.ID, c.Owner),
				Location:   fmt.Sprintf("%s#%s", def.Source, c.ID),
				Severity:   SeverityWarn,
			})
		}
	}
}

func answerDriftIssue(c spacedoc.Cell, evidence []SignalEvidence) *BaseDocIssue {
	for _, signal := range evidence {
		if signal.Signal != "active" || signal.Verdict == "held" {
			continue
		}
		return &BaseDocIssue{Projection: ProjectionAnswer, Code: "missing_provider", Message: fmt.Sprintf("cell %s owner %q is not an ACTIVE registered provider", c.ID, c.Owner), Location: c.ID, Severity: SeverityError}
	}
	for _, signal := range evidence {
		if signal.Signal == "reachable" && signal.Verdict != "held" {
			return &BaseDocIssue{Projection: ProjectionAnswer, Code: "provider_not_live", Message: fmt.Sprintf("cell %s owner %q is registered but not live: %s", c.ID, c.Owner, signal.Evidence), Location: c.ID, Severity: SeverityError}
		}
	}
	for _, signal := range evidence {
		if signal.Signal == "corpus_eval_fresh" && signal.Verdict != "held" {
			return &BaseDocIssue{Projection: ProjectionAnswer, Code: "corpus_quality_debt", Message: fmt.Sprintf("cell %s owner %q has direct corpus quality debt: %s", c.ID, c.Owner, signal.Evidence), Location: c.ID, Severity: SeverityWarn}
		}
	}
	for _, signal := range evidence {
		if signal.Signal == "eval_fresh" && signal.Verdict != "held" {
			return &BaseDocIssue{Projection: ProjectionAnswer, Code: "router_quality_debt", Message: fmt.Sprintf("cell %s owner %q has federated router quality debt: %s", c.ID, c.Owner, signal.Evidence), Location: c.ID, Severity: SeverityWarn}
		}
	}
	return nil
}

// skillCount counts the guiding skills named in a Guide row's owner cell. Skills
// are comma/plus-separated; prose fragments ("the Answer projection", "(none)")
// and empty segments are not counted. A segment counts as a skill when it is a
// hyphenated slug or a single bare word (e.g. "explore", "polish").
func skillCount(owner string) int {
	return len(skillTokens(owner))
}

func (r *BaseDocReport) add(issue BaseDocIssue) {
	r.Issues = append(r.Issues, issue)
	if issue.Severity == SeverityError {
		r.OK = false
	}
}
