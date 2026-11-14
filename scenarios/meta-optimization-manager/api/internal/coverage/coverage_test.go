package coverage

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/api-core/spacedoc"
)

// fakeReader returns canned denominators per projection (and optional errors).
type fakeReader struct {
	defs map[Projection]*spacedoc.SpaceDefinition
	errs map[Projection]error
}

func (f fakeReader) Read(_ context.Context, p Projection) (*spacedoc.SpaceDefinition, error) {
	if err := f.errs[p]; err != nil {
		return nil, err
	}
	if d, ok := f.defs[p]; ok {
		return d, nil
	}
	return nil, errors.New("no def")
}

func answerDef() *spacedoc.SpaceDefinition {
	return &spacedoc.SpaceDefinition{
		SchemaVersion:         "v1",
		Projection:            ProjectionAnswer,
		Owner:                 "search-hub",
		DenominatorConfidence: spacedoc.ConfidencePartial,
		Source:                "scenarios/search-hub/docs/spaces/answer-space.md",
		Cells: []spacedoc.Cell{
			{ID: "1", Question: "Surfaces", Owner: "ui-health.surfaces", Status: spacedoc.StatusNow, Basis: spacedoc.BasisDerived},
			{ID: "2", Question: "Domains", Owner: "architecture-cartographer.domain-map", Status: spacedoc.StatusInReach},
			{ID: "9", Question: "Wiring", Owner: "_(none)_", Status: spacedoc.StatusMissing},
		},
	}
}

func guideDef() *spacedoc.SpaceDefinition {
	return &spacedoc.SpaceDefinition{
		SchemaVersion:         "v1",
		Projection:            ProjectionGuide,
		Owner:                 "prompt-manager",
		DenominatorConfidence: spacedoc.ConfidenceSketch,
		Source:                "scenarios/prompt-manager/docs/spaces/guide-space.md",
		Cells: []spacedoc.Cell{
			{ID: "G1", Question: "Explore", Owner: "explore", Status: spacedoc.StatusNow},
			{ID: "G2", Question: "Audit", Owner: "screaming-architecture-audit, architecture-scope", Status: spacedoc.StatusNow},
			{ID: "G31", Question: "Concurrency", Owner: "_(none)_", Status: spacedoc.StatusMissing},
		},
	}
}

func actDef() *spacedoc.SpaceDefinition {
	return &spacedoc.SpaceDefinition{
		SchemaVersion:         "v1",
		Projection:            ProjectionAct,
		Owner:                 "program-runtime",
		DenominatorConfidence: spacedoc.ConfidencePartial,
		Source:                "scenarios/program-runtime/docs/spaces/act-space.md",
		Cells:                 []spacedoc.Cell{{ID: "A1", Question: "Act", Owner: "program-runtime.bindings", Status: spacedoc.StatusInReach}},
	}
}

// staticJoiner returns a fixed JoinResult.
type staticJoiner struct{ res JoinResult }

func (j staticJoiner) Join(context.Context, Projection, []spacedoc.Cell) JoinResult { return j.res }

type alternatingJoiner struct{ calls int }

func (j *alternatingJoiner) Join(_ context.Context, _ Projection, _ []spacedoc.Cell) JoinResult {
	j.calls++
	if j.calls%2 == 0 {
		return JoinResult{Available: true, Statuses: map[string]spacedoc.CellStatus{"1": spacedoc.StatusNow}}
	}
	return JoinResult{Available: true, Statuses: map[string]spacedoc.CellStatus{"1": spacedoc.StatusInReach}}
}

func TestGetStatusComputesCounts(t *testing.T) {
	svc := NewService(Deps{
		Reader: fakeReader{defs: map[Projection]*spacedoc.SpaceDefinition{ProjectionAnswer: answerDef()}},
		Joiner: staticJoiner{JoinResult{Available: true}},
		Clock:  fixedClock{},
	})
	st, err := svc.GetStatus(context.Background(), ProjectionAnswer)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Projections) != 1 {
		t.Fatalf("projections=%d", len(st.Projections))
	}
	pc := st.Projections[0]
	if pc.TotalCells != 3 || pc.NowCount != 1 || pc.InReachCount != 1 || pc.MissingCount != 1 {
		t.Errorf("counts: %+v", pc)
	}
	if pc.CoverageRatio < 0.33 || pc.CoverageRatio > 0.34 {
		t.Errorf("ratio=%v", pc.CoverageRatio)
	}
	if !pc.Available || pc.DenominatorConfidence != spacedoc.ConfidencePartial {
		t.Errorf("avail/conf: %+v", pc)
	}
	if !st.DeterminismChecked || !st.Deterministic || st.DeterminismEvidence == "" {
		t.Errorf("determinism self-check = checked:%t deterministic:%t evidence:%q", st.DeterminismChecked, st.Deterministic, st.DeterminismEvidence)
	}
}

func TestGetStatusPublishesActReachabilityBasis(t *testing.T) {
	svc := NewService(Deps{
		Reader: fakeReader{defs: map[Projection]*spacedoc.SpaceDefinition{ProjectionAct: actDef()}},
		Joiner: staticJoiner{JoinResult{
			Available:             true,
			Statuses:              map[string]spacedoc.CellStatus{"A1": spacedoc.StatusNow},
			ManifestScenarios:     73,
			TotalScenarios:        122,
			ReachableScenarios:    57,
			UnreachableScenarios:  16,
			ReachabilityCheckedAt: "2026-08-18T12:00:00Z",
		}},
		Clock: fixedClock{},
	})
	st, err := svc.GetStatus(context.Background(), ProjectionAct)
	if err != nil {
		t.Fatal(err)
	}
	pc := st.Projections[0]
	if pc.ManifestScenarios != 73 || pc.TotalScenarios != 122 || pc.ReachableScenarios != 57 || pc.UnreachableScenarios != 16 || pc.ReachabilityCheckedAt != "2026-08-18T12:00:00Z" {
		t.Fatalf("act basis = %+v", pc)
	}
}

func TestGetStatusSurfacesAnswerDeterminismMismatch(t *testing.T) {
	joiner := &alternatingJoiner{}
	svc := NewService(Deps{
		Reader: fakeReader{defs: map[Projection]*spacedoc.SpaceDefinition{ProjectionAnswer: answerDef()}},
		Joiner: joiner,
		Clock:  fixedClock{},
	})
	status, err := svc.GetStatus(context.Background(), ProjectionAnswer)
	if err != nil {
		t.Fatal(err)
	}
	if !status.DeterminismChecked || status.Deterministic {
		t.Fatalf("determinism = checked:%t deterministic:%t evidence:%q, want checked mismatch", status.DeterminismChecked, status.Deterministic, status.DeterminismEvidence)
	}
	if !strings.Contains(status.DeterminismEvidence, "disagreed") {
		t.Fatalf("determinism evidence = %q, want disagreement", status.DeterminismEvidence)
	}
}

func TestGetStatusDegradesWhenOwnerDown(t *testing.T) {
	svc := NewService(Deps{
		Reader: fakeReader{errs: map[Projection]error{ProjectionAnswer: errors.New("boom")}},
		Joiner: staticJoiner{JoinResult{Available: true}},
		Clock:  fixedClock{},
	})
	st, err := svc.GetStatus(context.Background(), ProjectionAnswer)
	if err != nil {
		t.Fatal(err)
	}
	pc := st.Projections[0]
	if pc.Available {
		t.Errorf("expected unavailable, got %+v", pc)
	}
	if pc.UnavailableReason == "" {
		t.Error("expected an honest reason")
	}
	if pc.CoverageRatio != 0 || pc.DenominatorConfidence != "" {
		t.Errorf("unavailable projection must omit ratio and confidence: %+v", pc)
	}
}

func TestGetStatusOmitsRatioAndConfidenceWhenJoinUnavailable(t *testing.T) {
	svc := NewService(Deps{
		Reader: fakeReader{defs: map[Projection]*spacedoc.SpaceDefinition{ProjectionAnswer: answerDef()}},
		Joiner: staticJoiner{JoinResult{Available: false, Reason: "search-hub registry unreachable"}},
		Clock:  fixedClock{},
	})
	st, err := svc.GetStatus(context.Background(), ProjectionAnswer)
	if err != nil {
		t.Fatal(err)
	}
	pc := st.Projections[0]
	if pc.Available {
		t.Fatalf("expected unavailable projection, got %+v", pc)
	}
	if pc.UnavailableReason != "search-hub registry unreachable" {
		t.Errorf("reason = %q", pc.UnavailableReason)
	}
	if pc.CoverageRatio != 0 || pc.DenominatorConfidence != "" {
		t.Errorf("unavailable projection must omit ratio and confidence: %+v", pc)
	}
}

func TestListCellsFilters(t *testing.T) {
	svc := NewService(Deps{
		Reader: fakeReader{defs: map[Projection]*spacedoc.SpaceDefinition{ProjectionAnswer: answerDef()}},
		Joiner: staticJoiner{JoinResult{Available: true}},
		Clock:  fixedClock{},
	})
	cells, err := svc.ListCells(context.Background(), ProjectionAnswer, spacedoc.StatusMissing)
	if err != nil {
		t.Fatal(err)
	}
	if len(cells) != 1 || cells[0].ID != "answer/9" {
		t.Errorf("cells = %+v", cells)
	}
}

func TestExplainCell(t *testing.T) {
	join := JoinResult{Available: true, Statuses: map[string]spacedoc.CellStatus{"1": spacedoc.StatusInReach}, OwnerResolved: map[string]bool{"1": true, "2": true}, Evidence: map[string][]SignalEvidence{
		"1": {
			{Signal: "active", Verdict: "held", Evidence: "ui-health.surfaces is ACTIVE"},
			{Signal: "reachable", Verdict: "held", Evidence: "reachable"},
			{Signal: "corpus_eval_fresh", Verdict: "held", Evidence: "fresh provider_direct eval run run-1"},
			{Signal: "eval_fresh", Verdict: "held", Evidence: "fresh non-degraded eval run run-1"},
		},
	}}
	svc := NewService(Deps{
		Reader: fakeReader{defs: map[Projection]*spacedoc.SpaceDefinition{ProjectionAnswer: answerDef()}},
		Joiner: staticJoiner{join},
		Clock:  fixedClock{},
	})
	cell, err := svc.ExplainCell(context.Background(), "answer/1")
	if err != nil {
		t.Fatal(err)
	}
	if cell.Question != "Surfaces" || len(cell.Citations) == 0 {
		t.Errorf("cell = %+v", cell)
	}
	if len(cell.SignalEvidence) != 4 {
		t.Fatalf("signal evidence = %+v", cell.SignalEvidence)
	}
	for _, signal := range []string{"active", "reachable", "corpus_eval_fresh", "eval_fresh"} {
		found := false
		for _, note := range cell.Notes {
			if strings.Contains(note, "answer signal "+signal) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("explanation omitted %s: %v", signal, cell.Notes)
		}
	}
	if _, err := svc.ExplainCell(context.Background(), "answer/999"); err == nil {
		t.Error("expected not-found")
	}
	if _, err := svc.ExplainCell(context.Background(), "bogus"); err == nil {
		t.Error("expected bad-id error")
	}
}

func TestValidateBaseDocsGuideGate(t *testing.T) {
	// Inject a guide def with a COVERED row that has no skill -> ERROR -> ok=false.
	bad := guideDef()
	bad.Cells = append(bad.Cells, spacedoc.Cell{ID: "G99", Question: "Orphan", Owner: "_(none)_", Status: spacedoc.StatusNow})
	svc := NewService(Deps{
		Reader: fakeReader{defs: map[Projection]*spacedoc.SpaceDefinition{ProjectionGuide: bad}},
		Joiner: staticJoiner{JoinResult{Available: true}},
		Clock:  fixedClock{},
	})
	rep, err := svc.ValidateBaseDocs(context.Background(), ProjectionGuide)
	if err != nil {
		t.Fatal(err)
	}
	if rep.OK {
		t.Error("expected ok=false from guide_row_no_skill ERROR")
	}
	var foundErr, foundInfo bool
	for _, is := range rep.Issues {
		if is.Code == "guide_row_no_skill" && is.Severity == SeverityError {
			foundErr = true
		}
		if is.Code == "guide_row_not_one_skill" {
			foundInfo = true // G2 has 2 skills
		}
	}
	if !foundErr {
		t.Error("missing guide_row_no_skill error")
	}
	if !foundInfo {
		t.Error("missing guide_row_not_one_skill info for multi-skill row")
	}
}

func TestValidateBaseDocsCleanGuide(t *testing.T) {
	svc := NewService(Deps{
		Reader: fakeReader{defs: map[Projection]*spacedoc.SpaceDefinition{ProjectionGuide: guideDef()}},
		Joiner: staticJoiner{JoinResult{Available: true}},
		Clock:  fixedClock{},
	})
	rep, err := svc.ValidateBaseDocs(context.Background(), ProjectionGuide)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.OK {
		t.Errorf("expected ok=true, issues=%+v", rep.Issues)
	}
}

func TestValidateBaseDocsAnswerChargesOnlyCorpusGate(t *testing.T) {
	join := JoinResult{Available: true, Statuses: map[string]spacedoc.CellStatus{"1": spacedoc.StatusInReach}, OwnerResolved: map[string]bool{"1": true, "2": true}, Evidence: map[string][]SignalEvidence{
		"1": {
			{Signal: "active", Verdict: "held", Evidence: "provider active"},
			{Signal: "reachable", Verdict: "held", Evidence: "provider reachable"},
			{Signal: "corpus_eval_fresh", Verdict: "held", Evidence: "direct pass"},
			{Signal: "eval_fresh", Verdict: "did_not_hold", Evidence: "federated routing recall below threshold"},
		},
	}}
	svc := NewService(Deps{
		Reader: fakeReader{defs: map[Projection]*spacedoc.SpaceDefinition{ProjectionAnswer: answerDef()}},
		Joiner: staticJoiner{join},
		Clock:  fixedClock{},
	})
	report, err := svc.ValidateBaseDocs(context.Background(), ProjectionAnswer)
	if err != nil {
		t.Fatal(err)
	}
	foundRouterDebt := false
	for _, issue := range report.Issues {
		if issue.Code == "eval_gate_unmet" || issue.Code == "corpus_quality_debt" {
			t.Fatalf("federated quality debt must not charge the provider corpus gate: %+v", issue)
		}
		if issue.Code != "router_quality_debt" {
			continue
		}
		foundRouterDebt = true
	}
	if !foundRouterDebt {
		t.Fatal("expected federated quality debt warning")
	}
}

func TestValidateBaseDocsAnswerReportsCorpusGate(t *testing.T) {
	join := JoinResult{Available: true, Statuses: map[string]spacedoc.CellStatus{"1": spacedoc.StatusInReach}, OwnerResolved: map[string]bool{"1": true, "2": true}, Evidence: map[string][]SignalEvidence{
		"1": {
			{Signal: "active", Verdict: "held", Evidence: "provider active"},
			{Signal: "reachable", Verdict: "held", Evidence: "provider reachable"},
			{Signal: "corpus_eval_fresh", Verdict: "did_not_hold", Evidence: "direct pass rate below threshold"},
			{Signal: "eval_fresh", Verdict: "held", Evidence: "federated pass"},
		},
	}}
	svc := NewService(Deps{
		Reader: fakeReader{defs: map[Projection]*spacedoc.SpaceDefinition{ProjectionAnswer: answerDef()}},
		Joiner: staticJoiner{join},
		Clock:  fixedClock{},
	})
	report, err := svc.ValidateBaseDocs(context.Background(), ProjectionAnswer)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, issue := range report.Issues {
		if issue.Code == "eval_gate_unmet" {
			t.Fatalf("direct corpus debt must use the explicit quality-debt code: %+v", issue)
		}
		if issue.Code == "corpus_quality_debt" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected direct corpus failure to remain visible as quality debt: %+v", report.Issues)
	}
}

// validateDefFor builds a validate denominator with the given per-id statuses,
// so the graduation cross-check can be exercised deterministically.
func validateDefFor(statuses map[string]spacedoc.CellStatus) *spacedoc.SpaceDefinition {
	def := &spacedoc.SpaceDefinition{
		SchemaVersion:         "v1",
		Projection:            ProjectionValidate,
		Owner:                 "test-genie",
		DenominatorConfidence: spacedoc.ConfidenceAuthoritative,
		Source:                "scenarios/test-genie/docs/spaces/validate-space.md",
	}
	for id, st := range statuses {
		def.Cells = append(def.Cells, spacedoc.Cell{ID: id, Question: id + " concern", Status: st})
	}
	return def
}

func findIssue(issues []BaseDocIssue, code string) *BaseDocIssue {
	for i := range issues {
		if issues[i].Code == code {
			return &issues[i]
		}
	}
	return nil
}

func TestGraduationFindingsUngraduatedPointer(t *testing.T) {
	// Guide G32 (Observability) is MISSING; Validate V18 is a live (NOW) phase.
	guide := guideDef()
	guide.Cells = append(guide.Cells, spacedoc.Cell{ID: "G32", Question: "Observability", Owner: "_(none)_", Status: spacedoc.StatusMissing})
	validate := validateDefFor(map[string]spacedoc.CellStatus{"V18": spacedoc.StatusNow})
	links := []graduationLink{{"G32", "V18", "Observability / telemetry"}}

	got := graduationFindings(links, guide, validate)
	is := findIssue(got, "ungraduated_pointer")
	if is == nil {
		t.Fatalf("expected ungraduated_pointer, got %+v", got)
	}
	if is.Severity != SeverityWarn {
		t.Errorf("severity = %v, want WARN (must not gate)", is.Severity)
	}
	if is.Location != "scenarios/prompt-manager/docs/spaces/guide-space.md#G32" {
		t.Errorf("location = %q", is.Location)
	}
}

func TestGraduationFindingsConsistentIsSilent(t *testing.T) {
	// G2 COVERED (now) with a live V4 phase, and G31 MISSING with a NON-live V20
	// candidate — both consistent, so nothing should fire. Also fires-on-MISSING-
	// only: an in_reach guide row over a live phase is not flagged.
	guide := guideDef() // has G1 now, G2 now, G31 missing
	guide.Cells = append(guide.Cells, spacedoc.Cell{ID: "G26", Question: "Dependencies", Owner: "platform-package-hardening", Status: spacedoc.StatusInReach})
	validate := validateDefFor(map[string]spacedoc.CellStatus{
		"V4":  spacedoc.StatusNow,     // backs G2 (covered) — fine
		"V20": spacedoc.StatusMissing, // candidate, backs G31 (missing) — consistent gap
		"V5":  spacedoc.StatusNow,     // backs G26 (in_reach) — must NOT fire (missing-only)
	})
	links := []graduationLink{
		{"G2", "V4", "Architecture"},
		{"G31", "V20", "Concurrency"},
		{"G26", "V5", "Dependencies"},
	}
	if got := graduationFindings(links, guide, validate); len(got) != 0 {
		t.Errorf("expected no findings, got %+v", got)
	}
}

func TestGraduationFindingsUnresolvedRef(t *testing.T) {
	// A link to a Validate id that does not exist -> graduation_ref_unresolved.
	guide := guideDef() // has G2
	validate := validateDefFor(map[string]spacedoc.CellStatus{"V4": spacedoc.StatusNow})
	links := []graduationLink{{"G2", "V999", "Bogus"}}

	got := graduationFindings(links, guide, validate)
	is := findIssue(got, "graduation_ref_unresolved")
	if is == nil {
		t.Fatalf("expected graduation_ref_unresolved, got %+v", got)
	}
	if is.Severity != SeverityWarn {
		t.Errorf("severity = %v, want WARN", is.Severity)
	}
}

func TestProductionGraduationLinksResolveAgainstRealDocs(t *testing.T) {
	// Guards the curated cross-walk against doc renumbering: every linked id must
	// exist in the live space docs. Skips cleanly if the docs are unreachable in
	// this environment (the check is exercised on real data in the live run).
	reader := NewSpaceReader()
	guide, gerr := reader.Read(context.Background(), ProjectionGuide)
	validate, verr := reader.Read(context.Background(), ProjectionValidate)
	if gerr != nil || verr != nil {
		t.Skipf("space docs unreachable (guide=%v validate=%v)", gerr, verr)
	}
	for _, is := range graduationFindings(graduationLinks, guide, validate) {
		if is.Code == "graduation_ref_unresolved" {
			t.Errorf("stale graduation link: %s (%s)", is.Message, is.Location)
		}
	}
}

func TestSkillCount(t *testing.T) {
	cases := map[string]int{
		"explore":                                              1,
		"explore + the Answer projection":                      1,
		"`explore` + the Answer projection":                    1,
		"screaming-architecture-audit, architecture-scope":     2,
		"`screaming-architecture-audit`, `architecture-scope`": 2,
		"_(none)_": 0,
		"":         0,
		"react-coherence, react-stability, polish": 3,
	}
	for in, want := range cases {
		if got := skillCount(in); got != want {
			t.Errorf("skillCount(%q)=%d want %d", in, got, want)
		}
	}
}

func TestProviderTokens(t *testing.T) {
	toks := providerTokens("ui-health.surfaces + cli-health.commands + code-facts (API)")
	want := map[string]bool{"ui-health.surfaces": true, "cli-health.commands": true, "code-facts": true}
	for _, tk := range toks {
		if !want[tk] {
			t.Errorf("unexpected token %q (got %v)", tk, toks)
		}
	}
	if len(providerTokens("_(none)_")) != 0 {
		t.Error("none should yield no tokens")
	}
}

type fixedClock struct{}

func (fixedClock) Now() time.Time                            { return time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC) }
func (fixedClock) NewTimer(d time.Duration) schedule.Timer   { return schedule.System().NewTimer(d) }
func (fixedClock) NewTicker(d time.Duration) schedule.Ticker { return schedule.System().NewTicker(d) }
func (fixedClock) Sleep(time.Duration)                       {}
