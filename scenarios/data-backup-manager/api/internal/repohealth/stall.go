package repohealth

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	// NoticeInterval is how often a lasting condition is repeated.
	NoticeInterval = 24 * time.Hour
	// MinStallThreshold is the shortest silence worth telling the operator about.
	MinStallThreshold = 48 * time.Hour
)

// StallThreshold is how long a plan may go without a successful backup before it
// counts as stalled: twice its schedule, never less than 48 hours. A plan with
// no (or an unreadable) schedule uses 48 hours.
func StallThreshold(schedule string) time.Duration {
	d, err := time.ParseDuration(strings.TrimSpace(schedule))
	if err != nil || d <= 0 {
		return MinStallThreshold
	}
	if t := 2 * d; t > MinStallThreshold {
		return t
	}
	return MinStallThreshold
}

// PlanStatus is what the notice evaluator needs to know about one plan.
type PlanStatus struct {
	PlanID        string
	Name          string
	Enabled       bool
	Schedule      string
	CreatedAt     time.Time
	LastSuccessAt time.Time
	// LastSuccessSource says where LastSuccessAt came from ("recorded run",
	// "repository snapshot"); empty when there is none.
	LastSuccessSource  string
	TargetCount        int
	Unprotected        int
	UnprotectedTargets []string
	// OpenDecision is the subject of an unanswered question about this plan's
	// destination, or empty.
	OpenDecision string
}

// Notice is one operator notification the evaluator wants sent.
type Notice struct {
	Key      string
	Kind     string
	Category string
	Title    string
	Body     string
	Urgency  string
	// Keys are every cadence key this notice covers (a digest covers several
	// plans). Empty means just Key.
	Keys []string
	// Plan details, kept so several stalled notices can be bundled.
	PlanName     string
	Days         int
	Unprotected  int
	TargetCount  int
	OpenDecision string
}

// CoveredKeys returns the cadence keys to mark once the notice is delivered.
func (n Notice) CoveredKeys() []string {
	if len(n.Keys) > 0 {
		return n.Keys
	}
	return []string{n.Key}
}

// Notice kinds and their hub categories.
const (
	KindStalled          = "stalled"
	KindDisabledUnprotec = "disabled_unprotected"
	KindExceptionActive  = "exception_active"

	CategoryStalled          = "backup-stalled"
	CategoryDisabledUnprotec = "backup-plan-disabled"
	CategoryExceptionActive  = "backup-exception-active"
)

// NoticeState records when each notice key was last sent.
type NoticeState map[string]time.Time

// due reports whether a repeating notice should be sent now.
func (s NoticeState) due(key string, now time.Time) bool {
	last, ok := s[key]
	return !ok || now.Sub(last) >= NoticeInterval
}

// EvaluateStalled returns one notice per enabled plan that has had no successful
// run for longer than its threshold, at most once per day while it lasts.
func EvaluateStalled(now time.Time, plans []PlanStatus, state NoticeState) []Notice {
	var out []Notice
	for _, p := range plans {
		if !p.Enabled {
			continue
		}
		since := p.LastSuccessAt
		if since.IsZero() {
			since = p.CreatedAt
		}
		if since.IsZero() || now.Sub(since) <= StallThreshold(p.Schedule) {
			continue
		}
		key := KindStalled + ":" + p.PlanID
		if !state.due(key, now) {
			continue
		}
		out = append(out, stalledNotice(now, p, since, key))
	}
	return out
}

func stalledNotice(now time.Time, p PlanStatus, since time.Time, key string) Notice {
	days := int(now.Sub(since).Hours() / 24)
	var b strings.Builder
	if p.LastSuccessAt.IsZero() {
		fmt.Fprintf(&b, "There is no successful backup on record for %q; it has been waiting %d %s.", p.Name, days, plural1(days, "day", "days"))
	} else {
		fmt.Fprintf(&b, "%q has not had a successful backup for %d %s (last: %s).", p.Name, days, plural1(days, "day", "days"), p.LastSuccessAt.Format("2 Jan"))
	}
	if p.TargetCount > 0 {
		if p.Unprotected > 0 {
			b.WriteString(" " + Capitalize(unprotectedPhrase(p.Unprotected, p.TargetCount, "backup copy")) + ".")
		} else {
			fmt.Fprintf(&b, " All %d of its items also have another copy.", p.TargetCount)
		}
	}
	if p.OpenDecision != "" {
		fmt.Fprintf(&b, " I asked you a question about this and it is still waiting for your answer: %q.", p.OpenDecision)
	} else {
		b.WriteString(" Nothing is waiting on you in the inbox, but backups are not running; the plan needs a look.")
	}
	urgency := "normal"
	if p.Unprotected > 0 {
		urgency = "high"
	}
	return Notice{
		Key: key, Kind: KindStalled, Category: CategoryStalled, Urgency: urgency,
		Title: fmt.Sprintf("Backups for %s have not succeeded for %d %s", p.Name, days, plural1(days, "day", "days")),
		Body:  b.String(), PlanName: p.Name, Days: days, Unprotected: p.Unprotected, TargetCount: p.TargetCount, OpenDecision: p.OpenDecision,
	}
}

// unprotectedPhrase says how many of a plan's items have no other copy, in
// grammatical English for one item, all items, or some of them.
func unprotectedPhrase(unprotected, total int, noun string) string {
	switch {
	case total == 1:
		return "its only item has no other " + noun
	case unprotected >= total:
		return fmt.Sprintf("all %d of its items have no other %s", total, noun)
	default:
		return fmt.Sprintf("%d of its %d items have no other %s", unprotected, total, noun)
	}
}

// BundleStalled folds the stalled notices that have no open question into one
// digest when there are two or more, so a handful of long-idle plans is one
// message rather than one each. A plan with an open question keeps its own
// notice, which names that question. Every plan is still named and each plan's
// daily cadence is tracked separately through CoveredKeys.
func BundleStalled(notices []Notice, now time.Time) []Notice {
	var own, bundled []Notice
	for _, n := range notices {
		if n.Kind == KindStalled && n.OpenDecision == "" {
			bundled = append(bundled, n)
		} else {
			own = append(own, n)
		}
	}
	if len(bundled) < 2 {
		return notices
	}
	sort.Slice(bundled, func(i, j int) bool { return bundled[i].Days > bundled[j].Days })
	var b strings.Builder
	keys := make([]string, 0, len(bundled))
	urgency := "normal"
	b.WriteString("These backup plans have gone too long without a successful backup. Nothing is waiting on you in the inbox for them; run them or give them a schedule.")
	for _, n := range bundled {
		keys = append(keys, n.Key)
		fmt.Fprintf(&b, "\n- %s: %d %s", n.PlanName, n.Days, plural1(n.Days, "day", "days"))
		if n.TargetCount > 0 && n.Unprotected > 0 {
			b.WriteString("; " + unprotectedPhrase(n.Unprotected, n.TargetCount, "copy"))
		}
		if n.Urgency == "high" {
			urgency = "high"
		}
	}
	digest := Notice{
		Key: KindStalled + "-digest:" + now.UTC().Format("2006-01-02"), Kind: KindStalled, Category: CategoryStalled, Urgency: urgency,
		Title: fmt.Sprintf("%d backup plans have not succeeded for too long", len(bundled)),
		Body:  b.String(), Keys: keys,
	}
	return append(own, digest)
}

// EvaluateDisabledUnprotected returns one consolidated notice, sent once per
// distinct set of switched-off plans, for plans that are disabled yet protect
// items nothing else protects. A silent disabled plan was the failure this
// feature exists to prevent.
func EvaluateDisabledUnprotected(plans []PlanStatus, state NoticeState) (Notice, bool) {
	var rows []PlanStatus
	for _, p := range plans {
		if !p.Enabled && p.Unprotected > 0 {
			rows = append(rows, p)
		}
	}
	if len(rows) == 0 {
		return Notice{}, false
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].PlanID < rows[j].PlanID })
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, fmt.Sprintf("%s:%d", r.PlanID, r.Unprotected))
	}
	sum := sha256.Sum256([]byte(strings.Join(ids, ",")))
	key := KindDisabledUnprotec + ":" + hex.EncodeToString(sum[:8])
	if _, sent := state[key]; sent {
		return Notice{}, false
	}
	lines := make([]string, 0, len(rows))
	for i, r := range rows {
		if i == 8 {
			lines = append(lines, fmt.Sprintf("and %d more", len(rows)-8))
			break
		}
		lines = append(lines, fmt.Sprintf("%s (%d %s)", r.Name, r.Unprotected, plural1(r.Unprotected, "item", "items")))
	}
	return Notice{
		Key: key, Kind: KindDisabledUnprotec, Category: CategoryDisabledUnprotec, Urgency: "normal",
		Title: fmt.Sprintf("%d switched-off backup %s protect things nothing else protects", len(rows), plural1(len(rows), "plan", "plans")),
		Body: "These backup plans are turned off, and the items in them have no other backup copy, so those items are not being backed up at all: " +
			strings.Join(lines, "; ") + ". I will not repeat this unless the list changes. Turn a plan back on if it should be running.",
	}, true
}

// EvaluateExceptionActive returns the standing warning for a destination whose
// backups run under recorded exceptions, at most once per day.
func EvaluateExceptionActive(now time.Time, destinationID, driveName string, active []Exception, state NoticeState) (Notice, bool) {
	if len(active) == 0 {
		return Notice{}, false
	}
	key := KindExceptionActive + ":" + destinationID
	if !state.due(key, now) {
		return Notice{}, false
	}
	first := active[0]
	for _, e := range active {
		if e.ApprovedAt.Before(first.ApprovedAt) {
			first = e
		}
	}
	return Notice{
		Key: key, Kind: KindExceptionActive, Category: CategoryExceptionActive, Urgency: "low",
		Title: fmt.Sprintf("Backups to %s run with %d accepted unreadable %s", driveName, len(active), plural1(len(active), "file", "files")),
		Body: fmt.Sprintf("Backups to %s are running, but %d unreadable leftover %s you accepted on %s still %s on the drive. Any other unreadable file, or a change to these, stops backups and I will ask again. A repair or a fresh backup set would remove this warning.",
			driveName, len(active), plural1(len(active), "file", "files"), first.ApprovedAt.Format("2 Jan"), plural1(len(active), "sits", "sit")),
	}, true
}
