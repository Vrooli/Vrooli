package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/usageattribution"

	"connectrpc.com/connect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// ReportFriction files a tool friction finding in the scenario-qa bug inbox
// with its evidence and marks it reported.
func (h *Handler) ReportFriction(ctx context.Context, req *connect.Request[domainpb.ReportFrictionRequest]) (*connect.Response[domainpb.ReportFrictionResponse], error) {
	fingerprint, since, now, err := h.frictionTarget(req.Msg.GetFingerprint(), req.Msg.GetSince())
	if err != nil {
		return nil, err
	}
	finding, err := h.frictionBoard.ReportFriction(ctx, fingerprint, since, now)
	if err != nil {
		return nil, frictionError(err)
	}
	return connect.NewResponse(&domainpb.ReportFrictionResponse{Fingerprint: finding.Fingerprint, Description: finding.Description, Status: finding.Status, BugId: finding.BugID}), nil
}

// DismissFriction sets a tool friction finding aside.
func (h *Handler) DismissFriction(ctx context.Context, req *connect.Request[domainpb.DismissFrictionRequest]) (*connect.Response[domainpb.DismissFrictionResponse], error) {
	fingerprint, since, now, err := h.frictionTarget(req.Msg.GetFingerprint(), req.Msg.GetSince())
	if err != nil {
		return nil, err
	}
	if err := h.frictionBoard.DismissFriction(ctx, fingerprint, since, now); err != nil {
		return nil, frictionError(err)
	}
	return connect.NewResponse(&domainpb.DismissFrictionResponse{Fingerprint: fingerprint, Status: usageattribution.StatusDismissed}), nil
}

// frictionTarget validates the fingerprint and the window it was listed in
// (RFC 3339; empty is the default week).
func (h *Handler) frictionTarget(fingerprint, since string) (string, time.Time, time.Time, error) {
	if h.frictionBoard == nil {
		return "", time.Time{}, time.Time{}, connect.NewError(connect.CodeUnavailable, usageattribution.ErrFrictionUnavailable)
	}
	fingerprint = strings.TrimSpace(fingerprint)
	if fingerprint == "" {
		return "", time.Time{}, time.Time{}, connect.NewError(connect.CodeInvalidArgument, errors.New("fingerprint is required"))
	}
	now := time.Now().UTC()
	var start time.Time
	if since = strings.TrimSpace(since); since != "" {
		var err error
		if start, err = time.Parse(time.RFC3339, since); err != nil {
			return "", time.Time{}, time.Time{}, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("since must be RFC 3339: %w", err))
		}
	}
	start, err := usageattribution.ToolWindowStart(start, now)
	if err != nil {
		return "", time.Time{}, time.Time{}, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return fingerprint, start, now, nil
}

// frictionError maps a report or dismiss failure: an unknown fingerprint is
// NotFound, an unreachable inbox or store is Unavailable, anything else is
// Internal.
func frictionError(err error) error {
	switch {
	case errors.Is(err, usageattribution.ErrUnknownFinding):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, usageattribution.ErrFrictionUnavailable):
		return connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
