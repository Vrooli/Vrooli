package validation

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"plan-manager/internal/clock"
	planmodel "plan-manager/internal/planmodel"

	"github.com/google/uuid"
)

// Service is the validation application surface.
type Service interface {
	ResolveReferences(ctx context.Context, planID, phaseID string) (ReferenceReport, error)
	ComputeStaleness(ctx context.Context, planID, phaseID string) (ReferenceReport, error)
	DeriveBaselineScope(ctx context.Context, planID, phaseID string) (BaselineScope, error)
	SyncBaseline(ctx context.Context, planID string) (BaselineCapture, error)
	StartValidation(ctx context.Context, planID, phaseID, idempotencyKey string) (ValidationOperation, bool, error)
	StartValidationTicket(ctx context.Context, req ValidationTicketRequest) (ValidationOperation, bool, error)
	SyncValidation(ctx context.Context, operationID string) (ValidationOperation, error)
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
	plans       PlanSource
	resolver    ReferenceResolver
	staleness   StalenessComputer
	runner      CommandRunner
	collections BaselineCollectionClient
	testRuns    TestRunClient
	inventories BaselineInventorySource
	results     ResultStore
	operations  OperationStore
	clock       clock.Clock
}

// Deps wires the validation Service. plans is required; resolver/staleness/runner/
// results are optional (nil => that capability degrades to a marked gap, never a
// false positive). A nil Results store means RunValidation still returns its live
// result but caches nothing — LastValidation then reports "no result yet".
type Deps struct {
	Plans       PlanSource
	Resolver    ReferenceResolver
	Staleness   StalenessComputer
	Runner      CommandRunner
	Collections BaselineCollectionClient
	TestRuns    TestRunClient
	Inventories BaselineInventorySource
	Results     ResultStore
	Operations  OperationStore
	Clock       clock.Clock
	// Commands remains accepted while older module wiring is migrated. It is not
	// used by producer-owned tickets and cannot dispatch validation work.
	Commands CommandReferenceValidator
}

// NewService constructs the validation Service.
func NewService(d Deps) Service {
	clk := d.Clock
	if clk == nil {
		clk = clock.System{}
	}
	return &service{
		plans:       d.Plans,
		resolver:    d.Resolver,
		staleness:   d.Staleness,
		runner:      d.Runner,
		collections: d.Collections,
		testRuns:    d.TestRuns,
		inventories: d.Inventories,
		results:     d.Results,
		operations:  d.Operations,
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
	boundary := effectiveBoundary(p, phaseID)
	if phaseID == "" && strings.TrimSpace(p.BaselineSet.Name) != "" {
		boundary = fullBaselineSetBoundary(boundary, p.BaselineSet.ScenarioTargets)
	}
	return deriveScope(p, refs, boundary), nil
}

// SyncBaseline reads an already-started collection exactly once. The caller is
// responsible for running the GCT capture and native wait actions first.
func (s *service) SyncBaseline(ctx context.Context, planID string) (BaselineCapture, error) {
	p, err := s.plans.GetPlan(ctx, planID)
	if err != nil {
		return BaselineCapture{}, err
	}
	baselineSet := p.BaselineSet
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

// StartValidationTicket persists producer actions only. The optional execution
// binding and explicit member selection are checked against the immutable
// captured inventory; no producer work starts and no upstream wait occurs.
func (s *service) StartValidationTicket(ctx context.Context, request ValidationTicketRequest) (ValidationOperation, bool, error) {
	planID, phaseID, idempotencyKey := request.PlanID, request.PhaseID, request.IdempotencyKey
	if s.operations == nil {
		return ValidationOperation{}, false, errors.New("durable validation operation store is unavailable")
	}
	p, err := s.plans.GetPlan(ctx, planID)
	if err != nil {
		return ValidationOperation{}, false, err
	}
	// Older rendered plans have no authored collection baseline. When the caller
	// binds validation to an execution that adopted a completed collection, that
	// immutable execution checkpoint is authoritative for this ticket. This
	// preserves the historical plan text while preventing a legacy ticket from
	// falling back to direct per-scenario commands.
	if strings.TrimSpace(request.ExecutionID) != "" && strings.TrimSpace(p.BaselineSet.Name) == "" && s.inventories != nil {
		if inventory, ok, inventoryErr := s.inventories.LatestBaselineInventory(ctx, p.ID); inventoryErr != nil {
			return ValidationOperation{}, false, fmt.Errorf("load execution baseline inventory: %w", inventoryErr)
		} else if ok && inventory.Complete && strings.TrimSpace(inventory.Name) != "" && len(inventory.ScenarioTargets) > 0 {
			p.BaselineSet = planmodel.BaselineSetIntent{
				Name:            inventory.Name,
				ScenarioTargets: uniqueSortedStrings(inventory.ScenarioTargets),
			}
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
	if phaseID == "" && strings.TrimSpace(validationPlan.BaselineSet.Name) != "" {
		boundary = fullBaselineSetBoundary(boundary, validationPlan.BaselineSet.ScenarioTargets)
	}
	if strings.TrimSpace(validationPlan.BaselineSet.Name) != "" && phaseID != "" {
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
	if phaseID == "" {
		// The final DoD is intentionally selector-free and always covers all
		// captured members. A caller cannot narrow it through ticket input.
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
	if len(selectedMembers) > 0 && len(validationPlan.BaselineSet.ScenarioTargets) > 0 && !containsAll(validationPlan.BaselineSet.ScenarioTargets, selectedMembers) {
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
		FullInventory:              phaseID == "",
		TestRuns:                   append([]TestRunEvidence(nil), request.TestRuns...),
		QueueReason:                "awaiting scheduler claim",
		Result: &Result{
			ID: "", PlanID: p.ID, PhaseID: phaseID, Staleness: staleReport.Overall,
			CommandsRun: checkCommands(checks), RequiredMembers: append([]string(nil), requiredMembers...),
			SelectedMembers: append([]string(nil), selectedMembers...),
		},
	}
	op.Result.ID = op.ID + ":result"
	for i, check := range checks {
		childID := fmt.Sprintf("%s:%d", op.ID, i+1)
		command := check.Command
		if check.Kind == ValidationCheckCollectionDiff {
			command = collectionDiffStartCommand(check, childID)
			// Collection diff owns a distinct native wait subcommand. `diff status`
			// is an inspection-only operation and deliberately has no --wait flag.
			op.ProducerWaitArgv = []string{"git-control-tower", "baseline", "collection", "diff", "wait", "--name", check.Baseline, "--operation-id", childID, "--json"}
		}
		op.Children = append(op.Children, ValidationChild{
			ID: childID, Check: check, Command: command,
			Oracle: check.Oracle, Status: ChildQueued, QueuedAt: now,
		})
	}
	op.SyncArgv = []string{"plan-manager", "validate", "sync", op.ID}
	op.QueueReason = "run the producer action, use its native wait/recovery command, then synchronize durable evidence"
	stored, created, err := s.operations.CreateOperation(ctx, op)
	if err != nil {
		return ValidationOperation{}, false, err
	}
	return stored, !created, nil
}

func collectionDiffStartCommand(check ValidationCheck, operationID string) string {
	args := []string{"git-control-tower", "baseline", "collection", "diff", "--name", check.Baseline, "--operation-id", operationID}
	for _, member := range check.Scenarios {
		args = append(args, "--member", member)
	}
	return strings.Join(args, " ")
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
		out = append(out, ValidationCheck{Kind: ValidationCheckPathSnapshotDiff, Baseline: snapshot.Name, Branch: branch, Paths: append([]string(nil), paths...), SemanticKey: "path-snapshot-diff:" + snapshot.Name + ":" + strings.Join(paths, ","), Command: "git-control-tower baseline path diff --before " + snapshot.Name + " --after <captured-after> --path " + strings.Join(paths, " ")})
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
func (s *service) GetValidationOperation(ctx context.Context, operationID string, wait bool) (ValidationOperation, error) {
	op, found, err := s.operations.GetOperation(ctx, strings.TrimSpace(operationID))
	if err != nil {
		return ValidationOperation{}, err
	}
	if !found {
		return ValidationOperation{}, ErrOperationNotFound{ID: operationID}
	}
	_ = wait
	return op, nil
}

// SyncValidation reads durable Git Control Tower state once and commits only
// terminal producer truth. Starting, waiting, recovery and parking remain GCT
// responsibilities exposed by the ticket's argv values.
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
		return op, nil
	}
	op.LastSyncedAt = s.now()
	for i := range op.Children {
		child := &op.Children[i]
		if child.Status == ChildTerminal {
			continue
		}
		if child.Check.Kind == ValidationCheckTestGenieRun {
			if s.testRuns == nil {
				op.QueueReason = "Test Genie evidence synchronization is unavailable"
				continue
			}
			run, readErr := s.testRuns.GetRun(ctx, child.Check.Scenario, child.Check.RunID)
			if readErr != nil {
				op.QueueReason = "test-genie run has not reached readable durable state: " + readErr.Error()
				continue
			}
			if !testRunTerminal(run.Status) {
				op.QueueReason = "test-genie run remains " + run.Status
				continue
			}
			child.Status, child.TerminalAt, child.ExternalID, child.Detail = ChildTerminal, s.now(), run.RunID, run.Detail
			// A Test Genie run attached to a baseline validation is evidence, not a
			// green-suite gate. Its paired collection diff classifies whether a
			// failure is new, pre-existing, or clean. Rejecting a terminal failed
			// run here would make a valid before/after comparison impossible.
			if testRunEvidenceAvailable(run.Status) {
				child.Verdict = VerdictPass
			} else {
				child.Verdict = VerdictFail
			}
			for j := range op.TestRuns {
				if op.TestRuns[j].Scenario == run.Scenario && op.TestRuns[j].RunID == run.RunID {
					op.TestRuns[j] = run
				}
			}
			continue
		}
		if child.Check.Kind != ValidationCheckCollectionDiff {
			child.Status, child.TerminalAt, child.Verdict = ChildTerminal, s.now(), VerdictUnknown
			child.Detail = "no typed producer synchronization is available for this validation check"
			continue
		}
		if s.collections == nil {
			op.QueueReason = "Git Control Tower validation synchronization is unavailable"
			continue
		}
		result, readErr := s.collections.GetCollectionDiff(ctx, child.Check.Baseline, child.Check.Branch, child.ID)
		if readErr != nil {
			op.QueueReason = "producer diff has not reached readable durable state: " + readErr.Error()
			continue
		}
		switch result.Classification {
		case "clean":
			child.Status, child.TerminalAt, child.ExternalID, child.Detail = ChildTerminal, s.now(), result.OperationID, result.Detail
			child.Verdict = VerdictPass
		case "regression":
			child.Status, child.TerminalAt, child.ExternalID, child.Detail = ChildTerminal, s.now(), result.OperationID, result.Detail
			child.Verdict = VerdictFail
		case "not-comparable":
			// A not-comparable diff is a TERMINAL producer outcome — a required
			// member went failed/skipped/stale or collection coverage was
			// incomplete, so the aggregate is not usable as a gate. It is
			// inconclusive (VerdictUnknown), NOT a regression, but it must
			// terminalize the ticket: leaving it non-terminal wedges the ticket
			// forever, and because unkeyed `validate start` coalesces to any active
			// (non-terminal) ticket, every subsequent start returns the wedged one.
			// Terminalizing lets the next `validate start` mint a fresh ticket
			// naturally (knw-1784053356805823492).
			child.Status, child.TerminalAt, child.ExternalID = ChildTerminal, s.now(), result.OperationID
			child.Verdict = VerdictUnknown
			child.Detail = joinDetails("collection diff not comparable: a required member is missing or incomparable (failed/skipped/stale); fix the member and re-run validation", result.Detail)
		default:
			// Still computing (e.g. not-ready) — remain non-terminal and wait for
			// the producer to reach a terminal classification.
			op.QueueReason = "producer diff remains " + result.Classification
			continue
		}
	}
	allTerminal := len(op.Children) > 0
	for _, child := range op.Children {
		if child.Status != ChildTerminal {
			allTerminal = false
			break
		}
	}
	if allTerminal {
		result := Result{ID: op.ID + ":result", PlanID: op.PlanID, PhaseID: op.PhaseID, Verdict: verdictFromChildren(op.Children), RanAt: s.now(), ExecutionID: op.ExecutionID, OperationID: op.ID, ScopeGeneration: op.ScopeGeneration, FullInventory: op.FullInventory, RequiredMembers: append([]string(nil), op.RequiredMembers...), SelectedMembers: append([]string(nil), op.SelectedMembers...)}
		if op.Result != nil {
			result.Staleness, result.CommandsRun = op.Result.Staleness, append([]string(nil), op.Result.CommandsRun...)
		}
		for _, child := range op.Children {
			result.Detail = joinDetails(result.Detail, child.Detail)
		}
		if s.results != nil {
			if err := s.results.SaveResult(ctx, result); err != nil {
				return ValidationOperation{}, err
			}
			op.ResultRef = result.ID
		}
		op.Result, op.Status, op.TerminalAt, op.QueueReason = &result, OperationTerminal, s.now(), ""
	}
	if err := s.operations.SaveOperation(ctx, op); err != nil {
		return ValidationOperation{}, err
	}
	return op, nil
}

func testRunTerminal(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "passed", "failed", "cancelled", "canceled", "stopped", "complete", "completed":
		return true
	}
	return false
}

func testRunPassed(status string) bool {
	return strings.EqualFold(strings.TrimSpace(status), "passed") || strings.EqualFold(strings.TrimSpace(status), "complete") || strings.EqualFold(strings.TrimSpace(status), "completed")
}

func testRunEvidenceAvailable(status string) bool {
	return testRunPassed(status) || strings.EqualFold(strings.TrimSpace(status), "failed")
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
		// Producer-owned tickets are resumed by their printed GCT/Test Genie
		// commands, then reconciled with SyncValidation. Never restart work here.
	}
	return nil
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

// fullBaselineSetBoundary is used only for plan-level/final validation. A phase
// may narrow its own scope, but an empty phase ID is the final certification and
// must exercise every scenario captured in the immutable collection inventory.
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
