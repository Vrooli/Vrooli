package orchestrator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"storage-manager/internal/cleanup"
	"storage-manager/internal/policy"
	"storage-manager/internal/providers"
)

type Policy struct {
	Version   string
	Profile   policy.ProfileName
	Providers map[string]cleanup.ProviderPolicy
	CreatedAt time.Time
}

type ProviderPlan struct {
	ProviderID      string
	ProviderVersion string
	Estimate        cleanup.Estimate
	Preview         cleanup.Preview
	Policy          cleanup.ProviderPolicy
}

type Plan struct {
	ID            string
	PolicyVersion string
	CreatedAt     time.Time
	Providers     []ProviderPlan
	TotalBytes    int64
	TotalItems    int
}

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

type AuditEvent struct {
	ID             string
	Time           time.Time
	Type           string
	PlanID         string
	ProviderID     string
	IdempotencyKey string
	Message        string
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

	// Disk-pressure intake state. pressure collapses duplicate reports of the
	// same event; autonomousApply is the kill switch for unattended deletion.
	pressure        *pressureGuard
	autonomousMu    sync.RWMutex
	autonomousApply bool
}

// defaultPressureDedupWindow is how long a completed autonomous execution
// suppresses another for the same partition and band. It is long enough to
// cover both safeguards reporting the same event and short enough that
// genuinely renewed pressure is acted on.
const defaultPressureDedupWindow = 5 * time.Minute

func NewService(registry *providers.Registry, store Store, clock cleanup.Clock) *Service {
	return &Service{
		registry: registry,
		store:    store,
		clock:    clock,
		pressure: newPressureGuard(defaultPressureDedupWindow),
		// Autonomous apply is on by default: the incident happened because
		// nothing acted overnight. The kill switch exists to turn remediation
		// off deliberately, not to require a deliberate act to turn it on.
		autonomousApply: true,
	}
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

func (s *Service) CurrentPolicy(ctx context.Context) (Policy, error) {
	if existing, ok, err := s.store.CurrentPolicy(ctx); err != nil {
		return Policy{}, err
	} else if ok {
		return existing, nil
	}
	return s.SetPolicyProfile(ctx, policy.ProfileBalanced)
}

func (s *Service) Plan(ctx context.Context, scope cleanup.ObservationScope) (Plan, error) {
	current, err := s.CurrentPolicy(ctx)
	if err != nil {
		return Plan{}, err
	}
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
	out := Plan{PolicyVersion: current.Version, CreatedAt: s.now(), Providers: providerPlans}
	for _, pp := range out.Providers {
		out.TotalBytes += pp.Estimate.EstimatedBytes
		out.TotalItems += pp.Estimate.ItemCount
	}
	out.ID = stablePlanID(out)
	if err := s.store.SavePlan(ctx, out); err != nil {
		return Plan{}, err
	}
	_ = s.audit(ctx, AuditEvent{Type: "plan.created", PlanID: out.ID, Message: fmt.Sprintf("%d providers", len(out.Providers))})
	return out, nil
}

func (s *Service) Apply(ctx context.Context, input ApplyInput) (ApplyReport, error) {
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
	_ = s.audit(ctx, AuditEvent{Type: "apply.completed", PlanID: plan.ID, IdempotencyKey: input.IdempotencyKey, Message: fmt.Sprintf("%d bytes", report.ReclaimedBytes)})
	return report, nil
}

func (s *Service) applyProvider(ctx context.Context, plan Plan, pp ProviderPlan, input ApplyInput) (cleanup.ApplyResult, bool, error) {
	provider, err := s.providerForPlan(pp)
	if err != nil {
		return cleanup.ApplyResult{}, false, err
	}
	if !providerPolicyRunnable(pp.Policy) {
		return cleanup.ApplyResult{}, false, nil
	}
	if skipReason := previewSkipReason(pp.Preview); skipReason != "" {
		_ = s.audit(ctx, AuditEvent{Type: "provider.skipped", PlanID: plan.ID, ProviderID: pp.ProviderID, IdempotencyKey: input.IdempotencyKey, Message: skipReason})
		return cleanup.ApplyResult{}, false, nil
	}
	if err := requireProviderApproval(pp, input); err != nil {
		return cleanup.ApplyResult{}, false, err
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
	_ = s.audit(ctx, AuditEvent{Type: "provider.applied", PlanID: plan.ID, ProviderID: pp.ProviderID, IdempotencyKey: input.IdempotencyKey, Message: fmt.Sprintf("%d bytes", result.ReclaimedBytes)})
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
	return "policy-" + hashJSON(struct {
		Name     policy.ProfileName
		Policies map[string]cleanup.ProviderPolicy
	}{Name: name, Policies: policies})[:16]
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
