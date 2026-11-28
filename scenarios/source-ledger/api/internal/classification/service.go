package classification

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/vrooli/api-core/schedule"
)

const (
	batchSize       = 32
	decisionTimeout = 150 * time.Second
	pollInterval    = 30 * time.Second
)

type Service struct {
	repo        repository
	client      DecideClient
	definitions []definition
}

func NewService(repo repository, client DecideClient, categories []Category) *Service {
	service := &Service{repo: repo, client: client}
	for _, category := range categories {
		service.definitions = append(service.definitions, buildDefinition(cloneCategory(category)))
	}
	return service
}

func cloneCategory(category Category) Category {
	category.Aliases = slices.Clone(category.Aliases)
	category.Examples = slices.Clone(category.Examples)
	category.Sources = slices.Clone(category.Sources)
	return category
}

// Categories exposes the taxonomy used by category recall. The catalog is
// shared across scopes and match lookups always use the caller's scope.
func (s *Service) Categories(context.Context, string) ([]Category, error) {
	categories := make([]Category, 0, len(s.definitions))
	for _, def := range s.definitions {
		categories = append(categories, cloneCategory(def.category))
	}
	return categories, nil
}

func (s *Service) MatchingEntries(ctx context.Context, scope string, ids []string) (map[string][]string, error) {
	matches := make(map[string][]string)
	for _, def := range s.definitions {
		if !slices.Contains(ids, def.category.ID) {
			continue
		}
		entries, err := s.repo.matchingEntries(ctx, scope, def)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			matches[entry] = append(matches[entry], def.category.ID)
		}
	}
	return matches, nil
}

// Coverage reports whether any selected definition still has work in scope.
func (s *Service) Coverage(ctx context.Context, scope string, ids []string) (bool, error) {
	for _, def := range s.definitions {
		if slices.Contains(ids, def.category.ID) {
			pending, err := s.repo.coverage(ctx, scope, def)
			if err != nil || pending {
				return pending, err
			}
		}
	}
	return false, nil
}

func (s *Service) Notices(ctx context.Context, scope, categoryID string, after int64, limit int) ([]Notice, error) {
	if scope == "" || categoryID == "" || after < 0 || limit < 1 || limit > 500 {
		return nil, fmt.Errorf("classification notices require a scope, category, non-negative cursor, and limit from 1 to 500")
	}
	for _, def := range s.definitions {
		if def.category.ID == categoryID {
			return s.repo.notices(ctx, scope, def, after, limit)
		}
	}
	return nil, nil
}

// Run owns the background classifier until cancellation. Each category has an
// independent append cursor and retry queue. Ordinary recall and appends never
// wait for a Decision Models call.
func (s *Service) Run(ctx context.Context, clock schedule.Clock, logger *slog.Logger) {
	if len(s.definitions) == 0 {
		return
	}
	timer := clock.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C():
			for _, def := range s.definitions {
				if ctx.Err() != nil {
					return
				}
				deferred, err := s.runCategory(ctx, def, clock)
				if err != nil && ctx.Err() == nil {
					logger.WarnContext(ctx, "memory classification pass failed", "category", def.category.ID, "error", err)
				} else if deferred > 0 {
					logger.WarnContext(ctx, "memory observations deferred for classification", "category", def.category.ID, "count", deferred)
				}
			}
			timer.Reset(pollInterval)
		}
	}
}

func (s *Service) runCategory(ctx context.Context, def definition, clock schedule.NowSource) (int, error) {
	batch, err := s.repo.prepare(ctx, def, clock.Now(), batchSize)
	if err != nil || len(batch.items) == 0 {
		return 0, err
	}
	requestCtx, cancel := context.WithTimeout(ctx, decisionTimeout)
	defer cancel()
	answers, routeErr := classify(requestCtx, s.client, def.category, batch.items)
	if len(answers) == 0 {
		return 0, routeErr
	}
	if err := s.repo.save(ctx, def, batch, answers, clock.Now()); err != nil {
		return 0, err
	}
	var deferred int
	for _, answer := range answers {
		if answer.failure != "" {
			deferred++
		}
	}
	return deferred, routeErr
}
