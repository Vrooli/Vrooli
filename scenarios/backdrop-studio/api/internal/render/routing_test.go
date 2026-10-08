package render

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"backdrop-studio/internal/catalog"
	"backdrop-studio/internal/imageengine"
)

// refusingGenerator stands in for a host with no route to the declared tier —
// no ai-gateway credentials, or no local model installed. It records whether it
// was called at all, because "refused by name" and "refused after spending a
// generation" are different outcomes.
type refusingGenerator struct {
	calls  int
	reason string
}

func (g *refusingGenerator) Generate(context.Context, imageengine.GenerationRequest) (imageengine.GenerationResult, error) {
	g.calls++
	return imageengine.GenerationResult{}, errors.New(g.reason)
}

// A style that declares the frontier tier on a host with no gateway must fail
// naming the tier, and must produce nothing.
//
// The failure mode this closes is the one the hardcoded routing constants were
// written to prevent, in reverse: silently serving a frontier-tier style from a
// local checkpoint would ship a picture wearing a label it did not earn, and
// every downstream consumer — the release path, the disclosure record, the
// operator reading a verdict ledger — would believe it.
func TestAFrontierTierIsRefusedByNameWhenNoLaneMeetsIt(t *testing.T) {
	generator := &refusingGenerator{reason: "ai-gateway: no credential for any image-capable provider"}
	store := NewStoreWithGenerator(&fakeExecutor{}, generator)
	style := catalog.Style{
		ID: "frontier-interior", Strategy: "guided", QualityTier: catalog.TierFrontierModel,
		Subject: "interior", Placements: []string{"full_bleed"}, Treatments: []string{"duotone"},
		Scaffold:   &catalog.ScaffoldBinding{Preset: "arcade", Conditioner: "edge"},
		Generation: &catalog.GenerationBlock{PromptTemplate: "a high-contrast modernist interior"},
	}

	job, err := store.SubmitWithContext(context.Background(), Request{
		Style: style, Surface: testSurface, Placement: "full_bleed", Seed: 3, Count: 1,
	})
	require.Error(t, err)
	require.Empty(t, job.Candidates, "a refused tier must produce no candidate at all")

	var refused *LaneRefusedError
	require.ErrorAs(t, err, &refused, "the refusal must be typed so the handler edge can map it")
	require.Equal(t, catalog.TierFrontierModel, refused.Tier)
	require.Equal(t, style.ID, refused.StyleID)

	// Every lane appears in the message with the reason it did not serve, so an
	// operator can tell an absent credential from an absent model from a
	// miswritten style without reading this code.
	message := err.Error()
	for _, lane := range []string{LaneProcedural, LaneLocalModel, LaneFrontierModel} {
		require.Containsf(t, message, lane, "the refusal must name lane %q it tried", lane)
	}
	require.Contains(t, message, "no credential for any image-capable provider",
		"the refusal must carry the reason the served lane gave, not only that it failed")
}

// The other half of the ceiling: a lane that fails is never replaced by a
// cheaper one. A frontier style must not quietly become a local render.
func TestARefusedFrontierTierNeverFallsBackToALocalModel(t *testing.T) {
	generator := &refusingGenerator{reason: "ai-gateway: unreachable"}
	store := NewStoreWithGenerator(&fakeExecutor{}, generator)
	style := catalog.Style{
		ID: "frontier-only", Strategy: "synthesized", QualityTier: catalog.TierFrontierModel,
		Subject: "celestial", Placements: []string{"full_bleed"}, Treatments: []string{"duotone"},
		Generation: &catalog.GenerationBlock{PromptTemplate: "a gilded celestial chart"},
	}
	_, err := store.SubmitWithContext(context.Background(), Request{
		Style: style, Surface: testSurface, Placement: "full_bleed", Seed: 3, Count: 1,
	})
	require.Error(t, err)
	require.Equal(t, 1, generator.calls,
		"the frontier lane is tried exactly once; a second call would be a downgrade wearing the frontier label")
}

// The rule the risk register names: a procedural tier never reaches a model.
//
// Asserted on the generator's call count rather than on the output, because a
// procedural style that reached a model and then happened to look right is
// still a style that spent a GPU — or a cloud charge — it never declared.
func TestAProceduralTierNeverReachesAModel(t *testing.T) {
	generator := &fakeGenerator{}
	store := NewStoreWithGenerator(&fakeExecutor{}, generator)

	for _, style := range []catalog.Style{
		{ID: "proc", Strategy: "procedural", Subject: "horizon", Placements: []string{"full_bleed"}},
		{ID: "proc-treated", Strategy: "procedural-treated", Subject: "horizon", Placements: []string{"full_bleed"}, Treatments: []string{"duotone"}},
		{
			ID: "vec", Strategy: "vector", Subject: "cartographic", Placements: []string{"full_bleed"},
			Inks: map[string]string{"$brand.primary": "#12327a", "$brand.background": "#efe7d3", "$brand.accent": "#c9432f"},
		},
	} {
		t.Run(style.ID, func(t *testing.T) {
			require.Equal(t, catalog.TierProcedural, style.EffectiveQualityTier())
			job, err := store.SubmitWithContext(context.Background(), Request{
				Style: style, Surface: testSurface, Placement: "full_bleed", Seed: 5, Count: 1,
			})
			require.NoError(t, err)
			require.NotEmpty(t, job.Candidates)
			require.Zero(t, generator.calls, "a procedural tier reached a model")
			require.Equal(t, LaneProcedural, job.Candidates[0].Routing.ServedLane)
		})
	}
}

// The catalog cannot express the disagreement in the first place. This is the
// structural half of the rule above: even if the render path were changed, a
// style that declares a model tier over a generator strategy is refused before
// it can be stored.
func TestTheCatalogRefusesATierItsStrategyCannotReach(t *testing.T) {
	for _, tc := range []struct {
		name, strategy, tier, want string
	}{
		{"a generator cannot spend a model", "procedural", catalog.TierLocalModel, "reaches no model"},
		{"a vector generator cannot spend a model", "vector", catalog.TierFrontierModel, "reaches no model"},
		{"a prompt cannot be drawn without one", "synthesized", catalog.TierProcedural, "drawn by a model"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := catalog.ValidateTierCoherence(catalog.Style{
				ID: "x", Strategy: tc.strategy, QualityTier: tc.tier,
			})
			require.ErrorContains(t, err, tc.want)
		})
	}
}

// A served render records the lane, the model, the execution tier and the cost.
// Each answers a question with no other source after the fact, which is why
// they are on the candidate rather than in a log line.
func TestAServedRenderRecordsItsLaneModelAndCost(t *testing.T) {
	generator := &costedGenerator{cost: 0.042}
	store := NewStoreWithGenerator(&fakeExecutor{}, generator)
	style := catalog.Style{
		ID: "frontier-interior", Strategy: "guided", QualityTier: catalog.TierFrontierModel,
		Subject: "interior", Placements: []string{"full_bleed"}, Treatments: []string{"duotone"},
		Scaffold:   &catalog.ScaffoldBinding{Preset: "arcade", Conditioner: "edge"},
		Generation: &catalog.GenerationBlock{PromptTemplate: "a high-contrast modernist interior"},
	}
	job, err := store.SubmitWithContext(context.Background(), Request{
		Style: style, Surface: testSurface, Placement: "full_bleed", Seed: 11, Count: 1,
	})
	require.NoError(t, err)
	routing := job.Candidates[0].Routing

	require.Equal(t, catalog.TierFrontierModel, routing.DeclaredTier)
	require.Equal(t, LaneFrontierModel, routing.ServedLane)
	require.Equal(t, "frontier/mock", routing.ModelID)
	require.Equal(t, "byok-cloud", routing.ExecutionTier)
	require.InDelta(t, 0.042, routing.CostUSD, 1e-9)
	require.True(t, routing.CostReported)

	// The walk is recorded, not just its verdict: the two cheaper lanes appear
	// with the reason each was not used.
	require.Len(t, routing.Attempts, 2)
	require.Equal(t, LaneProcedural, routing.Attempts[0].Lane)
	require.Equal(t, LaneLocalModel, routing.Attempts[1].Lane)

	// And the frontier lane is the only one that gets BYOK. A local-tier style
	// running the same code path must not.
	require.True(t, generator.last.AllowBYOK, "the frontier lane must be allowed to reach a paid provider")
	require.Equal(t, "any", generator.last.FallbackPolicy)
	require.Contains(t, job.Candidates[0].ProvenanceJSON, `"cost_usd":0.042`)
}

// A backend that reports no cost must not be recorded as free. "This route cost
// nothing" and "nobody reported a cost" are different facts, and only the first
// is a measurement.
func TestAnUnreportedCostIsNotRecordedAsZero(t *testing.T) {
	generator := &fakeGenerator{}
	store := NewStoreWithGenerator(&fakeExecutor{}, generator)
	style := catalog.Style{
		ID: "synth", Strategy: "synthesized", QualityTier: catalog.TierLocalModel,
		Subject: "figure", Placements: []string{"full_bleed"}, Treatments: []string{"duotone"},
		Generation: &catalog.GenerationBlock{PromptTemplate: "a quiet figure"},
	}
	job, err := store.SubmitWithContext(context.Background(), Request{
		Style: style, Surface: testSurface, Placement: "full_bleed", Seed: 13, Count: 1,
	})
	require.NoError(t, err)
	require.False(t, job.Candidates[0].Routing.CostReported)
	require.NotContains(t, job.Candidates[0].ProvenanceJSON, "cost_usd")
}

// costedGenerator is a frontier stand-in: it reports a model, a cloud execution
// tier, and a charge, which is what a real BYOK route returns.
type costedGenerator struct {
	cost float64
	last imageengine.GenerationRequest
	fake fakeGenerator
}

func (g *costedGenerator) Generate(ctx context.Context, req imageengine.GenerationRequest) (imageengine.GenerationResult, error) {
	g.last = req
	result, err := g.fake.Generate(ctx, req)
	if err != nil {
		return result, fmt.Errorf("costed generator: %w", err)
	}
	result.ModelID = "frontier/mock"
	result.Tier = "byok-cloud"
	result.CostUSD, result.CostReported = g.cost, true
	return result, nil
}

// The geometry probe must be asked the lane's own question.
//
// This is a defect caught on the live wire rather than in a unit: a
// frontier-tier render reached OpenRouter, but the canvas had been sized from
// the *local* default model, because the probe sent no routing policy and
// image-tools' selector therefore previewed a run nobody was going to make. The
// generation went out at 768x512 — Stable Diffusion 1.5's cap — to a provider
// with no such limit.
func TestTheGeometryProbeIsAskedTheServedLanesQuestion(t *testing.T) {
	executor := &fakeExecutor{}
	generator := &costedGenerator{}
	store := NewStoreWithGenerator(executor, generator)
	style := catalog.Style{
		ID: "frontier-interior", Strategy: "guided", QualityTier: catalog.TierFrontierModel,
		Subject: "interior", Placements: []string{"full_bleed"}, Treatments: []string{"duotone"},
		Scaffold:   &catalog.ScaffoldBinding{Preset: "arcade", Conditioner: "edge"},
		Generation: &catalog.GenerationBlock{PromptTemplate: "a high-contrast modernist interior"},
	}
	surface := Surface{ID: "web.hero", Width: 1440, Height: 720}
	_, err := store.SubmitWithContext(context.Background(), Request{
		Style: style, Surface: surface, Placement: "full_bleed", Seed: 17, Count: 1,
	})
	require.NoError(t, err)

	require.True(t, executor.lastGeometryRequest.AllowBYOK,
		"the probe must carry the frontier lane's BYOK permission or it previews the local default")
	require.Equal(t, "quality", executor.lastGeometryRequest.QualityPolicy)
	require.Equal(t, "image_to_image", executor.lastGeometryRequest.Operation,
		"a guided style conditions on a scaffold, which selects a different model than text-to-image")

	// And the consequence: a provider that owns its own geometry is asked for
	// the delivery size, not for another model's training resolution.
	require.Equal(t, surface.Width, generator.last.Width)
	require.Equal(t, surface.Height, generator.last.Height)
}

// The local lane asks the opposite question, and gets the local answer.
func TestTheLocalLaneIsSizedFromTheLocalModel(t *testing.T) {
	executor := &fakeExecutor{}
	generator := &fakeGenerator{}
	store := NewStoreWithGenerator(executor, generator)
	style := catalog.Style{
		ID: "synth", Strategy: "synthesized", QualityTier: catalog.TierLocalModel,
		Subject: "figure", Placements: []string{"full_bleed"}, Treatments: []string{"duotone"},
		Generation: &catalog.GenerationBlock{PromptTemplate: "a quiet figure"},
	}
	_, err := store.SubmitWithContext(context.Background(), Request{
		Style: style, Surface: Surface{ID: "web.hero", Width: 1440, Height: 720},
		Placement: "full_bleed", Seed: 19, Count: 1,
	})
	require.NoError(t, err)

	require.False(t, executor.lastGeometryRequest.AllowBYOK, "the local lane must never preview a paid route")
	require.Equal(t, "text_to_image", executor.lastGeometryRequest.Operation)
	require.Equal(t, fakeMaxEdge, generator.last.Width, "a 2:1 hero is capped at the model's long edge")
	require.Equal(t, fakeNativeEdge, generator.last.Height)
}
