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
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	telemetryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/telemetry"
)

var ErrProgramNotFound = errors.New("program not found")

type Runner interface {
	Execute(context.Context, string, string, bool) (Result, error)
}

type ExecutionLimits struct {
	Wall time.Duration
	CPU  time.Duration
}

type BudgetedRunner interface {
	ExecuteWithMetadataAndLimits(context.Context, string, string, string, string, bool, ExecutionLimits) (Result, error)
}

type UsageSampler interface {
	CPUTime(string) (time.Duration, bool)
}

type MetadataRunner interface {
	ExecuteWithMetadata(context.Context, string, string, string, string, bool) (Result, error)
}

type Result struct {
	Stdout            string
	ContextBytes      int64
	AgentBytes        int64
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
	ExecutionBudget func(string) (ExecutionLimits, error)
	ChargeExecution func(string, time.Duration, time.Duration) error
	// LibraryVersion is captured at submission time so a running session never
	// silently hot-swaps its program-library view.
	LibraryVersion func(string) string
	Events         interface {
		Append(*telemetryv1.ProgramEvent)
	}
	Preflight        func(string) []*programsv1.Diagnostic
	RecordUnresolved func(context.Context, string, string) error
}

type Service struct {
	clock           func() time.Time
	runner          Runner
	validateSession func(string) bool
	recordMemory    func(string, int64)
	executionBudget func(string) (ExecutionLimits, error)
	chargeExecution func(string, time.Duration, time.Duration) error
	libraryVersion  func(string) string
	events          interface {
		Append(*telemetryv1.ProgramEvent)
	}
	preflight        func(string) []*programsv1.Diagnostic
	recordUnresolved func(context.Context, string, string) error
	repo             Repository
	eventSequence    atomic.Int64
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
	return &Service{clock: clock, runner: options.Runner, validateSession: options.ValidateSession, recordMemory: options.RecordMemory, executionBudget: options.ExecutionBudget, chargeExecution: options.ChargeExecution, libraryVersion: options.LibraryVersion, events: options.Events, preflight: options.Preflight, recordUnresolved: options.RecordUnresolved, repo: repo}
}

func (s *Service) SubmitWithDiagnostics(ctx context.Context, sessionID, source string, provenance programsv1.Provenance, includeMaterialized bool, explain bool, async ...bool) (*programsv1.Program, []*programsv1.Diagnostic, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, nil, errors.New("session_id is required")
	}
	if s.validateSession != nil && !s.validateSession(sessionID) {
		return nil, nil, errors.New("session not found or reclaimed")
	}
	if strings.TrimSpace(source) == "" {
		return nil, nil, errors.New("source is required")
	}
	if provenance == programsv1.Provenance_PROVENANCE_UNSPECIFIED {
		return nil, nil, errors.New("provenance is required")
	}
	diagnostics := []*programsv1.Diagnostic(nil)
	if s.preflight != nil {
		diagnostics = s.preflight(source)
	}
	if explain || hasDiagnosticErrors(diagnostics) {
		now := s.clock().UTC().Format(time.RFC3339Nano)
		p := &programsv1.Program{Id: "prog_" + uuid.NewString(), SessionId: sessionID, Source: source, Provenance: provenance, CreatedAt: now, CompletedAt: now, Status: programsv1.ProgramStatus_PROGRAM_STATUS_FAILED, OutputLimitBytes: 4096, FailureShape: preflightCause(diagnostics), FailureCause: preflightFailureCause(diagnostics)}
		if explain && !hasDiagnosticErrors(diagnostics) {
			p.Status = programsv1.ProgramStatus_PROGRAM_STATUS_ACCEPTED
			p.CompletedAt = ""
			p.FailureShape = ""
			p.FailureCause = programsv1.FailureCause_FAILURE_CAUSE_UNSPECIFIED
		}
		if len(diagnostics) > 0 {
			p.FailureDetail = diagnostics[0].GetMessage()
			for _, diagnostic := range diagnostics {
				// Only an unreachable capability belongs in the ledger. A
				// protected-name assignment names something that resolves, so
				// recording it would corrupt the Act denominator's evidence.
				if IsUnresolvedNameDiagnostic(diagnostic) && s.recordUnresolved != nil {
					_ = s.recordUnresolved(ctx, sessionID, diagnostic.GetName())
				}
			}
		}
		if !explain {
			if err := s.repo.Save(ctx, p); err != nil {
				return nil, diagnostics, err
			}
			s.emitLifecycle(p, telemetryv1.EventKind_PROGRAM_SUBMITTED)
			s.emitLifecycle(p, telemetryv1.EventKind_PROGRAM_FAILED)
		}
		return clone(p), diagnostics, nil
	}

	p, _, err := s.submit(ctx, sessionID, source, provenance, includeMaterialized, async...)
	return p, diagnostics, err
}

func (s *Service) Submit(ctx context.Context, sessionID, source string, provenance programsv1.Provenance, includeMaterialized bool, async ...bool) (*programsv1.Program, error) {
	p, _, err := s.SubmitWithDiagnostics(ctx, sessionID, source, provenance, includeMaterialized, false, async...)
	return p, err
}

func (s *Service) submit(ctx context.Context, sessionID, source string, provenance programsv1.Provenance, includeMaterialized bool, async ...bool) (*programsv1.Program, []*programsv1.Diagnostic, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, nil, errors.New("session_id is required")
	}
	if s.validateSession != nil && !s.validateSession(sessionID) {
		return nil, nil, errors.New("session not found or reclaimed")
	}
	if strings.TrimSpace(source) == "" {
		return nil, nil, errors.New("source is required")
	}
	if provenance == programsv1.Provenance_PROVENANCE_UNSPECIFIED {
		return nil, nil, errors.New("provenance is required")
	}

	now := s.clock().UTC().Format(time.RFC3339Nano)
	p := &programsv1.Program{Id: "prog_" + uuid.NewString(), SessionId: sessionID, Source: source, Provenance: provenance, CreatedAt: now, Status: programsv1.ProgramStatus_PROGRAM_STATUS_ACCEPTED, OutputLimitBytes: 4096}
	if s.libraryVersion != nil {
		p.LibraryVersion = s.libraryVersion(sessionID)
	}
	if includeMaterialized {
		p.OutputLimitBytes = 65536
	}
	if err := s.repo.Save(ctx, p); err != nil {
		return nil, nil, err
	}
	s.emitLifecycle(p, telemetryv1.EventKind_PROGRAM_SUBMITTED)
	s.emitLifecycle(p, telemetryv1.EventKind_PROGRAM_ACCEPTED)
	if len(async) > 0 && async[0] {
		// Async programs outlive the submitting RPC by design. Preserve request
		// values for logging/tracing, but do not let client disconnects cancel
		// accepted work; the execution budget remains the authoritative bound.
		// #nosec G118 -- accepted async work is intentionally detached from the RPC.
		go s.execute(context.WithoutCancel(ctx), p, includeMaterialized)
		return clone(p), nil, nil
	}
	s.execute(ctx, p, includeMaterialized)
	return clone(p), nil, nil
}

func hasDiagnosticErrors(diagnostics []*programsv1.Diagnostic) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic != nil && diagnostic.GetSeverity() == "error" {
			return true
		}
	}
	return false
}

func (s *Service) execute(ctx context.Context, p *programsv1.Program, includeMaterialized bool) {
	started := s.clock()
	p.Status = programsv1.ProgramStatus_PROGRAM_STATUS_RUNNING
	_ = s.repo.Save(context.Background(), p)
	s.emitLifecycle(p, telemetryv1.EventKind_PROGRAM_RUNNING)
	limits := ExecutionLimits{}
	if s.executionBudget != nil {
		if value, err := s.executionBudget(p.SessionId); err != nil {
			s.fail(p, err)
			return
		} else {
			limits = value
		}
	}
	if limits.Wall > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, limits.Wall)
		defer cancel()
	}
	var result Result
	var runErr error
	var cpuBefore time.Duration
	if sampler, ok := s.runner.(UsageSampler); ok {
		cpuBefore, _ = sampler.CPUTime(p.SessionId)
	}
	if s.runner != nil {
		progress := func(update Result) {
			p.Stdout, p.ContextBytes, p.AgentBytes = update.Stdout, update.ContextBytes, update.AgentBytes
			p.Stdout = boundedText(p.Stdout, int(p.OutputLimitBytes))
			_ = s.repo.Save(context.Background(), p)
		}
		if budgeted, ok := s.runner.(BudgetedRunner); ok {
			if streaming, supportsStreaming := s.runner.(ProgressRunner); supportsStreaming {
				result, runErr = streaming.ExecuteWithMetadataAndLimitsAndProgress(ctx, p.SessionId, p.Id, p.Provenance.String(), p.Source, includeMaterialized, limits, progress)
			} else {
				result, runErr = budgeted.ExecuteWithMetadataAndLimits(ctx, p.SessionId, p.Id, p.Provenance.String(), p.Source, includeMaterialized, limits)
			}
		} else if metadataRunner, ok := s.runner.(MetadataRunner); ok {
			result, runErr = metadataRunner.ExecuteWithMetadata(ctx, p.SessionId, p.Id, p.Provenance.String(), p.Source, includeMaterialized)
		} else {
			result, runErr = s.runner.Execute(ctx, p.SessionId, p.Source, includeMaterialized)
		}
	}
	p.Stdout, p.ContextBytes, p.AgentBytes = result.Stdout, result.ContextBytes, result.AgentBytes
	p.Stdout = boundedText(p.Stdout, int(p.OutputLimitBytes))
	p.WallTimeMillis = time.Since(started).Milliseconds()
	if sampler, ok := s.runner.(UsageSampler); ok {
		if cpu, available := sampler.CPUTime(p.SessionId); available {
			if cpu > cpuBefore {
				cpu -= cpuBefore
			}
			p.CpuTimeMillis = cpu.Milliseconds()
		}
	}
	if s.recordMemory != nil {
		if sampler, ok := s.runner.(MemorySampler); ok {
			if bytes, available := sampler.MemoryBytes(p.SessionId); available {
				s.recordMemory(p.SessionId, bytes)
			}
		}
	}
	if runErr != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) && limits.Wall > 0 {
			runErr = fmt.Errorf("wall-clock budget exhausted: ceiling=%s consumed=%s", limits.Wall, time.Duration(p.WallTimeMillis)*time.Millisecond)
		}
		s.fail(p, runErr)
	} else {
		p.Status = programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED
		p.CompletedAt = s.clock().UTC().Format(time.RFC3339Nano)
		_ = s.repo.Save(context.Background(), p)
		s.emitLifecycle(p, telemetryv1.EventKind_PROGRAM_SUCCEEDED)
	}
	for _, invocation := range result.Invocations {
		if s.events != nil {
			s.appendEvent(&telemetryv1.ProgramEvent{EventId: uuid.NewString(), OccurredAt: s.clock().UTC().Format(time.RFC3339Nano), Kind: telemetryv1.EventKind_BINDING_INVOKED, ProgramId: p.Id, SessionId: p.SessionId, BindingId: invocation.BindingID, Effect: invocation.Effect, Provenance: p.Provenance.String()})
		}
	}
	if s.chargeExecution != nil {
		_ = s.chargeExecution(p.SessionId, time.Duration(p.WallTimeMillis)*time.Millisecond, time.Duration(p.CpuTimeMillis)*time.Millisecond)
	}
}

func (s *Service) fail(p *programsv1.Program, runErr error) {
	p.Status = programsv1.ProgramStatus_PROGRAM_STATUS_FAILED
	p.CompletedAt = s.clock().UTC().Format(time.RFC3339Nano)
	p.FailureDetail = runErr.Error()
	var deadlineErr *DeadlineExceededError
	if errors.As(runErr, &deadlineErr) {
		p.FailureShape = "deadline_exceeded"
		p.FailureCause = programsv1.FailureCause_FAILURE_CAUSE_DEADLINE_EXCEEDED
	} else {
		p.FailureShape, p.FailureCause = failureShape(runErr.Error())
	}
	_ = s.repo.Save(context.Background(), p)
	s.emitLifecycle(p, telemetryv1.EventKind_PROGRAM_FAILED)
}

func (s *Service) emitLifecycle(p *programsv1.Program, kind telemetryv1.EventKind) {
	if s.events != nil {
		event := &telemetryv1.ProgramEvent{EventId: uuid.NewString(), OccurredAt: s.clock().UTC().Format(time.RFC3339Nano), Kind: kind, ProgramId: p.Id, SessionId: p.SessionId, Provenance: p.Provenance.String(), FailureShape: p.FailureShape, ContextBytes: p.ContextBytes, Reason: p.FailureDetail}
		if kind == telemetryv1.EventKind_PROGRAM_FAILED {
			event.FailureLocation = failureLocation(p.FailureDetail)
		}
		s.appendEvent(event)
	}
}

func (s *Service) appendEvent(event *telemetryv1.ProgramEvent) {
	if s.events != nil {
		event.Sequence = s.eventSequence.Add(1)
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

func (s *Service) MineRefusals(ctx context.Context, includeOperator bool) []*programsv1.RefusalShape {
	out, err := s.repo.MineRefusals(ctx, includeOperator)
	if err != nil {
		return nil
	}
	return out
}

func (s *Service) MineUnresolvedBindings(ctx context.Context) []*programsv1.UnresolvedBindingShape {
	out, err := s.repo.MineUnresolvedBindings(ctx)
	if err != nil {
		return nil
	}
	return out
}

var (
	locationError = regexp.MustCompile(`(?i)(line\s+\d+|field\s+[a-z0-9_.-]+)`)
)

func failureShape(detail string) (string, programsv1.FailureCause) {
	lower := strings.ToLower(detail)
	classifications := []struct {
		needle string
		name   string
		cause  programsv1.FailureCause
	}{
		{"does not resolve", "unresolved_name", programsv1.FailureCause_FAILURE_CAUSE_UNRESOLVED_NAME},
		{"unknown field", "unknown_field", programsv1.FailureCause_FAILURE_CAUSE_UNKNOWN_FIELD},
		{"accepts named proto fields", "unknown_field", programsv1.FailureCause_FAILURE_CAUSE_UNKNOWN_FIELD},
		{"no determinable primary response", "ambiguous_response", programsv1.FailureCause_FAILURE_CAUSE_AMBIGUOUS_RESPONSE},
		{"unreachable", "unreachable_scenario", programsv1.FailureCause_FAILURE_CAUSE_UNREACHABLE_SCENARIO},
		{"not run eligible", "refused_not_run_eligible", programsv1.FailureCause_FAILURE_CAUSE_REFUSED_NOT_RUN_ELIGIBLE},
		{"run_eligible", "refused_not_run_eligible", programsv1.FailureCause_FAILURE_CAUSE_REFUSED_NOT_RUN_ELIGIBLE},
		{"requires confirmation", "refused_no_grant", programsv1.FailureCause_FAILURE_CAUSE_REFUSED_NO_GRANT},
		{"grant", "refused_no_grant", programsv1.FailureCause_FAILURE_CAUSE_REFUSED_NO_GRANT},
		{"inference spend", "inference_spend_exceeded", programsv1.FailureCause_FAILURE_CAUSE_INFERENCE_SPEND_EXCEEDED},
		{"delegated", "delegated_run_spend_exceeded", programsv1.FailureCause_FAILURE_CAUSE_DELEGATED_RUN_SPEND_EXCEEDED},
		{"syntaxerror", "kernel_syntax", programsv1.FailureCause_FAILURE_CAUSE_KERNEL_SYNTAX},
		{"bridge unavailable", "bridge_transport", programsv1.FailureCause_FAILURE_CAUSE_BRIDGE_TRANSPORT},
		{"transport", "bridge_transport", programsv1.FailureCause_FAILURE_CAUSE_BRIDGE_TRANSPORT},
	}
	for _, item := range classifications {
		if strings.Contains(lower, item.needle) {
			return item.name, item.cause
		}
	}
	return "kernel_runtime", programsv1.FailureCause_FAILURE_CAUSE_KERNEL_RUNTIME
}

func failureLocation(detail string) string {
	if match := locationError.FindString(detail); match != "" {
		return match
	}
	return "program"
}

func boundedText(s string, max int) string {
	if max < 1 {
		return ""
	}
	encoded := []byte(s)
	if len(encoded) <= max {
		return s
	}
	suffix := []byte("…")
	if len(suffix) >= max {
		return string(encoded[:max])
	}
	prefix := encoded[:max-len(suffix)]
	for len(prefix) > 0 && !utf8.Valid(prefix) {
		prefix = prefix[:len(prefix)-1]
	}
	return string(prefix) + string(suffix)
}

func clone(p *programsv1.Program) *programsv1.Program {
	q := *p
	return &q
}

// preflightCause names why a submission was refused before execution. An
// unresolved capability is `unresolved_name`; a refusal that only involves
// names which do resolve — assigning a protected runtime name — is
// `unclassified` rather than being mislabelled as a missing capability.
func preflightCause(diagnostics []*programsv1.Diagnostic) string {
	for _, diagnostic := range diagnostics {
		if IsUnresolvedNameDiagnostic(diagnostic) {
			return "unresolved_name"
		}
	}
	return "unclassified"
}

func preflightFailureCause(diagnostics []*programsv1.Diagnostic) programsv1.FailureCause {
	if preflightCause(diagnostics) == "unresolved_name" {
		return programsv1.FailureCause_FAILURE_CAUSE_UNRESOLVED_NAME
	}
	return programsv1.FailureCause_FAILURE_CAUSE_UNCLASSIFIED
}
