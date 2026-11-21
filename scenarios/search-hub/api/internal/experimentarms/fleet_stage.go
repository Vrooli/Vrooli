package experimentarms

import (
	"context"
	"errors"
	"fmt"
	"strings"

	aisearch "github.com/vrooli/ai-go/search"
)

var (
	ErrFleetGenerationUnavailable     = errors.New("Search Hub embedding generation operation is unavailable")
	ErrFleetStageRequiresAllocatedArm = errors.New("Search Hub embedding arm must be allocated before generation stage")
)

// FleetStagePeakEstimator is a trusted capacity-owner policy adapter. It
// derives a conservative peak estimate from the immutable recipe and the
// source-owned point count; request bodies never supply this value.
type FleetStagePeakEstimator interface {
	EstimateFleetStagePeakBytes(context.Context, string, aisearch.EmbeddingRecipe, uint64) (int64, error)
}

// FleetGenerationStage is the single production arm-to-generation operation.
// The source, recipe encoder, admission issuer, byte estimator, and adapter
// must all be composed at lifecycle construction; otherwise the route stays
// unavailable.
type FleetGenerationStage struct {
	registry  *Registry
	source    *ProviderDescriptionFleetSource
	encoder   FleetDescriptionEncoder
	adapter   *aisearch.QdrantFleetAdapter
	estimator FleetStagePeakEstimator
}

func (r *Registry) NewFleetGenerationStage(options aisearch.QdrantFleetOptions, encoder FleetDescriptionEncoder, estimator FleetStagePeakEstimator) (*FleetGenerationStage, error) {
	if r == nil || r.fleetOwnerID == "" || r.fleetAlias == "" {
		return nil, errors.New("fleet generation stage requires a lifecycle-bound Search Hub registry")
	}
	var missing []string
	if encoder == nil {
		missing = append(missing, "full-recipe runtime encoder")
	}
	if options.Admission == nil {
		missing = append(missing, "resource-owned exact-operation admission gate")
	}
	if estimator == nil {
		missing = append(missing, "trusted peak-byte estimator")
	}
	if len(missing) > 0 {
		return nil, errors.Join(aisearch.ErrFleetAdmissionRequired, fmt.Errorf("fleet generation stage dependencies are unavailable: %s", strings.Join(missing, ", ")))
	}
	source, err := r.NewProviderDescriptionFleetSource(encoder)
	if err != nil {
		return nil, err
	}
	options.Source = source
	adapter, err := r.newQdrantFleetAdapter(options)
	if err != nil {
		return nil, err
	}
	return &FleetGenerationStage{registry: r, source: source, encoder: encoder, adapter: adapter, estimator: estimator}, nil
}

// Stage reads the exact recipe from the immutable arm, pins an idempotent
// source snapshot, estimates the bounded candidate peak through the trusted
// owner adapter, then delegates physical writes to the admission-gated fleet
// adapter. No caller can supply the recipe, snapshot, target, or byte estimate.
func (s *FleetGenerationStage) Stage(ctx context.Context, armID string) (aisearch.FleetGeneration, error) {
	if s == nil || s.registry == nil || s.source == nil || s.encoder == nil || s.adapter == nil || s.estimator == nil {
		return aisearch.FleetGeneration{}, ErrFleetGenerationUnavailable
	}
	arm, err := s.registry.Get(ctx, strings.TrimSpace(armID))
	if err != nil {
		return aisearch.FleetGeneration{}, err
	}
	if arm.OwnerID != s.registry.fleetOwnerID {
		return aisearch.FleetGeneration{}, errors.New("embedding arm owner does not match the trusted Search Hub namespace")
	}
	if arm.State != StateAllocated {
		return aisearch.FleetGeneration{}, ErrFleetStageRequiresAllocatedArm
	}
	recipe, err := aisearch.EmbeddingRecipeFromCanonical(arm.RecipeCanonical, strings.TrimPrefix(arm.RecipeID, "sha256:"))
	if err != nil {
		return aisearch.FleetGeneration{}, fmt.Errorf("read immutable arm recipe: %w", err)
	}
	if s.encoder.Recipe().ID() != recipe.ID() {
		return aisearch.FleetGeneration{}, errors.New("trusted provider-description encoder does not match the arm's immutable recipe")
	}
	snapshot, err := s.source.CreateFleetSnapshotForArm(ctx, arm.ArmID, recipe.ID())
	if err != nil {
		return aisearch.FleetGeneration{}, err
	}
	pointCount, err := s.source.ExpectedFleetPointCount(ctx, recipe.ID(), snapshot.ID, snapshot.Watermark)
	if err != nil {
		return aisearch.FleetGeneration{}, fmt.Errorf("count pinned provider-description generation points: %w", err)
	}
	peakBytes, err := s.estimator.EstimateFleetStagePeakBytes(ctx, arm.ArmID, recipe, pointCount)
	if err != nil {
		return aisearch.FleetGeneration{}, fmt.Errorf("estimate admitted Search Hub generation peak: %w", err)
	}
	if peakBytes <= 0 {
		return aisearch.FleetGeneration{}, errors.New("trusted fleet peak estimate must be positive")
	}
	return s.adapter.Stage(ctx, arm.ArmID, recipe, snapshot.ID, peakBytes)
}
