package destinations

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"data-backup-manager/internal/destinationreadiness"
	"data-backup-manager/internal/destinations"
	"data-backup-manager/internal/engine"
	"data-backup-manager/internal/repohealth"

	"github.com/vrooli/api-core/lifecycle"
	"github.com/vrooli/api-core/operatordecision"
)

const (
	backupMonitorInterval = time.Minute
	// Operator notices are evaluated every quarter hour: they repeat at most
	// daily, and the evaluation reads per-plan history.
	backupNoticeCheckEvery = 15 * time.Minute
	// A destination is diagnosed at most this often, because a diagnosis walks the
	// whole repository directory.
	backupMinDiagnosisGap = 30 * time.Minute
	// backupNoticeStartupGrace delays operator notices after a monitor start.
	backupNoticeStartupGrace = 10 * time.Minute
	repoEvidenceTTL          = 6 * time.Hour
	repoEvidenceFailedTTL    = 30 * time.Minute
)

// BackupFailureWorker watches enabled plans. When a run is stopped by a
// destination-level reason it diagnoses read-only, applies what is safe, and
// raises ONE question through BackupFailureFlow, which it keeps in front of the
// operator until answered. It also tells the operator when backups have not
// succeeded for too long and when a switched-off plan leaves items unprotected.
type BackupFailureWorker struct {
	history    BackupHistory
	dests      destinations.Service
	readiness  *destinationreadiness.Service
	flow       *BackupFailureFlow
	volume     *VolumeRecoveryWorker
	usage      *repohealth.UsageEngine
	exceptions repohealth.ExceptionStore
	incidents  repohealth.IncidentStore
	notices    *repohealth.NoticeStateStore
	publisher  operatordecision.Publisher
	recoveries *destinationreadiness.FileRecoveryStore
	now        func() time.Time
	logger     *log.Logger
	interval   time.Duration

	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}

	lastDiagnosed   map[string]time.Time
	lastNoticeCheck time.Time
	// noticesNotBefore holds operator notices back for a short grace after the
	// monitor starts, so a restart (and any plan change made around it) settles
	// before anything is reported.
	noticesNotBefore time.Time
	evidence         map[string]repoEvidence
}

type repoEvidence struct {
	at       time.Time
	ok       bool
	latestBy map[string]time.Time
}

// BackupFailureWorkerDeps wires the worker. Every field is a seam.
type BackupFailureWorkerDeps struct {
	History    BackupHistory
	Dests      destinations.Service
	Readiness  *destinationreadiness.Service
	Flow       *BackupFailureFlow
	Volume     *VolumeRecoveryWorker
	Usage      *repohealth.UsageEngine
	Exceptions repohealth.ExceptionStore
	Incidents  repohealth.IncidentStore
	Notices    *repohealth.NoticeStateStore
	Publisher  operatordecision.Publisher
	Recoveries *destinationreadiness.FileRecoveryStore
	Now        func() time.Time
	Logger     *log.Logger
}

func newBackupFailureWorker(d BackupFailureWorkerDeps) *BackupFailureWorker {
	logger, now := d.Logger, d.Now
	if logger == nil {
		logger = log.Default()
	}
	if now == nil {
		now = time.Now
	}
	return &BackupFailureWorker{
		history: d.History, dests: d.Dests, readiness: d.Readiness, flow: d.Flow, volume: d.Volume,
		usage: d.Usage, exceptions: d.Exceptions, incidents: d.Incidents, notices: d.Notices,
		publisher: d.Publisher, recoveries: d.Recoveries, now: now, logger: logger, interval: backupMonitorInterval,
		lastDiagnosed: map[string]time.Time{}, evidence: map[string]repoEvidence{},
	}
}

// LifecycleHooks pauses the monitor around maintenance, like the volume monitor.
func (w *BackupFailureWorker) LifecycleHooks() lifecycle.Hooks {
	return lifecycle.Hooks{Quiesce: w.Stop, Resume: func(context.Context) error { return w.Start() }}
}

func (w *BackupFailureWorker) Start() error {
	if w == nil {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.cancel != nil {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	w.cancel, w.done = cancel, done
	w.noticesNotBefore = w.now().Add(backupNoticeStartupGrace)
	go func() {
		defer close(done)
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		w.reconcile(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.reconcile(ctx)
			}
		}
	}()
	return nil
}

func (w *BackupFailureWorker) Stop(ctx context.Context) error {
	if w == nil {
		return nil
	}
	w.mu.Lock()
	cancel, done := w.cancel, w.done
	w.cancel, w.done = nil, nil
	w.mu.Unlock()
	if cancel == nil {
		return nil
	}
	cancel()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *BackupFailureWorker) reconcile(ctx context.Context) {
	if w == nil || w.history == nil || w.dests == nil {
		return
	}
	plans, err := w.history.Plans(ctx)
	if err != nil {
		w.logger.Printf("backup failure scan could not list plans: %v", err)
		return
	}
	list, err := w.dests.ListDestinations(ctx, 100)
	if err != nil {
		w.logger.Printf("backup failure scan could not list destinations: %v", err)
		return
	}
	dests := make(map[string]destinations.Destination, len(list))
	for _, d := range list {
		dests[d.ID] = d
	}
	w.reconcileFailures(ctx, plans, dests)
	w.upkeepIncidents(ctx, plans)
	if !w.now().Before(w.noticesNotBefore) && w.now().Sub(w.lastNoticeCheck) >= backupNoticeCheckEvery {
		w.lastNoticeCheck = w.now()
		w.reconcileNotices(ctx, plans, dests)
	}
}

type failingRun struct {
	plan PlanRecord
	run  RunRecord
}

// reconcileFailures diagnoses each destination that stopped an enabled plan's
// latest run, once per new failed run.
func (w *BackupFailureWorker) reconcileFailures(ctx context.Context, plans []PlanRecord, dests map[string]destinations.Destination) {
	failing := map[string][]failingRun{}
	for _, p := range plans {
		if !p.Enabled {
			continue
		}
		run, ok, err := w.history.LatestRun(ctx, p.ID)
		if err != nil || !ok || !run.Terminal {
			continue
		}
		for _, destID := range p.DestinationIDs {
			if run.FailsDestination(destID) {
				failing[destID] = append(failing[destID], failingRun{plan: p, run: run})
			}
		}
	}
	ids := make([]string, 0, len(failing))
	for id := range failing {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, destID := range ids {
		dest, ok := dests[destID]
		if !ok || dest.BackendKind != destinations.BackendFilesystem {
			continue
		}
		if err := w.handleFailing(ctx, dest, failing[destID], plans); err != nil {
			w.logger.Printf("backup failure handling for destination %s: %v", destID, err)
		}
	}
}

func (w *BackupFailureWorker) handleFailing(ctx context.Context, dest destinations.Destination, items []failingRun, plans []PlanRecord) error {
	latest := items[0].run
	planIDs := make([]string, 0, len(items))
	for _, it := range items {
		planIDs = append(planIDs, it.plan.ID)
		if it.run.FinishedAt.After(latest.FinishedAt) {
			latest = it.run
		}
	}
	open, err := w.openIncident(ctx, dest.ID)
	if err != nil {
		return err
	}
	if open != nil && open.DiagnosedRunID == latest.ID {
		return nil
	}
	if open != nil && w.now().Sub(w.lastDiagnosed[dest.ID]) < backupMinDiagnosisGap {
		return nil
	}
	w.lastDiagnosed[dest.ID] = w.now()
	diag := w.diagnose(ctx, dest, plans)
	return w.act(ctx, dest, diag, open, latest.ID, planIDs)
}

func (w *BackupFailureWorker) openIncident(ctx context.Context, destinationID string) (*repohealth.Incident, error) {
	all, err := w.incidents.List(ctx)
	if err != nil {
		return nil, err
	}
	var found *repohealth.Incident
	for i := range all {
		if all[i].DestinationID == destinationID && all[i].State.Open() {
			found = &all[i]
		}
	}
	return found, nil
}

// act applies the safe steps for a diagnosis and, when nothing safe resolves it,
// opens or refreshes the one incident (and its question) for the condition.
func (w *BackupFailureWorker) act(ctx context.Context, dest destinations.Destination, diag repohealth.Diagnosis, open *repohealth.Incident, runID string, planIDs []string) error {
	switch diag.Cause {
	case repohealth.CauseNone:
		// Every read-only check passes now: the failure was transient. The safe
		// step is a retry.
		if open != nil {
			w.resolve(ctx, *open, "cleared on its own")
		}
		w.retry(ctx, planIDs)
		return nil
	case repohealth.CauseVolumeUnmounted:
		// The existing drive recovery owns an unmounted volume: its clean-check
		// remount, or its own repair question. No second question here.
		if w.volume != nil {
			if err := w.volume.reconcileDestination(ctx, dest); err != nil {
				return fmt.Errorf("drive recovery for unmounted destination: %w", err)
			}
		}
		return nil
	}
	offers := repohealth.OffersFor(diag, w.repairPlannable(ctx, dest))
	if !offers.Askable() {
		return fmt.Errorf("diagnosed %s but no action can be offered", diag.Cause)
	}
	inc, err := w.upsertIncident(ctx, dest, diag, offers, open, runID, planIDs)
	if err != nil || inc == nil {
		return err
	}
	if inc.State == repohealth.IncidentAwaiting {
		return w.flow.Raise(ctx, *inc)
	}
	return nil
}

// upsertIncident returns the incident for this exact condition, opening a new
// generation when the earlier answer did not hold, and superseding an open
// incident for a different condition. A nil result means nothing to raise.
func (w *BackupFailureWorker) upsertIncident(ctx context.Context, dest destinations.Destination, diag repohealth.Diagnosis, offers repohealth.Offers, open *repohealth.Incident, runID string, planIDs []string) (*repohealth.Incident, error) {
	id := repohealth.IncidentID(dest.ID, diag.Signature())
	now := w.now().UTC()
	if open != nil && open.ID != id {
		open.State, open.UpdatedAt = repohealth.IncidentSuperseded, now
		if err := w.incidents.Put(ctx, *open); err != nil {
			return nil, err
		}
	}
	existing, found, err := w.incidents.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	switch {
	case !found:
		inc := w.newIncident(id, 1, dest, diag, offers, runID, planIDs)
		return &inc, w.incidents.Put(ctx, inc)
	case !existing.State.Open():
		inc := w.newIncident(id, existing.Generation+1, dest, diag, offers, runID, planIDs)
		return &inc, w.incidents.Put(ctx, inc)
	}
	existing.DiagnosedRunID, existing.UpdatedAt = runID, now
	switch existing.State {
	case repohealth.IncidentApplied:
		// The answer was applied and the next run hit the same condition: it did
		// not hold. Ask again as a new generation.
		existing = w.reopen(existing, diag, offers, planIDs)
	case repohealth.IncidentDeclined, repohealth.IncidentFailed:
		if now.Sub(lastTouch(existing)) >= backupReaskAfter {
			existing = w.reopen(existing, diag, offers, planIDs)
		}
	}
	return &existing, w.incidents.Put(ctx, existing)
}

func lastTouch(inc repohealth.Incident) time.Time {
	if !inc.DeclinedAt.IsZero() {
		return inc.DeclinedAt
	}
	if !inc.AnsweredAt.IsZero() {
		return inc.AnsweredAt
	}
	return inc.UpdatedAt
}

func (w *BackupFailureWorker) newIncident(id string, generation int, dest destinations.Destination, diag repohealth.Diagnosis, offers repohealth.Offers, runID string, planIDs []string) repohealth.Incident {
	now := w.now().UTC()
	facts := repohealth.BuildFacts(diag, offers, id, generation)
	return repohealth.Incident{
		ID: id, DestinationID: dest.ID, Signature: diag.Signature(), Generation: generation,
		State: repohealth.IncidentAwaiting, PlanIDs: planIDs, Subject: facts.Subject,
		Diagnosis: diag, Offers: offers, AskID: backupAskID(id, generation), DiagnosedRunID: runID,
		CreatedAt: now, UpdatedAt: now, AskedAt: now,
	}
}

func (w *BackupFailureWorker) reopen(inc repohealth.Incident, diag repohealth.Diagnosis, offers repohealth.Offers, planIDs []string) repohealth.Incident {
	now := w.now().UTC()
	inc.Generation++
	inc.State, inc.Diagnosis, inc.Offers, inc.PlanIDs = repohealth.IncidentAwaiting, diag, offers, planIDs
	inc.Subject = repohealth.BuildFacts(diag, offers, inc.ID, inc.Generation).Subject
	inc.AskID = backupAskID(inc.ID, inc.Generation)
	inc.DeclinedAt, inc.AnsweredAt, inc.AnsweredBy, inc.Answer, inc.Result, inc.UpdatedAt = time.Time{}, time.Time{}, "", "", "", now
	inc.AskedAt = now
	return inc
}

// upkeepIncidents closes incidents whose destination works again, keeps each
// unanswered question in front of the operator, and asks again after a decline.
func (w *BackupFailureWorker) upkeepIncidents(ctx context.Context, plans []PlanRecord) {
	all, err := w.incidents.List(ctx)
	if err != nil {
		w.logger.Printf("backup failure upkeep could not list incidents: %v", err)
		return
	}
	for _, inc := range all {
		if !inc.State.Open() {
			continue
		}
		if w.recovered(ctx, inc, plans) {
			w.resolve(ctx, inc, "backups succeeded again")
			continue
		}
		switch inc.State {
		case repohealth.IncidentAwaiting:
			if err := w.flow.Raise(ctx, inc); err != nil {
				w.logger.Printf("backup failure question for destination %s: %v", inc.DestinationID, err)
			}
		case repohealth.IncidentLapsed:
			if w.stillFailing(ctx, inc, plans) {
				w.reask(ctx, inc)
			}
		case repohealth.IncidentDeclined, repohealth.IncidentFailed:
			if w.now().Sub(lastTouch(inc)) >= backupReaskAfter && w.stillFailing(ctx, inc, plans) {
				w.reask(ctx, inc)
			}
		}
	}
}

// reask opens the next generation of an incident's question and publishes it.
func (w *BackupFailureWorker) reask(ctx context.Context, inc repohealth.Incident) {
	inc = w.reopen(inc, inc.Diagnosis, inc.Offers, inc.PlanIDs)
	if err := w.incidents.Put(ctx, inc); err != nil {
		w.logger.Printf("backup failure re-ask could not be recorded: %v", err)
		return
	}
	if err := w.flow.Raise(ctx, inc); err != nil {
		w.logger.Printf("backup failure re-ask for destination %s: %v", inc.DestinationID, err)
	}
}

func (w *BackupFailureWorker) plansOf(inc repohealth.Incident, plans []PlanRecord) []PlanRecord {
	want := map[string]bool{}
	for _, id := range inc.PlanIDs {
		want[id] = true
	}
	var out []PlanRecord
	for _, p := range plans {
		if want[p.ID] && p.Enabled {
			out = append(out, p)
		}
	}
	return out
}

func (w *BackupFailureWorker) recovered(ctx context.Context, inc repohealth.Incident, plans []PlanRecord) bool {
	for _, p := range w.plansOf(inc, plans) {
		run, ok, err := w.history.LatestRun(ctx, p.ID)
		if err == nil && ok && run.Terminal && run.FinishedAt.After(inc.CreatedAt) && !run.FailsDestination(inc.DestinationID) &&
			(run.Status == "completed" || run.Status == "partial_failed") {
			return true
		}
	}
	return false
}

func (w *BackupFailureWorker) stillFailing(ctx context.Context, inc repohealth.Incident, plans []PlanRecord) bool {
	for _, p := range w.plansOf(inc, plans) {
		run, ok, err := w.history.LatestRun(ctx, p.ID)
		if err == nil && ok && run.Terminal && run.FailsDestination(inc.DestinationID) {
			return true
		}
	}
	return false
}

func (w *BackupFailureWorker) resolve(ctx context.Context, inc repohealth.Incident, why string) {
	now := w.now().UTC()
	inc.State, inc.ResolvedAt, inc.UpdatedAt, inc.Result = repohealth.IncidentResolved, now, now, why
	if err := w.incidents.Put(ctx, inc); err != nil {
		w.logger.Printf("backup failure incident could not be closed: %v", err)
		return
	}
	if w.publisher == nil {
		return
	}
	event := backupNotification("Backups to "+inc.Diagnosis.DriveSentence()+" are working again",
		"Backups to "+inc.Diagnosis.DriveSentence()+" succeeded again ("+why+"). Nothing more is needed from you.",
		categoryBackupRecovered, "normal", "backup-resolved:"+inc.ID+":"+fmt.Sprint(inc.Generation), now)
	if err := w.publisher.PublishDomainEvent(ctx, event); err != nil {
		w.logger.Printf("backup recovered notice failed: %v", err)
	}
}

func (w *BackupFailureWorker) retry(ctx context.Context, planIDs []string) {
	for _, id := range planIDs {
		if err := w.history.TriggerRun(ctx, id); err != nil {
			w.logger.Printf("backup retry for plan %s: %v", id, err)
		}
	}
}

// repairPlannable reports whether the control plane can plan a complete,
// identity-bound unmount, repair and remount for this destination's drive.
func (w *BackupFailureWorker) repairPlannable(ctx context.Context, dest destinations.Destination) bool {
	if w.flow == nil || w.flow.readiness == nil || dest.DeviceIdentity == nil || !dest.DeviceIdentity.StrongIdentity() ||
		strings.TrimSpace(dest.DeviceIdentity.Mountpoint) == "" {
		return false
	}
	_, err := w.flow.planRepairSteps(ctx, filepath.Clean(dest.DeviceIdentity.Mountpoint), *dest.DeviceIdentity)
	return err == nil
}

// diagnose runs the read-only checks for one destination.
func (w *BackupFailureWorker) diagnose(ctx context.Context, dest destinations.Destination, plans []PlanRecord) repohealth.Diagnosis {
	using := w.enabledPlansUsing(dest.ID, plans)
	successes := w.loadSuccesses(ctx, plans)
	labels := w.labelsFor(ctx, plans)
	return repohealth.Diagnose(ctx, repohealth.DiagnoseInput{
		Now: w.now(), DestinationID: dest.ID, DestinationName: dest.Name, Location: dest.Location,
		RepositoryRoot: dest.RepositoryLocation,
		Volume:         func(ctx context.Context) (repohealth.VolumeState, error) { return w.volumeState(ctx, dest) },
		Engine:         w.usage, Scan: w.usage.CachedScan, Scanner: w.usage.Scanner, Exceptions: w.exceptions,
		LastSuccess: func(_ context.Context, snaps []engine.Snapshot) (time.Time, string) {
			return w.lastSuccess(using, successes, labels, snaps)
		},
		PriorRepairAt: func(context.Context) time.Time { return w.priorRepair(dest) },
		Plans:         func(context.Context) repohealth.PlanSummary { return w.planSummary(using, plans, successes, labels) },
	})
}

func (w *BackupFailureWorker) enabledPlansUsing(destID string, plans []PlanRecord) []PlanRecord {
	var out []PlanRecord
	for _, p := range plans {
		if !p.Enabled {
			continue
		}
		for _, d := range p.DestinationIDs {
			if d == destID {
				out = append(out, p)
				break
			}
		}
	}
	return out
}

func (w *BackupFailureWorker) volumeState(ctx context.Context, dest destinations.Destination) (repohealth.VolumeState, error) {
	if dest.DeviceIdentity == nil || w.readiness == nil {
		return repohealth.VolumeState{}, nil
	}
	insp, err := w.readiness.InspectDevice(ctx, *dest.DeviceIdentity)
	if err != nil {
		return repohealth.VolumeState{Detail: err.Error()}, nil
	}
	return repohealth.VolumeState{
		Known: true, Mounted: insp.Mounted, ReadOnly: insp.ReadOnly || insp.DeviceWriteProtected, Label: dest.DeviceIdentity.Label,
		Detail: "at " + insp.Identity.Mountpoint,
	}, nil
}

// priorRepair is when a filesystem repair of this drive last completed through
// the recovery journal, or zero.
func (w *BackupFailureWorker) priorRepair(dest destinations.Destination) time.Time {
	if w.recoveries == nil || dest.DeviceIdentity == nil || strings.TrimSpace(w.recoveries.Root) == "" {
		return time.Time{}
	}
	entries, err := os.ReadDir(w.recoveries.Root)
	if err != nil {
		return time.Time{}
	}
	var latest time.Time
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		journal, err := w.recoveries.Load(context.Background(), strings.TrimSuffix(e.Name(), ".json"))
		if err != nil || filepath.Clean(journal.Location) != filepath.Clean(dest.DeviceIdentity.Mountpoint) {
			continue
		}
		for _, step := range journal.Steps {
			if step.Plan.Action == destinationreadiness.ActionRepairFilesystem && step.Status == destinationreadiness.RecoveryStepCompleted && journal.UpdatedAt.After(latest) {
				latest = journal.UpdatedAt
			}
		}
	}
	return latest.UTC()
}

func (w *BackupFailureWorker) loadSuccesses(ctx context.Context, plans []PlanRecord) map[string]PlanSuccess {
	out := make(map[string]PlanSuccess, len(plans))
	for _, p := range plans {
		s, err := w.history.PlanSuccess(ctx, p.ID)
		if err != nil {
			w.logger.Printf("backup success history for plan %s: %v", p.ID, err)
			continue
		}
		out[p.ID] = s
	}
	return out
}

func (w *BackupFailureWorker) labelsFor(ctx context.Context, plans []PlanRecord) map[string]TargetLabel {
	seen := map[string]bool{}
	var ids []string
	for _, p := range plans {
		for _, t := range p.TargetIDs {
			if !seen[t] {
				seen[t] = true
				ids = append(ids, t)
			}
		}
	}
	labels, err := w.history.TargetLabels(ctx, ids)
	if err != nil {
		w.logger.Printf("backup target labels: %v", err)
		return map[string]TargetLabel{}
	}
	return labels
}

// protectedElsewhere reports whether target id has another copy: a different,
// enabled plan that backed it up within that plan's own stall threshold.
func (w *BackupFailureWorker) protectedElsewhere(id string, except string, plans []PlanRecord, successes map[string]PlanSuccess) bool {
	for _, q := range plans {
		if q.ID == except || !q.Enabled {
			continue
		}
		for _, t := range q.TargetIDs {
			if t != id {
				continue
			}
			if at := successes[q.ID].ByTarget[id]; !at.IsZero() && w.now().Sub(at) <= repohealth.StallThreshold(q.Schedule) {
				return true
			}
		}
	}
	return false
}

func (w *BackupFailureWorker) unprotectedTargets(p PlanRecord, plans []PlanRecord, successes map[string]PlanSuccess) []string {
	var out []string
	for _, t := range p.TargetIDs {
		if !w.protectedElsewhere(t, p.ID, plans, successes) {
			out = append(out, t)
		}
	}
	return out
}

func (w *BackupFailureWorker) planSummary(using []PlanRecord, all []PlanRecord, successes map[string]PlanSuccess, labels map[string]TargetLabel) repohealth.PlanSummary {
	summary := repohealth.PlanSummary{}
	total, unprotected := map[string]bool{}, map[string]bool{}
	for _, p := range using {
		facts := repohealth.PlanFacts{PlanID: p.ID, PlanName: p.Name, Enabled: p.Enabled, Schedule: p.Schedule, TargetCount: len(p.TargetIDs)}
		for _, t := range w.unprotectedTargets(p, all, successes) {
			facts.UnprotectedTargets++
			unprotected[t] = true
		}
		for _, t := range p.TargetIDs {
			total[t] = true
		}
		summary.Plans = append(summary.Plans, facts)
	}
	summary.TargetTotal, summary.Unprotected = len(total), len(unprotected)
	for t := range unprotected {
		if l, ok := labels[t]; ok {
			summary.UnprotectedNames = append(summary.UnprotectedNames, l.String())
		}
	}
	sort.Strings(summary.UnprotectedNames)
	return summary
}

// lastSuccess is the later of the recorded run history and the newest
// repository snapshot belonging to one of these plans' targets.
func (w *BackupFailureWorker) lastSuccess(using []PlanRecord, successes map[string]PlanSuccess, labels map[string]TargetLabel, snaps []engine.Snapshot) (time.Time, string) {
	var recorded time.Time
	targets := map[string]bool{}
	for _, p := range using {
		if at := successes[p.ID].At; at.After(recorded) {
			recorded = at
		}
		for _, t := range p.TargetIDs {
			targets[t] = true
		}
	}
	repo := latestSnapshotFor(snaps, targets, labels)
	switch {
	case repo.After(recorded):
		return repo.UTC(), "repository snapshot"
	case !recorded.IsZero():
		return recorded.UTC(), "recorded run"
	default:
		return time.Time{}, ""
	}
}

func latestSnapshotFor(snaps []engine.Snapshot, targets map[string]bool, labels map[string]TargetLabel) time.Time {
	var latest time.Time
	for _, s := range snaps {
		started, err := time.Parse(time.RFC3339Nano, s.StartTime)
		if err != nil || !started.After(latest) {
			continue
		}
		for t := range targets {
			if l, ok := labels[t]; ok && repohealth.SnapshotSourceMatches(s.Source, l.Owner, l.Name) {
				latest = started
				break
			}
		}
	}
	return latest
}

// repoSuccess returns the newest repository snapshot time for the given
// targets on a destination, cached because it lists every snapshot.
func (w *BackupFailureWorker) repoSuccess(ctx context.Context, dest destinations.Destination, targets map[string]bool, labels map[string]TargetLabel) time.Time {
	ev, ok := w.evidence[dest.ID]
	ttl := repoEvidenceTTL
	if ok && !ev.ok {
		ttl = repoEvidenceFailedTTL
	}
	if !ok || w.now().Sub(ev.at) >= ttl {
		ev = repoEvidence{at: w.now()}
		snaps, err := w.usage.SnapshotList(ctx, dest.Name, "")
		if err == nil {
			ev.ok, ev.latestBy = true, map[string]time.Time{}
			for _, s := range snaps {
				started, perr := time.Parse(time.RFC3339Nano, s.StartTime)
				if perr != nil {
					continue
				}
				for _, l := range labels {
					if repohealth.SnapshotSourceMatches(s.Source, l.Owner, l.Name) && started.After(ev.latestBy[l.String()]) {
						ev.latestBy[l.String()] = started
					}
				}
			}
		}
		w.evidence[dest.ID] = ev
	}
	var latest time.Time
	for t := range targets {
		if l, ok := labels[t]; ok && ev.latestBy[l.String()].After(latest) {
			latest = ev.latestBy[l.String()]
		}
	}
	return latest
}

// reconcileNotices sends the stalled-backup, switched-off-plan and standing
// exception notices that are due.
func (w *BackupFailureWorker) reconcileNotices(ctx context.Context, plans []PlanRecord, dests map[string]destinations.Destination) {
	if w.notices == nil || w.publisher == nil {
		return
	}
	state, err := w.notices.Load()
	if err != nil {
		w.logger.Printf("backup notice state unreadable: %v", err)
		return
	}
	statuses := w.planStatuses(ctx, plans, dests)
	now := w.now().UTC()
	var due []repohealth.Notice
	due = append(due, repohealth.BundleStalled(repohealth.EvaluateStalled(now, statuses, state), now)...)
	if n, ok := repohealth.EvaluateDisabledUnprotected(statuses, state); ok {
		due = append(due, n)
	}
	due = append(due, w.exceptionNotices(ctx, dests, state)...)
	for _, n := range due {
		dedupe := n.Key
		if n.Kind != repohealth.KindDisabledUnprotec {
			dedupe += ":" + now.Format("2006-01-02")
		}
		if err := w.publisher.PublishDomainEvent(ctx, backupNotification(n.Title, n.Body, n.Category, n.Urgency, dedupe, now)); err != nil {
			w.logger.Printf("backup notice %s not delivered: %v", n.Kind, err)
			continue
		}
		for _, key := range n.CoveredKeys() {
			if err := w.notices.Mark(key, now); err != nil {
				w.logger.Printf("backup notice %s could not be recorded: %v", n.Kind, err)
			}
		}
	}
}

func (w *BackupFailureWorker) planStatuses(ctx context.Context, plans []PlanRecord, dests map[string]destinations.Destination) []repohealth.PlanStatus {
	successes := w.loadSuccesses(ctx, plans)
	labels := w.labelsFor(ctx, plans)
	open, _ := w.incidents.List(ctx)
	out := make([]repohealth.PlanStatus, 0, len(plans))
	for _, p := range plans {
		st := repohealth.PlanStatus{PlanID: p.ID, Name: p.Name, Enabled: p.Enabled, Schedule: p.Schedule, CreatedAt: p.CreatedAt, TargetCount: len(p.TargetIDs)}
		if s := successes[p.ID]; !s.At.IsZero() {
			st.LastSuccessAt, st.LastSuccessSource = s.At, "recorded run"
		}
		if p.Enabled && (st.LastSuccessAt.IsZero() || w.now().Sub(st.LastSuccessAt) > repohealth.StallThreshold(p.Schedule)) {
			// Possibly stalled: the repository may hold newer proof than the run log.
			targets := map[string]bool{}
			for _, t := range p.TargetIDs {
				targets[t] = true
			}
			for _, destID := range p.DestinationIDs {
				if dest, ok := dests[destID]; ok && dest.BackendKind == destinations.BackendFilesystem && w.usage != nil {
					if at := w.repoSuccess(ctx, dest, targets, labels); at.After(st.LastSuccessAt) {
						st.LastSuccessAt, st.LastSuccessSource = at, "repository snapshot"
					}
				}
			}
		}
		for _, t := range w.unprotectedTargets(p, plans, successes) {
			st.Unprotected++
			if l, ok := labels[t]; ok {
				st.UnprotectedTargets = append(st.UnprotectedTargets, l.String())
			}
		}
		for _, inc := range open {
			if inc.State == repohealth.IncidentAwaiting && contains(p.DestinationIDs, inc.DestinationID) {
				st.OpenDecision = inc.Subject
			}
		}
		out = append(out, st)
	}
	return out
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// exceptionNotices builds the daily standing warning for each destination whose
// recorded exceptions still apply: the excepted path is still an unreadable
// entry of the same kind.
func (w *BackupFailureWorker) exceptionNotices(ctx context.Context, dests map[string]destinations.Destination, state repohealth.NoticeState) []repohealth.Notice {
	if w.exceptions == nil || w.usage == nil {
		return nil
	}
	var out []repohealth.Notice
	ids := make([]string, 0, len(dests))
	for id := range dests {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		dest := dests[id]
		recorded, err := w.exceptions.List(ctx, id)
		if err != nil || len(recorded) == 0 {
			continue
		}
		var active []repohealth.Exception
		for _, ex := range recorded {
			_, statErr := w.usage.Scanner.Filesystem().Lstat(filepath.Join(dest.RepositoryLocation, filepath.FromSlash(ex.RelPath)))
			if statErr != nil && repohealth.ProblemFor(statErr) == ex.Problem {
				active = append(active, ex)
			}
		}
		name := dest.Name
		if dest.DeviceIdentity != nil && strings.TrimSpace(dest.DeviceIdentity.Label) != "" {
			name = "the " + dest.DeviceIdentity.Label + " drive"
		}
		if n, ok := repohealth.EvaluateExceptionActive(w.now().UTC(), id, name, active, state); ok {
			out = append(out, n)
		}
	}
	return out
}
