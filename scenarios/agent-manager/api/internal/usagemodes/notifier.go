package usagemodes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/decisionpage"
	"agent-manager/internal/usageforecast"

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
	// KindAutoSaveRule asks whether future run-out warnings should enable the
	// standing rule while a low-power mode is already covering the current one.
	KindAutoSaveRule = "auto-save-rule"
	KindEnding       = "ending"
	KindUntilOf      = "until-off"
	// KindTimed asks to start a fixed-length mode on one runner (a requested ask, e.g. a validation).
	KindTimed = "timed-start"
)

// Answer keys.
const (
	AnswerStart = "start"
	// AnswerAutoSave starts the mode now and records an auto-save rule for the runner.
	AnswerAutoSave = "auto_save"
	AnswerNotNow   = "not_now"
	AnswerRemind   = "remind"
	AnswerExtend   = "extend"
	AnswerLetEnd   = "let_end"
	AnswerKeepOn   = "keep_on"
	AnswerTurnOf   = "turn_off"
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
	// Reopen lets the next run-out projection raise a fresh decision after the
	// previous one was withdrawn because the window recovered.
	Reopen bool `json:"reopen,omitempty"`
	// Auto marks a window an auto-save rule already handled, so the rule starts
	// at most one mode per window even if the operator ends it early.
	Auto bool `json:"auto,omitempty"`
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
	// Standing holds until-off modes the operator chose to keep, so the daily
	// question stops; AutoSave holds the per-runner auto-save rules.
	Standing map[string]standingMode  `json:"standing,omitempty"`
	AutoSave map[string]*AutoSaveRule `json:"auto_save,omitempty"`
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
	if st.Standing == nil {
		st.Standing = map[string]standingMode{}
	}
	if st.AutoSave == nil {
		st.AutoSave = map[string]*AutoSaveRule{}
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
	if legacyAsk(ask) {
		return false // published without a subject key: raise it again under one
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
				// The current low-power mode already addresses this forecast. Do
				// not send a duplicate "start now" decision; offer only the
				// distinct future standing-rule choice when no rule exists.
				if n.load().AutoSave[runner] == nil {
					return n.offerAutoSaveRule(ctx, f, projected, now, runner, mode)
				}
				return nil
			}
		}
	}
	if handled, err := n.autoSaveFor(ctx, f, runner, until, projected, now); handled || err != nil {
		return err
	}
	st := n.load()
	scope := w.Scope()
	offer := st.Offers[scope]
	if offer != nil && now.Before(offer.ResetAt) && !reminder && !offer.Reopen {
		if offer.Declined || !offer.RemindAt.IsZero() {
			return nil
		}
		// Already offered this period: only an unpublished ask is retried.
		if ask := st.Asks[operatordecision.CorrelationID("umx", scope, strconv.Itoa(offer.Gen))]; !legacyAsk(ask) && (ask == nil || ask.Published || !ask.Outstanding(now)) {
			return nil
		}
	}
	if offer == nil || !now.Before(offer.ResetAt) {
		offer = &windowOffer{ResetAt: w.ResetAt.UTC(), Runner: runner}
		st.Offers[scope] = offer
	}
	offer.Reopen = false
	id := operatordecision.CorrelationID("umx", scope, strconv.Itoa(offer.Gen))
	allowance, who := allowanceName(w.Provider, w.Window, w.WindowMinutes), runnerTitle(runner)
	facts := operatordecision.Facts{
		Subject: fmt.Sprintf("%s may run out about %s", capitalize(allowance), usageforecast.Stamp(projected)),
		Question: fmt.Sprintf("%s is on track to run out about %s, %s before it resets. Switch new %s work to cheaper models until the reset?",
			capitalize(allowance), usageforecast.Stamp(projected), decisionpage.Duration(w.ResetAt.Sub(projected)), who),
		BigPicture: fmt.Sprintf("Your plan gives %s a fixed allowance per window. When it runs out, %s work stops until the reset at %s. ", who, who, usageforecast.Stamp(w.ResetAt)) +
			fmt.Sprintf("Switching sends new %s work to the next cheaper model until then, so the allowance lasts longer; work already running is not changed. ", who) +
			"How much it helps is not measured yet. You can end it any time from the Usage page. " +
			"This question comes back every time a window is projected to run out. The second choice below ends that: it starts now and makes the switch automatic for " + who + ", with a notice each time instead of a question, until you switch the rule off on the Usage page. " +
			"If you do not answer, nothing changes.",
		Options: []operatordecision.Option{
			{Key: AnswerStart, Label: "Use cheaper models until the reset (ask me again next time)"},
			{Key: AnswerAutoSave, Label: "Use cheaper models now, and do it automatically from now on (I will send a notice, not a question)"},
			{Key: AnswerNotNow, Label: "Not now: keep the usual models"},
			{Key: AnswerRemind, Label: "Ask me again in 30 minutes"},
		},
		Recommended:          AnswerAutoSave,
		RecommendationReason: "Running out stops all " + who + " work until the reset and cheaper models keep it going; a rule means you are not asked each time, every automatic start sends you a notice, and you can switch the rule off on the Usage page.",
		// Reversible, and deliberately no DefaultAnswer: nothing starts unless a person says so.
		Reversible: true,
		Urgency:    "high",
		ContextURL: n.link(ctx, "?window="+usageforecast.WindowRef(w)),
		Correlation: map[string]string{
			"kind": KindRunsOut, "decision_id": id, "scope": scope, "runner": runner, "until": until,
		},
		SubjectKey:   subjectFor(KindRunsOut, runner),
		TemplateName: "usage-mode",
		TemplateData: usageModeTemplateData(f, projected, runner, until, n.tierMap(ctx, runner)),
	}
	if err := n.raise(ctx, n.raiser(), id, subjectFor(KindRunsOut, runner), facts, nil); err != nil {
		return err
	}
	return n.save()
}

// offerAutoSaveRule asks about future projected run-outs while an active mode
// already covers the current forecast. It never starts or changes a mode.
func (n *Notifier) offerAutoSaveRule(ctx context.Context, f usageforecast.Forecast, projected, now time.Time, runner string, mode usagemode.Mode) error {
	w := f.Window
	if w == nil || f.Projected == nil {
		return nil
	}
	id := operatordecision.CorrelationID("uma", mode.ID, w.Scope())
	if n.alreadyAsked(id, now) {
		return nil
	}
	who := runnerTitle(runner)
	reset := w.ResetAt.UTC()
	until := reset
	if mode.ExpiresAt != nil && mode.ExpiresAt.Before(until) {
		until = mode.ExpiresAt.UTC()
	}
	window := map[string]any{
		"source": map[string]any{"status": "real", "label": "Agent Manager quota forecast", "asOf": f.AsOf.UTC().Format(time.RFC3339), "note": "Observed provider quota readings. The current low-power mode is already active; this decision changes only future automatic behavior."},
		"start":  w.Start.UTC().Format(time.RFC3339), "reset": reset.Format(time.RFC3339), "now": f.Anchor.UTC().Format(time.RFC3339),
		"readings": readingsFor(w, f), "safePace": true,
		"lowPower": []map[string]any{{"from": f.Anchor.UTC().Format(time.RFC3339), "to": until.Format(time.RFC3339), "label": "Current mode already active"}},
	}
	segments := []map[string]any{{"from": f.Anchor.UTC().Format(time.RFC3339), "to": until.Format(time.RFC3339), "mode": "low"}}
	options := []map[string]any{
		{"key": AnswerAutoSave, "title": "Enable automatic low power for future run-outs", "short": "A · Enable rule", "recommended": true, "pill": map[string]any{"text": "Future windows", "tone": "good"}, "bullets": []string{"Does not start or change a mode now.", "For future projected run-outs, switch new work to the lower-cost model until reset.", "You receive a notice each time; turn the rule off any time."}, "segments": segments},
		{"key": AnswerNotNow, "title": "Leave automatic low power off", "short": "B · Leave off", "pill": map[string]any{"text": "No rule", "tone": "info"}, "bullets": []string{"The current low-power mode continues until its scheduled end.", "Future run-out warnings will ask before starting a mode.", "No automatic rule is created."}, "segments": segments},
	}
	modelSummary := "The lower-cost model mapping could not be read from resource policy."
	modelMap := n.tierMap(ctx, runner)
	if len(modelMap) > 0 {
		models := make([]string, 0, len(modelMap))
		for model := range modelMap {
			models = append(models, model)
		}
		sort.Strings(models)
		pairs := make([]string, 0, len(models))
		for _, model := range models {
			pairs = append(pairs, model+" → "+modelMap[model])
		}
		modelSummary = strings.Join(pairs, "; ")
	}
	facts := operatordecision.Facts{
		Subject:     fmt.Sprintf("Set up future automatic low power for %s?", who),
		Question:    fmt.Sprintf("%s is already in low-power mode through %s. Should future projected run-outs switch to low power automatically?", who, usageforecast.Stamp(until)),
		BigPicture:  "The current low-power mode already covers this forecast, so this choice will not start or change a mode now. Enabling the rule applies low power automatically only on future projected run-outs and sends a notice each time. You can turn the rule off from the Usage page.",
		Options:     []operatordecision.Option{{Key: AnswerAutoSave, Label: "Enable automatic low power for future run-outs"}, {Key: AnswerNotNow, Label: "Leave automatic low power off; ask me next time"}},
		Recommended: AnswerAutoSave, RecommendationReason: "Low power already covers the current forecast; the rule affects future run-out windows and remains reversible.",
		Reversible: true, Urgency: "normal", ContextURL: n.link(ctx, "?window="+usageforecast.WindowRef(w)),
		Correlation: map[string]string{"kind": KindAutoSaveRule, "decision_id": id, "runner": runner},
		SubjectKey:  subjectFor(KindAutoSaveRule, runner), TemplateName: "usage-mode",
		TemplateData: map[string]any{
			"meta":           map[string]any{"title": "Make low power automatic for future run-outs?", "producer": "Agent Manager · model usage", "sentAt": now.UTC().Format(time.RFC3339), "timezone": usageforecast.DigestLocation().String(), "notice": "The current low-power mode is already active. This choice does not start or change a mode now; it changes only future run-out warnings."},
			"recommendation": map[string]any{"option": AnswerAutoSave, "headline": "Keep today’s mode as it is; choose whether future run-outs start low power automatically.", "why": fmt.Sprintf("The current mode is active through %s. The provider forecast is based on observed quota readings; savings from lower-cost models are not measured.", usageforecast.Stamp(until))},
			"window":         window, "options": options,
			"cause":      map[string]any{"title": "Why this question exists", "text": fmt.Sprintf("A real provider forecast projects this allowance to run out about %s, while %s is already in low-power mode through %s. This decision sets only the future standing rule.", usageforecast.Stamp(projected), who, usageforecast.Stamp(until))},
			"recurrence": map[string]any{"stop": "Enabling automatic low power replaces future questions with a notice."},
			"noAnswer":   map[string]any{"text": "Nothing changes. The current low-power mode stays as it is, and future run-out warnings remain decisions."},
			"technical":  []map[string]string{{"label": "Runner", "value": runner}, {"label": "Current mode", "value": mode.ID + " · until " + until.Format(time.RFC3339)}, {"label": "Quota window", "value": fmt.Sprintf("%s · %s · %d minutes", w.Provider, w.Window, w.WindowMinutes)}, {"label": "Forecast status", "value": f.Status + ": " + f.Reason}, {"label": "Lower-cost model mapping", "value": modelSummary}, {"label": "Current rule", "value": "off"}},
		},
	}
	return n.raise(ctx, n.raiser(), id, subjectFor(KindAutoSaveRule, runner), facts, map[string]string{"runner": runner, "mode_id": mode.ID})
}

func readingsFor(w *usageforecast.Window, f usageforecast.Forecast) []map[string]any {
	series := w.Series()
	readings := make([]map[string]any, 0, len(series))
	for _, point := range series {
		readings = append(readings, map[string]any{"t": point.At.UTC().Format(time.RFC3339), "used": point.Percent})
	}
	if len(readings) == 0 && w.HasPercent && !w.LastAt.IsZero() {
		readings = append(readings, map[string]any{"t": w.LastAt.UTC().Format(time.RFC3339), "used": f.Used})
	}
	return readings
}

func usageModeTemplateData(f usageforecast.Forecast, projected time.Time, runner, until string, modelMap map[string]string) map[string]any {
	w := f.Window
	series := w.Series()
	readings := make([]map[string]any, 0, len(series))
	for _, point := range series {
		readings = append(readings, map[string]any{"t": point.At.UTC().Format(time.RFC3339), "used": point.Percent})
	}
	if len(readings) == 0 && w.HasPercent && !w.LastAt.IsZero() {
		readings = append(readings, map[string]any{"t": w.LastAt.UTC().Format(time.RFC3339), "used": f.Used})
	}
	var paceNow map[string]any
	for i := len(f.Burn) - 1; i >= 0; i-- {
		if f.Burn[i].Available {
			paceNow = map[string]any{"ratePerHour": f.Burn[i].Rate, "label": "Pace now (usual models)"}
			break
		}
	}
	swaps := make([]string, 0, len(modelMap))
	models := make([]string, 0, len(modelMap))
	for model := range modelMap {
		models = append(models, model)
	}
	sort.Strings(models)
	for _, model := range models {
		swaps = append(swaps, model+" → "+modelMap[model])
	}
	now := f.Anchor
	window := map[string]any{
		"source": map[string]any{"status": "real", "label": "Agent Manager quota forecast", "asOf": f.AsOf.UTC().Format(time.RFC3339), "note": "Observed provider quota readings. Low-power savings are not measured and are not projected."},
		"start":  w.Start.UTC().Format(time.RFC3339), "reset": w.ResetAt.UTC().Format(time.RFC3339),
		"now": now.UTC().Format(time.RFC3339), "readings": readings, "safePace": true,
	}
	if paceNow != nil {
		window["paceNow"] = paceNow
	}
	fullSegments := []map[string]any{{"from": now.UTC().Format(time.RFC3339), "to": projected.UTC().Format(time.RFC3339), "mode": "full"}}
	if projected.Before(w.ResetAt) {
		fullSegments = append(fullSegments, map[string]any{"from": projected.UTC().Format(time.RFC3339), "to": w.ResetAt.UTC().Format(time.RFC3339), "mode": "stopped"})
	}
	lowSegments := []map[string]any{{"from": now.UTC().Format(time.RFC3339), "to": w.ResetAt.UTC().Format(time.RFC3339), "mode": "low"}}
	baseProjection := map[string]any{"runOut": projected.UTC().Format(time.RFC3339)}
	if f.BandStart != nil || f.BandEnd != nil {
		rangeData := map[string]any{}
		if f.BandStart != nil && !f.BandStart.After(w.ResetAt) {
			rangeData["early"] = f.BandStart.UTC().Format(time.RFC3339)
		}
		if f.BandEnd != nil && !f.BandEnd.After(w.ResetAt) {
			rangeData["late"] = f.BandEnd.UTC().Format(time.RFC3339)
		}
		if len(rangeData) > 0 {
			baseProjection["runOutRange"] = rangeData
		}
	}
	modelSummary := "The lower-cost model mapping could not be read from resource policy."
	if len(swaps) > 0 {
		modelSummary = strings.Join(swaps, "; ")
	}
	return map[string]any{
		"meta": map[string]any{
			"title":    "Use lower-cost models until the allowance resets?",
			"producer": "Agent Manager · model usage", "sentAt": now.UTC().Format(time.RFC3339),
			"timezone": usageforecast.DigestLocation().String(),
			"notice":   "The provider forecast is measured. Savings from lower-cost models have not been measured, so no low-power run-out number is shown.",
		},
		"recommendation": map[string]any{
			"option":   AnswerAutoSave,
			"headline": "Use lower-cost models now and make this switch automatic for future run-out warnings.",
			"why":      fmt.Sprintf("The usual-model forecast reaches exhaustion about %s before the provider resets. The effect of the lower-cost model mapping is not measured yet.", usageforecast.Stamp(projected)),
		},
		"window": window,
		"options": []map[string]any{
			{"key": AnswerStart, "title": "Use lower-cost models until reset", "short": "A · Low power", "pill": map[string]any{"text": "Until reset", "tone": "info"}, "bullets": []string{"New work uses the model mapping shown below.", "It returns to the usual models at reset.", "Savings are not measured; no low-power projection is shown."}, "segments": lowSegments},
			{"key": AnswerAutoSave, "title": "Use low power now and automatically next time", "short": "B · Automatic", "recommended": true, "pill": map[string]any{"text": "Recommended", "tone": "good"}, "bullets": []string{"Starts the same lower-cost mode now.", "Future run-out warnings start it automatically and send a notice.", "You can turn the rule off from the Usage page."}, "segments": lowSegments},
			{"key": AnswerNotNow, "title": "Keep the usual models", "short": "C · Keep usual", "pill": map[string]any{"text": "No change", "tone": "warn"}, "bullets": []string{"No usage mode starts.", "The current forecast shows when this allowance runs out.", "This decision is not asked again before reset."}, "projection": baseProjection, "segments": fullSegments},
			{"key": AnswerRemind, "title": "Ask me again in 30 minutes", "short": "D · Remind", "pill": map[string]any{"text": "Ask later", "tone": "info"}, "bullets": []string{"No usage mode starts now.", "The same real forecast remains in effect until then."}, "projection": baseProjection, "segments": fullSegments},
		},
		"cause": map[string]any{
			"title": "Why this question exists",
			"text":  fmt.Sprintf("The provider's recorded quota readings project that the usual models will use this allowance by %s, before its reset. The model policy maps each usual model to a lower-cost option, but the savings from those swaps are not measured yet.", usageforecast.Stamp(projected)),
		},
		"recurrence": map[string]any{"stop": "Automatic low power replaces future questions with a notice."},
		"noAnswer": map[string]any{
			"text": fmt.Sprintf("Nothing changes. New work stays on the usual models; if the forecast holds, this allowance is used by %s and work on it stops until reset.", usageforecast.Stamp(projected)),
		},
		"technical": []map[string]string{
			{"label": "Runner", "value": runner},
			{"label": "Quota window", "value": fmt.Sprintf("%s · %s · %d minutes", w.Provider, w.Window, w.WindowMinutes)},
			{"label": "Forecast status", "value": f.Status + ": " + f.Reason},
			{"label": "Lower-cost model mapping", "value": modelSummary},
			{"label": "Mode duration", "value": until},
		},
	}
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
	who := runnerTitle(runner)
	facts := operatordecision.Facts{
		Subject:  fmt.Sprintf("Test cheaper models on %s for %s?", who, decisionpage.Duration(length)),
		Question: fmt.Sprintf("An agent asks to test usage modes: switch new %s work to cheaper models for %s. Start the test?", who, decisionpage.Duration(length)),
		BigPicture: "This checks that usage-mode decisions work end to end. Starting creates a real mode: new " + who +
			" work goes to the next cheaper model until it expires, and work already running is not changed. You can end it any time from the Usage page. If you do not answer, nothing starts.",
		Options:     []operatordecision.Option{{Key: AnswerStart, Label: "Start the test"}, {Key: AnswerNotNow, Label: "Not now"}},
		Recommended: AnswerStart, RecommendationReason: "It ends by itself and can be ended early; it only proves the path works.",
		Reversible: true, Urgency: "normal",
		ContextURL:  n.link(ctx, ""),
		Correlation: map[string]string{"kind": KindTimed, "decision_id": id, "runner": runner, "until": until},
	}
	facts.Explainer = n.render(ctx, modePage{
		facts: facts,
		rows:  []operatordecision.Row{{Label: "Runs on", Value: who}, {Label: "Lasts", Value: decisionpage.Duration(length)}},
		effects: map[string]string{
			AnswerStart:  "New " + who + " work uses the cheaper models above until the test ends.",
			AnswerNotNow: "Nothing starts.",
		},
		noReply: "Nothing starts.",
	}, nil, runner)
	// The cap is per runner and small: an agent may ask, but it cannot fill the
	// operator's inbox.
	if err := n.raise(ctx, n.raiserCap(MaxOutstandingTimedAsks), id, subjectFor(KindTimed, runner), facts, nil); err != nil {
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
	if res.State == operatordecision.StateWithdrawn {
		// The hub closed the decision (replaced, or no longer confirmed). It is no
		// answer and never starts, extends or ends a mode.
		return finish(operatordecision.PendingStateWithdrawn, "withdrawn")
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
	needsHuman := answer == AnswerStart || answer == AnswerAutoSave || answer == AnswerKeepOn || answer == AnswerExtend || answer == AnswerTurnOf || answer == AnswerApprove
	if needsHuman && !verifiedHuman(res) {
		return finish(operatordecision.PendingStateRejected, "refused: the answer carries no verified human")
	}
	switch {
	case kind == KindAutoSaveRule && answer == AnswerAutoSave:
		runner, err := autoSaveRunner(params["runner"])
		if err != nil {
			return finish(operatordecision.PendingStateRejected, "rule not enabled: "+err.Error())
		}
		n.setAutoSave(runner, true, by)
		return finish(operatordecision.PendingStateApproved, "enabled the future auto-save rule for "+runner+"; the current mode was not changed")
	case kind == KindAutoSaveRule && answer == AnswerNotNow:
		return finish(operatordecision.PendingStateRejected, "left the auto-save rule off")
	case kind == KindRoleDefault && answer == AnswerApprove:
		// Approval is recorded; the change itself is the operator's to apply.
		return finish(operatordecision.PendingStateApproved, fmt.Sprintf("approved by %s: make %s the default for %s. Not applied: Agent Manager does not edit resource files. Apply this change:\n%s",
			by, params["to"], params["role"], ask.Extra["policy_change"]))
	case kind == KindRunsOut && answer == AnswerAutoSave:
		result, err := n.Modes.Start(ctx, StartRequest{Preset: PresetOneTierDown, Runner: params["runner"], Until: params["until"], StartedBy: by})
		if errors.Is(err, ErrInvalid) {
			return finish(operatordecision.PendingStateRejected, "not started: "+err.Error())
		}
		if err != nil {
			return true, err
		}
		st.TapStarted[result.Mode.ID] = true
		ask.Extra["mode_id"] = result.Mode.ID
		runner, _ := autoSaveRunner(params["runner"])
		n.setAutoSave(runner, true, by)
		if o := st.Offers[params["scope"]]; o != nil {
			o.Auto = true
		}
		return finish(operatordecision.PendingStateApproved, "started "+result.Mode.ID+" and turned on the auto-save rule for "+runner)
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
	case kind == KindUntilOf && answer == AnswerKeepOn:
		n.keepStanding(params["mode_id"], by)
		return finish(operatordecision.PendingStateApproved, "kept on; standing, so not asked again")
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
	n.dropEndedStanding(active)
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
	// Close decisions whose situation has cleared before anything new is raised.
	errs := n.reconcileAsks(ctx, forecasts, now)
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
		if f.Status == usageforecast.StatusWillExhaust && f.Projected != nil {
			runner := providerRunner(f.Window.Provider)
			if runner != "" && st.AutoSave[runner] == nil {
				for _, mode := range n.Modes.listAt(now) {
					if (usagemode.NormalizeRunner(mode.Scope.Runner) == runner || mode.Scope.Runner == "") && mode.Scope.Role == "" && mode.Scope.Profile == "" {
						if err := n.offerAutoSaveRule(ctx, f, *f.Projected, now, runner, mode); err != nil {
							errs = append(errs, fmt.Errorf("future auto-save rule decision %s: %w", runner, err))
						}
						break
					}
				}
			}
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
	work := modeWork(m)
	facts := operatordecision.Facts{
		Subject:  fmt.Sprintf("%s ends in %d minutes", operatordecision.Clip(m.Label, 80), int(left.Round(time.Minute).Minutes())),
		Question: fmt.Sprintf("%s ends %s. After that, new work goes back to its usual model. Extend it?", operatordecision.Clip(m.Label, 80), usageforecast.Stamp(*m.ExpiresAt)),
		BigPicture: fmt.Sprintf("%s sends %s to cheaper models so your allowance lasts longer, and it was set to end at %s. ", m.Label, work, usageforecast.Stamp(*m.ExpiresAt)) +
			"Letting it end returns new work to the usual, stronger models, which use the allowance faster. Extending keeps the cheaper models on longer. If you do not answer, it ends on time.",
		Options:              []operatordecision.Option{{Key: AnswerExtend, Label: label}, {Key: AnswerLetEnd, Label: "Let it end"}},
		Recommended:          AnswerLetEnd,
		RecommendationReason: "It was set to end now; extend only if the allowance is still tight.",
		Reversible:           true, Urgency: "normal",
		ContextURL:  n.link(ctx, ""),
		Correlation: map[string]string{"kind": KindEnding, "decision_id": id, "mode_id": m.ID, "extend_to": to},
	}
	facts.Explainer = n.render(ctx, modePage{
		facts: facts,
		rows:  modeRows(m),
		effects: map[string]string{
			AnswerExtend: "The cheaper models above stay on for new work until the new end time.",
			AnswerLetEnd: "New work goes back to the usual models at " + usageforecast.Stamp(*m.ExpiresAt) + ".",
		},
		noReply: "It ends on time and new work goes back to the usual models.",
	}, &m, "")
	return n.raise(ctx, n.raiser(), id, subjectFor(KindEnding, m.Scope.Runner), facts,
		map[string]string{"label": m.Label, "expires_at": m.ExpiresAt.UTC().Format(time.RFC3339)})
}

func (n *Notifier) raiseUntilOff(ctx context.Context, m usagemode.Mode, count int, now time.Time) error {
	if _, standing := n.load().Standing[m.ID]; standing {
		return nil // the operator chose to keep it on; it is not asked about again
	}
	id := operatordecision.CorrelationID("umo", m.ID, strconv.Itoa(count)) // one id per elapsed day
	if n.alreadyAsked(id, now) {
		return nil
	}
	facts := operatordecision.Facts{
		Subject:  operatordecision.Clip(m.Label, 80) + " is still on",
		Question: fmt.Sprintf("%s has been on for %s with no end time. Keep it on, or turn it off?", operatordecision.Clip(m.Label, 80), onFor(now.Sub(m.StartedAt))),
		Options: []operatordecision.Option{
			{Key: AnswerKeepOn, Label: "Keep it on and stop asking — you can turn it off any time on the Usage page"},
			{Key: AnswerTurnOf, Label: "Turn it off — new work goes back to the usual models"},
		},
		Recommended:          AnswerKeepOn,
		RecommendationReason: "A mode with no end time is a standing choice: keeping it changes nothing and saves quota, and it stays one tap from off on the Usage page; turn it off only if cheaper models are hurting the work.",
		BigPicture:           untilOffBigPicture(m),
		TechnicalDetails:     untilOffDetails(m),
		Reversible:           true, Urgency: "low",
		ContextURL:  n.link(ctx, ""),
		Correlation: map[string]string{"kind": KindUntilOf, "decision_id": id, "mode_id": m.ID},
	}
	facts.Explainer = n.render(ctx, modePage{
		facts: facts,
		rows:  modeRows(m),
		effects: map[string]string{
			AnswerKeepOn: "Nothing changes. New work keeps using the cheaper models above, I stop asking about this mode, and it stays listed on the Usage page and in the daily digest with an End now button.",
			AnswerTurnOf: "New work goes back to the usual, stronger models right away. They use the allowance faster.",
		},
		noReply: "It stays on, and I ask again tomorrow.",
	}, &m, "")
	return n.raise(ctx, n.raiser(), id, subjectFor(KindUntilOf, m.Scope.Runner), facts, map[string]string{"label": m.Label})
}

// onFor says how long a mode has been on, in hours under two days and in
// days after.
func onFor(d time.Duration) string {
	if hours := int(d.Hours()); hours < 48 {
		return fmt.Sprintf("%d hours", hours)
	}
	return fmt.Sprintf("about %d days", int(d.Hours()+12)/24)
}

// modeWork is "new Codex work", or "new agent work" for a mode on every runner.
func modeWork(m usagemode.Mode) string {
	if m.Scope.Runner != "" {
		return "new " + runnerTitle(m.Scope.Runner) + " work"
	}
	return "new agent work"
}

// modeRows are the page facts every decision about an existing mode shows.
func modeRows(m usagemode.Mode) []operatordecision.Row {
	end := "When you turn it off"
	if m.ExpiresAt != nil {
		end = usageforecast.Stamp(*m.ExpiresAt)
	}
	rows := []operatordecision.Row{
		{Label: "Applies to", Value: capitalize(modeWork(m))},
		{Label: "Started", Value: usageforecast.Stamp(m.StartedAt)},
		{Label: "Ends", Value: end},
	}
	if m.StartedBy != "" && !strings.Contains(m.StartedBy, "-") {
		rows = append(rows, operatordecision.Row{Label: "Started by", Value: m.StartedBy})
	}
	return rows
}

// untilOffBigPicture explains an until-off mode to an operator who has
// forgotten starting it: what it does, what each answer does, and what
// happens without one.
func untilOffBigPicture(m usagemode.Mode) string {
	var b strings.Builder
	fmt.Fprintf(&b, "This setting sends %s to cheaper models so your usage quota lasts longer. ", modeWork(m))
	fmt.Fprintf(&b, "It was turned on %s and has no end time, so I check with you once a day until you tell me it is a standing choice. ", usageforecast.Stamp(m.StartedAt))
	if n := len(m.ExemptRoles); n > 0 {
		fmt.Fprintf(&b, "%d kinds of work are exempt and keep the usual models. ", n)
	}
	b.WriteString("Keep it on and stop asking: nothing changes, and I do not ask about this mode again; it stays on the Usage page with an End now button. Turn it off and new work goes back to the usual, stronger models, which use quota faster. ")
	b.WriteString("If you do not answer, it stays on and I ask again tomorrow.")
	return b.String()
}

// untilOffDetails is the mode's exact configuration for the agent that acts
// on the answer.
func untilOffDetails(m usagemode.Mode) string {
	lines := []string{
		"Usage mode " + m.ID + " (" + describe(m) + ")",
		"Started " + m.StartedAt.UTC().Format(time.RFC3339) + nonEmpty(" by ", m.StartedBy) + "; until off",
	}
	runners := make([]string, 0, len(m.Runners))
	for r := range m.Runners {
		runners = append(runners, r)
	}
	sort.Strings(runners)
	for _, r := range runners {
		rs := m.Runners[r]
		from := make([]string, 0, len(rs.ModelMap))
		for k := range rs.ModelMap {
			from = append(from, k)
		}
		sort.Strings(from)
		var maps []string
		for _, k := range from {
			maps = append(maps, k+" -> "+rs.ModelMap[k])
		}
		line := r + ": " + strings.Join(maps, ", ")
		line += nonEmpty("; session model ", rs.SessionModel) + nonEmpty("; every model -> ", rs.OverrideModel) + nonEmpty("; effort ", rs.Effort)
		if rs.ContextLimitTokens > 0 {
			line += fmt.Sprintf("; context limit %d tokens", rs.ContextLimitTokens)
		}
		lines = append(lines, line)
	}
	if len(m.ExemptRoles) > 0 {
		lines = append(lines, "Exempt roles: "+strings.Join(m.ExemptRoles, ", "))
	}
	lines = append(lines, "Status: agent-manager usage mode status; end: agent-manager usage mode end "+m.ID)
	return strings.Join(lines, "\n")
}

func nonEmpty(prefix, value string) string {
	if value == "" {
		return ""
	}
	return prefix + value
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
	n.mu.Lock()
	defer n.mu.Unlock()
	var parts []string
	for _, m := range n.Modes.listAt(now) {
		end := "until turned off"
		if m.ExpiresAt != nil {
			end = "until " + usageforecast.Stamp(*m.ExpiresAt)
		}
		parts = append(parts, m.Label+" "+end+n.standingNote(m))
	}
	note := ""
	if len(parts) > 0 {
		note = "Usage mode on: " + strings.Join(parts, "; ")
	}
	if rules := n.autoSaveSummary(); rules != "" {
		if note != "" {
			note += ". "
		}
		note += rules
	}
	return note
}
