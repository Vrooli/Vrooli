package usagemodes

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"agent-manager/internal/usageforecast"

	"github.com/vrooli/cli-core/usagemode"
)

// Two standing choices let the operator answer a repeating question once.
//
//   - "Keep it on and stop asking" on an until-off mode records the mode as a
//     standing mode: the daily check stops for that mode, and the mode stays
//     visible in the Usage page, the daily digest and `usage mode status`.
//   - "Start now and do this automatically" on a run-out offer records an
//     auto-save rule for the runner: when one of its windows is projected to
//     run out, Save usage starts by itself and the operator gets a notice
//     instead of a question. The rule is listed and switched off from the
//     Usage page, `agent-manager usage mode auto-save`, or the API.
//
// Both are recorded only from a verified human's answer or a verified human's
// write, never from an agent's request.

// AutoSaveStarter is recorded as the mode's starter when a rule started it.
const AutoSaveStarter = "auto-save rule"

type standingMode struct {
	Since time.Time `json:"since"`
	By    string    `json:"by"`
}

// AutoSaveRule is one runner's standing rule.
type AutoSaveRule struct {
	Runner string    `json:"runner"`
	Since  time.Time `json:"since"`
	By     string    `json:"by"`
	// Starts counts the modes the rule has started; LastStart is the latest.
	Starts    int       `json:"starts"`
	LastStart time.Time `json:"last_start,omitempty"`
}

func autoSaveRunner(runner string) (string, error) {
	runner = usagemode.NormalizeRunner(runner)
	if runner != "claude" && runner != "codex" {
		return "", fmt.Errorf("%w: runner must be claude or codex", ErrInvalid)
	}
	return runner, nil
}

// AutoSaveRules lists the active rules, ordered by runner.
func (n *Notifier) AutoSaveRules() []AutoSaveRule {
	n.mu.Lock()
	defer n.mu.Unlock()
	out := make([]AutoSaveRule, 0, len(n.load().AutoSave))
	for _, rule := range n.load().AutoSave {
		out = append(out, *rule)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Runner < out[j].Runner })
	return out
}

// SetAutoSave turns a runner's rule on or off. The caller has verified that
// by is a human operator; turning a rule on starts nothing by itself.
func (n *Notifier) SetAutoSave(runner string, on bool, by string) error {
	runner, err := autoSaveRunner(runner)
	if err != nil {
		return err
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	n.setAutoSave(runner, on, by)
	return n.save()
}

func (n *Notifier) setAutoSave(runner string, on bool, by string) {
	st := n.load()
	if !on {
		delete(st.AutoSave, runner)
		return
	}
	if st.AutoSave[runner] == nil {
		st.AutoSave[runner] = &AutoSaveRule{Runner: runner, Since: n.now(), By: by}
	}
}

// autoSaveFor starts Save usage under the runner's rule when the window is
// projected to run out. It returns true when the rule dealt with the window
// (started a mode, or found the operator already declined or it already ran);
// false lets the caller ask the operator instead, which is also the fallback
// when the mode cannot be started.
func (n *Notifier) autoSaveFor(ctx context.Context, f usageforecast.Forecast, runner, until string, projected, now time.Time) (bool, error) {
	st := n.load()
	rule := st.AutoSave[runner]
	if rule == nil {
		return false, nil
	}
	w := f.Window
	scope := w.Scope()
	offer := st.Offers[scope]
	if offer != nil && now.Before(offer.ResetAt) && (offer.Declined || offer.Auto) {
		return true, nil // declined by the operator, or the rule already ran in this window
	}
	if offer == nil || !now.Before(offer.ResetAt) {
		offer = &windowOffer{ResetAt: w.ResetAt.UTC(), Runner: runner}
		st.Offers[scope] = offer
	}
	result, err := n.Modes.Start(ctx, StartRequest{Preset: PresetOneTierDown, Runner: runner, Until: until, StartedBy: AutoSaveStarter})
	if errors.Is(err, ErrInvalid) {
		return false, nil
	}
	if err != nil {
		return true, err
	}
	offer.Auto, offer.Reopen = true, false
	rule.Starts++
	rule.LastStart = now
	st.TapStarted[result.Mode.ID] = true // this notice replaces the generic "is on" one
	if err := n.save(); err != nil {
		return true, err
	}
	who := runnerTitle(runner)
	body := fmt.Sprintf("%s is on track to run out about %s, before it resets %s, so your auto-save rule switched new %s work to cheaper models until the reset. Work already running is not changed. "+
		"To end this early, tap End now on the Usage page. To stop the rule, switch it off on the Usage page or run: agent-manager usage mode auto-save off --runner %s",
		capitalize(allowanceName(w.Provider, w.Window, w.WindowMinutes)), usageforecast.Stamp(projected), usageforecast.Stamp(w.ResetAt), who, runner)
	return true, n.notice(ctx, now, usageforecast.NotificationCategoryModeStarted, "normal", result.Mode.Label+" started by your auto-save rule", body,
		"mode-auto-save|"+scope+"|"+w.ResetAt.UTC().Format(time.RFC3339), "?window="+usageforecast.WindowRef(w))
}

// keepStanding records an until-off mode as one the operator chose to keep.
func (n *Notifier) keepStanding(modeID, by string) {
	n.load().Standing[modeID] = standingMode{Since: n.now(), By: by}
}

// standingNote is the digest's words for a standing mode, or "".
func (n *Notifier) standingNote(m usagemode.Mode) string {
	if _, ok := n.load().Standing[m.ID]; ok && m.ExpiresAt == nil {
		return " (kept on by you; no daily question)"
	}
	return ""
}

// dropEndedStanding forgets standing marks for modes that are no longer active.
func (n *Notifier) dropEndedStanding(active map[string]usagemode.Mode) {
	st := n.load()
	for id := range st.Standing {
		if _, ok := active[id]; !ok {
			delete(st.Standing, id)
		}
	}
}

// autoSaveSummary is one digest line, empty without rules.
func (n *Notifier) autoSaveSummary() string {
	rules := n.load().AutoSave
	names := make([]string, 0, len(rules))
	for runner := range rules {
		names = append(names, runnerTitle(runner))
	}
	if len(names) == 0 {
		return ""
	}
	sort.Strings(names)
	return "Auto-save rule on for " + strings.Join(names, " and ")
}
