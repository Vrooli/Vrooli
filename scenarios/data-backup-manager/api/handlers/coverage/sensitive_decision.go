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
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"data-backup-manager/internal/coverage"
	"data-backup-manager/internal/destinationreadiness"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
)

const (
	sensitiveDecisionKind = "sensitive_coverage_review"
	sensitiveResultEvent  = "data_backup_manager.sensitive_coverage.result.v1"
	attentionFile         = "/home/matthalloran8/.vrooli/research/ATTENTION.md"
)

var actionableHeading = regexp.MustCompile(`(?m)^###\s+([^\s]+)\s+—\s+(.+?)\s*$`)

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
	return &SensitiveDecisionFlow{service: service, tasks: tasks, publisher: publisher, enabled: enabled, logger: logger, now: time.Now, readFile: os.ReadFile}
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
	list := make([]string, 0, len(snapshots))
	for _, row := range snapshots {
		line := fmt.Sprintf("- %s · %s / %s · %s", row.ID, row.Owner, row.Name, row.Warning)
		if row.ReacquisitionCommand != "" {
			line += " · owner reacquisition evidence is declared"
		}
		if row.ReacquisitionNeedsOperator {
			line += " · requires separate owner/operator authorization"
		}
		list = append(list, line)
	}
	attention := f.currentAttentionContext()
	technicalDetails := fmt.Sprintf("Attention context: %s. Review the current Actionable now items before choosing.\n\nCurrent sensitive candidates (%d):\n%s\n\nFor a per-target decision, choose Other and provide a JSON array with exactly one item per suggestion, for example [{\"suggestion_id\":\"stable-id\",\"decision\":\"protect\"},{\"suggestion_id\":\"another-id\",\"decision\":\"reacquire\"},{\"suggestion_id\":\"third-id\",\"decision\":\"exclude\",\"reason\":\"Owner confirmed this is ephemeral\"}]. Each decision must be protect, reacquire, or exclude; every exclusion needs a reason. Reacquire is accepted only with owner-declared command and evidence metadata; any separate owner/operator authorization remains required. The callback re-reads and fingerprints all candidates before any registration. The result is recorded and a follow-up notice is published. No locators, owner commands or credential values are included here.", attentionFile, len(snapshots), strings.Join(list, "\n"))
	facts := operatordecision.Facts{
		Subject:  "Review sensitive backup coverage",
		Question: fmt.Sprintf("Choose a decision for the %d current sensitive targets. Protect registers the targets for backup; reacquire records that the data should be restored from its owner; exclude records a reason and leaves it unregistered. For per-target choices, select Other and provide JSON as described below.", len(snapshots)),
		Options: []operatordecision.Option{
			{Key: "protect_all", Label: "Protect every listed target"},
			{Key: "reacquire_all", Label: "Reacquire every listed target when needed"},
		},
		Recommended:          "protect_all",
		RecommendationReason: "These are durable credential or token sources discovered by DBM. Protecting them preserves recovery options; review owner retention and warnings before deciding.",
		Reversible:           false,
		Urgency:              "high",
		Note:                 fmt.Sprintf("Attention context is included in the review details: %s. Current actionable items: %s.", attentionFile, attention),
		BigPicture:           "This one decision records an explicit disposition for each currently unregistered sensitive suggestion. Protect adds it to DBM's target catalog; reacquire and exclude leave it unregistered and record the disposition. Already-registered targets are outside this review and remain visible in the coverage report. Any changed candidate set invalidates this ask without applying it.",
		TechnicalDetails:     technicalDetails,
		Deadline:             f.now().Add(7 * 24 * time.Hour),
		Correlation:          map[string]string{"ask_id": askID, "candidate_fingerprint": fingerprint, "generation": genText},
		SubjectKey:           "dbm-sensitive-coverage",
	}
	if err := facts.Validate(); err != nil {
		return "", 0, fmt.Errorf("validate sensitive coverage decision: %w", err)
	}
	raiser := operatordecision.Raiser{Source: "data-backup-manager", Kind: sensitiveDecisionKind, Store: f.tasks, Publisher: f.publisher, Clock: sensitiveFlowClock{f}, MaxOutstanding: 1, Window: 7 * 24 * time.Hour}
	for _, prior := range priorAsks {
		if prior == nil || prior.Subject != "sensitive-coverage" || prior.ID == askID || prior.State != operatordecision.PendingStatePending {
			continue
		}
		if prior.EventID != "" {
			if err := raiser.PublishWithdrawal(ctx, operatordecision.Withdrawal{
				RequesterRef: prior.EventID,
				Reason:       "The sensitive coverage candidate set changed. A new review is being raised.",
				AssertedAt:   f.now().UTC(),
			}); err != nil {
				return "", 0, fmt.Errorf("withdraw stale sensitive coverage review: %w", err)
			}
		}
		now := f.now().UTC()
		prior.State, prior.ResolvedAt = operatordecision.PendingStateWithdrawn, &now
		prior.Extra["result"] = "Candidate set changed; no target was registered under this review."
		if err := f.tasks.Put(ctx, prior); err != nil {
			return "", 0, fmt.Errorf("record stale sensitive coverage review withdrawal: %w", err)
		}
	}
	ask := operatordecision.PendingAsk{ID: askID, Subject: "sensitive-coverage", Extra: map[string]string{"candidate_fingerprint": fingerprint, "generation": genText}}
	if _, err := raiser.Raise(ctx, ask, facts); err != nil {
		return "", 0, fmt.Errorf("raise sensitive coverage decision: %w", err)
	}
	return askID, len(snapshots), nil
}

type sensitiveFlowClock struct{ flow *SensitiveDecisionFlow }

func (c sensitiveFlowClock) Now() time.Time { return c.flow.now() }

func (f *SensitiveDecisionFlow) currentAttentionContext() string {
	if f.readFile == nil {
		return "unverified (reader unavailable)"
	}
	data, err := f.readFile(attentionFile)
	if err != nil {
		return "unverified (file unreadable)"
	}
	content := string(data)
	start := strings.Index(content, "## Actionable now")
	if start < 0 {
		return "unverified (no Actionable now section)"
	}
	section := content[start+len("## Actionable now"):]
	if next := strings.Index(section, "\n## "); next >= 0 {
		section = section[:next]
	}
	items := actionableHeading.FindAllStringSubmatch(section, -1)
	if len(items) == 0 {
		if strings.Contains(section, "No actionable items are recorded.") {
			return "none recorded"
		}
		return "unverified (entries could not be parsed)"
	}
	parts := make([]string, 0, min(len(items), 6))
	for _, item := range items[:min(len(items), 6)] {
		parts = append(parts, item[1]+" — "+item[2])
	}
	if len(items) > 6 {
		parts = append(parts, fmt.Sprintf("and %d more", len(items)-6))
	}
	return strings.Join(parts, "; ")
}

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
	choices, err := choicesForResolution(resolution, current)
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
	task.State, task.ResolvedAt = operatordecision.PendingStateApproved, &now
	task.Extra["result"] = summarizeSensitiveResult(choices, result)
	task.Extra["result_notice_pending"] = "true"
	if err := f.tasks.Put(ctx, task); err != nil {
		return true, err
	}
	return true, f.publishResult(ctx, task)
}

func verifiedSensitiveApproval(res operatordecision.DecisionResolution) bool {
	kind := strings.ToLower(strings.TrimSpace(res.AnsweredByIdentityKind))
	return res.State == operatordecision.StateAnswered && !res.ByDefault && strings.TrimSpace(res.AnsweredBy) != "" && strings.TrimSpace(res.AnsweredByIdentitySource) != "" && (kind == "human" || kind == "verified-human")
}

func choicesForResolution(res operatordecision.DecisionResolution, candidates []sensitiveSnapshot) ([]sensitiveChoice, error) {
	choices := make([]sensitiveChoice, 0, len(candidates))
	switch res.Answer {
	case "protect_all", "reacquire_all":
		action := strings.TrimSuffix(res.Answer, "_all")
		for _, candidate := range candidates {
			choices = append(choices, sensitiveChoice{SuggestionID: candidate.ID, Decision: action})
		}
	case "other":
		if err := json.Unmarshal([]byte(res.Note), &choices); err != nil {
			return nil, errors.New("Other must contain the documented JSON array")
		}
	default:
		return nil, errors.New("unknown answer")
	}
	if len(choices) != len(candidates) {
		return nil, errors.New("every current suggestion must have exactly one decision")
	}
	wanted := make(map[string]struct{}, len(candidates))
	for _, c := range candidates {
		wanted[c.ID] = struct{}{}
	}
	seen := map[string]struct{}{}
	for i, choice := range choices {
		if _, ok := wanted[choice.SuggestionID]; !ok {
			return nil, fmt.Errorf("unknown suggestion id %q", choice.SuggestionID)
		}
		if _, duplicate := seen[choice.SuggestionID]; duplicate {
			return nil, fmt.Errorf("suggestion %q is repeated", choice.SuggestionID)
		}
		seen[choice.SuggestionID] = struct{}{}
		switch choice.Decision {
		case "protect", "reacquire":
			if strings.TrimSpace(choice.Reason) != "" {
				return nil, fmt.Errorf("reason is only valid for exclude (%s)", choice.SuggestionID)
			}
			candidate := candidateByID(candidates, choice.SuggestionID)
			if choice.Decision == "reacquire" {
				if candidate.ReacquisitionCommand == "" || candidate.ReacquisitionEvidence == "" {
					return nil, fmt.Errorf("reacquisition is not supported by owner evidence (%s)", choice.SuggestionID)
				}
			}
			choices[i].OwnerCommand = candidate.ReacquisitionCommand
			choices[i].OwnerEvidence = candidate.ReacquisitionEvidence
			choices[i].OwnerNeedsAuthorization = candidate.ReacquisitionNeedsOperator
			choices[i].OwnerAuthorizationReason = candidate.ReacquisitionAuthReason
		case "exclude":
			if strings.TrimSpace(choice.Reason) == "" {
				return nil, fmt.Errorf("exclude requires a reason (%s)", choice.SuggestionID)
			}
		default:
			return nil, fmt.Errorf("invalid decision %q", choice.Decision)
		}
	}
	sort.Slice(choices, func(i, j int) bool { return choices[i].SuggestionID < choices[j].SuggestionID })
	return choices, nil
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

func summarizeSensitiveResult(choices []sensitiveChoice, result coverage.AcceptResult) string {
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
	return fmt.Sprintf("Recorded %d explicit decisions: %d protect, %d reacquire, %d exclude. Registered %d targets; %d registration failures. %d reacquisition decisions still require owner/operator authorization and are not authorized by this review.", len(choices), counts["protect"], counts["reacquire"], counts["exclude"], len(result.Accepted), len(result.Failed), operatorGated)
}

func snapshotCandidates(candidates []coverage.Suggestion) []sensitiveSnapshot {
	out := make([]sensitiveSnapshot, 0, len(candidates))
	for _, c := range candidates {
		out = append(out, sensitiveSnapshot{
			ID: c.ID, Owner: c.Owner, Name: c.Name, Locator: c.Locator, Kind: string(c.SourceKind), Critical: c.Critical, Warning: c.Warning,
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
