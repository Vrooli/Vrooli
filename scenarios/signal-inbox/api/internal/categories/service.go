package categories

import (
	"context"
	"strings"

	"signal-inbox/internal/inference"
	"signal-inbox/internal/signals"

	"github.com/vrooli/api-core/schedule"
)

type Service struct {
	repo      Repository
	clock     schedule.Clock
	inference inference.Client
}

func NewService(repo Repository, clk schedule.Clock, client inference.Client) *Service {
	return &Service{repo: repo, clock: clk, inference: client}
}

func (s *Service) Bootstrap(ctx context.Context) (Category, error) {
	return s.repo.EnsureUncategorized(ctx, s.clock.Now().UTC())
}

func (s *Service) Create(ctx context.Context, name, description string) (Category, error) {
	return s.repo.Create(ctx, Category{Name: name, Description: description, CreatedAt: s.clock.Now().UTC()})
}

func (s *Service) List(ctx context.Context, includeRetired bool) ([]Category, error) {
	return s.repo.List(ctx, includeRetired)
}

func (s *Service) Rename(ctx context.Context, id, name, description string) (Category, error) {
	return s.repo.Rename(ctx, id, name, description)
}

func (s *Service) Retire(ctx context.Context, id string) (Category, error) {
	uncategorized, err := s.Bootstrap(ctx)
	if err != nil {
		return Category{}, err
	}
	if id == uncategorized.ID {
		return Category{}, ErrReservedCategory{ID: id}
	}
	previous, err := s.repo.LatestConfirmedByCategory(ctx, id)
	if err != nil {
		return Category{}, err
	}
	retired, err := s.repo.Retire(ctx, id, s.clock.Now().UTC())
	if err != nil {
		return Category{}, err
	}
	for _, classification := range previous {
		_, err := s.repo.AppendClassification(ctx, Classification{
			SignalID:            classification.SignalID,
			ProposedCategoryID:  classification.ProposedCategoryID,
			ProposedConfidence:  classification.ProposedConfidence,
			Model:               classification.Model,
			ConfirmedCategoryID: uncategorized.ID,
			State:               StateOverridden,
			Reason:              "confirmed category retired",
			CreatedAt:           s.clock.Now().UTC(),
		})
		if err != nil {
			return Category{}, err
		}
	}
	return retired, nil
}

func (s *Service) GetClassification(ctx context.Context, signalID string) (Classification, bool, error) {
	return s.repo.LatestClassification(ctx, signalID)
}

func (s *Service) Confirm(ctx context.Context, signalID, categoryID string) (Classification, error) {
	category, err := s.repo.Get(ctx, categoryID)
	if err != nil {
		return Classification{}, err
	}
	if !category.Active() {
		return Classification{}, ErrInvalidCategory{Reason: "cannot confirm a retired category"}
	}
	proposal, found, err := s.repo.LatestClassification(ctx, signalID)
	if err != nil {
		return Classification{}, err
	}
	if !found {
		return Classification{}, ErrInvalidCategory{Reason: "signal has no classification proposal"}
	}
	state := StateConfirmed
	if proposal.ProposedCategoryID != categoryID {
		state = StateOverridden
	}
	return s.repo.AppendClassification(ctx, Classification{
		SignalID:            signalID,
		ProposedCategoryID:  proposal.ProposedCategoryID,
		ProposedConfidence:  proposal.ProposedConfidence,
		Model:               proposal.Model,
		ConfirmedCategoryID: categoryID,
		State:               state,
		Reason:              confirmationReason(state),
		CreatedAt:           s.clock.Now().UTC(),
	})
}

// Enrich implements signals.PostCapture. Classification is a derived record:
// its failures are recorded as uncategorized and never make capture fail.
func (s *Service) Enrich(ctx context.Context, signal signals.Signal) error {
	uncategorized, err := s.Bootstrap(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(signal.ExtractedContent) == "" || signal.NeedsAttention {
		return s.recordUncategorized(ctx, signal.ID, uncategorized.ID, "signal has no readable content")
	}
	if signals.InferenceDeferred(ctx) {
		return s.recordUncategorized(ctx, signal.ID, uncategorized.ID, "bulk archive classification deferred for local review")
	}
	categories, err := s.repo.List(ctx, false)
	if err != nil {
		return err
	}
	active := nonReserved(categories)
	if len(active) == 0 {
		return s.recordUncategorized(ctx, signal.ID, uncategorized.ID, "no operator-defined categories are available")
	}
	if len(active) == 1 {
		_, err := s.repo.AppendClassification(ctx, Classification{
			SignalID: signal.ID, ProposedCategoryID: active[0].ID, ProposedConfidence: 1,
			Model: "deterministic.single_category", State: StateProposed, CreatedAt: s.clock.Now().UTC(),
		})
		return err
	}
	if len(active) > 255 {
		return s.recordUncategorized(ctx, signal.ID, uncategorized.ID, "too many active categories for bounded decision")
	}
	if s.inference == nil {
		return s.recordUncategorized(ctx, signal.ID, uncategorized.ID, "ai-gateway classification is unavailable")
	}
	options := make([]inference.ClassificationOption, 0, len(active))
	for _, category := range active {
		description := strings.TrimSpace(category.Name)
		if strings.TrimSpace(category.Description) != "" {
			description += ": " + strings.TrimSpace(category.Description)
		}
		options = append(options, inference.ClassificationOption{ID: category.ID, Description: description})
	}
	result, err := s.inference.Classify(ctx, signal.ExtractedContent, options)
	if err != nil {
		return s.recordUncategorized(ctx, signal.ID, uncategorized.ID, "ai-gateway classification failed: "+err.Error())
	}
	if strings.TrimSpace(result.CategoryID) == "" || result.Confidence < 0 || result.Confidence > 1 || !isActiveCategory(active, result.CategoryID) {
		return s.recordUncategorized(ctx, signal.ID, uncategorized.ID, "ai-gateway returned an invalid category proposal")
	}
	_, err = s.repo.AppendClassification(ctx, Classification{SignalID: signal.ID, ProposedCategoryID: result.CategoryID, ProposedConfidence: result.Confidence, Model: result.Model, State: StateProposed, CreatedAt: s.clock.Now().UTC()})
	return err
}

func (s *Service) recordUncategorized(ctx context.Context, signalID, categoryID, reason string) error {
	_, err := s.repo.AppendClassification(ctx, Classification{SignalID: signalID, ProposedCategoryID: categoryID, State: StateUncategorized, Reason: reason, CreatedAt: s.clock.Now().UTC()})
	if err != nil {
		return err
	}
	return s.repo.EnqueueReclassification(ctx, signalID, reason, s.clock.Now().UTC())
}

func nonReserved(categories []Category) []Category {
	var out []Category
	for _, category := range categories {
		if !category.Reserved && category.Active() {
			out = append(out, category)
		}
	}
	return out
}

func isActiveCategory(categories []Category, id string) bool {
	for _, category := range categories {
		if category.ID == id && category.Active() && !category.Reserved {
			return true
		}
	}
	return false
}

func confirmationReason(state ClassificationState) string {
	if state == StateOverridden {
		return "operator override"
	}
	return "operator confirmation"
}

var _ signals.PostCapture = (*Service)(nil)
