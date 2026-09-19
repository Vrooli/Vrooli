package earning

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/schedule"
)

const earnedGrantLifetime = 100 * 365 * 24 * time.Hour

var (
	ErrSubmissionNotFound = errors.New("earning submission not found")
	ErrInvalidSubmission  = errors.New("invalid earning submission")
)

type Input struct {
	HolderID    string
	TokenTypeID string
	AmountMinor int64
	Reason      string
	DedupKey    string
}

type Submission struct {
	ID              string
	HolderID        string
	TokenTypeID     string
	AmountMinor     int64
	Reason          string
	DedupKey        string
	AdapterIdentity string
	ActorIdentity   string
	PayloadSummary  string
	GrantID         string
	SubmittedAt     time.Time
	Replayed        bool
}

type GrantRequest struct {
	TokenTypeID    string
	GrantSourceID  string
	Authorizer     string
	HolderID       string
	AmountMinor    int64
	ExpiresAt      time.Time
	IdempotencyKey string
	ActorIdentity  string
}

type GrantOutcome struct{ ID string }

// GrantIssuer is the only outbound mutation seam. The production adapter maps
// it to grants.Service.Create, which is responsible for the journal credit.
type GrantIssuer interface {
	Issue(context.Context, GrantRequest) (GrantOutcome, error)
}

type Service interface {
	Submit(context.Context, string, Input) (Submission, error)
	List(context.Context) ([]Submission, error)
}

type service struct {
	repository Repository
	grants     GrantIssuer
	clock      schedule.Clock
}

func NewService(repository Repository, grants GrantIssuer, clock schedule.Clock) Service {
	if clock == nil {
		clock = schedule.System()
	}
	return &service{repository: repository, grants: grants, clock: clock}
}

func (s *service) Submit(ctx context.Context, adapterIdentity string, input Input) (Submission, error) {
	adapterIdentity = strings.TrimSpace(adapterIdentity)
	normalizeInput(&input)
	if err := validateInput(adapterIdentity, input); err != nil {
		return Submission{}, err
	}
	if s.repository == nil || s.grants == nil {
		return Submission{}, errors.New("earning service dependencies unavailable")
	}

	existing, err := s.repository.GetByDedup(ctx, adapterIdentity, input.DedupKey)
	if err == nil {
		return replay(existing, input), nil
	}
	if !errors.Is(err, ErrSubmissionNotFound) {
		return Submission{}, err
	}

	now := s.clock.Now().UTC()
	submission := Submission{
		ID: deterministicSubmissionID(adapterIdentity, input.DedupKey), HolderID: input.HolderID, TokenTypeID: input.TokenTypeID,
		AmountMinor: input.AmountMinor, Reason: input.Reason, DedupKey: input.DedupKey,
		AdapterIdentity: adapterIdentity, ActorIdentity: adapterIdentity,
		PayloadSummary: summarize(input), SubmittedAt: now,
	}
	outcome, err := s.grants.Issue(ctx, GrantRequest{
		TokenTypeID: input.TokenTypeID, GrantSourceID: "earning:" + submission.ID,
		Authorizer: adapterIdentity, HolderID: input.HolderID, AmountMinor: input.AmountMinor,
		ExpiresAt: now.Add(earnedGrantLifetime), IdempotencyKey: grantIdempotencyKey(adapterIdentity, input.DedupKey),
		ActorIdentity: adapterIdentity,
	})
	if err != nil {
		return Submission{}, fmt.Errorf("issue earning grant: %w", err)
	}
	submission.GrantID = outcome.ID
	stored, created, err := s.repository.Store(ctx, submission)
	if err != nil {
		return Submission{}, err
	}
	if !created {
		return replay(stored, input), nil
	}
	return submission, nil
}

func (s *service) List(ctx context.Context) ([]Submission, error) {
	if s.repository == nil {
		return nil, errors.New("earning repository unavailable")
	}
	return s.repository.List(ctx)
}

func normalizeInput(input *Input) {
	input.HolderID = strings.TrimSpace(input.HolderID)
	input.TokenTypeID = strings.TrimSpace(input.TokenTypeID)
	input.Reason = strings.TrimSpace(input.Reason)
	input.DedupKey = strings.TrimSpace(input.DedupKey)
}

func validateInput(adapterIdentity string, input Input) error {
	switch {
	case adapterIdentity == "":
		return fmt.Errorf("%w: authenticated adapter identity is required", ErrInvalidSubmission)
	case input.HolderID == "":
		return fmt.Errorf("%w: holder id is required", ErrInvalidSubmission)
	case input.TokenTypeID == "":
		return fmt.Errorf("%w: token type id is required", ErrInvalidSubmission)
	case input.AmountMinor <= 0:
		return fmt.Errorf("%w: amount_minor must be positive", ErrInvalidSubmission)
	case input.Reason == "":
		return fmt.Errorf("%w: reason is required", ErrInvalidSubmission)
	case input.DedupKey == "":
		return fmt.Errorf("%w: dedup key is required", ErrInvalidSubmission)
	default:
		return nil
	}
}

func summarize(input Input) string {
	reasonDigest := sha256.Sum256([]byte(input.Reason))
	return fmt.Sprintf("holder=%s token_type=%s amount_minor=%d reason_sha256=%x",
		input.HolderID, input.TokenTypeID, input.AmountMinor, reasonDigest)
}

func grantIdempotencyKey(adapterIdentity, dedupKey string) string {
	digest := sha256.Sum256([]byte(adapterIdentity + "\x00" + dedupKey))
	return fmt.Sprintf("earning:%x", digest)
}

func deterministicSubmissionID(adapterIdentity, dedupKey string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte("token-economy/earning/"+adapterIdentity+"\x00"+dedupKey)).String()
}

func replay(stored Submission, input Input) Submission {
	stored.HolderID = input.HolderID
	stored.TokenTypeID = input.TokenTypeID
	stored.AmountMinor = input.AmountMinor
	stored.Reason = input.Reason
	stored.Replayed = true
	return stored
}
