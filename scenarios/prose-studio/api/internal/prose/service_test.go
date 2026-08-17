package prose

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
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
	out := make([]GatewayCandidate, req.K)
	for i := range out {
		out[i] = GatewayCandidate{Text: []string{"A crisp opening about trust.", "A vivid opening about craft.", "A practical opening about proof.", "A reflective opening about change.", "A measured opening about care."}[(callIndex+i)%5], Provider: "fake", Model: "fake-model", TemperatureSupport: "supported", CostMicros: 100, HintOrdinal: i + 1}
	}
	return out, nil
}

func TestRarestPolicySelectsNonFirstCandidate(t *testing.T) {
	set := textmetrics.SetMetrics{PairwiseSimilarity: [][]float64{{0, .9, .2}, {.9, 0, .1}, {.2, .1, 0}}}
	candidates := []Candidate{{ID: "first", SetIndex: 0, SetMeasurements: set, Eligibility: Eligibility{Eligible: true}}, {ID: "middle", SetIndex: 1, SetMeasurements: set, Eligibility: Eligibility{Eligible: true}}, {ID: "rare", SetIndex: 2, SetMeasurements: set, Eligibility: Eligibility{Eligible: true}}}
	selected := choose(candidates, "threshold_then_rarest", map[string]float64{"threshold": .5}, 1)
	require.NotNil(t, selected)
	require.Equal(t, "rare", selected.ID, "rarest policy must read the candidate's own similarity row")
}

func TestRarestPolicyHonoursThreshold(t *testing.T) {
	set := textmetrics.SetMetrics{PairwiseSimilarity: [][]float64{{0, .9, .2}, {.9, 0, .1}, {.2, .1, 0}}}
	candidates := []Candidate{{ID: "first", SetIndex: 0, SetMeasurements: set, Eligibility: Eligibility{Eligible: true}}, {ID: "middle", SetIndex: 1, SetMeasurements: set, Eligibility: Eligibility{Eligible: true}}, {ID: "rare", SetIndex: 2, SetMeasurements: set, Eligibility: Eligibility{Eligible: true}}}
	selected := choose(candidates, "threshold_then_rarest", map[string]float64{"threshold": .95}, 1)
	require.NotNil(t, selected)
	require.Equal(t, "first", selected.ID, "when no candidate clears the threshold, policy must fall back to the eligible set")
}

func TestUniformPolicyVariesWithSeed(t *testing.T) {
	candidates := []Candidate{{ID: "a", Eligibility: Eligibility{Eligible: true}}, {ID: "b", Eligibility: Eligibility{Eligible: true}}, {ID: "c", Eligibility: Eligibility{Eligible: true}}}
	first := choose(candidates, "sample_uniform", nil, 1)
	second := choose(candidates, "sample_uniform", nil, 2)
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
	result := gate(measurement, Constraints{MaxGrade: 1}, "A very long sentence with many words that should produce a measurable reading grade for this constraint.")
	require.False(t, result.Eligible)
	require.Contains(t, result.Reason, "max_grade")
}

func TestGateEnforcesRequiredFormat(t *testing.T) {
	measurement := textmetrics.Analyze("plain prose", nil)
	result := gate(measurement, Constraints{RequiredFormat: "bullet_list"}, "plain prose")
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

func TestCreateDocumentOwnsOutlineAndSectionSessions(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	_, err := s.CreateProfile(ctx, Profile{Key: "long-form", Sampler: Sampler{Kind: samplerDirect, K: 1}, Budget: Budget{MaxOutputTokens: 128}})
	require.NoError(t, err)

	doc, err := s.CreateDocument(ctx, Document{Title: "A measured document", ProfileKey: "long-form"}, nil)
	require.NoError(t, err)
	require.Equal(t, "assembled", doc.Status)
	require.Len(t, doc.SectionIDs, 5, "a provider one-line outline must still become the bounded five-section service path")
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
	require.Equal(t, []string{first.Candidates[0].ID}, second.Round.NegativeContext.Pinned)
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
