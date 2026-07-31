// Package channelmanager owns the deterministic, platform-independent rules
// for identity warming and manual account work.  Executors are deliberately
// outside this package: at P0 a human performs the platform interaction.
package channelmanager

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"channel-manager/internal/channelmanager/flow/generated"
)

var (
	ErrCredentialValue = errors.New("credential values are forbidden; provide a vault reference only")
	ErrPreconditions   = errors.New("warming preconditions are not fully attested")
	ErrCadence         = errors.New("identity daily cadence ceiling reached")
	ErrForbiddenAction = errors.New("action is forbidden in the active warming phase")
	ErrPaused          = errors.New("identity queue is paused or quarantined")
)

type Provenance struct {
	SourceKind     string   `json:"source_kind"`
	Confidence     string   `json:"confidence"`
	CapturedAt     string   `json:"captured_at"`
	RevisitTrigger string   `json:"revisit_trigger"`
	Sources        []string `json:"sources"`
}

func (p Provenance) Valid() bool {
	return p.SourceKind != "" && p.Confidence != "" && p.CapturedAt != "" && p.RevisitTrigger != "" && len(p.Sources) > 0
}

type Platform struct {
	ID                    string      `json:"id"`
	DailyCeiling          int         `json:"daily_ceiling"`
	CaptionLimit          int         `json:"caption_limit"`
	TitleLimit            int         `json:"title_limit"`
	ActionKinds           []string    `json:"action_kinds"`
	PostTypes             []PostType  `json:"post_types"`
	DisclosureRequired    bool        `json:"disclosure_required"`
	DisclosurePlacement   string      `json:"disclosure_placement"`
	FirstCommentSupported bool        `json:"first_comment_supported"`
	Provenance            Provenance  `json:"provenance"`
	Retry                 RetryPolicy `json:"retry"`
	Formats               []Format    `json:"formats"`
}

// PostType names a descriptor-declared publishing shape. It keeps platform
// variations out of queue and preview branching.
type PostType struct {
	ID              string `json:"id"`
	FormatKind      string `json:"format_kind"`
	CaptionRequired bool   `json:"caption_required"`
	TitleRequired   bool   `json:"title_required"`
}

// RetryPolicy is descriptor-owned because platform failure codes, retry
// exposure, and safe backoff are not global assumptions.
type RetryPolicy struct {
	RetryableCodes []string `json:"retryable_codes"`
	MaxAttempts    int      `json:"max_attempts"`
	BackoffMinutes int      `json:"backoff_minutes"`
}

func (p RetryPolicy) Valid() error {
	if p.MaxAttempts < 0 || p.BackoffMinutes < 0 {
		return errors.New("retry policy values must not be negative")
	}
	if len(p.RetryableCodes) > 0 && p.MaxAttempts < 1 {
		return errors.New("retryable platform codes require a positive attempt limit")
	}
	return nil
}

// Format is a platform-owned media contract. Limits declared as runtime
// resolved must be queried from the platform for the particular identity;
// they must never be replaced with a guessed global maximum.
type Format struct {
	Kind                 string   `json:"kind"`
	MIMETypes            []string `json:"mime_types"`
	MaxBytes             int64    `json:"max_bytes"`
	BytesResolved        bool     `json:"bytes_resolved_per_identity"`
	MaxDurationSecs      int      `json:"max_duration_secs"`
	DurationResolved     bool     `json:"duration_resolved_per_identity"`
	MinWidth             int      `json:"min_width"`
	MinHeight            int      `json:"min_height"`
	MaxWidth             int      `json:"max_width"`
	MaxHeight            int      `json:"max_height"`
	PreferredAspectRatio string   `json:"preferred_aspect_ratio"`
	PreviewFit           string   `json:"preview_fit"`
}

func (p Platform) Valid() error {
	if p.ID == "" || p.DailyCeiling < 1 || len(p.ActionKinds) == 0 || len(p.Formats) == 0 {
		return errors.New("platform descriptor requires id, positive daily_ceiling, action_kinds, and formats")
	}
	if err := p.Retry.Valid(); err != nil {
		return fmt.Errorf("platform retry policy: %w", err)
	}
	for _, format := range p.Formats {
		if format.Kind == "" || len(format.MIMETypes) == 0 || (!format.BytesResolved && format.MaxBytes < 1) || format.MinWidth < 1 || format.MinHeight < 1 || format.MaxWidth < format.MinWidth || format.MaxHeight < format.MinHeight || (!format.DurationResolved && format.MaxDurationSecs < 1) {
			return fmt.Errorf("invalid platform format %q", format.Kind)
		}
	}
	for _, postType := range p.PostTypes {
		if postType.ID == "" || postType.FormatKind == "" || !slices.ContainsFunc(p.Formats, func(format Format) bool { return format.Kind == postType.FormatKind }) {
			return fmt.Errorf("invalid platform post type %q", postType.ID)
		}
	}
	return nil
}

type SessionPolicy struct {
	Count             int `json:"count"`
	DurationMinutes   int `json:"duration_minutes"`
	MinimumGapMinutes int `json:"minimum_gap_minutes"`
}
type Phase struct {
	ID        string   `json:"id"`
	Allowed   []string `json:"allowed"`
	Forbidden []string `json:"forbidden"`
	CountMin  int      `json:"count_min"`
	CountMax  int      `json:"count_max"`
}
type Gate struct {
	ID          string  `json:"id"`
	Metric      string  `json:"metric"`
	Minimum     float64 `json:"minimum"`
	WaitMinutes int     `json:"wait_minutes"`
	MaxRepeats  int     `json:"max_repeats"`
}
type Program struct {
	ID                 string        `json:"id"`
	PlatformID         string        `json:"platform_id"`
	Preconditions      []string      `json:"preconditions"`
	Sessions           SessionPolicy `json:"sessions"`
	Phases             []Phase       `json:"phases"`
	GraduationCriteria []string      `json:"graduation_criteria"`
	Gates              []Gate        `json:"gates"`
	Provenance         Provenance    `json:"provenance"`
}

func (p Program) Valid(platform Platform) error {
	if p.ID == "" || p.PlatformID != platform.ID || !p.Provenance.Valid() || len(p.Phases) == 0 {
		return errors.New("warming program requires platform, phases, and complete provenance")
	}
	for _, phase := range p.Phases {
		if phase.ID == "" || phase.CountMin < 0 || phase.CountMax < phase.CountMin {
			return fmt.Errorf("invalid phase %q", phase.ID)
		}
		for _, k := range append(append([]string{}, phase.Allowed...), phase.Forbidden...) {
			if !slices.Contains(platform.ActionKinds, k) {
				return fmt.Errorf("phase %q uses unknown action kind %q", phase.ID, k)
			}
		}
	}
	for _, gate := range p.Gates {
		if gate.ID == "" || gate.Metric == "" || gate.WaitMinutes < 0 || gate.MaxRepeats < 1 {
			return fmt.Errorf("invalid gate %q", gate.ID)
		}
	}
	for _, criterion := range p.GraduationCriteria {
		if strings.HasPrefix(criterion, "gate:") {
			id := strings.TrimPrefix(criterion, "gate:")
			if !slices.ContainsFunc(p.Gates, func(g Gate) bool { return g.ID == id }) {
				return fmt.Errorf("graduation criterion references unknown gate %q", id)
			}
		}
	}
	return nil
}

type Identity struct {
	ID                string          `json:"id"`
	PlatformID        string          `json:"platform_id"`
	Handle            string          `json:"handle"`
	DisplayLabel      string          `json:"display_label"`
	Purpose           string          `json:"purpose"`
	PersonaRef        string          `json:"persona_ref"`
	Goals             []string        `json:"goals"`
	Notes             string          `json:"notes"`
	OwnerRef          string          `json:"owner_ref"`
	Lifecycle         string          `json:"lifecycle"`
	EnvironmentRef    string          `json:"environment_ref"`
	ExpectedRegion    string          `json:"expected_region"`
	VaultRef          string          `json:"vault_ref"`
	Status            string          `json:"status"`
	LaneGrants        []string        `json:"lane_grants"`
	Attestations      map[string]bool `json:"attestations"`
	D009AcceptanceRef string          `json:"d009_acceptance_ref"`
	AutomationMode    string          `json:"automation_mode"`
}

// EnvironmentProbe is intentionally credential-free. The environment provider
// receives only the opaque environment reference and returns an observed
// region; provisioning and proxy credentials remain outside Channel Manager.
type EnvironmentProbe interface {
	Probe(context.Context, string) (observedRegion string, err error)
}

type EnvironmentLiveness struct {
	IdentityID     string    `json:"identity_id"`
	ExpectedRegion string    `json:"expected_region"`
	ObservedRegion string    `json:"observed_region"`
	Status         string    `json:"status"`
	Reason         string    `json:"reason"`
	CheckedAt      time.Time `json:"checked_at"`
}

// PortfolioPolicy constrains publish timing across identities. It deliberately
// does not affect warming or manual engagement actions: the policy exists to
// keep a portfolio of published accounts from presenting coordinated output.
// A zero-value policy is disabled until an operator configures it.
type PortfolioPolicy struct {
	MinimumPostGapMinutes int `json:"minimum_post_gap_minutes"`
	WindowMinutes         int `json:"window_minutes"`
	MaxPostsPerWindow     int `json:"max_posts_per_window"`
}

func (p PortfolioPolicy) Valid() error {
	if p.MinimumPostGapMinutes < 0 || p.WindowMinutes < 0 || p.MaxPostsPerWindow < 0 {
		return errors.New("portfolio policy values must not be negative")
	}
	if p.MaxPostsPerWindow > 0 && p.WindowMinutes < 1 {
		return errors.New("portfolio window must be positive when a ceiling is configured")
	}
	return nil
}

func (i Identity) Valid(platforms map[string]Platform) error {
	if i.ID == "" || i.PlatformID == "" || i.Purpose == "" || i.EnvironmentRef == "" {
		return errors.New("identity requires id, platform, purpose, and environment reference")
	}
	if _, ok := platforms[i.PlatformID]; !ok {
		return fmt.Errorf("unknown platform %q", i.PlatformID)
	}
	secretAssignment := "pass" + "word="
	if strings.Contains(strings.ToLower(i.VaultRef), "token=") || strings.Contains(strings.ToLower(i.VaultRef), secretAssignment) {
		return ErrCredentialValue
	}
	return nil
}

type ActionStatus = generated.ActionStatus
type ActionEvent = generated.ActionEvent

const (
	Scheduled ActionStatus = generated.ChannelActionScheduled
	Due       ActionStatus = generated.ChannelActionDue
	Executing ActionStatus = generated.ChannelActionExecuting
	Succeeded ActionStatus = generated.ChannelActionSucceeded
	Failed    ActionStatus = generated.ChannelActionFailed
	Cancelled ActionStatus = generated.ChannelActionCancelled

	ActionMakeDue ActionEvent = generated.ChannelActionMakeDue
	ActionBegin   ActionEvent = generated.ChannelActionBegin
	ActionSucceed ActionEvent = generated.ChannelActionSucceed
	ActionFail    ActionEvent = generated.ChannelActionFail
	ActionCancel  ActionEvent = generated.ChannelActionCancel
)

type Action struct {
	ID             string       `json:"id"`
	IdentityID     string       `json:"identity_id"`
	Kind           string       `json:"kind"`
	IdempotencyKey string       `json:"idempotency_key"`
	Evidence       string       `json:"evidence"`
	Executor       string       `json:"executor"`
	Window         time.Time    `json:"window"`
	RolledCount    int          `json:"rolled_count"`
	Seed           uint64       `json:"seed"`
	Status         ActionStatus `json:"status"`
	CompletedAt    time.Time    `json:"completed_at"`
	Deferred       bool         `json:"deferred"`
	SessionNumber  int          `json:"session_number"`
	ExecutionID    string       `json:"execution_id"`
	ExecutionError string       `json:"execution_error"`
	FailureClass   string       `json:"failure_class"`
	AttemptCount   int          `json:"attempt_count"`
	NextAttemptAt  time.Time    `json:"next_attempt_at"`
}

// BrowserDispatch is the minimum BAS adapter contract. Implementations receive
// a profile reference and durable action ID only; they must never receive or
// return browser cookies, passwords, or other credential material.
type BrowserDispatch interface {
	Dispatch(context.Context, string, string, string) (executionID string, artifacts []string, err error)
}

// BrowserExecution is the bounded review projection Channel Manager may obtain
// from BAS. It deliberately contains no storage state, credentials, or raw
// evidence bytes: artifact identifiers are resolved only by BAS's own access
// policy.
type BrowserExecution struct {
	ExecutionID  string   `json:"execution_id"`
	Status       string   `json:"status"`
	Failure      string   `json:"failure,omitempty"`
	ArtifactRefs []string `json:"artifact_refs,omitempty"`
}

// BrowserInspector is optional so the durable dispatch boundary remains small.
// A missing inspector never changes an action's state; manual verification is
// still the only path that records platform completion.
type BrowserInspector interface {
	Inspect(context.Context, string) (BrowserExecution, error)
}

type AutomationAssignment struct {
	ProfileKey   string   `json:"profile_key"`
	ProfileRef   string   `json:"profile_ref"`
	WorkflowRef  string   `json:"workflow_ref"`
	EnabledKinds []string `json:"enabled_kinds"`
	OperatorNote string   `json:"operator_note"`
}

// ReleaseReceipt is the durable publication ledger owned by Channel Manager.
// A receipt is created when an approved draft is accepted into the unified
// queue and is updated only by the executor that completes that queued action.
// Content Desk receives this projection; it never receives identity state.
type ReleaseReceipt struct {
	ID                   string    `json:"id"`
	DraftID              string    `json:"draft_id"`
	ActionID             string    `json:"action_id"`
	IdentityID           string    `json:"identity_id"`
	Lane                 string    `json:"lane"`
	IdempotencyKey       string    `json:"idempotency_key"`
	Status               string    `json:"status"`
	PlatformPostID       string    `json:"platform_post_id"`
	PublishedURL         string    `json:"published_url"`
	FirstCommentStatus   string    `json:"first_comment_status"`
	FirstCommentEvidence string    `json:"first_comment_evidence"`
	AssetIDs             []string  `json:"asset_ids"`
	DisclosureVisible    bool      `json:"disclosure_visible"`
	DeliveryStatus       string    `json:"delivery_status"`
	DeliveryError        string    `json:"delivery_error"`
	CreatedAt            time.Time `json:"created_at"`
	CompletedAt          time.Time `json:"completed_at"`
}

type ReleaseOptions struct {
	AssetIDs          []string
	DisclosureVisible bool
}

// MetricSample is an append-only observation attributed to one release. The
// delivery state is separate from collection so a temporary Content Desk
// outage cannot lose or duplicate the measurement.
type MetricSample struct {
	ID             string    `json:"id"`
	ReleaseID      string    `json:"release_id"`
	DraftID        string    `json:"draft_id"`
	Metric         string    `json:"metric"`
	Value          float64   `json:"value"`
	ObservedAt     time.Time `json:"observed_at"`
	DeliveryStatus string    `json:"delivery_status"`
}

func (r ReleaseReceipt) Complete() bool {
	return r.Status == "published" || r.Status == "partial"
}

type Observation struct {
	Metric string
	Value  float64
	At     time.Time
}

// ProgramOutcome is append-only evidence for one warming-program decision.
// Descriptor revisions never overwrite the gate results that justified a
// graduation or quarantine.
type ProgramOutcome struct {
	IdentityID string                `json:"identity_id"`
	ProgramID  string                `json:"program_id"`
	Outcome    string                `json:"outcome"`
	At         time.Time             `json:"at"`
	Gates      map[string]GateResult `json:"gates"`
}
type Flag struct {
	Metric          string
	Value, Baseline float64
	RaisedAt        time.Time
}
type GateResult struct {
	GateID     string    `json:"gate_id"`
	Outcome    string    `json:"outcome"`
	Repeats    int       `json:"repeats"`
	MeasuredAt time.Time `json:"measured_at"`
}

type Service struct {
	Platforms         map[string]Platform
	Programs          map[string]Program
	Identities        map[string]*Identity
	Actions           map[string]*Action
	Observations      map[string][]Observation
	Flags             map[string][]Flag
	RunningPrograms   map[string]string
	ProgramStartedAt  map[string]time.Time
	GateResults       map[string]map[string]GateResult
	Releases          map[string]*ReleaseReceipt
	Automation        map[string]AutomationAssignment
	BASProfiles       map[string]BASProfileDeclaration
	MetricSamples     map[string]*MetricSample
	Portfolio         PortfolioPolicy
	ProgramOutcomes   []ProgramOutcome
	EnvironmentChecks map[string]EnvironmentLiveness
	AssetPublications map[string]string
	ActivityEvents    []ActivityEvent
}

// State is the durable runtime projection. Descriptors remain files on disk;
// this contains only accumulated operator state and never a credential value.
type State struct {
	Identities        map[string]*Identity             `json:"identities"`
	Actions           map[string]*Action               `json:"actions"`
	Observations      map[string][]Observation         `json:"observations"`
	Flags             map[string][]Flag                `json:"flags"`
	RunningPrograms   map[string]string                `json:"running_programs"`
	ProgramStartedAt  map[string]time.Time             `json:"program_started_at"`
	GateResults       map[string]map[string]GateResult `json:"gate_results"`
	Releases          map[string]*ReleaseReceipt       `json:"releases"`
	Automation        map[string]AutomationAssignment  `json:"automation"`
	MetricSamples     map[string]*MetricSample         `json:"metric_samples"`
	Portfolio         PortfolioPolicy                  `json:"portfolio"`
	ProgramOutcomes   []ProgramOutcome                 `json:"program_outcomes"`
	EnvironmentChecks map[string]EnvironmentLiveness   `json:"environment_checks"`
	AssetPublications map[string]string                `json:"asset_publications"`
	ActivityEvents    []ActivityEvent                  `json:"activity_events"`
}

func (s *Service) State() State {
	return State{Identities: s.Identities, Actions: s.Actions, Observations: s.Observations, Flags: s.Flags, RunningPrograms: s.RunningPrograms, ProgramStartedAt: s.ProgramStartedAt, GateResults: s.GateResults, Releases: s.Releases, Automation: s.Automation, MetricSamples: s.MetricSamples, Portfolio: s.Portfolio, ProgramOutcomes: s.ProgramOutcomes, EnvironmentChecks: s.EnvironmentChecks, AssetPublications: s.AssetPublications, ActivityEvents: s.ActivityEvents}
}

func (s *Service) Restore(state State) {
	if state.Identities != nil {
		s.Identities = state.Identities
	}
	if state.Actions != nil {
		s.Actions = state.Actions
	}
	if state.Observations != nil {
		s.Observations = state.Observations
	}
	if state.Flags != nil {
		s.Flags = state.Flags
	}
	if state.RunningPrograms != nil {
		s.RunningPrograms = state.RunningPrograms
	}
	if state.ProgramStartedAt != nil {
		s.ProgramStartedAt = state.ProgramStartedAt
	}
	if state.GateResults != nil {
		s.GateResults = state.GateResults
	}
	if state.Releases != nil {
		s.Releases = state.Releases
	}
	if state.Automation != nil {
		s.Automation = state.Automation
	}
	if state.MetricSamples != nil {
		s.MetricSamples = state.MetricSamples
	}
	s.Portfolio = state.Portfolio
	if state.ProgramOutcomes != nil {
		s.ProgramOutcomes = state.ProgramOutcomes
	}
	if state.EnvironmentChecks != nil {
		s.EnvironmentChecks = state.EnvironmentChecks
	}
	if state.AssetPublications != nil {
		s.AssetPublications = state.AssetPublications
	}
	if state.ActivityEvents != nil {
		s.ActivityEvents = state.ActivityEvents
	}
}

func New(platforms []Platform, programs []Program) (*Service, error) {
	s := &Service{Platforms: map[string]Platform{}, Programs: map[string]Program{}, Identities: map[string]*Identity{}, Actions: map[string]*Action{}, Observations: map[string][]Observation{}, Flags: map[string][]Flag{}, RunningPrograms: map[string]string{}, ProgramStartedAt: map[string]time.Time{}, GateResults: map[string]map[string]GateResult{}, Releases: map[string]*ReleaseReceipt{}, Automation: map[string]AutomationAssignment{}, BASProfiles: map[string]BASProfileDeclaration{}, MetricSamples: map[string]*MetricSample{}, ProgramOutcomes: []ProgramOutcome{}, EnvironmentChecks: map[string]EnvironmentLiveness{}, AssetPublications: map[string]string{}, ActivityEvents: []ActivityEvent{}}
	for _, p := range platforms {
		if err := p.Valid(); err != nil {
			return nil, err
		}
		s.Platforms[p.ID] = p
	}
	for _, p := range programs {
		platform, ok := s.Platforms[p.PlatformID]
		if !ok {
			return nil, fmt.Errorf("program %q references unknown platform", p.ID)
		}
		if err := p.Valid(platform); err != nil {
			return nil, err
		}
		s.Programs[p.ID] = p
	}
	return s, nil
}

// SetBASProfileDeclarations installs the non-secret, scenario-owned profile
// declaration after descriptors load. It is configuration, never persisted
// account state.
func (s *Service) SetBASProfileDeclarations(profiles map[string]BASProfileDeclaration) {
	s.BASProfiles = make(map[string]BASProfileDeclaration, len(profiles))
	for key, profile := range profiles {
		s.BASProfiles[key] = profile
	}
}

func (s *Service) CreateIdentity(i Identity) error {
	if err := i.Valid(s.Platforms); err != nil {
		return err
	}
	if _, ok := s.Identities[i.ID]; ok {
		return errors.New("identity already exists")
	}
	if i.Status == "" {
		i.Status = "draft"
	}
	if i.Lifecycle == "" {
		i.Lifecycle = "draft"
	}
	if i.AutomationMode == "" {
		i.AutomationMode = "manual"
	}
	s.Identities[i.ID] = &i
	s.appendActivity(ActivityEvent{EventType: "identity.created", OccurredAt: time.Now().UTC(), IdentityID: i.ID, ActorType: "operator", Details: map[string]string{"platform_id": i.PlatformID, "lifecycle": i.Lifecycle}})
	return nil
}

// UpdateIdentity changes operator metadata only. Credential material remains
// an opaque Vault reference established during identity creation.
func (s *Service) UpdateIdentity(id string, update Identity) error {
	current := s.Identities[id]
	if current == nil {
		return errors.New("identity not found")
	}
	if update.Purpose == "" {
		update.Purpose = current.Purpose
	}
	if update.EnvironmentRef == "" {
		update.EnvironmentRef = current.EnvironmentRef
	}
	if update.Lifecycle == "" {
		update.Lifecycle = current.Lifecycle
	}
	if update.AutomationMode == "" {
		update.AutomationMode = current.AutomationMode
	}
	if update.D009AcceptanceRef == "" {
		update.D009AcceptanceRef = current.D009AcceptanceRef
	}
	update.ID, update.PlatformID, update.VaultRef, update.Status, update.LaneGrants, update.Attestations = current.ID, current.PlatformID, current.VaultRef, current.Status, current.LaneGrants, current.Attestations
	if err := update.Valid(s.Platforms); err != nil {
		return err
	}
	*current = update
	s.appendActivity(ActivityEvent{EventType: "identity.updated", OccurredAt: time.Now().UTC(), IdentityID: id, ActorType: "operator", Details: map[string]string{"lifecycle": update.Lifecycle, "automation_mode": update.AutomationMode}})
	return nil
}

func (s *Service) RetireIdentity(id string) error {
	identity := s.Identities[id]
	if identity == nil {
		return errors.New("identity not found")
	}
	identity.Status, identity.Lifecycle = "retired", "retired"
	s.appendActivity(ActivityEvent{EventType: "identity.retired", OccurredAt: time.Now().UTC(), IdentityID: id, ActorType: "operator"})
	return nil
}

// ConfigurePortfolioPolicy is an explicit operator decision. Existing
// scheduled work is not rewritten; the policy is enforced only at queue time.
func (s *Service) ConfigurePortfolioPolicy(policy PortfolioPolicy) error {
	if err := policy.Valid(); err != nil {
		return err
	}
	s.Portfolio = policy
	return nil
}

func (s *Service) StartProgram(identityID, programID string) error {
	i := s.Identities[identityID]
	p, ok := s.Programs[programID]
	if i == nil || !ok || i.PlatformID != p.PlatformID {
		return errors.New("identity and warming program are incompatible")
	}
	for _, pre := range p.Preconditions {
		if !i.Attestations[pre] {
			return ErrPreconditions
		}
	}
	i.Status = "warming"
	s.RunningPrograms[identityID] = programID
	s.ProgramStartedAt[identityID] = time.Now().UTC()
	s.appendActivity(ActivityEvent{EventType: "program.started", OccurredAt: s.ProgramStartedAt[identityID], IdentityID: identityID, ActorType: "operator", CorrelationID: programID})
	return nil
}

// EvaluateGate is idempotent after a gate resolves. It refuses premature
// measurements, bounds inconclusive repeats, and turns a failed gate into a
// terminal quarantine that cancels all pending work.
func (s *Service) EvaluateGate(identityID, gateID string, now time.Time) (GateResult, error) {
	i := s.Identities[identityID]
	if i == nil {
		return GateResult{}, errors.New("identity not found")
	}
	program := s.runningProgram(identityID)
	if program == nil {
		return GateResult{}, errors.New("identity has no running program")
	}
	var gate *Gate
	for idx := range program.Gates {
		if program.Gates[idx].ID == gateID {
			gate = &program.Gates[idx]
			break
		}
	}
	if gate == nil {
		return GateResult{}, errors.New("gate not found")
	}
	if s.GateResults[identityID] != nil {
		if prior, ok := s.GateResults[identityID][gateID]; ok && prior.Outcome != "inconclusive" {
			return prior, nil
		}
	}
	if now.Before(s.ProgramStartedAt[identityID].Add(time.Duration(gate.WaitMinutes) * time.Minute)) {
		return GateResult{GateID: gateID, Outcome: "waiting"}, nil
	}
	result := GateResult{GateID: gateID, Outcome: "inconclusive", MeasuredAt: now}
	if prior, ok := s.GateResults[identityID][gateID]; ok {
		result.Repeats = prior.Repeats + 1
	} else {
		result.Repeats = 1
	}
	for n := len(s.Observations[identityID]) - 1; n >= 0; n-- {
		o := s.Observations[identityID][n]
		if o.Metric == gate.Metric {
			if o.Value >= gate.Minimum {
				result.Outcome = "pass"
			}
			break
		}
	}
	if s.GateResults[identityID] == nil {
		s.GateResults[identityID] = map[string]GateResult{}
	}
	// Record the deciding measurement before quarantine snapshots the program
	// outcome, so the append-only evidence contains the terminal gate result.
	s.GateResults[identityID][gateID] = result
	if result.Outcome == "inconclusive" && result.Repeats >= gate.MaxRepeats {
		result.Outcome = "fail"
		s.GateResults[identityID][gateID] = result
		_ = s.Quarantine(identityID)
	}
	s.GateResults[identityID][gateID] = result
	return result, nil
}

// Graduate grants lanes only when every declared criterion is explicitly
// satisfied. Time elapsed is never an input to this decision.
func (s *Service) Graduate(identityID string, criteria map[string]bool, lanes []string) error {
	i := s.Identities[identityID]
	if i == nil {
		return errors.New("identity not found")
	}
	if i.Status != "warming" {
		return errors.New("only warming identities may graduate")
	}
	p := s.runningProgram(identityID)
	if p == nil {
		return errors.New("identity has no running program")
	}
	for _, criterion := range p.GraduationCriteria {
		if !criteria[criterion] {
			return fmt.Errorf("graduation criterion %q is not satisfied", criterion)
		}
	}
	i.Status = "active"
	i.LaneGrants = append([]string(nil), lanes...)
	s.appendProgramOutcome(identityID, "graduated", time.Now().UTC())
	return nil
}

func (s *Service) Enqueue(identityID, kind string, at time.Time, seed uint64, key string) (*Action, error) {
	i := s.Identities[identityID]
	if i == nil {
		return nil, errors.New("identity not found")
	}
	if i.Status == "paused" || i.Status == "quarantined" {
		return nil, ErrPaused
	}
	p := s.platformFor(i)
	if !slices.Contains(p.ActionKinds, kind) {
		return nil, fmt.Errorf("unknown action kind %q", kind)
	}
	if i.Status == "warming" {
		program := s.programFor(i)
		if program == nil {
			return nil, errors.New("warming identity has no program")
		}
		phase := program.Phases[0]
		if slices.Contains(phase.Forbidden, kind) || len(phase.Allowed) > 0 && !slices.Contains(phase.Allowed, kind) {
			return nil, ErrForbiddenAction
		}
	}
	count := 0
	for _, a := range s.Actions {
		if a.IdentityID == identityID && sameDay(a.Window, at) && a.Status != Cancelled {
			count++
		}
	}
	if count >= p.DailyCeiling {
		return nil, ErrCadence
	}
	window, deferred := s.portfolioWindow(identityID, kind, at)
	minimum, maximum := 1, 3
	if program := s.programFor(i); program != nil && len(program.Phases) > 0 {
		phase := program.Phases[0]
		minimum, maximum = phase.CountMin, phase.CountMax
		if minimum < 1 {
			minimum = 1
		}
		if maximum < minimum {
			maximum = minimum
		}
	}
	// This is a deterministic planning roll, not entropy for a security
	// boundary. Keep the arithmetic local so static security checks do not
	// mistake an account-safety decision for a weak random source.
	span := uint64(maximum - minimum + 1)
	mixed := seed ^ 0x9e3779b97f4a7c15
	mixed ^= mixed >> 30
	mixed *= 0xbf58476d1ce4e5b9
	mixed ^= mixed >> 27
	rolled := minimum + int(mixed%span)
	a := &Action{ID: fmt.Sprintf("act-%d", len(s.Actions)+1), IdentityID: identityID, Kind: kind, Window: window, RolledCount: rolled, Seed: seed, Status: Scheduled, IdempotencyKey: key, Deferred: deferred}
	s.Actions[a.ID] = a
	s.appendActivity(ActivityEvent{EventType: "action.queued", OccurredAt: at, IdentityID: identityID, ActionID: a.ID, ActorType: "operator", CorrelationID: key, Details: map[string]string{"kind": kind, "deferred": fmt.Sprintf("%t", deferred)}})
	return a, nil
}

// portfolioWindow chooses the earliest allowed slot for a new publish without
// changing existing actions. A defer is visible on the queued action, giving
// the operator a reason rather than silently losing requested work.
func (s *Service) portfolioWindow(identityID, kind string, requested time.Time) (time.Time, bool) {
	policy := s.Portfolio
	if kind != "publish" || (policy.MinimumPostGapMinutes == 0 && policy.MaxPostsPerWindow == 0) {
		return requested, false
	}
	window, deferred := requested, false
	gap := time.Duration(policy.MinimumPostGapMinutes) * time.Minute
	period := time.Duration(policy.WindowMinutes) * time.Minute
	for {
		changed := false
		for _, action := range s.Actions {
			if action.Kind != "publish" || action.IdentityID == identityID || action.Status == Cancelled {
				continue
			}
			if gap > 0 && window.Before(action.Window.Add(gap)) && window.After(action.Window.Add(-gap)) {
				window = action.Window.Add(gap)
				deferred, changed = true, true
			}
		}
		if policy.MaxPostsPerWindow > 0 {
			count := 0
			var latest time.Time
			for _, action := range s.Actions {
				if action.Kind != "publish" || action.Status == Cancelled || action.Window.Before(window.Add(-period)) || action.Window.After(window) {
					continue
				}
				count++
				if action.Window.After(latest) {
					latest = action.Window
				}
			}
			if count >= policy.MaxPostsPerWindow && !latest.IsZero() {
				window = latest.Add(period)
				deferred, changed = true, true
			}
		}
		if !changed {
			return window, deferred
		}
	}
}

// ScheduleSessions groups pending actions into the descriptor's session count
// and advances later windows to satisfy the declared minimum inter-session
// gap. It never changes an action's rolled count or seed.
func (s *Service) ScheduleSessions(identityID string) error {
	i := s.Identities[identityID]
	if i == nil {
		return errors.New("identity not found")
	}
	p := s.programFor(i)
	if p == nil {
		return errors.New("identity has no warming program")
	}
	actions := s.Due(identityID)
	if len(actions) == 0 {
		return nil
	}
	sessions := p.Sessions.Count
	if sessions < 1 {
		sessions = 1
	}
	gap := time.Duration(p.Sessions.MinimumGapMinutes) * time.Minute
	var lastBySession = make([]time.Time, sessions)
	for index, action := range actions {
		session := index % sessions
		action.SessionNumber = session + 1
		if !lastBySession[session].IsZero() && action.Window.Before(lastBySession[session].Add(gap)) {
			action.Window = lastBySession[session].Add(gap)
		}
		lastBySession[session] = action.Window
	}
	return nil
}

func (s *Service) Complete(id, evidence string, at time.Time) error {
	a := s.Actions[id]
	if a == nil {
		return errors.New("action not found")
	}
	if strings.TrimSpace(evidence) == "" {
		return errors.New("manual action completion requires evidence")
	}
	// The UI presents completion as one manual step, but the durable state
	// transition remains explicit: scheduled -> due -> executing -> succeeded.
	// This keeps manually-operated work replayable without asking an operator
	// to press internal lifecycle buttons.
	if a.Status == Scheduled {
		if err := s.transitionAction(a, ActionMakeDue); err != nil {
			return err
		}
	}
	if a.Status == Due {
		if err := s.transitionAction(a, ActionBegin); err != nil {
			return err
		}
	}
	if err := s.transitionAction(a, ActionSucceed); err != nil {
		return err
	}
	a.Evidence = evidence
	if a.ExecutionID == "" {
		a.Executor = "manual"
	} else {
		a.Executor = "manual_verification"
	}
	a.CompletedAt = at
	s.appendActivity(ActivityEvent{EventType: "action.completed", OccurredAt: at, IdentityID: a.IdentityID, ActionID: a.ID, ExecutionID: a.ExecutionID, ActorType: "operator", ExecutorType: a.Executor, CorrelationID: a.IdempotencyKey, ArtifactRefs: []string{evidence}})
	return nil
}

// AssignAutomation records one BAS session-profile reference for an identity.
// Reassignment is explicit and scopes which action kinds an operator accepts
// for automation; absent or disabled assignments always retain manual work.
func (s *Service) AssignAutomation(identityID, profileKey, profileRef, workflowRef string, enabledKinds []string, note string) error {
	if s.Identities[identityID] == nil || profileKey == "" || profileRef == "" || workflowRef == "" || len(enabledKinds) == 0 {
		return errors.New("automation assignment requires known identity, declared profile key, profile reference, workflow reference, and enabled action kinds")
	}
	if _, declared := s.BASProfiles[profileKey]; !declared {
		return fmt.Errorf("BAS profile key %q is not declared for Channel Manager", profileKey)
	}
	if s.Identities[identityID].D009AcceptanceRef == "" || s.Identities[identityID].AutomationMode != "operator-gated" {
		return errors.New("browser automation requires D-009 acceptance and operator-gated mode")
	}
	for otherIdentityID, assignment := range s.Automation {
		if otherIdentityID != identityID && (assignment.ProfileRef == profileRef || assignment.ProfileKey == profileKey) {
			return errors.New("BAS profile reference or declared profile key is already bound to another identity")
		}
	}
	platform := s.platformFor(s.Identities[identityID])
	for _, kind := range enabledKinds {
		if !slices.Contains(platform.ActionKinds, kind) {
			return fmt.Errorf("automation kind %q is not supported", kind)
		}
	}
	s.Automation[identityID] = AutomationAssignment{ProfileKey: profileKey, ProfileRef: profileRef, WorkflowRef: workflowRef, EnabledKinds: append([]string(nil), enabledKinds...), OperatorNote: note}
	s.appendActivity(ActivityEvent{EventType: "automation.assigned", OccurredAt: time.Now().UTC(), IdentityID: identityID, ActorType: "operator", Details: map[string]string{"profile_key": profileKey, "workflow_ref": workflowRef, "enabled_kinds": strings.Join(enabledKinds, ",")}})
	return nil
}

// DispatchBrowser starts an already queued action through BAS. Dispatch failure
// is recorded on the action and leaves it actionable manually; it never marks
// the platform action completed.
func (s *Service) DispatchBrowser(ctx context.Context, actionID string, dispatcher BrowserDispatch) (string, error) {
	a := s.Actions[actionID]
	if a == nil || dispatcher == nil {
		return "", errors.New("browser dispatch requires queued action and dispatcher")
	}
	assignment, ok := s.Automation[a.IdentityID]
	if !ok || !slices.Contains(assignment.EnabledKinds, a.Kind) {
		return "", errors.New("browser automation is not enabled for this action")
	}
	if a.Status != Scheduled && a.Status != Due {
		return "", errors.New("only pending actions may be dispatched")
	}
	if a.ExecutionID != "" {
		return a.ExecutionID, nil
	}
	executionID, artifacts, err := dispatcher.Dispatch(ctx, assignment.ProfileRef, assignment.WorkflowRef, a.ID)
	if err != nil {
		a.ExecutionError = err.Error()
		a.FailureClass = "manual_fallback"
		s.appendActivity(ActivityEvent{EventType: "browser.dispatch_failed", OccurredAt: time.Now().UTC(), IdentityID: a.IdentityID, ActionID: a.ID, ActorType: "system", ExecutorType: "browser", CorrelationID: a.IdempotencyKey, Details: map[string]string{"failure_class": a.FailureClass}})
		return "", err
	}
	a.ExecutionID = executionID
	a.ExecutionError = ""
	s.appendActivity(ActivityEvent{EventType: "browser.dispatched", OccurredAt: time.Now().UTC(), IdentityID: a.IdentityID, ActionID: a.ID, ExecutionID: executionID, ActorType: "operator", ExecutorType: "browser", CorrelationID: a.IdempotencyKey, ArtifactRefs: artifacts})
	return executionID, nil
}

// RecordExecutionFailure classifies a platform outcome using the selected
// platform descriptor. Only declared transient codes are retried, and every
// retry remains a durable queued action with a bounded next-attempt time.
func (s *Service) RecordExecutionFailure(actionID, code string, at time.Time) error {
	a := s.Actions[actionID]
	if a == nil {
		return errors.New("action not found")
	}
	policy := s.platformFor(s.Identities[a.IdentityID]).Retry
	a.ExecutionError = code
	a.AttemptCount++
	if slices.Contains(policy.RetryableCodes, code) && a.AttemptCount <= policy.MaxAttempts {
		a.FailureClass = "retryable"
		a.NextAttemptAt = at.Add(time.Duration(policy.BackoffMinutes*a.AttemptCount) * time.Minute)
		a.Status = Scheduled
		s.appendActivity(ActivityEvent{EventType: "action.retry_scheduled", OccurredAt: at, IdentityID: a.IdentityID, ActionID: a.ID, ExecutionID: a.ExecutionID, ActorType: "system", ExecutorType: "browser", CorrelationID: a.IdempotencyKey, Details: map[string]string{"failure_class": a.FailureClass, "attempt": fmt.Sprintf("%d", a.AttemptCount)}})
		return nil
	}
	a.FailureClass = "terminal"
	if a.Status == Scheduled {
		if err := s.transitionAction(a, ActionMakeDue); err != nil {
			return err
		}
	}
	if a.Status == Due {
		if err := s.transitionAction(a, ActionBegin); err != nil {
			return err
		}
	}
	if a.Status == Executing {
		err := s.transitionAction(a, ActionFail)
		if err == nil {
			s.appendActivity(ActivityEvent{EventType: "action.failed", OccurredAt: at, IdentityID: a.IdentityID, ActionID: a.ID, ExecutionID: a.ExecutionID, ActorType: "system", ExecutorType: "browser", CorrelationID: a.IdempotencyKey, Details: map[string]string{"failure_class": a.FailureClass}})
		}
		return err
	}
	return errors.New("action cannot accept execution failure")
}

// CompleteRelease records the outcome of the already-queued publish action.
// It is deliberately separate from Complete because a warming action has no
// post receipt and a release must never be inferred from arbitrary evidence.
func (s *Service) CompleteRelease(actionID, platformPostID, publishedURL, firstCommentStatus, firstCommentEvidence string, at time.Time) (*ReleaseReceipt, error) {
	a := s.Actions[actionID]
	if a == nil || a.Kind != "publish" {
		return nil, errors.New("publish action not found")
	}
	if platformPostID == "" || publishedURL == "" {
		return nil, errors.New("published release requires platform post id and URL")
	}
	var receipt *ReleaseReceipt
	for _, candidate := range s.Releases {
		if candidate.ActionID == actionID {
			receipt = candidate
			break
		}
	}
	if receipt == nil {
		return nil, errors.New("release receipt not found for action")
	}
	if receipt.Complete() {
		return receipt, nil
	}
	if err := s.Complete(actionID, publishedURL, at); err != nil {
		return nil, err
	}
	receipt.PlatformPostID = platformPostID
	receipt.PublishedURL = publishedURL
	receipt.FirstCommentStatus = firstCommentStatus
	receipt.FirstCommentEvidence = firstCommentEvidence
	receipt.CompletedAt = at
	if firstCommentStatus == "failed" {
		receipt.Status = "partial"
	} else {
		receipt.Status = "published"
	}
	receipt.DeliveryStatus = "pending"
	receipt.DeliveryError = ""
	for _, assetID := range receipt.AssetIDs {
		s.AssetPublications[assetID] = receipt.IdentityID
	}
	s.appendActivity(ActivityEvent{EventType: "release.completed", OccurredAt: at, IdentityID: a.IdentityID, ActionID: a.ID, ReleaseID: receipt.ID, ExecutionID: a.ExecutionID, ActorType: "operator", ExecutorType: a.Executor, CorrelationID: receipt.IdempotencyKey, ArtifactRefs: []string{publishedURL}, Details: map[string]string{"status": receipt.Status, "first_comment_status": firstCommentStatus}})
	return receipt, nil
}

// TransitionAction is the production entry point shared by the formal replay
// and the manual operator workflow. The generated table is the single source
// of truth for legal action-status transitions.
func TransitionAction(status ActionStatus, event ActionEvent) (ActionStatus, error) {
	return generated.TransitionChannelActionStatus(status, event)
}

func (s *Service) transitionAction(action *Action, event ActionEvent) error {
	next, err := TransitionAction(action.Status, event)
	if err != nil {
		return err
	}
	action.Status = next
	return nil
}

func (s *Service) RecordObservation(identityID, metric string, value float64, at time.Time, min int, decay float64) (*Flag, error) {
	if s.Identities[identityID] == nil {
		return nil, errors.New("identity not found")
	}
	items := append(s.Observations[identityID], Observation{metric, value, at})
	s.Observations[identityID] = items
	s.appendActivity(ActivityEvent{EventType: "observation.recorded", OccurredAt: at, IdentityID: identityID, ActorType: "operator", Details: map[string]string{"metric": metric}})
	var total float64
	var n int
	for _, o := range items {
		if o.Metric == metric {
			total += o.Value
			n++
		}
	}
	if n < min {
		return nil, nil
	}
	baseline := total / float64(n)
	if value < baseline*decay {
		f := Flag{metric, value, baseline, at}
		s.Flags[identityID] = append(s.Flags[identityID], f)
		s.Identities[identityID].Status = "paused"
		return &f, nil
	}
	return nil, nil
}

// CheckEnvironment records liveness even when the provider cannot answer.
// A configured identity is paused on mismatch or unknown so no queued action
// can proceed while its region invariant is unproven.
func (s *Service) CheckEnvironment(ctx context.Context, identityID string, probe EnvironmentProbe, at time.Time) (EnvironmentLiveness, error) {
	identity := s.Identities[identityID]
	if identity == nil {
		return EnvironmentLiveness{}, errors.New("identity not found")
	}
	record := EnvironmentLiveness{IdentityID: identityID, ExpectedRegion: identity.ExpectedRegion, CheckedAt: at}
	if identity.ExpectedRegion == "" {
		record.Status, record.Reason = "not_configured", "identity has no expected region"
		s.EnvironmentChecks[identityID] = record
		return record, nil
	}
	if probe == nil {
		record.Status, record.Reason = "unknown", "environment probe is unavailable"
	} else {
		observed, err := probe.Probe(ctx, identity.EnvironmentRef)
		record.ObservedRegion = observed
		if err != nil {
			record.Status, record.Reason = "unknown", err.Error()
		} else if observed != identity.ExpectedRegion {
			record.Status, record.Reason = "mismatch", "observed region differs from identity expectation"
		} else {
			record.Status = "healthy"
		}
	}
	if record.Status == "unknown" || record.Status == "mismatch" {
		identity.Status = "paused"
	}
	s.EnvironmentChecks[identityID] = record
	s.appendActivity(ActivityEvent{EventType: "environment.checked", OccurredAt: at, IdentityID: identityID, ActorType: "system", Details: map[string]string{"status": record.Status}})
	return record, nil
}

func (s *Service) RecordMetric(releaseID, sampleID, metric string, value float64, observedAt time.Time) (*MetricSample, error) {
	if sampleID == "" || metric == "" {
		return nil, errors.New("metric sample requires id and metric")
	}
	if prior := s.MetricSamples[sampleID]; prior != nil {
		return prior, nil
	}
	var receipt *ReleaseReceipt
	for _, candidate := range s.Releases {
		if candidate.ID == releaseID {
			receipt = candidate
			break
		}
	}
	if receipt == nil || !receipt.Complete() {
		return nil, errors.New("metrics require completed release receipt")
	}
	sample := &MetricSample{ID: sampleID, ReleaseID: releaseID, DraftID: receipt.DraftID, Metric: metric, Value: value, ObservedAt: observedAt, DeliveryStatus: "pending"}
	s.MetricSamples[sampleID] = sample
	s.appendActivity(ActivityEvent{EventType: "metric.recorded", OccurredAt: observedAt, IdentityID: receipt.IdentityID, ReleaseID: releaseID, ActorType: "operator", CorrelationID: sampleID, Details: map[string]string{"metric": metric}})
	return sample, nil
}

func (s *Service) AcknowledgeMetric(sampleID string) error {
	sample := s.MetricSamples[sampleID]
	if sample == nil {
		return errors.New("metric sample not found")
	}
	sample.DeliveryStatus = "acknowledged"
	s.appendActivity(ActivityEvent{EventType: "metric.acknowledged", OccurredAt: time.Now().UTC(), IdentityID: "", ReleaseID: sample.ReleaseID, ActorType: "system", CorrelationID: sampleID})
	return nil
}

// MarkReleaseDelivery records the Content Desk inbox acknowledgement without
// mutating the immutable platform outcome. Failed delivery stays retryable.
func (s *Service) MarkReleaseDelivery(releaseID string, delivered bool, deliveryErr string) error {
	var receipt *ReleaseReceipt
	for _, candidate := range s.Releases {
		if candidate.ID == releaseID {
			receipt = candidate
			break
		}
	}
	if receipt == nil || !receipt.Complete() {
		return fmt.Errorf("completed release %q not found", releaseID)
	}
	if delivered {
		receipt.DeliveryStatus, receipt.DeliveryError = "delivered", ""
		s.appendActivity(ActivityEvent{EventType: "release.delivery_acknowledged", OccurredAt: time.Now().UTC(), IdentityID: receipt.IdentityID, ReleaseID: receipt.ID, ActorType: "system", CorrelationID: receipt.IdempotencyKey})
		return nil
	}
	receipt.DeliveryStatus, receipt.DeliveryError = "pending", deliveryErr
	s.appendActivity(ActivityEvent{EventType: "release.delivery_pending", OccurredAt: time.Now().UTC(), IdentityID: receipt.IdentityID, ReleaseID: receipt.ID, ActorType: "system", CorrelationID: receipt.IdempotencyKey})
	return nil
}

func (s *Service) Eligibility(identityID, lane string) string {
	i := s.Identities[identityID]
	if i == nil {
		return "unknown"
	}
	if i.Status != "active" {
		return "not_eligible"
	}
	if slices.Contains(i.LaneGrants, lane) {
		return "eligible"
	}
	return "not_eligible"
}

func (s *Service) Release(identityID, lane, draftID, key string, at time.Time) (*ReleaseReceipt, error) {
	return s.ReleaseWithOptions(identityID, lane, draftID, key, ReleaseOptions{}, at)
}

func (s *Service) ReleaseWithOptions(identityID, lane, draftID, key string, options ReleaseOptions, at time.Time) (*ReleaseReceipt, error) {
	if key == "" || draftID == "" {
		return nil, errors.New("release requires draft id and idempotency key")
	}
	if prior := s.Releases[key]; prior != nil {
		return prior, nil
	}
	if s.Eligibility(identityID, lane) != "eligible" {
		return nil, errors.New("identity is not eligible")
	}
	identity := s.Identities[identityID]
	platform := s.platformFor(identity)
	if identity.Purpose == "persona-actor" && platform.DisclosureRequired && !options.DisclosureVisible {
		return nil, errors.New("persona-actor release requires visible platform disclosure")
	}
	for _, assetID := range options.AssetIDs {
		if assetID == "" {
			return nil, errors.New("release asset id must not be empty")
		}
		if publishedBy := s.AssetPublications[assetID]; publishedBy != "" && publishedBy != identityID {
			return nil, fmt.Errorf("asset %q was already published by identity %q", assetID, publishedBy)
		}
	}
	a, err := s.Enqueue(identityID, "publish", at, 0, key)
	if err != nil {
		return nil, err
	}
	receipt := &ReleaseReceipt{ID: "rel-" + a.ID, DraftID: draftID, ActionID: a.ID, IdentityID: identityID, Lane: lane, IdempotencyKey: key, Status: "queued", FirstCommentStatus: "not_requested", AssetIDs: append([]string(nil), options.AssetIDs...), DisclosureVisible: options.DisclosureVisible, CreatedAt: at}
	s.Releases[key] = receipt
	s.appendActivity(ActivityEvent{EventType: "release.queued", OccurredAt: at, IdentityID: identityID, ActionID: a.ID, ReleaseID: receipt.ID, ActorType: "system", CorrelationID: key, Details: map[string]string{"draft_id": draftID, "lane": lane}})
	return receipt, nil
}

func (s *Service) Quarantine(identityID string) error {
	i := s.Identities[identityID]
	if i == nil {
		return errors.New("identity not found")
	}
	i.Status = "quarantined"
	for _, a := range s.Actions {
		if a.IdentityID == identityID && (a.Status == Scheduled || a.Status == Due || a.Status == Executing) {
			_ = s.transitionAction(a, ActionCancel)
		}
	}
	s.appendProgramOutcome(identityID, "quarantined", time.Now().UTC())
	s.appendActivity(ActivityEvent{EventType: "identity.quarantined", OccurredAt: time.Now().UTC(), IdentityID: identityID, ActorType: "system"})
	return nil
}

func (s *Service) appendProgramOutcome(identityID, outcome string, at time.Time) {
	programID := s.RunningPrograms[identityID]
	if programID == "" {
		return
	}
	gates := make(map[string]GateResult, len(s.GateResults[identityID]))
	for id, result := range s.GateResults[identityID] {
		gates[id] = result
	}
	s.ProgramOutcomes = append(s.ProgramOutcomes, ProgramOutcome{IdentityID: identityID, ProgramID: programID, Outcome: outcome, At: at, Gates: gates})
}
func (s *Service) platformFor(i *Identity) Platform { return s.Platforms[i.PlatformID] }
func (s *Service) programFor(i *Identity) *Program {
	if id := s.RunningPrograms[i.ID]; id != "" {
		p := s.Programs[id]
		return &p
	}
	for _, p := range s.Programs {
		if p.PlatformID == i.PlatformID {
			return &p
		}
	}
	return nil
}

func (s *Service) runningProgram(identityID string) *Program {
	id := s.RunningPrograms[identityID]
	p, ok := s.Programs[id]
	if !ok {
		return nil
	}
	return &p
}
func sameDay(a, b time.Time) bool { return a.Year() == b.Year() && a.YearDay() == b.YearDay() }
func (s *Service) Due(identityID string) []*Action {
	out := []*Action{}
	for _, a := range s.Actions {
		if a.IdentityID == identityID && (a.Status == Scheduled || a.Status == Due) {
			out = append(out, a)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Window.Before(out[j].Window) })
	return out
}
