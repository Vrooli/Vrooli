// Package programs accepts bounded programs, executes them through a kernel
// seam, and retains a compact provenance-bearing corpus.
package programs

import (
	"context"
	"errors"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	telemetryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/telemetry"
)

var ErrProgramNotFound = errors.New("program not found")

type Runner interface {
	Execute(context.Context, string, string, bool) (Result, error)
}

type MetadataRunner interface {
	ExecuteWithMetadata(context.Context, string, string, string, string, bool) (Result, error)
}

type Result struct {
	Stdout            string
	ContextBytes      int64
	MaterializedBytes int64
	OutputLimitBytes  int64
	Invocations       []Invocation
}

type Options struct {
	Clock           func() time.Time
	Runner          Runner
	ValidateSession func(string) bool
	Store           SQLExecutor
	RecordMemory    func(string, int64)
	Events          interface {
		Append(*telemetryv1.ProgramEvent)
	}
}

type Service struct {
	clock           func() time.Time
	runner          Runner
	validateSession func(string) bool
	recordMemory    func(string, int64)
	events          interface {
		Append(*telemetryv1.ProgramEvent)
	}
	repo Repository
}

func NewService(options Options) *Service {
	clock := options.Clock
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC() }
	}
	repo := Repository(newMemoryRepository())
	if options.Store != nil {
		repo = NewRepository(options.Store)
	}
	return &Service{clock: clock, runner: options.Runner, validateSession: options.ValidateSession, recordMemory: options.RecordMemory, events: options.Events, repo: repo}
}

func (s *Service) Submit(ctx context.Context, sessionID, source string, provenance programsv1.Provenance, includeMaterialized bool) (*programsv1.Program, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, errors.New("session_id is required")
	}
	if s.validateSession != nil && !s.validateSession(sessionID) {
		return nil, errors.New("session not found or reclaimed")
	}
	if strings.TrimSpace(source) == "" {
		return nil, errors.New("source is required")
	}
	if provenance == programsv1.Provenance_PROVENANCE_UNSPECIFIED {
		return nil, errors.New("provenance is required")
	}

	p := &programsv1.Program{Id: "prog_" + uuid.NewString(), SessionId: sessionID, Source: source, Provenance: provenance, CreatedAt: s.clock().UTC().Format(time.RFC3339Nano), Status: "succeeded"}
	var invocations []Invocation
	if s.runner != nil {
		var result Result
		var runErr error
		if metadataRunner, ok := s.runner.(MetadataRunner); ok {
			result, runErr = metadataRunner.ExecuteWithMetadata(ctx, sessionID, p.Id, provenance.String(), source, includeMaterialized)
		} else {
			result, runErr = s.runner.Execute(ctx, sessionID, source, includeMaterialized)
		}
		invocations = result.Invocations
		p.Stdout, p.ContextBytes = result.Stdout, result.ContextBytes
		limit := 4096
		if includeMaterialized {
			limit = 65536
		}
		p.OutputLimitBytes = int64(limit)
		p.Stdout = boundedText(p.Stdout, limit)
		if runErr != nil {
			p.Status = "failed"
			p.FailureDetail = runErr.Error()
			var deadlineErr *DeadlineExceededError
			if errors.As(runErr, &deadlineErr) {
				p.FailureShape = "deadline_exceeded"
			} else {
				p.FailureShape = failureShape(runErr.Error())
			}
		}
		if sampler, ok := s.runner.(MemorySampler); ok && s.recordMemory != nil {
			if bytes, available := sampler.MemoryBytes(sessionID); available {
				s.recordMemory(sessionID, bytes)
			}
		}
	}
	if p.OutputLimitBytes == 0 {
		p.OutputLimitBytes = 4096
	}
	if err := s.repo.Save(ctx, p); err != nil {
		return nil, err
	}
	if s.events != nil {
		for _, invocation := range invocations {
			s.appendEvent(&telemetryv1.ProgramEvent{EventId: uuid.NewString(), OccurredAt: p.CreatedAt, Kind: telemetryv1.EventKind_BINDING_INVOKED, ProgramId: p.Id, SessionId: p.SessionId, BindingId: invocation.BindingID, Effect: invocation.Effect, Provenance: p.Provenance.String()})
		}
		s.appendEvent(&telemetryv1.ProgramEvent{EventId: uuid.NewString(), OccurredAt: p.CreatedAt, Kind: telemetryv1.EventKind_PROGRAM_SUBMITTED, ProgramId: p.Id, SessionId: p.SessionId, Provenance: p.Provenance.String()})
		kind := telemetryv1.EventKind_PROGRAM_SUCCEEDED
		if p.Status == "failed" {
			kind = telemetryv1.EventKind_PROGRAM_FAILED
		}
		event := &telemetryv1.ProgramEvent{EventId: uuid.NewString(), OccurredAt: p.CreatedAt, Kind: kind, ProgramId: p.Id, SessionId: p.SessionId, Provenance: p.Provenance.String(), FailureShape: p.FailureShape, ContextBytes: p.ContextBytes, Reason: p.FailureDetail}
		if p.Status == "failed" {
			event.FailureLocation = failureLocation(p.FailureDetail)
		}
		s.appendEvent(event)
	}
	return clone(p), nil
}

func (s *Service) appendEvent(event *telemetryv1.ProgramEvent) {
	if s.events != nil {
		s.events.Append(event)
	}
}

func (s *Service) Get(ctx context.Context, id string) (*programsv1.Program, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) List(ctx context.Context, sessionID string, includeOperator bool) []*programsv1.Program {
	out, err := s.repo.List(ctx, sessionID, includeOperator)
	if err != nil {
		return nil
	}
	sort.Slice(out, func(i, j int) bool { return out[i].GetCreatedAt() < out[j].GetCreatedAt() })
	return out
}

func (s *Service) MineFailures(ctx context.Context, includeOperator bool) []*programsv1.FailureShape {
	return s.MineFailuresSince(ctx, includeOperator, time.Time{})
}

func (s *Service) MineFailuresSince(ctx context.Context, includeOperator bool, since time.Time) []*programsv1.FailureShape {
	out, err := s.repo.MineFailures(ctx, includeOperator, since)
	if err != nil {
		return nil
	}
	return out
}

var (
	lineError     = regexp.MustCompile(`(?i)(line\s+\d+|field\s+[a-z0-9_.-]+|[a-z_]+error)`)
	locationError = regexp.MustCompile(`(?i)(line\s+\d+|field\s+[a-z0-9_.-]+)`)
)

func failureShape(detail string) string {
	if match := lineError.FindString(strings.ToLower(detail)); match != "" {
		return match
	}
	return "runtime error"
}

func failureLocation(detail string) string {
	if match := locationError.FindString(detail); match != "" {
		return match
	}
	return "program"
}

func boundedText(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func clone(p *programsv1.Program) *programsv1.Program {
	q := *p
	return &q
}
