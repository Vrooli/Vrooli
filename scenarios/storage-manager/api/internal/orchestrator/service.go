package orchestrator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/eventbus"
	journalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/journal"
	journalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/journal/journal_v1connect"
	"storage-manager/internal/cleanup"
	"storage-manager/internal/policy"
	"storage-manager/internal/providers"
)

type Policy struct {
	Version           string
	Profile           policy.ProfileName
	Providers         map[string]cleanup.ProviderPolicy
	StandingApprovals map[string]StandingApproval
	CreatedAt         time.Time
}

// StandingApproval is host-local authority for one conditional provider. The
// subject constraints remain provider-owned; this record only says that the
// operator explicitly enabled that provider on this host.
type StandingApproval struct {
	ApprovedAt         time.Time         `json:"approved_at"`
	ApprovedBy         string            `json:"approved_by"`
	HostID             string            `json:"host_id"`
	SubjectConstraints map[string]string `json:"subject_constraints,omitempty"`
}

type ProviderPlan struct {
	ProviderID      string
	ProviderVersion string
	Estimate        cleanup.Estimate
	Preview         cleanup.Preview
	Policy          cleanup.ProviderPolicy
}

type Plan struct {
	ID                string
	PolicyVersion     string
	CreatedAt         time.Time
	Providers         []ProviderPlan
	TotalBytes        int64
	TotalItems        int
	CensusID          string
	CensusStatus      string
	CensusStartedAt   time.Time
	CensusCompletedAt time.Time
}

const (
	CensusStatusRunning  = "running"
	CensusStatusComplete = "complete"
	CensusStatusPartial  = "partial"
	CensusStatusFailed   = "failed"
)

type ApplyInput struct {
	PlanID         string
	PolicyVersion  string
	ApprovalMode   cleanup.ApprovalMode
	ApprovalToken  string
	IdempotencyKey string
}

type ApplyReport struct {
	PlanID         string
	IdempotencyKey string
	AlreadyApplied bool
	Results        []cleanup.ApplyResult
	ReclaimedBytes int64
}

// WarningGrowthTarget is the actionable growth row consumed by the warning
// pressure path. The reader deliberately returns one row: pressure handling
// needs the fastest unbounded owner, not a second copy of the full report.
type WarningGrowthTarget struct {
	OwnerKind         string
	OwnerID           string
	EntryName         string
	CurrentBytes      int64
	SlopeBytesPerHour float64
}

// WarningBugReport is the typed hand-off to the report-bug writer. The
// orchestrator does not know how scenario-qa stores entries.
type WarningBugReport struct {
	Title          string
	SignalType     string
	Severity       string
	Repro          []string
	Expected       string
	Actual         string
	Description    string
	Context        map[string]string
	HonestyFlags   []string
	IdempotencyKey string
}

// WarningDependencies keep pressure policy independent from growth storage and
// the external bug-inbox transport. Both callbacks are optional and fail
// closed: cleanup still runs if either observation or reporting is unavailable.
type WarningDependencies struct {
	FastestUnbounded func(context.Context) (WarningGrowthTarget, bool, error)
	FileBug          func(context.Context, WarningBugReport) (string, error)
}

type AuditEvent struct {
	ID             string
	Time           time.Time
	Type           string
	PlanID         string
	ProviderID     string
	IdempotencyKey string
	Message        string
	ReclaimedBytes int64
	Redacted       bool
}

type Store interface {
	SavePolicy(context.Context, Policy) error
	CurrentPolicy(context.Context) (Policy, bool, error)
	SavePlan(context.Context, Plan) error
	GetPlan(context.Context, string) (Plan, bool, error)
	SaveApply(context.Context, ApplyReport) error
	ApplyByKey(context.Context, string) (ApplyReport, bool, error)
	AddAudit(context.Context, AuditEvent) error
	ListAudit(context.Context) ([]AuditEvent, error)
}

type Service struct {
	registry *providers.Registry
	store    Store
	clock    cleanup.Clock
	hostID   func() string
	// recoveryContext is owned by the API process, not by an individual RPC.
	// Server-owned recovery work must survive a caller disconnect but stop when
	// the service itself shuts down.
	recoveryContext context.Context

	// Disk-pressure intake state. pressure collapses duplicate reports of the
	// same event; autonomousApply is the kill switch for unattended deletion.
	pressure        *pressureGuard
	warningMu       sync.Mutex
	warningDeps     WarningDependencies
	warningBugLast  map[string]time.Time
	warningBugBusy  map[string]bool
	autonomousMu    sync.RWMutex
	autonomousApply bool

	// Census jobs are server-owned. A request may stop waiting without
	// cancelling the measurement, so a slow filesystem walk cannot be turned
	// into a misleading partial-as-total response or abandoned work.
	censusMu       sync.Mutex
	censusJobs     map[string]*censusJob
	censusSeq      uint64
	latestCensusID string

	recoveryMu       sync.Mutex
	recoveryRuns     map[string]*RecoveryRun
	recoverySeq      uint64
	recoveryInstance string
	recoveryGate     sync.Mutex
	recoveryBusy     bool
	recoveryLockPath string
	events           eventPublisher
	journal          journalAppender
}

type censusJob struct {
	done     chan struct{}
	result   Plan
	err      error
	consumed bool
}

// defaultPressureDedupWindow is how long a completed autonomous execution
// suppresses another for the same partition and band. It is long enough to
// cover both safeguards reporting the same event and short enough that
// genuinely renewed pressure is acted on.
const defaultPressureDedupWindow = 5 * time.Minute

func NewService(registry *providers.Registry, store Store, clock cleanup.Clock) *Service {
	return NewServiceWithContext(context.Background(), registry, store, clock)
}

// NewServiceWithContext constructs a service whose server-owned workers share
// the supplied process lifetime. The context must be cancelled during API
// shutdown.
func NewServiceWithContext(serviceContext context.Context, registry *providers.Registry, store Store, clock cleanup.Clock) *Service {
	if serviceContext == nil {
		serviceContext = context.Background()
	}
	return &Service{
		registry:        registry,
		store:           store,
		clock:           clock,
		recoveryContext: serviceContext,
		pressure:        newPressureGuard(defaultPressureDedupWindow),
		warningBugLast:  make(map[string]time.Time),
		warningBugBusy:  make(map[string]bool),
		// Autonomous apply is on by default: the incident happened because
		// nothing acted overnight. The kill switch exists to turn remediation
		// off deliberately, not to require a deliberate act to turn it on.
		autonomousApply:  true,
		censusJobs:       make(map[string]*censusJob),
		recoveryRuns:     make(map[string]*RecoveryRun),
		recoveryInstance: fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano()),
		hostID: func() string {
			host, _ := os.Hostname()
			return strings.TrimSpace(host)
		},
		// Lifecycle injects VROOLI_EVENTS_API_BASE when the event service is
		// available. Keep construction side-effect free: discovery during API
		// startup can delay or prevent the health listener from binding.
		events: eventbus.Client{BaseURL: os.Getenv("VROOLI_EVENTS_API_BASE")},
	}
}

type eventPublisher interface {
	PublishDomainEvent(context.Context, eventbus.DomainEvent) error
}

type journalAppender interface {
	AppendRecovery(context.Context, RecoveryRun) error
}

type journalClient struct {
	client journalconnect.JournalServiceClient
}

func NewJournalAppender(baseURL string) journalAppender {
	if strings.TrimSpace(baseURL) == "" {
		return nil
	}
	return journalClient{client: journalconnect.NewJournalServiceClient(http.DefaultClient, strings.TrimRight(baseURL, "/"))}
}

func (c journalClient) AppendRecovery(ctx context.Context, run RecoveryRun) error {
	_, err := c.client.AppendEntry(ctx, connect.NewRequest(&journalv1.AppendEntryRequest{
		Kind: "work-record", Trigger: run.Trigger,
		Approach: fmt.Sprintf("recovery ladder run %s; action=%s", run.ID, run.Action),
		Evidence: fmt.Sprintf("run_id=%s reclaimed_bytes=%d target_free_bytes=%d", run.ID, run.ReclaimedBytes, run.TargetFreeBytes),
		Outcome:  run.StoppedBecause, Scope: "storage-manager",
		Body: fmt.Sprintf("Storage recovery run %s reclaimed %d bytes and stopped because %s.", run.ID, run.ReclaimedBytes, run.StoppedBecause),
	}))
	return err
}

func (s *Service) SetEventPublisher(p eventPublisher) { s.events = p }

func (s *Service) SetJournalAppender(a journalAppender) { s.journal = a }

// SetWarningDependencies wires the read-only growth projection and the
// report-bug transport used by warning pressure. It is a production seam so
// tests can prove the action without contacting Prompt Manager.
func (s *Service) SetWarningDependencies(deps WarningDependencies) {
	s.warningMu.Lock()
	s.warningDeps = deps
	s.warningMu.Unlock()
}

func (s *Service) Catalog() []cleanup.ProviderMetadata {
	return s.registry.List()
}

func (s *Service) SetPolicyProfile(ctx context.Context, name policy.ProfileName) (Policy, error) {
	profile, err := policy.BuildProfile(name, s.registry.List())
	if err != nil {
		return Policy{}, err
	}
	out := Policy{
		Version:   stablePolicyVersion(name, profile.Defaults),
		Profile:   name,
		Providers: profile.Defaults,
		CreatedAt: s.now(),
	}
	for _, meta := range s.registry.List() {
		if err := policy.ValidateProviderPolicy(meta, out.Providers[meta.ID]); err != nil {
			return Policy{}, err
		}
	}
	if err := s.store.SavePolicy(ctx, out); err != nil {
		return Policy{}, err
	}
	_ = s.audit(ctx, AuditEvent{Type: "policy.saved", Message: string(name)})
	return out, nil
}

// SetStandingApproval records host-local authority for one conditional
// provider. The approval is deliberately separate from the profile: changing
// a cleanup profile must never silently grant a root-owned action.
func (s *Service) SetStandingApproval(ctx context.Context, providerID string, approval StandingApproval) (Policy, error) {
	providerID = strings.TrimSpace(providerID)
	provider, ok := s.registry.Get(providerID)
	if !ok || provider.Metadata().SafetyTier != cleanup.SafetyTierConditional {
		return Policy{}, fmt.Errorf("standing approval requires a registered conditional provider: %s", providerID)
	}
	if strings.TrimSpace(approval.HostID) == "" || strings.TrimSpace(approval.ApprovedBy) == "" || approval.ApprovedAt.IsZero() {
		return Policy{}, fmt.Errorf("standing approval requires approved_by, approved_at, and host_id")
	}
	if s.hostID == nil || strings.TrimSpace(s.hostID()) == "" {
		return Policy{}, fmt.Errorf("standing approval requires the current host identity")
	}
	if strings.TrimSpace(approval.HostID) != strings.TrimSpace(s.hostID()) {
		return Policy{}, fmt.Errorf("standing approval host_id %q does not match current host", approval.HostID)
	}
	current, err := s.CurrentPolicy(ctx)
	if err != nil {
		return Policy{}, err
	}
	if current.StandingApprovals == nil {
		current.StandingApprovals = map[string]StandingApproval{}
	}
	current.StandingApprovals[providerID] = approval
	if err := s.store.SavePolicy(ctx, current); err != nil {
		return Policy{}, err
	}
	_ = s.audit(ctx, AuditEvent{Type: "standing_approval.saved", ProviderID: providerID, Message: "host-local approval recorded"})
	return current, nil
}

// RevokeStandingApproval removes authority for one conditional provider. A
// missing entry is a successful idempotent revoke.
func (s *Service) RevokeStandingApproval(ctx context.Context, providerID string) (Policy, error) {
	current, err := s.CurrentPolicy(ctx)
	if err != nil {
		return Policy{}, err
	}
	delete(current.StandingApprovals, strings.TrimSpace(providerID))
	if err := s.store.SavePolicy(ctx, current); err != nil {
		return Policy{}, err
	}
	_ = s.audit(ctx, AuditEvent{Type: "standing_approval.revoked", ProviderID: strings.TrimSpace(providerID), Message: "host-local approval revoked"})
	return current, nil
}

func (s *Service) CurrentPolicy(ctx context.Context) (Policy, error) {
	if existing, ok, err := s.store.CurrentPolicy(ctx); err != nil {
		return Policy{}, err
	} else if ok {
		// Older in-memory callers and pre-profile databases may carry a
		// deliberately minimal policy row. Preserve that row until an operator
		// chooses a profile; a zero profile cannot be reconciled safely.
		if existing.Profile == "" {
			return existing, nil
		}
		version, providers, added, reconcileErr := policy.ReconcilePolicy(existing.Profile, existing.Version, existing.Providers, existing.CreatedAt, s.registry.List())
		if reconcileErr != nil {
			return Policy{}, reconcileErr
		}
		if len(added) == 0 {
			return existing, nil
		}
		existing.Version = version
		existing.Providers = providers
		if err := s.store.SavePolicy(ctx, existing); err != nil {
			return Policy{}, err
		}
		_ = s.audit(ctx, AuditEvent{Type: "policy.reconciled", Message: fmt.Sprintf("added %d provider(s) at %s defaults: %s", len(added), existing.Profile, strings.Join(added, ", "))})
		return existing, nil
	}
	return s.SetPolicyProfile(ctx, policy.ProfileBalanced)
}

func (s *Service) Plan(ctx context.Context, scope cleanup.ObservationScope) (Plan, error) {
	// If a previous caller stopped waiting after the HTTP deadline, reuse its
	// still-owned job instead of starting a second full-host walk. This makes a
	// follow-up plan a retrieval of the completed census rather than a poll.
	if job := s.latestUnconsumedCensus(); job != nil {
		plan, err := waitCensus(ctx, job)
		if err == nil || ctx.Err() == nil {
			s.markCensusConsumedJob(job)
		}
		return plan, err
	}
	id, err := s.StartCensus(ctx, scope)
	if err != nil {
		return Plan{}, err
	}
	plan, err := s.WaitCensus(ctx, id)
	if err == nil || ctx.Err() == nil {
		s.markCensusConsumed(id)
	}
	return plan, err
}

// StartCensus dispatches a tracked measurement that is independent of the
// caller's request lifetime. It returns before providers are walked.
func (s *Service) StartCensus(_ context.Context, scope cleanup.ObservationScope) (string, error) {
	id := fmt.Sprintf("census-%d", atomic.AddUint64(&s.censusSeq, 1))
	job := &censusJob{done: make(chan struct{})}
	scope.CompleteCensus = true
	s.censusMu.Lock()
	s.censusJobs[id] = job
	s.latestCensusID = id
	s.censusMu.Unlock()
	go s.runCensus(id, scope, job)
	return id, nil
}

// WaitCensus waits for one server-owned census. Cancelling this wait does not
// cancel the underlying job; another caller can retrieve the same result by
// census ID once it completes.
func (s *Service) WaitCensus(ctx context.Context, id string) (Plan, error) {
	s.censusMu.Lock()
	job, ok := s.censusJobs[id]
	s.censusMu.Unlock()
	if !ok {
		return Plan{}, fmt.Errorf("census %q not found", id)
	}
	return waitCensus(ctx, job)
}

func waitCensus(ctx context.Context, job *censusJob) (Plan, error) {
	select {
	case <-job.done:
		return job.result, job.err
	case <-ctx.Done():
		return Plan{}, ctx.Err()
	}
}

func (s *Service) latestUnconsumedCensus() *censusJob {
	s.censusMu.Lock()
	defer s.censusMu.Unlock()
	job := s.censusJobs[s.latestCensusID]
	if job == nil || job.consumed {
		return nil
	}
	return job
}

func (s *Service) markCensusConsumed(id string) {
	s.censusMu.Lock()
	defer s.censusMu.Unlock()
	if job := s.censusJobs[id]; job != nil {
		job.consumed = true
	}
}

func (s *Service) markCensusConsumedJob(job *censusJob) {
	s.censusMu.Lock()
	defer s.censusMu.Unlock()
	job.consumed = true
}

func (s *Service) runCensus(id string, scope cleanup.ObservationScope, job *censusJob) {
	plan, err := s.planSync(context.Background(), id, scope)
	s.censusMu.Lock()
	job.result = plan
	job.err = err
	s.censusMu.Unlock()
	close(job.done)
}

func (s *Service) planSync(ctx context.Context, censusID string, scope cleanup.ObservationScope) (Plan, error) {
	current, err := s.CurrentPolicy(ctx)
	if err != nil {
		return Plan{}, err
	}
	started := s.now()
	providerPlans := make([]ProviderPlan, 0)
	for _, meta := range s.registry.List() {
		provider, ok := s.registry.Get(meta.ID)
		if !ok {
			return Plan{}, fmt.Errorf("provider %q missing from registry", meta.ID)
		}
		providerPolicy := current.Providers[meta.ID]
		estimate, err := provider.Estimate(ctx, cleanup.EstimateRequest{Scope: scope, Policy: providerPolicy})
		if err != nil {
			return Plan{}, fmt.Errorf("estimate %s: %w", meta.ID, err)
		}
		preview, err := provider.Preview(ctx, cleanup.PreviewRequest{Scope: scope, Policy: providerPolicy, Estimate: estimate})
		if err != nil {
			return Plan{}, fmt.Errorf("preview %s: %w", meta.ID, err)
		}
		providerPlans = append(providerPlans, ProviderPlan{
			ProviderID:      meta.ID,
			ProviderVersion: meta.Version,
			Estimate:        estimate,
			Preview:         preview,
			Policy:          providerPolicy,
		})
	}
	sort.Slice(providerPlans, func(i, j int) bool { return providerPlans[i].ProviderID < providerPlans[j].ProviderID })
	out := Plan{
		PolicyVersion:     current.Version,
		CreatedAt:         s.now(),
		Providers:         providerPlans,
		CensusID:          censusID,
		CensusStatus:      CensusStatusComplete,
		CensusStartedAt:   started,
		CensusCompletedAt: s.now(),
	}
	for _, pp := range out.Providers {
		out.TotalBytes += pp.Estimate.EstimatedBytes
		out.TotalItems += pp.Estimate.ItemCount
		if len(pp.Preview.Warnings) > 0 {
			out.CensusStatus = CensusStatusPartial
		}
	}
	out.ID = stablePlanID(out)
	if err := s.store.SavePlan(ctx, out); err != nil {
		return Plan{}, err
	}
	_ = s.audit(ctx, AuditEvent{Type: "plan.created", PlanID: out.ID, Message: fmt.Sprintf("%d providers", len(out.Providers))})
	return out, nil
}

func (s *Service) Apply(ctx context.Context, input ApplyInput) (ApplyReport, error) {
	// Apply is server-owned once approval and replay validation begin. A CLI
	// request may hit its transport deadline while a large trash root is being
	// removed; cancelling the filesystem context at that point would leave an
	// unreported half-apply. The idempotency record below is the durable
	// retrieval point for the completed result.
	ctx = context.WithoutCancel(ctx)
	if strings.TrimSpace(input.IdempotencyKey) == "" {
		return ApplyReport{}, fmt.Errorf("idempotency key is required")
	}
	if prior, ok, err := s.store.ApplyByKey(ctx, input.IdempotencyKey); err != nil {
		return ApplyReport{}, err
	} else if ok {
		prior.AlreadyApplied = true
		_ = s.audit(ctx, AuditEvent{Type: "apply.replayed", PlanID: prior.PlanID, IdempotencyKey: input.IdempotencyKey, Message: "idempotent replay"})
		return prior, nil
	}
	plan, ok, err := s.store.GetPlan(ctx, input.PlanID)
	if err != nil {
		return ApplyReport{}, err
	}
	if !ok {
		return ApplyReport{}, fmt.Errorf("plan %q not found", input.PlanID)
	}
	if input.PolicyVersion != plan.PolicyVersion {
		return ApplyReport{}, fmt.Errorf("policy version mismatch: got %q want %q", input.PolicyVersion, plan.PolicyVersion)
	}
	if input.ApprovalMode != cleanup.ApprovalModeNone && strings.TrimSpace(input.ApprovalToken) == "" {
		return ApplyReport{}, fmt.Errorf("approval token is required for %s approval", input.ApprovalMode)
	}

	report := ApplyReport{PlanID: plan.ID, IdempotencyKey: input.IdempotencyKey}
	for _, pp := range plan.Providers {
		result, applied, err := s.applyProvider(ctx, plan, pp, input)
		if err != nil {
			return ApplyReport{}, err
		}
		if !applied {
			continue
		}
		report.Results = append(report.Results, result)
		report.ReclaimedBytes += result.ReclaimedBytes
	}
	if err := s.store.SaveApply(ctx, report); err != nil {
		return ApplyReport{}, err
	}
	_ = s.audit(ctx, AuditEvent{Type: "apply.completed", PlanID: plan.ID, IdempotencyKey: input.IdempotencyKey, Message: fmt.Sprintf("%d bytes", report.ReclaimedBytes), ReclaimedBytes: report.ReclaimedBytes})
	_ = s.audit(ctx, AuditEvent{Type: "plan.applied", PlanID: plan.ID, IdempotencyKey: input.IdempotencyKey, Message: fmt.Sprintf("%d bytes", report.ReclaimedBytes), ReclaimedBytes: report.ReclaimedBytes})
	return report, nil
}

func (s *Service) applyProvider(ctx context.Context, plan Plan, pp ProviderPlan, input ApplyInput) (cleanup.ApplyResult, bool, error) {
	provider, err := s.providerForPlan(pp)
	if err != nil {
		return cleanup.ApplyResult{}, false, err
	}
	oneOffApproval := oneOffConditionalApproval(pp, provider.Metadata(), input)
	if !providerPolicyRunnable(pp.Policy) && !oneOffApproval {
		return cleanup.ApplyResult{}, false, nil
	}
	if skipReason := previewSkipReason(pp.Preview); skipReason != "" {
		if oneOffApproval && skipReason == "provider disabled by policy" {
			_ = s.audit(ctx, AuditEvent{Type: "provider.one_off_approved", PlanID: plan.ID, ProviderID: pp.ProviderID, IdempotencyKey: input.IdempotencyKey, Message: "conditional provider executed under explicit operator approval"})
		} else {
			_ = s.audit(ctx, AuditEvent{Type: "provider.skipped", PlanID: plan.ID, ProviderID: pp.ProviderID, IdempotencyKey: input.IdempotencyKey, Message: skipReason})
			return cleanup.ApplyResult{}, false, nil
		}
	}
	if !oneOffApproval {
		if err := requireProviderApproval(pp, input); err != nil {
			return cleanup.ApplyResult{}, false, err
		}
	}
	result, err := provider.Apply(ctx, cleanup.ApplyRequest{
		PlanID:          plan.ID,
		PolicyVersion:   plan.PolicyVersion,
		ProviderVersion: pp.ProviderVersion,
		ApprovalMode:    input.ApprovalMode,
		IdempotencyKey:  input.IdempotencyKey,
		Preview:         pp.Preview,
	})
	if err != nil {
		_ = s.audit(ctx, AuditEvent{Type: "apply.failed", PlanID: plan.ID, ProviderID: pp.ProviderID, IdempotencyKey: input.IdempotencyKey, Message: cleanup.Redact(err.Error()), Redacted: true})
		return cleanup.ApplyResult{}, false, err
	}
	// Individual items that could not be removed are reported as warnings
	// rather than as a failed Apply, because one unremovable entry must not
	// abandon the thousands behind it. They still have to reach the audit log:
	// a run that silently reclaimed less than it planned is exactly the kind of
	// degradation that goes unnoticed until the disk is full again. The
	// provider has already redacted these messages.
	if len(result.Warnings) > 0 {
		_ = s.audit(ctx, AuditEvent{
			Type:           "apply.partial",
			PlanID:         plan.ID,
			ProviderID:     pp.ProviderID,
			IdempotencyKey: input.IdempotencyKey,
			Message: fmt.Sprintf("%d of %d items could not be removed: %s",
				len(result.SkippedItems), len(pp.Preview.Items), strings.Join(result.Warnings, "; ")),
			Redacted: true,
		})
	}
	_ = s.audit(ctx, AuditEvent{Type: "provider.applied", PlanID: plan.ID, ProviderID: pp.ProviderID, IdempotencyKey: input.IdempotencyKey, Message: fmt.Sprintf("%d bytes repairs=%d retries=%d", result.ReclaimedBytes, result.Repairs, result.RetryAttempts), ReclaimedBytes: result.ReclaimedBytes})
	return result, true, nil
}

func (s *Service) providerForPlan(pp ProviderPlan) (cleanup.Provider, error) {
	provider, ok := s.registry.Get(pp.ProviderID)
	if !ok {
		return nil, fmt.Errorf("provider %q missing from registry", pp.ProviderID)
	}
	if current := provider.Metadata().Version; pp.ProviderVersion != current {
		return nil, fmt.Errorf("provider %s version mismatch: plan has %q current is %q", pp.ProviderID, pp.ProviderVersion, current)
	}
	return provider, nil
}

func providerPolicyRunnable(policy cleanup.ProviderPolicy) bool {
	return policy.Enabled && policy.ApprovalMode != cleanup.ApprovalModeDisabled
}

// oneOffConditionalApproval permits a single conditional provider whose
// durable policy is disabled by default to run only when the operator has
// supplied an explicit approval token. It does not mutate policy and cannot
// open safe or forbidden providers.
func oneOffConditionalApproval(pp ProviderPlan, meta cleanup.ProviderMetadata, input ApplyInput) bool {
	return !pp.Policy.Enabled &&
		pp.Policy.ApprovalMode != cleanup.ApprovalModeDisabled &&
		meta.SafetyTier == cleanup.SafetyTierConditional &&
		input.ApprovalMode == cleanup.ApprovalModeOperator &&
		strings.TrimSpace(input.ApprovalToken) != "" &&
		pp.Preview.BlockedReason == "provider disabled by policy" &&
		len(pp.Preview.Items) > 0
}

func previewSkipReason(preview cleanup.Preview) string {
	if preview.BlockedReason != "" {
		return preview.BlockedReason
	}
	if len(preview.Items) == 0 {
		return "no preview items"
	}
	return ""
}

func requireProviderApproval(pp ProviderPlan, input ApplyInput) error {
	if pp.Policy.ApprovalMode == cleanup.ApprovalModeNone || input.ApprovalMode == pp.Policy.ApprovalMode {
		return nil
	}
	// An operator approval is also accepted for safe_with_owner providers.
	// Their provider contract explicitly permits either owner or operator
	// approval; keeping the orchestrator gate aligned with that contract lets a
	// single cleanup plan with mixed safe tiers be applied atomically.
	if pp.Policy.ApprovalMode == cleanup.ApprovalModeOwner && input.ApprovalMode == cleanup.ApprovalModeOperator {
		return nil
	}
	return fmt.Errorf("provider %s requires %s approval", pp.ProviderID, pp.Policy.ApprovalMode)
}

func (s *Service) Audit(ctx context.Context) ([]AuditEvent, error) {
	return s.store.ListAudit(ctx)
}

func (s *Service) audit(ctx context.Context, event AuditEvent) error {
	event.ID = stableAuditID(event, s.now())
	event.Time = s.now()
	event.Message = cleanup.Redact(event.Message)
	return s.store.AddAudit(ctx, event)
}

func (s *Service) now() time.Time {
	if s.clock != nil {
		return s.clock.Now()
	}
	return time.Now().UTC()
}

func stablePolicyVersion(name policy.ProfileName, policies map[string]cleanup.ProviderPolicy) string {
	return policy.StableVersion(name, policies)
}

func stablePlanID(plan Plan) string {
	type providerFingerprint struct {
		ProviderID      string
		ProviderVersion string
		Policy          cleanup.ProviderPolicy
		Items           []cleanup.PreviewItem
		BlockedReason   string
	}
	fp := struct {
		PolicyVersion string
		Providers     []providerFingerprint
	}{PolicyVersion: plan.PolicyVersion}
	for _, pp := range plan.Providers {
		fp.Providers = append(fp.Providers, providerFingerprint{
			ProviderID:      pp.ProviderID,
			ProviderVersion: pp.ProviderVersion,
			Policy:          pp.Policy,
			Items:           pp.Preview.Items,
			BlockedReason:   pp.Preview.BlockedReason,
		})
	}
	return "plan-" + hashJSON(fp)[:24]
}

func stableAuditID(event AuditEvent, at time.Time) string {
	return "audit-" + hashJSON(struct {
		Event AuditEvent
		Time  time.Time
	}{Event: event, Time: at})[:16]
}

func hashJSON(v any) string {
	raw, _ := json.Marshal(v)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

type MemoryStore struct {
	mu      sync.Mutex
	policy  Policy
	hasPol  bool
	plans   map[string]Plan
	applies map[string]ApplyReport
	audit   []AuditEvent
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{plans: map[string]Plan{}, applies: map[string]ApplyReport{}}
}

func (s *MemoryStore) SavePolicy(_ context.Context, p Policy) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.policy = p
	s.hasPol = true
	return nil
}

func (s *MemoryStore) CurrentPolicy(context.Context) (Policy, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.policy, s.hasPol, nil
}

func (s *MemoryStore) SavePlan(_ context.Context, p Plan) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.plans[p.ID] = p
	return nil
}

func (s *MemoryStore) GetPlan(_ context.Context, id string) (Plan, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.plans[id]
	return p, ok, nil
}

func (s *MemoryStore) SaveApply(_ context.Context, r ApplyReport) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.applies[r.IdempotencyKey] = r
	return nil
}

func (s *MemoryStore) ApplyByKey(_ context.Context, key string) (ApplyReport, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.applies[key]
	return r, ok, nil
}

func (s *MemoryStore) AddAudit(_ context.Context, e AuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.audit = append(s.audit, e)
	return nil
}

func (s *MemoryStore) ListAudit(context.Context) ([]AuditEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := append([]AuditEvent(nil), s.audit...)
	sort.Slice(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	return out, nil
}
