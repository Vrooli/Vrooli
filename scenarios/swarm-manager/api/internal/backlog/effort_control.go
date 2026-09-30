package backlog

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"swarm-manager/internal/descendantbudget"
	"swarm-manager/internal/identity"
	"swarm-manager/internal/providerpool"
	"swarm-manager/internal/repairbudget"
)

// ErrEffortPolicyUnavailable reports that the approved effort workspace policy
// could not be read at admission. It is a configuration gap, not a validation
// error in the caller's authority.
var ErrEffortPolicyUnavailable = errors.New("approved effort policy is unavailable")

// ErrRepairAccountingUnavailable reports that cumulative repair accounting was
// never installed on the owner service. It is refused rather than silently
// admitting unaccounted repairs.
var ErrRepairAccountingUnavailable = errors.New("repair accounting is not configured")

// ErrProviderAccountingUnavailable reports that provider/credential-pool
// reservation accounting was never installed on the owner service. It is
// refused rather than silently admitting unaccounted provider dispatch.
var ErrProviderAccountingUnavailable = errors.New("provider pool accounting is not configured")

// ErrDescendantAccountingUnavailable reports that effort-wide descendant
// capacity accounting was never installed on the owner service. It is refused
// rather than silently admitting an unaccounted descendant against the shared
// depth/active/premium allowance.
var ErrDescendantAccountingUnavailable = errors.New("descendant accounting is not configured")

// EffortControlStore is the durable owner persistence for the effort-control
// aggregate. It stores exactly one revision per effort identity and never
// derives authority from a second writable policy.
type EffortControlStore interface {
	Load(effortID string) (identity.EffortControl, error)
	Save(control identity.EffortControl) error
	List() ([]identity.EffortControl, error)
}

// FileEffortControlStore persists one JSON document per effort identity under
// a single directory. Writes are atomic (temp file + rename) so a crashed
// admission cannot leave a half-written revision in authoritative state.
type FileEffortControlStore struct {
	rootDir string
}

// NewFileEffortControlStore roots the store at the given directory.
func NewFileEffortControlStore(rootDir string) *FileEffortControlStore {
	return &FileEffortControlStore{rootDir: rootDir}
}

func (store *FileEffortControlStore) path(effortID string) (string, error) {
	effortID = strings.TrimSpace(effortID)
	if effortID == "" || effortID == "." || effortID == ".." ||
		strings.ContainsAny(effortID, `/\`) {
		return "", fmt.Errorf("effort_id must be a single path segment")
	}
	return filepath.Join(store.rootDir, effortID+".json"), nil
}

// Load returns the stored revision, or ErrNotFound when the effort has never
// been admitted.
func (store *FileEffortControlStore) Load(effortID string) (identity.EffortControl, error) {
	path, err := store.path(effortID)
	if err != nil {
		return identity.EffortControl{}, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return identity.EffortControl{}, ErrNotFound
		}
		return identity.EffortControl{}, err
	}
	var control identity.EffortControl
	if err := json.Unmarshal(raw, &control); err != nil {
		return identity.EffortControl{}, fmt.Errorf("decode effort control %q: %w", effortID, err)
	}
	return control, nil
}

// Save writes the revision atomically.
func (store *FileEffortControlStore) Save(control identity.EffortControl) error {
	path, err := store.path(control.EffortID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(control, "", "  ")
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".effort-control-*.tmp")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		os.Remove(tempName)
		return err
	}
	if err := temp.Close(); err != nil {
		os.Remove(tempName)
		return err
	}
	if err := os.Rename(tempName, path); err != nil {
		os.Remove(tempName)
		return err
	}
	return nil
}

// List returns every admitted effort revision, sorted by effort identity so
// callers get deterministic output.
func (store *FileEffortControlStore) List() ([]identity.EffortControl, error) {
	entries, err := os.ReadDir(store.rootDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	controls := make([]identity.EffortControl, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		control, err := store.Load(strings.TrimSuffix(entry.Name(), ".json"))
		if err != nil {
			return nil, err
		}
		controls = append(controls, control)
	}
	sort.Slice(controls, func(i, j int) bool { return controls[i].EffortID < controls[j].EffortID })
	return controls, nil
}

// EffortPolicyResolution is the owner-resolved projection of one approved
// effort policy onto an admission payload. RepairLimitsPresent distinguishes an
// authored repair-limit block from an omitted one, so an authoring gap preserves
// the caller's bound instead of silently zeroing the cumulative waste limit in
// force.
type EffortPolicyResolution struct {
	Binding             identity.PolicyBinding
	Candidate           identity.CandidatePolicyBinding
	RepairLimits        identity.RepairLimits
	RepairLimitsPresent bool
}

// EffortPolicySource resolves the approved policy revision, the immutable
// candidate policy and the reviewed repair limits for an effort. It is the seam
// that binds live owner policy at admission without creating a second writable
// policy authority.
type EffortPolicySource interface {
	ResolveEffortPolicy(effortID string) (EffortPolicyResolution, error)
}

// FileEffortPolicySource reads the approved effort workspace `effort.json`.
// The policy binding digest covers only the reviewed policy object plus the
// effort slug, so lifecycle timestamps and handoff notes cannot masquerade as
// a policy change.
type FileEffortPolicySource struct {
	Root string
}

type effortPolicyDocument struct {
	Slug   string          `json:"slug"`
	Policy json.RawMessage `json:"policy"`
}

type effortPolicyConditionalFallback struct {
	CandidateRunnerModel string `json:"candidate_runner_model"`
	CandidateModel       string `json:"candidate_model"`
	Provider             string `json:"provider"`
	EligibilityState     string `json:"eligibility_state"`
}

type effortPolicyModelSelection struct {
	ModelSelection struct {
		WorkerPreference struct {
			Runner         string `json:"runner"`
			Model          string `json:"model"`
			Effort         string `json:"effort"`
			FallbackRunner string `json:"fallback_runner"`
			FallbackModel  string `json:"fallback_model"`
		} `json:"worker_preference"`
		ConditionalPaidFallback effortPolicyConditionalFallback `json:"conditional_paid_fallback"`
	} `json:"model_selection"`
}

// effortPolicyRepairBounds is the reviewed cumulative waste bound as authored in
// the approved effort workspace policy. The nested keys mirror effort.json:
// repair_limits holds the count caps, repair_time_limits holds the active-time
// caps, and the attempt caps live at the policy top level.
type effortPolicyRepairBounds struct {
	RepairLimits struct {
		Fingerprint int `json:"fingerprint"`
		Component   int `json:"component"`
		Effort      int `json:"effort"`
	} `json:"repair_limits"`
	RepairTimeLimits struct {
		ComponentActiveMinutes int     `json:"component_active_minutes"`
		EffortFraction         float64 `json:"effort_fraction_of_approved_work_allowance"`
	} `json:"repair_time_limits"`
	MaxProbeAttempts     int `json:"max_probe_attempts"`
	MaxTransportAttempts int `json:"max_transport_attempts"`
	MaxStatusReads       int `json:"max_status_reads"`
}

// projectRepairLimits maps the authored policy fields onto the admission
// payload. present=false means the policy authored no repair-limit block at all,
// so the caller's reviewed bound is preserved rather than replaced with zeroes.
func projectRepairLimits(bounds effortPolicyRepairBounds) (identity.RepairLimits, bool) {
	limits := identity.RepairLimits{
		PerFingerprint:         bounds.RepairLimits.Fingerprint,
		PerComponent:           bounds.RepairLimits.Component,
		PerEffort:              bounds.RepairLimits.Effort,
		ComponentActiveMinutes: bounds.RepairTimeLimits.ComponentActiveMinutes,
		EffortFraction:         bounds.RepairTimeLimits.EffortFraction,
		ProbeAttempts:          bounds.MaxProbeAttempts,
		TransportAttempts:      bounds.MaxTransportAttempts,
		StatusReads:            bounds.MaxStatusReads,
	}
	if limits == (identity.RepairLimits{}) {
		return identity.RepairLimits{}, false
	}
	return limits, true
}

// withheldCandidates projects the conditional paid fallback onto the immutable
// candidate-policy withholding list. A route whose eligibility is recorded as
// withheld cannot be selected as a last-resort fallback; the campaign's own
// funding/qualification state is the source of that refusal, never a global
// default.
func withheldCandidates(fallback effortPolicyConditionalFallback) []string {
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(fallback.EligibilityState)), "withheld") {
		return nil
	}
	specs := make([]string, 0, 3)
	for _, candidate := range []string{fallback.CandidateRunnerModel, fallback.CandidateModel} {
		if trimmed := strings.TrimSpace(candidate); trimmed != "" {
			specs = append(specs, trimmed)
		}
	}
	return specs
}

// ResolveEffortPolicy binds the policy digest, candidate policy and reviewed
// repair limits from the live effort workspace revision.
func (source FileEffortPolicySource) ResolveEffortPolicy(effortID string) (EffortPolicyResolution, error) {
	effortID = strings.TrimSpace(effortID)
	if effortID == "" || strings.ContainsAny(effortID, `/\`) {
		return EffortPolicyResolution{}, fmt.Errorf("effort_id must be a single path segment")
	}
	path := filepath.Join(source.Root, effortID, "effort.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return EffortPolicyResolution{}, fmt.Errorf("read approved effort policy: %w", err)
	}
	var document effortPolicyDocument
	if err := json.Unmarshal(raw, &document); err != nil {
		return EffortPolicyResolution{}, fmt.Errorf("decode approved effort policy: %w", err)
	}
	if len(document.Policy) == 0 {
		return EffortPolicyResolution{}, fmt.Errorf("approved effort policy is empty")
	}

	var compacted bytes.Buffer
	if err := json.Compact(&compacted, document.Policy); err != nil {
		return EffortPolicyResolution{}, fmt.Errorf("canonicalize approved effort policy: %w", err)
	}
	sum := sha256.Sum256(append([]byte(strings.TrimSpace(document.Slug)+"\x00"), compacted.Bytes()...))
	binding := identity.PolicyBinding{Source: path, Digest: "sha256:" + hex.EncodeToString(sum[:])}

	var selection effortPolicyModelSelection
	if err := json.Unmarshal(document.Policy, &selection); err != nil {
		return EffortPolicyResolution{}, fmt.Errorf("decode candidate policy: %w", err)
	}
	preference := selection.ModelSelection.WorkerPreference
	candidate := identity.CandidatePolicyBinding{
		EconomicalRunner: strings.TrimSpace(preference.Runner),
		EconomicalModel:  strings.TrimSpace(preference.Model),
		EconomicalEffort: strings.TrimSpace(preference.Effort),
		FallbackRunner:   strings.TrimSpace(preference.FallbackRunner),
		FallbackModel:    strings.TrimSpace(preference.FallbackModel),
		Withheld:         withheldCandidates(selection.ModelSelection.ConditionalPaidFallback),
	}.Bind()

	var bounds effortPolicyRepairBounds
	if err := json.Unmarshal(document.Policy, &bounds); err != nil {
		return EffortPolicyResolution{}, fmt.Errorf("decode repair policy: %w", err)
	}
	repairLimits, repairLimitsPresent := projectRepairLimits(bounds)

	return EffortPolicyResolution{
		Binding:             binding,
		Candidate:           candidate,
		RepairLimits:        repairLimits,
		RepairLimitsPresent: repairLimitsPresent,
	}, nil
}

// EffortControlService is the owner admission and amendment surface for the
// effort-control aggregate. Every mutation is version-checked against the
// durable prior revision before it is persisted.
type EffortControlService struct {
	store          EffortControlStore
	policy         EffortPolicySource
	repairs        *repairbudget.Ledger
	repairRoot     string
	providerRoot   string
	descendantRoot string
}

// NewEffortControlService builds the owner service over durable persistence
// and an optional live policy source. A nil policy source preserves whatever
// binding the caller supplied, which tests and non-effort callers rely on.
func NewEffortControlService(store EffortControlStore, policy EffortPolicySource) *EffortControlService {
	return &EffortControlService{store: store, policy: policy}
}

// WithRepairLedger installs cumulative repair accounting bound to the effort
// identity. It is optional and additive: without it the service retains its
// prior admission behavior, and repair reservations are refused with
// ErrRepairAccountingUnavailable rather than admitted unaccounted.
func (service *EffortControlService) WithRepairLedger(ledger *repairbudget.Ledger) *EffortControlService {
	service.repairs = ledger
	return service
}

// WithRepairLedgerRoot installs a per-effort durable repair-ledger root. Each
// admitted effort gets its own ledger document, so cumulative investment can
// never leak between efforts and no second writable policy is created.
func (service *EffortControlService) WithRepairLedgerRoot(root string) *EffortControlService {
	service.repairRoot = strings.TrimSpace(root)
	return service
}

// WithProviderPoolRoot installs a shared provider/credential-pool accounting
// root. Pool documents are shared across the whole effort so concurrent or
// recursive workers cannot oversubscribe one credential/allowance pool.
func (service *EffortControlService) WithProviderPoolRoot(root string) *EffortControlService {
	service.providerRoot = strings.TrimSpace(root)
	return service
}

// WithDescendantRoot installs a per-effort durable descendant-capacity root.
// Each admitted effort gets its own document, and every branch of that effort
// draws on the one shared depth/active/premium allowance.
func (service *EffortControlService) WithDescendantRoot(root string) *EffortControlService {
	service.descendantRoot = strings.TrimSpace(root)
	return service
}

// repairLedger resolves the durable ledger for one effort. An explicitly
// installed single ledger (single-effort tests and callers) takes precedence;
// otherwise every effort gets its own document under the configured root.
func (service *EffortControlService) repairLedger(effortID string) (*repairbudget.Ledger, error) {
	if service.repairs != nil {
		return service.repairs, nil
	}
	if service.repairRoot == "" {
		return nil, ErrRepairAccountingUnavailable
	}
	effortID = strings.TrimSpace(effortID)
	if effortID == "" || effortID == "." || effortID == ".." || strings.ContainsAny(effortID, `/\`) {
		return nil, fmt.Errorf("effort_id must be a single path segment")
	}
	return repairbudget.New(effortID, filepath.Join(service.repairRoot, effortID+".json")), nil
}

// BeginRepair reserves and charges one intervention against the effort's
// approved cumulative repair limits. It refuses before any effect when the
// effort is unknown, accounting is not configured, or a limit is reached.
func (service *EffortControlService) BeginRepair(effortID string, event repairbudget.Event) (repairbudget.Admission, error) {
	ledger, err := service.repairLedger(effortID)
	if err != nil {
		return repairbudget.Admission{}, err
	}
	control, err := service.store.Load(effortID)
	if err != nil {
		return repairbudget.Admission{}, err
	}
	return ledger.Begin(resolveRepairTimeLimits(control), event)
}

// resolveRepairTimeLimits binds the reviewed diversion fraction to the approved
// aggregate work allowance. The authored RepairLimits may carry the fraction
// without the numeric denominator; the aggregate wall bound is the approved work
// allowance the fraction applies to, so the ledger can enforce the time cap
// without a second writable policy. An unbounded/zero aggregate leaves the
// fraction unenforceable and only the per-component cap applies.
func resolveRepairTimeLimits(control identity.EffortControl) identity.RepairLimits {
	limits := control.RepairLimits
	if limits.ApprovedActiveMinutes > 0 {
		return limits
	}
	if seconds := control.AggregateLimits.MaxWallSeconds; seconds > 0 {
		limits.ApprovedActiveMinutes = int((seconds + 59) / 60)
	}
	return limits
}

// FinishRepair records the disposition of a reserved intervention. It charges
// nothing further, so an interrupted attempt stays charged from Begin.
func (service *EffortControlService) FinishRepair(effortID, attemptID, outcome string, evidence ...string) (repairbudget.Event, error) {
	ledger, err := service.repairLedger(effortID)
	if err != nil {
		return repairbudget.Event{}, err
	}
	return ledger.Finish(attemptID, outcome, evidence...)
}

// RepairTotals returns the cumulative repair investment for one effort.
func (service *EffortControlService) RepairTotals(effortID string) (repairbudget.Totals, error) {
	ledger, err := service.repairLedger(effortID)
	if err != nil {
		return repairbudget.Totals{}, err
	}
	return ledger.Totals()
}

// AcknowledgeDispatch records the route owner's start acknowledgement for a
// reserved intervention, so a later lost callback can be reconciled under the
// same operation identity rather than replaced.
func (service *EffortControlService) AcknowledgeDispatch(effortID, attemptID, ownerWork, ownerRun string) (repairbudget.Event, error) {
	ledger, err := service.repairLedger(effortID)
	if err != nil {
		return repairbudget.Event{}, err
	}
	return ledger.Acknowledge(attemptID, ownerWork, ownerRun)
}

// MarkDispatchUncertain records that a dispatch's start response was lost. A
// replacement stays refused until the operation is reconciled.
func (service *EffortControlService) MarkDispatchUncertain(effortID, attemptID, reason string) (repairbudget.Event, error) {
	ledger, err := service.repairLedger(effortID)
	if err != nil {
		return repairbudget.Event{}, err
	}
	return ledger.MarkUncertain(attemptID, reason)
}

// ReconcileDispatch records an owner observation for a possibly active
// operation and returns its durable dispatch state.
func (service *EffortControlService) ReconcileDispatch(effortID, attemptID string, ownerActive bool, ownerRun, reason string) (repairbudget.DispatchState, error) {
	ledger, err := service.repairLedger(effortID)
	if err != nil {
		return repairbudget.DispatchState{}, err
	}
	return ledger.Reconcile(attemptID, ownerActive, ownerRun, reason)
}

// FallbackDispatch re-dispatches a reconciled operation to another qualified
// route under the same charged identity, preserving the remaining allowance.
func (service *EffortControlService) FallbackDispatch(effortID, attemptID, ownerWork, ownerRun string) (repairbudget.Event, error) {
	ledger, err := service.repairLedger(effortID)
	if err != nil {
		return repairbudget.Event{}, err
	}
	return ledger.Fallback(attemptID, ownerWork, ownerRun)
}

// DispatchState returns the route-owner state for one effort operation.
func (service *EffortControlService) DispatchState(effortID, attemptID string) (repairbudget.DispatchState, error) {
	ledger, err := service.repairLedger(effortID)
	if err != nil {
		return repairbudget.DispatchState{}, err
	}
	return ledger.Dispatch(attemptID)
}

// OutstandingDispatches lists non-terminal operations for one effort.
func (service *EffortControlService) OutstandingDispatches(effortID string) ([]repairbudget.DispatchState, error) {
	ledger, err := service.repairLedger(effortID)
	if err != nil {
		return nil, err
	}
	return ledger.Outstanding()
}

// CancelDispatch fences an operation terminal before effects are forwarded to
// an owner, so a delayed start cannot revive cancelled work.
func (service *EffortControlService) CancelDispatch(effortID, attemptID, reason string) (repairbudget.Event, error) {
	ledger, err := service.repairLedger(effortID)
	if err != nil {
		return repairbudget.Event{}, err
	}
	return ledger.Cancel(attemptID, reason)
}

// QualifyLastResortCandidate decides whether the admitted effort's immutable
// candidate-policy binding permits a last-resort fallback to select the proposed
// runner/model. It is the owner answer that refuses a withheld, unbound,
// unqualified or unknown-billing candidate before any dispatch effect.
func (service *EffortControlService) QualifyLastResortCandidate(effortID string, candidate identity.Candidate) (identity.CandidateQualification, error) {
	control, err := service.store.Load(effortID)
	if err != nil {
		return identity.CandidateQualification{}, err
	}
	qualification := identity.CandidateQualification{
		Runner: strings.TrimSpace(candidate.Runner),
		Model:  strings.TrimSpace(candidate.Model),
	}
	if err := control.CandidatePolicy.QualifyLastResort(candidate); err != nil {
		return qualification, err
	}
	qualification.Qualified = true
	return qualification, nil
}

// providerPool resolves the shared provider/credential-pool document for one
// pool reference. The document is keyed by the pool, not the effort, so
// descendants of the same effort cannot each mint a private allowance.
func (service *EffortControlService) providerPool(poolRef string) (*providerpool.Pool, error) {
	poolRef = strings.TrimSpace(poolRef)
	if poolRef == "" || strings.ContainsAny(poolRef, `/\`) {
		return nil, fmt.Errorf("pool reference must be a single path segment")
	}
	if service.providerRoot == "" {
		return nil, ErrProviderAccountingUnavailable
	}
	return providerpool.NewForRoot(service.providerRoot, poolRef)
}

// ObserveProviderLimit records a typed runner-limit observation for one pool.
// A known reset is a time condition; an unknown reset requires a later explicit
// recovery observation, never a polling heartbeat.
func (service *EffortControlService) ObserveProviderLimit(effortID string, obs providerpool.Observation) (providerpool.State, error) {
	if _, err := service.store.Load(effortID); err != nil {
		return providerpool.State{}, err
	}
	pool, err := service.providerPool(obs.Pool)
	if err != nil {
		return providerpool.State{}, err
	}
	return pool.Observe(obs)
}

// ObserveProviderFailure records an observed AI Gateway provider failure as a
// pool observation. It translates the gateway's failure-class vocabulary to the
// pool's recovery classes and carries the observed Retry-After/reset/source
// provenance verbatim. A failure class with no shared-pool recovery condition
// (a per-attempt execution fault) returns the pool's current state unchanged,
// so a forwarder can report every provider failure without pausing the pool for
// an unrelated transient fault.
func (service *EffortControlService) ObserveProviderFailure(effortID string, failure providerpool.ProviderFailure) (providerpool.State, error) {
	if _, err := service.store.Load(effortID); err != nil {
		return providerpool.State{}, err
	}
	obs, ok := providerpool.ObservationFromProviderFailure(failure)
	if !ok {
		pool, err := service.providerPool(failure.Pool)
		if err != nil {
			return providerpool.State{}, err
		}
		return pool.State()
	}
	pool, err := service.providerPool(obs.Pool)
	if err != nil {
		return providerpool.State{}, err
	}
	return pool.Observe(obs)
}

// ProviderPoolState returns the current admission state for one pool.
func (service *EffortControlService) ProviderPoolState(effortID, poolRef string) (providerpool.State, error) {
	if _, err := service.store.Load(effortID); err != nil {
		return providerpool.State{}, err
	}
	pool, err := service.providerPool(poolRef)
	if err != nil {
		return providerpool.State{}, err
	}
	return pool.State()
}

// ReserveProviderSlot accounts one dispatch against the effort's approved
// aggregate grant before the owner is contacted. The concurrency and unit caps
// come from the admitted effort revision, so a child cannot exceed the one
// shared allowance.
func (service *EffortControlService) ReserveProviderSlot(effortID string, reservation providerpool.Reservation) (providerpool.Reservation, error) {
	control, err := service.store.Load(effortID)
	if err != nil {
		return providerpool.Reservation{}, err
	}
	pool, err := service.providerPool(reservation.Pool)
	if err != nil {
		return providerpool.Reservation{}, err
	}
	reservation.EffortID = effortID
	limit := providerPoolLimit(control)
	return pool.Reserve(limit, reservation)
}

// SettleProviderSlot records terminal usage. An unknown measurement stays
// conservatively reserved until a later known settle resolves it.
func (service *EffortControlService) SettleProviderSlot(effortID, poolRef, attemptID string, used int64, known bool) (providerpool.Reservation, error) {
	if _, err := service.store.Load(effortID); err != nil {
		return providerpool.Reservation{}, err
	}
	pool, err := service.providerPool(poolRef)
	if err != nil {
		return providerpool.Reservation{}, err
	}
	return pool.Settle(attemptID, used, known)
}

// ReleaseProviderSlot finalizes a reservation whose effect never started.
func (service *EffortControlService) ReleaseProviderSlot(effortID, poolRef, attemptID string) (providerpool.Reservation, error) {
	if _, err := service.store.Load(effortID); err != nil {
		return providerpool.Reservation{}, err
	}
	pool, err := service.providerPool(poolRef)
	if err != nil {
		return providerpool.Reservation{}, err
	}
	return pool.Release(attemptID)
}

// ProviderReservations lists the pool's reservations for reconciliation.
func (service *EffortControlService) ProviderReservations(effortID, poolRef string) ([]providerpool.Reservation, error) {
	if _, err := service.store.Load(effortID); err != nil {
		return nil, err
	}
	pool, err := service.providerPool(poolRef)
	if err != nil {
		return nil, err
	}
	return pool.Reservations()
}

// ProviderPoolTotals returns held and settled investment for one pool.
func (service *EffortControlService) ProviderPoolTotals(effortID, poolRef string) (providerpool.Totals, error) {
	if _, err := service.store.Load(effortID); err != nil {
		return providerpool.Totals{}, err
	}
	pool, err := service.providerPool(poolRef)
	if err != nil {
		return providerpool.Totals{}, err
	}
	return pool.Totals()
}

// ProviderPoolLimitRefresh reports one pool whose persisted bound was re-derived
// from the admitted effort revision.
type ProviderPoolLimitRefresh struct {
	Pool  string             `json:"pool"`
	Limit providerpool.Limit `json:"limit"`
	State providerpool.State `json:"state"`
}

// providerPoolLimit projects the reviewed aggregate grant onto the shared
// provider-pool bound. MaxTokens zero means the effort left the shared unit
// allowance unspecified; MaxConcurrency is always at least one by validation.
func providerPoolLimit(control identity.EffortControl) providerpool.Limit {
	return providerpool.Limit{
		MaxConcurrent: control.AggregateLimits.MaxConcurrency,
		MaxUnits:      control.AggregateLimits.MaxTokens,
	}
}

// RefreshProviderPoolLimits re-derives the shared provider-pool bounds from the
// admitted effort revision and persists them on every pool attributed to this
// effort. Because a pool's stored bound is otherwise refreshed only on its next
// reservation, an amendment must call this to keep the persisted bound, and any
// read of it, consistent with the reviewed grant. A pool shared with a different
// effort is left untouched so identity is preserved. An absent provider-pool
// root reports ErrProviderAccountingUnavailable rather than pretending success.
func (service *EffortControlService) RefreshProviderPoolLimits(effortID string) ([]ProviderPoolLimitRefresh, error) {
	control, err := service.store.Load(effortID)
	if err != nil {
		return nil, err
	}
	reports, configured, err := service.refreshProviderPoolLimits(control)
	if err != nil {
		return reports, err
	}
	if !configured {
		return nil, ErrProviderAccountingUnavailable
	}
	return reports, nil
}

// refreshProviderPoolLimits applies an already-loaded revision. The configured
// result distinguishes "accounting not installed" (best-effort skip on
// amendment) from an actual persistence failure.
func (service *EffortControlService) refreshProviderPoolLimits(control identity.EffortControl) ([]ProviderPoolLimitRefresh, bool, error) {
	if service.providerRoot == "" {
		return nil, false, nil
	}
	pools, err := providerpool.ListForRoot(service.providerRoot)
	if err != nil {
		return nil, true, err
	}
	limit := providerPoolLimit(control)
	var reports []ProviderPoolLimitRefresh
	for _, pool := range pools {
		attributed, err := pool.AttributedTo(control.EffortID)
		if err != nil {
			return reports, true, err
		}
		if !attributed {
			continue
		}
		state, err := pool.SetLimit(limit)
		if err != nil {
			return reports, true, err
		}
		reports = append(reports, ProviderPoolLimitRefresh{Pool: pool.Pool(), Limit: limit, State: state})
	}
	return reports, true, nil
}

// descendantBudget resolves the durable descendant-capacity document for one
// effort. The document is keyed by the effort, so every branch of the same
// effort shares one allowance and cannot mint a private one.
func (service *EffortControlService) descendantBudget(effortID string) (*descendantbudget.Budget, error) {
	effortID = strings.TrimSpace(effortID)
	if effortID == "" || effortID == "." || effortID == ".." || strings.ContainsAny(effortID, `/\`) {
		return nil, fmt.Errorf("effort_id must be a single path segment")
	}
	if service.descendantRoot == "" {
		return nil, ErrDescendantAccountingUnavailable
	}
	return descendantbudget.NewForRoot(service.descendantRoot, effortID)
}

// AdmitDescendant charges one descendant slot against the effort's approved
// aggregate grant before the descendant is dispatched. The depth, active and
// premium caps come from the admitted effort revision, so a nested planner or
// concurrent worker cannot exceed the one shared allowance.
func (service *EffortControlService) AdmitDescendant(effortID string, reservation descendantbudget.Reservation) (descendantbudget.Reservation, error) {
	control, err := service.store.Load(effortID)
	if err != nil {
		return descendantbudget.Reservation{}, err
	}
	budget, err := service.descendantBudget(effortID)
	if err != nil {
		return descendantbudget.Reservation{}, err
	}
	reservation.EffortID = effortID
	limit := descendantbudget.Limit{
		MaxDepth:              control.AggregateLimits.MaxDepth,
		MaxActiveDescendants:  control.AggregateLimits.MaxActiveDescendants,
		MaxPremiumDescendants: control.AggregateLimits.MaxPremiumDescendants,
	}
	return budget.Admit(limit, reservation)
}

// ReleaseDescendant ends one descendant reservation, returning its slot to the
// shared allowance.
func (service *EffortControlService) ReleaseDescendant(effortID, attemptID string) (descendantbudget.Reservation, error) {
	if _, err := service.store.Load(effortID); err != nil {
		return descendantbudget.Reservation{}, err
	}
	budget, err := service.descendantBudget(effortID)
	if err != nil {
		return descendantbudget.Reservation{}, err
	}
	return budget.Release(attemptID)
}

// DescendantReservations lists the effort's descendant reservations for
// reconciliation.
func (service *EffortControlService) DescendantReservations(effortID string) ([]descendantbudget.Reservation, error) {
	if _, err := service.store.Load(effortID); err != nil {
		return nil, err
	}
	budget, err := service.descendantBudget(effortID)
	if err != nil {
		return nil, err
	}
	return budget.Reservations()
}

// DescendantTotals returns active and ended descendant capacity for one effort.
func (service *EffortControlService) DescendantTotals(effortID string) (descendantbudget.Totals, error) {
	if _, err := service.store.Load(effortID); err != nil {
		return descendantbudget.Totals{}, err
	}
	budget, err := service.descendantBudget(effortID)
	if err != nil {
		return descendantbudget.Totals{}, err
	}
	return budget.Totals()
}

// AggregateChildGrant is the effort-wide child-limit projection resolved from
// the admitted revision that references a canonical plan. Zero fields mean the
// effort does not bound that dimension, so the caller keeps its documented
// default rather than treating zero as an unlimited allowance.
type AggregateChildGrant struct {
	EffortID       string
	MaxConcurrency int
	MaxRecursion   int
	MaxWaitSeconds int
}

// ChildGrantForPlan resolves the admitted effort whose work references include
// the canonical plan and projects its aggregate child limits. It is the owner
// answer that binds previously hard-coded child limits to the approved
// aggregate grant. ok=false means no admitted effort references the plan, so
// the caller keeps its documented defaults.
func (service *EffortControlService) ChildGrantForPlan(planID string) (AggregateChildGrant, bool, error) {
	planID = strings.TrimSpace(planID)
	if planID == "" {
		return AggregateChildGrant{}, false, nil
	}
	controls, err := service.store.List()
	if err != nil {
		return AggregateChildGrant{}, false, err
	}
	for _, control := range controls {
		for _, reference := range control.WorkReferences {
			if strings.EqualFold(strings.TrimSpace(reference.Kind), "plan") && strings.TrimSpace(reference.ID) == planID {
				return AggregateChildGrant{
					EffortID:       control.EffortID,
					MaxConcurrency: control.AggregateLimits.MaxConcurrency,
					MaxRecursion:   control.AggregateLimits.MaxDepth,
					MaxWaitSeconds: control.AggregateLimits.MaxWaitSeconds,
				}, true, nil
			}
		}
	}
	return AggregateChildGrant{}, false, nil
}

// Admit creates the first revision of an effort. Admitting over an existing
// effort is a revision conflict: authority changes only through Amend.
func (service *EffortControlService) Admit(control identity.EffortControl) (identity.EffortControl, error) {
	if err := service.bindPolicy(&control); err != nil {
		return identity.EffortControl{}, err
	}
	control.Completion = identity.CompletionStanding{}
	current, err := service.store.Load(control.EffortID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return identity.EffortControl{}, err
	}
	var currentPtr *identity.EffortControl
	if err == nil {
		currentPtr = &current
	}
	admitted, err := control.Admit(currentPtr)
	if err != nil {
		return identity.EffortControl{}, err
	}
	if err := service.store.Save(admitted); err != nil {
		return identity.EffortControl{}, err
	}
	return admitted, nil
}

// Amend changes the reviewed authority at the exact next revision and carries
// the prior runtime completion standing forward.
func (service *EffortControlService) Amend(control identity.EffortControl) (identity.EffortControl, error) {
	if err := service.bindPolicy(&control); err != nil {
		return identity.EffortControl{}, err
	}
	current, err := service.store.Load(control.EffortID)
	if err != nil {
		return identity.EffortControl{}, err
	}
	control.Completion = current.Completion
	amended, err := control.Amend(current)
	if err != nil {
		return identity.EffortControl{}, err
	}
	if err := service.store.Save(amended); err != nil {
		return identity.EffortControl{}, err
	}
	return amended, nil
}

// Get returns the current admitted revision.
func (service *EffortControlService) Get(effortID string) (identity.EffortControl, error) {
	return service.store.Load(effortID)
}

// MarkEvidenceComplete records owner evidence completion without ever setting
// human product acceptance.
func (service *EffortControlService) MarkEvidenceComplete(effortID string, refs ...string) (identity.EffortControl, error) {
	control, err := service.store.Load(effortID)
	if err != nil {
		return identity.EffortControl{}, err
	}
	control.Completion = control.Completion.MarkEvidenceComplete(refs...)
	if err := service.store.Save(control); err != nil {
		return identity.EffortControl{}, err
	}
	return control, nil
}

// HumanAccept records an authenticated human product disposition. A blank
// actor is refused so evidence completion cannot substitute for a human.
func (service *EffortControlService) HumanAccept(effortID, actor string) (identity.EffortControl, error) {
	control, err := service.store.Load(effortID)
	if err != nil {
		return identity.EffortControl{}, err
	}
	updated, err := control.Completion.HumanAccept(actor)
	if err != nil {
		return identity.EffortControl{}, err
	}
	control.Completion = updated
	if err := service.store.Save(control); err != nil {
		return identity.EffortControl{}, err
	}
	return control, nil
}

// bindPolicy stamps the live approved policy, immutable candidate policy and
// reviewed repair limits onto the revision before validation. An authored
// repair-limit block replaces the caller's bound; an omitted block preserves it,
// so the approved effort policy is the single source of the cumulative waste
// limit without a second writable authority.
func (service *EffortControlService) bindPolicy(control *identity.EffortControl) error {
	if service.policy == nil {
		return nil
	}
	resolution, err := service.policy.ResolveEffortPolicy(control.EffortID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrEffortPolicyUnavailable, err)
	}
	control.PolicyBinding = resolution.Binding
	control.CandidatePolicy = resolution.Candidate
	if resolution.RepairLimitsPresent {
		control.RepairLimits = resolution.RepairLimits
	}
	return nil
}
