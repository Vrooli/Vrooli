package validation_record

import (
	"context"
	"strings"

	"development-toolchain-validator/internal/clock"

	"github.com/google/uuid"
)

// Service is the application-layer surface the handlers depend on. The
// validation_run worker is the only writer (via Append); read paths are
// available to handlers + report composition.
type Service interface {
	Append(ctx context.Context, in AppendInput) (Record, error)
	Get(ctx context.Context, id string) (Record, error)
	List(ctx context.Context, f ListFilter, pageSize int, pageToken string) (ListResult, error)
}

type service struct {
	repo  Repository
	clock clock.Clock
}

// NewService constructs the production Service.
func NewService(repo Repository, clk clock.Clock) Service {
	return &service{repo: repo, clock: clk}
}

var _ Service = (*service)(nil)

func (s *service) Append(ctx context.Context, in AppendInput) (Record, error) {
	in.SubjectID = strings.TrimSpace(in.SubjectID)
	in.GoldenSlug = strings.TrimSpace(in.GoldenSlug)
	if in.SubjectID == "" {
		return Record{}, ErrInvalidRecord{Field: "subject_id", Reason: "required"}
	}
	if in.GoldenSlug == "" {
		return Record{}, ErrInvalidRecord{Field: "golden_slug", Reason: "required"}
	}
	if in.TupleKind == TupleKindUnspecified {
		return Record{}, ErrInvalidRecord{Field: "tuple_kind", Reason: "required"}
	}
	if in.Verdict == VerdictUnspecified {
		return Record{}, ErrInvalidRecord{Field: "verdict", Reason: "required"}
	}
	now := s.clock.Now().UTC()
	if in.EndedAt.IsZero() {
		in.EndedAt = now
	}
	if in.StartedAt.IsZero() {
		in.StartedAt = in.EndedAt
	}
	durationMS := in.EndedAt.Sub(in.StartedAt).Milliseconds()
	if durationMS < 0 {
		durationMS = 0
	}
	r := Record{
		ID:                           uuid.NewString(),
		TupleKind:                    in.TupleKind,
		SubjectID:                    in.SubjectID,
		GoldenSlug:                   in.GoldenSlug,
		StartedAt:                    in.StartedAt.UTC(),
		EndedAt:                      in.EndedAt.UTC(),
		DurationMS:                   durationMS,
		TokensUsed:                   in.TokensUsed,
		CostUSDMicro:                 in.CostUSDMicro,
		Verdict:                      in.Verdict,
		DiffHash:                     in.DiffHash,
		DiffPathCount:                in.DiffPathCount,
		AgentManagerRunID:            in.AgentManagerRunID,
		ManifestTemplateVersionAtRun: in.ManifestTemplateVersionAtRun,
		ManifestSkillVersionAtRun:    in.ManifestSkillVersionAtRun,
		ErrorMessage:                 in.ErrorMessage,
		ToolDetail:                   in.ToolDetail,
		ToolRawOutput:                in.ToolRawOutput,
	}
	if err := s.repo.Append(ctx, r); err != nil {
		return Record{}, err
	}
	return r, nil
}

func (s *service) Get(ctx context.Context, id string) (Record, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Record{}, ErrInvalidRecord{Field: "id", Reason: "required"}
	}
	return s.repo.Get(ctx, id)
}

func (s *service) List(ctx context.Context, f ListFilter, pageSize int, pageToken string) (ListResult, error) {
	return s.repo.List(ctx, f, pageSize, pageToken)
}
