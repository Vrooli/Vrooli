package usagemodes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/usageforecast"
	"agent-manager/internal/usagelabel"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
	"github.com/vrooli/cli-core/usagemode"
)

// DecisionEventType is the generic decision contract notification-hub
// subscribes to; the kind is carried in the correlation.
const DecisionEventType = "agent_manager.usage_mode.decision_requested.v1"

// Decision kinds (Correlation["kind"]).
const (
	KindRunsOut = "runs-out"
	KindEnding  = "ending"
	KindUntilOf = "until-off"
	// KindTimed asks to start a fixed-length mode on one runner (a requested ask, e.g. a validation).
	KindTimed = "timed-start"
)

// Answer keys.
const (
	AnswerStart  = "start"
	AnswerNotNow = "not_now"
	AnswerRemind = "remind"
	AnswerExtend = "extend"
	AnswerLetEnd = "let_end"
	AnswerKeepOn = "keep_on"
	AnswerTurnOf = "turn_off"
)

// Timings and thresholds.
const (
	RemindAfter       = 30 * time.Minute
	EndingLead        = 15 * time.Minute
	minEndingLifetime = 20 * time.Minute
	UntilOffEvery     = 24 * time.Hour
	// AskLengthStep is the granularity of a requested timed mode, and
	// MaxOutstandingTimedAsks the most undecided timed asks per runner.
	AskLengthStep           = 15 * time.Minute
	MaxOutstandingTimedAsks = 2
	staleEnded              = 6 * time.Hour
	reachMinAge             = 30 * time.Minute
	reachEvery              = time.Hour
	// UnreachedShare is the part of a mode's observed burn that must come from
	// work it cannot change before the operator is told.
	UnreachedShare = 0.4
	// UnreachedMinPoints keeps a tiny sample from raising the warning.
	UnreachedMinPoints = 2.0
	stateRetention     = 30 * 24 * time.Hour
	decisionSubject    = "usage-mode"
)

// NotificationCategoryQualityDrop is the usage-mode-quality-drop category
// declared in .vrooli/notifications.json, off until the operator turns it on.
const NotificationCategoryQualityDrop = "usage-mode-quality-drop"

const (
	qualityMinAge = time.Hour
	qualityEvery  = time.Hour
	// evidenceEvery bounds how often the evidence source is asked for results.
	evidenceEvery = time.Hour
)

// EvidenceFunc adapts a function to EvidenceSource.
type EvidenceFunc func(ctx context.Context, now time.Time) ([]EvidenceReport, error)

// Ready calls the function.
func (f EvidenceFunc) Ready(ctx context.Context, now time.Time) ([]EvidenceReport, error) {
	return f(ctx, now)
}

// SetEvidence connects the evidence source and turns the evidence-ready notice
// on. The notice category is off by default at the hub, so what reaches the
// operator is still the operator's choice.
func (n *Notifier) SetEvidence(source EvidenceSource) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.Evidence, n.EvidenceReady = source, source != nil
}

// QualityFinding is a rise in operator-reported failures and corrections under
// a mode against the operator's own prior period, on the same model.
type QualityFinding struct {
	Rate, Baseline                float64 // shares of classified operator messages
	DifferenceLow, DifferenceHigh float64 // 95% range of Rate minus Baseline; Low is above zero
	Messages, BaselineMessages    int64
	Model                         string
	ClassifierPrecision           float64 // the classifier's measured hold-out precision for the bucket
}

// QualitySource finds a proven rise in operator-reported failures and
// corrections under a mode. It returns false unless the signal passed its
// accuracy gate, the cell and the baseline are large enough, and the rise is
// outside chance. interventions.ModeQuality implements it.
type QualitySource interface {
	QualityDrop(ctx context.Context, mode usagemode.Mode, now time.Time) (QualityFinding, bool, error)
}

// ReachSource measures how much of a mode's observed burn came from work the
// mode cannot change (usage attribution implements it).
type ReachSource interface {
	Unreached(ctx context.Context, mode usagemode.Mode, now time.Time) (share, observedPoints float64, err error)
}

// EvidenceReport names one mode whose comparison now has enough finished
// outcomes to read, with the plain sentence the notification carries.
type EvidenceReport struct {
	ModeID string
	Label  string
	Line   string
}

// EvidenceSource reports the modes whose evidence view has a result.
type EvidenceSource interface {
	Ready(ctx context.Context, now time.Time) ([]EvidenceReport, error)
}

type windowOffer struct {
	ResetAt  time.Time `json:"reset_at"`
	Declined bool      `json:"declined,omitempty"`
	RemindAt time.Time `json:"remind_at,omitempty"`
	Gen      int       `json:"gen"`
	Runner   string    `json:"runner"`
}

type endedNotice struct {
	ID        string    `json:"id"`
	Label     string    `json:"label"`
	Providers []string  `json:"providers"`
	Line      string    `json:"line"`
	At        time.Time `json:"at"`
}

type notifierState struct {
	Asks       map[string]*operatordecision.PendingAsk `json:"asks"`
	Seen       map[string]usagemode.Mode               `json:"seen"`
	TapStarted map[string]bool                         `json:"tap_started"`
	Offers     map[string]*windowOffer                 `json:"offers"`
	Ended      []endedNotice                           `json:"ended"`
	Started    []usagemode.Mode                        `json:"started"`
	ReachDone  map[string]bool                         `json:"reach_done"`
	ReachAt    map[string]time.Time                    `json:"reach_at"`
	// QualityDone and QualityAt gate the quality-drop notice: once per mode,
	// checked at most hourly.
	QualityDone map[string]bool      `json:"quality_done"`
	QualityAt   map[string]time.Time `json:"quality_at"`
	// EvidenceDone holds the modes already told that their evidence is ready.
	EvidenceDone map[string]bool `json:"evidence_done"`
}

// Notifier owns the usage mode decisions and notices: the run-out offer, the
// ending-soon and until-off decisions, and the plain ended, started and
// can't-reach notices. It satisfies usageforecast.ModeLifecycle and
// operatordecision.DomainEffect. State lives beside the mode file.
type Notifier struct {
	Modes     *Service
	Publisher usageforecast.Publisher
	Reach     ReachSource
	// Quality raises the quality-drop notice. Nil leaves it off.
	Quality QualitySource
	// Evidence feeds the evidence-ready notice. It is sent only when
	// EvidenceReady is true and Evidence is set: off by default.
	Evidence      EvidenceSource
	EvidenceReady bool
	// evidenceAt is when the source was last asked: the analysis is not cheap.
	evidenceAt time.Time
	UsageURL   func(context.Context) string
	Now        func() time.Time
	// Path overrides the state file (tests); empty is notifier.json beside the mode file.
	Path string

	mu sync.Mutex
	st *notifierState
}

var (
	_ usageforecast.ModeLifecycle   = (*Notifier)(nil)
	_ operatordecision.DomainEffect = (*Notifier)(nil)
)

func (n *Notifier) now() time.Time {
	if n.Now != nil {
		return n.Now().UTC()
	}
	return time.Now().UTC()
}

func (n *Notifier) path() string {
	if n.Path != "" {
		return n.Path
	}
	return filepath.Join(filepath.Dir(n.Modes.path()), "notifier.json")
}

// load reads the state once; callers hold n.mu.
func (n *Notifier) load() *notifierState {
	if n.st != nil {
		return n.st
	}
	st := &notifierState{}
	if raw, err := os.ReadFile(n.path()); err == nil {
		_ = json.Unmarshal(raw, st)
	}
	if st.Asks == nil {
		st.Asks = map[string]*operatordecision.PendingAsk{}
	}
	if st.Seen == nil {
		st.Seen = map[string]usagemode.Mode{}
	}
	if st.TapStarted == nil {
		st.TapStarted = map[string]bool{}
	}
	if st.Offers == nil {
		st.Offers = map[string]*windowOffer{}
	}
	if st.ReachDone == nil {
		st.ReachDone = map[string]bool{}
	}
	if st.ReachAt == nil {
		st.ReachAt = map[string]time.Time{}
	}
	if st.QualityDone == nil {
		st.QualityDone = map[string]bool{}
	}
	if st.QualityAt == nil {
		st.QualityAt = map[string]time.Time{}
	}
	if st.EvidenceDone == nil {
		st.EvidenceDone = map[string]bool{}
	}
	n.st = st
	return st
}

func (n *Notifier) save() error {
	raw, err := json.MarshalIndent(n.load(), "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(n.path()), 0o700); err != nil {
		return err
	}
	tmp := n.path() + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, n.path())
}

// askStore adapts the state to operatordecision.PendingStore. The notifier
// lock is held by every caller.
type askStore struct{ n *Notifier }

func (s askStore) Get(_ context.Context, id string) (*operatordecision.PendingAsk, error) {
	return s.n.load().Asks[id], nil
}

func (s askStore) Put(_ context.Context, ask *operatordecision.PendingAsk) error {
	s.n.load().Asks[ask.ID] = ask
	return s.n.save()
}

func (s askStore) CountOutstandingForSubject(_ context.Context, subject string, now time.Time) (int, error) {
	count := 0
	for _, ask := range s.n.load().Asks {
		if ask.Subject == subject && ask.Outstanding(now) {
			count++
		}
	}
	return count, nil
}

type nowSource struct{ n *Notifier }

func (c nowSource) Now() time.Time { return c.n.now() }

func (n *Notifier) raiser() operatordecision.Raiser {
	return n.raiserCap(8)
}

func (n *Notifier) raiserCap(max int) operatordecision.Raiser {
	return operatordecision.Raiser{
		Source: "agent-manager", EventType: DecisionEventType, Store: askStore{n}, Publisher: n.Publisher,
		Clock: nowSource{n}, MaxOutstanding: max, Window: 24 * time.Hour,
	}
}

// subjectFor is the flood-cap key. It is fixed per kind and runner, never per
// ask id: a key that varies with the ask would give every ask its own cap.
func subjectFor(kind, runner string) string {
	if runner == "" {
		return decisionSubject + ":" + kind
	}
	return decisionSubject + ":" + kind + ":" + usagemode.NormalizeRunner(runner)
}

// pendingAskFor is the ask the raiser stores. The decision's own parameters are
// kept on it, because Apply must act on what this process raised and not on a
// correlation echoed back through the hub.
func pendingAskFor(id, subject string, correlation map[string]string, extra map[string]string) operatordecision.PendingAsk {
	stored := map[string]string{}
	for key, value := range correlation {
		stored[key] = value
	}
	for key, value := range extra {
		stored[key] = value
	}
	return operatordecision.PendingAsk{ID: id, Subject: subject, Extra: stored}
}

// alreadyAsked is true once an ask with this id exists in any state, except an
// unpublished one that is still outstanding, which the raiser may retry. A
// resolved or expired ask is never raised again for the same id.
func (n *Notifier) alreadyAsked(id string, now time.Time) bool {
	ask := n.load().Asks[id]
	if ask == nil {
		return false
	}
	return !(ask.State == operatordecision.PendingStatePending && !ask.Published && ask.Outstanding(now))
}

func (n *Notifier) link(ctx context.Context, query string) string {
	if n.UsageURL == nil {
		return ""
	}
	if base := n.UsageURL(ctx); base != "" {
		return base + query
	}
	return ""
}

func providerRunner(provider string) string {
	for runner, p := range runnerProvider {
		if p == provider {
			return runner
		}
	}
	return ""
}

func runnerTitle(runner string) string {
	if runner == "claude" {
		return "Claude"
	}
	return "Codex"
}

// ---- run-out offer ----

// WillExhaust offers to start a one-tier-down mode until the window resets. It
// asks once per window period: a later answer of "Not now", or no answer, is
// not asked again before the window resets.
func (n *Notifier) WillExhaust(ctx context.Context, f usageforecast.Forecast, projected, now time.Time) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.offerExhaust(ctx, f, projected, now, false)
}

func (n *Notifier) offerExhaust(ctx context.Context, f usageforecast.Forecast, projected, now time.Time, reminder bool) error {
	w := f.Window
	runner := providerRunner(w.Provider)
	if runner == "" || w.ResetAt.IsZero() || !w.ResetAt.After(now) {
		return nil
	}
	until := ""
	switch w.WindowMinutes {
	case 300:
		until = "5h-reset"
	case 10080:
		until = "weekly-reset"
	default:
		return nil
	}
	for _, mode := range n.Modes.listAt(now) {
		if usagemode.NormalizeRunner(mode.Scope.Runner) == runner || mode.Scope.Runner == "" {
			if mode.Scope.Role == "" && mode.Scope.Profile == "" {
				return nil
			}
		}
	}
	st := n.load()
	scope := w.Scope()
	offer := st.Offers[scope]
	if offer != nil && now.Before(offer.ResetAt) && !reminder {
		if offer.Declined || !offer.RemindAt.IsZero() {
			return nil
		}
		// Already offered this period: only an unpublished ask is retried.
		if ask := st.Asks[operatordecision.CorrelationID("umx", scope, strconv.Itoa(offer.Gen))]; ask == nil || ask.Published || !ask.Outstanding(now) {
			return nil
		}
	}
	if offer == nil || !now.Before(offer.ResetAt) {
		offer = &windowOffer{ResetAt: w.ResetAt.UTC(), Runner: runner}
		st.Offers[scope] = offer
	}
	id := operatordecision.CorrelationID("umx", scope, strconv.Itoa(offer.Gen))
	title := usagelabel.Title(w.Provider, w.Window, w.WindowMinutes)
	facts := operatordecision.Facts{
		Subject: fmt.Sprintf("%s may run out about %s", title, usageforecast.Stamp(projected)),
		Question: fmt.Sprintf("%s is projected to run out about %s, before it resets %s. Start Save usage (one tier down) for new %s work until the reset? It might help; the effect is not measured yet.",
			title, usageforecast.Stamp(projected), usageforecast.Stamp(w.ResetAt), runnerTitle(runner)),
		BigPicture: "Save usage moves new work to the next cheaper model until the window resets. Running sessions are not changed. You can end it any time from the Usage page.",
		Options: []operatordecision.Option{
			{Key: AnswerStart, Label: "Start until reset"},
			{Key: AnswerNotNow, Label: "Not now"},
			{Key: AnswerRemind, Label: "Remind me in 30 min"},
		},
		Recommended: AnswerStart,
		// Reversible, and deliberately no DefaultAnswer: nothing starts unless a person says so.
		Reversible: true,
		Urgency:    "high",
		ContextURL: n.link(ctx, "?window="+usageforecast.WindowRef(w)),
		Correlation: map[string]string{
			"kind": KindRunsOut, "decision_id": id, "scope": scope, "runner": runner, "until": until,
		},
	}
	_, err := n.raiser().Raise(ctx, pendingAskFor(id, subjectFor(KindRunsOut, runner), facts.Correlation, nil), facts)
	if err != nil {
		return err
	}
	return n.save()
}

// AskToStart raises one decision to start a Save mode of a fixed length on one
// runner. It is the same contract as the run-out offer: an ask only, started by
// the verified human who answers. One per runner and length per day.
func (n *Notifier) AskToStart(ctx context.Context, runner string, length time.Duration) (string, error) {
	runner = usagemode.NormalizeRunner(runner)
	if runner != "claude" && runner != "codex" {
		return "", fmt.Errorf("%w: runner must be claude or codex", ErrInvalid)
	}
	if length < 5*time.Minute || length > 24*time.Hour {
		return "", fmt.Errorf("%w: length must be between 5 minutes and 24 hours", ErrInvalid)
	}
	// Whole quarter hours only: an id per distinct length would let a caller
	// mint a new decision with every request.
	if length = length.Round(AskLengthStep); length < AskLengthStep {
		length = AskLengthStep
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	now := n.now()
	until := length.String()
	id := operatordecision.CorrelationID("umt", runner, until, now.Format("2006-01-02"))
	text := fmt.Sprintf("%d-minute", int(length.Minutes()))
	if length%time.Hour == 0 {
		text = fmt.Sprintf("%d-hour", int(length.Hours()))
	}
	facts := operatordecision.Facts{
		Subject:  fmt.Sprintf("Usage-mode validation: start a %s Save mode on %s?", text, runnerTitle(runner)),
		Question: fmt.Sprintf("Usage-mode validation. Start a %s Save mode on %s to test usage modes?", text, runnerTitle(runner)),
		BigPicture: "This tests the usage mode decision path. Answering Start creates a real mode that moves new " + runnerTitle(runner) +
			" work to the next cheaper model until it expires. Running sessions are not changed. It can be ended any time from the Usage page.",
		Options:     []operatordecision.Option{{Key: AnswerStart, Label: "Start the test mode"}, {Key: AnswerNotNow, Label: "Not now"}},
		Recommended: AnswerStart, Reversible: true, Urgency: "normal",
		ContextURL:  n.link(ctx, ""),
		Correlation: map[string]string{"kind": KindTimed, "decision_id": id, "runner": runner, "until": until},
	}
	// The cap is per runner and small: an agent may ask, but it cannot fill the
	// operator's inbox.
	if _, err := n.raiserCap(MaxOutstandingTimedAsks).Raise(ctx, pendingAskFor(id, subjectFor(KindTimed, runner), facts.Correlation, nil), facts); err != nil {
		return "", err
	}
	return id, n.save()
}

// AskState reports a decision's state ("pending", "approved", "rejected") and its note.
func (n *Notifier) AskState(id string) (state, note string, ok bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	ask := n.load().Asks[id]
	if ask == nil {
		return "", "", false
	}
	return ask.State, ask.Extra["note"], true
}

// ---- resolution ----

// Handles claims only usage mode decisions.
func (n *Notifier) Handles(sourceEventType string) bool { return sourceEventType == DecisionEventType }

// verifiedHuman reports whether the hub carried a named human's own answer. A
// default, an expiry or an answer without an identity never counts.
func verifiedHuman(res operatordecision.DecisionResolution) bool {
	by := strings.TrimSpace(res.AnsweredBy)
	return res.State == operatordecision.StateAnswered && !res.ByDefault && by != "" && by != "default"
}

// Apply acts on one resolution. It is replay-safe: an ask that is no longer
// pending is a benign no-op, so a redelivery never starts or ends a mode twice.
func (n *Notifier) Apply(ctx context.Context, res operatordecision.DecisionResolution) (bool, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	id := strings.TrimSpace(res.Correlation["decision_id"])
	st := n.load()
	ask := st.Asks[id]
	if id == "" || ask == nil || ask.State != operatordecision.PendingStatePending {
		return true, nil
	}
	now := n.now()
	by := strings.TrimSpace(res.AnsweredBy)
	if ask.Extra == nil {
		ask.Extra = map[string]string{}
	}
	finish := func(state, note string) (bool, error) {
		ask.State, ask.ResolvedBy = state, by
		at := now
		ask.ResolvedAt = &at
		ask.Extra["answer"], ask.Extra["note"] = res.Answer, note
		return true, n.save()
	}
	if res.State != operatordecision.StateAnswered {
		return finish(operatordecision.PendingStateRejected, "no answer: "+res.State)
	}
	if res.Late {
		return finish(operatordecision.PendingStateRejected, "refused: the answer arrived after the decision expired")
	}
	// What to act on is what this process stored when it raised the ask. The
	// echoed correlation only locates the ask; it is never trusted for a
	// runner, an end time or a mode id.
	params := ask.Extra
	kind, answer := params["kind"], strings.TrimSpace(res.Answer)
	needsHuman := answer == AnswerStart || answer == AnswerExtend || answer == AnswerTurnOf || answer == AnswerApprove
	if needsHuman && !verifiedHuman(res) {
		return finish(operatordecision.PendingStateRejected, "refused: the answer carries no verified human")
	}
	switch {
	case kind == KindRoleDefault && answer == AnswerApprove:
		// Approval is recorded; the change itself is the operator's to apply.
		return finish(operatordecision.PendingStateApproved, fmt.Sprintf("approved by %s: make %s the default for %s. Not applied: Agent Manager does not edit resource files. Apply this change:\n%s",
			by, params["to"], params["role"], ask.Extra["policy_change"]))
	case (kind == KindRunsOut || kind == KindTimed) && answer == AnswerStart:
		result, err := n.Modes.Start(ctx, StartRequest{Preset: PresetOneTierDown, Runner: params["runner"], Until: params["until"], StartedBy: by})
		if errors.Is(err, ErrInvalid) {
			return finish(operatordecision.PendingStateRejected, "not started: "+err.Error())
		}
		if err != nil {
			return true, err // transient: leave pending so the hub redelivers
		}
		st.TapStarted[result.Mode.ID] = true
		ask.Extra["mode_id"] = result.Mode.ID
		return finish(operatordecision.PendingStateApproved, "started "+result.Mode.ID)
	case (kind == KindRunsOut || kind == KindTimed) && answer == AnswerNotNow:
		if o := st.Offers[params["scope"]]; o != nil {
			o.Declined = true
		}
		return finish(operatordecision.PendingStateRejected, "declined for this window")
	case kind == KindRunsOut && answer == AnswerRemind:
		if o := st.Offers[params["scope"]]; o != nil {
			o.RemindAt = now.Add(RemindAfter)
		}
		return finish(operatordecision.PendingStateRejected, "remind later")
	case kind == KindEnding && answer == AnswerExtend:
		mode, err := n.Modes.Extend(ctx, params["mode_id"], params["extend_to"])
		if errors.Is(err, ErrInvalid) {
			return finish(operatordecision.PendingStateRejected, "not extended: "+err.Error())
		}
		if err != nil {
			return true, err
		}
		return finish(operatordecision.PendingStateApproved, "extended "+mode.ID)
	case kind == KindUntilOf && answer == AnswerTurnOf:
		if _, _, err := n.Modes.End([]string{params["mode_id"]}, false); err != nil {
			return true, err
		}
		return finish(operatordecision.PendingStateApproved, "ended")
	case answer == AnswerLetEnd || answer == AnswerKeepOn:
		return finish(operatordecision.PendingStateRejected, answer)
	}
	return finish(operatordecision.PendingStateRejected, "unknown answer "+answer)
}

// ---- lifecycle ----

func modeProviders(m usagemode.Mode) []string {
	var out []string
	for runner := range m.Runners {
		if p := runnerProvider[usagemode.NormalizeRunner(runner)]; p != "" {
			out = append(out, p)
		}
	}
	return out
}

func endedLine(m usagemode.Mode, now time.Time) string {
	if m.ExpiresAt != nil && !now.Before(*m.ExpiresAt) {
		return fmt.Sprintf("%s ended: it reached its end time.", m.Label)
	}
	return fmt.Sprintf("%s ended early: it was turned off or its quota window reset.", m.Label)
}

// Observe diffs the active modes against the last seen set: a vanished mode
// becomes an ended notice; a new one a started notice unless a tap started it.
func (n *Notifier) Observe(_ context.Context, now time.Time) {
	n.mu.Lock()
	defer n.mu.Unlock()
	st := n.load()
	active := map[string]usagemode.Mode{}
	for _, m := range n.Modes.listAt(now) {
		active[m.ID] = m
	}
	for id, m := range st.Seen {
		if _, ok := active[id]; ok {
			continue
		}
		delete(st.Seen, id)
		if m.ExpiresAt != nil && now.Sub(*m.ExpiresAt) > staleEnded {
			continue
		}
		st.Ended = append(st.Ended, endedNotice{ID: id, Label: m.Label, Providers: modeProviders(m), Line: endedLine(m, now), At: now})
	}
	for id, m := range active {
		if _, seen := st.Seen[id]; !seen && !st.TapStarted[id] && now.Sub(m.StartedAt) < time.Hour {
			st.Started = append(st.Started, m)
		}
		delete(st.TapStarted, id)
		st.Seen[id] = m
	}
	cutoff := now.Add(-stateRetention)
	for id, ask := range st.Asks {
		if ask.RaisedAt.Before(cutoff) {
			delete(st.Asks, id)
		}
	}
	for scope, o := range st.Offers {
		if o.ResetAt.Before(cutoff) {
			delete(st.Offers, scope)
		}
	}
	_ = n.save()
}

// EndedFor lists unannounced ended modes that covered the provider.
func (n *Notifier) EndedFor(provider string) []usageforecast.EndedMode {
	n.mu.Lock()
	defer n.mu.Unlock()
	var out []usageforecast.EndedMode
	for _, e := range n.load().Ended {
		for _, p := range e.Providers {
			if p == provider {
				out = append(out, usageforecast.EndedMode{ID: e.ID, Line: e.Line})
				break
			}
		}
	}
	return out
}

// Announced drops ended notices that another notification already carried.
func (n *Notifier) Announced(ids []string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	drop := map[string]bool{}
	for _, id := range ids {
		drop[id] = true
	}
	st := n.load()
	kept := st.Ended[:0]
	for _, e := range st.Ended {
		if !drop[e.ID] {
			kept = append(kept, e)
		}
	}
	st.Ended = kept
	_ = n.save()
}

func (n *Notifier) notice(ctx context.Context, now time.Time, category, urgency, title, body, dedupe, query string) error {
	if n.Publisher == nil {
		return errors.New("no publisher")
	}
	facts := map[string]any{"category": category, "urgency": urgency, "sensitivity": "private", "title": title, "body": body, "dedupe_key": dedupe}
	if link := n.link(ctx, query); link != "" {
		facts["context_url"] = link
	}
	return n.Publisher.PublishDomainEvent(ctx, eventbus.DomainEvent{Source: "agent-manager", EventType: usageforecast.NotificationEventType, Occurred: now, Payload: facts})
}

// Flush publishes what is due and returns the failures.
func (n *Notifier) Flush(ctx context.Context, forecasts []usageforecast.Forecast, now time.Time) []error {
	n.mu.Lock()
	defer n.mu.Unlock()
	st := n.load()
	var errs []error
	keptEnded := st.Ended[:0]
	for _, e := range st.Ended {
		if err := n.notice(ctx, now, usageforecast.NotificationCategoryModeEnded, "low", e.Label+" ended", e.Line, "mode-ended|"+e.ID, ""); err != nil {
			errs = append(errs, fmt.Errorf("mode ended notice %s: %w", e.ID, err))
			keptEnded = append(keptEnded, e)
		}
	}
	st.Ended = keptEnded
	keptStarted := st.Started[:0]
	for _, m := range st.Started {
		end := "until turned off"
		if m.ExpiresAt != nil {
			end = "until " + usageforecast.Stamp(*m.ExpiresAt)
		}
		if err := n.notice(ctx, now, usageforecast.NotificationCategoryModeStarted, "low", m.Label+" is on", fmt.Sprintf("%s started %s by %s.", m.Label, end, m.StartedBy), "mode-started|"+m.ID, ""); err != nil {
			errs = append(errs, fmt.Errorf("mode started notice %s: %w", m.ID, err))
			keptStarted = append(keptStarted, m)
		}
	}
	st.Started = keptStarted

	for _, m := range n.Modes.listAt(now) {
		if err := n.flushMode(ctx, m, now); err != nil {
			errs = append(errs, err)
		}
	}
	for _, f := range forecasts {
		if f.Window == nil {
			continue
		}
		o := st.Offers[f.Window.Scope()]
		if o == nil {
			continue
		}
		switch {
		case !o.RemindAt.IsZero() && !now.Before(o.RemindAt):
			o.RemindAt = time.Time{}
			if f.Status == usageforecast.StatusWillExhaust && f.Projected != nil {
				o.Gen++
				if err := n.offerExhaust(ctx, f, *f.Projected, now, true); err != nil {
					errs = append(errs, err)
				}
			}
		case f.Status == usageforecast.StatusWillExhaust && f.Projected != nil && !o.Declined && o.RemindAt.IsZero():
			if err := n.offerExhaust(ctx, f, *f.Projected, now, false); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if err := n.flushEvidence(ctx, now); err != nil {
		errs = append(errs, err)
	}
	if err := n.save(); err != nil {
		errs = append(errs, err)
	}
	return errs
}

// flushEvidence tells the operator once per mode that its comparison has a
// result. It sends nothing unless EvidenceReady is on and a source is set.
func (n *Notifier) flushEvidence(ctx context.Context, now time.Time) error {
	if !n.EvidenceReady || n.Evidence == nil || now.Sub(n.evidenceAt) < evidenceEvery {
		return nil
	}
	n.evidenceAt = now
	ready, err := n.Evidence.Ready(ctx, now)
	if err != nil {
		return fmt.Errorf("usage mode evidence: %w", err)
	}
	st := n.load()
	var errs []error
	for _, report := range ready {
		if report.ModeID == "" || st.EvidenceDone[report.ModeID] {
			continue
		}
		if err := n.notice(ctx, now, usageforecast.NotificationCategoryModeEvidence, "low", report.Label+": result ready", report.Line, "mode-evidence|"+report.ModeID, "?tab=evidence"); err != nil {
			errs = append(errs, fmt.Errorf("mode evidence notice %s: %w", report.ModeID, err))
			continue
		}
		st.EvidenceDone[report.ModeID] = true
	}
	return errors.Join(errs...)
}

func (n *Notifier) flushMode(ctx context.Context, m usagemode.Mode, now time.Time) error {
	var errs []error
	if m.ExpiresAt != nil {
		left := m.ExpiresAt.Sub(now)
		if left > 0 && left <= EndingLead && m.ExpiresAt.Sub(m.StartedAt) >= minEndingLifetime {
			errs = append(errs, n.raiseEnding(ctx, m, left, now))
		}
	}
	if m.Until.Kind == usagemode.UntilOff {
		if count := int(now.Sub(m.StartedAt) / UntilOffEvery); count >= 1 {
			errs = append(errs, n.raiseUntilOff(ctx, m, count, now))
		}
	}
	errs = append(errs, n.checkReach(ctx, m, now))
	errs = append(errs, n.checkQuality(ctx, m, now))
	return errors.Join(errs...)
}

func (n *Notifier) raiseEnding(ctx context.Context, m usagemode.Mode, left time.Duration, now time.Time) error {
	id := operatordecision.CorrelationID("ume", m.ID, m.ExpiresAt.UTC().Format(time.RFC3339))
	if n.alreadyAsked(id, now) { // once per expiry, whatever the answer
		return nil
	}
	to, label := ExtendTwoHours, "Extend 2 hours"
	if runner := usagemode.NormalizeRunner(m.Scope.Runner); runner != "" && m.Until.Kind == usagemode.UntilWindowReset && m.Until.WindowMinutes == 300 {
		if _, ok := n.Modes.WeeklyResetAfter(ctx, runner, *m.ExpiresAt); ok {
			to, label = ExtendWeeklyReset, "Extend to weekly reset"
		}
	}
	facts := operatordecision.Facts{
		Subject:     fmt.Sprintf("%s ends in %d minutes", m.Label, int(left.Round(time.Minute).Minutes())),
		Question:    fmt.Sprintf("%s ends %s. After that, new work goes back to its usual model. Extend it?", m.Label, usageforecast.Stamp(*m.ExpiresAt)),
		Options:     []operatordecision.Option{{Key: AnswerExtend, Label: label}, {Key: AnswerLetEnd, Label: "Let it end"}},
		Recommended: AnswerLetEnd, Reversible: true, Urgency: "normal",
		ContextURL:  n.link(ctx, ""),
		Correlation: map[string]string{"kind": KindEnding, "decision_id": id, "mode_id": m.ID, "extend_to": to},
	}
	_, err := n.raiser().Raise(ctx, pendingAskFor(id, subjectFor(KindEnding, m.Scope.Runner), facts.Correlation, nil), facts)
	return err
}

func (n *Notifier) raiseUntilOff(ctx context.Context, m usagemode.Mode, count int, now time.Time) error {
	id := operatordecision.CorrelationID("umo", m.ID, strconv.Itoa(count)) // one id per elapsed day
	if n.alreadyAsked(id, now) {
		return nil
	}
	hours := int(now.Sub(m.StartedAt).Hours())
	facts := operatordecision.Facts{
		Subject:     m.Label + " is still on",
		Question:    fmt.Sprintf("%s has run for %d hours with no end time. Keep it on, or turn it off?", m.Label, hours),
		Options:     []operatordecision.Option{{Key: AnswerKeepOn, Label: "Keep it on"}, {Key: AnswerTurnOf, Label: "Turn off"}},
		Recommended: AnswerKeepOn, Reversible: true, Urgency: "low",
		ContextURL:  n.link(ctx, ""),
		Correlation: map[string]string{"kind": KindUntilOf, "decision_id": id, "mode_id": m.ID},
	}
	_, err := n.raiser().Raise(ctx, pendingAskFor(id, subjectFor(KindUntilOf, m.Scope.Runner), facts.Correlation, nil), facts)
	return err
}

// checkReach tells the operator once per mode when much of the burn since it
// started came from work it cannot change.
func (n *Notifier) checkReach(ctx context.Context, m usagemode.Mode, now time.Time) error {
	st := n.load()
	if n.Reach == nil || st.ReachDone[m.ID] || now.Sub(m.StartedAt) < reachMinAge || now.Sub(st.ReachAt[m.ID]) < reachEvery {
		return nil
	}
	st.ReachAt[m.ID] = now
	share, observed, err := n.Reach.Unreached(ctx, m, now)
	if err != nil || observed < UnreachedMinPoints || share < UnreachedShare {
		return nil
	}
	st.ReachDone[m.ID] = true
	body := fmt.Sprintf("About %.0f%% of the quota used since %s started (%.1f points, estimated) came from work it cannot change, such as desktop apps or tools started outside Vrooli. That use is still counted.",
		share*100, m.Label, observed)
	return n.notice(ctx, now, usageforecast.NotificationCategoryModeUnreachable, "normal", "Some work isn't covered by "+m.Label, body, "mode-unreachable|"+m.ID, "")
}

// checkQuality tells the operator once per mode when their own failure and
// correction reports rose under it beyond chance. The wording states what is
// counted, by what, and how sure: the counts come from a local classifier and
// carry its measured precision; a rise is not proof the mode caused it.
func (n *Notifier) checkQuality(ctx context.Context, m usagemode.Mode, now time.Time) error {
	st := n.load()
	if n.Quality == nil || st.QualityDone[m.ID] || now.Sub(m.StartedAt) < qualityMinAge || now.Sub(st.QualityAt[m.ID]) < qualityEvery {
		return nil
	}
	st.QualityAt[m.ID] = now
	finding, ok, err := n.Quality.QualityDrop(ctx, m, now)
	if err != nil || !ok {
		return nil
	}
	st.QualityDone[m.ID] = true
	body := fmt.Sprintf("Failure and correction reports rose from %.0f%% to %.0f%% of your messages on %s while %s was on (%d messages now, %d before; the rise is %.0f to %.0f points). A local classifier counted them and was right about %.0f%% of the time it flagged one, so this can be wrong, and the mode may not be the cause. You may want full power.",
		finding.Baseline*100, finding.Rate*100, finding.Model, m.Label, finding.Messages, finding.BaselineMessages,
		finding.DifferenceLow*100, finding.DifferenceHigh*100, finding.ClassifierPrecision*100)
	return n.notice(ctx, now, NotificationCategoryQualityDrop, "normal", "More failure reports under "+m.Label, body, "mode-quality-drop|"+m.ID, "?tab=evidence")
}

// DigestNote names the active modes in the daily digest.
func (n *Notifier) DigestNote(now time.Time) string {
	var parts []string
	for _, m := range n.Modes.listAt(now) {
		end := "until turned off"
		if m.ExpiresAt != nil {
			end = "until " + usageforecast.Stamp(*m.ExpiresAt)
		}
		parts = append(parts, m.Label+" "+end)
	}
	if len(parts) == 0 {
		return ""
	}
	return "Usage mode on: " + strings.Join(parts, "; ")
}
