package destinations

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"data-backup-manager/internal/destinationreadiness"
	"data-backup-manager/internal/destinations"
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
	readiness  *destinationreadiness.Service
	volume     *RecoveryDecisionFlow
	recoveries destinationreadiness.RecoveryStore
	asks       operatordecision.PendingStore
	publisher  operatordecision.Publisher
	raiser     operatordecision.Raiser
	incidents  repohealth.IncidentStore
	exceptions repohealth.ExceptionStore
	scanner    *repohealth.Scanner
	usage      interface{ InvalidateScan(string) }
	history    BackupHistory
	dests      destinations.Service
	now        func() time.Time
	logger     *log.Logger
	enabled    bool
}

// BackupFailureFlowDeps wires the flow. Every field is a seam.
type BackupFailureFlowDeps struct {
	Readiness  *destinationreadiness.Service
	Volume     *RecoveryDecisionFlow
	Recoveries destinationreadiness.RecoveryStore
	Asks       operatordecision.PendingStore
	Publisher  operatordecision.Publisher
	Incidents  repohealth.IncidentStore
	Exceptions repohealth.ExceptionStore
	Scanner    *repohealth.Scanner
	Usage      interface{ InvalidateScan(string) }
	History    BackupHistory
	Dests      destinations.Service
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
		history: d.History, dests: d.Dests, now: now, logger: logger, enabled: d.Enabled,
	}
	f.raiser = operatordecision.Raiser{
		Source: recoveryDecisionSource, Kind: backupFailureKind, Store: d.Asks, Publisher: d.Publisher,
		Clock: backupFlowClock{f}, MaxOutstanding: 3, Window: backupDecisionWindow,
	}
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

// Raise publishes the incident's question, idempotently: while the same
// generation is outstanding this is a no-op, and a publication that failed is
// retried here until it succeeds. The facts come from the diagnosis stored with
// the incident, so a retry never changes the question under the same identity.
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
	if _, err := f.raiser.Raise(ctx, ask, facts); err != nil {
		return fmt.Errorf("raise backup failure decision: %w", err)
	}
	return nil
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
// anything.
func (f *BackupFailureFlow) Apply(ctx context.Context, res operatordecision.DecisionResolution) (bool, error) {
	if !f.ready() {
		return false, nil
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
		return true, f.finish(ctx, ask, operatordecision.PendingStateRejected, "", noticeText{
			title: "Backup question is out of date", body: "This question no longer matches what is on record, so nothing was changed. If backups are still blocked I will ask again.",
			category: categoryBackupNeedsAttention, urgency: "high",
		})
	}
	return true, f.route(ctx, ask, inc, res)
}

// lapse closes a question the hub expired unanswered. Nothing was answered and
// nothing is changed; the monitor asks again at once, so a question is always
// outstanding while backups are blocked. No result notice is sent: the daily
// stalled-backup notice and the fresh question already say so.
func (f *BackupFailureFlow) lapse(ctx context.Context, ask *operatordecision.PendingAsk, inc repohealth.Incident) error {
	inc.State, inc.UpdatedAt, inc.Result = repohealth.IncidentLapsed, f.now().UTC(), "the question expired unanswered"
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
	inc.State, inc.DeclinedAt, inc.Answer, inc.UpdatedAt = repohealth.IncidentDeclined, f.now().UTC(), res.Answer, f.now().UTC()
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
	inc.State, inc.UpdatedAt = repohealth.IncidentApplying, f.now().UTC()
	if err := f.incidents.Put(ctx, inc); err != nil {
		return fmt.Errorf("record accept in progress: %w", err)
	}
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
	if len(recorded) == 0 && len(changed) > 0 {
		return f.failApplied(ctx, ask, inc, res, "The unreadable files changed since I asked, so I recorded nothing. I will ask again.")
	}
	started := f.retryPlans(ctx, inc)
	inc.State, inc.Answer, inc.AnsweredBy, inc.AnsweredAt, inc.UpdatedAt = repohealth.IncidentApplied, res.Answer, res.AnsweredBy, f.now().UTC(), f.now().UTC()
	inc.Result = fmt.Sprintf("recorded %d exception(s)", len(recorded))
	if err := f.incidents.Put(ctx, inc); err != nil {
		return fmt.Errorf("record applied accept: %w", err)
	}
	body := fmt.Sprintf("I recorded your approval to skip %d unreadable leftover %s. Nothing on the drive was touched.", len(recorded), plural(len(recorded), "file", "files"))
	if len(changed) > 0 {
		body += fmt.Sprintf(" %d other %s changed since I asked and %s not covered; if that blocks backups I will ask again.", len(changed), plural(len(changed), "file", "files"), plural(len(changed), "is", "are"))
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

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func (f *BackupFailureFlow) failApplied(ctx context.Context, ask *operatordecision.PendingAsk, inc repohealth.Incident, res operatordecision.DecisionResolution, detail string) error {
	inc.State, inc.Answer, inc.AnsweredBy, inc.AnsweredAt, inc.UpdatedAt = repohealth.IncidentFailed, res.Answer, res.AnsweredBy, f.now().UTC(), f.now().UTC()
	inc.Result = detail
	if err := f.incidents.Put(ctx, inc); err != nil {
		return fmt.Errorf("record failed backup action: %w", err)
	}
	return f.finish(ctx, ask, operatordecision.PendingStateRejected, res.AnsweredBy, noticeText{
		title: "Backup recovery needs attention", body: detail,
		category: categoryBackupNeedsAttention, urgency: "high",
	})
}

// retryPlans asks for a run of every enabled plan that writes to the incident's
// destination, returning how many were started.
func (f *BackupFailureFlow) retryPlans(ctx context.Context, inc repohealth.Incident) int {
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
	inc.RepairJournalID, inc.State, inc.UpdatedAt = journal.ID, repohealth.IncidentApplying, f.now().UTC()
	if err := f.incidents.Put(ctx, inc); err != nil {
		return fmt.Errorf("record repair in progress: %w", err)
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
	started := f.retryPlans(ctx, inc)
	inc.State, inc.Answer, inc.AnsweredBy, inc.AnsweredAt, inc.UpdatedAt = repohealth.IncidentApplied, res.Answer, res.AnsweredBy, f.now().UTC(), f.now().UTC()
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

// applyFresh makes no change: registering a new destination and re-pointing the
// plan is work for an agent. The answer records the request and tells the
// operator exactly what to ask for.
func (f *BackupFailureFlow) applyFresh(ctx context.Context, ask *operatordecision.PendingAsk, inc repohealth.Incident, res operatordecision.DecisionResolution) error {
	inc.State, inc.Answer, inc.AnsweredBy, inc.AnsweredAt, inc.UpdatedAt = repohealth.IncidentAgentRequested, res.Answer, res.AnsweredBy, f.now().UTC(), f.now().UTC()
	inc.Result = "agent setup requested"
	if err := f.incidents.Put(ctx, inc); err != nil {
		return fmt.Errorf("record agent request: %w", err)
	}
	return f.finish(ctx, ask, operatordecision.PendingStateApproved, res.AnsweredBy, noticeText{
		title:    "A fresh backup set needs an agent",
		body:     "You chose a fresh backup set on " + inc.Diagnosis.DriveSentence() + ". I cannot create one myself. Ask an agent: \"In Data Backup Manager, register a new backup destination beside the existing one on this drive, point the plan at it, and keep the old destination for restores only.\" Nothing was changed, so backups stay stopped and I will keep reminding you daily until that is done.",
		category: categoryBackupNeedsAttention, urgency: "high",
	})
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
