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
	PreflightSession func(context.Context, string, string) []*programsv1.Diagnostic
	RecordUnresolved func(context.Context, string, string, string) error
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
	preflightSession func(context.Context, string, string) []*programsv1.Diagnostic
	recordUnresolved func(context.Context, string, string, string) error
	repo             Repository
	eventSequence    atomic.Int64

	// terminalWaiters is the notification side of WaitForProgram. One API
	// process owns the runner, so every terminal transition happens in this
	// process and an in-process broadcast is complete — no store polling, and
	// no client-side loop.
	waiterMu        sync.Mutex
	terminalWaiters map[string][]chan struct{}
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
	return &Service{clock: clock, runner: options.Runner, validateSession: options.ValidateSession, recordMemory: options.RecordMemory, executionBudget: options.ExecutionBudget, chargeExecution: options.ChargeExecution, libraryVersion: options.LibraryVersion, events: options.Events, preflight: options.Preflight, preflightSession: options.PreflightSession, recordUnresolved: options.RecordUnresolved, repo: repo}
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
	if s.preflightSession != nil {
		diagnostics = s.preflightSession(ctx, sessionID, source)
	} else if s.preflight != nil {
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
					_ = s.recordUnresolved(ctx, sessionID, diagnostic.GetName(), provenance.String())
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
	// A synchronous submission is bounded well inside the HTTP write deadline.
	// Without this the handler could outlive the deadline, the connection was
	// severed mid-write, and the caller saw `unexpected EOF` — a string outside
	// the closed failure-cause vocabulary that named neither the limit nor the
	// remedy. Exceeding the bound is now a typed `deadline_exceeded` that names
	// the async path, and the program keeps running under its session budget
	// rather than being discarded.
	syncCtx, cancel := context.WithTimeout(ctx, SyncExecutionBudget)
	defer cancel()
	done := make(chan struct{})
	// #nosec G118 -- the goroutine is bounded by the session execution budget.
	go func() {
		defer close(done)
		s.execute(context.WithoutCancel(ctx), p, includeMaterialized)
	}()
	select {
	case <-done:
		return clone(p), nil, nil
	case <-syncCtx.Done():
		pending := clone(p)
		return pending, nil, &SyncDeadlineExceededError{Limit: SyncExecutionBudget, ProgramID: p.Id}
	}
}

// SyncExecutionBudget bounds a synchronous submission. It mirrors
// budgets.SyncSubmit; the value is duplicated here rather than imported to keep
// internal/programs free of a dependency on the handler-side budget package,
// and TestSyncExecutionBudgetMatchesTheLadder pins the two together.
const SyncExecutionBudget = 2 * time.Minute

// SyncDeadlineExceededError reports that a synchronous submission outran its
// bound. The program is still running: it is addressable by id through
// WaitForProgram, which is what the message tells the caller to do.
type SyncDeadlineExceededError struct {
	Limit     time.Duration
	ProgramID string
}

func (e *SyncDeadlineExceededError) Error() string {
	return fmt.Sprintf(
		"deadline_exceeded: synchronous submission exceeded %s; the program is still running as %s — "+
			"resubmit with --async and block once with `programs wait`",
		e.Limit, e.ProgramID)
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
		s.notifyTerminal(p.Id)
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
	s.notifyTerminal(p.Id)
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

// IsTerminal reports whether a status can still change.
func IsTerminal(status programsv1.ProgramStatus) bool {
	switch status {
	case programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED,
		programsv1.ProgramStatus_PROGRAM_STATUS_FAILED,
		programsv1.ProgramStatus_PROGRAM_STATUS_CANCELLED:
		return true
	default:
		return false
	}
}

// Wait blocks until the program reaches a terminal state, the deadline
// elapses, or the caller's context is cancelled. The returned bool reports
// whether the program is terminal; false means the deadline arrived first and
// the caller may wait again on the same id.
//
// This is the primitive that replaced a 50ms client-side GetProgram loop. The
// wait is registered *before* the state is re-read, so a program that
// completes between the read and the registration still wakes the waiter
// rather than hanging until the deadline.
func (s *Service) Wait(ctx context.Context, id string, timeout time.Duration) (*programsv1.Program, bool, error) {
	notify := s.registerTerminalWaiter(id)
	defer s.releaseTerminalWaiter(id, notify)

	program, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, false, err
	}
	if IsTerminal(program.GetStatus()) {
		return program, true, nil
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-notify:
		program, err = s.repo.Get(ctx, id)
		if err != nil {
			return nil, false, err
		}
		return program, IsTerminal(program.GetStatus()), nil
	case <-timer.C:
		program, err = s.repo.Get(ctx, id)
		if err != nil {
			return nil, false, err
		}
		return program, IsTerminal(program.GetStatus()), nil
	case <-ctx.Done():
		return nil, false, ctx.Err()
	}
}

func (s *Service) registerTerminalWaiter(id string) chan struct{} {
	notify := make(chan struct{}, 1)
	s.waiterMu.Lock()
	defer s.waiterMu.Unlock()
	if s.terminalWaiters == nil {
		s.terminalWaiters = make(map[string][]chan struct{})
	}
	s.terminalWaiters[id] = append(s.terminalWaiters[id], notify)
	return notify
}

func (s *Service) releaseTerminalWaiter(id string, notify chan struct{}) {
	s.waiterMu.Lock()
	defer s.waiterMu.Unlock()
	waiters := s.terminalWaiters[id]
	for index, candidate := range waiters {
		if candidate == notify {
			s.terminalWaiters[id] = append(waiters[:index], waiters[index+1:]...)
			break
		}
	}
	if len(s.terminalWaiters[id]) == 0 {
		delete(s.terminalWaiters, id)
	}
}

// notifyTerminal wakes every waiter on a program that just reached a terminal
// state. Sends are non-blocking onto buffered channels, so a slow or departed
// waiter can never stall the execution path that produced the result.
func (s *Service) notifyTerminal(id string) {
	s.waiterMu.Lock()
	waiters := append([]chan struct{}(nil), s.terminalWaiters[id]...)
	s.waiterMu.Unlock()
	for _, notify := range waiters {
		select {
		case notify <- struct{}{}:
		default:
		}
	}
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

func (s *Service) MineUnresolvedBindings(ctx context.Context, includeOperator bool) []*programsv1.UnresolvedBindingShape {
	out, err := s.repo.MineUnresolvedBindings(ctx, includeOperator)
	if err != nil {
		return nil
	}
	return out
}

var locationError = regexp.MustCompile(`(?i)(line\s+\d+|field\s+[a-z0-9_.-]+)`)

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
		// A connection severed by a deadline is a transport failure, not a
		// defect in the program. These four shapes all reached the corpus as
		// `kernel_runtime` with a raw Python exception in the detail, which
		// pointed a reader at their own source instead of at the boundary that
		// actually failed.
		{"remotedisconnected", "bridge_transport", programsv1.FailureCause_FAILURE_CAUSE_BRIDGE_TRANSPORT},
		{"remote end closed connection", "bridge_transport", programsv1.FailureCause_FAILURE_CAUSE_BRIDGE_TRANSPORT},
		{"unexpected eof", "bridge_transport", programsv1.FailureCause_FAILURE_CAUSE_BRIDGE_TRANSPORT},
		{"connection reset", "bridge_transport", programsv1.FailureCause_FAILURE_CAUSE_BRIDGE_TRANSPORT},
		// `deadline_exceeded` is checked late because several needles above are
		// substrings of ordinary deadline messages; a caller that hit a budget
		// should see the budget, not a guess at what it was doing.
		{"deadline_exceeded", "deadline_exceeded", programsv1.FailureCause_FAILURE_CAUSE_DEADLINE_EXCEEDED},
		{"timed out", "deadline_exceeded", programsv1.FailureCause_FAILURE_CAUSE_DEADLINE_EXCEEDED},
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
