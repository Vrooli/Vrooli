package validation

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"

	internalexecution "plan-manager/internal/execution"
	planmodel "plan-manager/internal/planmodel"

	"github.com/vrooli/api-core/schedule"

	"github.com/google/uuid"
)

// Service is the validation application surface.
type Service interface {
	ResolveReferences(ctx context.Context, planID, phaseID string) (ReferenceReport, error)
	ComputeStaleness(ctx context.Context, planID, phaseID string) (ReferenceReport, error)
	DeriveBaselineScope(ctx context.Context, planID, phaseID string) (BaselineScope, error)
	SyncBaseline(ctx context.Context, planID, baselineName string) (BaselineCapture, error)
	StartValidation(ctx context.Context, planID, phaseID, idempotencyKey string) (ValidationOperation, bool, error)
	StartValidationTicket(ctx context.Context, req ValidationTicketRequest) (ValidationOperation, bool, error)
	SyncValidation(ctx context.Context, operationID string) (ValidationOperation, error)
	GetValidationOperation(ctx context.Context, operationID string) (ValidationOperation, error)
	RecoverPending(ctx context.Context) error
	RunValidation(ctx context.Context, planID, phaseID string) (Result, error)
	// LastValidation returns the most recent STORED validation result for a
	// plan/phase (the cheap read path the execution context server uses). ok=false
	// when none has been recorded yet, or when no result store is wired.
	LastValidation(ctx context.Context, planID, phaseID string) (Result, bool, error)
	VerifyDefinitionOfDone(ctx context.Context, planID string) (Result, bool, error)
}

type service struct {
	plans       PlanSource
	resolver    ReferenceResolver
	staleness   StalenessComputer
	collections BaselineCollectionClient
	testRuns    TestRunClient
	inventories BaselineInventorySource
	results     ResultStore
	operations  OperationStore
	receipts    ReceiptClient
	telemetry   internalexecution.TelemetrySink
	clock       schedule.Clock
}

// Deps wires the validation Service. plans is required; resolver/staleness/
// results are optional (nil => that capability degrades to a marked gap, never a
// false positive). A nil Results store means RunValidation still returns its live
// result but caches nothing — LastValidation then reports "no result yet".
type Deps struct {
	Plans       PlanSource
	Resolver    ReferenceResolver
	Staleness   StalenessComputer
	Collections BaselineCollectionClient
	TestRuns    TestRunClient
	Inventories BaselineInventorySource
	Results     ResultStore
	Operations  OperationStore
	Receipts    ReceiptClient
	Telemetry   internalexecution.TelemetrySink
	Clock       schedule.Clock
	// Commands remains accepted while older module wiring is migrated. It is not
	// used by producer-owned tickets and cannot dispatch validation work.
	Commands CommandReferenceValidator
}

// NewService constructs the validation Service.
func NewService(d Deps) Service {
	clk := d.Clock
	if clk == nil {
		clk = schedule.System()
	}
	return &service{
		plans:       d.Plans,
		resolver:    d.Resolver,
		staleness:   d.Staleness,
		collections: d.Collections,
		testRuns:    d.TestRuns,
		inventories: d.Inventories,
		results:     d.Results,
		operations:  d.Operations,
		receipts:    d.Receipts,
		telemetry:   d.Telemetry,
		clock:       clk,
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

func (s *service) DeriveBaselineScope(ctx context.Context, planID, phaseID string) (BaselineScope, error) {
	p, err := s.plans.GetPlan(ctx, planID)
	if err != nil {
		return BaselineScope{}, err
	}
	refs, err := s.scopedReferences(p, phaseID)
	if err != nil {
		return BaselineScope{}, err
	}
	boundary := effectiveBoundary(p, phaseID)
	if phaseID == "" && strings.TrimSpace(p.BaselineSet.Name) != "" {
		boundary = fullBaselineSetBoundary(boundary, p.BaselineSet.ScenarioTargets)
	}
	scope := deriveScope(p, refs, boundary)
	scope.Provenance = scopeProvenance(p, phaseID)
	return scope, nil
}

// SyncBaseline reads an already-started collection exactly once. The caller is
// responsible for running the GCT capture and native wait actions first.
func (s *service) SyncBaseline(ctx context.Context, planID, executionBaselineName string) (BaselineCapture, error) {
	p, err := s.plans.GetPlan(ctx, planID)
	if err != nil {
		return BaselineCapture{}, err
	}
	baselineSet := p.BaselineSet
	// A legacy execution may explicitly adopt a trustworthy recaptured
	// collection whose ticket differs from the authored plan ticket. The
	// execution owns that replacement; keep the plan as the source of scope
	// defaults, but read the collection named by the execution.
	if strings.TrimSpace(executionBaselineName) != "" {
		baselineSet.Name = strings.TrimSpace(executionBaselineName)
	}
	base := BaselineCapture{BaselineName: baselineSet.Name, ScenarioTargets: append([]string(nil), baselineSet.ScenarioTargets...), RepoPaths: append([]string(nil), baselineSet.RepoPaths...)}
	if strings.TrimSpace(baselineSet.Name) == "" {
		base.Detail = "legacy plan has no collection baseline ticket"
		return base, nil
	}
	if s.collections == nil {
		base.Detail = "git-control-tower collection client unavailable"
		return base, nil
	}
	result, err := s.collections.GetCollection(ctx, baselineSet.Name, "")
	if err != nil {
		return BaselineCapture{}, err
	}
	base.CollectionBranch, base.Required, base.Ready, base.Pending, base.Failed, base.Skipped, base.Stale = result.Branch, result.Required, result.Ready, result.Pending, result.Failed, result.Skipped, result.Stale
	base.Members, base.PathSnapshots = append([]BaselineCollectionMember(nil), result.Members...), append([]BaselinePathSnapshot(nil), result.PathSnapshots...)
	// GCT is authoritative for append-only extension membership. Persist the
	// returned inventory rather than the authored plan's original target list.
	base.ScenarioTargets = base.ScenarioTargets[:0]
	for _, member := range result.Members {
		if member.Scenario != "" {
			base.ScenarioTargets = append(base.ScenarioTargets, member.Scenario)
		}
	}
	base.ScenarioTargets = uniqueSortedStrings(base.ScenarioTargets)
	if !result.Complete() {
		base.Detail = fmt.Sprintf("baseline collection %s coverage incomplete: required=%d ready=%d pending=%d failed=%d skipped=%d stale=%d", baselineSet.Name, result.Required, result.Ready, result.Pending, result.Failed, result.Skipped, result.Stale)
		return base, nil
	}
	base.Captured, base.Detail = true, fmt.Sprintf("baseline collection %s synchronized with complete behavioral coverage", baselineSet.Name)
	return base, nil
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
	return s.StartValidationTicket(ctx, ValidationTicketRequest{PlanID: planID, PhaseID: phaseID, IdempotencyKey: idempotencyKey})
}

// StartValidationTicket compiles one canonical Test Genie receipt. The optional
// execution binding and member selection are checked against the immutable
// captured inventory; Plan Manager never creates provider command walls.
func (s *service) StartValidationTicket(ctx context.Context, request ValidationTicketRequest) (ValidationOperation, bool, error) {
	planID, phaseID, idempotencyKey := request.PlanID, request.PhaseID, request.IdempotencyKey
	if s.operations == nil {
		return ValidationOperation{}, false, errors.New("durable validation operation store is unavailable")
	}
	if s.receipts == nil {
		return ValidationOperation{}, false, errors.New("canonical Test Genie receipt service is unavailable")
	}
	p, err := s.plans.GetPlan(ctx, planID)
	if err != nil {
		return ValidationOperation{}, false, err
	}
	// When validation is bound to an execution, its immutable checkpoint is
	// authoritative even if the authored plan still names an older collection.
	// Recapture adoption deliberately preserves the plan text, so retaining the
	// plan ticket here would dispatch validation against the superseded baseline.
	if strings.TrimSpace(request.ExecutionID) != "" && s.inventories != nil {
		if inventory, ok, inventoryErr := s.inventories.LatestBaselineInventory(ctx, p.ID); inventoryErr != nil {
			return ValidationOperation{}, false, fmt.Errorf("load execution baseline inventory: %w", inventoryErr)
		} else if ok && inventory.Complete && strings.TrimSpace(inventory.Name) != "" && len(inventory.ScenarioTargets) > 0 {
			p.BaselineSet.Name = inventory.Name
			p.BaselineSet.ScenarioTargets = uniqueSortedStrings(inventory.ScenarioTargets)
		}
	}
	refs, err := s.scopedReferences(p, phaseID)
	if err != nil {
		return ValidationOperation{}, false, err
	}
	validationPlan, inventory, hasInventory, err := s.planWithCapturedBaselineInventory(ctx, p)
	if err != nil {
		return ValidationOperation{}, false, err
	}
	boundary := effectiveBoundary(validationPlan, phaseID)
	certification := phaseID == "" && validationPlan.CompletionPolicy.RequiresCertification()
	behavioralComparison := certification || phaseRequestsBehavioralComparison(validationPlan, phaseID)
	if certification && strings.TrimSpace(validationPlan.BaselineSet.Name) != "" {
		boundary = fullBaselineSetBoundary(boundary, validationPlan.BaselineSet.ScenarioTargets)
	}
	// An incomplete baseline is diagnostic context for ordinary validation. It
	// must not turn a focused phase into a scope error merely because the phase
	// reaches a scenario that was not present when the optional inventory was
	// captured. Inventory containment is required only when the caller has
	// explicitly requested behavioral comparison (or certification).
	if behavioralComparison && strings.TrimSpace(validationPlan.BaselineSet.Name) != "" && phaseID != "" {
		if outside := scenariosOutsideBaselineInventory(boundary, refs, validationPlan.BaselineSet.ScenarioTargets); len(outside) > 0 {
			return ValidationOperation{}, false, fmt.Errorf("validation scope requests scenario(s) outside captured baseline inventory: %s", strings.Join(outside, ", "))
		}
	}
	checks := compileValidationChecks(validationPlan, refs, boundary)
	requiredMembers := collectionMembers(checks)
	selectedMembers := uniqueSortedStrings(request.SelectedMembers)
	if len(selectedMembers) == 0 {
		selectedMembers = append([]string(nil), requiredMembers...)
	}
	if certification {
		// Explicit certification covers all captured members; ordinary final
		// observations are not promoted into certification by phase position.
		requiredMembers = uniqueSortedStrings(validationPlan.BaselineSet.ScenarioTargets)
		selectedMembers = append([]string(nil), requiredMembers...)
		for i := range checks {
			if checks[i].Kind == ValidationCheckCollectionDiff {
				checks[i].Scenarios = nil
				checks[i].SemanticKey = "collection-diff:" + checks[i].Baseline + ":all"
			}
		}
	} else if len(requiredMembers) > 0 && !containsAll(selectedMembers, requiredMembers) {
		return ValidationOperation{}, false, fmt.Errorf("selected validation members must include required minimum: %s", strings.Join(requiredMembers, ", "))
	} else if len(selectedMembers) > 0 {
		for i := range checks {
			if checks[i].Kind == ValidationCheckCollectionDiff {
				checks[i].Scenarios = append([]string(nil), selectedMembers...)
				checks[i].SemanticKey = "collection-diff:" + checks[i].Baseline + ":" + strings.Join(selectedMembers, ",")
			}
		}
	}
	if behavioralComparison && len(selectedMembers) > 0 && len(validationPlan.BaselineSet.ScenarioTargets) > 0 && !containsAll(validationPlan.BaselineSet.ScenarioTargets, selectedMembers) {
		return ValidationOperation{}, false, fmt.Errorf("selected validation members are outside captured baseline inventory")
	}
	for _, run := range request.TestRuns {
		if strings.TrimSpace(run.Scenario) == "" || strings.TrimSpace(run.RunID) == "" {
			return ValidationOperation{}, false, errors.New("test-genie evidence requires scenario and run id")
		}
		checks = append(checks, ValidationCheck{Kind: ValidationCheckTestGenieRun, Scenario: strings.TrimSpace(run.Scenario), RunID: strings.TrimSpace(run.RunID), Oracle: true, SemanticKey: "test-genie:" + strings.TrimSpace(run.Scenario) + ":" + strings.TrimSpace(run.RunID), Command: "test-genie runs status " + strings.TrimSpace(run.Scenario) + " " + strings.TrimSpace(run.RunID)})
	}
	checks = replaceRepoDiffWithCapturedPathEvidence(checks, sourceEvidencePaths(validationPlan, boundary), inventory, hasInventory)
	staleReport, _ := s.ComputeStaleness(ctx, planID, phaseID)
	now := s.now()
	op := ValidationOperation{
		SchemaVersion:  CurrentOperationSchemaVersion,
		ID:             uuid.NewString(),
		PlanID:         p.ID,
		PhaseID:        phaseID,
		IdempotencyKey: strings.TrimSpace(idempotencyKey),
		Status:         OperationQueued,
		QueuedAt:       now,
		// These legacy transport-budget fields deliberately remain zero. Git
		// Control Tower and Test Genie own their own waiting, recovery, and
		// parking policy; Plan Manager only stores producer tickets and syncs
		// their durable terminal evidence.
		QueueBudgetSeconds:         0,
		ExecutionBudgetSeconds:     0,
		TransportWaitBudgetSeconds: 0,
		RecommendedWaitSeconds:     0,
		ScopeFingerprint:           scopeFingerprint(validationPlan, phaseID, checks),
		ExecutionID:                strings.TrimSpace(request.ExecutionID),
		ScopeGeneration:            request.ScopeGeneration,
		RequiredMembers:            requiredMembers,
		SelectedMembers:            selectedMembers,
		FullInventory:              certification,
		TestRuns:                   append([]TestRunEvidence(nil), request.TestRuns...),
		QueueReason:                "awaiting scheduler claim",
		Result: &Result{
			ID: "", PlanID: p.ID, PhaseID: phaseID, Staleness: staleReport.Overall,
			RequiredMembers: append([]string(nil), requiredMembers...),
			SelectedMembers: append([]string(nil), selectedMembers...),
		},
	}
	op.Result.ID = op.ID + ":result"
	op.SyncArgv = []string{"plan-manager", "validate", "sync", op.ID}
	op.QueueReason = "Test Genie owns the receipt and every provider child operation"
	return s.startReceiptValidation(ctx, request, validationPlan, boundary, refs, op)
}

func phaseRequestsBehavioralComparison(p planmodel.Plan, phaseID string) bool {
	if strings.TrimSpace(phaseID) == "" {
		return false
	}
	for _, phase := range p.Phases {
		if phase.ID == phaseID {
			return phase.ValidationScope.CompareBehavior
		}
	}
	return false
}

func collectionMembers(checks []ValidationCheck) []string {
	var members []string
	for _, check := range checks {
		if check.Kind == ValidationCheckCollectionDiff {
			members = append(members, check.Scenarios...)
		}
	}
	return uniqueSortedStrings(members)
}

func containsAll(selected, required []string) bool {
	set := make(map[string]struct{}, len(selected))
	for _, member := range selected {
		set[member] = struct{}{}
	}
	for _, member := range required {
		if _, ok := set[member]; !ok {
			return false
		}
	}
	return true
}

// planWithCapturedBaselineInventory freezes validation to the target inventory
// that was actually captured at execution start. It deliberately preserves the
// plan's name/policy and phase boundary; only the mutable target inventory is
// substituted. No checkpoint means validation is occurring before execution
// starts, so authored intent remains the correct source.
func (s *service) planWithCapturedBaselineInventory(ctx context.Context, p planmodel.Plan) (planmodel.Plan, BaselineInventory, bool, error) {
	if s.inventories == nil || strings.TrimSpace(p.BaselineSet.Name) == "" {
		return p, BaselineInventory{}, false, nil
	}
	inventory, ok, err := s.inventories.LatestBaselineInventory(ctx, p.ID)
	if err != nil {
		return planmodel.Plan{}, BaselineInventory{}, false, fmt.Errorf("load captured baseline inventory: %w", err)
	}
	if !ok || strings.TrimSpace(inventory.Name) != strings.TrimSpace(p.BaselineSet.Name) || len(inventory.ScenarioTargets) == 0 {
		return p, BaselineInventory{}, false, nil
	}
	p.BaselineSet.ScenarioTargets = uniqueSortedStrings(inventory.ScenarioTargets)
	return p, inventory, true, nil
}

// sourceEvidencePaths mirrors the repo-diff selection compiled for validation.
// A HEAD-SHA allowlist is part of the captured source boundary too, even when a
// phase has no additional repo paths of its own.
func sourceEvidencePaths(p planmodel.Plan, boundary planmodel.ChangeBoundary) []string {
	paths := append([]string(nil), boundary.RepoPaths()...)
	if p.RegressionAnchor.Strategy == planmodel.AnchorStrategyHeadShaAllowlist {
		paths = append(paths, p.RegressionAnchor.AllowlistPaths...)
	}
	return uniqueSortedStrings(paths)
}

// replaceRepoDiffWithCapturedPathEvidence swaps only the legacy informational
// command for a typed GCT path-delta child once the execution checkpoint has a
// concrete source snapshot reference. Before capture, the legacy report remains
// honest rather than guessing a snapshot identity.
func replaceRepoDiffWithCapturedPathEvidence(checks []ValidationCheck, paths []string, inventory BaselineInventory, hasInventory bool) []ValidationCheck {
	if !hasInventory || len(paths) == 0 || len(inventory.PathSnapshots) == 0 {
		return checks
	}
	out := make([]ValidationCheck, 0, len(checks)+len(inventory.PathSnapshots))
	for _, check := range checks {
		if check.Kind != ValidationCheckRepoDiff {
			out = append(out, check)
		}
	}
	for _, snapshot := range inventory.PathSnapshots {
		if strings.TrimSpace(snapshot.Name) == "" {
			continue
		}
		branch := snapshot.Branch
		if branch == "" {
			branch = inventory.Branch
		}
		out = append(out, ValidationCheck{Kind: ValidationCheckPathSnapshotDiff, Baseline: snapshot.Name, Branch: branch, Paths: append([]string(nil), paths...), SemanticKey: "path-snapshot-diff:" + snapshot.Name + ":" + strings.Join(paths, ",")})
	}
	return deduplicateChecks(out)
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

// GetValidationOperation is a cheap inspection only. Legacy wait routes remain
// readable for migration but never create a second lifecycle owner.
func (s *service) GetValidationOperation(ctx context.Context, operationID string) (ValidationOperation, error) {
	op, found, err := s.operations.GetOperation(ctx, strings.TrimSpace(operationID))
	if err != nil {
		return ValidationOperation{}, err
	}
	if !found {
		return ValidationOperation{}, ErrOperationNotFound{ID: operationID}
	}
	if s.receipts != nil {
		if receipt, receiptErr := s.receipts.GetValidation(ctx, op.ID); receiptErr == nil {
			op = projectReceipt(op, receipt, s.now())
		}
	}
	return op, nil
}

// SyncValidation reads the canonical Test Genie receipt once and persists its
// projection. Producer scheduling, waits, retries, and cancellation stay upstream.
func (s *service) SyncValidation(ctx context.Context, operationID string) (ValidationOperation, error) {
	if s.operations == nil {
		return ValidationOperation{}, errors.New("durable validation operation store is unavailable")
	}
	op, found, err := s.operations.GetOperation(ctx, strings.TrimSpace(operationID))
	if err != nil {
		return ValidationOperation{}, err
	}
	if !found {
		return ValidationOperation{}, ErrOperationNotFound{ID: operationID}
	}
	if op.Terminal() {
		err := s.persistReceiptProjection(ctx, &op)
		return op, err
	}
	if s.receipts != nil {
		receipt, receiptErr := s.receipts.GetValidation(ctx, op.ID)
		if receiptErr != nil {
			return ValidationOperation{}, receiptErr
		}
		op = projectReceipt(op, receipt, s.now())
		if err := s.persistReceiptProjection(ctx, &op); err != nil {
			return ValidationOperation{}, err
		}
		return op, nil
	}
	return ValidationOperation{}, errors.New("canonical receipt unavailable; retain historical operation as evidence and use the plan completion policy")
}

// Persist terminal evidence before its operation. Inspection alone must never
// make sync skip this write, including receipts reused in a terminal state.
func (s *service) persistReceiptProjection(ctx context.Context, op *ValidationOperation) error {
	if op.Terminal() && op.Result != nil && s.results != nil {
		if err := s.results.SaveResult(ctx, *op.Result); err != nil {
			return err
		}
		op.ResultRef = op.Result.ID
	}
	return s.operations.SaveOperation(ctx, *op)
}

// RecoverPending refreshes canonical observations without claiming or restarting
// producer work. Historical operations without receipts remain readable as-is.
func (s *service) RecoverPending(ctx context.Context) error {
	if s.operations == nil || s.receipts == nil {
		return nil
	}
	operations, err := s.operations.ListNonTerminalOperations(ctx)
	if err != nil {
		return err
	}
	var failures []error
	for _, op := range operations {
		if _, err := s.SyncValidation(ctx, op.ID); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func (s *service) RunValidation(ctx context.Context, planID, phaseID string) (Result, error) {
	_ = ctx
	_ = phaseID
	return Result{}, ErrProducerTicketRequired{PlanID: planID}
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
	_ = ctx
	return Result{}, false, ErrProducerTicketRequired{PlanID: planID}
}

// isOracleCommand classifies legacy rendered commands during parsing only. New
// tickets use typed producer evidence and never execute this command text.
func isOracleCommand(cmd string) bool {
	return strings.HasPrefix(strings.TrimSpace(cmd), "git-control-tower baseline diff ")
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

// effectiveBoundary returns the boundary that scopes validation for a phase.
// Explicit declarations win. An undeclared phase derives a boundary from its
// affected areas and scenario code references; only a phase with neither falls
// back to the plan boundary.
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
				if derived := derivedPhaseBoundary(ph); !derived.IsZero() {
					return derived
				}
			}
		}
	}
	return p.ChangeBoundary
}

func scopeProvenance(p planmodel.Plan, phaseID string) string {
	if strings.TrimSpace(phaseID) == "" {
		return "plan"
	}
	for _, ph := range p.Phases {
		if ph.ID != phaseID {
			continue
		}
		switch ph.ValidationScope.Mode {
		case planmodel.ValidationScopeNarrow, planmodel.ValidationScopeFullPlan:
			return "explicit"
		}
		if !ph.ChangeBoundary.IsZero() {
			return "explicit"
		}
		if !derivedPhaseBoundary(ph).IsZero() {
			return "derived"
		}
		return "plan"
	}
	return "plan"
}

func derivedPhaseBoundary(ph planmodel.Phase) planmodel.ChangeBoundary {
	allow := make([]string, 0, len(ph.AffectedAreas)+len(ph.References))
	for _, area := range ph.AffectedAreas {
		path := strings.TrimSpace(strings.ReplaceAll(area, "\\", "/"))
		if path == "" {
			continue
		}
		if scenario := scenarioFromTarget(path); scenario != "" {
			allow = append(allow, "scenarios/"+scenario+"/**")
			continue
		}
		allow = append(allow, path)
	}
	for _, ref := range ph.References {
		if ref.Kind != planmodel.ReferenceCode || ref.Future {
			continue
		}
		if scenario := scenarioFromTarget(ref.Target); scenario != "" {
			allow = append(allow, "scenarios/"+scenario+"/**")
		}
	}
	return planmodel.ChangeBoundary{AcceptanceAllow: allow}.Normalized()
}

// fullBaselineSetBoundary is used only for plan-level/final certification.
// A phase may narrow its own scope, but an empty phase ID is the final
// certification and must exercise every scenario captured in the immutable
// collection inventory. Ordinary completion does not enter this path merely
// because a plan has reached its final phase.
func fullBaselineSetBoundary(boundary planmodel.ChangeBoundary, scenarios []string) planmodel.ChangeBoundary {
	for _, scenario := range uniqueSortedStrings(scenarios) {
		boundary.AcceptanceAllow = append(boundary.AcceptanceAllow, "scenarios/"+scenario+"/**")
	}
	return boundary.Normalized()
}

func scenariosOutsideBaselineInventory(boundary planmodel.ChangeBoundary, refs []planmodel.Reference, inventory []string) []string {
	allowed := make(map[string]struct{}, len(inventory))
	for _, scenario := range inventory {
		if scenario = strings.TrimSpace(scenario); scenario != "" {
			allowed[scenario] = struct{}{}
		}
	}
	requested := make([]string, 0)
	requested = append(requested, boundary.AffectedScenarios()...)
	for _, ref := range refs {
		if ref.Kind == planmodel.ReferenceCode && !ref.Future {
			if scenario := scenarioFromTarget(ref.Target); scenario != "" {
				requested = append(requested, scenario)
			}
		}
	}
	outside := make([]string, 0)
	for _, scenario := range uniqueSortedStrings(requested) {
		if _, ok := allowed[scenario]; !ok {
			outside = append(outside, scenario)
		}
	}
	return outside
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
