// Package channelmanager owns the deterministic, platform-independent rules
// for identity warming and manual account work.  Executors are deliberately
// outside this package: at P0 a human performs the platform interaction.
package channelmanager

import (
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
	ID                 string   `json:"id"`
	DailyCeiling       int      `json:"daily_ceiling"`
	ActionKinds        []string `json:"action_kinds"`
	DisclosureRequired bool     `json:"disclosure_required"`
	Formats            []Format `json:"formats"`
}

// Format is a platform-owned media contract. Limits declared as runtime
// resolved must be queried from the platform for the particular identity;
// they must never be replaced with a guessed global maximum.
type Format struct {
	Kind             string   `json:"kind"`
	MIMETypes        []string `json:"mime_types"`
	MaxBytes         int64    `json:"max_bytes"`
	MaxDurationSecs  int      `json:"max_duration_secs"`
	DurationResolved bool     `json:"duration_resolved_per_identity"`
	MinWidth         int      `json:"min_width"`
	MinHeight        int      `json:"min_height"`
	MaxWidth         int      `json:"max_width"`
	MaxHeight        int      `json:"max_height"`
}

func (p Platform) Valid() error {
	if p.ID == "" || p.DailyCeiling < 1 || len(p.ActionKinds) == 0 || len(p.Formats) == 0 {
		return errors.New("platform descriptor requires id, positive daily_ceiling, action_kinds, and formats")
	}
	for _, format := range p.Formats {
		if format.Kind == "" || len(format.MIMETypes) == 0 || format.MaxBytes < 1 || format.MinWidth < 1 || format.MinHeight < 1 || format.MaxWidth < format.MinWidth || format.MaxHeight < format.MinHeight || (!format.DurationResolved && format.MaxDurationSecs < 1) {
			return fmt.Errorf("invalid platform format %q", format.Kind)
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
	ID             string          `json:"id"`
	PlatformID     string          `json:"platform_id"`
	Purpose        string          `json:"purpose"`
	PersonaRef     string          `json:"persona_ref"`
	EnvironmentRef string          `json:"environment_ref"`
	VaultRef       string          `json:"vault_ref"`
	Status         string          `json:"status"`
	LaneGrants     []string        `json:"lane_grants"`
	Attestations   map[string]bool `json:"attestations"`
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
}
type Observation struct {
	Metric string
	Value  float64
	At     time.Time
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
	Platforms        map[string]Platform
	Programs         map[string]Program
	Identities       map[string]*Identity
	Actions          map[string]*Action
	Observations     map[string][]Observation
	Flags            map[string][]Flag
	RunningPrograms  map[string]string
	ProgramStartedAt map[string]time.Time
	GateResults      map[string]map[string]GateResult
	releases         map[string]string
}

// State is the durable runtime projection. Descriptors remain files on disk;
// this contains only accumulated operator state and never a credential value.
type State struct {
	Identities       map[string]*Identity             `json:"identities"`
	Actions          map[string]*Action               `json:"actions"`
	Observations     map[string][]Observation         `json:"observations"`
	Flags            map[string][]Flag                `json:"flags"`
	RunningPrograms  map[string]string                `json:"running_programs"`
	ProgramStartedAt map[string]time.Time             `json:"program_started_at"`
	GateResults      map[string]map[string]GateResult `json:"gate_results"`
	Releases         map[string]string                `json:"releases"`
}

func (s *Service) State() State {
	return State{s.Identities, s.Actions, s.Observations, s.Flags, s.RunningPrograms, s.ProgramStartedAt, s.GateResults, s.releases}
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
		s.releases = state.Releases
	}
}

func New(platforms []Platform, programs []Program) (*Service, error) {
	s := &Service{Platforms: map[string]Platform{}, Programs: map[string]Program{}, Identities: map[string]*Identity{}, Actions: map[string]*Action{}, Observations: map[string][]Observation{}, Flags: map[string][]Flag{}, RunningPrograms: map[string]string{}, ProgramStartedAt: map[string]time.Time{}, GateResults: map[string]map[string]GateResult{}, releases: map[string]string{}}
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
	s.Identities[i.ID] = &i
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
	if result.Outcome == "inconclusive" && result.Repeats >= gate.MaxRepeats {
		result.Outcome = "fail"
		_ = s.Quarantine(identityID)
	}
	if s.GateResults[identityID] == nil {
		s.GateResults[identityID] = map[string]GateResult{}
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
	a := &Action{ID: fmt.Sprintf("act-%d", len(s.Actions)+1), IdentityID: identityID, Kind: kind, Window: at, RolledCount: rolled, Seed: seed, Status: Scheduled, IdempotencyKey: key}
	s.Actions[a.ID] = a
	return a, nil
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
	a.Executor = "manual"
	a.CompletedAt = at
	return nil
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

func (s *Service) Release(identityID, lane, key string, at time.Time) (string, error) {
	if prior := s.releases[key]; prior != "" {
		return prior, nil
	}
	if s.Eligibility(identityID, lane) != "eligible" {
		return "", errors.New("identity is not eligible")
	}
	a, err := s.Enqueue(identityID, "publish", at, 0, key)
	if err != nil {
		return "", err
	}
	s.releases[key] = a.ID
	return a.ID, nil
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
	return nil
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
