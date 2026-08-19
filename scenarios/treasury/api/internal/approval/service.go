// Package approval owns the in-scenario human approval queue.
package approval

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/api-core/schedule"
	approvalflow "treasury/internal/approval/flow"
	"treasury/internal/authorization"
)

var (
	ErrInvalid  = errors.New("invalid approval")
	ErrNotFound = errors.New("approval not found")
)

type Status string

const (
	StatusQueued   Status = "queued"
	StatusApproved Status = "approved"
	StatusDeclined Status = "declined"
	StatusExpired  Status = "expired"
)

type Request struct {
	ID, AuthorizationID, MandateID, RequestingAgent string
	AmountMinor                                     int64
	Currency, Counterparty                          string
	Status                                          Status
	ResolverIdentity                                string
	CreatedAt, ExpiresAt, ResolvedAt                time.Time
}

type RelayAttempt struct {
	ID, ApprovalID, Outcome, Error string
	AttemptedAt                    time.Time
}

type Controller interface {
	Approve(context.Context, string) (authorization.Record, error)
	Release(context.Context, string) (authorization.Record, error)
}

type Relay interface {
	Relay(context.Context, Request) error
}

type Service struct {
	repository Repository
	controller Controller
	relay      Relay
	clock      schedule.Clock
}

func NewService(repository Repository, controller Controller, relay Relay, clock schedule.Clock) *Service {
	return &Service{repository: repository, controller: controller, relay: relay, clock: clock}
}

func (s *Service) Admit(ctx context.Context, in authorization.ApprovalAdmission) error {
	if s.repository == nil || s.clock == nil {
		return fmt.Errorf("%w: repository and clock are required", ErrInvalid)
	}
	now := s.clock.Now().UTC()
	request := Request{ID: strings.TrimSpace(in.ID), AuthorizationID: strings.TrimSpace(in.AuthorizationID), MandateID: strings.TrimSpace(in.MandateID), RequestingAgent: strings.TrimSpace(in.RequestingAgent), AmountMinor: in.AmountMinor, Currency: strings.ToUpper(strings.TrimSpace(in.Currency)), Counterparty: strings.ToLower(strings.TrimSpace(in.Counterparty)), Status: StatusQueued, CreatedAt: now, ExpiresAt: in.ExpiresAt.UTC()}
	if request.ID == "" || request.AuthorizationID == "" || request.MandateID == "" || request.RequestingAgent == "" || request.AmountMinor <= 0 || request.Currency == "" || request.Counterparty == "" || !request.ExpiresAt.After(now) {
		return fmt.Errorf("%w: complete unexpired authorization projection is required", ErrInvalid)
	}
	created, err := s.repository.Create(ctx, request)
	if err != nil {
		return err
	}
	if created.AuthorizationID != request.AuthorizationID || created.MandateID != request.MandateID || created.RequestingAgent != request.RequestingAgent || created.AmountMinor != request.AmountMinor || created.Currency != request.Currency || created.Counterparty != request.Counterparty || !created.ExpiresAt.Equal(request.ExpiresAt) {
		return fmt.Errorf("%w: approval id was already used for a different authorization", ErrInvalid)
	}
	attempt := RelayAttempt{ID: request.ID + ":relay:1", ApprovalID: request.ID, Outcome: "skipped", AttemptedAt: now}
	if s.relay != nil {
		if err := s.relay.Relay(ctx, request); err != nil {
			attempt.Outcome, attempt.Error = "failed", err.Error()
		} else {
			attempt.Outcome = "sent"
		}
	}
	// Relay evidence is operationally important but cannot be allowed to tear
	// down the already-durable human gate. A storage-wide failure will surface
	// through health checks; the approval remains queued and authoritative.
	_ = s.repository.RecordRelay(ctx, attempt)
	return nil
}

func (s *Service) Resolve(ctx context.Context, id string, resolution Status, resolver string) (Request, error) {
	resolver = strings.TrimSpace(resolver)
	if resolver == "" || resolution != StatusApproved && resolution != StatusDeclined {
		return Request{}, fmt.Errorf("%w: resolver and approved or declined resolution are required", ErrInvalid)
	}
	current, err := s.repository.Get(ctx, strings.TrimSpace(id))
	if err != nil {
		return Request{}, err
	}
	if current.Status != StatusQueued {
		return Request{}, fmt.Errorf("%w: approval is already %s", ErrInvalid, current.Status)
	}
	event := approvalflow.ApprovalApprove
	if resolution == StatusDeclined {
		event = approvalflow.ApprovalDecline
	}
	if _, err := approvalflow.TransitionApproval(approvalflow.ApprovalState{Status: approvalflow.ApprovalStatus(current.Status)}, event); err != nil {
		return Request{}, err
	}
	if s.controller == nil || s.clock == nil {
		return Request{}, fmt.Errorf("%w: controller and clock are required", ErrInvalid)
	}
	if resolution == StatusApproved {
		_, err = s.controller.Approve(ctx, current.AuthorizationID)
	} else {
		_, err = s.controller.Release(ctx, current.AuthorizationID)
	}
	if err != nil {
		return Request{}, fmt.Errorf("apply authorization resolution: %w", err)
	}
	return s.repository.Resolve(ctx, current.ID, resolution, resolver, s.clock.Now().UTC().Format(time.RFC3339Nano))
}

func (s *Service) Get(ctx context.Context, id string) (Request, error) {
	return s.repository.Get(ctx, strings.TrimSpace(id))
}

func (s *Service) Expire(ctx context.Context, id string) (Request, error) {
	if s.controller == nil || s.clock == nil {
		return Request{}, fmt.Errorf("%w: controller and clock are required", ErrInvalid)
	}
	current, err := s.repository.Get(ctx, strings.TrimSpace(id))
	if err != nil {
		return Request{}, err
	}
	if current.Status != StatusQueued || s.clock.Now().UTC().Before(current.ExpiresAt) {
		return Request{}, fmt.Errorf("%w: approval is not queued past expiry", ErrInvalid)
	}
	if _, err := approvalflow.TransitionApproval(approvalflow.ApprovalState{Status: approvalflow.ApprovalStatus(current.Status)}, approvalflow.ApprovalExpire); err != nil {
		return Request{}, err
	}
	if _, err := s.controller.Release(ctx, current.AuthorizationID); err != nil {
		return Request{}, fmt.Errorf("release expired authorization: %w", err)
	}
	return s.repository.Resolve(ctx, current.ID, StatusExpired, "system:expiry", s.clock.Now().UTC().Format(time.RFC3339Nano))
}

func (s *Service) RelayAttempts(ctx context.Context, id string) ([]RelayAttempt, error) {
	return s.repository.ListRelayAttempts(ctx, strings.TrimSpace(id))
}

var _ authorization.ApprovalQueue = (*Service)(nil)
