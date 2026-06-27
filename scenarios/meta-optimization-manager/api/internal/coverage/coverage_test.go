package coverage

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

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

// staticJoiner returns a fixed JoinResult.
type staticJoiner struct{ res JoinResult }

func (j staticJoiner) Join(context.Context, Projection, []spacedoc.Cell) JoinResult { return j.res }

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
}

func TestNumeratorJoinRecomputesAnswer(t *testing.T) {
	// search-hub providers list returns ui-health live but NOT cartographer.
	runner := func(_ context.Context, name string, args ...string) ([]byte, error) {
		return []byte(`{"providers":[{"provider_id":"ui-health.surfaces"},{"id":"code-facts"}]}`), nil
	}
	j := NewNumeratorJoinerWithRunner(runner)
	res := j.Join(context.Background(), ProjectionAnswer, answerDef().Cells)
	if !res.Available {
		t.Fatal("expected available")
	}
	// cell 1 (ui-health.surfaces) stays NOW.
	if res.Statuses["1"] != spacedoc.StatusNow {
		t.Errorf("cell1 = %v", res.Statuses["1"])
	}
	// cell 2 authored IN_REACH; cartographer not live -> keeps authored (no entry).
	if st, ok := res.Statuses["2"]; ok && st == spacedoc.StatusNow {
		t.Errorf("cell2 should not be promoted to NOW: %v", st)
	}
}

func TestNumeratorJoinDegradesOnError(t *testing.T) {
	runner := func(context.Context, string, ...string) ([]byte, error) { return nil, errors.New("down") }
	res := NewNumeratorJoinerWithRunner(runner).Join(context.Background(), ProjectionAnswer, answerDef().Cells)
	if res.Available {
		t.Error("expected unavailable")
	}
	if res.Reason == "" {
		t.Error("expected reason")
	}
}

func TestAnswerDriftDowngrade(t *testing.T) {
	// Authored NOW cell 1, but live providers does NOT include ui-health -> downgrade.
	runner := func(context.Context, string, ...string) ([]byte, error) {
		return []byte(`{"providers":[{"provider_id":"code-facts"}]}`), nil
	}
	res := NewNumeratorJoinerWithRunner(runner).Join(context.Background(), ProjectionAnswer, answerDef().Cells)
	if res.Statuses["1"] != spacedoc.StatusInReach {
		t.Errorf("cell1 authored-NOW with dead provider should downgrade to in_reach, got %v", res.Statuses["1"])
	}
}

func TestNumeratorJoinRecomputesValidate(t *testing.T) {
	raw := []byte(`{
		"selfHealth": {
			"catalog": {"phases": [
				{"provider":"green-health"},
				{"provider":"red-health"},
				{"provider":"autofix-health"}
			]},
			"ledger": {"phases": [
				{"provider":"green-health","failureRate":0},
				{"provider":"red-health","failureRate":0.25},
				{"provider":"autofix-health","failureRate":0}
			]},
			"conformance": [
				{"provider":"green-health","autofix":{"pending":0}},
				{"provider":"red-health","autofix":{"pending":0}},
				{"provider":"autofix-health","autofix":{"pending":2}}
			]
		}
	}`)
	runner := func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name != "test-genie" || len(args) != 2 || args[0] != "health" || args[1] != "--json" {
			t.Fatalf("unexpected registry read: %s %v", name, args)
		}
		return raw, nil
	}
	cells := []spacedoc.Cell{
		{ID: "V1", Owner: "`green-health`", Status: spacedoc.StatusInReach},
		{ID: "V2", Owner: "`red-health`", Status: spacedoc.StatusNow},
		{ID: "V3", Owner: "`autofix-health`", Status: spacedoc.StatusNow},
		{ID: "V4", Owner: "`unknown-health`", Status: spacedoc.StatusMissing},
	}
	res := NewNumeratorJoinerWithRunner(runner).Join(context.Background(), ProjectionValidate, cells)
	if !res.Available {
		t.Fatalf("expected available: %s", res.Reason)
	}
	if got := res.Statuses["V1"]; got != spacedoc.StatusNow {
		t.Errorf("green phase = %v, want now", got)
	}
	if got := res.Statuses["V2"]; got != spacedoc.StatusInReach {
		t.Errorf("red phase = %v, want in_reach", got)
	}
	if got := res.Statuses["V3"]; got != spacedoc.StatusInReach {
		t.Errorf("autofix-pending phase = %v, want in_reach", got)
	}
	if _, ok := res.Statuses["V4"]; ok {
		t.Errorf("unknown phase should keep authored status, got overlay %v", res.Statuses["V4"])
	}
}

func TestNumeratorJoinRecomputesGuide(t *testing.T) {
	raw := []byte(`[
		{"nodeId":"explore","score":0.9},
		{"nodeId":"polish","score":0.4},
		{"nodeId":"architecture-scope","score":0.8},
		{"nodeId":"screaming-architecture-audit","score":0.7}
	]`)
	runner := func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name != "prompt-manager" || len(args) != 3 || args[0] != "graph" || args[1] != "health" || args[2] != "--json" {
			t.Fatalf("unexpected registry read: %s %v", name, args)
		}
		return raw, nil
	}
	cells := []spacedoc.Cell{
		{ID: "G1", Owner: "`explore` + the Answer projection", Status: spacedoc.StatusInReach},
		{ID: "G2", Owner: "`screaming-architecture-audit`, `architecture-scope`", Status: spacedoc.StatusInReach},
		{ID: "G3", Owner: "`polish`", Status: spacedoc.StatusNow},
		{ID: "G4", Owner: "`explore`, `missing-skill`", Status: spacedoc.StatusNow},
		{ID: "G5", Owner: "`missing-skill`", Status: spacedoc.StatusMissing},
	}
	res := NewNumeratorJoinerWithRunner(runner).Join(context.Background(), ProjectionGuide, cells)
	if !res.Available {
		t.Fatalf("expected available: %s", res.Reason)
	}
	if got := res.Statuses["G1"]; got != spacedoc.StatusNow {
		t.Errorf("single-word healthy skill = %v, want now", got)
	}
	if got := res.Statuses["G2"]; got != spacedoc.StatusNow {
		t.Errorf("multi-skill healthy row = %v, want now", got)
	}
	if got := res.Statuses["G3"]; got != spacedoc.StatusInReach {
		t.Errorf("unhealthy skill = %v, want in_reach", got)
	}
	if got := res.Statuses["G4"]; got != spacedoc.StatusInReach {
		t.Errorf("partially resolved row = %v, want in_reach", got)
	}
	if _, ok := res.Statuses["G5"]; ok {
		t.Errorf("unresolved guide row should keep authored status, got overlay %v", res.Statuses["G5"])
	}
}

func TestCapturedRegistryFixturesStillMatch(t *testing.T) {
	testGenieRaw := readCoverageTestdata(t, "test_genie_health.json")
	validateIndex := validateStatusIndex(testGenieRaw)
	if _, ok := validateIndex["structure-health"]; !ok {
		t.Fatal("captured test-genie health fixture no longer exposes structure-health")
	}
	if !validateIndex["storage-health"].autofixPending {
		t.Fatal("captured test-genie health fixture should expose storage-health pending autofix work")
	}

	promptManagerRaw := readCoverageTestdata(t, "pm_graph_health.json")
	guideScores := guideScoreIndex(promptManagerRaw)
	for _, skill := range []string{"explore", "idea-workshop", "performance", "polish"} {
		if _, ok := guideScores[skill]; !ok {
			t.Fatalf("captured prompt-manager graph health fixture no longer exposes %q", skill)
		}
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
	svc := NewService(Deps{
		Reader: fakeReader{defs: map[Projection]*spacedoc.SpaceDefinition{ProjectionAnswer: answerDef()}},
		Joiner: staticJoiner{JoinResult{Available: true}},
		Clock:  fixedClock{},
	})
	cell, err := svc.ExplainCell(context.Background(), "answer/1")
	if err != nil {
		t.Fatal(err)
	}
	if cell.Question != "Surfaces" || len(cell.Citations) == 0 {
		t.Errorf("cell = %+v", cell)
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

func readCoverageTestdata(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

type fixedClock struct{}

func (fixedClock) Now() time.Time { return time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC) }
