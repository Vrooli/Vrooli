package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/storage"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/development"
	"swarm-manager/internal/transitionrunner"
	"swarm-manager/internal/transitions"
)

func (s *Server) registerDevelopmentRoutes(repoRoot string) {
	if s.eventDB == nil {
		return
	}
	loadWorkItem := func(ctx context.Context, ref string) (backlog.BacklogItem, error) {
		kind, name, _ := strings.Cut(ref, "/")
		root, err := s.fileRoots.Pick(ctx, storage.ClassData)
		if err != nil {
			return backlog.BacklogItem{}, err
		}
		// Use the routed root even for this read: an adaptive approval must not bind a
		// live backlog item merely because its fixture was missing.
		item, err := backlog.NewFileStore(root).LoadItem(backlog.BacklogKind(kind), name)
		if errors.Is(err, backlog.ErrNotFound) {
			return backlog.BacklogItem{}, development.ErrNotFound
		}
		if err != nil {
			return backlog.BacklogItem{}, err
		}
		return item, nil
	}
	validate := func(ctx context.Context, ref string) error {
		item, err := loadWorkItem(ctx, ref)
		if err != nil {
			return err
		}
		if item.PlanRef == nil {
			return fmt.Errorf("development admission requires the item's canonical plan: %w", development.ErrDenied)
		}
		if backlog.IsArchived(item) || backlog.IsTerminalStatus(item.Status) || backlog.IsInFlightStatus(item.Status) || backlog.IsReviewStatus(item.Status) {
			return fmt.Errorf("development admission requires an unarchived, idle item: %w", development.ErrDenied)
		}
		if s.backlogHandler == nil {
			return fmt.Errorf("backlog plan authority is unavailable: %w", development.ErrDenied)
		}
		if err := s.backlogHandler.ValidateAcceptedPlan(ctx, item); err != nil {
			return fmt.Errorf("development admission requires an accepted current plan: %w", development.ErrDenied)
		}
		return nil
	}
	// Keep the resolver registry installed even while no product-specific
	// resolver is qualified. Acceptance then reports the missing owner rather
	// than silently treating worker completion as product evidence.
	s.developmentSvc = development.NewService(development.NewSQLiteRepository(s.eventDB), development.Reviewer{RepoRoot: repoRoot}, validate, development.NewResolverRegistry())
	s.developmentSvc.SetPlanReferenceValidator(func(ctx context.Context, ref string, proposed development.PlanReference) error {
		item, err := loadWorkItem(ctx, ref)
		if err != nil {
			return err
		}
		if item.PlanRef == nil || item.PlanRef.Provider != proposed.Provider || item.PlanRef.PlanID != proposed.PlanID || item.PlanRef.Slug != proposed.Slug || item.PlanRef.Role != proposed.Role {
			return fmt.Errorf("development proposal plan does not match the backlog item's canonical plan: %w", development.ErrConflict)
		}
		return nil
	})
	if s.executionSvc != nil {
		s.executionSvc.SetPlanWorkGuard(func(ctx context.Context, ref string) error {
			err := s.guardDevelopmentItem(ctx, ref)
			if errors.Is(err, development.ErrDenied) {
				return apierr.Wrap(err, http.StatusConflict, "adaptive-plan items require their selected execution strategy")
			}
			return err
		})
	}
	if s.backlogHandler != nil {
		s.backlogHandler.SetDevelopmentLookup(func(ctx context.Context, ref string) (bool, error) {
			_, err := s.developmentSvc.Get(ctx, ref)
			if errors.Is(err, development.ErrNotFound) {
				return false, nil
			}
			return err == nil, err
		})
	}
	authentication, err := authn.FromEnvironment(os.Getenv)
	if err != nil {
		panic(fmt.Errorf("configure development operator authentication: %w", err))
	}
	development.RegisterRoutes(s.router, s.developmentSvc, authentication)
}

// A retained adaptive strategy cannot fall through to phased plan execution.
// The internal transition key remains separate so the runner cannot interpret
// an adaptive grant as an ordinary plan slice.
func (s *Server) guardDevelopmentWorkShape(ctx context.Context, definition transitions.Definition, subject string) error {
	if definition.Subject != "backlog-item" || s.developmentSvc == nil {
		return nil
	}
	if definition.Key == transitionrunner.ContractDevelopmentTransition {
		return nil
	}
	return s.guardDevelopmentItem(ctx, subject)
}

func (s *Server) guardDevelopmentItem(ctx context.Context, subject string) error {
	_, err := s.developmentSvc.Get(ctx, subject)
	if errors.Is(err, development.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return fmt.Errorf("adaptive-plan items cannot use phased plan transitions: %w", development.ErrDenied)
}
