package validation

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"plan-manager/internal/clock"
	planmodel "plan-manager/internal/planmodel"
	"plan-manager/internal/readiness"

	"github.com/google/uuid"
)

// Service is the validation application surface.
type Service interface {
	ResolveReferences(ctx context.Context, planID, phaseID string) (ReferenceReport, error)
	ComputeStaleness(ctx context.Context, planID, phaseID string) (ReferenceReport, error)
	DeriveBaselineScope(ctx context.Context, planID, phaseID string) (BaselineScope, error)
	// CaptureBaseline captures the regression-anchor's baseline snapshot for a plan
	// from its typed anchor intent (scenario + baseline name), shelling
	// git-control-tower through the command-runner seam, then validates the
	// finalized snapshot status before calling it captured. This is the
	// execution-start "capture the before when before is true" action; it degrades
	// honestly (Captured=false + Detail) when the anchor intent is incomplete,
	// git-control-tower is unavailable, or the finalized manifest lacks its V2
	// run anchor — never a fabricated capture.
	CaptureBaseline(ctx context.Context, planID string) (BaselineCapture, error)
	StartValidation(ctx context.Context, planID, phaseID, idempotencyKey string) (ValidationOperation, bool, error)
	GetValidationOperation(ctx context.Context, operationID string, wait bool) (ValidationOperation, error)
	RecoverPending(ctx context.Context) error
	RunValidation(ctx context.Context, planID, phaseID string) (Result, error)
	// LastValidation returns the most recent STORED validation result for a
	// plan/phase (the cheap read path the execution context server uses). ok=false
	// when none has been recorded yet, or when no result store is wired.
	LastValidation(ctx context.Context, planID, phaseID string) (Result, bool, error)
	VerifyDefinitionOfDone(ctx context.Context, planID string) (Result, bool, error)
}

type service struct {
	plans      PlanSource
	resolver   ReferenceResolver
	staleness  StalenessComputer
	runner     CommandRunner
	results    ResultStore
	operations OperationStore
	clock      clock.Clock
	commands   CommandReferenceValidator
	opMu       sync.Mutex
	active     map[string]chan struct{}
	workers    chan struct{}
}

// Deps wires the validation Service. plans is required; resolver/staleness/runner/
// results are optional (nil => that capability degrades to a marked gap, never a
// false positive). A nil Results store means RunValidation still returns its live
// result but caches nothing — LastValidation then reports "no result yet".
type Deps struct {
	Plans      PlanSource
	Resolver   ReferenceResolver
	Staleness  StalenessComputer
	Runner     CommandRunner
	Results    ResultStore
	Operations OperationStore
	Clock      clock.Clock
	Commands   CommandReferenceValidator
}

// NewService constructs the validation Service.
func NewService(d Deps) Service {
	clk := d.Clock
	if clk == nil {
		clk = clock.System{}
	}
	return &service{
		plans:      d.Plans,
		resolver:   d.Resolver,
		staleness:  d.Staleness,
		runner:     d.Runner,
		results:    d.Results,
		operations: d.Operations,
		clock:      clk,
		commands:   d.Commands,
		active:     make(map[string]chan struct{}),
		workers:    make(chan struct{}, maxValidationConcurrency),
	}
}

var _ Service = (*service)(nil)

// scopedReferences returns the references in scope: a phase's references when
// phaseID is set, else the plan-level references.
func (s *service) scopedReferences(p planmodel.Plan, phaseID string) ([]planmodel.Reference, error) {
	if phaseID == "" {
		return p.References, nil
	}
	for _, ph := range p.Phases {
		if ph.ID == phaseID {
			return ph.References, nil
		}
	}
	return nil, ErrPhaseNotFound{PlanID: p.ID, PhaseID: phaseID}
}

func (s *service) ResolveReferences(ctx context.Context, planID, phaseID string) (ReferenceReport, error) {
	p, err := s.plans.GetPlan(ctx, planID)
	if err != nil {
		return ReferenceReport{}, err
	}
	refs, err := s.scopedReferences(p, phaseID)
	if err != nil {
		return ReferenceReport{}, err
	}
	resolved, degraded := s.resolveAll(ctx, refs)
	return ReferenceReport{References: resolved, Degraded: degraded}, nil
}

// resolveAll resolves every reference, degrading honestly. A nil resolver or a
// per-reference error marks that reference UNRESOLVED (or FUTURE preserved) and
// flags the report degraded.
func (s *service) resolveAll(ctx context.Context, refs []planmodel.Reference) ([]planmodel.Reference, bool) {
	out := make([]planmodel.Reference, 0, len(refs))
	degraded := false
	for _, ref := range refs {
		if ref.Future {
			ref.Resolution = planmodel.ResolutionFuture
			out = append(out, ref)
			continue
		}
		if s.resolver == nil {
			ref.Resolution = planmodel.ResolutionUnresolved
			ref.Note = "code-facts unavailable"
			degraded = true
			out = append(out, ref)
			continue
		}
		got, err := s.resolver.Resolve(ctx, ref)
		if err != nil {
			ref.Resolution = planmodel.ResolutionUnresolved
			ref.Note = "resolve failed: " + err.Error()
			degraded = true
			out = append(out, ref)
			continue
		}
		out = append(out, got)
	}
	return out, degraded
}

func (s *service) ComputeStaleness(ctx context.Context, planID, phaseID string) (ReferenceReport, error) {
	report, err := s.ResolveReferences(ctx, planID, phaseID)
	if err != nil {
		return ReferenceReport{}, err
	}
	// The regression anchor's HeadSha is the "before" point against which a
	// still-present reference is graded fresh vs lightly-stale. The platform
	// freshness engine is scenario-artifact scoped today, so per-reference
	// change magnitude remains git-sourced until that substrate exposes a
	// reference-level contract.
	headSha := ""
	if p, gerr := s.plans.GetPlan(ctx, planID); gerr == nil {
		headSha = p.RegressionAnchor.HeadSha
	}
	overall := planmodel.StalenessFresh
	anyKnown := false
	for i := range report.References {
		ref := report.References[i]
		if ref.Future {
			continue // proposed code is never "stale"
		}
		if s.staleness == nil {
			ref.Staleness = planmodel.StalenessUnknown
			report.Degraded = true
			report.References[i] = ref
			continue
		}
		tier, factor, err := s.staleness.Compute(ctx, ref)
		if err != nil {
			ref.Staleness = planmodel.StalenessUnknown
			report.Degraded = true
			report.References[i] = ref
			continue
		}
		// Refine the existence floor: a still-present reference (FRESH) whose code
		// changed since the anchor is LIGHTLY_STALE ("small diffs in referenced
		// code"). DEFINITELY_STALE (moved/deleted) is never downgraded. Absent a
		// HeadSha or a git runner, the floor's FRESH stands — honest, never guessed.
		if tier == planmodel.StalenessFresh {
			if t2, f2, refined := s.gitChangeTier(ctx, headSha, ref); refined {
				tier, factor = t2, f2
			}
		}
		ref.Staleness = tier
		ref.ChangeFactor = factor
		report.References[i] = ref
		anyKnown = true
		if stalenessRank(tier) > stalenessRank(overall) {
			overall = tier
		}
	}
	if !anyKnown {
		overall = planmodel.StalenessUnknown
	}
	report.Overall = overall
	return report, nil
}

// gitChangeTier upgrades a still-present (FRESH) reference to LIGHTLY_STALE when
// its location has changed since the anchor's HeadSha, with a change factor from
// the diff magnitude. Returns refined=false (keep FRESH) when there is no anchor
// sha, no runner, the tool is absent, the ref is not file-backed, or nothing
// changed. Uses `git diff --numstat <sha> -- <target>` because the live
// freshness engine only reports scenario-artifact staleness, not per-reference
// code drift. Empty output (exit 0) means unchanged; non-empty means changed.
func (s *service) gitChangeTier(ctx context.Context, headSha string, ref planmodel.Reference) (planmodel.StalenessTier, float64, bool) {
	headSha = strings.TrimSpace(headSha)
	if headSha == "" || s.runner == nil {
		return "", 0, false
	}
	if ref.Kind != planmodel.ReferenceCode && ref.Kind != planmodel.ReferenceDoc {
		return "", 0, false
	}
	out, err := s.runner(ctx, "git", "diff", "--numstat", headSha, "--", ref.Target)
	if err != nil {
		return "", 0, false // tool absent / bad sha → keep the existence floor
	}
	added, deleted, changed := parseNumstat(string(out))
	if !changed {
		return "", 0, false
	}
	factor := float64(added+deleted) / 200.0
	switch {
	case factor > 1:
		factor = 1
	case factor <= 0:
		factor = 0.05 // changed but tiny — still a non-zero signal
	}
	return planmodel.StalenessLightlyStale, factor, true
}

// parseNumstat sums the added/deleted columns of `git diff --numstat` output.
// Binary files report "-" for both counts and are treated as changed with zero
// line magnitude. changed=false only when there are no data rows at all.
func parseNumstat(out string) (added, deleted int, changed bool) {
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		changed = true
		if a, err := strconv.Atoi(fields[0]); err == nil {
			added += a
		}
		if d, err := strconv.Atoi(fields[1]); err == nil {
			deleted += d
		}
	}
	return added, deleted, changed
}

func (s *service) DeriveBaselineScope(ctx context.Context, planID, phaseID string) (BaselineScope, error) {
	p, err := s.plans.GetPlan(ctx, planID)
	if err != nil {
		return BaselineScope{}, err
	}
	refs, err := s.scopedReferences(p, phaseID)
	if err != nil {
		return BaselineScope{}, err
	}
	return deriveScope(p, refs, effectiveBoundary(p, phaseID)), nil
}

func (s *service) CaptureBaseline(ctx context.Context, planID string) (BaselineCapture, error) {
	p, err := s.plans.GetPlan(ctx, planID)
	if err != nil {
		return BaselineCapture{}, err
	}
	anchor := p.RegressionAnchor
	scenario := strings.TrimSpace(anchor.Scenario)
	name := strings.TrimSpace(anchor.BaselineName)
	// An unconfirmed authoring placeholder (e.g. "<scenario>") is not a real
	// target — degrade honestly so the agent confirms the anchor intent.
	if scenario == "" || strings.ContainsAny(scenario, "<>") {
		return BaselineCapture{BaselineName: name, Detail: "regression-anchor scenario is unset or still a placeholder; confirm the anchor intent before execution"}, nil
	}
	if name == "" {
		name = scenario + "-baseline"
	}
	if s.runner == nil {
		return BaselineCapture{Scenario: scenario, BaselineName: name, Detail: "no command runner configured (git-control-tower unavailable)"}, nil
	}
	reason := "execution-start baseline for plan " + planIdentifier(p)
	if _, err := s.runner(ctx, "git-control-tower", "baseline", "snapshot", "--scenario", scenario, "--name", name, "--reason", reason); err != nil {
		return BaselineCapture{Scenario: scenario, BaselineName: name, Detail: "baseline snapshot failed: " + err.Error()}, nil
	}
	health := s.validateCapturedBaseline(ctx, scenario, name)
	if !health.usable {
		return BaselineCapture{
			Scenario:        scenario,
			BaselineName:    name,
			RunID:           health.runID,
			SchemaVersion:   health.schemaVersion,
			DegradedReasons: health.degradedReasons,
			Detail:          health.detail,
		}, nil
	}
	return BaselineCapture{
		Captured:      true,
		Scenario:      scenario,
		BaselineName:  name,
		RunID:         health.runID,
		SchemaVersion: health.schemaVersion,
		Detail:        fmt.Sprintf("captured baseline %s for %s anchored to Test Genie run %s", name, scenario, health.runID),
	}, nil
}

type baselineCaptureHealth struct {
	usable          bool
	runID           string
	schemaVersion   int
	degradedReasons []string
	detail          string
}

func (s *service) validateCapturedBaseline(ctx context.Context, scenario, name string) baselineCaptureHealth {
	out, err := s.runner(ctx, "git-control-tower", "baseline", "snapshot", "status", "--scenario", scenario, "--name", name, "--wait", "--json")
	if err != nil {
		return baselineCaptureHealth{
			detail: fmt.Sprintf("baseline snapshot started but finalization failed: %v; operation can be resumed by baseline name/run id and %q is not yet a usable oracle", err, name),
		}
	}
	health, parseErr := parseBaselineStatusHealth(out)
	if parseErr != nil {
		return baselineCaptureHealth{
			detail: fmt.Sprintf("baseline snapshot completed but its V2 run anchor was unreadable: %v; treat %q as unusable", parseErr, name),
		}
	}
	return health
}

func parseBaselineStatusHealth(out []byte) (baselineCaptureHealth, error) {
	if len(strings.TrimSpace(string(out))) == 0 {
		return baselineCaptureHealth{}, errors.New("empty baseline status output")
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(out, &root); err != nil {
		return baselineCaptureHealth{}, fmt.Errorf("decode snapshot status JSON: %w", err)
	}
	status := jsonString(root, "status")
	if status != "" && status != "ready" {
		detail := jsonString(root, "error")
		if detail == "" {
			detail = status
		}
		return baselineCaptureHealth{}, fmt.Errorf("snapshot status %s", detail)
	}
	var manifest map[string]json.RawMessage
	if raw := root["baseline"]; len(raw) > 0 {
		_ = json.Unmarshal(raw, &manifest)
	}
	if len(manifest) == 0 {
		return baselineCaptureHealth{}, errors.New("ready snapshot has no baseline manifest")
	}
	schemaVersion := jsonInt(manifest, "schemaVersion", "schema_version")
	var run map[string]json.RawMessage
	if raw := manifest["run"]; len(raw) > 0 {
		_ = json.Unmarshal(raw, &run)
	}
	runID := jsonString(run, "runId", "run_id")
	if schemaVersion != 2 || runID == "" {
		return baselineCaptureHealth{runID: runID, schemaVersion: schemaVersion}, fmt.Errorf("expected schema V2 with one non-empty run anchor (schema=%d run=%q)", schemaVersion, runID)
	}
	return baselineCaptureHealth{usable: true, runID: runID, schemaVersion: schemaVersion}, nil
}

func jsonString(values map[string]json.RawMessage, keys ...string) string {
	for _, key := range keys {
		var value string
		if raw := values[key]; len(raw) > 0 && json.Unmarshal(raw, &value) == nil {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func jsonInt(values map[string]json.RawMessage, keys ...string) int {
	for _, key := range keys {
		var value int
		if raw := values[key]; len(raw) > 0 && json.Unmarshal(raw, &value) == nil {
			return value
		}
	}
	return 0
}

// planIdentifier prefers the human slug for log/reason text, falling back to id.
func planIdentifier(p planmodel.Plan) string {
	if s := strings.TrimSpace(p.Slug); s != "" {
		return s
	}
	return strings.TrimSpace(p.ID)
}

const (
	defaultQueueBudget         = 2 * time.Minute
	defaultExecutionBudget     = 30 * time.Minute
	defaultTransportWaitBudget = 15 * time.Minute
	maxValidationConcurrency   = 4
)

// StartValidation persists the complete child plan before any command is
// dispatched. A repeated scoped idempotency key returns the original operation
// and never creates a second child set.
func (s *service) StartValidation(ctx context.Context, planID, phaseID, idempotencyKey string) (ValidationOperation, bool, error) {
	if s.operations == nil {
		return ValidationOperation{}, false, errors.New("durable validation operation store is unavailable")
	}
	p, err := s.plans.GetPlan(ctx, planID)
	if err != nil {
		return ValidationOperation{}, false, err
	}
	refs, err := s.scopedReferences(p, phaseID)
	if err != nil {
		return ValidationOperation{}, false, err
	}
	checks := compileValidationChecks(p, refs, effectiveBoundary(p, phaseID))
	staleReport, _ := s.ComputeStaleness(ctx, planID, phaseID)
	now := s.now()
	op := ValidationOperation{
		SchemaVersion:              CurrentOperationSchemaVersion,
		ID:                         uuid.NewString(),
		PlanID:                     p.ID,
		PhaseID:                    phaseID,
		IdempotencyKey:             strings.TrimSpace(idempotencyKey),
		Status:                     OperationQueued,
		QueuedAt:                   now,
		QueueBudgetSeconds:         int(defaultQueueBudget.Seconds()),
		ExecutionBudgetSeconds:     int(defaultExecutionBudget.Seconds()),
		TransportWaitBudgetSeconds: int(defaultTransportWaitBudget.Seconds()),
		RecommendedWaitSeconds:     int((defaultQueueBudget + commandExecutionTimeout).Seconds()),
		ScopeFingerprint:           scopeFingerprint(p, phaseID, checks),
		QueueReason:                "awaiting scheduler claim",
		Result: &Result{
			ID: "", PlanID: p.ID, PhaseID: phaseID, Staleness: staleReport.Overall,
			CommandsRun: checkCommands(checks),
		},
	}
	op.Result.ID = op.ID + ":result"
	for i, check := range checks {
		op.Children = append(op.Children, ValidationChild{
			ID: fmt.Sprintf("%s:%d", op.ID, i+1), Check: check, Command: check.Command,
			Oracle: check.Oracle, Status: ChildQueued, QueuedAt: now,
		})
	}
	stored, created, err := s.operations.CreateOperation(ctx, op)
	if err != nil {
		return ValidationOperation{}, false, err
	}
	if !stored.Terminal() {
		s.launchValidation(context.WithoutCancel(ctx), stored.ID)
	}
	return stored, !created, nil
}

func scopeFingerprint(p planmodel.Plan, phaseID string, checks []ValidationCheck) string {
	parts := []string{p.ID, p.ContentHash, phaseID}
	for _, check := range checks {
		parts = append(parts, string(check.Kind)+":"+check.SemanticKey)
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return fmt.Sprintf("sha256:%x", sum[:])
}

func checkCommands(checks []ValidationCheck) []string {
	commands := make([]string, 0, len(checks))
	for _, check := range checks {
		commands = append(commands, check.Command)
	}
	return commands
}

// GetValidationOperation inspects or waits once on a durable operation. A
// transport deadline only ends this attachment; the returned operation remains
// non-terminal and server-owned work continues.
func (s *service) GetValidationOperation(ctx context.Context, operationID string, wait bool) (ValidationOperation, error) {
	op, found, err := s.operations.GetOperation(ctx, strings.TrimSpace(operationID))
	if err != nil {
		return ValidationOperation{}, err
	}
	if !found {
		return ValidationOperation{}, ErrOperationNotFound{ID: operationID}
	}
	if op.Terminal() || !wait {
		if !op.Terminal() {
			s.launchValidation(context.WithoutCancel(ctx), op.ID)
		}
		return op, nil
	}
	done := s.launchValidation(context.WithoutCancel(ctx), op.ID)
	attachmentEnded := false
	select {
	case <-done:
	case <-ctx.Done():
		// Detach without canceling the server-owned execution. A durable reread
		// below is intentionally followed by a typed non-success outcome unless
		// terminal truth won the race.
		attachmentEnded = true
	}
	op, found, err = s.operations.GetOperation(context.WithoutCancel(ctx), op.ID)
	if err != nil {
		return ValidationOperation{}, err
	}
	if !found {
		return ValidationOperation{}, ErrOperationNotFound{ID: operationID}
	}
	if !op.Terminal() && attachmentEnded {
		return op, ErrAttachmentEnded{OperationID: op.ID, Cause: ctx.Err()}
	}
	if !op.Terminal() {
		return op, fmt.Errorf("validation dispatcher ended before durable terminal checkpoint for %s", op.ID)
	}
	return op, nil
}

// RecoverPending reattaches every queued/running record after process restart.
func (s *service) RecoverPending(ctx context.Context) error {
	if s.operations == nil {
		return nil
	}
	operations, err := s.operations.ListNonTerminalOperations(ctx)
	if err != nil {
		return err
	}
	for _, op := range operations {
		changed := false
		for i := range op.Children {
			if op.Children[i].Status == ChildRunning {
				op.Children[i].Status = ChildQueued
				op.Children[i].StartedAt = ""
				op.Children[i].Error = &OperationError{Code: "claim_recovered", Detail: "recovered unfinished child claim after restart"}
				changed = true
			}
		}
		if changed {
			op.Status, op.QueueReason = OperationQueued, "recovered unfinished child claim after restart"
			if err := s.operations.SaveOperation(context.WithoutCancel(ctx), op); err != nil {
				return err
			}
		}
		s.launchValidation(context.WithoutCancel(ctx), op.ID)
	}
	return nil
}

func (s *service) launchValidation(ctx context.Context, operationID string) <-chan struct{} {
	s.opMu.Lock()
	if done, ok := s.active[operationID]; ok {
		s.opMu.Unlock()
		return done
	}
	done := make(chan struct{})
	s.active[operationID] = done
	s.opMu.Unlock()
	go func() {
		defer func() {
			s.opMu.Lock()
			delete(s.active, operationID)
			close(done)
			s.opMu.Unlock()
		}()
		s.executeValidationOperation(ctx, operationID)
	}()
	return done
}

func (s *service) executeValidationOperation(ctx context.Context, operationID string) {
	op, found, err := s.operations.GetOperation(ctx, operationID)
	if err != nil || !found || op.Terminal() {
		return
	}
	// An operation owns at most one running child. This is an intentional
	// per-operation share: it ensures a large operation cannot occupy every
	// global worker while other queued operations are waiting. Crucially, this
	// loop creates no goroutine for queued children; only this operation runner
	// exists until a child has actually claimed a global worker.
	for _, child := range op.Children {
		if child.Status == ChildTerminal {
			continue
		}
		select {
		case s.workers <- struct{}{}:
		case <-ctx.Done():
			return
		}
		if !s.markOperationClaimed(operationID) {
			<-s.workers
			return
		}
		executionBudget := time.Duration(op.ExecutionBudgetSeconds) * time.Second
		if executionBudget <= 0 {
			executionBudget = defaultExecutionBudget
		}
		// The execution clock begins at the durable claim boundary. Queue
		// residence cannot consume either this operation budget or the child
		// dispatch timeout.
		execCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), executionBudget)
		s.runValidationChild(execCtx, operationID, child.ID)
		cancel()
		<-s.workers
	}

	op, found, err = s.operations.GetOperation(context.WithoutCancel(ctx), operationID)
	if err != nil || !found {
		return
	}
	// A failed child checkpoint must never be papered over by a terminal parent.
	// Leave the operation resumable so a later attachment or restart can replay
	// only the children whose terminal truth was not durably recorded.
	for _, child := range op.Children {
		if child.Status != ChildTerminal {
			return
		}
	}
	resultID := op.ID + ":result"
	if op.Result != nil && op.Result.ID != "" {
		resultID = op.Result.ID
	}
	result := Result{
		ID: resultID, PlanID: op.PlanID, PhaseID: op.PhaseID,
		Verdict: verdictFromChildren(op.Children), RanAt: s.now(),
	}
	if op.Result != nil {
		result.Staleness = op.Result.Staleness
		result.CommandsRun = append([]string(nil), op.Result.CommandsRun...)
	}
	for _, child := range op.Children {
		result.Detail = joinDetails(result.Detail, child.Detail)
	}
	if p, perr := s.plans.GetPlan(context.WithoutCancel(ctx), op.PlanID); perr == nil {
		readinessResult := readiness.Evaluate(context.WithoutCancel(ctx), p, readiness.Options{
			PhaseID: op.PhaseID, Mode: readiness.PreflightMode(),
			CommandValidator: s.readinessCommandValidator(), ReferenceResolver: s.resolver,
		})
		result.CommandFindings = append(result.CommandFindings, commandFindingsFromReadiness(readinessResult.Findings)...)
		result.Verdict = combineValidationVerdicts(result.Verdict, verdictFromReadiness(readinessResult.Verdict))
		result.Detail = joinDetails(result.Detail, readinessResult.Detail)
	} else {
		result.Verdict = VerdictUnknown
		result.Detail = joinDetails(result.Detail, "validation readiness unavailable: "+perr.Error())
	}

	persistCtx := context.Background()
	resultPersisted := false
	if s.results != nil {
		if err := s.results.SaveResult(persistCtx, result); err != nil {
			op.Error = &OperationError{Code: "result_persistence_failed", Detail: err.Error()}
			result.Verdict = VerdictUnknown
		} else {
			resultPersisted = true
			if committed, found, err := s.results.GetResult(persistCtx, result.ID); err == nil && found {
				// A crash replay cannot replace the first committed terminal truth.
				result = committed
			}
		}
	}
	if !resultPersisted && s.results != nil {
		// Result persistence is the terminal commit boundary. Keep the operation
		// non-terminal and recoverable rather than publishing success-shaped state.
		op.Error = &OperationError{Code: "result_persistence_failed", Detail: result.Detail}
		_ = s.operations.SaveOperation(persistCtx, op)
		return
	}
	op.Status = OperationTerminal
	op.TerminalAt = s.now()
	op.Result = &result
	if resultPersisted {
		op.ResultRef = result.ID
	}
	_ = s.operations.SaveOperation(persistCtx, op)
}

func (s *service) markOperationClaimed(operationID string) bool {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	op, found, err := s.operations.GetOperation(context.Background(), operationID)
	if err != nil || !found || op.Terminal() {
		return false
	}
	if op.StartedAt == "" {
		op.StartedAt = s.now()
		op.Attempt++
	}
	op.Status = OperationRunning
	op.QueueReason = ""
	return s.operations.SaveOperation(context.Background(), op) == nil
}

func (s *service) expireValidationChild(operationID, childID string, cause error) {
	s.updateChild(operationID, childID, func(child *ValidationChild) {
		child.Status = ChildTerminal
		child.TerminalAt = s.now()
		child.Verdict = VerdictUnknown
		child.Detail = fmt.Sprintf("unknown %s: execution budget ended before dispatch", child.Command)
		child.Error = &OperationError{Code: "execution_timeout", Detail: cause.Error()}
	})
}

func (s *service) runValidationChild(ctx context.Context, operationID, childID string) {
	child, ok := s.updateChild(operationID, childID, func(child *ValidationChild) {
		child.Status = ChildRunning
		child.Attempt++
		if child.StartedAt == "" {
			child.StartedAt = s.now()
		}
	})
	if !ok {
		return
	}
	name, args := splitCommand(child.Command)
	var output []byte
	var runErr error
	if name == "" {
		runErr = errors.New("empty command")
	} else if s.runner == nil {
		runErr = fmt.Errorf("command %q: %w", name, ErrToolNotFound)
	} else {
		output, runErr = s.runner(ctx, name, args...)
	}
	classified := classifyCommandRun(child.Command, child.Oracle, runErr)
	s.updateChild(operationID, childID, func(child *ValidationChild) {
		child.Status = ChildTerminal
		child.TerminalAt = s.now()
		child.Verdict = classified.verdict
		child.Detail = classified.detail
		child.ExternalID = externalRunID(output)
		child.Error = operationErrorFor(runErr)
	})
}

func (s *service) updateChild(operationID, childID string, mutate func(*ValidationChild)) (ValidationChild, bool) {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	op, found, err := s.operations.GetOperation(context.Background(), operationID)
	if err != nil || !found || op.Terminal() {
		return ValidationChild{}, false
	}
	for i := range op.Children {
		if op.Children[i].ID != childID {
			continue
		}
		mutate(&op.Children[i])
		if err := s.operations.SaveOperation(context.Background(), op); err != nil {
			return ValidationChild{}, false
		}
		return op.Children[i], true
	}
	return ValidationChild{}, false
}

func verdictFromChildren(children []ValidationChild) Verdict {
	passed := 0
	unknown := false
	for _, child := range children {
		if !child.Oracle {
			continue
		}
		switch child.Verdict {
		case VerdictFail:
			return VerdictFail
		case VerdictPass:
			passed++
		default:
			unknown = true
		}
	}
	if unknown || passed == 0 {
		return VerdictUnknown
	}
	return VerdictPass
}

func operationErrorFor(err error) *OperationError {
	if err == nil {
		return nil
	}
	code := "command_failed"
	switch {
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		code = "execution_timeout"
	case errors.Is(err, ErrToolNotFound):
		code = "tool_unavailable"
	default:
		var exitErr CommandExitError
		if errors.As(err, &exitErr) && exitErr.Code == 2 {
			code = "not_comparable"
		}
	}
	return &OperationError{Code: code, Detail: err.Error()}
}

func externalRunID(output []byte) string {
	var value any
	if len(output) == 0 || json.Unmarshal(output, &value) != nil {
		return ""
	}
	return findRunID(value)
}

func findRunID(value any) string {
	switch typed := value.(type) {
	case map[string]any:
		for _, key := range []string{"runId", "run_id"} {
			if id, ok := typed[key].(string); ok && strings.TrimSpace(id) != "" {
				return strings.TrimSpace(id)
			}
		}
		for _, nested := range typed {
			if id := findRunID(nested); id != "" {
				return id
			}
		}
	case []any:
		for _, nested := range typed {
			if id := findRunID(nested); id != "" {
				return id
			}
		}
	}
	return ""
}

func (s *service) RunValidation(ctx context.Context, planID, phaseID string) (Result, error) {
	if s.operations != nil {
		op, _, err := s.StartValidation(ctx, planID, phaseID, "")
		if err != nil {
			return Result{}, err
		}
		op, err = s.GetValidationOperation(ctx, op.ID, true)
		if err != nil {
			return Result{}, err
		}
		if op.Result == nil || !op.Terminal() {
			return Result{}, fmt.Errorf("validation operation %s remains %s; reattach by operation id", op.ID, op.Status)
		}
		return *op.Result, nil
	}
	p, err := s.plans.GetPlan(ctx, planID)
	if err != nil {
		return Result{}, err
	}
	refs, err := s.scopedReferences(p, phaseID)
	if err != nil {
		return Result{}, err
	}
	scope := deriveScope(p, refs, effectiveBoundary(p, phaseID))
	staleReport, _ := s.ComputeStaleness(ctx, planID, phaseID)
	res := Result{
		ID:          uuid.NewString(),
		PlanID:      p.ID, // canonical id so the execution context server reads the same key
		PhaseID:     phaseID,
		CommandsRun: scope.Commands,
		Staleness:   staleReport.Overall,
		RanAt:       s.now(),
	}
	res.Verdict, res.Detail = s.runCommands(ctx, scope.Commands)
	readinessResult := readiness.Evaluate(ctx, p, readiness.Options{
		PhaseID:           phaseID,
		Mode:              readiness.PreflightMode(),
		CommandValidator:  s.readinessCommandValidator(),
		ReferenceResolver: s.resolver,
	})
	res.CommandFindings = append(res.CommandFindings, commandFindingsFromReadiness(readinessResult.Findings)...)
	res.Verdict = combineValidationVerdicts(res.Verdict, verdictFromReadiness(readinessResult.Verdict))
	res.Detail = joinDetails(res.Detail, readinessResult.Detail)
	// Persist for the cheap-read context path (status/next). Best-effort: a cache
	// write failure must not fail the live validation the agent asked for.
	if s.results != nil {
		_ = s.results.SaveResult(ctx, res)
	}
	return res, nil
}

func (s *service) readinessCommandValidator() readiness.CommandValidator {
	if s.commands == nil {
		return nil
	}
	return commandValidatorAdapter{s.commands}
}

type commandValidatorAdapter struct {
	validator CommandReferenceValidator
}

func (a commandValidatorAdapter) ValidateCommandReference(ctx context.Context, req readiness.CommandRequest) (readiness.CommandResult, error) {
	if a.validator == nil {
		return readiness.CommandResult{}, fmt.Errorf("CLI Health command validator unavailable")
	}
	result, err := a.validator.ValidateCommandReference(ctx, CommandReferenceRequest{
		CommandText: req.CommandText,
		Qualifiers:  append([]string(nil), req.Qualifiers...),
	})
	if err != nil {
		return readiness.CommandResult{}, err
	}
	issues := make([]readiness.CommandIssue, 0, len(result.Issues))
	for _, issue := range result.Issues {
		issues = append(issues, readiness.CommandIssue{Code: issue.Code, Message: issue.Message})
	}
	return readiness.CommandResult{
		Verdict:         result.Verdict,
		ValidationLevel: result.ValidationLevel,
		Issues:          issues,
		Suggestions:     append([]string(nil), result.Suggestions...),
		Guidance:        append([]string(nil), result.Guidance...),
	}, nil
}

func commandFindingsFromReadiness(findings []readiness.Finding) []CommandFinding {
	out := make([]CommandFinding, 0, len(findings))
	for _, finding := range findings {
		out = append(out, CommandFinding{
			CommandText: finding.CommandText,
			Verdict:     string(verdictForReadinessFinding(finding)),
			Level:       finding.Level,
			Message:     finding.Message,
			Location:    finding.Location,
			IssueCodes:  append([]string(nil), finding.IssueCodes...),
			Suggestions: append([]string(nil), finding.Suggestions...),
			Guidance:    append([]string(nil), finding.Guidance...),
		})
	}
	return out
}

func verdictForReadinessFinding(finding readiness.Finding) Verdict {
	switch finding.Severity {
	case readiness.SeverityFail:
		return VerdictFail
	case readiness.SeverityWarning, readiness.SeverityUnknown:
		return VerdictUnknown
	default:
		return VerdictPass
	}
}

func verdictFromReadiness(verdict readiness.Verdict) Verdict {
	switch verdict {
	case readiness.VerdictFail:
		return VerdictFail
	case readiness.VerdictPass:
		return VerdictPass
	default:
		return VerdictUnknown
	}
}

func combineValidationVerdicts(a, b Verdict) Verdict {
	if a == VerdictFail || b == VerdictFail {
		return VerdictFail
	}
	if a == VerdictUnknown || b == VerdictUnknown || a == VerdictUnspecified || b == VerdictUnspecified {
		return VerdictUnknown
	}
	return VerdictPass
}

func joinDetails(parts ...string) string {
	var nonEmpty []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			nonEmpty = append(nonEmpty, part)
		}
	}
	return strings.Join(nonEmpty, "\n")
}

// LastValidation returns the most recent STORED validation result for a
// plan/phase — the cheap read the execution context server uses for status/next
// so those verbs never shell a subprocess. ok=false when nothing has been run yet
// or no store is wired.
func (s *service) LastValidation(ctx context.Context, planID, phaseID string) (Result, bool, error) {
	if s.results == nil {
		return Result{}, false, nil
	}
	if p, err := s.plans.GetPlan(ctx, planID); err == nil {
		planID = p.ID
	}
	return s.results.LastResult(ctx, planID, phaseID)
}

func (s *service) VerifyDefinitionOfDone(ctx context.Context, planID string) (Result, bool, error) {
	p, err := s.plans.GetPlan(ctx, planID)
	if err != nil {
		return Result{}, false, err
	}
	commands := p.RegressionAnchor.Commands
	if len(commands) == 0 && !p.RegressionAnchor.Unavailable {
		// Wizard-authored plans carry the anchor as captured prose with no explicit
		// commands. Derive the oracle command set from the plan's connected code so
		// DoD verifies against a real diff oracle instead of always degrading to
		// UNKNOWN (the authoring→DoD gap).
		commands = deriveScope(p, p.References, p.ChangeBoundary).Commands
	}
	res := Result{PlanID: planID, CommandsRun: commands, RanAt: s.now()}
	if p.RegressionAnchor.Unavailable || len(commands) == 0 {
		res.Verdict = VerdictUnknown
		res.Detail = "regression anchor unavailable; DoD cannot be verified against an oracle"
		return res, false, nil
	}
	res.Verdict, res.Detail = s.runCommands(ctx, commands)
	return res, res.Verdict == VerdictPass, nil
}

// isOracleCommand reports whether a derived command has trustworthy pass/fail
// exit semantics. Only a git-control-tower baseline diff is an oracle (exit 0
// safe, 1 regression, 2 not-comparable). A bare `git diff`/`git diff --stat`
// exits 0 essentially always, so it is INFORMATIONAL — run for its output and
// surfaced to the agent, but it never determines the verdict (treating it as an
// oracle is how "validation passed" used to mean only "git ran").
func isOracleCommand(cmd string) bool {
	return strings.HasPrefix(strings.TrimSpace(cmd), "git-control-tower baseline diff ")
}

// runCommands runs the derived command set and computes a verdict from the
// ORACLE commands only. A tool that is not installed yields UNKNOWN for that
// command (not FAIL — absence of git-control-tower must not look like a
// regression); a baseline diff exit 2 ("not comparable") is UNKNOWN; any other
// non-zero oracle exit is FAIL. PASS requires at least one oracle to have run
// cleanly with no oracle failing or going unknown. With no oracle command at all
// (e.g. only an informational repo-level diff), the verdict is UNKNOWN — honest,
// never a fabricated pass.
func (s *service) runCommands(ctx context.Context, commands []string) (Verdict, string) {
	if len(commands) == 0 {
		return VerdictUnknown, "no baseline commands derived"
	}
	if s.runner == nil {
		return VerdictUnknown, "no command runner configured (git-control-tower unavailable)"
	}
	var (
		details       []string
		oraclePassed  int
		oracleFailed  bool
		oracleUnknown bool
	)
	for _, cmd := range commands {
		result, ok := s.runOneCommand(ctx, cmd)
		if !ok {
			continue
		}
		details = append(details, result.detail)
		if !result.oracle {
			continue
		}
		switch result.verdict {
		case VerdictPass:
			oraclePassed++
		case VerdictFail:
			oracleFailed = true
		default:
			oracleUnknown = true
		}
	}
	detail := strings.Join(details, "\n")
	switch {
	case oracleFailed:
		return VerdictFail, detail
	case oracleUnknown:
		return VerdictUnknown, detail
	case oraclePassed > 0:
		return VerdictPass, detail
	default:
		return VerdictUnknown, detail
	}
}

type commandRunResult struct {
	oracle  bool
	verdict Verdict
	detail  string
}

func (s *service) runOneCommand(ctx context.Context, cmd string) (commandRunResult, bool) {
	name, args := splitCommand(cmd)
	if name == "" {
		return commandRunResult{}, false
	}
	oracle := isOracleCommand(cmd)
	_, err := s.runner(ctx, name, args...)
	return classifyCommandRun(cmd, oracle, err), true
}

func classifyCommandRun(cmd string, oracle bool, err error) commandRunResult {
	result := commandRunResult{oracle: oracle, verdict: VerdictPass, detail: fmt.Sprintf("ok %s", cmd)}
	if err == nil {
		return result
	}
	if errors.Is(err, ErrToolNotFound) {
		result.verdict = VerdictUnknown
		result.detail = fmt.Sprintf("unknown %s: tool not found", cmd)
		return result
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		result.verdict = VerdictUnknown
		result.detail = fmt.Sprintf("unknown %s: execution attachment ended (%v)", cmd, err)
		return result
	}
	var exitErr CommandExitError
	if errors.As(err, &exitErr) && exitErr.Code == 2 {
		result.verdict = VerdictUnknown
		result.detail = fmt.Sprintf("unknown %s: not comparable (exit 2)", cmd)
		return result
	}
	result.verdict = VerdictFail
	result.detail = fmt.Sprintf("FAIL %s: %v", cmd, err)
	return result
}

func (s *service) now() string {
	return s.clock.Now().UTC().Format("2006-01-02T15:04:05.999999999Z07:00")
}

// deriveScope computes the exact baseline/validation command set across all
// affected locations for a plan/phase. The CHANGE BOUNDARY is the source of
// truth: affected scenarios come from acceptance_allow first, supplemented by
// scenario-scoped code references (a reference to a scenario outside the boundary
// is still included, so validation never under-covers). Non-scenario allow globs
// and non-scenario references are repo-level paths with no scenario baseline
// oracle today.
//
// The typed compiler emits one baseline-diff oracle per scenario and at most one
// informational repo diff. Snapshot status is capture metadata, not work.
func deriveScope(p planmodel.Plan, refs []planmodel.Reference, boundary planmodel.ChangeBoundary) BaselineScope {
	checks := compileValidationChecks(p, refs, boundary)
	locations := make([]string, 0, len(checks))
	// Locations report the intended boundary even when no executable oracle can
	// yet be formed (for example a legacy plan without a baseline name).
	for _, scenario := range boundary.AffectedScenarios() {
		locations = appendUnique(locations, "scenarios/"+scenario)
	}
	for _, ref := range refs {
		if ref.Kind == planmodel.ReferenceCode && !ref.Future {
			if scenario := scenarioFromTarget(ref.Target); scenario != "" {
				locations = appendUnique(locations, "scenarios/"+scenario)
			}
		}
	}
	if len(boundary.RepoPaths()) > 0 || p.RegressionAnchor.Strategy == planmodel.AnchorStrategyHeadShaAllowlist {
		locations = appendUnique(locations, "repo")
	}
	for _, check := range checks {
		if check.Scenario != "" {
			locations = appendUnique(locations, "scenarios/"+check.Scenario)
		}
		if check.Kind == ValidationCheckRepoDiff {
			locations = appendUnique(locations, "repo")
		}
	}
	return BaselineScope{Commands: checkCommands(checks), Locations: locations}
}

// effectiveBoundary returns the boundary that scopes validation for a phase: the
// phase's own boundary when it declares one (a narrowing refinement), otherwise
// the plan-level boundary.
func effectiveBoundary(p planmodel.Plan, phaseID string) planmodel.ChangeBoundary {
	if strings.TrimSpace(phaseID) != "" {
		for _, ph := range p.Phases {
			if ph.ID == phaseID {
				switch ph.ValidationScope.Mode {
				case planmodel.ValidationScopeNarrow:
					return ph.ValidationScope.Boundary
				case planmodel.ValidationScopeFullPlan:
					return p.ChangeBoundary
				}
				if !ph.ChangeBoundary.IsZero() {
					return ph.ChangeBoundary
				}
			}
		}
	}
	return p.ChangeBoundary
}

func scenarioFromTarget(target string) string {
	target = strings.TrimPrefix(target, "./")
	const prefix = "scenarios/"
	if !strings.HasPrefix(target, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(target, prefix)
	if i := strings.IndexByte(rest, '/'); i > 0 {
		return rest[:i]
	}
	return rest
}

// splitCommand splits a shell-ish command string into a name + args by
// whitespace. Sufficient for the derived baseline commands (no quoting); the
// LookPath guard in execRunner contains the exec.
func splitCommand(cmd string) (string, []string) {
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return "", nil
	}
	return fields[0], fields[1:]
}

func stalenessRank(t planmodel.StalenessTier) int {
	switch t {
	case planmodel.StalenessFresh:
		return 1
	case planmodel.StalenessLightlyStale:
		return 2
	case planmodel.StalenessDefinitelyStale:
		return 3
	default:
		return 0
	}
}

func appendUnique(list []string, v string) []string {
	for _, e := range list {
		if e == v {
			return list
		}
	}
	return append(list, v)
}
