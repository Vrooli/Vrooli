package coverage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"data-backup-manager/internal/attention"
	"data-backup-manager/internal/coverage"
	"data-backup-manager/internal/destinationreadiness"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
)

const (
	sensitiveDecisionKind = "sensitive_coverage_review"
	sensitiveResultEvent  = "data_backup_manager.sensitive_coverage.result.v1"
)

type sensitiveChoice struct {
	SuggestionID             string `json:"suggestion_id"`
	Decision                 string `json:"decision"`
	Reason                   string `json:"reason,omitempty"`
	OwnerCommand             string `json:"owner_reacquisition_command,omitempty"`
	OwnerEvidence            string `json:"owner_reacquisition_evidence,omitempty"`
	OwnerNeedsAuthorization  bool   `json:"owner_reacquisition_needs_operator,omitempty"`
	OwnerAuthorizationReason string `json:"owner_reacquisition_authorization_reason,omitempty"`
}

type sensitiveSnapshot struct {
	ID                         string `json:"id"`
	Owner                      string `json:"owner"`
	OwnerKind                  string `json:"owner_kind,omitempty"`
	Name                       string `json:"name"`
	Locator                    string `json:"locator"`
	Kind                       string `json:"kind"`
	Critical                   bool   `json:"critical"`
	Warning                    string `json:"warning"`
	ReacquisitionCommand       string `json:"reacquisition_command,omitempty"`
	ReacquisitionEvidence      string `json:"reacquisition_evidence,omitempty"`
	ReacquisitionNeedsOperator bool   `json:"reacquisition_needs_operator,omitempty"`
	ReacquisitionAuthReason    string `json:"reacquisition_auth_reason,omitempty"`
}

// SensitiveDecisionFlow owns the one batched, human-verified review of the
// current sensitive coverage suggestions. Ask records and outcomes are durable;
// no sensitive target is registered unless the signed callback approves it.
type SensitiveDecisionFlow struct {
	service   coverage.Service
	tasks     *destinationreadiness.FileDecisionStore
	publisher operatordecision.Publisher
	policy    *PolicyStore
	reader    operatordecision.AskReader
	enabled   bool
	logger    *log.Logger
	now       func() time.Time
	readFile  func(string) ([]byte, error)
	mu        sync.Mutex
}

func NewSensitiveDecisionFlow(service coverage.Service, tasks *destinationreadiness.FileDecisionStore, publisher operatordecision.Publisher, enabled bool, logger *log.Logger) *SensitiveDecisionFlow {
	if logger == nil {
		logger = log.Default()
	}
	flow := &SensitiveDecisionFlow{service: service, tasks: tasks, publisher: publisher, enabled: enabled, logger: logger, now: time.Now, readFile: os.ReadFile}
	if tasks != nil && filepath.IsAbs(tasks.Root) {
		// The standing rule sits beside the ask ledger, never inside it: the ledger
		// treats every file in its directory as an ask.
		flow.policy = &PolicyStore{Path: filepath.Join(filepath.Dir(filepath.Clean(tasks.Root)), "sensitive-coverage-policy", "policy.json")}
	}
	return flow
}

// WithAskReader gives the flow the Hub read-back it needs to adopt an answer as
// the standing consent.
func (f *SensitiveDecisionFlow) WithAskReader(reader operatordecision.AskReader) *SensitiveDecisionFlow {
	f.reader = reader
	return f
}

func (f *SensitiveDecisionFlow) Ready() bool {
	return f != nil && f.enabled && f.service != nil && f.tasks != nil && f.publisher != nil
}

func (f *SensitiveDecisionFlow) Request(ctx context.Context) (string, int, error) {
	if !f.Ready() {
		return "", 0, errors.New("verified Notification Hub callback is not configured")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.policy != nil {
		standing, err := f.policy.Load()
		if err != nil {
			return "", 0, fmt.Errorf("read sensitive coverage standing policy: %w", err)
		}
		if standing.Active {
			// The operator already consented to everything that holds secrets, so there
			// is nothing to ask: apply the rule and report what it covered.
			swept, err := f.sweepLocked(ctx)
			if err != nil {
				return "", 0, fmt.Errorf("apply the standing sensitive coverage rule: %w", err)
			}
			return "", len(swept.Registered), nil
		}
	}
	report, err := f.service.Report(ctx)
	if err != nil {
		return "", 0, fmt.Errorf("read sensitive coverage candidates: %w", err)
	}
	candidates := report.Sensitive
	if len(candidates) == 0 {
		return "", 0, coverage.ErrNoSensitiveSuggestions
	}
	snapshots := snapshotCandidates(candidates)
	fingerprint, err := candidateFingerprint(snapshots)
	if err != nil {
		return "", 0, err
	}
	priorAsks, err := f.tasks.ListAsks(ctx)
	if err != nil {
		return "", 0, fmt.Errorf("list prior sensitive coverage decisions: %w", err)
	}
	generation := 0
	var pendingSame *operatordecision.PendingAsk
	for _, prior := range priorAsks {
		if prior == nil || prior.Subject != "sensitive-coverage" || prior.Extra["candidate_fingerprint"] != fingerprint {
			continue
		}
		priorGeneration, _ := strconv.Atoi(prior.Extra["generation"])
		if priorGeneration >= generation {
			generation = priorGeneration + 1
		}
		if prior.State == operatordecision.PendingStatePending && (pendingSame == nil || prior.RaisedAt.After(pendingSame.RaisedAt)) {
			pendingSame = prior
		}
	}
	if pendingSame != nil {
		generation, _ = strconv.Atoi(pendingSame.Extra["generation"])
	}
	genText := strconv.Itoa(generation)
	askID := operatordecision.CorrelationID("dbm-sensitive-coverage", fingerprint, genText)
	var earlier []time.Time
	for _, prior := range priorAsks {
		if prior != nil && prior.Subject == "sensitive-coverage" && prior.ID != askID && !prior.RaisedAt.IsZero() {
			earlier = append(earlier, prior.RaisedAt)
		}
	}
	sort.Slice(earlier, func(i, j int) bool { return earlier[i].Before(earlier[j]) })
	facts := sensitiveReviewFacts(candidates, snapshots, askID, fingerprint, genText, attention.Digest(f.readFile, 6), f.now().Add(7*24*time.Hour), earlier)
	if err := facts.Validate(); err != nil {
		return "", 0, fmt.Errorf("validate sensitive coverage decision: %w", err)
	}
	raiser := operatordecision.Raiser{Source: "data-backup-manager", Kind: sensitiveDecisionKind, Store: f.tasks, Publisher: f.publisher, Clock: sensitiveFlowClock{f}, MaxOutstanding: 1, Window: 7 * 24 * time.Hour}
	// A review for a changed candidate set revises the open decision under the same
	// subject key: the Hub updates it in place when the answer menu is unchanged and
	// otherwise closes it as superseded by this one, so the operator never holds two.
	// Locally the stale reviews close first (the raiser allows one outstanding ask),
	// and reopen if the new one cannot be raised.
	var closed []*operatordecision.PendingAsk
	reopen := func() {
		for _, prior := range closed {
			prior.State, prior.ResolvedAt = operatordecision.PendingStatePending, nil
			delete(prior.Extra, "result")
			if err := f.tasks.Put(ctx, prior); err != nil {
				f.logger.Printf("sensitive coverage: reopen stale review %s: %v", prior.ID, err)
			}
		}
	}
	for _, prior := range priorAsks {
		if prior == nil || prior.Subject != "sensitive-coverage" || prior.ID == askID || prior.State != operatordecision.PendingStatePending {
			continue
		}
		now := f.now().UTC()
		prior.State, prior.ResolvedAt = operatordecision.PendingStateWithdrawn, &now
		prior.Extra["result"] = "Candidate set changed; no target was registered under this review."
		if err := f.tasks.Put(ctx, prior); err != nil {
			reopen()
			return "", 0, fmt.Errorf("record stale sensitive coverage review withdrawal: %w", err)
		}
		closed = append(closed, prior)
	}
	ask := operatordecision.PendingAsk{ID: askID, Subject: "sensitive-coverage", Extra: map[string]string{"candidate_fingerprint": fingerprint, "generation": genText}}
	if _, err := raiser.Raise(ctx, ask, facts); err != nil {
		reopen()
		return "", 0, fmt.Errorf("raise sensitive coverage decision: %w", err)
	}
	return askID, len(snapshots), nil
}

type sensitiveFlowClock struct{ flow *SensitiveDecisionFlow }

func (c sensitiveFlowClock) Now() time.Time { return c.flow.now() }

func (f *SensitiveDecisionFlow) Handles(eventType string) bool {
	return eventType == operatordecision.DecisionRequestedEventType("data-backup-manager", sensitiveDecisionKind)
}

func (f *SensitiveDecisionFlow) Apply(ctx context.Context, resolution operatordecision.DecisionResolution) (bool, error) {
	if !f.Ready() || !f.Handles(resolution.SourceEventType) {
		return false, nil
	}
	askID := strings.TrimSpace(resolution.AskID)
	if askID == "" {
		return true, nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	task, err := f.tasks.Get(ctx, askID)
	if err != nil {
		return true, err
	}
	if task == nil || task.Subject != "sensitive-coverage" || (task.State != operatordecision.PendingStatePending && task.State != "applying") {
		if task != nil && task.Extra["result_notice_pending"] == "true" {
			return true, f.publishResult(ctx, task)
		}
		return true, nil
	}
	if task.State == operatordecision.PendingStatePending && !verifiedSensitiveApproval(resolution) {
		return true, f.finishWithoutApply(ctx, task, resolution, "No verified human approval was received. Sensitive targets were left unchanged.")
	}
	report, err := f.service.Report(ctx)
	if err != nil {
		return true, f.finishWithoutApply(ctx, task, resolution, "DBM could not revalidate current suggestions; no target was registered.")
	}
	current := snapshotCandidates(report.Sensitive)
	fingerprint, err := candidateFingerprint(current)
	if err != nil {
		return true, err
	}
	if fingerprint != task.Extra["candidate_fingerprint"] || fingerprint != resolution.Correlation["candidate_fingerprint"] {
		return true, f.finishWithoutApply(ctx, task, resolution, "The sensitive coverage candidates changed while the decision was pending. No target was registered; request a new review.")
	}
	if resolution.Answer == answerLater {
		return true, f.finishWithoutApply(ctx, task, resolution, "You chose not now. No target was registered; request a new review when ready.")
	}
	choices, plan, err := choicesForResolution(resolution, current, f.now().UTC())
	if err != nil {
		return true, f.finishWithoutApply(ctx, task, resolution, "The answer did not provide a complete, valid disposition for every target. No target was registered. "+err.Error())
	}
	protected := make([]string, 0, len(choices))
	for _, choice := range choices {
		if choice.Decision == "protect" {
			protected = append(protected, choice.SuggestionID)
		}
	}
	// Persist an applying claim before effects. If the process stops mid-batch,
	// a signed redelivery resumes through idempotent target upserts, while the
	// mutex prevents two live callbacks from racing.
	task.State, task.ResolvedBy = "applying", resolution.AnsweredBy
	encoded, _ := json.Marshal(choices)
	task.Extra["decisions"] = string(encoded)
	if err := f.tasks.Put(ctx, task); err != nil {
		return true, err
	}
	result := coverage.AcceptResult{}
	if len(protected) > 0 {
		result, err = f.service.AcceptSensitive(ctx, protected)
	}
	if err != nil {
		return true, err
	}
	now := f.now().UTC()
	if plan.standing {
		// The verified answer is the consent. Recording it after the batch lets a
		// redelivered answer resume here, and recording it at all means a later
		// sweep, not another question, covers anything that holds secrets.
		if f.policy == nil {
			return true, errors.New("the standing rule cannot be recorded: no policy store is configured")
		}
		answeredAt := now
		if parsed, perr := time.Parse(time.RFC3339, resolution.AnsweredAt); perr == nil {
			answeredAt = parsed.UTC()
		}
		if _, err := f.policy.Update(func(p *StandingPolicy) error {
			f.grant(p, task.ID, resolution.AnsweredBy, resolution.AnsweredByIdentityKind, resolution.AnsweredByIdentitySource, answeredAt, plan.excluded)
			return nil
		}); err != nil {
			return true, fmt.Errorf("record the standing rule: %w", err)
		}
	}
	task.State, task.ResolvedAt = operatordecision.PendingStateApproved, &now
	task.Extra["result"] = summarizeSensitiveResult(choices, result, plan.standing)
	task.Extra["result_notice_pending"] = "true"
	if err := f.tasks.Put(ctx, task); err != nil {
		return true, err
	}
	return true, f.publishResult(ctx, task)
}

func verifiedSensitiveApproval(res operatordecision.DecisionResolution) bool {
	return res.State == operatordecision.StateAnswered && !res.ByDefault && strings.TrimSpace(res.AnsweredBy) != "" && strings.TrimSpace(res.AnsweredByIdentitySource) != "" && isVerifiedHumanKind(res.AnsweredByIdentityKind)
}

// isVerifiedHumanKind reports the identity kind the Hub records for a person it
// verified (the same kind the drive-recovery decisions require).
func isVerifiedHumanKind(kind string) bool {
	return strings.EqualFold(strings.TrimSpace(kind), "verified_human")
}

// resolutionPlan is what an answer asks for beyond its per-item choices.
type resolutionPlan struct {
	// standing records the operator's consent to cover future candidates too.
	standing bool
	// excluded are items a standing answer keeps out of the rule.
	excluded []PolicyExclusion
}

// otherAnswer is the object form of Other: per-item decisions plus an optional
// standing rule. The bare JSON array stays valid and means no standing rule.
type otherAnswer struct {
	Standing  bool              `json:"standing"`
	Decisions []sensitiveChoice `json:"decisions"`
}

const leftOutOfGroupsReason = "Left out when choosing which groups to back up."

func choicesForResolution(res operatordecision.DecisionResolution, candidates []sensitiveSnapshot, now time.Time) ([]sensitiveChoice, resolutionPlan, error) {
	var plan resolutionPlan
	choices := make([]sensitiveChoice, 0, len(candidates))
	switch {
	case res.Answer == answerStanding:
		plan.standing = true
		for _, candidate := range candidates {
			choices = append(choices, sensitiveChoice{SuggestionID: candidate.ID, Decision: "protect"})
		}
	case res.Answer == answerProtectAll || res.Answer == answerReacquireAll:
		action := strings.TrimSuffix(res.Answer, "_all")
		for _, candidate := range candidates {
			choices = append(choices, sensitiveChoice{SuggestionID: candidate.ID, Decision: action})
		}
	case res.Answer == answerAsRecommended:
		for _, candidate := range candidates {
			choices = append(choices, sensitiveChoice{SuggestionID: candidate.ID, Decision: recommendedDisposition(candidate)})
		}
	case res.Answer == answerGroups:
		response, _ := res.Response(groupsQuestionID)
		kinds := map[string]bool{}
		for _, key := range response.Selected {
			kinds[key] = true
		}
		if len(kinds) == 0 {
			return nil, plan, errors.New("no group was switched on")
		}
		matched := 0
		for _, candidate := range candidates {
			if kinds[groupKey(candidate.OwnerKind)] {
				matched++
				choices = append(choices, sensitiveChoice{SuggestionID: candidate.ID, Decision: "protect"})
			} else {
				choices = append(choices, sensitiveChoice{SuggestionID: candidate.ID, Decision: "exclude", Reason: leftOutOfGroupsReason})
			}
		}
		if matched == 0 {
			return nil, plan, errors.New("the chosen groups hold none of the current items")
		}
	case res.Answer == "other":
		note := strings.TrimSpace(res.Note)
		if strings.HasPrefix(note, "{") {
			var object otherAnswer
			if err := json.Unmarshal([]byte(note), &object); err != nil {
				return nil, plan, errors.New("Other must contain the documented JSON")
			}
			choices, plan.standing = object.Decisions, object.Standing
		} else if err := json.Unmarshal([]byte(note), &choices); err != nil {
			return nil, plan, errors.New("Other must contain the documented JSON")
		}
	default:
		return nil, plan, errors.New("unknown answer")
	}
	if len(choices) != len(candidates) {
		return nil, plan, errors.New("every current suggestion must have exactly one decision")
	}
	wanted := make(map[string]struct{}, len(candidates))
	for _, c := range candidates {
		wanted[c.ID] = struct{}{}
	}
	seen := map[string]struct{}{}
	for i, choice := range choices {
		if _, ok := wanted[choice.SuggestionID]; !ok {
			return nil, plan, fmt.Errorf("unknown suggestion id %q", choice.SuggestionID)
		}
		if _, duplicate := seen[choice.SuggestionID]; duplicate {
			return nil, plan, fmt.Errorf("suggestion %q is repeated", choice.SuggestionID)
		}
		seen[choice.SuggestionID] = struct{}{}
		switch choice.Decision {
		case "protect", "reacquire":
			if strings.TrimSpace(choice.Reason) != "" {
				return nil, plan, fmt.Errorf("reason is only valid for exclude (%s)", choice.SuggestionID)
			}
			candidate := candidateByID(candidates, choice.SuggestionID)
			if choice.Decision == "reacquire" {
				if candidate.ReacquisitionCommand == "" || candidate.ReacquisitionEvidence == "" {
					return nil, plan, fmt.Errorf("reacquisition is not supported by owner evidence (%s)", choice.SuggestionID)
				}
			}
			choices[i].OwnerCommand = candidate.ReacquisitionCommand
			choices[i].OwnerEvidence = candidate.ReacquisitionEvidence
			choices[i].OwnerNeedsAuthorization = candidate.ReacquisitionNeedsOperator
			choices[i].OwnerAuthorizationReason = candidate.ReacquisitionAuthReason
		case "exclude":
			if strings.TrimSpace(choice.Reason) == "" {
				return nil, plan, fmt.Errorf("exclude requires a reason (%s)", choice.SuggestionID)
			}
			if plan.standing && res.Answer == "other" {
				candidate := candidateByID(candidates, choice.SuggestionID)
				plan.excluded = append(plan.excluded, PolicyExclusion{Owner: candidate.Owner, Name: candidate.Name, Reason: choice.Reason, By: res.AnsweredBy, At: now})
			}
		default:
			return nil, plan, fmt.Errorf("invalid decision %q", choice.Decision)
		}
		if plan.standing && choice.Decision == "reacquire" {
			return nil, plan, fmt.Errorf("a standing rule backs items up; reacquire is a per-item choice (%s)", choice.SuggestionID)
		}
	}
	sort.Slice(choices, func(i, j int) bool { return choices[i].SuggestionID < choices[j].SuggestionID })
	return choices, plan, nil
}

func candidateByID(candidates []sensitiveSnapshot, id string) sensitiveSnapshot {
	for _, candidate := range candidates {
		if candidate.ID == id {
			return candidate
		}
	}
	return sensitiveSnapshot{}
}

func (f *SensitiveDecisionFlow) finishWithoutApply(ctx context.Context, task *operatordecision.PendingAsk, resolution operatordecision.DecisionResolution, result string) error {
	now := f.now().UTC()
	task.State, task.ResolvedAt, task.ResolvedBy = operatordecision.PendingStateRejected, &now, resolution.AnsweredBy
	task.Extra["result"] = result
	task.Extra["result_notice_pending"] = "true"
	if err := f.tasks.Put(ctx, task); err != nil {
		return err
	}
	return f.publishResult(ctx, task)
}

func (f *SensitiveDecisionFlow) publishResult(ctx context.Context, task *operatordecision.PendingAsk) error {
	if task.Extra["result_notice_pending"] != "true" {
		return nil
	}
	occurred := f.now().UTC()
	if task.ResolvedAt != nil {
		occurred = task.ResolvedAt.UTC()
	}
	event := eventbus.DomainEvent{Source: "data-backup-manager", EventType: sensitiveResultEvent, Occurred: occurred, Payload: map[string]any{
		"ask_id": task.ID, "result": task.Extra["result"],
	}}
	if err := f.publisher.PublishDomainEvent(ctx, event); err != nil {
		return err
	}
	task.Extra["result_notice_pending"] = "false"
	return f.tasks.Put(ctx, task)
}

func summarizeSensitiveResult(choices []sensitiveChoice, result coverage.AcceptResult, standing bool) string {
	counts := map[string]int{}
	for _, choice := range choices {
		counts[choice.Decision]++
	}
	operatorGated := 0
	for _, choice := range choices {
		if choice.Decision == "reacquire" && choice.OwnerNeedsAuthorization {
			operatorGated++
		}
	}
	text := fmt.Sprintf("Recorded %d explicit decisions: %d protect, %d reacquire, %d exclude. Registered %d targets; %d registration failures. %d reacquisition decisions still require owner/operator authorization and are not authorized by this review.", len(choices), counts["protect"], counts["reacquire"], counts["exclude"], len(result.Accepted), len(result.Failed), operatorGated)
	if standing {
		text += " Standing rule on: new places that hold secrets are backed up automatically and listed in a weekly note. Turn it off with `data-backup-manager coverage revoke-sensitive-policy`."
	}
	return text
}

func snapshotCandidates(candidates []coverage.Suggestion) []sensitiveSnapshot {
	out := make([]sensitiveSnapshot, 0, len(candidates))
	for _, c := range candidates {
		out = append(out, sensitiveSnapshot{
			ID: c.ID, Owner: c.Owner, OwnerKind: c.OwnerKind, Name: c.Name, Locator: c.Locator, Kind: string(c.SourceKind), Critical: c.Critical, Warning: c.Warning,
			ReacquisitionCommand: c.ReacquisitionCommand, ReacquisitionEvidence: c.ReacquisitionEvidence,
			ReacquisitionNeedsOperator: c.ReacquisitionNeedsOperator, ReacquisitionAuthReason: c.ReacquisitionAuthReason,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func candidateFingerprint(candidates []sensitiveSnapshot) (string, error) {
	data, err := json.Marshal(candidates)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
