package planimport

import (
	"context"
	"encoding/json"
	"fmt"

	"swarm-manager/internal/identity"
)

// ImportedRef is a created backlog item the bridge reports back.
type ImportedRef struct {
	Kind  string `json:"kind"`
	Name  string `json:"name"`
	Title string `json:"title"`
}

// BatchLander lands a batch JSON payload atomically (all-or-nothing) and returns
// the created item refs. *backlog.Handler satisfies it via an adapter, so this
// package need not import backlog.
type BatchLander interface {
	ImportBatch(ctx context.Context, payloadJSON string, prov identity.Provenance) ([]ImportedRef, error)
}

// Service fetches an authored plan and lands its phases as a provenance-stamped
// linear depends_on chain via the existing atomic batch-create.
type Service struct {
	fetcher PlanFetcher
	lander  BatchLander
}

// NewService wires the plan fetcher and the batch lander.
func NewService(fetcher PlanFetcher, lander BatchLander) *Service {
	return &Service{fetcher: fetcher, lander: lander}
}

// Result reports what an import produced.
type Result struct {
	Slug  string        `json:"slug"`
	Items []ImportedRef `json:"items"`
	Count int           `json:"count"`
}

// Import fetches the plan, translates its phases, and lands them atomically.
func (s *Service) Import(ctx context.Context, planID string, prov identity.Provenance) (Result, error) {
	plan, err := s.fetcher.GetPlan(ctx, planID)
	if err != nil {
		return Result{}, err
	}
	payload, err := Translate(plan)
	if err != nil {
		return Result{}, err
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return Result{}, fmt.Errorf("planimport: marshal payload: %w", err)
	}
	refs, err := s.lander.ImportBatch(ctx, string(payloadJSON), prov)
	if err != nil {
		return Result{}, err
	}
	return Result{Slug: plan.GetSlug(), Items: refs, Count: len(refs)}, nil
}
