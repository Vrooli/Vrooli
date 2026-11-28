package coverage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/askreader"
	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
	"github.com/vrooli/api-core/retention"

	"data-backup-manager/internal/coverage"
)

// The standing rule for sensitive coverage.
//
// Data Backup Manager never adds a place that holds secrets without the
// operator's consent. A standing rule is one recorded consent that covers every
// future candidate: "back up anything that holds secrets from now on, and tell
// me what was added". The consent is the verified-human answer to the Hub
// decision that offered it, so the record carries who answered, through which
// verified identity, when, and which ask it was. Revoking it, or excluding one
// item from it, only ever reduces what is added, so both are open to the local
// API without a Hub round trip.

// ErrPolicyRequest marks a policy request the operator can fix and retry.
var ErrPolicyRequest = errors.New("sensitive coverage policy request refused")

const (
	// policyNoteInterval is how often the operator hears about what the rule
	// added. Backing up more is never urgent, so additions are batched.
	policyNoteInterval = 7 * 24 * time.Hour
	policyLedgerLimit  = 60
	policyAddedLimit   = 200
)

// PolicyAddition is one place the standing rule registered on its own.
type PolicyAddition struct {
	Owner string    `json:"owner"`
	Name  string    `json:"name"`
	Group string    `json:"group"`
	Bytes int64     `json:"bytes"`
	At    time.Time `json:"at"`
}

// declaredGroup labels additions that hold no secrets: targets an owner declared.
const declaredGroup = "Declared by their owner"

// PolicyExclusion is one item the rule must never add.
type PolicyExclusion struct {
	Owner  string    `json:"owner"`
	Name   string    `json:"name"`
	Reason string    `json:"reason"`
	By     string    `json:"by"`
	At     time.Time `json:"at"`
}

// PolicyLedgerEntry is one grant, revocation, exclusion or note, kept so the
// rule's history is visible.
type PolicyLedgerEntry struct {
	At     time.Time `json:"at"`
	Kind   string    `json:"kind"` // granted | revoked | excluded | included | added | noted
	By     string    `json:"by,omitempty"`
	Detail string    `json:"detail"`
}

// StandingPolicy is the persisted standing rule and everything needed to show it.
type StandingPolicy struct {
	Active     bool   `json:"active"`
	GrantAskID string `json:"grant_ask_id,omitempty"`
	// GrantBasis says how the consent was given: "option" for a button answer
	// that offered the rule, "free_text" for the operator's own words on the
	// decision, taken as the instruction. GrantText is that text, verbatim.
	GrantBasis      string              `json:"grant_basis,omitempty"`
	GrantText       string              `json:"grant_text,omitempty"`
	GrantedBy       string              `json:"granted_by,omitempty"`
	GrantedByKind   string              `json:"granted_by_identity_kind,omitempty"`
	GrantedBySource string              `json:"granted_by_identity_source,omitempty"`
	GrantedAt       *time.Time          `json:"granted_at,omitempty"`
	RevokedBy       string              `json:"revoked_by,omitempty"`
	RevokedAt       *time.Time          `json:"revoked_at,omitempty"`
	Excluded        []PolicyExclusion   `json:"excluded,omitempty"`
	Unreported      []PolicyAddition    `json:"unreported,omitempty"`
	AddedTotal      int                 `json:"added_total"`
	LastNoteAt      *time.Time          `json:"last_note_at,omitempty"`
	LastSweepAt     *time.Time          `json:"last_sweep_at,omitempty"`
	Ledger          []PolicyLedgerEntry `json:"ledger,omitempty"`
}

func (p *StandingPolicy) excluded(owner, name string) bool {
	for _, e := range p.Excluded {
		if e.Owner == owner && e.Name == name {
			return true
		}
	}
	return false
}

func (p *StandingPolicy) log(at time.Time, kind, by, detail string) {
	p.Ledger = append(p.Ledger, PolicyLedgerEntry{At: at, Kind: kind, By: by, Detail: detail})
	if over := len(p.Ledger) - policyLedgerLimit; over > 0 {
		p.Ledger = append([]PolicyLedgerEntry(nil), p.Ledger[over:]...)
	}
}

// PolicyStore persists the standing rule as one JSON file.
type PolicyStore struct {
	Path string
	mu   sync.Mutex
}

func (s *PolicyStore) valid() error {
	if s == nil || !filepath.IsAbs(s.Path) {
		return errors.New("sensitive coverage policy store needs an absolute path")
	}
	return nil
}

// Load returns the stored rule, or an inactive empty one when none was ever set.
func (s *PolicyStore) Load() (StandingPolicy, error) {
	if err := s.valid(); err != nil {
		return StandingPolicy{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load()
}

func (s *PolicyStore) load() (StandingPolicy, error) {
	data, err := os.ReadFile(s.Path)
	if os.IsNotExist(err) {
		return StandingPolicy{}, nil
	}
	if err != nil {
		return StandingPolicy{}, err
	}
	var p StandingPolicy
	if err := json.Unmarshal(data, &p); err != nil {
		return StandingPolicy{}, fmt.Errorf("read sensitive coverage policy: %w", err)
	}
	if err := p.validate(); err != nil {
		return StandingPolicy{}, fmt.Errorf("read sensitive coverage standing policy: %w", err)
	}
	return p, nil
}

func (p StandingPolicy) validate() error {
	if !p.Active {
		return nil
	}
	if strings.TrimSpace(p.GrantAskID) == "" {
		return errors.New("active rule is missing its verified decision id")
	}
	if strings.TrimSpace(p.GrantedBy) == "" || !isVerifiedHumanKind(p.GrantedByKind) || strings.TrimSpace(p.GrantedBySource) == "" {
		return errors.New("active rule is missing verified human identity evidence")
	}
	if p.GrantedAt == nil || p.GrantedAt.IsZero() {
		return errors.New("active rule is missing its consent time")
	}
	switch p.GrantBasis {
	case "option":
	case "free_text":
		if strings.TrimSpace(p.GrantText) == "" {
			return errors.New("free-text consent is missing the operator's words")
		}
	default:
		return fmt.Errorf("active rule has unknown consent basis %q", p.GrantBasis)
	}
	return nil
}

// Update loads the rule, lets fn change it and persists the result atomically.
// fn returning an error leaves the stored rule untouched.
func (s *PolicyStore) Update(fn func(*StandingPolicy) error) (StandingPolicy, error) {
	if err := s.valid(); err != nil {
		return StandingPolicy{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.load()
	if err != nil {
		return StandingPolicy{}, err
	}
	if err := fn(&p); err != nil {
		return p, err
	}
	if err := p.validate(); err != nil {
		return p, fmt.Errorf("write sensitive coverage standing policy: %w", err)
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return StandingPolicy{}, err
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return StandingPolicy{}, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.Path), ".policy-*.tmp")
	if err != nil {
		return StandingPolicy{}, err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return StandingPolicy{}, err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return StandingPolicy{}, err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return StandingPolicy{}, err
	}
	if err := tmp.Close(); err != nil {
		return StandingPolicy{}, err
	}
	if err := os.Rename(tmp.Name(), s.Path); err != nil {
		return StandingPolicy{}, err
	}
	return p, nil
}

// Policy is the rule as the operator sees it.
func (f *SensitiveDecisionFlow) Policy(context.Context) (StandingPolicy, error) {
	if f == nil || f.policy == nil {
		return StandingPolicy{}, errors.New("sensitive coverage policy is not configured")
	}
	return f.policy.Load()
}

// Revoke ends the standing rule. Items it already added stay registered; new
// candidates wait for a decision again.
func (f *SensitiveDecisionFlow) Revoke(_ context.Context, by string) (StandingPolicy, error) {
	if f == nil || f.policy == nil {
		return StandingPolicy{}, errors.New("sensitive coverage policy is not configured")
	}
	by = strings.TrimSpace(by)
	if by == "" {
		by = "local API caller"
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	now := f.now().UTC()
	return f.policy.Update(func(p *StandingPolicy) error {
		if !p.Active {
			return fmt.Errorf("%w: there is no standing rule to revoke", ErrPolicyRequest)
		}
		p.Active, p.RevokedBy, p.RevokedAt = false, by, &now
		p.log(now, "revoked", by, "Standing rule turned off. New places that hold secrets wait for a decision again; what it already added stays backed up.")
		return nil
	})
}

// Exclude keeps one item out of the standing rule, for example one the operator
// does not want copied. The item is not removed if it is already registered.
func (f *SensitiveDecisionFlow) Exclude(_ context.Context, owner, name, reason, by string) (StandingPolicy, error) {
	if f == nil || f.policy == nil {
		return StandingPolicy{}, errors.New("sensitive coverage policy is not configured")
	}
	owner, name, reason, by = strings.TrimSpace(owner), strings.TrimSpace(name), strings.TrimSpace(reason), strings.TrimSpace(by)
	if owner == "" || name == "" {
		return StandingPolicy{}, fmt.Errorf("%w: an exclusion needs the item's owner and name", ErrPolicyRequest)
	}
	if reason == "" {
		return StandingPolicy{}, fmt.Errorf("%w: an exclusion needs a reason", ErrPolicyRequest)
	}
	if by == "" {
		by = "local API caller"
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	now := f.now().UTC()
	return f.policy.Update(func(p *StandingPolicy) error {
		if p.excluded(owner, name) {
			return nil
		}
		p.Excluded = append(p.Excluded, PolicyExclusion{Owner: owner, Name: name, Reason: reason, By: by, At: now})
		p.log(now, "excluded", by, fmt.Sprintf("%s / %s is kept out of the standing rule: %s", owner, name, reason))
		return nil
	})
}

// grant records the operator's consent from a verified Decision answer. It is
// idempotent for the same ask, so a redelivered answer changes nothing.
func (f *SensitiveDecisionFlow) grant(p *StandingPolicy, askID, by, kind, source string, at time.Time, excluded []PolicyExclusion) {
	if p.Active && p.GrantAskID == askID {
		return
	}
	p.Active, p.GrantAskID, p.GrantBasis, p.GrantText = true, askID, "option", ""
	p.GrantedBy, p.GrantedByKind, p.GrantedBySource = by, kind, source
	p.GrantedAt, p.RevokedBy, p.RevokedAt = &at, "", nil
	for _, e := range excluded {
		if !p.excluded(e.Owner, e.Name) {
			p.Excluded = append(p.Excluded, e)
		}
	}
	// The grant starts the note clock: what the answer itself added is in the
	// answer's own result notice, so the first weekly note covers what comes after.
	p.LastNoteAt = &at
	p.log(at, "granted", by, "Standing rule turned on: anything new that holds secrets is backed up automatically and listed in a weekly note.")
}

// SweepResult is what one pass of the standing rule did.
type SweepResult struct {
	Active     bool
	Registered []PolicyAddition
	Failed     int
	NoteSent   bool
}

// Sweep applies the standing rule to the current candidates: it registers each
// one the rule covers (never an excluded one), remembers what it added, and, at
// most weekly, sends one note listing the additions. With no active rule it does
// nothing, so a place that holds secrets is never added without consent.
func (f *SensitiveDecisionFlow) Sweep(ctx context.Context) (SweepResult, error) {
	if f == nil || f.service == nil || f.policy == nil {
		return SweepResult{}, errors.New("sensitive coverage policy is not configured")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.sweepLocked(ctx)
}

func (f *SensitiveDecisionFlow) sweepLocked(ctx context.Context) (SweepResult, error) {
	policy, err := f.policy.Load()
	if err != nil {
		return SweepResult{}, err
	}
	if !policy.Active {
		return SweepResult{}, nil
	}
	result := SweepResult{Active: true}
	report, err := f.service.Report(ctx)
	if err != nil {
		return result, fmt.Errorf("read sensitive coverage candidates: %w", err)
	}
	now := f.now().UTC()
	covered := make([]string, 0, len(report.Sensitive))
	bySuggestion := map[string]PolicyAddition{}
	for _, c := range report.Sensitive {
		if policy.excluded(c.Owner, c.Name) {
			continue
		}
		covered = append(covered, c.ID)
		bySuggestion[c.ID] = PolicyAddition{Owner: c.Owner, Name: c.Name, Group: groupTitle(c.OwnerKind), Bytes: c.ApproxBytes, At: now}
	}
	if len(covered) > 0 {
		accepted, acceptErr := f.service.AcceptSensitive(ctx, covered)
		if acceptErr != nil {
			// Nothing was registered; the next sweep tries again.
			f.logger.Printf("sensitive coverage standing rule: %v", acceptErr)
			result.Failed = len(covered)
		} else {
			result.Failed = len(accepted.Failed)
			for _, a := range accepted.Accepted {
				if added, ok := bySuggestion[a.SuggestionID]; ok {
					result.Registered = append(result.Registered, added)
				}
			}
		}
	}
	// The same answer covers what owners declare without secrets in it: a target
	// an owner declares as durable storage is registered, never left for someone
	// to remember to accept. Undeclared discoveries are not in this list.
	declared := map[string]PolicyAddition{}
	for _, c := range report.Recommended {
		declared[c.ID] = PolicyAddition{Owner: c.Owner, Name: c.Name, Group: declaredGroup, Bytes: c.ApproxBytes, At: now}
	}
	if len(declared) > 0 {
		accepted, acceptErr := f.service.AcceptDefaults(ctx, coverage.AcceptOptions{})
		if acceptErr != nil {
			f.logger.Printf("sensitive coverage standing rule (declared targets): %v", acceptErr)
			result.Failed += len(declared)
		} else {
			result.Failed += len(accepted.Failed)
			for _, a := range accepted.Accepted {
				if added, ok := declared[a.SuggestionID]; ok {
					result.Registered = append(result.Registered, added)
				}
			}
		}
	}
	sort.Slice(result.Registered, func(i, j int) bool { return result.Registered[i].Bytes > result.Registered[j].Bytes })
	sendNote := false
	var note noteBody
	updated, err := f.policy.Update(func(p *StandingPolicy) error {
		if !p.Active {
			return nil
		}
		p.LastSweepAt = &now
		p.Unreported = append(p.Unreported, result.Registered...)
		if over := len(p.Unreported) - policyAddedLimit; over > 0 {
			p.Unreported = append([]PolicyAddition(nil), p.Unreported[over:]...)
		}
		p.AddedTotal += len(result.Registered)
		for _, a := range result.Registered {
			p.log(now, "added", "standing rule", fmt.Sprintf("%s / %s (%s) backed up under the standing rule.", a.Owner, a.Name, retention.FormatBytes(a.Bytes)))
		}
		if len(p.Unreported) > 0 && (p.LastNoteAt == nil || now.Sub(*p.LastNoteAt) >= policyNoteInterval) {
			sendNote, note = true, standingNote(p.Unreported, now)
		}
		return nil
	})
	if err != nil {
		return result, err
	}
	if sendNote && f.publisher != nil {
		if err := f.publishNote(ctx, note); err != nil {
			f.logger.Printf("sensitive coverage standing rule note: %v", err)
			return result, nil
		}
		result.NoteSent = true
		_, err = f.policy.Update(func(p *StandingPolicy) error {
			p.Unreported, p.LastNoteAt = nil, &now
			p.log(now, "noted", "standing rule", fmt.Sprintf("Weekly note sent listing %d additions.", len(updated.Unreported)))
			return nil
		})
	}
	return result, err
}

// noteBody is the weekly note about what the standing rule added.
type noteBody struct{ title, body, dedupe string }

const sensitiveNoteEvent = "data_backup_manager.sensitive_coverage.notification.v1"

func standingNote(added []PolicyAddition, at time.Time) noteBody {
	var total int64
	for _, a := range added {
		total += a.Bytes
	}
	secrets := 0
	for _, a := range added {
		if a.Group != declaredGroup {
			secrets++
		}
	}
	what := "places that hold secrets"
	switch {
	case secrets == 0:
		what = "places their owners declared"
	case secrets < len(added):
		what = fmt.Sprintf("places (%d hold secrets, the rest are declared by their owners)", secrets)
	}
	title := fmt.Sprintf("Backed up %d new %s", len(added), what)
	if len(added) == 1 {
		title = "Backed up 1 new place"
		if secrets == 1 {
			title = "Backed up 1 new place that holds secrets"
		}
	}
	lines := make([]string, 0, len(added)+3)
	lines = append(lines, fmt.Sprintf("Your standing rule added %d new %s to your backups (%s in all):", len(added), what, retention.FormatBytes(total)))
	shown := added
	if len(shown) > 12 {
		shown = shown[:12]
	}
	for _, a := range shown {
		lines = append(lines, fmt.Sprintf("- %s / %s (%s)", ownerName(a.Owner), a.Name, retention.FormatBytes(a.Bytes)))
	}
	if len(added) > len(shown) {
		lines = append(lines, fmt.Sprintf("- and %d more", len(added)-len(shown)))
	}
	lines = append(lines, "To keep one out: data-backup-manager coverage exclude-sensitive --owner OWNER --name NAME --reason WHY. To turn the rule off: data-backup-manager coverage revoke-sensitive-policy.")
	return noteBody{title: title, body: strings.Join(lines, "\n"), dedupe: "sensitive-coverage-weekly-" + at.Format("2006-01-02")}
}

func (f *SensitiveDecisionFlow) publishNote(ctx context.Context, note noteBody) error {
	return f.publisher.PublishDomainEvent(ctx, eventbus.DomainEvent{
		Source: "data-backup-manager", EventType: sensitiveNoteEvent, Occurred: f.now().UTC(),
		Payload: map[string]any{
			"title": note.title, "body": note.body, "urgency": "low", "sensitivity": "private",
			"category": "backup", "dedupe_key": note.dedupe,
		},
	})
}

// Hub ask facts an adopted consent must show. The decision is DBM's own, about
// the sensitive coverage subject, answered and not defaulted.
const sensitiveSubjectKey = "dbm-sensitive-coverage"

// ErrConsentNotAdoptable marks an answer that cannot stand as the operator's
// consent: nothing is recorded and nothing is registered.
var ErrConsentNotAdoptable = errors.New("that Hub answer cannot be recorded as the standing consent")

// AdoptAnswerAsConsent records the operator's free-text answer on a sensitive
// coverage decision as the standing consent, and applies it. Only the Hub's own
// record counts: the ask is read back as DBM's ask-reader principal and must be
// this scenario's decision about sensitive coverage, answered by a verified
// human in their own words (Other). The consent keeps the Hub ask id, the answer
// time, the verified identity and the operator's text verbatim, and is revocable
// like any standing rule. Stale local records of the review are closed without
// raising a new Hub ask.
func (f *SensitiveDecisionFlow) AdoptAnswerAsConsent(ctx context.Context, hubAskID string) (StandingPolicy, SweepResult, error) {
	if f == nil || f.policy == nil || f.service == nil {
		return StandingPolicy{}, SweepResult{}, errors.New("sensitive coverage policy is not configured")
	}
	if f.reader == nil {
		return StandingPolicy{}, SweepResult{}, errors.New("the Hub ask reader is not configured; the answer cannot be confirmed with the Hub")
	}
	hubAskID = strings.TrimSpace(hubAskID)
	if hubAskID == "" {
		return StandingPolicy{}, SweepResult{}, fmt.Errorf("%w: the Hub ask id is required", ErrPolicyRequest)
	}
	ask, err := f.reader.ReadAsk(ctx, hubAskID)
	if err != nil {
		return StandingPolicy{}, SweepResult{}, fmt.Errorf("confirm the answer with the Hub: %w", err)
	}
	switch {
	case ask.Source != "data-backup-manager" || ask.SubjectKey != sensitiveSubjectKey:
		return StandingPolicy{}, SweepResult{}, fmt.Errorf("%w: it is not this scenario's sensitive coverage decision", ErrConsentNotAdoptable)
	case ask.State != operatordecision.StateAnswered:
		return StandingPolicy{}, SweepResult{}, fmt.Errorf("%w: the decision is %s, not answered", ErrConsentNotAdoptable, ask.State)
	case !isVerifiedHumanKind(ask.AnsweredByIdentityKind), strings.TrimSpace(ask.AnsweredBy) == "", strings.TrimSpace(ask.AnsweredByIdentitySource) == "":
		return StandingPolicy{}, SweepResult{}, fmt.Errorf("%w: it was not answered by a verified human (the Hub records identity kind %q, source %q, answerer present: %t)",
			ErrConsentNotAdoptable, ask.AnsweredByIdentityKind, ask.AnsweredByIdentitySource, strings.TrimSpace(ask.AnsweredBy) != "")
	case ask.Answer != "other" || strings.TrimSpace(ask.Note) == "":
		return StandingPolicy{}, SweepResult{}, fmt.Errorf("%w: only the operator's own words on the decision can be taken as the instruction; a menu answer has its own meaning", ErrConsentNotAdoptable)
	}
	answeredAt := f.now().UTC()
	if parsed, perr := time.Parse(time.RFC3339Nano, ask.AnsweredAt); perr == nil {
		answeredAt = parsed.UTC()
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, err := f.policy.Update(func(p *StandingPolicy) error {
		if p.Active && p.GrantAskID == hubAskID {
			return nil
		}
		f.grant(p, hubAskID, ask.AnsweredBy, ask.AnsweredByIdentityKind, ask.AnsweredByIdentitySource, answeredAt, nil)
		p.GrantBasis, p.GrantText = "free_text", ask.Note
		// What the answer adds now is reported in the first note, not held for a week.
		p.LastNoteAt = nil
		return nil
	}); err != nil {
		return StandingPolicy{}, SweepResult{}, fmt.Errorf("record the standing consent: %w", err)
	}
	if err := f.closeReviewsLocked(ctx, ask, answeredAt); err != nil {
		return StandingPolicy{}, SweepResult{}, err
	}
	swept, err := f.sweepLocked(ctx)
	policy, _ := f.policy.Load()
	return policy, swept, err
}

// closeReviewsLocked closes every local record of a sensitive review that is
// still pending, because the operator has answered it. No Hub ask is raised.
func (f *SensitiveDecisionFlow) closeReviewsLocked(ctx context.Context, ask askreader.Ask, at time.Time) error {
	asks, err := f.tasks.ListAsks(ctx)
	if err != nil {
		return fmt.Errorf("list local sensitive reviews: %w", err)
	}
	for _, task := range asks {
		if task == nil || task.Subject != "sensitive-coverage" || task.State != operatordecision.PendingStatePending {
			continue
		}
		task.State, task.ResolvedAt, task.ResolvedBy = operatordecision.PendingStateApproved, &at, ask.AnsweredBy
		task.Extra["result"] = "The operator answered in their own words; DBM recorded the standing rule from that answer. Nothing was registered under this review's own menu."
		task.Extra["result_notice_pending"] = "false"
		if err := f.tasks.Put(ctx, task); err != nil {
			return fmt.Errorf("close local sensitive review %s: %w", task.ID, err)
		}
	}
	return nil
}
