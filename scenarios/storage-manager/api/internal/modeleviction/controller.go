// Package modeleviction runs storage-manager's model-store disk policy: on a
// cadence it decides (providers.DecideModelEvictions), tells the operator when
// a filesystem is in a pressure band or a store is over its budget, and turns
// each eviction selection into one Notification Hub decision. An approved
// decision evicts exactly the approved models, each through its owner's fenced
// operation, and only those still eligible at that moment.
package modeleviction

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/api-core/hubsender"
	"github.com/vrooli/api-core/operatordecision"

	"storage-manager/internal/providers"
)

const (
	// Source and Kind name the decision: storage-manager.model-eviction.decision_requested.v1.
	Source = "storage-manager"
	Kind   = "model-eviction"
	// SubjectKey keeps at most one open model-eviction decision at the Hub; a
	// changed selection updates it in place.
	SubjectKey = "model-eviction"
	// AskWindow is how long one selection stays open before it is re-asserted.
	AskWindow = 7 * 24 * time.Hour

	correlationAskID = "model_eviction_ask_id"
	extraSelection   = "selection"
	extraAnswer      = "answer"
	extraOutcome     = "outcome"
	answerApprove    = "approve"
	answerDecline    = "decline"
	stateApproved    = operatordecision.PendingStateApproved
	stateRejected    = operatordecision.PendingStateRejected
)

// Decider is the provider surface the controller needs.
type Decider interface {
	Decide(context.Context) providers.ModelEvictionDecision
	ApplyApproved(ctx context.Context, approved []providers.ModelCandidate, reason, requestedBy string) []providers.ModelEvictionOutcome
	Policy() providers.ModelRetentionPolicy
}

// NoticeSender sends the operator a plain notice.
type NoticeSender interface {
	Send(context.Context, hubsender.Notice) error
}

// Controller owns the cadence, the notices and the decision lifecycle.
type Controller struct {
	Decider Decider
	Raiser  operatordecision.Raiser
	Store   *FileStore
	Notices NoticeSender
	Now     func() time.Time
	Logf    func(string, ...any)
}

func (c *Controller) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c *Controller) logf(format string, args ...any) {
	if c.Logf != nil {
		c.Logf(format, args...)
	}
}

// Run ticks at the policy's interval until ctx ends. gate defers a tick while
// the scenario's background work is paused.
func (c *Controller) Run(ctx context.Context, gate func(func() error) error) {
	interval := c.Decider.Policy().CheckInterval
	tick := func() {
		run := func() error { return c.Tick(ctx) }
		var err error
		if gate != nil {
			err = gate(run)
		} else {
			err = run()
		}
		if err != nil {
			c.logf("model retention: %v", err)
		}
	}
	tick()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tick()
		}
	}
}

// TickResult says what one tick did, for logs and tests.
type TickResult struct {
	Decision     providers.ModelEvictionDecision
	NoticeSent   bool
	AskID        string
	Raised       bool
	Withdrawn    int
	AutoOutcomes []providers.ModelEvictionOutcome
}

// Tick decides once and acts on the decision.
func (c *Controller) Tick(ctx context.Context) error {
	_, err := c.TickResult(ctx)
	return err
}

// TickResult is Tick with its result.
func (c *Controller) TickResult(ctx context.Context) (TickResult, error) {
	d := c.Decider.Decide(ctx)
	res := TickResult{Decision: d}
	var errs []string
	sent, err := c.notify(ctx, d)
	res.NoticeSent = sent
	if err != nil {
		errs = append(errs, "notice: "+err.Error())
	}
	if c.Decider.Policy().Approval == providers.ModelApprovalAutomatic {
		if len(d.Selected) > 0 {
			res.AutoOutcomes = c.Decider.ApplyApproved(ctx, d.Selected, "automatic model retention ("+selectionReason(d)+")", "storage-manager model retention")
			if err := c.sendOutcome(ctx, "automatic", res.AutoOutcomes); err != nil {
				errs = append(errs, "outcome notice: "+err.Error())
			}
		}
	} else if err := c.reconcileAsk(ctx, d, &res); err != nil {
		errs = append(errs, "decision: "+err.Error())
	}
	if len(errs) > 0 {
		return res, fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return res, nil
}

// notify sends one notice per distinct situation per day while a filesystem
// is in a pressure band, a store is over budget, or a store is unobserved.
func (c *Controller) notify(ctx context.Context, d providers.ModelEvictionDecision) (bool, error) {
	over, broken := d.OverBudget(), d.Errors()
	if d.Band == providers.ModelBandOK && len(over) == 0 && len(broken) == 0 {
		return false, nil
	}
	if c.Notices == nil || c.Store == nil {
		return false, nil
	}
	title, body, urgency := noticeText(d, over, broken, c.Decider.Policy().CheckInterval)
	situation := []string{string(d.Band)}
	for _, s := range over {
		situation = append(situation, "over:"+s.Resource)
	}
	for _, s := range broken {
		situation = append(situation, "error:"+s.Resource)
	}
	for _, s := range d.Selected {
		situation = append(situation, providers.ModelItemID(s))
	}
	key := "model-retention:" + c.now().UTC().Format("2006-01-02") + ":" + shortHash(strings.Join(situation, "|"))
	changed, err := c.Store.SwapNoticeKey(ctx, key)
	if err != nil || !changed {
		return false, err
	}
	if err := c.Notices.Send(ctx, hubsender.Notice{Title: title, Body: body, Urgency: urgency, SensitivityLabel: "private", IdempotencyKey: key}); err != nil {
		_ = c.Store.ForgetNoticeKey(ctx)
		return false, err
	}
	return true, nil
}

func noticeText(d providers.ModelEvictionDecision, over, broken []providers.ModelStoreSummary, interval time.Duration) (string, string, string) {
	var parts []string
	urgency := "normal"
	for _, fs := range d.Filesystems {
		if fs.Band == providers.ModelBandOK {
			continue
		}
		parts = append(parts, fmt.Sprintf("The disk holding downloaded AI models is %.1f%% free (%s free).", fs.FreePercent, humanBytes(fs.FreeBytes)))
		if fs.Band == providers.ModelBandEvict {
			urgency = "high"
		}
	}
	for _, s := range over {
		parts = append(parts, fmt.Sprintf("%s models use %s, %s over their %s budget.", s.Resource, humanBytes(s.StoreBytes), humanBytes(s.OverBudgetBytes), humanBytes(s.BudgetBytes)))
	}
	for _, s := range broken {
		parts = append(parts, fmt.Sprintf("Storage Manager could not check %s models: %s", s.Resource, s.Error))
	}
	if len(d.Selected) > 0 {
		parts = append(parts, fmt.Sprintf("%d idle models (%s) can be removed; a decision with the list is waiting for you.", len(d.Selected), humanBytes(d.SelectedBytes)))
	} else {
		held := 0
		for _, c := range d.Candidates {
			if !c.Eligible {
				held++
			}
		}
		parts = append(parts, fmt.Sprintf("No model can be removed yet: all %d are in use, required, recently downloaded, or not yet tracked. Storage Manager checks again every %s.", held, intervalText(interval)))
	}
	title := "Downloaded AI models need disk attention"
	if d.Band == providers.ModelBandOK && len(over) > 0 {
		title = "A model store is over its disk budget"
	}
	return title, strings.Join(parts, " "), urgency
}

// selection is what one ask approves: exactly these identities.
type selection struct {
	Models []providers.ModelCandidate `json:"models"`
	Reason string                     `json:"reason"`
}

func selectionID(d providers.ModelEvictionDecision) string {
	ids := make([]string, 0, len(d.Selected))
	for _, c := range d.Selected {
		ids = append(ids, providers.ModelItemID(c)+"#"+fmt.Sprint(c.Bytes))
	}
	sort.Strings(ids)
	return "model-eviction-" + shortHash(strings.Join(ids, "|"))
}

func selectionReason(d providers.ModelEvictionDecision) string {
	var reasons []string
	for _, fs := range d.Filesystems {
		if fs.Band == providers.ModelBandEvict {
			reasons = append(reasons, fmt.Sprintf("%s is %.1f%% free", fs.Path, fs.FreePercent))
		}
	}
	for _, s := range d.OverBudget() {
		reasons = append(reasons, fmt.Sprintf("%s is %s over budget", s.Resource, humanBytes(s.OverBudgetBytes)))
	}
	return strings.Join(reasons, "; ")
}

// reconcileAsk keeps exactly one pending ask matching the current selection.
func (c *Controller) reconcileAsk(ctx context.Context, d providers.ModelEvictionDecision, res *TickResult) error {
	if c.Store == nil {
		return nil
	}
	pending, err := c.Store.Pending(ctx)
	if err != nil {
		return err
	}
	if len(d.Selected) == 0 {
		for _, ask := range pending {
			ok, err := c.Raiser.Withdraw(ctx, ask.ID, SubjectKey, "No downloaded model needs removing any more: disk space and store budgets are back within policy, or the models were used again.")
			if err != nil {
				return err
			}
			if ok {
				res.Withdrawn++
			}
		}
		return nil
	}
	id := selectionID(d)
	res.AskID = id
	existing, err := c.Store.Get(ctx, id)
	if err != nil {
		return err
	}
	if existing != nil && (existing.State == stateApproved || existing.State == stateRejected) {
		// The operator already answered this exact selection; a changed
		// selection is a new ask.
		return nil
	}
	now := c.now().UTC()
	for _, ask := range pending {
		if ask.ID == id {
			continue
		}
		// Superseded locally only: the Hub updates its one open decision for
		// SubjectKey in place when the new selection is raised below.
		ask.State = operatordecision.PendingStateWithdrawn
		ask.ResolvedAt = &now
		if err := c.Store.Put(ctx, &ask); err != nil {
			return err
		}
	}
	reason := selectionReason(d)
	encoded, err := json.Marshal(selection{Models: d.Selected, Reason: reason})
	if err != nil {
		return err
	}
	facts := DecisionFacts(d, id, reason)
	if err := facts.Validate(); err != nil {
		return fmt.Errorf("model eviction facts: %w", err)
	}
	raised, err := c.Raiser.Raise(ctx, operatordecision.PendingAsk{ID: id, Subject: SubjectKey, Extra: map[string]string{extraSelection: string(encoded)}}, facts)
	res.Raised = raised
	return err
}

// DecisionFacts is the operator-facing decision for one selection.
func DecisionFacts(d providers.ModelEvictionDecision, id, reason string) operatordecision.Facts {
	exact := true
	var list []string
	for _, c := range d.Selected {
		exact = exact && c.ReacquireExact
		list = append(list, fmt.Sprintf("- %s (%s): %s, unused for %s", c.Name, c.Resource, humanBytes(c.UniqueBytes), idleText(c.IdleFor)))
	}
	urgency := "normal"
	if d.Band == providers.ModelBandEvict {
		urgency = "high"
	}
	var reacquire []string
	for _, c := range d.Selected {
		reacquire = append(reacquire, fmt.Sprintf("- %s: %s", c.Name, strings.Join(c.Reacquire, " ")))
	}
	question := fmt.Sprintf("Storage Manager wants to remove these downloaded models because %s:\n%s\nEach can be downloaded again when needed.", reason, strings.Join(list, "\n"))
	bigPicture := "Downloaded AI models are the largest files Storage Manager can safely give back. It keeps every required model and anything used recently, and asks before removing the rest."
	if len(question) > operatordecision.MaxQuestion {
		question = fmt.Sprintf("Remove the %d idle downloaded models listed below to free %s? Each can be downloaded again when needed.", len(d.Selected), humanBytes(d.SelectedBytes))
		bigPicture += "\n\nWhy now: " + reason + "\n\n" + strings.Join(list, "\n")
	}
	details := "Each removal goes through the model store's own fenced eviction: it refuses a model that is loaded, has a request in flight, changed digest, or became required, and it writes the re-download command and expected size to the store's receipts first.\n\nRe-download commands:\n" + strings.Join(reacquire, "\n") +
		"\n\nDecision:\n" + strings.Join(providers.DecisionLines(d), "\n")
	return operatordecision.Facts{
		Subject:  fmt.Sprintf("Remove %d idle AI models to free %s?", len(d.Selected), humanBytes(d.SelectedBytes)),
		Question: question,
		Options: []operatordecision.Option{
			{Key: answerApprove, Label: "Remove these models"},
			{Key: answerDecline, Label: "Keep them"},
		},
		Recommended:          answerApprove,
		RecommendationReason: "None of these models is required, loaded, or used within the idle window, so removing them frees the space with no effect on current work. Each is removed only if it is still idle at that moment.",
		Reversible:           exact,
		Urgency:              urgency,
		BigPicture:           truncateRunes(bigPicture, operatordecision.MaxBigPictureCodePoints),
		TechnicalDetails:     truncateRunes(details, operatordecision.MaxTechnicalDetailsCodePoints),
		Correlation: map[string]string{correlationAskID: id},
		SubjectKey:  SubjectKey,
	}
}

// Resolver applies the operator's answer to a model-eviction decision.
type Resolver struct {
	Decider Decider
	Store   *FileStore
	Notices NoticeSender
	Now     func() time.Time
	// Async runs the evictions after the answer is recorded, so the webhook
	// answers promptly. Nil runs them inline.
	Async func(func())
	Logf  func(string, ...any)
}

// Handles reports whether the resolution belongs to a model-eviction decision.
func (r *Resolver) Handles(sourceEventType string) bool {
	return sourceEventType == operatordecision.DecisionRequestedEventType(Source, Kind)
}

// Apply records the answer and, on approval, evicts the approved models that
// are still eligible. Unknown, decided or withdrawn asks are a benign no-op.
func (r *Resolver) Apply(ctx context.Context, resolution operatordecision.DecisionResolution) (bool, error) {
	id := strings.TrimSpace(resolution.Correlation[correlationAskID])
	if id == "" || r.Store == nil {
		return true, nil
	}
	ask, err := r.Store.Get(ctx, id)
	if err != nil {
		return false, err
	}
	if ask == nil || ask.State != operatordecision.PendingStatePending {
		return true, nil
	}
	now := time.Now().UTC()
	if r.Now != nil {
		now = r.Now().UTC()
	}
	if resolution.State == operatordecision.StateWithdrawn {
		if resolution.SupersededByAskID == "" {
			ask.State = operatordecision.PendingStateWithdrawn
			ask.ResolvedAt = &now
			return true, r.Store.Put(ctx, ask)
		}
		return true, nil
	}
	approved := strings.EqualFold(strings.TrimSpace(resolution.Answer), answerApprove)
	ask.ResolvedAt = &now
	ask.ResolvedBy = resolution.AnsweredBy
	if ask.Extra == nil {
		ask.Extra = map[string]string{}
	}
	if !approved {
		ask.State = stateRejected
		ask.Extra[extraAnswer] = answerDecline
		return true, r.Store.Put(ctx, ask)
	}
	ask.State = stateApproved
	ask.Extra[extraAnswer] = answerApprove
	if err := r.Store.Put(ctx, ask); err != nil {
		return false, err
	}
	var sel selection
	if err := json.Unmarshal([]byte(ask.Extra[extraSelection]), &sel); err != nil {
		return true, fmt.Errorf("model eviction ask %s has an unreadable selection: %w", id, err)
	}
	evict := func() {
		outcomes := r.Decider.ApplyApproved(context.Background(), sel.Models, "operator approved decision "+id+" ("+sel.Reason+")", "operator via storage-manager decision "+id)
		if encoded, err := json.Marshal(outcomes); err == nil {
			ask.Extra[extraOutcome] = string(encoded)
			if err := r.Store.Put(context.Background(), ask); err != nil && r.Logf != nil {
				r.Logf("model eviction %s: record outcome: %v", id, err)
			}
		}
		c := Controller{Store: r.Store, Notices: r.Notices}
		if err := c.sendOutcome(context.Background(), id, outcomes); err != nil && r.Logf != nil {
			r.Logf("model eviction %s: outcome notice: %v", id, err)
		}
	}
	if r.Async != nil {
		r.Async(evict)
	} else {
		evict()
	}
	return true, nil
}

func (c *Controller) sendOutcome(ctx context.Context, id string, outcomes []providers.ModelEvictionOutcome) error {
	if c.Notices == nil || len(outcomes) == 0 {
		return nil
	}
	var freed int64
	var lines []string
	removed := 0
	for _, o := range outcomes {
		switch {
		case o.Evicted:
			removed++
			freed += o.FreedBytes
			lines = append(lines, fmt.Sprintf("Removed %s (%s); to bring it back: %s", o.Model, humanBytes(o.FreedBytes), o.Reacquire))
		case o.Refused != "":
			lines = append(lines, fmt.Sprintf("Kept %s: %s", o.Model, o.Refused))
		default:
			lines = append(lines, fmt.Sprintf("Could not remove %s: %s", o.Model, o.Error))
		}
	}
	urgency := "low"
	if removed < len(outcomes) {
		urgency = "normal"
	}
	return c.Notices.Send(ctx, hubsender.Notice{
		Title: fmt.Sprintf("Removed %d of %d idle AI models, freed %s", removed, len(outcomes), humanBytes(freed)),
		Body:  strings.Join(lines, "\n"), Urgency: urgency, SensitivityLabel: "private",
		IdempotencyKey: "model-eviction-outcome:" + id + ":" + shortHash(strings.Join(lines, "|")),
	})
}

func truncateRunes(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}

func shortHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])[:16]
}

// humanBytes uses binary units, the units store budgets are declared in.
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	value, exp := float64(n), 0
	for value >= unit && exp < 4 {
		value /= unit
		exp++
	}
	return fmt.Sprintf("%.1f %s", value, []string{"B", "KiB", "MiB", "GiB", "TiB"}[exp])
}

func intervalText(d time.Duration) string {
	if d%time.Hour == 0 {
		return fmt.Sprintf("%d hours", int(d.Hours()))
	}
	return fmt.Sprintf("%d minutes", int(d.Minutes()))
}

func idleText(d time.Duration) string {
	days := int(d.Hours() / 24)
	if days < 1 {
		return "less than a day"
	}
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}
