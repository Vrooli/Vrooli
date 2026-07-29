package categories

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"signal-inbox/internal/clock"
	"signal-inbox/internal/inference"
	"signal-inbox/internal/signals"
)

type Service struct {
	repo      Repository
	clock     clock.Clock
	inference inference.Client
}

func NewService(repo Repository, clk clock.Clock, client inference.Client) *Service {
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
	if len(nonReserved(categories)) == 0 {
		return s.recordUncategorized(ctx, signal.ID, uncategorized.ID, "no operator-defined categories are available")
	}
	if s.inference == nil {
		return s.recordUncategorized(ctx, signal.ID, uncategorized.ID, "ai-gateway classification is unavailable")
	}
	output, err := s.inference.Classify(ctx, classificationPrompt(signal.ExtractedContent, categories))
	if err != nil {
		return s.recordUncategorized(ctx, signal.ID, uncategorized.ID, "ai-gateway classification failed: "+err.Error())
	}
	proposal, err := parseProposal(output)
	if err != nil || !isActiveCategory(categories, proposal.CategoryID) {
		reason := "ai-gateway returned an invalid category proposal"
		if err != nil {
			reason += ": " + err.Error()
		}
		return s.recordUncategorized(ctx, signal.ID, uncategorized.ID, reason)
	}
	_, err = s.repo.AppendClassification(ctx, Classification{SignalID: signal.ID, ProposedCategoryID: proposal.CategoryID, ProposedConfidence: proposal.Confidence, Model: proposal.Model, State: StateProposed, CreatedAt: s.clock.Now().UTC()})
	return err
}

func (s *Service) recordUncategorized(ctx context.Context, signalID, categoryID, reason string) error {
	_, err := s.repo.AppendClassification(ctx, Classification{SignalID: signalID, ProposedCategoryID: categoryID, State: StateUncategorized, Reason: reason, CreatedAt: s.clock.Now().UTC()})
	if err != nil {
		return err
	}
	return s.repo.EnqueueReclassification(ctx, signalID, reason, s.clock.Now().UTC())
}

type proposal struct {
	CategoryID string  `json:"category_id"`
	Confidence float64 `json:"confidence"`
	Model      string  `json:"model"`
}

func parseProposal(output string) (proposal, error) {
	var parsed proposal
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		return proposal{}, fmt.Errorf("decode classification proposal: %w", err)
	}
	if strings.TrimSpace(parsed.CategoryID) == "" || parsed.Confidence < 0 || parsed.Confidence > 1 {
		return proposal{}, fmt.Errorf("proposal must name a category id and confidence in [0,1]")
	}
	return parsed, nil
}

func classificationPrompt(content string, categories []Category) string {
	definitions := make([]string, 0, len(categories))
	for _, category := range categories {
		if category.Reserved || !category.Active() {
			continue
		}
		definitions = append(definitions, fmt.Sprintf("- id=%q description=%q", category.ID, category.Description))
	}
	return "Classify the signal into exactly one supplied category. Return JSON only: {\"category_id\": string, \"confidence\": number 0..1, \"model\": string}.\nCategories:\n" + strings.Join(definitions, "\n") + "\nSignal:\n" + content
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
