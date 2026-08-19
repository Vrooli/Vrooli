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
	ID, AuthorizationID, BookID, MandateID, RequestingAgent string
	AmountMinor                                             int64
	Currency, Counterparty                                  string
	Status                                                  Status
	ResolverIdentity                                        string
	CreatedAt, ExpiresAt, ResolvedAt                        time.Time
}

type RelayAttempt struct {
	ID, ApprovalID, Outcome, Error string
	AttemptedAt                    time.Time
}

type Controller interface {
	Get(context.Context, string) (authorization.Record, error)
	Approve(context.Context, string) (authorization.Record, error)
	Release(context.Context, string) (authorization.Record, error)
}

type Relay interface {
	Relay(context.Context, Request) error
}

type TerminalRecorder interface {
	RecordApprovalTerminal(context.Context, string, authorization.Record, string, string, time.Time) error
}

type Service struct {
	repository Repository
	controller Controller
	relay      Relay
	evidence   TerminalRecorder
	clock      schedule.Clock
}

func NewService(repository Repository, controller Controller, relay Relay, clock schedule.Clock, evidence ...TerminalRecorder) *Service {
	service := &Service{repository: repository, controller: controller, relay: relay, clock: clock}
	if len(evidence) > 0 {
		service.evidence = evidence[0]
	}
	return service
}

func (s *Service) Admit(ctx context.Context, in authorization.ApprovalAdmission) error {
	if s.repository == nil || s.clock == nil {
		return fmt.Errorf("%w: repository and clock are required", ErrInvalid)
	}
	now := s.clock.Now().UTC()
	request := Request{ID: strings.TrimSpace(in.ID), AuthorizationID: strings.TrimSpace(in.AuthorizationID), BookID: strings.TrimSpace(in.BookID), MandateID: strings.TrimSpace(in.MandateID), RequestingAgent: strings.TrimSpace(in.RequestingAgent), AmountMinor: in.AmountMinor, Currency: strings.ToUpper(strings.TrimSpace(in.Currency)), Counterparty: strings.ToLower(strings.TrimSpace(in.Counterparty)), Status: StatusQueued, CreatedAt: now, ExpiresAt: in.ExpiresAt.UTC()}
	if request.ID == "" || request.AuthorizationID == "" || request.BookID == "" || request.MandateID == "" || request.RequestingAgent == "" || request.AmountMinor <= 0 || request.Currency == "" || request.Counterparty == "" || !request.ExpiresAt.After(now) {
		return fmt.Errorf("%w: complete unexpired authorization projection is required", ErrInvalid)
	}
	created, err := s.repository.Create(ctx, request)
	if err != nil {
		return err
	}
	if created.AuthorizationID != request.AuthorizationID || created.BookID != request.BookID || created.MandateID != request.MandateID || created.RequestingAgent != request.RequestingAgent || created.AmountMinor != request.AmountMinor || created.Currency != request.Currency || created.Counterparty != request.Counterparty || !created.ExpiresAt.Equal(request.ExpiresAt) {
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
	authorizationRecord, err := s.controller.Get(ctx, current.AuthorizationID)
	if err != nil {
		return Request{}, fmt.Errorf("load authorization for approval: %w", err)
	}
	if authorizationRecord.BookID != current.BookID {
		return Request{}, fmt.Errorf("%w: approval and authorization books do not match", ErrInvalid)
	}
	if resolution == StatusApproved {
		authorizationRecord, err = s.controller.Approve(ctx, current.AuthorizationID)
	} else {
		authorizationRecord, err = s.controller.Release(ctx, current.AuthorizationID)
	}
	if err != nil {
		return Request{}, fmt.Errorf("apply authorization resolution: %w", err)
	}
	resolvedAt := s.clock.Now().UTC()
	if resolution == StatusDeclined && s.evidence != nil {
		if err := s.evidence.RecordApprovalTerminal(context.WithoutCancel(ctx), current.ID, authorizationRecord, string(StatusDeclined), resolver, resolvedAt); err != nil {
			return Request{}, fmt.Errorf("record declined attempt evidence: %w", err)
		}
	}
	return s.repository.Resolve(ctx, current.ID, resolution, resolver, resolvedAt.Format(time.RFC3339Nano))
}

func (s *Service) Get(ctx context.Context, id string) (Request, error) {
	return s.repository.Get(ctx, strings.TrimSpace(id))
}

func (s *Service) List(ctx context.Context, status Status, bookID ...string) ([]Request, error) {
	if s.repository == nil {
		return nil, fmt.Errorf("%w: repository is required", ErrInvalid)
	}
	if status != "" && status != StatusQueued && status != StatusApproved && status != StatusDeclined && status != StatusExpired {
		return nil, fmt.Errorf("%w: unsupported approval status %q", ErrInvalid, status)
	}
	book := ""
	if len(bookID) > 0 {
		book = strings.TrimSpace(bookID[0])
	}
	return s.repository.List(ctx, status, book)
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
	authorizationRecord, err := s.controller.Release(ctx, current.AuthorizationID)
	if err != nil {
		return Request{}, fmt.Errorf("release expired authorization: %w", err)
	}
	resolvedAt := s.clock.Now().UTC()
	if s.evidence != nil {
		if err := s.evidence.RecordApprovalTerminal(context.WithoutCancel(ctx), current.ID, authorizationRecord, string(StatusExpired), "system:expiry", resolvedAt); err != nil {
			return Request{}, fmt.Errorf("record expired attempt evidence: %w", err)
		}
	}
	return s.repository.Resolve(ctx, current.ID, StatusExpired, "system:expiry", resolvedAt.Format(time.RFC3339Nano))
}

func (s *Service) RelayAttempts(ctx context.Context, id string) ([]RelayAttempt, error) {
	return s.repository.ListRelayAttempts(ctx, strings.TrimSpace(id))
}

var _ authorization.ApprovalQueue = (*Service)(nil)
