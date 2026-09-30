// Package repairbudget owns cumulative, durable repair accounting for one
// finite effort. It enforces the three reviewed limits (per failure
// fingerprint, per owning component, and across the whole effort) plus the
// separate probe, transport-retry and status-read allowances.
//
// An intervention is charged once, at Begin, so an interrupted attempt still
// consumes cumulative investment and cannot be replayed for free. A new plan,
// worker, round or leader restart does not reset a total. Repeated reports of
// the same attempt identity are idempotent; conflicting reuse of an attempt
// identity is refused.
//
// Each repair carries a charge class. Declared planned infrastructure is
// delivery, not diversion: it is retained for visibility but does not consume
// the cumulative diversion limits. Unplanned dependency repair and repeated
// failed interventions are diversion and do consume them. A fingerprint's class
// is fixed at first charge, so renaming or reclassifying repeated repair cannot
// reset an open circuit.
package repairbudget

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"swarm-manager/internal/identity"
)

// Intervention kinds. Repair consumes the cumulative fingerprint/component/
// effort limits; the other kinds consume their own bounded allowance.
const (
	KindRepair         = "repair"
	KindProbe          = "probe"
	KindTransportRetry = "transport_retry"
	KindStatusRead     = "status_read"
)

// Charge classes. Diversion is the default because the ledger records
// interventions, which the reviewed policy treats as unplanned unless they are
// explicitly attributed to declared planned infrastructure. Only diversion
// consumes the cumulative fingerprint/component/effort limits; planned delivery
// is retained separately so it is never charged twice against the diversion
// bound.
const (
	ClassDiversion = "diversion"
	ClassPlanned   = "planned"
)

// Event lifecycle discriminators. Started reserves and charges one
// intervention before its effect; Finished records the terminal disposition.
// The dispatch events track the route owner for one reserved operation so a
// lost start response can be reconciled under the original identity instead of
// launching a duplicate executor.
const (
	EventStarted      = "started"
	EventFinished     = "finished"
	EventAcknowledged = "acknowledged"
	EventUncertain    = "dispatch_uncertain"
	EventReconciled   = "reconciled"
	EventFallback     = "fallback"
)

// Reviewed default allowances for the non-repair interventions when the effort
// policy leaves them unconfigured (zero). See the large-effort-orchestration
// recovery reference.
const (
	DefaultProbeAttempts     = 2
	DefaultTransportAttempts = 2
	DefaultStatusReads       = 2
)

// Sentinel errors. Callers map these to their own transport error shapes; the
// package stays free of HTTP concerns.
var (
	ErrBudgetExhausted  = errors.New("cumulative repair budget is exhausted")
	ErrUnknownAttempt   = errors.New("attempt has not been started")
	ErrIdentityConflict = errors.New("attempt identity is reused with different work")
	ErrStaleEvidence    = errors.New("repeat fingerprint without new evidence")
	// ErrReclassificationRefused reports an attempt to change an existing
	// fingerprint's charge class. Planned delivery and diversion repair have
	// separate allowances, so a class swap cannot be used to reopen a circuit
	// or to move an already-charged repair off the diversion bound.
	ErrReclassificationRefused = errors.New("repair fingerprint charge class cannot change")
	// ErrDispatchUncertain reports that a reserved operation's start response
	// was lost. Its replacement is refused until the original dispatch is
	// reconciled, so no second executor can run the same work.
	ErrDispatchUncertain = errors.New("dispatch is uncertain and must be reconciled before replacement")
	// ErrExecutorActive reports that an owner has already acknowledged this
	// operation. A second active executor is refused until the first is
	// reconciled terminal.
	ErrExecutorActive = errors.New("an executor is already active for this operation")
	// ErrAttemptTerminal reports that the operation already recorded a terminal
	// disposition. A late acknowledgement or fallback cannot revive it.
	ErrAttemptTerminal = errors.New("attempt is already terminal")
)

// Event is one immutable append to the repair ledger. Started events are the
// unit of charging; finished events record disposition but never re-charge.
type Event struct {
	AttemptID   string   `json:"attempt_id"`
	Fingerprint string   `json:"fingerprint"`
	Component   string   `json:"component"`
	Kind        string   `json:"kind"`
	Class       string   `json:"class,omitempty"`
	Event       string   `json:"event"`
	Hypothesis  string   `json:"hypothesis,omitempty"`
	NewEvidence bool     `json:"new_evidence,omitempty"`
	Evidence    []string `json:"evidence,omitempty"`
	OwnerWork   string   `json:"owner_work,omitempty"`
	OwnerRun    string   `json:"owner_run,omitempty"`
	Outcome     string   `json:"outcome,omitempty"`
	Reason      string   `json:"reason,omitempty"`
	// ObservedActive is set on a reconciled event: true means the owner still
	// holds an active executor for the operation, so replacement stays refused.
	ObservedActive bool   `json:"observed_active,omitempty"`
	At             string `json:"at"`
}

// Totals is the cumulative, effort-wide investment summary. ByFingerprint,
// ByComponent and Effort count diversion repair interventions only, because
// those are the charges the reviewed waste bound constrains. Planned counts
// declared planned-infrastructure repairs, which are retained for visibility
// but do not consume the diversion bound. ByClass counts every repair charge by
// class; ByKind counts every started kind so probe/transport/status usage stays
// visible.
//
// ComponentActiveMinutes and EffortActiveMinutes summarize the accrued diversion
// active time that the reviewed time bound constrains. An open (started,
// unfinished) attempt contributes its elapsed time on every read so an
// interrupted repair can never read as free.
type Totals struct {
	ByFingerprint          map[string]int     `json:"by_fingerprint"`
	ByComponent            map[string]int     `json:"by_component"`
	ByKind                 map[string]int     `json:"by_kind"`
	ByClass                map[string]int     `json:"by_class"`
	ComponentActiveMinutes map[string]float64 `json:"component_active_minutes,omitempty"`
	Effort                 int                `json:"effort"`
	Planned                int                `json:"planned"`
	EffortActiveMinutes    float64            `json:"effort_active_minutes,omitempty"`
}

// Admission reports the result of a Begin call.
type Admission struct {
	Charged         bool   `json:"charged"`
	AlreadyRecorded bool   `json:"already_recorded"`
	Totals          Totals `json:"totals"`
}

type state struct {
	EffortID string  `json:"effort_id"`
	Events   []Event `json:"events"`
}

// Ledger is the durable, single-writer repair accounting for one effort
// identity. Mutations are serialized in-process and persisted atomically.
type Ledger struct {
	effortID string
	path     string
	now      func() time.Time
	mu       sync.Mutex
}

// New roots a ledger for one effort identity at the given file path.
func New(effortID, path string) *Ledger {
	return &Ledger{effortID: strings.TrimSpace(effortID), path: path, now: time.Now}
}

// SetClock overrides the clock; tests use it to make timestamps deterministic.
func (l *Ledger) SetClock(now func() time.Time) { l.now = now }

// Begin records the start of an intervention and charges it against cumulative
// investment exactly once. It returns an error before any effect is admitted
// when the relevant limit is reached or the attempt identity conflicts.
func (l *Ledger) Begin(limits identity.RepairLimits, event Event) (Admission, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if err := normalize(&event, now); err != nil {
		return Admission{}, err
	}
	current, err := l.load()
	if err != nil {
		return Admission{}, err
	}

	for _, existing := range current.Events {
		if existing.AttemptID != event.AttemptID || existing.Event != EventStarted {
			continue
		}
		if existing.Fingerprint != event.Fingerprint || existing.Component != event.Component || existing.Kind != event.Kind || effectiveClass(existing.Class) != effectiveClass(event.Class) {
			return Admission{}, fmt.Errorf("%w: attempt %q", ErrIdentityConflict, event.AttemptID)
		}
		return Admission{Charged: false, AlreadyRecorded: true, Totals: totals(current, now)}, nil
	}

	currentTotals := totals(current, now)
	if err := admit(limits, event, current.Events, currentTotals); err != nil {
		return Admission{}, err
	}

	current.Events = append(current.Events, event)
	if err := l.save(current); err != nil {
		return Admission{}, err
	}
	return Admission{Charged: true, Totals: totals(current, now)}, nil
}

// Finish records the disposition of a started intervention. A repeated finish
// is idempotent and never charges twice. A finish for an unknown attempt is
// refused so a caller cannot close work it never reserved.
func (l *Ledger) Finish(attemptID, outcome string, evidence ...string) (Event, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	attemptID = strings.TrimSpace(attemptID)
	current, err := l.load()
	if err != nil {
		return Event{}, err
	}

	var started *Event
	for i := range current.Events {
		if current.Events[i].AttemptID != attemptID {
			continue
		}
		switch current.Events[i].Event {
		case EventStarted:
			started = &current.Events[i]
		case EventFinished:
			return current.Events[i], nil
		}
	}
	if started == nil {
		return Event{}, fmt.Errorf("%w: %q", ErrUnknownAttempt, attemptID)
	}

	finished := Event{
		AttemptID:   started.AttemptID,
		Fingerprint: started.Fingerprint,
		Component:   started.Component,
		Kind:        started.Kind,
		Class:       effectiveClass(started.Class),
		Event:       EventFinished,
		Outcome:     strings.TrimSpace(outcome),
		Evidence:    append([]string(nil), evidence...),
		At:          l.now().UTC().Format(time.RFC3339),
	}
	current.Events = append(current.Events, finished)
	if err := l.save(current); err != nil {
		return Event{}, err
	}
	return finished, nil
}

// Totals returns the cumulative investment summary for this effort.
func (l *Ledger) Totals() (Totals, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	current, err := l.load()
	if err != nil {
		return Totals{}, err
	}
	return totals(current, l.now()), nil
}

func normalize(event *Event, now time.Time) error {
	event.AttemptID = strings.TrimSpace(event.AttemptID)
	event.Fingerprint = strings.TrimSpace(event.Fingerprint)
	event.Component = strings.TrimSpace(event.Component)
	event.Kind = strings.TrimSpace(event.Kind)
	if event.AttemptID == "" {
		return fmt.Errorf("attempt_id is required")
	}
	if event.Fingerprint == "" {
		return fmt.Errorf("fingerprint is required")
	}
	if event.Component == "" {
		return fmt.Errorf("component is required")
	}
	switch event.Kind {
	case KindRepair, KindProbe, KindTransportRetry, KindStatusRead:
	default:
		return fmt.Errorf("unknown intervention kind %q", event.Kind)
	}
	if event.Class == "" {
		event.Class = ClassDiversion
	}
	switch event.Class {
	case ClassDiversion, ClassPlanned:
	default:
		return fmt.Errorf("unknown charge class %q", event.Class)
	}
	if event.Event == "" {
		event.Event = EventStarted
	}
	if event.Event != EventStarted {
		return fmt.Errorf("Begin only accepts started events")
	}
	if event.At == "" {
		event.At = now.UTC().Format(time.RFC3339)
	}
	return nil
}

func admit(limits identity.RepairLimits, event Event, events []Event, current Totals) error {
	switch event.Kind {
	case KindRepair:
		for _, existing := range events {
			if existing.Event != EventStarted || existing.Kind != KindRepair || existing.Fingerprint != event.Fingerprint {
				continue
			}
			if effectiveClass(existing.Class) != effectiveClass(event.Class) {
				return fmt.Errorf("%w: fingerprint %q", ErrReclassificationRefused, event.Fingerprint)
			}
		}
		if effectiveClass(event.Class) == ClassPlanned {
			// Declared planned infrastructure is delivery. It is retained for
			// visibility but does not consume the diversion waste bound, so
			// normal planned work is not charged twice.
			return nil
		}
		if !event.NewEvidence {
			for _, existing := range events {
				if existing.Event == EventStarted && existing.Kind == KindRepair && existing.Fingerprint == event.Fingerprint {
					return fmt.Errorf("%w: fingerprint %q", ErrStaleEvidence, event.Fingerprint)
				}
			}
		}
		if limits.PerFingerprint > 0 && current.ByFingerprint[event.Fingerprint]+1 > limits.PerFingerprint {
			return fmt.Errorf("%w: fingerprint %q at %d/%d", ErrBudgetExhausted, event.Fingerprint, current.ByFingerprint[event.Fingerprint], limits.PerFingerprint)
		}
		if limits.PerComponent > 0 && current.ByComponent[event.Component]+1 > limits.PerComponent {
			return fmt.Errorf("%w: component %q at %d/%d", ErrBudgetExhausted, event.Component, current.ByComponent[event.Component], limits.PerComponent)
		}
		if limits.PerEffort > 0 && current.Effort+1 > limits.PerEffort {
			return fmt.Errorf("%w: effort at %d/%d", ErrBudgetExhausted, current.Effort, limits.PerEffort)
		}
		// Active-time caps are complementary to the count caps and only bound
		// diversion: whichever threshold is reached first stops admission. Open
		// attempts contribute elapsed time so an interrupted repair still counts.
		if limits.ComponentActiveMinutes > 0 && current.ComponentActiveMinutes[event.Component] >= float64(limits.ComponentActiveMinutes) {
			return fmt.Errorf("%w: component %q at %.1f/%d active minutes", ErrBudgetExhausted, event.Component, current.ComponentActiveMinutes[event.Component], limits.ComponentActiveMinutes)
		}
		if bound := limits.EffortActiveMinutesLimit(); bound > 0 && current.EffortActiveMinutes >= float64(bound) {
			return fmt.Errorf("%w: effort at %.1f/%d diversion active minutes", ErrBudgetExhausted, current.EffortActiveMinutes, bound)
		}
	case KindProbe, KindTransportRetry, KindStatusRead:
		limit := kindLimit(limits, event.Kind)
		used := countKindForFingerprint(events, event.Kind, event.Fingerprint)
		if used+1 > limit {
			return fmt.Errorf("%w: %s %q at %d/%d", ErrBudgetExhausted, event.Kind, event.Fingerprint, used, limit)
		}
	}
	return nil
}

func kindLimit(limits identity.RepairLimits, kind string) int {
	switch kind {
	case KindProbe:
		if limits.ProbeAttempts > 0 {
			return limits.ProbeAttempts
		}
		return DefaultProbeAttempts
	case KindTransportRetry:
		if limits.TransportAttempts > 0 {
			return limits.TransportAttempts
		}
		return DefaultTransportAttempts
	case KindStatusRead:
		if limits.StatusReads > 0 {
			return limits.StatusReads
		}
		return DefaultStatusReads
	default:
		return 0
	}
}

func countKindForFingerprint(events []Event, kind, fingerprint string) int {
	count := 0
	for _, event := range events {
		if event.Event == EventStarted && event.Kind == kind && event.Fingerprint == fingerprint {
			count++
		}
	}
	return count
}

func totals(current state, now time.Time) Totals {
	result := Totals{
		ByFingerprint: map[string]int{},
		ByComponent:   map[string]int{},
		ByKind:        map[string]int{},
		ByClass:       map[string]int{},
	}
	for _, event := range current.Events {
		if event.Event != EventStarted {
			continue
		}
		result.ByKind[event.Kind]++
		if event.Kind != KindRepair {
			continue
		}
		class := effectiveClass(event.Class)
		result.ByClass[class]++
		if class == ClassPlanned {
			result.Planned++
			continue
		}
		result.Effort++
		result.ByFingerprint[event.Fingerprint]++
		result.ByComponent[event.Component]++
	}
	result.ComponentActiveMinutes, result.EffortActiveMinutes = activeRepairMinutes(current.Events, now)
	return result
}

// activeRepairMinutes accrues diversion repair active time by owning component
// and across the effort. A finished attempt contributes its recorded interval;
// an open (started, unfinished) attempt contributes the elapsed time to now so
// an interrupted repair cannot read as free. A malformed or reversed timestamp
// contributes nothing rather than an invented duration, and planned delivery is
// excluded because the time bound constrains diversion only.
func activeRepairMinutes(events []Event, now time.Time) (map[string]float64, float64) {
	finishedAt := map[string]time.Time{}
	for _, event := range events {
		if event.Event != EventFinished {
			continue
		}
		if at, err := time.Parse(time.RFC3339, event.At); err == nil {
			finishedAt[event.AttemptID] = at
		}
	}
	byComponent := map[string]float64{}
	var effort float64
	for _, event := range events {
		if event.Event != EventStarted || event.Kind != KindRepair || effectiveClass(event.Class) != ClassDiversion {
			continue
		}
		start, err := time.Parse(time.RFC3339, event.At)
		if err != nil {
			continue
		}
		end := now
		if at, ok := finishedAt[event.AttemptID]; ok {
			end = at
		}
		if end.Before(start) {
			continue
		}
		minutes := end.Sub(start).Minutes()
		byComponent[event.Component] += minutes
		effort += minutes
	}
	return byComponent, effort
}

// effectiveClass treats a pre-class durable event as diversion so a ledger
// written before charge classes existed keeps its cumulative charges and
// cannot silently reopen a closed circuit.
func effectiveClass(class string) string {
	if strings.TrimSpace(class) == ClassPlanned {
		return ClassPlanned
	}
	return ClassDiversion
}

func (l *Ledger) load() (state, error) {
	raw, err := os.ReadFile(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return state{EffortID: l.effortID}, nil
		}
		return state{}, fmt.Errorf("read repair ledger: %w", err)
	}
	var current state
	if err := json.Unmarshal(raw, &current); err != nil {
		return state{}, fmt.Errorf("decode repair ledger: %w", err)
	}
	if current.EffortID == "" {
		current.EffortID = l.effortID
	}
	if current.EffortID != l.effortID {
		return state{}, fmt.Errorf("repair ledger effort mismatch: ledger holds %q, want %q", current.EffortID, l.effortID)
	}
	return current, nil
}

func (l *Ledger) save(current state) error {
	current.EffortID = l.effortID
	if err := os.MkdirAll(filepath.Dir(l.path), 0o700); err != nil {
		return fmt.Errorf("create repair ledger dir: %w", err)
	}
	data, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal repair ledger: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(l.path), ".repair-ledger-*.tmp")
	if err != nil {
		return fmt.Errorf("create repair ledger temp: %w", err)
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
	if err := os.Rename(tempName, l.path); err != nil {
		os.Remove(tempName)
		return err
	}
	return nil
}
