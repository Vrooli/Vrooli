package main

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
)

// ownerReadContext owns the shared timeout, provider-readiness, configured
// client, and repository-scope setup for owner-backed read APIs.
func (s *Server) ownerReadContext(parent context.Context, repoID string, timeout time.Duration, owner string, clientConfigured bool) (context.Context, context.CancelFunc, ResolvedRepo, error) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	if s.capabilities == nil || !s.capabilities.IsAvailable(ctx, owner) {
		cancel()
		return nil, nil, ResolvedRepo{}, connect.NewError(connect.CodeUnavailable, fmt.Errorf("%s is not available", owner))
	}
	if !clientConfigured {
		cancel()
		return nil, nil, ResolvedRepo{}, connect.NewError(connect.CodeUnavailable, fmt.Errorf("%s client is not configured", owner))
	}
	repo, err := s.resolveRepositorySelection(ctx, repoID)
	if err != nil {
		cancel()
		return nil, nil, ResolvedRepo{}, err
	}
	return ctx, cancel, repo, nil
}
