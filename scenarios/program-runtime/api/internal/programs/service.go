// Package programs accepts bounded programs, executes them through a kernel
// seam, and retains a compact provenance-bearing corpus.
package programs

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	telemetryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/telemetry"
)

type Runner interface {
	Execute(context.Context, string, string, bool) (Result, error)
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
	Events          interface {
		Append(*telemetryv1.ProgramEvent)
	}
}

type Service struct {
	mu              sync.RWMutex
	clock           func() time.Time
	runner          Runner
	validateSession func(string) bool
	events          interface {
		Append(*telemetryv1.ProgramEvent)
	}
	programs map[string]*programsv1.Program
}

func NewService(options Options) *Service {
	clock := options.Clock
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC() }
	}
	return &Service{clock: clock, runner: options.Runner, validateSession: options.ValidateSession, events: options.Events, programs: make(map[string]*programsv1.Program)}
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
	p := &programsv1.Program{Id: "prog_" + uuid.NewString(), SessionId: sessionID, Source: source, Provenance: provenance, CreatedAt: s.clock().Format(time.RFC3339Nano), Status: "succeeded"}
	if p.Provenance == programsv1.Provenance_PROVENANCE_UNSPECIFIED {
		return nil, errors.New("provenance is required")
	}
	s.appendEvent(&telemetryv1.ProgramEvent{EventId: uuid.NewString(), OccurredAt: p.CreatedAt, Kind: telemetryv1.EventKind_PROGRAM_SUBMITTED, ProgramId: p.Id, SessionId: p.SessionId, Provenance: p.Provenance.String()})
	if s.runner != nil {
		result, err := s.runner.Execute(ctx, sessionID, source, includeMaterialized)
		p.Stdout, p.ContextBytes = result.Stdout, result.ContextBytes
		p.OutputLimitBytes = result.OutputLimitBytes
		for _, invocation := range result.Invocations {
			s.appendEvent(&telemetryv1.ProgramEvent{EventId: uuid.NewString(), OccurredAt: p.CreatedAt, Kind: telemetryv1.EventKind_BINDING_INVOKED, ProgramId: p.Id, SessionId: p.SessionId, BindingId: invocation.BindingID, Effect: invocation.Effect, Provenance: p.Provenance.String()})
		}
		limit := 4096
		if includeMaterialized {
			limit = 65536
		}
		p.OutputLimitBytes = int64(limit)
		p.Stdout = boundedText(p.Stdout, limit)
		if err != nil {
			p.Status = "failed"
			p.FailureDetail = err.Error()
			p.FailureShape = failureShape(err.Error())
		}
	}
	s.mu.Lock()
	s.programs[p.Id] = clone(p)
	s.mu.Unlock()
	if s.events != nil {
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

func (s *Service) Get(id string) (*programsv1.Program, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.programs[id]
	if !ok {
		return nil, errors.New("program not found")
	}
	return clone(p), nil
}

func (s *Service) List(sessionID string, includeOperator bool) []*programsv1.Program {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*programsv1.Program, 0)
	for _, p := range s.programs {
		if sessionID != "" && p.SessionId != sessionID {
			continue
		}
		if !includeOperator && p.Provenance == programsv1.Provenance_PROVENANCE_OPERATOR {
			continue
		}
		out = append(out, clone(p))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt < out[j].CreatedAt })
	return out
}

func (s *Service) MineFailures(includeOperator bool) []*programsv1.FailureShape {
	counts := map[string]int64{}
	for _, p := range s.List("", includeOperator) {
		if p.Status == "failed" {
			counts[p.FailureShape]++
		}
	}
	out := make([]*programsv1.FailureShape, 0, len(counts))
	for shape, count := range counts {
		out = append(out, &programsv1.FailureShape{Shape: shape, Count: count})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Shape < out[j].Shape })
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
func clone(p *programsv1.Program) *programsv1.Program { q := *p; return &q }

var _ = fmt.Sprintf
