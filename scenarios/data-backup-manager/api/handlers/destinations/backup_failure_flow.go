package destinations

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"data-backup-manager/internal/destinationreadiness"
	"data-backup-manager/internal/destinations"
	"data-backup-manager/internal/engine"
	"data-backup-manager/internal/repohealth"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
)

const (
	backupFailureKind      = "backup_failure"
	backupNotificationType = "data_backup_manager.backup_health.notification.v1"
	// A backup-failure question stays outstanding for a week before DBM publishes
	// a fresh generation of it; the daily stalled-backup notice names it in
	// between, so it is never out of sight while unanswered.
	backupDecisionWindow = 7 * 24 * time.Hour
	// A declined or failed answer is asked again after a day.
	backupReaskAfter = 24 * time.Hour

	categoryBackupRecovered      = "backup-recovery-completed"
	categoryBackupNeedsAttention = "backup-recovery-needs-attention"
)

// BackupFailureFlow raises and applies the one question DBM asks when backups to
// a destination cannot proceed. It reuses the volume recovery machinery: a
// durable ask store and Raiser, the signed Notification Hub callback, the
// control-plane volume owner for drive repair, and a retryable result notice.
// An answer is applied only from a verified human identity.
type BackupFailureFlow struct {
	readiness        *destinationreadiness.Service
	volume           *RecoveryDecisionFlow
	recoveries       destinationreadiness.RecoveryStore
	asks             AskStore
	publisher        operatordecision.Publisher
	raiser           operatordecision.Raiser
	ledger           decisionLedger
	incidents        repohealth.IncidentStore
	exceptions       repohealth.ExceptionStore
	scanner          *repohealth.Scanner
	usage            interface{ InvalidateScan(string) }
	repoRepair       engine.RepoDamageRepair
	history          BackupHistory
	dests            destinations.Service
	migrations       FreshMigrationCoordinator
	migrationMu      sync.Mutex
	migrationRunning map[string]bool
	migrationWG      sync.WaitGroup
	now              func() time.Time
	logger           *log.Logger
	enabled          bool
}

// BackupFailureFlowDeps wires the flow. Every field is a seam.
type BackupFailureFlowDeps struct {
	Readiness  *destinationreadiness.Service
	Volume     *RecoveryDecisionFlow
	Recoveries destinationreadiness.RecoveryStore
	Asks       AskStore
	Publisher  operatordecision.Publisher
	Incidents  repohealth.IncidentStore
	Exceptions repohealth.ExceptionStore
	Scanner    *repohealth.Scanner
	Usage      interface{ InvalidateScan(string) }
	RepoRepair engine.RepoDamageRepair
	History    BackupHistory
	Dests      destinations.Service
	Migrations FreshMigrationCoordinator
	// Enabled is whether the signed callback is mounted; without it a question
	// could be asked but never answered, so none is raised.
	Enabled bool
	Now     func() time.Time
	Logger  *log.Logger
}

func newBackupFailureFlow(d BackupFailureFlowDeps) *BackupFailureFlow {
	logger := d.Logger
	if logger == nil {
		logger = log.Default()
	}
	now := d.Now
	if now == nil {
		now = time.Now
	}
	f := &BackupFailureFlow{
		readiness: d.Readiness, volume: d.Volume, recoveries: d.Recoveries, asks: d.Asks, publisher: d.Publisher,
		incidents: d.Incidents, exceptions: d.Exceptions, scanner: d.Scanner, usage: d.Usage,
		repoRepair: d.RepoRepair,
		history:    d.History, dests: d.Dests, migrations: d.Migrations, migrationRunning: map[string]bool{}, now: now, logger: logger, enabled: d.Enabled,
	}
	f.raiser = operatordecision.Raiser{
		Source: recoveryDecisionSource, Kind: backupFailureKind, Store: d.Asks, Publisher: d.Publisher,
		Clock: backupFlowClock{f}, MaxOutstanding: 3, Window: backupDecisionWindow,
	}
	f.ledger = decisionLedger{asks: d.Asks, raiser: f.raiser, publisher: d.Publisher, clock: func() time.Time { return f.now() }, logger: logger}
	return f
}

type backupFlowClock struct{ f *BackupFailureFlow }

func (c backupFlowClock) Now() time.Time { return c.f.now() }

func (f *BackupFailureFlow) ready() bool {
	return f != nil && f.enabled && f.publisher != nil && f.asks != nil && f.incidents != nil
}

func backupAskID(incidentID string, generation int) string {
	return operatordecision.CorrelationID("dbm-backup-failure", incidentID, strconv.Itoa(generation))
}

// Raise publishes the incident's question under the destination's subject key,
// idempotently. While the question is outstanding this is a no-op until the
// re-assertion interval passes, when it is re-sent so the hub keeps it open; a
// changed diagnosis is a revision of the same open decision at the hub, never a
// second one; and a publication that failed is retried here until it succeeds.
// The facts come from the diagnosis stored with the incident.
func (f *BackupFailureFlow) Raise(ctx context.Context, inc repohealth.Incident) error {
	if !f.ready() {
		return errors.New("Notification Hub decision delivery is not configured")
	}
	facts := repohealth.BuildFacts(inc.Diagnosis, inc.Offers, inc.ID, inc.Generation)
	// The hub closes a non-critical question unanswered at its deadline (a day by
	// default). Hold this one open for the full window; a stored time keeps the
	// facts identical across publication retries.
	asked := inc.AskedAt
	if asked.IsZero() {
		asked = inc.CreatedAt
	}
	if !asked.IsZero() {
		facts.Deadline = asked.UTC().Add(backupDecisionWindow)
	}
	if err := facts.Validate(); err != nil {
		return fmt.Errorf("validate backup decision facts: %w", err)
	}
	ask := operatordecision.PendingAsk{
		ID: backupAskID(inc.ID, inc.Generation), Subject: inc.ID,
		Extra: map[string]string{
			"incident_id": inc.ID, "generation": strconv.Itoa(inc.Generation), "destination_id": inc.DestinationID,
		},
	}
	if err := f.ledger.raise(ctx, ask, facts, backupSubjectKey(inc.DestinationID)); err != nil {
		return fmt.Errorf("raise backup failure decision: %w", err)
	}
	return nil
}

// Settle reconciles Notification Hub with the incident ledger, and is safe to
// run on every sweep. A question is wanted only while an incident for its
// destination is awaiting a decision and it is that incident's current
// generation. Any other pending question is withdrawn with the reason in plain
// words: backups work again, the problem changed, the destination or plan went
// away, or the operator dealt with it in DBM. A question published before
// subject keys existed is closed by its event id. When a newer question stands
// for the same destination the hub already holds it under the same key, so the
// older one is only retired here.
func (f *BackupFailureFlow) Settle(ctx context.Context) error {
	if !f.ready() {
		return nil
	}
	asks, err := f.asks.ListAsks(ctx)
	if err != nil {
		return fmt.Errorf("list backup decisions: %w", err)
	}
	incidents, err := f.incidents.List(ctx)
	if err != nil {
		return fmt.Errorf("list backup incidents: %w", err)
	}
	byID := make(map[string]repohealth.Incident, len(incidents))
	awaiting := map[string]repohealth.Incident{}
	for _, inc := range incidents {
		byID[inc.ID] = inc
		if inc.State == repohealth.IncidentAwaiting {
			awaiting[inc.DestinationID] = inc
		}
	}
	var errs []error
	for _, ask := range asks {
		if ask.State != operatordecision.PendingStatePending {
			continue
		}
		inc, found := byID[ask.Extra["incident_id"]]
		generation, _ := strconv.Atoi(ask.Extra["generation"])
		if ask.Extra["decision_type"] == freshDestinationDecisionType && found && inc.Generation == generation && inc.State == repohealth.IncidentAgentRequested && inc.FreshMigrationID == ask.Extra["migration_id"] && (inc.FreshMigrationStage == "awaiting_destination_selection" || inc.FreshMigrationStage == "selection_recorded" || inc.FreshMigrationStage == "migrating") {
			continue
		}
		if ask.Extra["decision_type"] == freshDestinationRegistrationDecisionType && found && inc.Generation == generation && inc.State == repohealth.IncidentAgentRequested && inc.FreshMigrationID == ask.Extra["migration_id"] && inc.FreshMigrationRegistrationAskID == ask.ID && (inc.FreshMigrationStage == "local_destination_proposed" || inc.FreshMigrationStage == "local_destination_registration_pending" || inc.FreshMigrationStage == "awaiting_destination_selection" || inc.FreshMigrationStage == "migrating" || inc.FreshMigrationStage == "complete") {
			continue
		}
		current, replaced := awaiting[ask.Extra["destination_id"]]
		if found && replaced && inc.ID == current.ID && inc.Generation == generation {
			if err := f.ledger.finishLegacy(ctx, ask); err != nil {
				errs = append(errs, err)
			}
			continue
		}
		if err := f.ledger.withdraw(ctx, ask, backupWithdrawalReason(inc, generation, found, replaced), !replaced); err != nil {
			errs = append(errs, err)
		}
	}
	for _, inc := range incidents {
		if inc.State == repohealth.IncidentAgentRequested && inc.FreshMigrationStage == "local_destination_proposed" {
			if err := f.askFreshDestinationRegistration(ctx, inc, inc.FreshMigrationRegistrationGeneration); err != nil {
				errs = append(errs, fmt.Errorf("reconcile local destination confirmation %s: %w", inc.FreshMigrationID, err))
			}
		}
		if inc.State == repohealth.IncidentAgentRequested && inc.FreshMigrationStage == "local_destination_registration_pending" {
			_, registrationErr := f.completeFreshDestinationRegistration(ctx, inc)
			if registrationErr != nil {
				f.logger.Printf("fresh destination registration remains pending incident=%s: %v", inc.ID, registrationErr)
			} else if ask, askErr := f.asks.Get(ctx, inc.FreshMigrationRegistrationAskID); askErr != nil {
				errs = append(errs, askErr)
			} else if ask != nil {
				if err := f.recordFreshRegistrationResult(ctx, ask, inc.FreshMigrationRegistrationBy); err != nil {
					errs = append(errs, err)
				}
			}
		}
		if inc.State == repohealth.IncidentAgentRequested {
			if err := f.settleFreshSelectionOutbox(ctx, inc); err != nil {
				errs = append(errs, fmt.Errorf("settle destination selection result %s: %w", inc.FreshMigrationID, err))
			}
		}
		if inc.State == repohealth.IncidentAgentRequested && inc.FreshMigrationStage == "awaiting_destination_selection" && inc.FreshMigrationRegisteredDestID != "" && inc.FreshMigrationRegistrationAskID != "" {
			if ask, err := f.asks.Get(ctx, inc.FreshMigrationRegistrationAskID); err != nil {
				errs = append(errs, err)
			} else if ask != nil && ask.State == operatordecision.PendingStatePending {
				if err := f.recordFreshRegistrationResult(ctx, ask, inc.FreshMigrationRegistrationBy); err != nil {
					errs = append(errs, err)
				}
			}
		}
		if inc.State == repohealth.IncidentAgentRequested && inc.FreshMigrationStage == "selection_deferred" {
			if inc.FreshMigrationSelectionRetryAt.IsZero() || !f.now().Before(inc.FreshMigrationSelectionRetryAt) {
				inc.FreshMigrationStage = "awaiting_destination_selection"
				inc.FreshMigrationSelectionRetryAt = time.Time{}
				inc.UpdatedAt = f.now().UTC()
				if err := f.incidents.Put(ctx, inc); err != nil {
					errs = append(errs, fmt.Errorf("resume deferred destination selection %s: %w", inc.FreshMigrationID, err))
				}
			}
		}
		if inc.State == repohealth.IncidentAgentRequested && inc.FreshMigrationStage == "awaiting_destination_selection" {
			if _, err := f.askFreshDestination(ctx, inc); err != nil {
				errs = append(errs, err)
			}
		}
		if inc.State == repohealth.IncidentAgentRequested && inc.FreshMigrationStage == "selection_recorded" && inc.FreshMigrationID != "" {
			if f.migrations == nil {
				continue
			}
			if err := f.migrations.Begin(ctx, inc.FreshMigrationID, inc.FreshMigrationPlanID, inc.DestinationID, inc.FreshMigrationDestinationID, inc.FreshMigrationTargets); err != nil {
				errs = append(errs, fmt.Errorf("resume persisted migration selection %s: %w", inc.FreshMigrationID, err))
				continue
			}
			inc.FreshMigrationStage = "migrating"
			inc.FreshMigrationDetail = "The selected destination and exact target set are durable; DBM is resuming migration through its owner workflow."
			inc.UpdatedAt = f.now().UTC()
			if err := f.incidents.Put(ctx, inc); err != nil {
				errs = append(errs, err)
			}
		}
		if inc.State == repohealth.IncidentAgentRequested && inc.FreshMigrationStage == "migrating" && inc.FreshMigrationSelectionAskID != "" {
			selectionAsk, askErr := f.asks.Get(ctx, inc.FreshMigrationSelectionAskID)
			if askErr != nil {
				errs = append(errs, askErr)
			} else if selectionAsk != nil && selectionAsk.State == operatordecision.PendingStatePending {
				if err := f.finish(ctx, selectionAsk, operatordecision.PendingStateApproved, inc.FreshMigrationSelectionBy, noticeText{title: "Fresh repository migration started", body: "DBM recovered the durable destination selection and migration journal after restart. It will verify all 14 captures and scratch restores before changing the original plan. The old repository remains unchanged.", category: categoryBackupNeedsAttention, urgency: "normal"}); err != nil {
					errs = append(errs, err)
				}
			}
		}
	}
	if err := f.resumeFreshMigrations(ctx); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// backupWithdrawalReason is what the operator reads on a closed question.
func backupWithdrawalReason(inc repohealth.Incident, generation int, found, replaced bool) string {
	drive := inc.Diagnosis.DriveSentence()
	switch {
	case !found:
		return "This question no longer matches anything DBM has on record, so it was closed."
	case replaced:
		return withdrawnReplacedReason
	case inc.State == repohealth.IncidentResolved:
		outcome := "succeeded again"
		if inc.Result == resolvedCleared {
			outcome = "are no longer blocked"
		}
		if inc.ResolvedAt.IsZero() {
			return "Backups to " + drive + " " + outcome + ". Nothing more is needed from you."
		}
		return "Backups to " + drive + " " + outcome + " at " + inc.ResolvedAt.UTC().Format("15:04 UTC") + ". Nothing more is needed from you."
	case inc.State == repohealth.IncidentSuperseded:
		if strings.TrimSpace(inc.Result) != "" {
			return inc.Result
		}
		return "The problem with backups to " + drive + " changed, so this question no longer applies."
	case generation > 0 && generation < inc.Generation:
		return "This was an earlier round of the question about backups to " + drive + "; DBM has since moved on to a newer round, so this one was closed."
	case inc.State == repohealth.IncidentDeclined:
		return "You dismissed or declined this question, so it was closed. DBM asks again after a day if backups to " + drive + " are still blocked."
	case inc.State == repohealth.IncidentFailed:
		return "The last action taken for " + drive + " did not work, so this question was closed. DBM asks again after a day if backups are still blocked."
	case inc.State == repohealth.IncidentLapsed:
		return "This question expired unanswered, so it was closed. DBM asks again if backups to " + drive + " are still blocked."
	case inc.State == repohealth.IncidentApplying:
		return "The action you approved for " + drive + " is being carried out, so this question was closed."
	case inc.State == repohealth.IncidentApplied:
		return "The action you approved for " + drive + " was carried out, so this question was closed. DBM asks again if backups still fail."
	case inc.State == repohealth.IncidentAgentRequested:
		return "You asked for an agent to set up a fresh backup set for " + drive + ", so this question was closed."
	default:
		return "This question no longer matches the state of backups to " + drive + ", so it was closed."
	}
}

// Handles identifies the one decision producer this flow owns.
func (f *BackupFailureFlow) Handles(sourceEventType string) bool {
	return sourceEventType == operatordecision.DecisionRequestedEventType(recoveryDecisionSource, backupFailureKind)
}

type backupDecisionRef struct {
	incidentID string
	generation int
	askID      string
}

func parseBackupDecisionRef(res operatordecision.DecisionResolution) (backupDecisionRef, bool) {
	id := strings.TrimSpace(res.Correlation["incident_id"])
	gen, err := strconv.Atoi(strings.TrimSpace(res.Correlation["generation"]))
	if id == "" || err != nil || gen < 1 {
		return backupDecisionRef{}, false
	}
	return backupDecisionRef{incidentID: id, generation: gen, askID: backupAskID(id, gen)}, true
}

// Apply carries out a resolved answer. Replays and unknown asks are benign
// no-ops; only a verified human answer to an option that was offered changes
// anything. A resolution that says the hub withdrew the decision is never an
// answer. An answer that names a question DBM had already withdrawn, or an
// older generation of the same condition, is never carried out (it may be a
// replay of an approval that was already used) and is never silent: the
// operator is told once that nothing was changed.
func (f *BackupFailureFlow) Apply(ctx context.Context, res operatordecision.DecisionResolution) (bool, error) {
	if !f.ready() {
		return false, nil
	}
	if res.Correlation["decision_type"] == freshDestinationDecisionType {
		return f.applyFreshDestination(ctx, res)
	}
	if res.Correlation["decision_type"] == freshDestinationRegistrationDecisionType {
		return f.applyFreshDestinationRegistration(ctx, res)
	}
	ref, ok := parseBackupDecisionRef(res)
	if !ok {
		return false, nil
	}
	ask, err := f.asks.Get(ctx, ref.askID)
	if err != nil {
		return true, err
	}
	if ask == nil || ask.Extra["incident_id"] != ref.incidentID || ask.Extra["generation"] != strconv.Itoa(ref.generation) {
		return true, nil
	}
	if res.State == operatordecision.StateWithdrawn {
		return true, f.ledger.hubWithdrew(ctx, ask, res)
	}
	if ask.State == operatordecision.PendingStateWithdrawn {
		return true, f.answerToWithdrawn(ctx, ask, res)
	}
	if ask.State != operatordecision.PendingStatePending {
		f.logger.Printf("backup failure decision replay ignored incident=%s local_state=%s", ref.incidentID, ask.State)
		if ask.Extra["result_notification_pending"] == "true" {
			return true, f.publishPendingResult(ctx, ask)
		}
		return true, nil
	}
	inc, found, err := f.incidents.Get(ctx, ref.incidentID)
	if err != nil {
		return true, err
	}
	if found && !inc.State.Open() {
		return true, f.finish(ctx, ask, operatordecision.PendingStateRejected, "", noticeText{
			title: "Backup question was already settled", body: "Backups to " + inc.Diagnosis.DriveSentence() + " were already working again (or the problem changed) when your answer arrived, so nothing was changed.",
			category: categoryBackupNeedsAttention, urgency: "normal",
		})
	}
	if !found || inc.Generation != ref.generation {
		body := "This question no longer matches what is on record, so nothing was changed. If backups are still blocked I will ask again."
		if found && inc.State == repohealth.IncidentAwaiting {
			body = "Your answer was for an earlier version of this question, so nothing was changed. The question waiting for you now is the current one."
		}
		return true, f.finish(ctx, ask, operatordecision.PendingStateRejected, "", noticeText{
			title: "Backup question is out of date", body: body,
			category: categoryBackupNeedsAttention, urgency: "high",
		})
	}
	return true, f.route(ctx, ask, inc, res)
}

// answerToWithdrawn handles an answer to a question DBM had already withdrawn:
// nothing is carried out, and the operator is told once.
func (f *BackupFailureFlow) answerToWithdrawn(ctx context.Context, ask *operatordecision.PendingAsk, res operatordecision.DecisionResolution) error {
	if res.State != operatordecision.StateAnswered {
		return nil
	}
	f.logger.Printf("backup failure answer to a withdrawn question ignored incident=%s", ask.Extra["incident_id"])
	body := "Your answer arrived after this question was closed because the situation changed, so nothing was changed. If backups are still blocked I will ask again."
	if current, ok := f.awaitingFor(ctx, ask.Extra["destination_id"]); ok {
		body = "Your answer was for a question that had already been replaced, so nothing was changed. The question waiting for you now is the current one: " + current.Subject + "."
	}
	return f.ledger.lateAnswer(ctx, ask, backupNotification("Backup question was already closed", body,
		categoryBackupNeedsAttention, "normal", "late-answer:"+ask.ID, f.now()))
}

// awaitingFor is the incident whose question is in front of the operator for a
// destination, if any.
func (f *BackupFailureFlow) awaitingFor(ctx context.Context, destinationID string) (repohealth.Incident, bool) {
	all, err := f.incidents.List(ctx)
	if err != nil {
		return repohealth.Incident{}, false
	}
	for _, inc := range all {
		if inc.DestinationID == destinationID && inc.State == repohealth.IncidentAwaiting {
			return inc, true
		}
	}
	return repohealth.Incident{}, false
}

// lapse closes a question the hub expired unanswered. Nothing was answered and
// nothing is changed; the monitor asks again at once, so a question is always
// outstanding while backups are blocked. No result notice is sent: the daily
// stalled-backup notice and the fresh question already say so.
func (f *BackupFailureFlow) lapse(ctx context.Context, ask *operatordecision.PendingAsk, inc repohealth.Incident) error {
	if err := inc.Transition(repohealth.IncidentLapsed); err != nil {
		return err
	}
	inc.UpdatedAt, inc.Result = f.now().UTC(), "the question expired unanswered"
	if err := f.incidents.Put(ctx, inc); err != nil {
		return fmt.Errorf("record lapsed backup question: %w", err)
	}
	if ask.Extra == nil {
		ask.Extra = map[string]string{}
	}
	now := f.now().UTC()
	ask.State, ask.ResolvedAt, ask.ResolvedBy = operatordecision.PendingStateRejected, &now, ""
	ask.Extra["result_notification_pending"] = "false"
	return f.asks.Put(ctx, ask)
}

func (f *BackupFailureFlow) route(ctx context.Context, ask *operatordecision.PendingAsk, inc repohealth.Incident, res operatordecision.DecisionResolution) error {
	if res.State == operatordecision.StateExpired {
		return f.lapse(ctx, ask, inc)
	}
	if !isVerifiedRepairApproval(res) {
		return f.decline(ctx, ask, inc, res, backupDeniedDetail(res))
	}
	switch res.Answer {
	case repohealth.OptAccept:
		if !inc.Offers.Accept {
			return f.decline(ctx, ask, inc, res, "Skipping unreadable files was not one of the choices offered, so nothing was changed.")
		}
		return f.applyAccept(ctx, ask, inc, res)
	case repohealth.OptKopiaQuarantine:
		if !inc.Offers.KopiaQuarantine {
			return f.decline(ctx, ask, inc, res, "Kopia quarantine was not one of the choices offered, so nothing was changed.")
		}
		return f.applyKopiaQuarantine(ctx, ask, inc, res)
	case repohealth.OptRepair:
		if !inc.Offers.Repair {
			return f.decline(ctx, ask, inc, res, "A repair was not one of the choices offered, so nothing was changed.")
		}
		return f.applyRepair(ctx, ask, inc, res)
	case repohealth.OptFresh:
		if !inc.Offers.Fresh {
			return f.decline(ctx, ask, inc, res, "A fresh backup set was not one of the choices offered, so nothing was changed.")
		}
		return f.applyFresh(ctx, ask, inc, res)
	case repohealth.OptLeave:
		return f.decline(ctx, ask, inc, res, "You chose to leave backups stopped for now.")
	case "other":
		return f.decline(ctx, ask, inc, res, "You answered Other"+noteSuffix(res)+". Nothing was changed.")
	default:
		return f.decline(ctx, ask, inc, res, "That answer was not one of the choices offered, so nothing was changed.")
	}
}

func (f *BackupFailureFlow) applyKopiaQuarantine(ctx context.Context, ask *operatordecision.PendingAsk, inc repohealth.Incident, res operatordecision.DecisionResolution) error {
	if f.repoRepair == nil || f.dests == nil || !repohealth.OffersFor(inc.Diagnosis, false).KopiaQuarantine {
		return f.decline(ctx, ask, inc, res, "Kopia's identity-bound exact-temp quarantine is unavailable for this incident, so nothing was changed.")
	}
	if inc.State != repohealth.IncidentApplying {
		if err := f.beginApplying(ctx, &inc, res); err != nil {
			return fmt.Errorf("record Kopia quarantine in progress: %w", err)
		}
	} else if !sameIncidentApproval(inc, res) {
		return errors.New("persisted applying incident does not match its verified approval")
	}
	return f.resumeKopiaQuarantine(ctx, ask, inc, res)
}

func (f *BackupFailureFlow) resumeKopiaQuarantine(ctx context.Context, ask *operatordecision.PendingAsk, inc repohealth.Incident, res operatordecision.DecisionResolution) error {
	scanner, entries, err := f.validateKopiaQuarantineResume(ask, inc, res)
	if err != nil {
		return f.requireApplyingReview(ctx, ask, inc, err.Error())
	}
	var quarantined int
	err = f.dests.WithResolvedDestination(ctx, inc.DestinationID, func(current destinations.Destination) error {
		if err := f.validateKopiaQuarantineDestination(ctx, scanner, inc, current, entries); err != nil {
			return err
		}
		for _, entry := range entries {
			result, repairErr := f.repoRepair.RepoQuarantineTempBlob(ctx, current.Name, entry.RelPath)
			if repairErr != nil {
				return fmt.Errorf("quarantine %s: %w", entry.RelPath, repairErr)
			}
			if !validQuarantineResult(result, current.Name, current.RepositoryLocation, entry.RelPath) {
				return fmt.Errorf("quarantine owner returned an incomplete or mismatched result for %s", entry.RelPath)
			}
			if !containsString(inc.QuarantinedPaths, entry.RelPath) {
				inc.QuarantinedPaths = append(inc.QuarantinedPaths, entry.RelPath)
				sort.Strings(inc.QuarantinedPaths)
				inc.UpdatedAt = f.now().UTC()
				if putErr := f.incidents.Put(ctx, inc); putErr != nil {
					return fmt.Errorf("record quarantined path %s: %w", entry.RelPath, putErr)
				}
			}
			quarantined++
		}
		return f.verifyKopiaQuarantineCleared(ctx, scanner, inc)
	})
	if err != nil {
		return f.requireApplyingReview(ctx, ask, inc, "The approved Kopia quarantine did not finish safely: "+err.Error()+" The incident remains applying and will retry only after revalidating the same device and exact paths.")
	}
	started := f.retryPlansAfter(ctx, inc, inc.AnsweredAt)
	if err := inc.Transition(repohealth.IncidentApplied); err != nil {
		return err
	}
	inc.UpdatedAt = f.now().UTC()
	inc.Result = fmt.Sprintf("Kopia quarantined %d exact orphan session temps on the same volume; their bytes were preserved and no final blobs were written", quarantined)
	if err := f.incidents.Put(ctx, inc); err != nil {
		return fmt.Errorf("record Kopia quarantine result: %w", err)
	}
	body := fmt.Sprintf("Kopia moved %d exact unreadable session temps into same-volume quarantine. Their bytes remain preserved; no final repository blobs were written. I retried %d backup plan(s).", quarantined, started)
	return f.finish(ctx, ask, operatordecision.PendingStateApproved, res.AnsweredBy, noticeText{title: "Kopia repository temp quarantined", body: body, category: categoryBackupRecovered, urgency: "normal"})
}

func (f *BackupFailureFlow) validateKopiaQuarantineResume(ask *operatordecision.PendingAsk, inc repohealth.Incident, res operatordecision.DecisionResolution) (engine.RepoDamageScanner, []repohealth.Entry, error) {
	if ask == nil || (ask.State != operatordecision.PendingStatePending && ask.State != operatordecision.PendingStateApproved) {
		return nil, nil, errors.New("The Kopia quarantine approval record is unavailable. Inspect the incident before continuing.")
	}
	if res.Answer != repohealth.OptKopiaQuarantine {
		return nil, nil, errors.New("The persisted Kopia quarantine approval no longer names this exact operation. Inspect the incident before continuing.")
	}
	if inc.Diagnosis.Cause != repohealth.CauseLeftoverTempUnreadable || !repohealth.OffersFor(inc.Diagnosis, false).KopiaQuarantine {
		return nil, nil, errors.New("The Kopia quarantine incident no longer matches an exact set of orphan session temps on an identified drive. Inspect the incident before continuing.")
	}
	if f.repoRepair == nil || f.dests == nil || inc.Diagnosis.DestinationDevice == nil || !inc.Diagnosis.DestinationDevice.StrongIdentity() {
		return nil, nil, errors.New("The Kopia quarantine owner or stable drive identity is unavailable. Inspect the incident before continuing.")
	}
	scanner, ok := f.usage.(engine.RepoDamageScanner)
	if !ok {
		return nil, nil, errors.New("Kopia's typed damage scan is unavailable for revalidation. Inspect the incident before continuing.")
	}
	entries := append([]repohealth.Entry(nil), inc.Diagnosis.Unreadable...)
	sort.Slice(entries, func(i, j int) bool { return entries[i].RelPath < entries[j].RelPath })
	allowed := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		allowed[entry.RelPath] = struct{}{}
	}
	for _, rel := range inc.QuarantinedPaths {
		if _, exists := allowed[rel]; !exists {
			return nil, nil, errors.New("The persisted quarantine progress contains a path outside the approved set. Inspect the incident before continuing.")
		}
	}
	return scanner, entries, nil
}

func (f *BackupFailureFlow) validateKopiaQuarantineDestination(ctx context.Context, scanner engine.RepoDamageScanner, inc repohealth.Incident, current destinations.Destination, entries []repohealth.Entry) error {
	identityMatches := current.ID == inc.DestinationID && current.Name == inc.Diagnosis.DestinationName &&
		current.BackendKind == destinations.BackendFilesystem && current.DeviceIdentity != nil &&
		current.DeviceIdentity.StrongIdentity() && current.DeviceIdentity.MatchesDevice(*inc.Diagnosis.DestinationDevice) &&
		filepath.Clean(current.RepositoryLocation) == filepath.Clean(inc.Diagnosis.RepositoryRoot)
	if !identityMatches {
		return errors.New("destination identity or repository root changed after the operator reviewed the quarantine set")
	}
	if f.usage != nil {
		f.usage.InvalidateScan(inc.Diagnosis.RepositoryRoot)
	}
	scan, err := scanner.RepoDamageScan(ctx, inc.Diagnosis.DestinationName)
	if err != nil || filepath.Clean(scan.Root) != filepath.Clean(inc.Diagnosis.RepositoryRoot) || !onlyApprovedQuarantineEntries(scan.Unreadable, entries) {
		return errors.New("Kopia's current typed damage scan changed or is incomplete")
	}
	return nil
}

func (f *BackupFailureFlow) verifyKopiaQuarantineCleared(ctx context.Context, scanner engine.RepoDamageScanner, inc repohealth.Incident) error {
	if f.usage != nil {
		f.usage.InvalidateScan(inc.Diagnosis.RepositoryRoot)
	}
	after, err := scanner.RepoDamageScan(ctx, inc.Diagnosis.DestinationName)
	if err != nil || filepath.Clean(after.Root) != filepath.Clean(inc.Diagnosis.RepositoryRoot) || len(after.Unreadable) != 0 {
		return errors.New("Kopia damage scan still reports unreadable entries after the approved quarantine set")
	}
	return nil
}

func onlyApprovedQuarantineEntries(observed []engine.RepoDamageEntry, approved []repohealth.Entry) bool {
	allowed := make(map[string]repohealth.Entry, len(approved))
	for _, entry := range approved {
		allowed[entry.RelPath] = entry
	}
	for _, entry := range observed {
		want, ok := allowed[entry.RelativePath]
		if !ok || entry.Class != string(want.Class) || entry.BlobKind != string(want.BlobKind) || entry.Problem != string(want.Problem) || !entry.Orphan {
			return false
		}
	}
	return true
}

func validQuarantineResult(result engine.RepoQuarantineResult, repo, root, relativePath string) bool {
	expectedOriginal := filepath.Clean(filepath.Join(root, filepath.FromSlash(relativePath)))
	if result.Repository != repo || filepath.Clean(result.OriginalPath) != expectedOriginal || !result.Committed ||
		!result.QuarantineSameVolume || result.SourceDataDeleted || result.FinalBlobWritten || !filepath.IsAbs(result.QuarantinePath) {
		return false
	}
	relativeQuarantine, err := filepath.Rel(filepath.Dir(filepath.Clean(root)), filepath.Clean(result.QuarantinePath))
	if err != nil {
		return false
	}
	parts := strings.Split(filepath.ToSlash(relativeQuarantine), "/")
	if len(parts) < 2 || !strings.HasPrefix(parts[0], filepath.Base(filepath.Clean(root))+"-recovery-quarantine-") ||
		strings.HasPrefix(filepath.ToSlash(relativeQuarantine), "../") {
		return false
	}
	return strings.Join(parts[1:], "/") == filepath.ToSlash(relativePath)
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func noteSuffix(res operatordecision.DecisionResolution) string {
	if note := strings.TrimSpace(res.Note); note != "" {
		return " (" + note + ")"
	}
	return ""
}

func backupDeniedDetail(res operatordecision.DecisionResolution) string {
	if res.State == operatordecision.StateDismissed {
		return "You dismissed the question without answering, so nothing was changed."
	}
	return "The answer did not come from a verified person, so nothing was changed."
}

func (f *BackupFailureFlow) decline(ctx context.Context, ask *operatordecision.PendingAsk, inc repohealth.Incident, res operatordecision.DecisionResolution, detail string) error {
	if err := inc.Transition(repohealth.IncidentDeclined); err != nil {
		return err
	}
	inc.DeclinedAt, inc.Answer, inc.UpdatedAt = f.now().UTC(), res.Answer, f.now().UTC()
	inc.Result = detail
	if err := f.incidents.Put(ctx, inc); err != nil {
		return fmt.Errorf("record declined backup decision: %w", err)
	}
	return f.finish(ctx, ask, operatordecision.PendingStateRejected, "", noticeText{
		title:    "Backups to " + inc.Diagnosis.DriveSentence() + " stay stopped",
		body:     detail + " I will ask again tomorrow and keep reminding you every day while backups are stopped.",
		category: categoryBackupNeedsAttention, urgency: "high",
	})
}

// applyAccept records an exception for each exact unreadable path the operator
// was shown, after re-checking each against the drive as it is now. A path that
// is no longer an unreadable leftover temporary file of the same kind is not
// recorded; the retry run then blocks on it and asks again.
func (f *BackupFailureFlow) applyAccept(ctx context.Context, ask *operatordecision.PendingAsk, inc repohealth.Incident, res operatordecision.DecisionResolution) error {
	if f.exceptions == nil || inc.Diagnosis.Cause != repohealth.CauseLeftoverTempUnreadable {
		return f.decline(ctx, ask, inc, res, "Skipping files only applies to unreadable leftover temporary files, so nothing was changed.")
	}
	if inc.State != repohealth.IncidentApplying {
		if err := f.beginApplying(ctx, &inc, res); err != nil {
			return fmt.Errorf("record accept in progress: %w", err)
		}
	} else if !sameIncidentApproval(inc, res) {
		return errors.New("persisted applying incident does not match its verified approval")
	}
	return f.resumeAccept(ctx, ask, inc, res)
}

func (f *BackupFailureFlow) resumeAccept(ctx context.Context, ask *operatordecision.PendingAsk, inc repohealth.Incident, res operatordecision.DecisionResolution) error {
	fsys := f.scanner.Filesystem()
	var recorded, changed []string
	for _, entry := range inc.Diagnosis.Unreadable {
		info, err := fsys.Lstat(filepath.Join(inc.Diagnosis.RepositoryRoot, filepath.FromSlash(entry.RelPath)))
		switch {
		case err != nil && repohealth.ProblemFor(err) == entry.Problem && entry.Class == repohealth.ClassLeftoverTemp:
			ex := repohealth.Exception{
				DestinationID: inc.DestinationID, RelPath: entry.RelPath, Class: entry.Class, BlobKind: entry.BlobKind,
				Problem: entry.Problem, ApprovedBy: res.AnsweredBy, ApprovedBySource: res.AnsweredByIdentitySource,
				ApprovedAt: f.now().UTC(), AskID: ask.ID,
			}
			if addErr := f.exceptions.Add(ctx, ex); addErr != nil {
				return f.failApplied(ctx, ask, inc, res, "I could not record your approval: "+addErr.Error())
			}
			recorded = append(recorded, entry.RelPath)
		case err == nil && info.Mode().IsRegular():
			// Readable again: nothing to except.
		default:
			changed = append(changed, entry.RelPath)
		}
	}
	if f.usage != nil {
		f.usage.InvalidateScan(inc.Diagnosis.RepositoryRoot)
	}
	inc.ExceptionPaths = recorded
	inc.UpdatedAt = f.now().UTC()
	if err := f.incidents.Put(ctx, inc); err != nil {
		return fmt.Errorf("record accepted exception paths: %w", err)
	}
	if len(recorded) == 0 && len(changed) > 0 {
		return f.failApplied(ctx, ask, inc, res, "The unreadable files changed since I asked, so I recorded nothing. I will ask again.")
	}
	started := f.retryPlansAfter(ctx, inc, inc.AnsweredAt)
	if err := inc.Transition(repohealth.IncidentApplied); err != nil {
		return err
	}
	inc.UpdatedAt = f.now().UTC()
	inc.Result = fmt.Sprintf("recorded %d exception(s)", len(recorded))
	if err := f.incidents.Put(ctx, inc); err != nil {
		return fmt.Errorf("record applied accept: %w", err)
	}
	body := fmt.Sprintf("I recorded your approval to skip %d unreadable leftover %s. Nothing on the drive was touched.", len(recorded), operatordecision.Plural(len(recorded), "file", "files"))
	if len(changed) > 0 {
		body += fmt.Sprintf(" %d other %s changed since I asked and %s not covered; if that blocks backups I will ask again.", len(changed), operatordecision.Plural(len(changed), "file", "files"), operatordecision.Plural(len(changed), "is", "are"))
	}
	if started > 0 {
		body += " A backup was started now. I will warn you each day while these files remain; any other unreadable file stops backups and I will ask again."
	} else {
		body += " No backup could be started right now; the next scheduled run will use it."
	}
	return f.finish(ctx, ask, operatordecision.PendingStateApproved, res.AnsweredBy, noticeText{
		title: "Backups to " + inc.Diagnosis.DriveSentence() + " can resume", body: body,
		category: categoryBackupRecovered, urgency: "normal",
	})
}

func (f *BackupFailureFlow) failApplied(ctx context.Context, ask *operatordecision.PendingAsk, inc repohealth.Incident, res operatordecision.DecisionResolution, detail string) error {
	if inc.State == repohealth.IncidentAwaiting {
		if err := f.beginApplying(ctx, &inc, res); err != nil {
			return err
		}
	}
	if inc.State != repohealth.IncidentFailed {
		if err := inc.Transition(repohealth.IncidentFailed); err != nil {
			return err
		}
	}
	if inc.Answer == "" {
		inc.Answer, inc.AnsweredBy, inc.AnsweredAt = res.Answer, res.AnsweredBy, f.now().UTC()
	}
	inc.UpdatedAt = f.now().UTC()
	inc.Result = detail
	if err := f.incidents.Put(ctx, inc); err != nil {
		return fmt.Errorf("record failed backup action: %w", err)
	}
	return f.finish(ctx, ask, operatordecision.PendingStateRejected, res.AnsweredBy, noticeText{
		title: "Backup recovery needs attention", body: detail,
		category: categoryBackupNeedsAttention, urgency: "high",
	})
}

func sameIncidentApproval(inc repohealth.Incident, res operatordecision.DecisionResolution) bool {
	return inc.Answer == res.Answer && inc.AnsweredBy == res.AnsweredBy &&
		inc.AnsweredByIdentityKind == res.AnsweredByIdentityKind && inc.AnsweredByIdentitySource == res.AnsweredByIdentitySource &&
		inc.AnsweredAt.Equal(parseDecisionTime(res.AnsweredAt))
}

func parseDecisionTime(value string) time.Time {
	parsed, _ := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
	return parsed
}

func (f *BackupFailureFlow) beginApplying(ctx context.Context, inc *repohealth.Incident, res operatordecision.DecisionResolution) error {
	if inc == nil || !isVerifiedRepairApproval(res) {
		return errors.New("applying backup recovery requires a verified human approval")
	}
	answeredAt := parseDecisionTime(res.AnsweredAt)
	if answeredAt.IsZero() {
		answeredAt = parseDecisionTime(res.ResolvedAt)
	}
	if answeredAt.IsZero() {
		answeredAt = f.now().UTC()
	}
	if err := inc.Transition(repohealth.IncidentApplying); err != nil {
		return err
	}
	inc.Answer, inc.AnsweredBy = res.Answer, res.AnsweredBy
	inc.AnsweredByIdentityKind, inc.AnsweredByIdentitySource = res.AnsweredByIdentityKind, res.AnsweredByIdentitySource
	inc.AnsweredAt, inc.UpdatedAt = answeredAt.UTC(), f.now().UTC()
	return f.incidents.Put(ctx, *inc)
}

// ResumeApplying reattaches durable work after process restart. Accepting
// unreadable-file exceptions is idempotent and safe to replay; volume repair
// resumes only from a durable journal with no ambiguous in-flight step. An
// ambiguous journal stays applying and is surfaced for human inspection.
func (f *BackupFailureFlow) ResumeApplying(ctx context.Context, inc repohealth.Incident) error {
	if inc.State != repohealth.IncidentApplying {
		return nil
	}
	if !f.ready() {
		return errors.New("backup incident recovery is not configured")
	}
	task, err := f.asks.Get(ctx, inc.AskID)
	if err != nil {
		return err
	}
	res := operatordecision.DecisionResolution{
		AskID: inc.AskID, State: operatordecision.StateAnswered, Answer: inc.Answer,
		AnsweredBy: inc.AnsweredBy, AnsweredByIdentityKind: inc.AnsweredByIdentityKind,
		AnsweredByIdentitySource: inc.AnsweredByIdentitySource, AnsweredAt: inc.AnsweredAt.UTC().Format(time.RFC3339Nano),
		Correlation: map[string]string{"incident_id": inc.ID, "generation": strconv.Itoa(inc.Generation)},
	}
	if !isVerifiedRepairApproval(res) {
		return f.requireApplyingReview(ctx, task, inc, "The applying incident lacks complete verified approval evidence. Inspect the incident before continuing.")
	}
	switch inc.Answer {
	case repohealth.OptAccept:
		if task == nil || (task.State != operatordecision.PendingStatePending && task.State != operatordecision.PendingStateApproved) {
			return f.requireApplyingReview(ctx, task, inc, "The accepted-file decision record is unavailable. Inspect the incident before continuing.")
		}
		return f.resumeAccept(ctx, task, inc, res)
	case repohealth.OptKopiaQuarantine:
		if task == nil || (task.State != operatordecision.PendingStatePending && task.State != operatordecision.PendingStateApproved) {
			return f.requireApplyingReview(ctx, task, inc, "The Kopia quarantine approval record is unavailable. Inspect the incident before continuing.")
		}
		return f.resumeKopiaQuarantine(ctx, task, inc, res)
	case repohealth.OptRepair:
		if task == nil || (task.State != operatordecision.PendingStatePending && task.State != operatordecision.PendingStateApproved) {
			return f.requireApplyingReview(ctx, task, inc, "The repair decision record is unavailable. Inspect the incident before continuing.")
		}
		dest, getErr := f.dests.GetDestination(ctx, inc.DestinationID)
		if getErr != nil || dest.DeviceIdentity == nil || !dest.DeviceIdentity.StrongIdentity() {
			return f.requireApplyingReview(ctx, task, inc, "The approved repair target no longer has a strong device identity. Inspect it before continuing.")
		}
		if f.history != nil {
			active, activeErr := f.history.ActiveRun(ctx, inc.DestinationID)
			if activeErr != nil || active {
				return nil
			}
		}
		return f.resumeRepair(ctx, task, inc, res, dest)
	case repohealth.OptFresh:
		if task == nil || (task.State != operatordecision.PendingStatePending && task.State != operatordecision.PendingStateApproved) {
			return f.requireApplyingReview(ctx, task, inc, "The fresh-repository migration approval record is unavailable. Inspect the incident before continuing.")
		}
		return f.applyFresh(ctx, task, inc, res)
	default:
		return f.requireApplyingReview(ctx, task, inc, "The applying incident has an unsupported answer. Inspect it before continuing.")
	}
}

func (f *BackupFailureFlow) requireApplyingReview(ctx context.Context, ask *operatordecision.PendingAsk, inc repohealth.Incident, detail string) error {
	inc.Result, inc.UpdatedAt = detail, f.now().UTC()
	if err := f.incidents.Put(ctx, inc); err != nil {
		return fmt.Errorf("record applying recovery hold: %w", err)
	}
	if ask == nil || ask.State != operatordecision.PendingStatePending {
		return nil
	}
	return f.finish(ctx, ask, operatordecision.PendingStateApproved, inc.AnsweredBy, noticeText{
		title: "Approved backup recovery needs inspection", body: detail,
		category: categoryBackupNeedsAttention, urgency: "high",
	})
}

// retryPlansAfter asks for a run of every enabled plan that writes to the
// incident's destination, unless a run already started after the persisted
// approval time. That read closes the crash window after TriggerRun succeeds
// but before the incident records Applied.
func (f *BackupFailureFlow) retryPlansAfter(ctx context.Context, inc repohealth.Incident, approvedAt time.Time) int {
	if f.history == nil {
		return 0
	}
	plans, err := f.history.Plans(ctx)
	if err != nil {
		f.logger.Printf("backup retry: list plans: %v", err)
		return 0
	}
	wanted := map[string]bool{}
	for _, id := range inc.PlanIDs {
		wanted[id] = true
	}
	started := 0
	for _, p := range plans {
		if !p.Enabled || !wanted[p.ID] {
			continue
		}
		latest, exists, latestErr := f.history.LatestRun(ctx, p.ID)
		if latestErr != nil {
			f.logger.Printf("backup retry: verify latest run for plan %s: %v", p.ID, latestErr)
			continue // do not risk a duplicate side effect when idempotency is unknown.
		}
		if exists && !approvedAt.IsZero() && !latest.StartedAt.IsZero() && !latest.StartedAt.Before(approvedAt) {
			started++ // prior attempt already scheduled this recovery's run.
			continue
		}
		if err := f.history.TriggerRun(ctx, p.ID); err != nil {
			f.logger.Printf("backup retry: trigger plan %s: %v", p.ID, err)
			continue
		}
		started++
	}
	return started
}

// applyRepair delegates to the existing volume repair path: unmount, offline
// filesystem repair and read/write remount through the control-plane owner,
// bound to the drive's identity and journalled so a crash cannot repeat a repair.
func (f *BackupFailureFlow) applyRepair(ctx context.Context, ask *operatordecision.PendingAsk, inc repohealth.Incident, res operatordecision.DecisionResolution) error {
	// Defense in depth for a decision raised before the offer was tightened: a
	// tool that cannot repair metadata must never run and be reported as a
	// repair. A read-only mount is the one cause a dirty-flag reset fixes.
	if inc.Diagnosis.RepairLimitation != "" && inc.Diagnosis.Cause != repohealth.CauseVolumeReadOnly {
		return f.failApplied(ctx, ask, inc, res, "The only repair this host offers for this filesystem cannot fix this condition, so I did not run it. Nothing was changed.")
	}
	dest, err := f.dests.GetDestination(ctx, inc.DestinationID)
	if err != nil || dest.DeviceIdentity == nil || !dest.DeviceIdentity.StrongIdentity() {
		return f.failApplied(ctx, ask, inc, res, "I cannot repair this drive: its identity is not recorded strongly enough to act on safely. Nothing was changed.")
	}
	if f.history != nil {
		active, activeErr := f.history.ActiveRun(ctx, inc.DestinationID)
		if activeErr != nil || active {
			return f.failApplied(ctx, ask, inc, res, "A backup is running to this drive, so I did not take it offline. Nothing was changed; I will ask again.")
		}
	}
	if inc.State != repohealth.IncidentApplying {
		root := filepath.Clean(dest.DeviceIdentity.Mountpoint)
		steps, err := f.planRepairSteps(ctx, root, *dest.DeviceIdentity)
		if err != nil {
			return f.failApplied(ctx, ask, inc, res, "The control plane could not prepare the repair, so nothing was changed: "+err.Error())
		}
		journal, err := destinationreadiness.NewRecoveryJournal(root, *dest.DeviceIdentity, steps, f.now().UTC())
		if err != nil {
			return f.failApplied(ctx, ask, inc, res, "I could not prepare the repair journal, so nothing was changed: "+err.Error())
		}
		journal.ID = operatordecision.CorrelationID("dbm-backup-repair", inc.ID, strconv.Itoa(inc.Generation))
		if err := f.recoveries.Save(ctx, journal); err != nil {
			return f.failApplied(ctx, ask, inc, res, "I could not persist the repair journal, so nothing was changed: "+err.Error())
		}
		inc.RepairJournalID = journal.ID
		if err := f.beginApplying(ctx, &inc, res); err != nil {
			return fmt.Errorf("record repair in progress: %w", err)
		}
	} else if !sameIncidentApproval(inc, res) {
		return errors.New("persisted applying incident does not match its verified approval")
	}
	return f.resumeRepair(ctx, ask, inc, res, dest)
}

func (f *BackupFailureFlow) resumeRepair(ctx context.Context, ask *operatordecision.PendingAsk, inc repohealth.Incident, res operatordecision.DecisionResolution, dest destinations.Destination) error {
	if strings.TrimSpace(inc.RepairJournalID) == "" {
		return f.requireApplyingReview(ctx, ask, inc, "The approved repair has no durable recovery journal. Inspect the drive before starting another repair.")
	}
	journal, err := f.recoveries.Load(ctx, inc.RepairJournalID)
	if err != nil {
		return f.requireApplyingReview(ctx, ask, inc, "The approved repair journal could not be recovered ("+err.Error()+"). Inspect the drive before continuing.")
	}
	if dest.DeviceIdentity == nil || !dest.DeviceIdentity.StrongIdentity() || !journal.Identity.MatchesDevice(*dest.DeviceIdentity) {
		return f.requireApplyingReview(ctx, ask, inc, "The approved repair journal no longer matches the destination's strong device identity. Inspect the drive before continuing.")
	}
	if journal.State == destinationreadiness.RecoveryPaused || journal.State == destinationreadiness.RecoveryFailed {
		return f.requireApplyingReview(ctx, ask, inc, "The approved repair paused at an uncertain step. Inspect the drive and recovery journal before resuming.")
	}
	for _, step := range journal.Steps {
		if step.Status == destinationreadiness.RecoveryStepInFlight || step.Status == destinationreadiness.RecoveryStepBlocked {
			return f.requireApplyingReview(ctx, ask, inc, "The approved repair stopped at an uncertain step. Inspect the drive and recovery journal before resuming.")
		}
	}
	confirmations := make(map[int]string, len(journal.Steps))
	for i, step := range journal.Steps {
		if step.Plan.RequiresConfirm {
			confirmations[i] = step.Plan.ConfirmationPhrase
		}
	}
	updated, resumeErr := f.readiness.ResumeRecovery(ctx, f.recoveries, journal, confirmations, true, false)
	if resumeErr != nil || updated.State != destinationreadiness.RecoveryCompleted {
		detail := updated.LastError
		if resumeErr != nil {
			detail = resumeErr.Error()
		}
		if detail == "" {
			detail = "the repair did not reach a verified finished state"
		}
		return f.failApplied(ctx, ask, inc, res, "The drive repair did not finish: "+detail+". Check the drive before backing up to it.")
	}
	if f.volume != nil {
		if _, verifyErr := f.volume.verifyMountedRecoveryLocation(ctx, updated.Location, updated.Identity); verifyErr != nil {
			return f.failApplied(ctx, ask, inc, res, "The repair finished but I could not confirm the drive is mounted and writable: "+verifyErr.Error())
		}
	}
	if f.usage != nil {
		f.usage.InvalidateScan(inc.Diagnosis.RepositoryRoot)
	}
	started := f.retryPlansAfter(ctx, inc, inc.AnsweredAt)
	if err := inc.Transition(repohealth.IncidentApplied); err != nil {
		return err
	}
	inc.UpdatedAt = f.now().UTC()
	inc.Result = "drive repaired and remounted"
	if err := f.incidents.Put(ctx, inc); err != nil {
		return fmt.Errorf("record applied repair: %w", err)
	}
	body := "The drive was taken offline, repaired, and mounted again, and I confirmed it is writable."
	if started > 0 {
		body += " A backup was started now; if the same files are still unreadable it will stop and I will ask again."
	}
	return f.finish(ctx, ask, operatordecision.PendingStateApproved, res.AnsweredBy, noticeText{
		title: repohealth.Capitalize(inc.Diagnosis.DriveSentence()) + " was repaired", body: body,
		category: categoryBackupRecovered, urgency: "normal",
	})
}

// planRepairSteps plans unmount, repair and read/write remount for the drive.
// Every step must be supported by the control plane or none runs.
func (f *BackupFailureFlow) planRepairSteps(ctx context.Context, root string, identity destinationreadiness.DeviceIdentity) ([]destinationreadiness.Plan, error) {
	var steps []destinationreadiness.Plan
	for _, action := range []destinationreadiness.PreparationAction{
		destinationreadiness.ActionUnmount, destinationreadiness.ActionRepairFilesystem, destinationreadiness.ActionMountReadWrite,
	} {
		plan, err := f.readiness.PlanPreparation(ctx, destinationreadiness.PlanInput{Location: root, Action: action, ExpectedDevice: identity})
		if err != nil {
			return nil, err
		}
		if !plan.Supported {
			return nil, fmt.Errorf("%s is not supported on this host: %s", action, plan.UnsupportedReason)
		}
		steps = append(steps, plan)
	}
	return steps, nil
}

// applyFresh persists the verified choice as a durable migration handoff. The
// decision does not carry a destination, capacity proof, or credential source,
// so provisioning a repository here would require guessing operator-owned
// inputs. Keep the old repository intact and make that exact hold visible.
func (f *BackupFailureFlow) applyFresh(ctx context.Context, ask *operatordecision.PendingAsk, inc repohealth.Incident, res operatordecision.DecisionResolution) error {
	if !isVerifiedRepairApproval(res) || res.Answer != repohealth.OptFresh {
		return errors.New("fresh-repository migration requires its verified human decision")
	}
	if !inc.Offers.Fresh {
		return errors.New("fresh-repository migration was not offered for this incident")
	}
	if ask == nil {
		return errors.New("fresh-repository migration callback has no durable decision record")
	}
	migrationID := repohealth.FreshMigrationID(inc.ID, inc.Generation)
	if inc.FreshMigrationID != "" && inc.FreshMigrationID != migrationID {
		return errors.New("persisted fresh-repository migration journal identity does not match this incident generation")
	}
	if inc.State == repohealth.IncidentApplying {
		if !sameIncidentApproval(inc, res) {
			return errors.New("persisted migration approval does not match callback")
		}
	} else if inc.State == repohealth.IncidentAgentRequested {
		// The incident journal is committed before the ask/result notice. If the
		// process stopped in that window, replay only the notice; never restart
		// destination selection or any repository operation.
		if !sameIncidentApproval(inc, res) || !freshMigrationStageRecorded(inc.FreshMigrationStage) || inc.FreshMigrationID != migrationID {
			return errors.New("persisted migration handoff does not match callback")
		}
	} else if err := f.beginApplying(ctx, &inc, res); err != nil {
		return fmt.Errorf("record fresh-repository migration approval: %w", err)
	}
	// This decision class approves the risk of a fresh set; it does not select
	// where to create it. In particular, never assume the damaged Elements
	// destination is writable or has enough capacity, and never redirect plans
	// before fresh captures and restore drills have been proven.
	inc.FreshMigrationStage = "awaiting_destination_selection"
	inc.FreshMigrationID = migrationID
	inc.FreshMigrationDetail = "No destination location, capacity proof, or credential authority was supplied by the decision; old repository and plan bindings are preserved."
	inc.Result = inc.FreshMigrationDetail
	inc.UpdatedAt = f.now().UTC()
	if err := inc.Transition(repohealth.IncidentAgentRequested); err != nil {
		return err
	}
	if err := f.incidents.Put(ctx, inc); err != nil {
		return fmt.Errorf("record fresh-repository migration handoff: %w", err)
	}
	selectionReady, selectionErr := f.askFreshDestination(ctx, inc)
	if selectionErr != nil {
		f.logger.Printf("fresh repository destination decision could not be raised incident=%s: %v", inc.ID, selectionErr)
	}
	if selectionReady {
		return f.finish(ctx, ask, operatordecision.PendingStateApproved, res.AnsweredBy, noticeText{
			title:    "Fresh repository destination needs your selection",
			body:     "Your verified choice to start a fresh backup set was recorded. Choose one currently registered destination in the follow-up decision. DBM will prove capacity and credentials again, then capture and scratch-drill all 14 targets before changing plan bindings. The old repository remains unchanged.",
			category: categoryBackupNeedsAttention, urgency: "high",
		})
	}
	return f.finish(ctx, ask, operatordecision.PendingStateApproved, res.AnsweredBy, noticeText{
		title:    "Fresh repository migration is waiting for destination details",
		body:     "Your verified choice to migrate from the damaged repository on " + inc.Diagnosis.DriveSentence() + " was recorded. No repository or plan was changed. The migration is waiting for an operator-selected destination, a capacity check, and credential authority; after those are supplied, capture and drill every target before changing plan bindings. The old repository remains preserved.",
		category: categoryBackupNeedsAttention, urgency: "high",
	})
}

func freshMigrationStageRecorded(stage string) bool {
	switch stage {
	case "awaiting_destination_selection", "selection_deferred", "selection_recorded", "migrating", "complete":
		return true
	default:
		return false
	}
}

type noticeText struct{ title, body, category, urgency string }

// finish records the terminal ask state with its result notice, then publishes
// the notice; a publication failure leaves the notice pending for the next
// delivery or sweep.
func (f *BackupFailureFlow) finish(ctx context.Context, ask *operatordecision.PendingAsk, state, by string, n noticeText) error {
	if ask.Extra == nil {
		ask.Extra = map[string]string{}
	}
	now := f.now().UTC()
	ask.State, ask.ResolvedAt, ask.ResolvedBy = state, &now, by
	ask.Extra["result_title"], ask.Extra["result_body"] = n.title, n.body
	ask.Extra["result_category"], ask.Extra["result_urgency"] = n.category, n.urgency
	ask.Extra["result_notification_pending"] = "true"
	if err := f.asks.Put(ctx, ask); err != nil {
		return fmt.Errorf("persist backup decision result: %w", err)
	}
	return f.publishPendingResult(ctx, ask)
}

func (f *BackupFailureFlow) publishPendingResult(ctx context.Context, ask *operatordecision.PendingAsk) error {
	if ask == nil || ask.Extra["result_notification_pending"] != "true" {
		return nil
	}
	event := backupNotification(ask.Extra["result_title"], ask.Extra["result_body"], ask.Extra["result_category"], ask.Extra["result_urgency"],
		"backup-result:"+ask.ID+":"+ask.Extra["result_category"], f.now())
	if ask.ResolvedAt != nil {
		event.Occurred = ask.ResolvedAt.UTC()
	}
	if err := f.publisher.PublishDomainEvent(ctx, event); err != nil {
		return fmt.Errorf("publish backup decision result: %w", err)
	}
	ask.Extra["result_notification_pending"] = "false"
	return f.asks.Put(ctx, ask)
}

// backupNotification builds a plain operator notice. It carries no internal
// identifiers; the dedupe key is the only technical field.
func backupNotification(title, body, category, urgency, dedupeKey string, now time.Time) eventbus.DomainEvent {
	return eventbus.DomainEvent{
		Source:    recoveryDecisionSource,
		EventType: backupNotificationType,
		Occurred:  now.UTC(),
		Payload: map[string]any{
			"title": title, "body": body, "category": category, "urgency": urgency,
			"sensitivity": "private", "dedupe_key": dedupeKey,
		},
	}
}

var _ operatordecision.DomainEffect = (*BackupFailureFlow)(nil)
