package planlog

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"

	"plan-manager/internal/clock"
	planmodel "plan-manager/internal/planmodel"

	"github.com/google/uuid"
)

// runIDEnv is the orchestration-layer attribution key. Add* falls back to it
// when the caller does not supply a run id.
const runIDEnv = "VROOLI_AGENT_MANAGER_RUN_ID"

// recentSummaryLimit is how many entries a compact LogSummary surfaces.
const recentSummaryLimit = 5

// Service is the log application surface — the typed execution-log ledger.
type Service interface {
	AddDecision(ctx context.Context, in AddInputs) (Entry, bool, GuidedStep, error)
	AddFinding(ctx context.Context, in AddInputs) (Entry, bool, GuidedStep, error)
	AddBug(ctx context.Context, in AddInputs) (Entry, bool, GuidedStep, error)
	AddRecord(ctx context.Context, in AddInputs) (Entry, bool, GuidedStep, error)
	AddNote(ctx context.Context, in AddInputs) (Entry, bool, GuidedStep, error)

	ListEntries(ctx context.Context, f Filter) ([]Entry, Summary, GuidedStep, error)
	GetEntry(ctx context.Context, id string) (Entry, GuidedStep, error)
	UpdateEntry(ctx context.Context, id string, in UpdateInputs) (Entry, GuidedStep, error)
	ReassignEntry(ctx context.Context, id string, phaseID string) (Entry, GuidedStep, error)
	PromoteEntry(ctx context.Context, id string, toType EntryType, title, detail string, severity Severity) (Entry, Entry, GuidedStep, error)
	SyncEntry(ctx context.Context, id string) (Entry, GuidedStep, error)

	// Summarize is the cheap roll-up the execution domain reads through a seam for
	// its just-in-time context and handoff. Scoped by the filter.
	Summarize(ctx context.Context, f Filter) (Summary, []Entry, error)
}

type service struct {
	repo     Repository
	resolver Resolver
	bugs     BugReporter
	records  RecordWriter
	clock    clock.Clock
}

// Deps wires the log Service. Repo is required. Resolver is optional (nil =>
// the plan_or_execution handle is treated verbatim as a plan id). The downstream
// sinks are optional (nil => the documented pending stub; a failed sync is never
// fatal).
type Deps struct {
	Repo     Repository
	Resolver Resolver
	Bugs     BugReporter
	Records  RecordWriter
	Clock    clock.Clock
}

// NewService constructs the log Service.
func NewService(d Deps) Service {
	clk := d.Clock
	if clk == nil {
		clk = clock.System{}
	}
	bugs := d.Bugs
	if bugs == nil {
		bugs = DefaultBugReporter()
	}
	rec := d.Records
	if rec == nil {
		rec = DefaultRecordWriter()
	}
	return &service{
		repo:     d.Repo,
		resolver: d.Resolver,
		bugs:     bugs,
		records:  rec,
		clock:    clk,
	}
}

var _ Service = (*service)(nil)

func (s *service) AddDecision(ctx context.Context, in AddInputs) (Entry, bool, GuidedStep, error) {
	return s.addEntry(ctx, planmodel.LogEntryDecision, in)
}

func (s *service) AddFinding(ctx context.Context, in AddInputs) (Entry, bool, GuidedStep, error) {
	return s.addEntry(ctx, planmodel.LogEntryFinding, in)
}

func (s *service) AddBug(ctx context.Context, in AddInputs) (Entry, bool, GuidedStep, error) {
	return s.addEntry(ctx, planmodel.LogEntryBugReport, in)
}

func (s *service) AddRecord(ctx context.Context, in AddInputs) (Entry, bool, GuidedStep, error) {
	return s.addEntry(ctx, planmodel.LogEntryRecord, in)
}

func (s *service) AddNote(ctx context.Context, in AddInputs) (Entry, bool, GuidedStep, error) {
	return s.addEntry(ctx, planmodel.LogEntryNote, in)
}

func (s *service) addEntry(ctx context.Context, typ EntryType, in AddInputs) (Entry, bool, GuidedStep, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return Entry{}, false, GuidedStep{}, ErrInvalidEntry{Reason: string(typ) + " title is required"}
	}
	scope, err := s.resolve(ctx, in.PlanOrExecution)
	if err != nil {
		return Entry{}, false, GuidedStep{}, err
	}
	phaseID, err := normalizePhaseID(scope, in.PhaseID)
	if err != nil {
		return Entry{}, false, GuidedStep{}, err
	}
	runID := strings.TrimSpace(in.RunID)
	if runID == "" {
		runID = strings.TrimSpace(os.Getenv(runIDEnv))
	}
	key := strings.TrimSpace(in.IdempotencyKey)

	// Idempotency + attribution dedup: a retry returns the existing entry rather
	// than a duplicate. The unique indexes are the cross-process backstop.
	if existing, ok, derr := s.findDuplicate(ctx, scope.PlanID, scope.ExecutionID, typ, title, key, runID); derr != nil {
		return Entry{}, false, GuidedStep{}, derr
	} else if ok {
		return existing, true, stepForEntry(existing), nil
	}

	now := s.now()
	e := Entry{
		ID:               uuid.NewString(),
		Type:             typ,
		PlanID:           scope.PlanID,
		ExecutionID:      scope.ExecutionID,
		PhaseID:          phaseID,
		Title:            title,
		Detail:           strings.TrimSpace(in.Detail),
		Severity:         in.Severity,
		SyncStatus:       planmodel.DefaultSyncStatusForType(typ),
		SourceCommand:    strings.TrimSpace(in.SourceCommand),
		Evidence:         trimmedNonEmpty(in.Evidence),
		AttributionRunID: runID,
		IdempotencyKey:   key,
		CreatedAt:        now,
		UpdatedAt:        now,
		Bug:              in.Bug,
		Record:           in.Record,
	}
	if typ == planmodel.LogEntryFinding {
		e.Triage = planmodel.TriageCandidate // findings file as CANDIDATE; never auto-promoted
	}
	// Bug reports and records attempt downstream forwarding now; a failure is
	// non-fatal — the entry persists with a degraded sync status.
	e = s.attemptSync(ctx, e)

	if err := s.repo.SaveEntry(ctx, e); err != nil {
		if isUniqueViolation(err) {
			if winner, ok, derr := s.findDuplicate(ctx, scope.PlanID, scope.ExecutionID, typ, title, key, runID); derr == nil && ok {
				return winner, true, stepForEntry(winner), nil
			}
		}
		return Entry{}, false, GuidedStep{}, err
	}
	return e, false, stepForEntry(e), nil
}

func (s *service) ListEntries(ctx context.Context, f Filter) ([]Entry, Summary, GuidedStep, error) {
	f, err := s.resolveFilter(ctx, f)
	if err != nil {
		return nil, Summary{}, GuidedStep{}, err
	}
	entries, err := s.repo.ListEntries(ctx, f)
	if err != nil {
		return nil, Summary{}, GuidedStep{}, err
	}
	summary := planmodel.SummarizeLog(entries, recentSummaryLimit)
	return entries, summary, stepForList(summary), nil
}

func (s *service) GetEntry(ctx context.Context, id string) (Entry, GuidedStep, error) {
	e, err := s.load(ctx, id)
	if err != nil {
		return Entry{}, GuidedStep{}, err
	}
	return e, stepForEntry(e), nil
}

func (s *service) UpdateEntry(ctx context.Context, id string, in UpdateInputs) (Entry, GuidedStep, error) {
	e, err := s.load(ctx, id)
	if err != nil {
		return Entry{}, GuidedStep{}, err
	}
	if t := strings.TrimSpace(in.Title); t != "" {
		e.Title = t
	}
	if d := strings.TrimSpace(in.Detail); d != "" {
		e.Detail = d
	}
	if in.Severity != "" {
		e.Severity = in.Severity
	}
	if in.Triage != "" {
		e.Triage = in.Triage
	}
	e.Evidence = append(e.Evidence, trimmedNonEmpty(in.AddEvidence)...)
	e.UpdatedAt = s.now()
	if err := s.repo.SaveEntry(ctx, e); err != nil {
		return Entry{}, GuidedStep{}, err
	}
	return e, stepForEntry(e), nil
}

func (s *service) ReassignEntry(ctx context.Context, id string, phaseID string) (Entry, GuidedStep, error) {
	e, err := s.load(ctx, id)
	if err != nil {
		return Entry{}, GuidedStep{}, err
	}
	handle := e.PlanID
	if e.ExecutionID != "" {
		handle = e.ExecutionID
	}
	scope, err := s.resolve(ctx, handle)
	if err != nil {
		return Entry{}, GuidedStep{}, err
	}
	normalized, err := normalizePhaseID(scope, phaseID)
	if err != nil {
		return Entry{}, GuidedStep{}, err
	}
	e.PhaseID = normalized
	e.UpdatedAt = s.now()
	if err := s.repo.SaveEntry(ctx, e); err != nil {
		return Entry{}, GuidedStep{}, err
	}
	return e, stepForEntry(e), nil
}

func (s *service) PromoteEntry(ctx context.Context, id string, toType EntryType, title, detail string, severity Severity) (Entry, Entry, GuidedStep, error) {
	src, err := s.load(ctx, id)
	if err != nil {
		return Entry{}, Entry{}, GuidedStep{}, err
	}
	if src.Type != planmodel.LogEntryFinding {
		return Entry{}, Entry{}, GuidedStep{}, ErrNotPromotable{Reason: "only finding entries can be promoted"}
	}
	if toType != planmodel.LogEntryBugReport && toType != planmodel.LogEntryRecord {
		return Entry{}, Entry{}, GuidedStep{}, ErrNotPromotable{Reason: "promotion target must be bug_report or record"}
	}
	// Idempotency: a finding promotes at most once. A repeat call returns the
	// existing promotion rather than filing a second downstream bug/record.
	if src.Triage == planmodel.TriagePromoted {
		if existing, ok, derr := s.findPromotion(ctx, src); derr != nil {
			return Entry{}, Entry{}, GuidedStep{}, derr
		} else if ok {
			return existing, src, stepForEntry(existing), nil
		}
	}
	now := s.now()
	promoted := Entry{
		ID:               uuid.NewString(),
		Type:             toType,
		PlanID:           src.PlanID,
		ExecutionID:      src.ExecutionID,
		PhaseID:          src.PhaseID,
		Title:            firstNonEmpty(strings.TrimSpace(title), src.Title),
		Detail:           firstNonEmpty(strings.TrimSpace(detail), src.Detail),
		Severity:         pickSeverity(severity, src.Severity),
		SyncStatus:       planmodel.DefaultSyncStatusForType(toType),
		Evidence:         append([]string(nil), src.Evidence...),
		AttributionRunID: src.AttributionRunID,
		SourceCommand:    "plan-manager log promote",
		PromotedFromID:   src.ID,
		CreatedAt:        now,
		UpdatedAt:        now,
		Bug:              src.Bug,
		Record:           src.Record,
	}
	promoted = s.attemptSync(ctx, promoted)
	// Preserve the original finding, marked promoted with the link recorded.
	src.Triage = planmodel.TriagePromoted
	src.UpdatedAt = now
	if err := s.repo.SaveEntry(ctx, promoted); err != nil {
		return Entry{}, Entry{}, GuidedStep{}, err
	}
	if err := s.repo.SaveEntry(ctx, src); err != nil {
		return Entry{}, Entry{}, GuidedStep{}, err
	}
	return promoted, src, stepForEntry(promoted), nil
}

func (s *service) SyncEntry(ctx context.Context, id string) (Entry, GuidedStep, error) {
	e, err := s.load(ctx, id)
	if err != nil {
		return Entry{}, GuidedStep{}, err
	}
	if e.Type != planmodel.LogEntryBugReport && e.Type != planmodel.LogEntryRecord {
		return Entry{}, GuidedStep{}, ErrInvalidEntry{Reason: "only bug_report and record entries have downstream sync"}
	}
	if e.SyncStatus == planmodel.LogSyncSynced {
		return e, stepForEntry(e), nil // already synced; idempotent retry
	}
	e = s.attemptSync(ctx, e)
	e.UpdatedAt = s.now()
	if err := s.repo.SaveEntry(ctx, e); err != nil {
		return Entry{}, GuidedStep{}, err
	}
	return e, stepForEntry(e), nil
}

func (s *service) Summarize(ctx context.Context, f Filter) (Summary, []Entry, error) {
	f, err := s.resolveFilter(ctx, f)
	if err != nil {
		return Summary{}, nil, err
	}
	entries, err := s.repo.ListEntries(ctx, f)
	if err != nil {
		return Summary{}, nil, err
	}
	return planmodel.SummarizeLog(entries, recentSummaryLimit), entries, nil
}

// --- helpers -----------------------------------------------------------------

// attemptSync forwards a bug/record entry to its downstream sink. It is NEVER
// fatal: success → SYNCED + reference; unavailable → PENDING + detail; any other
// downstream error → SYNC_FAILED + detail. Non-syncable types are returned
// unchanged.
func (s *service) attemptSync(ctx context.Context, e Entry) Entry {
	var (
		ref DownstreamRef
		err error
	)
	switch e.Type {
	case planmodel.LogEntryBugReport:
		ref, err = s.bugs.FileBug(ctx, e)
	case planmodel.LogEntryRecord:
		ref, err = s.records.WriteRecord(ctx, e)
	default:
		return e
	}
	e.Downstream = ref
	e.Capture = ref.Capture
	if err == nil {
		e.SyncStatus = planmodel.LogSyncSynced
		e.Downstream.SyncedAt = s.now()
		return e
	}
	var unavailable ErrDownstreamUnavailable
	if errors.As(err, &unavailable) {
		e.SyncStatus = planmodel.LogSyncPending
		if e.Downstream.Detail == "" {
			e.Downstream.Detail = unavailable.Reason
		}
		return e
	}
	e.SyncStatus = planmodel.LogSyncFailed
	if e.Downstream.Detail == "" {
		e.Downstream.Detail = err.Error()
	}
	return e
}

func (s *service) findDuplicate(ctx context.Context, planID, executionID string, typ EntryType, title, key, runID string) (Entry, bool, error) {
	if key != "" {
		if e, ok, err := s.repo.FindByIdempotencyKey(ctx, planID, key); err != nil || ok {
			return e, ok, err
		}
	}
	if runID == "" {
		return Entry{}, false, nil
	}
	// Attribution dedup: same (plan, execution, run id, type, normalized title).
	// plan_id keeps a plan-handle entry (empty execution id) scoped to its plan.
	entries, err := s.repo.ListEntries(ctx, Filter{PlanID: planID, ExecutionID: executionID, Type: typ})
	if err != nil {
		return Entry{}, false, err
	}
	for _, e := range entries {
		if e.AttributionRunID == runID && strings.EqualFold(strings.TrimSpace(e.Title), strings.TrimSpace(title)) {
			return e, true, nil
		}
	}
	return Entry{}, false, nil
}

// findPromotion returns the existing bug_report/record promoted from src, if any.
// It scopes the scan to src's plan/execution so it stays cheap.
func (s *service) findPromotion(ctx context.Context, src Entry) (Entry, bool, error) {
	entries, err := s.repo.ListEntries(ctx, Filter{PlanID: src.PlanID, ExecutionID: src.ExecutionID})
	if err != nil {
		return Entry{}, false, err
	}
	for _, e := range entries {
		if e.PromotedFromID == src.ID {
			return e, true, nil
		}
	}
	return Entry{}, false, nil
}

func (s *service) resolve(ctx context.Context, handle string) (Scope, error) {
	handle = strings.TrimSpace(handle)
	if handle == "" {
		return Scope{}, ErrInvalidEntry{Reason: "a plan id/slug or execution id is required"}
	}
	if s.resolver == nil {
		return Scope{PlanID: handle}, nil
	}
	scope, ok, err := s.resolver.Resolve(ctx, handle)
	if err != nil {
		return Scope{}, err
	}
	if !ok {
		return Scope{}, ErrInvalidEntry{Reason: "plan or execution not found: " + handle}
	}
	if strings.TrimSpace(scope.PlanID) == "" {
		return Scope{}, ErrInvalidEntry{Reason: "plan or execution not found: " + handle}
	}
	scope.PlanID = strings.TrimSpace(scope.PlanID)
	scope.ExecutionID = strings.TrimSpace(scope.ExecutionID)
	scope.CurrentPhaseID = strings.TrimSpace(scope.CurrentPhaseID)
	return scope, nil
}

// resolveFilter resolves the filter's PlanID handle (which may be a slug or an
// execution id) into canonical plan/execution scoping.
func (s *service) resolveFilter(ctx context.Context, f Filter) (Filter, error) {
	handle := strings.TrimSpace(f.PlanID)
	if handle == "" || s.resolver == nil {
		return f, nil
	}
	scope, ok, err := s.resolver.Resolve(ctx, handle)
	if err != nil {
		return Filter{}, err
	}
	if !ok {
		return f, nil // leave the handle as a literal plan id filter
	}
	f.PlanID = scope.PlanID
	if scope.ExecutionID != "" && f.ExecutionID == "" {
		f.ExecutionID = scope.ExecutionID
		f.PlanID = "" // an execution scope is more specific than the plan
	}
	return f, nil
}

func normalizePhaseID(scope Scope, raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if scope.ExecutionID != "" {
			return strings.TrimSpace(scope.CurrentPhaseID), nil
		}
		return "", nil
	}
	for _, ph := range scope.Phases {
		if raw == strings.TrimSpace(ph.ID) {
			return strings.TrimSpace(ph.ID), nil
		}
	}
	if n, err := strconv.Atoi(raw); err == nil {
		if n <= 0 {
			return "", ErrInvalidEntry{Reason: "phase ordinal must be 1 or greater"}
		}
		for i, ph := range scope.Phases {
			order := ph.Order
			if order <= 0 {
				order = i + 1
			}
			if order == n {
				return strings.TrimSpace(ph.ID), nil
			}
		}
		return "", ErrInvalidEntry{Reason: "phase ordinal " + raw + " is not present on plan " + scope.PlanID}
	}
	return "", ErrInvalidEntry{Reason: "phase not found on plan " + scope.PlanID + ": " + raw}
}

func (s *service) load(ctx context.Context, id string) (Entry, error) {
	e, ok, err := s.repo.GetEntry(ctx, strings.TrimSpace(id))
	if err != nil {
		return Entry{}, err
	}
	if !ok {
		return Entry{}, ErrEntryNotFound{ID: id}
	}
	return e, nil
}

func (s *service) now() string { return s.clock.Now().UTC().Format(logTimeFormat) }

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint") || strings.Contains(msg, "constraint failed")
}

func trimmedNonEmpty(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if t := strings.TrimSpace(s); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func pickSeverity(override, fallback Severity) Severity {
	if override != "" {
		return override
	}
	return fallback
}
