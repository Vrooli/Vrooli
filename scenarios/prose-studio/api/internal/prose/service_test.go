package prose

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

type fakeGateway struct{ calls []GatewayRequest }

func (f *fakeGateway) Generate(_ context.Context, req GatewayRequest) ([]GatewayCandidate, error) {
	callIndex := len(f.calls)
	f.calls = append(f.calls, req)
	out := make([]GatewayCandidate, req.K)
	for i := range out {
		out[i] = GatewayCandidate{Text: []string{"A crisp opening about trust.", "A vivid opening about craft.", "A practical opening about proof.", "A reflective opening about change.", "A measured opening about care."}[(callIndex+i)%5], Provider: "fake", Model: "fake-model", ContextWindow: 32768, TemperatureSupport: "supported", CostMicros: 100, HintOrdinal: i + 1}
	}
	return out, nil
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
	require.Equal(t, "deterministic section feature vectors", assembled.Coherence.(map[string]any)["basis"])
}
