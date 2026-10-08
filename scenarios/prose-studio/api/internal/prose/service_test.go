package prose

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/textmetrics"
	_ "modernc.org/sqlite"
)

type fakeGateway struct{ calls []GatewayRequest }

func (f *fakeGateway) Generate(_ context.Context, req GatewayRequest) ([]GatewayCandidate, error) {
	callIndex := len(f.calls)
	f.calls = append(f.calls, req)
	if strings.HasPrefix(req.Query, "Create a concise ordered outline") {
		return []GatewayCandidate{{Text: `[{"intent":"opening","summary":"Establish the central problem and promise.","target_words":120},{"intent":"evidence","summary":"Show the evidence and explain its significance.","target_words":180},{"intent":"conclusion","summary":"Close with the practical implication for the reader.","target_words":100}]`, Provider: "fake", Model: "fake-model", TemperatureSupport: "supported", CostMicros: 100}}, nil
	}
	out := make([]GatewayCandidate, req.K)
	for i := range out {
		seed := []string{"A crisp opening about trust.", "A vivid opening about craft.", "A practical opening about proof.", "A reflective opening about change.", "A measured opening about care."}[(callIndex+i)%5]
		out[i] = GatewayCandidate{Text: padToRequestedBand(req.Query, seed), Provider: "fake", Model: "fake-model", TemperatureSupport: "supported", CostMicros: 100, HintOrdinal: i + 1}
	}
	return out, nil
}

var bandPattern = regexp.MustCompile(`between (\d+) and (\d+) words`)

// padToRequestedBand makes the fixture answer the length contract the section
// prompt states. The previous fixture returned a five-word sentence for every
// section and every test passed, because the section gate carried no word floor
// at all — the fixture was only honest by accident of a missing constraint.
func padToRequestedBand(query, seed string) string {
	match := bandPattern.FindStringSubmatch(query)
	if match == nil {
		return seed
	}
	low, _ := strconv.Atoi(match[1])
	high, _ := strconv.Atoi(match[2])
	target := (low + high) / 2
	words := strings.Fields(seed)
	for len(words) < target {
		words = append(words, "filler")
	}
	return strings.Join(words[:target], " ") + "."
}

func TestRarestPolicySelectsNonFirstCandidate(t *testing.T) {
	set := textmetrics.SetMetrics{PairwiseSimilarity: [][]float64{{0, .9, .2}, {.9, 0, .1}, {.2, .1, 0}}}
	candidates := []Candidate{{ID: "first", SetIndex: 0, SetMeasurements: set, Eligibility: Eligibility{Eligible: true}}, {ID: "middle", SetIndex: 1, SetMeasurements: set, Eligibility: Eligibility{Eligible: true}}, {ID: "rare", SetIndex: 2, SetMeasurements: set, Eligibility: Eligibility{Eligible: true}}}
	selected := choose(candidates, "threshold_then_rarest", map[string]float64{"threshold": .5}, 1, nil)
	require.NotNil(t, selected)
	require.Equal(t, "rare", selected.ID, "rarest policy must read the candidate's own similarity row")
}

func TestRarestPolicyHonoursThreshold(t *testing.T) {
	set := textmetrics.SetMetrics{PairwiseSimilarity: [][]float64{{0, .9, .2}, {.9, 0, .1}, {.2, .1, 0}}}
	candidates := []Candidate{{ID: "first", SetIndex: 0, SetMeasurements: set, Eligibility: Eligibility{Eligible: true}}, {ID: "middle", SetIndex: 1, SetMeasurements: set, Eligibility: Eligibility{Eligible: true}}, {ID: "rare", SetIndex: 2, SetMeasurements: set, Eligibility: Eligibility{Eligible: true}}}
	selected := choose(candidates, "threshold_then_rarest", map[string]float64{"threshold": .95}, 1, nil)
	require.NotNil(t, selected)
	require.Equal(t, "first", selected.ID, "when no candidate clears the threshold, policy must fall back to the eligible set")
}

func TestUniformPolicyVariesWithSeed(t *testing.T) {
	candidates := []Candidate{{ID: "a", Eligibility: Eligibility{Eligible: true}}, {ID: "b", Eligibility: Eligibility{Eligible: true}}, {ID: "c", Eligibility: Eligibility{Eligible: true}}}
	first := choose(candidates, "sample_uniform", nil, 1, nil)
	second := choose(candidates, "sample_uniform", nil, 2, nil)
	require.NotEqual(t, first.ID, second.ID, "uniform policy must use the persisted seed rather than a fixed middle element")
}

func TestInstructionCarriesExemplars(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	_, err := s.CreateStyle(ctx, Style{Key: "exemplar-voice", Exemplars: []string{"Short, concrete sentences.", "Show the evidence before the claim."}})
	require.NoError(t, err)
	_, err = s.CreateProfile(ctx, Profile{Key: "exemplar-profile", StyleRefs: []string{"exemplar-voice"}})
	require.NoError(t, err)
	resolved, err := s.ResolveProfile(ctx, "exemplar-profile")
	require.NoError(t, err)
	require.Contains(t, resolved.InstructionText, "Short, concrete sentences.")
	require.Contains(t, resolved.InstructionText, "Show the evidence before the claim.")
}

func TestInstructionCarriesLexiconAndTargets(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	_, err := s.CreateStyle(ctx, Style{Key: "lexicon-voice", Lexicon: []string{"proof", "builder"}, Targets: map[string]float64{"mattr": .7}})
	require.NoError(t, err)
	_, err = s.CreateProfile(ctx, Profile{Key: "lexicon-profile", StyleRefs: []string{"lexicon-voice"}})
	require.NoError(t, err)
	resolved, err := s.ResolveProfile(ctx, "lexicon-profile")
	require.NoError(t, err)
	require.Contains(t, resolved.InstructionText, "proof")
	require.Contains(t, resolved.InstructionText, "builder")
	require.Contains(t, resolved.InstructionText, "Target mattr")
}

func TestCoherenceVariesWithInput(t *testing.T) {
	repetitive := textmetrics.CrossSectionRepetition([]string{"proof builds trust", "proof builds trust"})
	different := textmetrics.CrossSectionRepetition([]string{"proof builds trust", "orchards change slowly"})
	require.NotEqual(t, repetitive, different)
}

func TestFeasibilityRefusesOversizedProfile(t *testing.T) {
	s, _ := newTestService(t)
	_, err := s.CreateProfile(context.Background(), Profile{Key: "oversized", Budget: Budget{MaxOutputTokens: 100}, ContextPolicy: ContextPolicy{FullTextTokenBudget: 80, SummarizeBeyond: 80, DeclaredContextCeiling: 200}})
	require.ErrorIs(t, err, ErrContextInfeasible)
}

func TestGateEnforcesReadingGrade(t *testing.T) {
	measurement := textmetrics.Analyze("A very long sentence with many words that should produce a measurable reading grade for this constraint.", nil)
	result := gate(measurement, Constraints{MaxGrade: 1}, "A very long sentence with many words that should produce a measurable reading grade for this constraint.", gateInputs{})
	require.False(t, result.Eligible)
	require.Contains(t, result.Reason, "max_grade")
}

func TestGateEnforcesRequiredFormat(t *testing.T) {
	measurement := textmetrics.Analyze("plain prose", nil)
	result := gate(measurement, Constraints{RequiredFormat: "bullet_list"}, "plain prose", gateInputs{})
	require.False(t, result.Eligible)
	require.Contains(t, result.Reason, "required_format")
}

func TestConformanceReportsEveryDeclaredTarget(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	_, err := s.CreateStyle(ctx, Style{Key: "targeted", Targets: map[string]float64{"mattr": .1, "type_token_ratio": .1, "flesch_kincaid": 0}})
	require.NoError(t, err)
	report, err := s.Conformance(ctx, "targeted", "Proof builds trust.")
	require.NoError(t, err)
	verdicts := report["verdicts"].(map[string]map[string]any)
	require.Len(t, verdicts, 3)
}

func TestOutputCapScalesWithCandidateCount(t *testing.T) {
	base := Profile{Budget: Budget{MaxOutputTokens: 100}, Sampler: Sampler{Kind: samplerDirect, K: 1}}
	set := base
	set.Sampler = Sampler{Kind: samplerVSStandard, K: 10}
	require.Greater(t, effectiveOutputCap(set), effectiveOutputCap(base))
}

type shortGateway struct{}

func (shortGateway) Generate(context.Context, GatewayRequest) ([]GatewayCandidate, error) {
	return []GatewayCandidate{{Text: "one candidate", Provider: "fake", Model: "fake", MaxOutputTokensEffective: 100, MaxOutputTokensSource: "request"}}, nil
}

func TestShortSetDegradesWithNamedReason(t *testing.T) {
	db, err := sql.Open("sqlite", "file:prose-short-test?mode=memory&cache=shared")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, EnsureSchema(db))
	s := NewWithGateway(db, shortGateway{})
	ctx := context.Background()
	_, err = s.CreateProfile(ctx, Profile{Key: "short", Sampler: Sampler{Kind: samplerVSStandard, K: 3}, Budget: Budget{MaxOutputTokens: 100}})
	require.NoError(t, err)
	out, err := s.Generate(ctx, GenerateRequest{ProfileKey: "short", Query: "q", IncludeCandidates: true})
	require.NoError(t, err)
	require.NotNil(t, out.Degraded)
	require.Equal(t, "short_candidate_set", out.Degraded.Kind)
}

func TestSectionCommitWritesCommittedCandidate(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	_, err := s.CreateProfile(ctx, Profile{Key: "section", Sampler: Sampler{Kind: samplerDirect, K: 1}})
	require.NoError(t, err)
	out, err := s.Generate(ctx, GenerateRequest{ProfileKey: "section", Query: "opening", IncludeCandidates: true})
	require.NoError(t, err)
	section := Section{ID: "section-1", DocumentID: "doc-1", ProfileKey: "section", SessionID: out.Session.ID}
	require.NoError(t, s.commitSectionCandidate(ctx, section.ID, out.Candidates[0].ID, section))
	var stored Section
	require.NoError(t, s.loadJSON(ctx, "prose_sections", section.ID, &stored))
	require.Equal(t, out.Candidates[0].ID, stored.CommittedCandidateID)
}

func TestContextSnapshotNamesItsInputs(t *testing.T) {
	s, _ := newTestService(t)
	snapshot := s.buildContextSnapshot(context.Background(), Document{OutlineID: "outline-1"}, []Section{{ID: "section-1"}}, []string{"proof", "conclusion"})
	require.Equal(t, "outline-1", snapshot.OutlineRef)
	require.Equal(t, []string{"section-1"}, snapshot.PriorSectionRefs)
	require.Equal(t, []string{"proof", "conclusion"}, snapshot.FollowingIntents)
}

func TestAssembleSectionContextCarriesLongFormInputsWithoutIdentifiers(t *testing.T) {
	s, _ := newTestService(t)
	contextText, err := s.assembleSectionContext(context.Background(), Document{Title: "A durable title"}, "Opening\nEvidence\nClose", Section{Position: 1, Intent: "Evidence", Summary: "Show the measured result", TargetWords: 180}, 3, nil)
	require.NoError(t, err)
	require.Contains(t, contextText, "A durable title")
	require.Contains(t, contextText, "Opening\nEvidence\nClose")
	require.Contains(t, contextText, "Current section: 2 of 3")
	require.Contains(t, contextText, "Length target: approximately 180 words")
	require.NotRegexp(t, recordIdentifierPattern, contextText)
}

func TestStructuredSingleCandidatePreservesDeclaredJSON(t *testing.T) {
	texts, _, err := decodeCandidates(GatewayRequest{K: 1, SchemaJSON: outlineSchema}, `[{"intent":"opening","summary":"hook","target_words":100}]`)
	require.NoError(t, err)
	require.Equal(t, `[{"intent":"opening","summary":"hook","target_words":100}]`, texts[0])
}

func TestCreateDocumentOwnsOutlineAndSectionSessions(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	_, err := s.CreateProfile(ctx, Profile{Key: "long-form", Sampler: Sampler{Kind: samplerDirect, K: 1}, Budget: Budget{MaxOutputTokens: 128}, Composition: CompositionPolicy{SectionCount: 3}})
	require.NoError(t, err)

	doc, err := s.CreateDocument(ctx, Document{Title: "A measured document", ProfileKey: "long-form"}, nil)
	require.NoError(t, err)
	require.Equal(t, "assembled", doc.Status)
	require.Len(t, doc.SectionIDs, 3)
	require.NotEmpty(t, doc.OutlineID)
	for _, section := range doc.Sections {
		require.NotEmpty(t, section.SessionID)
		require.NotEmpty(t, section.CommittedCandidateID)
		require.Equal(t, doc.OutlineID, section.Context.OutlineRef)
	}
}

func newTestService(t *testing.T) (*Service, *fakeGateway) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:prose-test?mode=memory&cache=shared")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, EnsureSchema(db))
	fake := &fakeGateway{}
	return NewWithGateway(db, fake), fake
}

func TestStyleResolutionDetectsCyclesAndConflicts(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	_, err := s.CreateStyle(ctx, Style{Key: "base", Directives: []string{"plain"}, Targets: map[string]float64{"mattr": 0.5}})
	require.NoError(t, err)
	_, err = s.CreateStyle(ctx, Style{Key: "voice", Parent: "base", Targets: map[string]float64{"mattr": 0.5}})
	require.NoError(t, err)
	_, err = s.CreateStyle(ctx, Style{Key: "base", Parent: "voice"})
	require.Error(t, err)
	_, err = s.CreateStyle(ctx, Style{Key: "conflict", Parent: "base", Targets: map[string]float64{"mattr": 0.9}})
	require.NoError(t, err)
	_, err = s.ResolveProfile(ctx, "missing")
	if err == nil {
		t.Fatal("missing profile unexpectedly resolved")
	}
	_, err = s.resolveStyle(ctx, "conflict", map[string]bool{})
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrStyleResolutionConflict), err)
}

func TestGenerateMeasuresSetContainsUncalibratedHintsAndAttributesCost(t *testing.T) {
	s, gateway := newTestService(t)
	ctx := context.Background()
	_, err := s.CreateStyle(ctx, Style{Key: "voice", Directives: []string{"use concrete verbs"}})
	require.NoError(t, err)
	_, err = s.CreateProfile(ctx, Profile{Key: "blog", StyleRefs: []string{"voice"}, Sampler: Sampler{Kind: "vs_standard", K: 3, Tau: .7, TemperatureStance: "ignored"}, SelectionPolicy: "threshold_then_rarest", GatewayRole: "write.diverse", Budget: Budget{MaxOutputTokens: 512}})
	require.NoError(t, err)
	out, err := s.Generate(ctx, GenerateRequest{ProfileKey: "blog", Query: "why governed writing matters", IncludeCandidates: true})
	require.NoError(t, err)
	require.Len(t, out.Candidates, 3)
	require.Equal(t, 1, len(gateway.calls))
	require.Equal(t, "write.diverse", gateway.calls[0].Role)
	var cost int64
	for _, c := range out.Candidates {
		require.NotNil(t, c.VerbalizedHint)
		require.False(t, c.VerbalizedHint.Calibrated)
		require.NotNil(t, c.Measurements)
		require.True(t, c.Provenance.MachineGenerated)
		cost += c.Provenance.CostMicros
	}
	require.Equal(t, out.Round.TotalCostMicros, cost)
	resolved, err := s.ResolveProfile(ctx, "blog")
	require.NoError(t, err)
	require.Contains(t, resolved.InstructionText, "use concrete verbs")
	if out.Selected != nil {
		_, err = s.SessionAction(ctx, "commit", out.Session.ID, out.Selected.ID)
		require.NoError(t, err)
		_, err = s.CreateStyle(ctx, Style{Key: "voice", Directives: []string{"new version"}})
		require.Error(t, err, "committed style reference must be immutable")
	}
}

func TestSemanticProfileRecordsLexicalFallbackWhenGatewayCannotEmbed(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	require.NoError(t, func() error {
		_, err := s.CreateProfile(ctx, Profile{Key: "semantic", MeasurementTiers: []string{"deterministic_and_semantic"}, Sampler: Sampler{Kind: samplerVSStandard, K: 2}, Budget: Budget{MaxOutputTokens: 100}})
		return err
	}())
	out, err := s.Generate(ctx, GenerateRequest{ProfileKey: "semantic", Query: "subject", IncludeCandidates: true})
	require.NoError(t, err)
	require.Contains(t, out.Round.MeasurementBasis, "lexical")
	require.Contains(t, out.Round.MeasurementFallback, "embedding_unavailable")
	require.NotEmpty(t, out.Round.LexicalSetMeasurements)
}

func TestConformanceSeparatesAntiPatternAndPreferredSpansAndHonoursComparator(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	_, err := s.CreateStyle(ctx, Style{Key: "marketing", AntiPatterns: []string{"hype drift"}, Lexicon: []string{"evidence"}, Targets: map[string]float64{"flesch_kincaid_grade_max": 20}, TargetDirections: map[string]string{"flesch_kincaid_grade_max": "at_most"}})
	require.NoError(t, err)
	report, err := s.Conformance(ctx, "marketing", "Evidence supports this grounded claim. Avoid hype drift.")
	require.NoError(t, err)
	require.NotEmpty(t, report["anti_pattern_spans"])
	require.NotEmpty(t, report["preferred_lexicon_spans"])
	verdicts := report["verdicts"].(map[string]map[string]any)
	require.Equal(t, "at_most", verdicts["flesch_kincaid_grade_max"]["direction"])
}

func TestTransformsRecordSourceAndAxisCellsArePlanned(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	_, err := s.CreateProfile(ctx, Profile{Key: "transform", Sampler: Sampler{Kind: samplerDirect, K: 1}, Budget: Budget{MaxOutputTokens: 100}})
	require.NoError(t, err)
	out, err := s.Generate(ctx, GenerateRequest{ProfileKey: "transform", Query: "subject", IncludeCandidates: true})
	require.NoError(t, err)
	derived, err := s.TransformCandidate(ctx, out.Candidates[0].ID, "reading_level", map[string]any{"target_grade": 8})
	require.NoError(t, err)
	require.Equal(t, []string{out.Candidates[0].ID}, derived.DerivedFrom)
	require.Equal(t, out.Candidates[0].ID, derived.Transform.SourceCandidate)
	space := AxisSpace{Key: "marketing", Axes: []Axis{{Name: "audience", Variants: []string{"builder", "operator"}}, {Name: "framing", Variants: []string{"evidence", "lesson"}}}}
	cells := PlanAxisCells(space)
	require.Len(t, cells, 4)
	require.Len(t, ComputeCellCoverage(space, []Candidate{{AxisCell: cells[0].Key}}).Missed, 3)
}

func TestCompositeGenerationCoversEveryAxisCell(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	_, err := s.CreateProfile(ctx, Profile{Key: "composite", Sampler: Sampler{Kind: samplerComposite, K: 1}, Budget: Budget{MaxOutputTokens: 100}})
	require.NoError(t, err)
	space := AxisSpace{Key: "marketing", Axes: []Axis{{Name: "audience", Variants: []string{"builder", "operator"}}, {Name: "framing", Variants: []string{"evidence", "lesson"}}}}
	result, err := s.GenerateComposite(ctx, GenerateRequest{ProfileKey: "composite", Query: "subject"}, space)
	require.NoError(t, err)
	require.Len(t, result.Candidates, 4)
	require.Len(t, result.Coverage.Planned, 4)
	require.Len(t, result.Coverage.Covered, 4)
	require.Empty(t, result.Coverage.Missed)
	for _, candidate := range result.Candidates {
		require.NotEmpty(t, candidate.AxisCell)
	}
}

func TestDeclarationsAreFileAuthorityAndDeletedFilesBecomeUnregistered(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	root := t.TempDir()
	dir := filepath.Join(root, ".vrooli", "prose-studio")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	content := []byte(`{"schema_version":"1","kind":"profile","key":"content/blog","created_by":"content-desk","record":{"sampler":{"kind":"direct","k":1},"gateway_role":"write.default","budget":{"max_output_tokens":256}}}`)
	first := filepath.Join(dir, "blog.json")
	second := filepath.Join(dir, "collision.json")
	require.NoError(t, os.WriteFile(first, content, 0o644))
	require.NoError(t, os.WriteFile(second, content, 0o644))
	declarations, err := s.Reindex(ctx, root)
	require.NoError(t, err)
	require.Len(t, declarations, 2)
	require.Contains(t, []string{declarations[0].Status, declarations[1].Status}, "collision")
	_, err = s.CreateProfile(ctx, Profile{Key: "content/blog"})
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrProfileDeclared), err)
	require.NoError(t, os.Remove(first))
	declarations, err = s.Reindex(ctx, root)
	require.NoError(t, err)
	foundUnregistered := false
	for _, d := range declarations {
		if d.Path == first && d.Status == "unregistered" {
			foundUnregistered = true
		}
	}
	require.True(t, foundUnregistered)
}

func TestRegistryIsIntrospectable(t *testing.T) {
	s, _ := newTestService(t)
	registry := s.Registry()
	require.NotEmpty(t, registry.Samplers)
	require.NotEmpty(t, registry.Policies)
	for _, policy := range registry.Policies {
		require.NotNil(t, policy.ParameterSchema)
	}
}

func TestRerollPreservesSessionAndSendsNegativeContext(t *testing.T) {
	s, gateway := newTestService(t)
	ctx := context.Background()
	_, err := s.CreateStyle(ctx, Style{Key: "voice", Directives: []string{"use concrete verbs"}})
	require.NoError(t, err)
	_, err = s.CreateProfile(ctx, Profile{Key: "reroll", StyleRefs: []string{"voice"}, Sampler: Sampler{Kind: "direct", K: 3}, Budget: Budget{MaxOutputTokens: 128}})
	require.NoError(t, err)
	first, err := s.Generate(ctx, GenerateRequest{ProfileKey: "reroll", Query: "write an opening", IncludeCandidates: true})
	require.NoError(t, err)
	_, err = s.SessionAction(ctx, "pin", first.Session.ID, first.Candidates[0].ID)
	require.NoError(t, err)
	second, err := s.Reroll(ctx, first.Session.ID, true)
	require.NoError(t, err)
	require.Len(t, second.Candidates, 2)
	require.Equal(t, []string{first.Candidates[0].Text}, second.Round.NegativeContext.Pinned)
	require.Equal(t, second.Round.NegativeContext.Pinned, gateway.calls[1].Negative.Pinned)
}

func TestLongFormAssemblyUsesOrderedCommittedSectionsAndCoherence(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	require.NoError(t, func() error {
		_, err := s.CreateStyle(ctx, Style{Key: "blog-voice", Directives: []string{"use concrete verbs"}})
		return err
	}())
	require.NoError(t, func() error {
		_, err := s.CreateProfile(ctx, Profile{Key: "blog-profile", StyleRefs: []string{"blog-voice"}, Sampler: Sampler{Kind: "direct", K: 1}, Budget: Budget{MaxOutputTokens: 256}})
		return err
	}())
	doc, err := s.CreateDocument(ctx, Document{Title: "Governed writing", ProfileKey: "blog-profile", StyleKey: "blog-voice", OutlineID: "outline-1"}, []Section{
		{Position: 0, Intent: "opening", Context: ContextSnapshot{OutlineRef: "outline-1", EstimatedTokens: 80}},
		{Position: 1, Intent: "proof", Context: ContextSnapshot{OutlineRef: "outline-1", PriorSectionRefs: []string{"section-0"}, EstimatedTokens: 120}},
	})
	require.NoError(t, err)
	require.Len(t, doc.SectionIDs, 2)
	first, err := s.Generate(ctx, GenerateRequest{ProfileKey: "blog-profile", Query: "opening", IncludeCandidates: true})
	require.NoError(t, err)
	second, err := s.Generate(ctx, GenerateRequest{ProfileKey: "blog-profile", Query: "proof", IncludeCandidates: true})
	require.NoError(t, err)
	var firstSection, secondSection Section
	require.NoError(t, s.loadJSON(ctx, "prose_sections", doc.SectionIDs[0], &firstSection))
	require.NoError(t, s.loadJSON(ctx, "prose_sections", doc.SectionIDs[1], &secondSection))
	firstSection.CommittedCandidateID = first.Candidates[0].ID
	secondSection.CommittedCandidateID = second.Candidates[0].ID
	require.NoError(t, s.saveSection(ctx, firstSection))
	require.NoError(t, s.saveSection(ctx, secondSection))
	assembled, err := s.AssembleDocument(ctx, doc.ID)
	require.NoError(t, err)
	require.Equal(t, "assembled", assembled.Status)
	require.Contains(t, assembled.AssembledText, first.Candidates[0].Text)
	require.Contains(t, assembled.AssembledText, second.Candidates[0].Text)
	require.Less(t, strings.Index(assembled.AssembledText, first.Candidates[0].Text), strings.Index(assembled.AssembledText, second.Candidates[0].Text))
	require.Contains(t, assembled.Coherence.(map[string]any)["basis"], "textmetrics.CrossSectionRepetition")
}

func TestVerbalizedDecodeRanksByProbabilityAndKeepsEmissionOrder(t *testing.T) {
	req := GatewayRequest{Strategy: samplerVSStandard, K: 3}
	value := `[{"text":"tail reading","probability":0.05},{"text":"modal reading","probability":0.60},{"text":"middle reading","probability":0.20}]`

	texts, ordinals, err := decodeCandidates(req, value)
	require.NoError(t, err)

	// Emission order is preserved: reordering the set by the model's own
	// probability would make a quality proxy the presentation order.
	require.Equal(t, []string{"tail reading", "modal reading", "middle reading"}, texts)
	// Rank 1 is the highest verbalized probability, wherever it was emitted.
	require.Equal(t, []int{3, 1, 2}, ordinals)
}

func TestVerbalizedDecodeRejectsMalformedSets(t *testing.T) {
	req := GatewayRequest{Strategy: samplerVSStandard, K: 2}

	_, _, err := decodeCandidates(req, `[{"text":"  ","probability":0.1}]`)
	require.ErrorIs(t, err, ErrMalformedCandidateSet)

	_, _, err = decodeCandidates(req, `[{"text":"fine","probability":1.4}]`)
	require.ErrorIs(t, err, ErrMalformedCandidateSet)

	_, _, err = decodeCandidates(req, `[]`)
	require.Error(t, err)
}

func TestDirectDecodeClaimsNoOrderingSignal(t *testing.T) {
	// A strategy that elicits no probability must report ordinal zero rather
	// than a positional index dressed up as the model's ordering.
	texts, ordinals, err := decodeCandidates(GatewayRequest{Strategy: samplerDirect, K: 2}, `["one","two"]`)
	require.NoError(t, err)
	require.Equal(t, []string{"one", "two"}, texts)
	require.Equal(t, []int{0, 0}, ordinals)
}

func TestVerbalizedSchemaAndInstructionCarryCountAndThreshold(t *testing.T) {
	require.Equal(t, vsCandidateSchema, gatewaySchema(GatewayRequest{Strategy: samplerVSStandard, K: 5}))
	require.Contains(t, vsCandidateSchema, `"probability"`)

	instruction := verbalizedInstruction(5, 0.10)
	// The count and threshold cannot ride in the schema subset, so they must be
	// in words or the strategy is a label rather than a technique.
	require.Contains(t, instruction, "Return 5 substantially different responses")
	require.Contains(t, instruction, "below 0.10")
	require.Contains(t, instruction, "Do not make a candidate strange for the sake of novelty")
}

func TestProfileRefusesUnexercisableVerbalizedConfiguration(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	_, err := s.CreateStyle(ctx, Style{Key: "vs-voice", Directives: []string{"plain"}})
	require.NoError(t, err)

	_, err = s.CreateProfile(ctx, Profile{Key: "vs-single", StyleRefs: []string{"vs-voice"}, Sampler: Sampler{Kind: samplerVSStandard, K: 1}})
	require.ErrorContains(t, err, "requires k >= 2")

	_, err = s.CreateProfile(ctx, Profile{Key: "vs-tau", StyleRefs: []string{"vs-voice"}, Sampler: Sampler{Kind: samplerVSStandard, K: 3, Tau: 1.5}})
	require.ErrorContains(t, err, "tau must fall in [0,1]")

	_, err = s.CreateProfile(ctx, Profile{Key: "vs-locality", StyleRefs: []string{"vs-voice"}, Sampler: Sampler{Kind: samplerVSStandard, K: 3, Tau: 0.1}, Locality: "nowhere"})
	require.ErrorIs(t, err, ErrUnknownLocality)
}

// declaredProfile writes one valid declaration into root's declaration dir.
func declaredProfile(t *testing.T, root, file, key string) string {
	t.Helper()
	dir := filepath.Join(root, ".vrooli", "prose-studio")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	body := `{"schema_version":"1","kind":"profile","key":"` + key + `","created_by":"content-desk","record":{"sampler":{"kind":"direct","k":1},"gateway_role":"write.default","budget":{"max_output_tokens":256}}}`
	path := filepath.Join(dir, file)
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
	return path
}

func TestReindexNeverUnregistersAnotherConsumersDeclarations(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	rootA, rootB := t.TempDir(), t.TempDir()
	pathA := declaredProfile(t, rootA, "a.json", "consumer-a/blog")
	pathB := declaredProfile(t, rootB, "b.json", "consumer-b/blog")

	_, err := s.Reindex(ctx, rootA)
	require.NoError(t, err)
	_, err = s.Reindex(ctx, rootB)
	require.NoError(t, err)

	// Rescanning A must not claim B's declaration disappeared: a scan is
	// authoritative only over the subtree it walked.
	declarations, err := s.Reindex(ctx, rootA)
	require.NoError(t, err)
	for _, d := range declarations {
		if d.Path == pathB {
			t.Fatalf("scan of %s unregistered another consumer's declaration %s", rootA, pathB)
		}
	}
	require.FileExists(t, pathA)

	// B is still resolvable, which is the property the bug actually destroyed.
	_, err = s.latestProfile(ctx, "consumer-b/blog")
	require.NoError(t, err)
}

func TestReindexWithoutRootRefusesRatherThanUnregisteringEverything(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	root := t.TempDir()
	declaredProfile(t, root, "a.json", "consumer-a/blog")
	_, err := s.Reindex(ctx, root)
	require.NoError(t, err)

	// No root given and no default configured: this must fail loudly instead of
	// scanning nothing and calling every registered declaration missing.
	_, err = s.Reindex(ctx, "")
	require.ErrorIs(t, err, ErrDeclarationRootMissing)
	_, err = s.latestProfile(ctx, "consumer-a/blog")
	require.NoError(t, err, "a refused reindex must leave records intact")

	// A configured default makes the rootless call mean "my own declarations".
	s.SetDeclarationsRoot(root)
	_, err = s.Reindex(ctx, "")
	require.NoError(t, err)
	_, err = s.latestProfile(ctx, "consumer-a/blog")
	require.NoError(t, err)
}

func TestReindexRefusesAnUnresolvableRoot(t *testing.T) {
	s, _ := newTestService(t)
	_, err := s.Reindex(context.Background(), filepath.Join(t.TempDir(), "absent"))
	require.ErrorIs(t, err, ErrDeclarationRootMissing)
}

func TestContextWindowNeverReachesTheCaller(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	_, err := s.CreateStyle(ctx, Style{Key: "cw-voice", Directives: []string{"plain"}})
	require.NoError(t, err)
	_, err = s.CreateProfile(ctx, Profile{Key: "cw", StyleRefs: []string{"cw-voice"}, Sampler: Sampler{Kind: samplerDirect, K: 1}})
	require.NoError(t, err)

	resolved, err := s.ResolveProfile(ctx, "cw")
	require.NoError(t, err)
	// The resolved profile intentionally carries no provider context-window fact.
	resolvedJSON, err := json.Marshal(resolved)
	require.NoError(t, err)
	require.NotContains(t, string(resolvedJSON), "context_window")

	out, err := s.Generate(ctx, GenerateRequest{ProfileKey: "cw", Query: "write something", IncludeCandidates: true})
	require.NoError(t, err)
	provenanceJSON, err := json.Marshal(out.Candidates[0].Provenance)
	require.NoError(t, err)
	require.NotContains(t, string(provenanceJSON), "context_window")

	// A profile whose ceiling would exceed any plausible window must still be
	// accepted while the window is unknown: an undeclared ceiling refuses nothing.
	_, err = s.CreateProfile(ctx, Profile{Key: "cw-huge", StyleRefs: []string{"cw-voice"}, Sampler: Sampler{Kind: samplerDirect, K: 1}, ContextPolicy: ContextPolicy{FullTextTokenBudget: 900000, SummarizeBeyond: 900000}})
	require.NoError(t, err)
}

func TestComparabilityRefusesRoundsGeneratedUnderDifferentConditions(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	_, err := s.CreateStyle(ctx, Style{Key: "cmp-voice", Directives: []string{"plain"}})
	require.NoError(t, err)
	_, err = s.CreateProfile(ctx, Profile{Key: "cmp-direct", StyleRefs: []string{"cmp-voice"}, Sampler: Sampler{Kind: samplerDirect, K: 3}})
	require.NoError(t, err)
	_, err = s.CreateProfile(ctx, Profile{Key: "cmp-vs", StyleRefs: []string{"cmp-voice"}, Sampler: Sampler{Kind: samplerVSStandard, K: 3, Tau: 0.1}})
	require.NoError(t, err)

	directA, err := s.Generate(ctx, GenerateRequest{ProfileKey: "cmp-direct", Query: "q"})
	require.NoError(t, err)
	directB, err := s.Generate(ctx, GenerateRequest{ProfileKey: "cmp-direct", Query: "q"})
	require.NoError(t, err)
	vs, err := s.Generate(ctx, GenerateRequest{ProfileKey: "cmp-vs", Query: "q"})
	require.NoError(t, err)

	// Same conditions compare.
	require.NoError(t, s.CompareRounds(ctx, directA.Round.ID, directB.Round.ID))
	// Strategy is the variable under test, so two sets drawn under otherwise
	// identical conditions must compare — that comparison is the measurement
	// the scenario exists to make.
	require.NoError(t, s.CompareRounds(ctx, directA.Round.ID, vs.Round.ID))

	// The threshold is excluded from the key as a strategy-owned parameter, but
	// it stays on the round: dropping it from comparability must not drop it
	// from the record.
	require.InDelta(t, 0.1, vs.Round.Strategy.Tau, 1e-9)
	require.Equal(t, samplerVSStandard, vs.Round.Strategy.Kind)
}

func TestComparabilityStillRefusesGenuinelyDifferentConditions(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	_, err := s.CreateStyle(ctx, Style{Key: "cond-voice", Directives: []string{"plain"}})
	require.NoError(t, err)
	_, err = s.CreateProfile(ctx, Profile{Key: "cond-k3", StyleRefs: []string{"cond-voice"}, Sampler: Sampler{Kind: samplerVSStandard, K: 3, Tau: 0.1}})
	require.NoError(t, err)
	_, err = s.CreateProfile(ctx, Profile{Key: "cond-k5", StyleRefs: []string{"cond-voice"}, Sampler: Sampler{Kind: samplerVSStandard, K: 5, Tau: 0.1}})
	require.NoError(t, err)

	k3, err := s.Generate(ctx, GenerateRequest{ProfileKey: "cond-k3", Query: "q"})
	require.NoError(t, err)
	k5, err := s.Generate(ctx, GenerateRequest{ProfileKey: "cond-k5", Query: "q"})
	require.NoError(t, err)

	// Candidate count changes the pairwise structure a set-diversity number
	// summarises, so it stays a condition even though strategy no longer is.
	require.Error(t, s.CompareRounds(ctx, k3.Round.ID, k5.Round.ID))
}

// --- long-form composition -------------------------------------------------

func TestSectionsCarryPriorCommittedContext(t *testing.T) {
	s, fake := newTestService(t)
	ctx := context.Background()
	_, err := s.CreateProfile(ctx, Profile{Key: "ctx-form", Sampler: Sampler{Kind: samplerDirect, K: 1}, Budget: Budget{MaxOutputTokens: 128}, Composition: CompositionPolicy{SectionCount: 3}})
	require.NoError(t, err)

	doc, err := s.CreateDocument(ctx, Document{Title: "A continued document", ProfileKey: "ctx-form"}, nil)
	require.NoError(t, err)
	require.Len(t, doc.Sections, 3)

	// The regression: commitSectionCandidate records the committed identifier on
	// its own copy of the section, so the caller's slice used to keep an
	// uncommitted copy. Every later section then skipped every earlier one and
	// was drafted blind, which is what made a document restate itself.
	sectionQueries := sectionPrompts(fake)
	require.Len(t, sectionQueries, 3)
	require.NotContains(t, sectionQueries[0], "Prior section", "the opening section has no prior text to carry")
	require.Contains(t, sectionQueries[1], "Prior section 1", "the second section must see the committed first section")
	require.Contains(t, sectionQueries[2], "Prior section 1")
	require.Contains(t, sectionQueries[2], "Prior section 2")

	stored := map[string]Section{}
	for _, id := range doc.SectionIDs {
		var section Section
		require.NoError(t, s.loadJSON(ctx, "prose_sections", id, &section))
		stored[id] = section
	}
	last := stored[doc.SectionIDs[2]]
	require.Len(t, last.Context.PriorSectionRefs, 2, "the context snapshot must name what it actually carried")
	require.NotZero(t, last.Context.EstimatedTokens, "a snapshot that carried prior text cannot estimate zero tokens")
	require.Len(t, last.Context.FullTextSectionRefs, 2, "a short document must carry prior sections whole rather than summarising them")
	require.Empty(t, last.Context.SummarizedSectionRefs)

	var first Candidate
	require.NoError(t, s.loadJSON(ctx, "prose_candidates", stored[doc.SectionIDs[0]].CommittedCandidateID, &first))
	require.Contains(t, sectionQueries[2], first.Text, "the committed prose itself must reach the prompt, not only a reference to it")
}

func TestRepairRewritesTheMostRedundantSectionWhenCoherenceFails(t *testing.T) {
	s, fake := newTestService(t)
	ctx := context.Background()
	_, err := s.CreateProfile(ctx, Profile{
		Key:         "repair-form",
		Sampler:     Sampler{Kind: samplerDirect, K: 1},
		Budget:      Budget{MaxOutputTokens: 128},
		Composition: CompositionPolicy{SectionCount: 3, MaxRepairRounds: 2},
		// Deliberately unreachable for this fixture, whose sections share most of
		// their words. The point under test is that a failed verdict causes work
		// rather than only being written down.
		Coherence: CoherenceThresholds{MaxCrossSectionRepetition: 0.01},
	})
	require.NoError(t, err)

	doc, err := s.CreateDocument(ctx, Document{Title: "A repeating document", ProfileKey: "repair-form"}, nil)
	require.NoError(t, err)
	require.Len(t, sectionPrompts(fake), 5, "three sections plus two bounded repair attempts")

	coherence, ok := doc.Coherence.(map[string]any)
	require.True(t, ok)
	verdict := coherence["verdict"].(map[string]any)
	require.Equal(t, false, verdict["coherent"], "an exhausted repair budget must report the honest failed verdict, not a pass")
	require.Len(t, doc.SectionIDs, 3, "repair rewrites a section in place rather than appending one")
}

func TestRepairIsSkippedWhenTheVerdictPasses(t *testing.T) {
	s, fake := newTestService(t)
	ctx := context.Background()
	_, err := s.CreateProfile(ctx, Profile{Key: "quiet-form", Sampler: Sampler{Kind: samplerDirect, K: 1}, Budget: Budget{MaxOutputTokens: 128}, Composition: CompositionPolicy{SectionCount: 3, MaxRepairRounds: 2}})
	require.NoError(t, err)
	doc, err := s.CreateDocument(ctx, Document{Title: "A quiet document", ProfileKey: "quiet-form"}, nil)
	require.NoError(t, err)
	require.Len(t, sectionPrompts(fake), 3, "no declared threshold means nothing to repair")
	require.True(t, coherenceVerdictPassed(doc.Coherence))
}

func TestSectionPromptStatesContinuityRequirement(t *testing.T) {
	s, fake := newTestService(t)
	ctx := context.Background()
	_, err := s.CreateProfile(ctx, Profile{Key: "cont-form", Sampler: Sampler{Kind: samplerDirect, K: 1}, Budget: Budget{MaxOutputTokens: 128}, Composition: CompositionPolicy{SectionCount: 3}})
	require.NoError(t, err)
	_, err = s.CreateDocument(ctx, Document{Title: "A progressing document", ProfileKey: "cont-form"}, nil)
	require.NoError(t, err)

	sectionQueries := sectionPrompts(fake)
	require.Len(t, sectionQueries, 3)
	require.Contains(t, sectionQueries[0], "this is the opening section")
	require.Contains(t, sectionQueries[1], "Do not restate", "prior text without an instruction produces paraphrase, not progression")
}

func TestContinuationPolicySelectsLeastRedundantCandidate(t *testing.T) {
	prior := "Server owned runs survive the caller because test genie takes ownership when the command returns."
	candidates := []Candidate{
		{ID: "echo", Text: prior, Eligibility: Eligibility{Eligible: true}},
		{ID: "advance", Text: "Operators reconnect later and inspect the recorded identifier without repeating hours of work.", Eligibility: Eligibility{Eligible: true}},
	}
	selected := choose(candidates, policyContinuation, nil, 1, &SelectionContext{PriorText: []string{prior}})
	require.NotNil(t, selected)
	require.Equal(t, "advance", selected.ID, "continuation must not select the candidate that repeats committed text")
}

func TestContinuationPolicyPrefersTheTargetLength(t *testing.T) {
	short := Candidate{ID: "short", Text: "Too brief.", Measurements: textmetrics.Metrics{WordCount: 10}, Eligibility: Eligibility{Eligible: true}}
	right := Candidate{ID: "right", Text: "About the right size for the cell.", Measurements: textmetrics.Metrics{WordCount: 100}, Eligibility: Eligibility{Eligible: true}}
	selected := choose([]Candidate{short, right}, policyContinuation, nil, 1, &SelectionContext{TargetWords: 100})
	require.NotNil(t, selected)
	require.Equal(t, "right", selected.ID)
}

func TestContinuationPolicyIgnoresVerbalizedHint(t *testing.T) {
	// The hint is uncalibrated, so it must not become a selection signal by the
	// back door when a new policy is added.
	first := Candidate{ID: "first", Text: "One distinct passage.", VerbalizedHint: &VerbalizedHint{Ordinal: 1}, Eligibility: Eligibility{Eligible: true}}
	second := Candidate{ID: "second", Text: "One distinct passage.", VerbalizedHint: &VerbalizedHint{Ordinal: 9}, Eligibility: Eligibility{Eligible: true}}
	selected := choose([]Candidate{first, second}, policyContinuation, nil, 1, &SelectionContext{})
	require.NotNil(t, selected)
	require.Equal(t, "first", selected.ID, "identical candidates must resolve by order, never by hint rank")
}

func TestResolveSectionCountDerivesFromWordBudget(t *testing.T) {
	profile := Profile{Composition: CompositionPolicy{TargetSectionWords: 350}}
	require.Equal(t, 3, resolveSectionCount(Document{}, profile, 900), "a short article floors at the minimum section count")
	require.Equal(t, 6, resolveSectionCount(Document{}, profile, 2000), "a longer article earns more sections rather than longer ones")
	require.Equal(t, 4, resolveSectionCount(Document{SectionCount: 4}, profile, 2000), "an explicit document override wins")
	require.Equal(t, 5, resolveSectionCount(Document{}, Profile{Composition: CompositionPolicy{SectionCount: 5}}, 900), "an explicit profile count wins over derivation")
	require.Equal(t, defaultMaxSections, resolveSectionCount(Document{}, profile, 100000), "derivation stays bounded")
}

func TestSectionWordBandGivesAFloorAndACeiling(t *testing.T) {
	minWords, maxWords := sectionWordBand(Profile{}, 400)
	require.Equal(t, 300, minWords)
	require.Equal(t, 500, maxWords)
	require.Greater(t, minWords, 0, "a zero floor is what allowed a 1100-word outline to be satisfied by 490 words")

	tight, ceiling := sectionWordBand(Profile{Composition: CompositionPolicy{SectionWordTolerance: 0.1}}, 400)
	require.Equal(t, 360, tight)
	require.Equal(t, 440, ceiling)
}

func TestSectionUndershootIsIneligible(t *testing.T) {
	metrics := textmetrics.Metrics{WordCount: 120, SentenceCount: 6}
	eligibility := gate(metrics, Constraints{MinWords: 300, MaxWords: 500}, "some prose", gateInputs{})
	require.False(t, eligibility.Eligible)
	require.Equal(t, "min_words:300", eligibility.Reason)
}

func TestDecodeOutlineHonoursExpectedCount(t *testing.T) {
	three := `[{"intent":"a","summary":"a","target_words":10},{"intent":"b","summary":"b","target_words":10},{"intent":"c","summary":"c","target_words":10}]`
	_, err := decodeOutline(three, 4, 6)
	require.ErrorIs(t, err, ErrMalformedOutline, "an outline below the declared floor is malformed")
	_, err = decodeOutline(three, 1, 2)
	require.ErrorIs(t, err, ErrMalformedOutline, "an outline above the declared ceiling is malformed")

	// Inside the band the exact count is the model's to choose: section count is
	// a policy range, and demanding one number turned an ordinary outline into a
	// hard failure that no retry could recover.
	outline, err := decodeOutline(three, 2, 5)
	require.NoError(t, err)
	require.Len(t, outline, 3)
}

func TestOutlineSchemaDoesNotPinASectionCount(t *testing.T) {
	require.Contains(t, outlineSchema, `"type":"array"`, "an object schema with section_1..section_3 as required properties pinned every document to three sections")
	require.NotContains(t, outlineSchema, "section_1")
	// The gateway refuses minItems/maxItems as outside its enforceable subset
	// rather than ignoring them, so a schema carrying them fails the request
	// outright. The band belongs in the instruction and the decoder.
	require.NotContains(t, outlineSchema, "minItems")
	require.NotContains(t, outlineSchema, "maxItems")
}

func TestResolveSectionPlanIsExactOnlyWhenDeclared(t *testing.T) {
	want, low, high := resolveSectionPlan(Document{}, Profile{Composition: CompositionPolicy{TargetSectionWords: 350, MinSections: 4, MaxSections: 7}}, 1400)
	require.Equal(t, 4, want)
	require.Equal(t, 4, low)
	require.Equal(t, 7, high)

	want, low, high = resolveSectionPlan(Document{SectionCount: 5}, Profile{}, 1400)
	require.Equal(t, 5, want)
	require.Equal(t, 5, low, "an explicitly declared count stays exact")
	require.Equal(t, 5, high)
}

func TestDecodeOutlineRecoversKeyedObjectInOrder(t *testing.T) {
	keyed := `{"section_2":{"intent":"b","summary":"b","target_words":10},"section_1":{"intent":"a","summary":"a","target_words":10}}`
	outline, err := decodeOutline(keyed, 2, 2)
	require.NoError(t, err)
	require.Equal(t, "a", outline[0].Intent, "keyed recovery must order by key, not by JSON member order")
	require.Equal(t, "b", outline[1].Intent)
}

func TestRichProseAdmitsStructureThatParagraphBans(t *testing.T) {
	withList := "Run the suite and wait once.\n- start the run\n- record the identifier"
	metrics := textmetrics.Analyze(withList, nil)
	require.False(t, matchesRequiredFormat("paragraph", metrics, withList), "paragraph bans list markers, which is why forcing it banned commands")
	require.True(t, matchesRequiredFormat("rich_prose", metrics, withList))
}

func TestSectionRequiredFormatHonoursTheProfile(t *testing.T) {
	require.Equal(t, "paragraph", sectionRequiredFormat(Profile{}))
	require.Equal(t, "rich_prose", sectionRequiredFormat(Profile{Composition: CompositionPolicy{SectionFormat: "rich_prose"}}))
}

func TestSectionSelectionPolicyDefaultsToContinuation(t *testing.T) {
	require.Equal(t, policyContinuation, sectionSelectionPolicy(Profile{SelectionPolicy: "threshold_then_rarest"}), "sections must not inherit the ideation policy")
	require.Equal(t, "take_first", sectionSelectionPolicy(Profile{Composition: CompositionPolicy{SectionSelectionPolicy: "take_first"}}))
}

func TestMostRedundantSectionNamesTheRepeater(t *testing.T) {
	texts := []string{
		"The runner takes ownership of the suite the moment the command returns.",
		"Operators reconnect later and read the recorded evidence at their leisure.",
		"The runner takes ownership of the suite the moment the command returns.",
	}
	worst := mostRedundantSection(texts)
	require.Contains(t, []int{0, 2}, worst, "the duplicated pair must be named, not the distinct section")
	require.Equal(t, -1, mostRedundantSection([]string{"only one"}))
}

func TestCoherenceVerdictPassedRefusesAnAbsentVerdict(t *testing.T) {
	require.False(t, coherenceVerdictPassed(nil), "an unassembled document must not be treated as coherent")
	require.False(t, coherenceVerdictPassed(map[string]any{"verdict": map[string]any{"coherent": false}}))
	require.True(t, coherenceVerdictPassed(map[string]any{"verdict": map[string]any{"coherent": true}}))
}

func TestAssemblyReportsSemanticMeasurementAsUnavailableRatherThanPassing(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	_, err := s.CreateProfile(ctx, Profile{Key: "sem-form", Sampler: Sampler{Kind: samplerDirect, K: 1}, Budget: Budget{MaxOutputTokens: 128}, Composition: CompositionPolicy{SectionCount: 3}, Coherence: CoherenceThresholds{MaxSemanticSectionRepetition: 0.5}})
	require.NoError(t, err)
	doc, err := s.CreateDocument(ctx, Document{Title: "A document without embeddings", ProfileKey: "sem-form"}, nil)
	require.NoError(t, err)

	coherence, ok := doc.Coherence.(map[string]any)
	require.True(t, ok)
	require.Equal(t, false, coherence["semantic_measured"], "a gateway with no embedding surface must say so")
	require.Contains(t, coherence["semantic_unavailable"], "embedding_unavailable")
	require.NotContains(t, coherence, "semantic_section_repetition", "an unmeasured value must not be published as a number")
}

// sectionPrompts isolates the section-generation calls. The summarize path also
// crosses the gateway, so "not the outline call" is not the same thing as "a
// section call".
func sectionPrompts(fake *fakeGateway) []string {
	var out []string
	for _, call := range fake.calls {
		if strings.Contains(call.Query, "HARD SECTION LENGTH") {
			out = append(out, call.Query)
		}
	}
	return out
}

func TestAssembledDocumentReportsItsOwnProvenance(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	_, err := s.CreateProfile(ctx, Profile{Key: "prov-form", Sampler: Sampler{Kind: samplerDirect, K: 1}, Budget: Budget{MaxOutputTokens: 128}, Composition: CompositionPolicy{SectionCount: 3}})
	require.NoError(t, err)
	doc, err := s.CreateDocument(ctx, Document{Title: "A costed document", ProfileKey: "prov-form"}, nil)
	require.NoError(t, err)

	// A long-form run costs an outline call plus one call per section. Reading a
	// standalone generate call's accounting as the document's reported a
	// different request entirely.
	require.Equal(t, 3, doc.Provenance.SectionCount)
	require.Greater(t, doc.Provenance.WordCount, 100)
	require.Greater(t, doc.Provenance.TotalCostMicros, int64(0))
	require.Equal(t, []string{"fake"}, doc.Provenance.Providers)
	require.Equal(t, []string{"fake-model"}, doc.Provenance.Models)
}

func TestSectionSamplerDefaultsToDirectDraws(t *testing.T) {
	kind, count := sectionSampler(Profile{Sampler: Sampler{Kind: samplerVSStandard, K: 5}})
	require.Equal(t, samplerDirect, kind, "sections must not inherit the outline's verbalized sampler")
	require.Equal(t, defaultSectionCandidates, count, "continuation still needs a set to choose from")

	kind, count = sectionSampler(Profile{Composition: CompositionPolicy{SectionSamplerKind: samplerVSStandard, SectionCandidates: 4}})
	require.Equal(t, samplerVSStandard, kind)
	require.Equal(t, 4, count)
}

func TestEligibleCandidateNeverReturnsARejectedCandidate(t *testing.T) {
	// The previous fallback took candidates[0] whenever the policy declined to
	// choose, which committed text the constraint gate had already rejected.
	rejected := Candidate{ID: "short", Eligibility: Eligibility{Eligible: false, Reason: "min_words:300"}}
	ok := Candidate{ID: "sound", Eligibility: Eligibility{Eligible: true}}
	require.Nil(t, eligibleCandidate(GenerateResponse{Candidates: []Candidate{rejected}}))
	require.Equal(t, "sound", eligibleCandidate(GenerateResponse{Candidates: []Candidate{rejected, ok}}).ID)
	require.Equal(t, "sound", eligibleCandidate(GenerateResponse{Selected: &ok, Candidates: []Candidate{rejected, ok}}).ID)
	require.Equal(t, "sound", eligibleCandidate(GenerateResponse{Selected: &rejected, Candidates: []Candidate{rejected, ok}}).ID, "an ineligible policy choice must not be committed")
}

func TestClosestMissReportsTheLongestDraw(t *testing.T) {
	candidates := []Candidate{
		{Measurements: textmetrics.Metrics{WordCount: 120}},
		{Measurements: textmetrics.Metrics{WordCount: 181}},
		{Measurements: textmetrics.Metrics{WordCount: 96}},
	}
	require.Equal(t, 181, closestMiss(candidates))
	require.Equal(t, 0, closestMiss(nil))
}

func TestSectionOutputCapLeavesRoomForInvisibleTokens(t *testing.T) {
	// Twice the word ceiling was wrong by roughly an order of magnitude against
	// a reasoning model, whose reasoning tokens are charged to the same budget:
	// a 312-word section drew a 624-token cap and came back as a 12-word
	// truncated fragment.
	require.Equal(t, defaultSectionOutputFloor, sectionOutputCap(Profile{}, 312))
	require.Equal(t, 6000, sectionOutputCap(Profile{}, 1000))
	require.Equal(t, 900, sectionOutputCap(Profile{Composition: CompositionPolicy{SectionMaxOutputTokens: 900}}, 312), "an explicit declaration wins")
	require.GreaterOrEqual(t, sectionOutputCap(Profile{}, 312), 312*2, "the cap must never fall below the prose it is meant to hold")
}

func TestGenerateWithProfileCarriesTheSelectionPolicy(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	_, err := s.CreateProfile(ctx, Profile{Key: "carry", Sampler: Sampler{Kind: samplerDirect, K: 2}, SelectionPolicy: "coverage", Budget: Budget{MaxOutputTokens: 256}})
	require.NoError(t, err)
	stored, err := s.latestProfile(ctx, "carry")
	require.NoError(t, err)

	// The section path computes a policy and passes it in. Dropping it here
	// meant the stored ideation policy was used on every section instead.
	stored.SelectionPolicy = policyContinuation
	out, err := s.generateWithProfile(ctx, GenerateRequest{ProfileKey: "carry", Query: "a query", IncludeCandidates: true}, stored)
	require.NoError(t, err)
	require.NotNil(t, out.Selected, "coverage returns no single selection; continuation must actually have been applied")
}

func TestDegenerateEmbeddingIsNotAMeasurement(t *testing.T) {
	svc := &Service{gateway: stubEmbedder{dimension: 3}}
	_, _, err := svc.semanticSectionRepetition(context.Background(), []string{"one", "two"})
	require.ErrorContains(t, err, "embedding_degenerate", "a three-component embedding cannot support a similarity threshold")

	svc = &Service{gateway: stubEmbedder{dimension: minimumEmbeddingDimension}}
	_, basis, err := svc.semanticSectionRepetition(context.Background(), []string{"one", "two"})
	require.NoError(t, err)
	require.Contains(t, basis, "semantic cosine similarity")
}

type stubEmbedder struct{ dimension int }

func (stubEmbedder) Generate(context.Context, GatewayRequest) ([]GatewayCandidate, error) {
	return nil, errors.New("not used")
}

func (s stubEmbedder) Embed(_ context.Context, req EmbeddingRequest) (EmbeddingResponse, error) {
	out := EmbeddingResponse{Dimension: s.dimension}
	for i := range req.Texts {
		vector := make([]float64, s.dimension)
		for j := range vector {
			vector[j] = float64((i + j) % 7)
		}
		out.Vectors = append(out.Vectors, vector)
	}
	return out, nil
}

// --- provider robustness ---------------------------------------------------

// flakyGateway wraps the fixture and injects provider failures on demand, so a
// long-form composition can be driven through the failure modes a real provider
// produces: a malformed candidate set, an unreachable resource, a caller-side
// defect that no retry can fix.
type flakyGateway struct {
	inner *fakeGateway
	fail  func(call int, req GatewayRequest) error
	calls int
	seen  []GatewayRequest
}

func (f *flakyGateway) Generate(ctx context.Context, req GatewayRequest) ([]GatewayCandidate, error) {
	f.calls++
	f.seen = append(f.seen, req)
	if f.fail != nil {
		if err := f.fail(f.calls, req); err != nil {
			return nil, err
		}
	}
	return f.inner.Generate(ctx, req)
}

func newFlakyService(t *testing.T, fail func(call int, req GatewayRequest) error) (*Service, *flakyGateway) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:prose-test?mode=memory&cache=shared")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, EnsureSchema(db))
	gw := &flakyGateway{inner: &fakeGateway{}, fail: fail}
	return NewWithGateway(db, gw), gw
}

func isSectionCall(req GatewayRequest) bool {
	return strings.Contains(req.Query, "HARD SECTION LENGTH")
}

func TestTransientProviderFailureDoesNotDiscardTheDocument(t *testing.T) {
	failed := false
	s, gw := newFlakyService(t, func(_ int, req GatewayRequest) error {
		if isSectionCall(req) && !failed {
			failed = true
			return fmt.Errorf("%w: candidate 4 carries no text", ErrMalformedCandidateSet)
		}
		return nil
	})
	ctx := context.Background()
	_, err := s.CreateProfile(ctx, Profile{Key: "flaky", Sampler: Sampler{Kind: samplerDirect, K: 1}, Budget: Budget{MaxOutputTokens: 128}, Composition: CompositionPolicy{SectionCount: 3}})
	require.NoError(t, err)

	doc, err := s.CreateDocument(ctx, Document{Title: "A document that survived a bad response", ProfileKey: "flaky"}, nil)
	require.NoError(t, err, "one malformed candidate set must not discard an otherwise complete article")
	require.Equal(t, "assembled", doc.Status)
	require.Len(t, doc.SectionIDs, 3)
	require.True(t, failed, "the failure must actually have been injected")
	require.Greater(t, gw.calls, 3)
}

func TestDeterministicRequestErrorIsNotRetried(t *testing.T) {
	s, gw := newFlakyService(t, func(_ int, req GatewayRequest) error {
		if isSectionCall(req) {
			return fmt.Errorf("gateway refused the assembled context: %w", ErrContextInfeasible)
		}
		return nil
	})
	ctx := context.Background()
	_, err := s.CreateProfile(ctx, Profile{Key: "infeasible", Sampler: Sampler{Kind: samplerDirect, K: 1}, Budget: Budget{MaxOutputTokens: 128}, Composition: CompositionPolicy{SectionCount: 3}})
	require.NoError(t, err)

	_, err = s.CreateDocument(ctx, Document{Title: "A document that cannot fit", ProfileKey: "infeasible"}, nil)
	require.ErrorIs(t, err, ErrContextInfeasible)

	sectionCalls := 0
	for _, req := range gw.seen {
		if isSectionCall(req) {
			sectionCalls++
		}
	}
	require.Equal(t, 1, sectionCalls, "a caller-side defect yields the same answer every time; retrying only spends money")
}

func TestExhaustedAttemptsNameTheStepAndTheCause(t *testing.T) {
	s, _ := newFlakyService(t, func(_ int, req GatewayRequest) error {
		if isSectionCall(req) {
			return errors.New("ai-gateway inference: provider value failed local validation")
		}
		return nil
	})
	ctx := context.Background()
	_, err := s.CreateProfile(ctx, Profile{Key: "doomed", Sampler: Sampler{Kind: samplerDirect, K: 1}, Budget: Budget{MaxOutputTokens: 128}, Composition: CompositionPolicy{SectionCount: 3, GatewayAttempts: 2}})
	require.NoError(t, err)

	_, err = s.CreateDocument(ctx, Document{Title: "A document that never lands", ProfileKey: "doomed"}, nil)
	require.ErrorContains(t, err, "section 0 generation failed after 2 attempts")
	require.ErrorContains(t, err, "failed local validation", "the provider's own reason must survive the wrapper")
}

func TestVerbalizedSetFallsBackToADirectDraw(t *testing.T) {
	// A set that will not decode is a failure of the envelope, not of the
	// content. Re-asking the same sampler for the same shape tends to fail the
	// same way, so the last attempt drops the envelope.
	s, gw := newFlakyService(t, func(_ int, req GatewayRequest) error {
		if isSectionCall(req) && req.Strategy == samplerVSStandard {
			return fmt.Errorf("%w: candidate 4 carries no text", ErrMalformedCandidateSet)
		}
		return nil
	})
	ctx := context.Background()
	_, err := s.CreateProfile(ctx, Profile{Key: "envelope", Sampler: Sampler{Kind: samplerDirect, K: 1}, Budget: Budget{MaxOutputTokens: 128}, Composition: CompositionPolicy{SectionCount: 3, SectionSamplerKind: samplerVSStandard, SectionCandidates: 5}})
	require.NoError(t, err)

	doc, err := s.CreateDocument(ctx, Document{Title: "A document whose envelope kept failing", ProfileKey: "envelope"}, nil)
	require.NoError(t, err)
	require.Len(t, doc.SectionIDs, 3)

	var sawVerbalized, sawDirect bool
	for _, req := range gw.seen {
		if !isSectionCall(req) {
			continue
		}
		sawVerbalized = sawVerbalized || req.Strategy == samplerVSStandard
		sawDirect = sawDirect || req.Strategy == samplerDirect
	}
	require.True(t, sawVerbalized, "the declared sampler must be tried first")
	require.True(t, sawDirect, "the last attempt must drop to a draw with no envelope to get wrong")
}

func TestFailedSummarisationDegradesRatherThanDiscardingTheDocument(t *testing.T) {
	s, _ := newFlakyService(t, func(_ int, req GatewayRequest) error {
		if req.Role == "extract.structured" {
			return errors.New("ai-gateway embedding: typed inference is unavailable")
		}
		return nil
	})
	ctx := context.Background()
	_, err := s.CreateProfile(ctx, Profile{
		Key:           "degraded",
		Sampler:       Sampler{Kind: samplerDirect, K: 1},
		Budget:        Budget{MaxOutputTokens: 128},
		ContextPolicy: ContextPolicy{FullTextTokenBudget: 1, SummarizeBeyond: 1},
		Composition:   CompositionPolicy{SectionCount: 3},
	})
	require.NoError(t, err)

	doc, err := s.CreateDocument(ctx, Document{Title: "A document whose summariser was down", ProfileKey: "degraded"}, nil)
	require.NoError(t, err, "a support call that cannot complete must not cost the whole article")
	require.Len(t, doc.SectionIDs, 3)

	var last Section
	require.NoError(t, s.loadJSON(ctx, "prose_sections", doc.SectionIDs[2], &last))
	require.NotEmpty(t, last.Context.DegradedSummaryRefs, "a locally derived summary must be distinguishable from a written one")
	require.Len(t, last.Context.DegradedSummaryRefs, len(last.Context.SummarizedSectionRefs))
	for _, id := range last.Context.DegradedSummaryRefs {
		require.NotEmpty(t, last.Context.SectionSummaries[id], "a degraded summary still has to carry text")
	}
}

func TestDeterministicExcerptIsBoundedAndNonEmpty(t *testing.T) {
	long := strings.TrimSpace(strings.Repeat("The runner keeps the evidence after the caller disconnects. ", 30))
	excerpt := deterministicExcerpt(long)
	require.NotEmpty(t, excerpt)
	require.LessOrEqual(t, len(strings.Fields(excerpt)), excerptWordBudget+12)
	require.Less(t, len(excerpt), len(long))
	require.Equal(t, "", deterministicExcerpt("   "))
	require.Equal(t, "no terminator here.", deterministicExcerpt("no terminator here"), "a passage with no sentence break still has to summarise to something, and it is terminated")
}

func TestDeterministicRequestErrorClassification(t *testing.T) {
	for _, err := range []error{ErrBudgetExceeded, ErrContextInfeasible, ErrEmptyQuery, ErrPromptContainsIdentifier, ErrUnknownLocality, ErrProfileUnregistered, context.Canceled} {
		require.True(t, deterministicRequestError(fmt.Errorf("wrapped: %w", err)), "%v must not be retried", err)
	}
	for _, err := range []error{ErrMalformedCandidateSet, ErrMalformedOutline, errors.New("provider value failed local validation")} {
		require.False(t, deterministicRequestError(fmt.Errorf("wrapped: %w", err)), "%v is a property of one response and is worth re-asking", err)
	}
	require.False(t, deterministicRequestError(nil))
}

// --- document discovery ----------------------------------------------------

func TestListDocumentsAnswersWhatWasWritten(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	_, err := s.CreateProfile(ctx, Profile{Key: "listing", Sampler: Sampler{Kind: samplerDirect, K: 1}, Budget: Budget{MaxOutputTokens: 128}, Composition: CompositionPolicy{SectionCount: 3}})
	require.NoError(t, err)

	first, err := s.CreateDocument(ctx, Document{Title: "The first article", ProfileKey: "listing"}, nil)
	require.NoError(t, err)
	second, err := s.CreateDocument(ctx, Document{Title: "The second article", ProfileKey: "listing"}, nil)
	require.NoError(t, err)

	// Nothing previously enumerated documents: a caller could create one,
	// assemble one by identifier, or resume one by identifier, but had no way to
	// discover an identifier it was never told.
	listed, err := s.ListDocuments(ctx, 0, "")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(listed), 2)
	require.Equal(t, second.ID, listed[0].ID, "newest first")
	require.Equal(t, first.ID, listed[1].ID)
	require.Equal(t, "The second article", listed[0].Title)
	require.Equal(t, 3, listed[0].SectionCount)
	require.Greater(t, listed[0].WordCount, 100)
	require.Greater(t, listed[0].TotalCostMicros, int64(0))
	require.False(t, listed[0].CreatedAt.IsZero(), "a listing ordered by recency has to carry a creation time")
}

func TestListDocumentsHonoursLimitAndStatus(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	_, err := s.CreateProfile(ctx, Profile{Key: "limits", Sampler: Sampler{Kind: samplerDirect, K: 1}, Budget: Budget{MaxOutputTokens: 128}, Composition: CompositionPolicy{SectionCount: 3}})
	require.NoError(t, err)
	for i := 0; i < 3; i++ {
		_, err := s.CreateDocument(ctx, Document{Title: fmt.Sprintf("Article %d", i), ProfileKey: "limits"}, nil)
		require.NoError(t, err)
	}

	limited, err := s.ListDocuments(ctx, 2, "")
	require.NoError(t, err)
	require.Len(t, limited, 2)

	assembled, err := s.ListDocuments(ctx, 0, "assembled")
	require.NoError(t, err)
	require.NotEmpty(t, assembled)
	for _, doc := range assembled {
		require.Equal(t, "assembled", doc.Status)
	}

	none, err := s.ListDocuments(ctx, 0, "no-such-status")
	require.NoError(t, err)
	require.Empty(t, none)
}

func TestGetDocumentReturnsTheProseAndItsMeasurements(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	_, err := s.CreateProfile(ctx, Profile{Key: "reading", Sampler: Sampler{Kind: samplerDirect, K: 1}, Budget: Budget{MaxOutputTokens: 128}, Composition: CompositionPolicy{SectionCount: 3}})
	require.NoError(t, err)
	created, err := s.CreateDocument(ctx, Document{Title: "A readable article", ProfileKey: "reading"}, nil)
	require.NoError(t, err)

	fetched, err := s.GetDocument(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, created.ID, fetched.ID)
	require.NotEmpty(t, fetched.AssembledText, "reading a document must return its prose")
	require.Len(t, fetched.Outline, 3, "and the plan it was written against")
	require.NotNil(t, fetched.Coherence)
	require.Equal(t, 3, fetched.Provenance.SectionCount)

	_, err = s.GetDocument(ctx, "")
	require.ErrorIs(t, err, ErrDocumentIDRequired)
	_, err = s.GetDocument(ctx, "not-a-document")
	require.ErrorIs(t, err, ErrDocumentNotFound, "a missing document is not found, not an invalid argument")
}

func TestGateRejectsSectionThatRestatesItsPredecessor(t *testing.T) {
	previous := "The runner owns the execution after the command returns. Logs stream to durable storage and the run identifier stays addressable across machines."
	restated := "Execution is owned by the runner once the command has returned. Durable storage receives streamed logs, and the run identifier remains addressable across machines."
	advanced := "A colleague inherits the release decision hours later. They reconnect from another workstation, read the persisted evidence, and ship without reconstructing telemetry."

	constraints := Constraints{MinSectionNovelty: 0.6}
	selection := &SelectionContext{PriorText: []string{previous}}

	rejected := gate(textmetrics.Analyze(restated, nil), constraints, restated, gateInputs{Selection: selection})
	require.False(t, rejected.Eligible)
	require.Contains(t, rejected.Reason, "min_section_novelty")

	accepted := gate(textmetrics.Analyze(advanced, nil), constraints, advanced, gateInputs{Selection: selection})
	require.True(t, accepted.Eligible, "a section on new material must stay eligible: %s", accepted.Reason)
}

func TestGateSkipsNoveltyWhenThereIsNoPredecessor(t *testing.T) {
	// An opening section has nothing to repeat. The constraint must not apply
	// rather than pass, so a document's first section is never gated on a
	// comparison that does not exist.
	text := "Durable validation begins where the caller ends."
	constraints := Constraints{MinSectionNovelty: 0.9}
	require.True(t, gate(textmetrics.Analyze(text, nil), constraints, text, gateInputs{}).Eligible)
	require.True(t, gate(textmetrics.Analyze(text, nil), constraints, text, gateInputs{Selection: &SelectionContext{}}).Eligible)
}

func TestSectionRetryDirectiveNamesTheConstraintThatFailed(t *testing.T) {
	novelty := []Candidate{{Eligibility: Eligibility{Eligible: false, Reason: "min_section_novelty:0.60:measured:0.31"}}}
	require.Contains(t, sectionRetryDirective(novelty, 300), "restated the preceding section")
	require.NotContains(t, sectionRetryDirective(novelty, 300), "Write a longer section")

	short := []Candidate{{Eligibility: Eligibility{Eligible: false, Reason: "min_words:300"}, Measurements: textmetrics.Metrics{WordCount: 120}}}
	require.Contains(t, sectionRetryDirective(short, 300), "Write a longer section")
}

func TestSectionProfileCarriesDeclaredNoveltyFloorIntoItsConstraints(t *testing.T) {
	// The gate itself is covered above. What this covers is the wiring between
	// the declared coherence threshold and the constraint the gate reads: the
	// floor is declared on the profile's coherence block, and sections are
	// gated through Constraints, so a missing assignment between the two
	// disables the gate everywhere while every other test still passes.
	profile := Profile{Coherence: CoherenceThresholds{MinSectionNovelty: 0.65}}
	sectionProfile := profile
	sectionProfile.Constraints.MinSectionNovelty = sectionProfile.Coherence.MinSectionNovelty
	require.Equal(t, 0.65, sectionProfile.Constraints.MinSectionNovelty)

	previous := "The runner owns the execution after the command returns. Logs stream to durable storage and the identifier stays addressable."
	restated := "Execution is owned by the runner once the command returns. Durable storage receives the logs and the identifier remains addressable."
	result := gate(textmetrics.Analyze(restated, nil), sectionProfile.Constraints, restated, gateInputs{Selection: &SelectionContext{PriorText: []string{previous}}})
	require.False(t, result.Eligible, "a declared floor must reach the gate that enforces it")
	require.Contains(t, result.Reason, "min_section_novelty")
}

func TestGateRejectsProseThatShowsNoDeclaredArtifact(t *testing.T) {
	artifacts := []string{"test-genie runs wait --json", "vrooli scenario test", "docs/TESTING.md"}
	constraints := Constraints{MinArtifacts: 2}

	vague := "The operator starts the suite, records the run identifier, and waits once on the server-owned run without holding a terminal open."
	concrete := "Start the suite with vrooli scenario test, then block exactly once on test-genie runs wait --json rather than polling."

	rejected := gate(textmetrics.Analyze(vague, nil), constraints, vague, gateInputs{Artifacts: artifacts})
	require.False(t, rejected.Eligible, "prose that only describes the mechanism must not pass a concreteness floor")
	require.Contains(t, rejected.Reason, "min_artifacts")

	accepted := gate(textmetrics.Analyze(concrete, nil), constraints, concrete, gateInputs{Artifacts: artifacts})
	require.True(t, accepted.Eligible, "prose quoting two artifacts must pass: %s", accepted.Reason)
}

func TestGateSkipsArtifactFloorWhenTheBriefDeclaredNone(t *testing.T) {
	// Failing a passage for quoting nothing when nothing was offered blames the
	// writer for the brief. The constraint does not apply rather than passing.
	text := "The operator waits once on the server-owned run."
	constraints := Constraints{MinArtifacts: 3}
	require.True(t, gate(textmetrics.Analyze(text, nil), constraints, text, gateInputs{}).Eligible)
}

func TestSectionRetryDirectiveDistinguishesVaguenessFromRestatement(t *testing.T) {
	artifactMiss := []Candidate{{Eligibility: Eligibility{Eligible: false, Reason: "min_artifacts:2:measured:0"}}}
	directive := sectionRetryDirective(artifactMiss, 300)
	require.Contains(t, directive, "without ever showing it")
	require.NotContains(t, directive, "restated the preceding section")
	require.NotContains(t, directive, "Write a longer section")
}
