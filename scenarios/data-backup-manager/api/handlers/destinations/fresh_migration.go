package destinations

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"data-backup-manager/internal/repohealth"
	"data-backup-manager/internal/repositorymigration"
	"github.com/vrooli/api-core/operatordecision"
)

const freshDestinationDecisionType = "fresh_destination_selection"
const freshDestinationRegistrationDecisionType = "fresh_destination_registration"

const (
	freshRegistrationPendingCategory  = "fresh-destination-registration-pending"
	freshRegistrationCompleteCategory = "fresh-destination-registration-complete"
)

type FreshMigrationCandidate struct {
	DestinationID string
	Name          string
	Location      string
}

type FreshMigrationPlan struct {
	PlanID  string
	Targets []string
}

type FreshMigrationProgress struct {
	Stage    string
	Complete bool
}

// FreshMigrationCoordinator is the adapter from the decision flow to the
// resumable repositorymigration owner. Candidates must already be registered,
// writable, identity-bound, and capacity/credential proven. Begin only persists
// the exact operator-selected inputs; Resume owns all migration effects.
type FreshMigrationCoordinator interface {
	Candidates(context.Context, repohealth.Incident) (FreshMigrationPlan, []FreshMigrationCandidate, error)
	RegisterLocalDestination(context.Context, string, string, string, string, []string) (FreshMigrationCandidate, error)
	Begin(context.Context, string, string, string, string, []string) error
	Resume(context.Context, string) (FreshMigrationProgress, error)
}

type repositoryMigrationCoordinator struct {
	effects *repositorymigration.DBMEffects
	owner   *repositorymigration.Service
}

// NewRepositoryMigrationCoordinator wires the handler decision flow to the
// durable migration owner and its DBM production effects adapter.
func NewRepositoryMigrationCoordinator(effects *repositorymigration.DBMEffects, owner *repositorymigration.Service) FreshMigrationCoordinator {
	return &repositoryMigrationCoordinator{effects: effects, owner: owner}
}

func (c *repositoryMigrationCoordinator) Candidates(ctx context.Context, inc repohealth.Incident) (FreshMigrationPlan, []FreshMigrationCandidate, error) {
	if c == nil || c.effects == nil || c.owner == nil {
		return FreshMigrationPlan{}, nil, fmt.Errorf("repository migration coordinator is not configured")
	}
	planID, targets, err := c.effects.PlanForIncident(ctx, inc.PlanIDs, inc.DestinationID)
	if err != nil {
		return FreshMigrationPlan{}, nil, err
	}
	dests, err := c.effects.Candidates(ctx, planID, inc.DestinationID, targets)
	if err != nil {
		return FreshMigrationPlan{}, nil, err
	}
	candidates := make([]FreshMigrationCandidate, 0, len(dests))
	for _, d := range dests {
		candidates = append(candidates, FreshMigrationCandidate{DestinationID: d.ID, Name: d.Name, Location: d.Location})
	}
	return FreshMigrationPlan{PlanID: planID, Targets: targets}, candidates, nil
}

func (c *repositoryMigrationCoordinator) Begin(ctx context.Context, id, planID, oldDestinationID, newDestinationID string, targets []string) error {
	_, err := c.owner.Begin(ctx, id, planID, oldDestinationID, newDestinationID, targets)
	return err
}

func (c *repositoryMigrationCoordinator) RegisterLocalDestination(ctx context.Context, migrationID, planID, oldDestinationID, location string, targetIDs []string) (FreshMigrationCandidate, error) {
	if c == nil || c.effects == nil || c.owner == nil {
		return FreshMigrationCandidate{}, errors.New("repository migration coordinator is not configured")
	}
	d, err := c.effects.RegisterLocalDestination(ctx, migrationID, planID, oldDestinationID, location, targetIDs)
	if err != nil {
		return FreshMigrationCandidate{}, err
	}
	return FreshMigrationCandidate{DestinationID: d.ID, Name: d.Name, Location: d.Location}, nil
}

func (c *repositoryMigrationCoordinator) Resume(ctx context.Context, id string) (FreshMigrationProgress, error) {
	m, err := c.owner.Resume(ctx, id)
	return FreshMigrationProgress{Stage: string(m.Stage), Complete: m.Stage == repositorymigration.StageComplete}, err
}

func freshDestinationAskID(migrationID string, generation int) string {
	return operatordecision.CorrelationID("dbm-fresh-destination", migrationID, strconv.Itoa(generation))
}

func freshDestinationRegistrationAskID(migrationID string, generation int) string {
	return operatordecision.CorrelationID("dbm-fresh-register", migrationID, strconv.Itoa(generation))
}

func proposedLocalDestinationPath(raw string) (string, error) {
	location := strings.TrimSpace(raw)
	if location == "" || len(location) > 512 || !filepath.IsAbs(location) || strings.Contains(location, "://") || strings.ContainsAny(location, "\x00\r\n\t") {
		return "", errors.New("an absolute local filesystem directory is required")
	}
	location = filepath.Clean(location)
	if location == string(filepath.Separator) || filepath.VolumeName(location)+string(filepath.Separator) == location {
		return "", errors.New("filesystem root is not a destination bundle directory")
	}
	return location, nil
}

func (f *BackupFailureFlow) reissueFreshDestinationAsk(ctx context.Context, inc repohealth.Incident, answeredGeneration int, detail string) error {
	if inc.FreshMigrationStage != "awaiting_destination_selection" {
		return nil
	}
	if answeredGeneration >= inc.FreshMigrationSelectionGeneration {
		inc.FreshMigrationSelectionGeneration = answeredGeneration + 1
	} else {
		inc.FreshMigrationSelectionGeneration++
	}
	inc.FreshMigrationStage = "awaiting_destination_selection"
	inc.FreshMigrationDetail = detail
	inc.FreshMigrationSelectionAskID = freshDestinationAskID(inc.FreshMigrationID, answeredGeneration)
	inc.UpdatedAt = f.now().UTC()
	return f.incidents.Put(ctx, inc)
}

func freshDestinationSubjectKey(migrationID string) string {
	key := "fresh-migration:" + strings.TrimSpace(migrationID)
	if operatordecision.SubjectKeyRule.MatchString(key) {
		return key
	}
	return "fresh-migration:" + operatordecision.CorrelationID(migrationID)
}

func (f *BackupFailureFlow) askFreshDestination(ctx context.Context, inc repohealth.Incident) (bool, error) {
	if f.migrations == nil {
		return false, nil
	}
	plan, candidates, err := f.migrations.Candidates(ctx, inc)
	if err != nil {
		return false, fmt.Errorf("find eligible fresh migration destinations: %w", err)
	}
	if plan.PlanID == "" || len(plan.Targets) != 14 {
		return false, fmt.Errorf("fresh migration requires one exact 14-target source plan")
	}
	if inc.FreshMigrationPlanID == "" {
		inc.FreshMigrationPlanID = plan.PlanID
		inc.FreshMigrationTargets = append([]string(nil), plan.Targets...)
		inc.UpdatedAt = f.now().UTC()
		if err := f.incidents.Put(ctx, inc); err != nil {
			return false, fmt.Errorf("persist exact fresh migration source plan: %w", err)
		}
	} else if inc.FreshMigrationPlanID != plan.PlanID || !sameFreshTargets(inc.FreshMigrationTargets, plan.Targets) {
		return false, f.blockFreshPlanChange(ctx, inc)
	}
	// Notification Hub decisions have a bounded menu. Never silently omit a
	// registered destination: ask only when every eligible candidate fits.
	if len(candidates) > 6 {
		return false, fmt.Errorf("%d destinations are eligible, more than one decision can present; narrow the registered destination set", len(candidates))
	}
	options := make([]operatordecision.Option, 0, len(candidates)+2)
	for _, candidate := range candidates {
		id := strings.TrimSpace(candidate.DestinationID)
		if id == "" || len(id) > 56 {
			return false, fmt.Errorf("eligible destination has an invalid decision identity")
		}
		label := strings.TrimSpace(candidate.Name)
		if label == "" {
			label = id
		}
		if strings.TrimSpace(candidate.Location) != "" {
			label += " — " + strings.TrimSpace(candidate.Location)
		}
		label = truncateOptionLabel(label, operatordecision.MaxOptionLabel)
		options = append(options, operatordecision.Option{Key: "dest:" + id, Label: label})
	}
	options = append(options,
		operatordecision.Option{Key: "register-local", Label: "Register a new local filesystem destination"},
		operatordecision.Option{Key: "defer", Label: "Keep the old repository and decide later"},
	)
	selectionGeneration := inc.FreshMigrationSelectionGeneration
	if selectionGeneration < 1 {
		selectionGeneration = 1
	}
	selectionNote := inc.FreshMigrationDetail
	selectionQuestion := "You approved a fresh backup set. Select one currently registered destination, or choose Register a new local filesystem destination and put its absolute directory in the note field. A typed path is only a proposal; DBM asks you to confirm before it creates a repository. No migration starts until you select the registered destination afterward."
	if len(candidates) == 0 {
		selectionNote = "No currently registered filesystem destination passes the exact 14-target readiness, identity and capacity checks. You may defer or propose a local filesystem directory for DBM's destination owner to validate."
		selectionQuestion = "No currently registered destination currently passes checks for this 14-target plan. Choose Register a new local filesystem destination and put its absolute directory in the note field, or defer. DBM asks separately before creating a repository, then asks you to select the registered destination before migration starts."
	}
	facts := operatordecision.Facts{
		Subject:  "Select a registered destination for the fresh backup set",
		Question: selectionQuestion,
		Options:  options, Reversible: false, Urgency: "high",
		Note:             selectionNote,
		BigPicture:       "DBM will capture each target to the selected destination, restore every captured snapshot to scratch, and switch plan bindings only after all 14 target receipts pass. The damaged repository remains unchanged.",
		TechnicalDetails: "Only currently registered, identity-bound filesystem destinations with a passing capacity observation and matching credential-authority reference are offered. Remote destinations and destinations with unknown capacity or identity are not offered. Register a new local filesystem destination accepts only an absolute local filesystem path in the note field and asks a separate verified-human confirmation before the destination owner creates anything. Provider, credential and secret inputs are not accepted.",
		Correlation:      map[string]string{"decision_type": freshDestinationDecisionType, "incident_id": inc.ID, "generation": fmt.Sprint(inc.Generation), "migration_id": inc.FreshMigrationID, "selection_generation": strconv.Itoa(selectionGeneration)},
	}
	if err := facts.Validate(); err != nil {
		return false, fmt.Errorf("validate destination selection decision: %w", err)
	}
	ask := operatordecision.PendingAsk{ID: freshDestinationAskID(inc.FreshMigrationID, selectionGeneration), Subject: inc.ID, Extra: map[string]string{
		"incident_id": inc.ID, "generation": fmt.Sprint(inc.Generation), "destination_id": inc.DestinationID,
		"decision_type": freshDestinationDecisionType, "migration_id": inc.FreshMigrationID, "selection_generation": strconv.Itoa(selectionGeneration),
	}}
	if err := f.ledger.raise(ctx, ask, facts, freshDestinationSubjectKey(inc.FreshMigrationID)); err != nil {
		return false, err
	}
	return true, nil
}

func sameFreshTargets(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	a, b := append([]string(nil), left...), append([]string(nil), right...)
	sort.Strings(a)
	sort.Strings(b)
	for i := range a {
		if strings.TrimSpace(a[i]) == "" || strings.TrimSpace(a[i]) != strings.TrimSpace(b[i]) || (i > 0 && a[i] == a[i-1]) {
			return false
		}
	}
	return true
}

func (f *BackupFailureFlow) blockFreshPlanChange(ctx context.Context, inc repohealth.Incident) error {
	inc.FreshMigrationStage = "plan_changed"
	inc.FreshMigrationDetail = "The exact source plan or target set changed after the fresh-repository approval; DBM stopped before creating or using a new destination. A current plan requires a new verified decision."
	inc.UpdatedAt = f.now().UTC()
	if err := f.incidents.Put(ctx, inc); err != nil {
		return err
	}
	ask, err := f.asks.Get(ctx, backupAskID(inc.ID, inc.Generation))
	if err != nil {
		return err
	}
	if ask == nil {
		return errors.New("fresh migration approval ask is missing while reporting a plan change")
	}
	if ask.Extra == nil {
		ask.Extra = map[string]string{}
	}
	ask.Extra["result_title"] = "Fresh repository approval needs review"
	ask.Extra["result_body"] = "The original 14-target plan changed after approval. DBM did not create a destination or start migration; review the current backup decision before continuing."
	ask.Extra["result_category"], ask.Extra["result_urgency"] = "fresh-migration-plan-changed", "high"
	ask.Extra["result_notification_pending"] = "true"
	if err := f.asks.Put(ctx, ask); err != nil {
		return err
	}
	return f.publishPendingResult(ctx, ask)
}

func truncateOptionLabel(label string, maxBytes int) string {
	if len(label) <= maxBytes {
		return label
	}
	for len(label) > maxBytes {
		_, size := utf8.DecodeLastRuneInString(label)
		label = label[:len(label)-size]
	}
	return label
}

func (f *BackupFailureFlow) askFreshDestinationRegistration(ctx context.Context, inc repohealth.Incident, generation int) error {
	if f.migrations == nil {
		return errors.New("destination registration owner is not configured")
	}
	askID := freshDestinationRegistrationAskID(inc.FreshMigrationID, generation)
	facts := operatordecision.Facts{
		Subject:    "Confirm the new local backup destination",
		Question:   "Register a new encrypted local filesystem repository at this exact operator-supplied directory?",
		Options:    []operatordecision.Option{{Key: "register", Label: "Register through DBM destination owner"}, {Key: "defer", Label: "Keep the old repository and decide later"}},
		Reversible: false, Urgency: "high",
		Note:             "Local path: " + inc.FreshMigrationProposedLocation,
		BigPicture:       "DBM will ask its destination owner to validate readiness and provision one local filesystem repository at the path above. The owner manages the credential reference; no provider or secret value is accepted here. Migration still requires a separate selection of the resulting registered destination.",
		TechnicalDetails: "The previous decision's Other text is only a path proposal. This separate verified-human answer is required before the owner creates anything. The filesystem owner rejects protected-root paths and destinations whose readiness is not proven.",
		Correlation:      map[string]string{"decision_type": freshDestinationRegistrationDecisionType, "incident_id": inc.ID, "generation": strconv.Itoa(inc.Generation), "migration_id": inc.FreshMigrationID, "selection_generation": strconv.Itoa(generation)},
	}
	if err := facts.Validate(); err != nil {
		return fmt.Errorf("validate local destination registration decision: %w", err)
	}
	ask := operatordecision.PendingAsk{ID: askID, Subject: inc.ID, Extra: map[string]string{
		"decision_type": freshDestinationRegistrationDecisionType, "incident_id": inc.ID, "generation": strconv.Itoa(inc.Generation),
		"migration_id": inc.FreshMigrationID, "selection_generation": strconv.Itoa(generation),
	}}
	return f.ledger.raise(ctx, ask, facts, freshDestinationSubjectKey(inc.FreshMigrationID+"-register"))
}

func (f *BackupFailureFlow) applyFreshDestination(ctx context.Context, res operatordecision.DecisionResolution) (bool, error) {
	migrationID := strings.TrimSpace(res.Correlation["migration_id"])
	incidentID := strings.TrimSpace(res.Correlation["incident_id"])
	generation, genErr := strconv.Atoi(strings.TrimSpace(res.Correlation["generation"]))
	selectionGeneration, selectionErr := strconv.Atoi(strings.TrimSpace(res.Correlation["selection_generation"]))
	if migrationID == "" || incidentID == "" || genErr != nil || generation < 1 || selectionErr != nil || selectionGeneration < 1 {
		return true, nil
	}
	ask, err := f.asks.Get(ctx, freshDestinationAskID(migrationID, selectionGeneration))
	if err != nil {
		return true, err
	}
	if ask == nil || ask.Extra["decision_type"] != freshDestinationDecisionType || ask.Extra["incident_id"] != incidentID || ask.Extra["generation"] != strconv.Itoa(generation) || ask.Extra["selection_generation"] != strconv.Itoa(selectionGeneration) {
		return true, nil
	}
	if res.State == operatordecision.StateWithdrawn {
		return true, f.ledger.hubWithdrew(ctx, ask, res)
	}
	if ask.State != operatordecision.PendingStatePending {
		return true, f.publishPendingResult(ctx, ask)
	}
	inc, found, err := f.incidents.Get(ctx, incidentID)
	if err != nil {
		return true, err
	}
	if !found || inc.Generation != generation || inc.State != repohealth.IncidentAgentRequested || inc.FreshMigrationID != migrationID || (inc.FreshMigrationStage != "awaiting_destination_selection" && inc.FreshMigrationStage != "selection_recorded" && inc.FreshMigrationStage != "migrating") {
		return true, f.finish(ctx, ask, operatordecision.PendingStateRejected, "", noticeText{title: "Fresh migration selection expired", body: "The migration question no longer matches the current DBM record. No repository or plan was changed.", category: categoryBackupNeedsAttention, urgency: "normal"})
	}
	if !isVerifiedRepairApproval(res) {
		if err := f.reissueFreshDestinationAsk(ctx, inc, selectionGeneration, "The prior destination response was not a verified-human selection; DBM is waiting for a new explicit destination choice."); err != nil {
			return true, err
		}
		return true, f.finish(ctx, ask, operatordecision.PendingStateRejected, "", noticeText{title: "Fresh migration was not started", body: "A verified-human destination selection is required. The old repository and plan remain unchanged.", category: categoryBackupNeedsAttention, urgency: "normal"})
	}
	if res.Answer == "defer" {
		inc.FreshMigrationSelectionAskID = ask.ID
		inc.FreshMigrationSelectionBy = res.AnsweredBy
		inc.FreshMigrationSelectionGeneration = selectionGeneration + 1
		inc.FreshMigrationStage = "selection_deferred"
		inc.FreshMigrationSelectionRetryAt = f.now().UTC().Add(24 * time.Hour)
		inc.FreshMigrationDetail = "The verified human deferred destination selection; no repository or plan effects have started."
		inc.UpdatedAt = f.now().UTC()
		if err := f.incidents.Put(ctx, inc); err != nil {
			return true, err
		}
		return true, f.finish(ctx, ask, operatordecision.PendingStateRejected, res.AnsweredBy, noticeText{title: "Fresh migration deferred", body: "The old repository and plan remain unchanged. DBM will ask again in 24 hours if the incident remains active.", category: categoryBackupNeedsAttention, urgency: "normal"})
	}
	if res.Answer == "register-local" {
		location, pathErr := proposedLocalDestinationPath(res.Note)
		if pathErr != nil {
			if err := f.reissueFreshDestinationAsk(ctx, inc, selectionGeneration, "The proposed path was not a usable absolute local directory; DBM has not provisioned a destination."); err != nil {
				return true, err
			}
			return true, f.finish(ctx, ask, operatordecision.PendingStateRejected, res.AnsweredBy, noticeText{title: "Local destination path needs correction", body: "DBM accepts only an absolute local filesystem directory in the note. No repository or plan was changed; submit a corrected path when ready.", category: categoryBackupNeedsAttention, urgency: "normal"})
		}
		inc.FreshMigrationProposedLocation = location
		inc.FreshMigrationSelectionAskID = ask.ID
		inc.FreshMigrationSelectionBy = res.AnsweredBy
		inc.FreshMigrationRegistrationAskID = freshDestinationRegistrationAskID(migrationID, selectionGeneration)
		inc.FreshMigrationRegistrationGeneration = selectionGeneration
		inc.FreshMigrationSelectionGeneration = selectionGeneration + 1
		inc.FreshMigrationStage = "local_destination_proposed"
		inc.FreshMigrationDetail = "A verified human proposed a local filesystem path; DBM has not created a repository or changed plan bindings."
		inc.UpdatedAt = f.now().UTC()
		if err := f.incidents.Put(ctx, inc); err != nil {
			return true, err
		}
		if err := f.askFreshDestinationRegistration(ctx, inc, selectionGeneration); err != nil {
			return true, err
		}
		return true, f.finish(ctx, ask, operatordecision.PendingStateRejected, res.AnsweredBy, noticeText{title: "Confirm the local destination path", body: "DBM recorded the proposed local path and will ask separately before provisioning it. No repository or plan has been changed.", category: categoryBackupNeedsAttention, urgency: "normal"})
	}
	if !strings.HasPrefix(res.Answer, "dest:") {
		if err := f.reissueFreshDestinationAsk(ctx, inc, selectionGeneration, "The prior response did not select an offered destination. DBM is waiting for a new explicit destination choice."); err != nil {
			return true, err
		}
		return true, f.finish(ctx, ask, operatordecision.PendingStateRejected, "", noticeText{title: "Fresh migration selection was invalid", body: "The answer did not select a currently offered registered destination. No repository or plan was changed.", category: categoryBackupNeedsAttention, urgency: "normal"})
	}
	if f.migrations == nil {
		return true, f.finish(ctx, ask, operatordecision.PendingStateRejected, "", noticeText{title: "Fresh migration is unavailable", body: "DBM has no configured migration owner. No repository or plan was changed.", category: categoryBackupNeedsAttention, urgency: "high"})
	}
	selectedID := strings.TrimPrefix(res.Answer, "dest:")
	if inc.FreshMigrationStage != "awaiting_destination_selection" && (inc.FreshMigrationDestinationID != selectedID || inc.FreshMigrationSelectionAskID != ask.ID) {
		return true, f.finish(ctx, ask, operatordecision.PendingStateRejected, "", noticeText{title: "Fresh migration selection replay did not match", body: "The persisted migration selection differs from this callback. No repository or plan was changed.", category: categoryBackupNeedsAttention, urgency: "high"})
	}
	plan, candidates, err := f.migrations.Candidates(ctx, inc)
	if err != nil {
		return true, err
	}
	if plan.PlanID != inc.FreshMigrationPlanID || !sameFreshTargets(inc.FreshMigrationTargets, plan.Targets) {
		if err := f.blockFreshPlanChange(ctx, inc); err != nil {
			return true, err
		}
		return true, f.finish(ctx, ask, operatordecision.PendingStateRejected, "", noticeText{title: "Fresh migration plan changed", body: "The approved source plan or exact target set changed. DBM did not create a local destination or start migration; review the current backup decision before continuing.", category: "fresh-migration-plan-changed", urgency: "high"})
	}
	selected := false
	for _, candidate := range candidates {
		if candidate.DestinationID == selectedID {
			selected = true
			break
		}
	}
	if !selected || plan.PlanID == "" || len(plan.Targets) != 14 {
		if err := f.reissueFreshDestinationAsk(ctx, inc, selectionGeneration, "The selected destination no longer passes current identity, capacity and readiness checks; DBM is waiting for a new explicit choice."); err != nil {
			return true, err
		}
		return true, f.finish(ctx, ask, operatordecision.PendingStateRejected, "", noticeText{title: "Selected destination is no longer eligible", body: "DBM could not re-prove that exact registered destination for this 14-target plan. No repository or plan was changed; select again after reviewing destination readiness.", category: categoryBackupNeedsAttention, urgency: "high"})
	}
	// Persist the human selection before the migration journal or any effects.
	if inc.FreshMigrationStage == "awaiting_destination_selection" {
		inc.FreshMigrationPlanID, inc.FreshMigrationDestinationID = plan.PlanID, selectedID
		inc.FreshMigrationTargets = append([]string(nil), plan.Targets...)
		inc.FreshMigrationSelectionAskID = ask.ID
		inc.FreshMigrationSelectionBy = res.AnsweredBy
		inc.FreshMigrationStage, inc.FreshMigrationDetail = "selection_recorded", "The verified human selected a currently registered destination; migration effects have not started."
		inc.UpdatedAt = f.now().UTC()
		if err := f.incidents.Put(ctx, inc); err != nil {
			return true, fmt.Errorf("persist fresh migration destination selection: %w", err)
		}
	}
	if err := f.migrations.Begin(ctx, migrationID, plan.PlanID, inc.DestinationID, selectedID, plan.Targets); err != nil {
		return true, f.finish(ctx, ask, operatordecision.PendingStateRejected, res.AnsweredBy, noticeText{title: "Fresh migration could not be prepared", body: "DBM could not persist the destination-bound migration journal. No repository or plan effects were started; the old repository remains preserved.", category: categoryBackupNeedsAttention, urgency: "high"})
	}
	inc.FreshMigrationStage, inc.FreshMigrationDetail = "migrating", "The migration journal is durable. DBM will capture and scratch-drill all 14 targets before switching the original plan."
	inc.UpdatedAt = f.now().UTC()
	if err := f.incidents.Put(ctx, inc); err != nil {
		return true, fmt.Errorf("persist migration start: %w", err)
	}
	return true, f.finish(ctx, ask, operatordecision.PendingStateApproved, res.AnsweredBy, noticeText{title: "Fresh repository migration started", body: "DBM recorded your destination selection. It will capture and scratch-drill all 14 targets, then switch the plan only after every receipt and a fresh capacity proof pass. The old repository remains unchanged.", category: categoryBackupNeedsAttention, urgency: "normal"})
}

func (f *BackupFailureFlow) applyFreshDestinationRegistration(ctx context.Context, res operatordecision.DecisionResolution) (bool, error) {
	migrationID := strings.TrimSpace(res.Correlation["migration_id"])
	incidentID := strings.TrimSpace(res.Correlation["incident_id"])
	generation, genErr := strconv.Atoi(strings.TrimSpace(res.Correlation["generation"]))
	selectionGeneration, selectionErr := strconv.Atoi(strings.TrimSpace(res.Correlation["selection_generation"]))
	if migrationID == "" || incidentID == "" || genErr != nil || generation < 1 || selectionErr != nil || selectionGeneration < 1 {
		return true, nil
	}
	askID := freshDestinationRegistrationAskID(migrationID, selectionGeneration)
	ask, err := f.asks.Get(ctx, askID)
	if err != nil {
		return true, err
	}
	if ask == nil || ask.Extra["decision_type"] != freshDestinationRegistrationDecisionType || ask.Extra["incident_id"] != incidentID || ask.Extra["generation"] != strconv.Itoa(generation) || ask.Extra["selection_generation"] != strconv.Itoa(selectionGeneration) {
		return true, nil
	}
	if res.State == operatordecision.StateWithdrawn {
		return true, f.ledger.hubWithdrew(ctx, ask, res)
	}
	if ask.State != operatordecision.PendingStatePending {
		return true, f.publishPendingResult(ctx, ask)
	}
	inc, found, err := f.incidents.Get(ctx, incidentID)
	if err != nil {
		return true, err
	}
	if !found || inc.Generation != generation || inc.State != repohealth.IncidentAgentRequested || inc.FreshMigrationID != migrationID || (inc.FreshMigrationStage != "local_destination_proposed" && inc.FreshMigrationStage != "local_destination_registration_pending") {
		return true, f.finish(ctx, ask, operatordecision.PendingStateRejected, "", noticeText{title: "Local destination request expired", body: "This request no longer matches the current DBM migration record. No destination or plan change was made.", category: categoryBackupNeedsAttention, urgency: "normal"})
	}
	if !isVerifiedRepairApproval(res) {
		inc.FreshMigrationStage = "awaiting_destination_selection"
		inc.FreshMigrationDetail = "The local destination registration response was not verified; no destination was provisioned. A new explicit choice is required."
		inc.UpdatedAt = f.now().UTC()
		if err := f.incidents.Put(ctx, inc); err != nil {
			return true, err
		}
		return true, f.finish(ctx, ask, operatordecision.PendingStateRejected, "", noticeText{title: "Local destination was not registered", body: "A verified-human confirmation is required before DBM can provision a destination. No migration started.", category: categoryBackupNeedsAttention, urgency: "normal"})
	}
	if res.Answer == "defer" {
		inc.FreshMigrationSelectionGeneration = selectionGeneration + 1
		inc.FreshMigrationSelectionRetryAt = f.now().UTC().Add(24 * time.Hour)
		inc.FreshMigrationStage = "selection_deferred"
		inc.FreshMigrationDetail = "The verified human deferred local destination registration; no repository or plan effects have started."
		inc.UpdatedAt = f.now().UTC()
		if err := f.incidents.Put(ctx, inc); err != nil {
			return true, err
		}
		return true, f.finish(ctx, ask, operatordecision.PendingStateRejected, res.AnsweredBy, noticeText{title: "Fresh migration deferred", body: "No local repository was created. The old repository and plan remain unchanged; DBM will ask again in 24 hours if the incident remains active.", category: categoryBackupNeedsAttention, urgency: "normal"})
	}
	if res.Answer != "register" {
		inc.FreshMigrationStage = "awaiting_destination_selection"
		inc.FreshMigrationDetail = "The registration answer was not one of the offered choices; no destination was provisioned. A new explicit choice is required."
		inc.UpdatedAt = f.now().UTC()
		if err := f.incidents.Put(ctx, inc); err != nil {
			return true, err
		}
		return true, f.finish(ctx, ask, operatordecision.PendingStateRejected, "", noticeText{title: "Local destination answer was invalid", body: "Choose the explicit DBM registration option or defer. Free-form text cannot authorize a provider or credential.", category: categoryBackupNeedsAttention, urgency: "normal"})
	}
	if _, err := proposedLocalDestinationPath(inc.FreshMigrationProposedLocation); err != nil {
		inc.FreshMigrationStage = "awaiting_destination_selection"
		inc.FreshMigrationDetail = "The persisted path proposal was no longer valid; no destination was provisioned. A new path is required."
		inc.UpdatedAt = f.now().UTC()
		if putErr := f.incidents.Put(ctx, inc); putErr != nil {
			return true, putErr
		}
		return true, f.finish(ctx, ask, operatordecision.PendingStateRejected, "", noticeText{title: "Local destination path is invalid", body: "DBM no longer has a valid absolute local path for this request. No repository was created.", category: categoryBackupNeedsAttention, urgency: "high"})
	}
	plan, _, planErr := f.migrations.Candidates(ctx, inc)
	if planErr != nil {
		return true, planErr
	}
	if plan.PlanID != inc.FreshMigrationPlanID || !sameFreshTargets(inc.FreshMigrationTargets, plan.Targets) {
		if err := f.blockFreshPlanChange(ctx, inc); err != nil {
			return true, err
		}
		return true, f.finish(ctx, ask, operatordecision.PendingStateRejected, "", noticeText{title: "Fresh migration plan changed", body: "The approved source plan or exact target set changed. DBM did not create a local destination or start migration; review the current backup decision before continuing.", category: "fresh-migration-plan-changed", urgency: "high"})
	}
	inc.FreshMigrationRegistrationBy = res.AnsweredBy
	inc.FreshMigrationStage = "local_destination_registration_pending"
	inc.FreshMigrationDetail = "A verified human approved registration of the exact proposed local path; DBM is asking its destination owner to validate and provision it."
	inc.UpdatedAt = f.now().UTC()
	if err := f.incidents.Put(ctx, inc); err != nil {
		return true, err
	}
	_, registrationErr := f.completeFreshDestinationRegistration(ctx, inc)
	if registrationErr != nil {
		inc.FreshMigrationDetail = "DBM destination owner has not yet returned a ready registered destination: " + registrationErr.Error()
		inc.UpdatedAt = f.now().UTC()
		_ = f.incidents.Put(ctx, inc)
		return true, f.finish(ctx, ask, operatordecision.PendingStateApproved, res.AnsweredBy, noticeText{title: "Local destination needs owner reconciliation", body: "DBM recorded your verified request, but its destination owner has not returned a ready destination. No migration or plan switch started. Review the destination's DBM provisioning state and retry after the owner reports it ready.", category: freshRegistrationPendingCategory, urgency: "high"})
	}
	return true, f.finish(ctx, ask, operatordecision.PendingStateApproved, res.AnsweredBy, noticeText{title: "Local destination registered for review", body: "DBM's destination owner registered a local repository at the supplied path. It will appear in a separate decision only if exact 14-target capacity and device readiness pass. Select that registered destination in the follow-up decision before any migration starts.", category: freshRegistrationCompleteCategory, urgency: "normal"})
}

func (f *BackupFailureFlow) completeFreshDestinationRegistration(ctx context.Context, inc repohealth.Incident) (FreshMigrationCandidate, error) {
	if f.migrations == nil {
		return FreshMigrationCandidate{}, errors.New("destination registration owner is unavailable")
	}
	candidate, err := f.migrations.RegisterLocalDestination(ctx, inc.FreshMigrationID, inc.FreshMigrationPlanID, inc.DestinationID, inc.FreshMigrationProposedLocation, inc.FreshMigrationTargets)
	if err != nil {
		return FreshMigrationCandidate{}, err
	}
	current, found, err := f.incidents.Get(ctx, inc.ID)
	if err != nil {
		return FreshMigrationCandidate{}, err
	}
	if !found || current.Generation != inc.Generation || current.FreshMigrationID != inc.FreshMigrationID || current.FreshMigrationStage != "local_destination_registration_pending" {
		return FreshMigrationCandidate{}, errors.New("incident changed while destination owner was provisioning")
	}
	current.FreshMigrationRegisteredDestID = candidate.DestinationID
	current.FreshMigrationStage = "awaiting_destination_selection"
	current.FreshMigrationDetail = "Destination owner returned a registered repository. It must pass the exact target-set capacity and readiness checks, and still requires a separate verified-human selection before migration."
	current.UpdatedAt = f.now().UTC()
	if err := f.incidents.Put(ctx, current); err != nil {
		return FreshMigrationCandidate{}, err
	}
	if _, err := f.askFreshDestination(ctx, current); err != nil {
		return FreshMigrationCandidate{}, err
	}
	return candidate, nil
}

func (f *BackupFailureFlow) settleFreshSelectionOutbox(ctx context.Context, inc repohealth.Incident) error {
	if inc.FreshMigrationSelectionAskID == "" || inc.FreshMigrationStage == "selection_recorded" || inc.FreshMigrationStage == "migrating" || inc.FreshMigrationStage == "complete" {
		return nil
	}
	ask, err := f.asks.Get(ctx, inc.FreshMigrationSelectionAskID)
	if err != nil || ask == nil || ask.State != operatordecision.PendingStatePending {
		return err
	}
	currentAskID := freshDestinationAskID(inc.FreshMigrationID, inc.FreshMigrationSelectionGeneration)
	if inc.FreshMigrationStage == "awaiting_destination_selection" && ask.ID == currentAskID {
		return nil
	}
	notice := noticeText{title: "Fresh destination response recorded", body: "DBM recorded the response. Review the current destination decision for the next required step; no migration has started.", category: categoryBackupNeedsAttention, urgency: "normal"}
	if inc.FreshMigrationStage == "local_destination_proposed" {
		notice.title = "Confirm the local destination path"
		notice.body = "DBM recorded the proposed local path and raised a separate verified-human confirmation. No repository or plan has been changed."
	} else if inc.FreshMigrationStage == "selection_deferred" {
		notice.title = "Fresh migration deferred"
		notice.body = "The old repository and plan remain unchanged. DBM will ask again in 24 hours if the incident remains active."
	}
	return f.finish(ctx, ask, operatordecision.PendingStateRejected, inc.FreshMigrationSelectionBy, notice)
}

func (f *BackupFailureFlow) recordFreshRegistrationResult(ctx context.Context, ask *operatordecision.PendingAsk, by string) error {
	if ask == nil {
		return nil
	}
	notice := noticeText{
		title:    "Local destination registered for review",
		body:     "DBM's destination owner returned a ready local repository. DBM will offer it in a separate decision only if exact 14-target capacity and device readiness pass; migration still requires your explicit selection.",
		category: freshRegistrationCompleteCategory, urgency: "normal",
	}
	if ask.State == operatordecision.PendingStatePending {
		return f.finish(ctx, ask, operatordecision.PendingStateApproved, by, notice)
	}
	if ask.Extra["result_category"] == freshRegistrationCompleteCategory {
		return f.publishPendingResult(ctx, ask)
	}
	ask.Extra["result_title"], ask.Extra["result_body"] = notice.title, notice.body
	ask.Extra["result_category"], ask.Extra["result_urgency"] = notice.category, notice.urgency
	ask.Extra["result_notification_pending"] = "true"
	if err := f.asks.Put(ctx, ask); err != nil {
		return err
	}
	return f.publishPendingResult(ctx, ask)
}

func (f *BackupFailureFlow) resumeFreshMigrations(ctx context.Context) error {
	if f.migrations == nil || !f.ready() {
		return nil
	}
	incidents, err := f.incidents.List(ctx)
	if err != nil {
		return err
	}
	for _, inc := range incidents {
		if inc.State != repohealth.IncidentAgentRequested || inc.FreshMigrationStage != "migrating" || inc.FreshMigrationID == "" {
			continue
		}
		f.migrationMu.Lock()
		if f.migrationRunning[inc.FreshMigrationID] {
			f.migrationMu.Unlock()
			continue
		}
		f.migrationRunning[inc.FreshMigrationID] = true
		f.migrationWG.Add(1)
		f.migrationMu.Unlock()
		go f.runFreshMigration(ctx, inc)
	}
	return nil
}

func (f *BackupFailureFlow) runFreshMigration(ctx context.Context, inc repohealth.Incident) {
	defer f.migrationWG.Done()
	defer func() { f.migrationMu.Lock(); delete(f.migrationRunning, inc.FreshMigrationID); f.migrationMu.Unlock() }()
	progress, err := f.migrations.Resume(ctx, inc.FreshMigrationID)
	current, found, readErr := f.incidents.Get(ctx, inc.ID)
	if readErr != nil || !found {
		return
	}
	previousDetail := current.FreshMigrationDetail
	if err != nil {
		current.FreshMigrationDetail = "Migration paused at " + progress.Stage + ": " + err.Error() + ". The old repository and plan remain unchanged unless all target receipts and final capacity proof have passed."
		current.FreshMigrationStage = "migrating"
	} else if progress.Complete {
		current.FreshMigrationDetail = "All 14 target captures, scratch restore drills, and final capacity proof passed; the original plan now uses the selected destination."
	}
	current.UpdatedAt = f.now().UTC()
	if f.incidents.Put(ctx, current) != nil {
		return
	}
	if ask, askErr := f.asks.Get(ctx, current.FreshMigrationSelectionAskID); askErr == nil && ask != nil {
		if progress.Complete {
			ask.Extra["result_title"] = "Fresh repository migration completed"
			ask.Extra["result_body"] = "DBM verified captures and scratch restores for all 14 targets, rechecked destination capacity, and switched the original plan. The previous repository remains preserved."
			ask.Extra["result_category"] = categoryBackupRecovered + "-fresh-migration-complete"
		} else if err != nil {
			ask.Extra["result_title"] = "Fresh repository migration needs attention"
			ask.Extra["result_body"] = current.FreshMigrationDetail
			ask.Extra["result_category"] = categoryBackupNeedsAttention + "-fresh-migration-paused"
		} else {
			return
		}
		if err != nil && previousDetail == current.FreshMigrationDetail {
			return
		}
		ask.Extra["result_urgency"], ask.Extra["result_notification_pending"] = "high", "true"
		if f.asks.Put(ctx, ask) != nil {
			return
		}
		_ = f.publishPendingResult(ctx, ask)
		if progress.Complete {
			current.FreshMigrationStage = "complete"
			current.UpdatedAt = f.now().UTC()
			_ = f.incidents.Put(ctx, current)
		}
	}
}

func (f *BackupFailureFlow) WaitFreshMigrations(ctx context.Context) error {
	if f == nil {
		return nil
	}
	done := make(chan struct{})
	go func() { f.migrationWG.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// RetryPendingResultNotices replays the durable ask-result outbox. The
// lifecycle-managed worker calls it on each reconciliation pass so a process
// restart or transient Notification Hub outage cannot lose migration notices.
func (f *BackupFailureFlow) RetryPendingResultNotices(ctx context.Context) error {
	if f == nil || f.asks == nil {
		return nil
	}
	asks, err := f.asks.ListAsks(ctx)
	if err != nil {
		return fmt.Errorf("list pending backup result notices: %w", err)
	}
	var errs []error
	for _, ask := range asks {
		if ask.Extra["result_notification_pending"] != "true" {
			continue
		}
		if err := f.publishPendingResult(ctx, ask); err != nil {
			errs = append(errs, fmt.Errorf("publish backup result for ask %s: %w", ask.ID, err))
		}
	}
	return errors.Join(errs...)
}
