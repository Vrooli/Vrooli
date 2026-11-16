package drills

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/eventbus"
)

type Service interface {
	Preview(ctx context.Context, planID, targetID, destinationID string) (Preview, error)
	Run(ctx context.Context, planID, targetID, destinationID, idempotencyKey string, scheduled bool) (Drill, error)
	// RunMigrationSnapshot drills one exact destination-bound migration capture
	// before that destination belongs to the source plan. It uses the same durable
	// drill repository, readiness gate and verified restore worker as ordinary drills.
	RunMigrationSnapshot(ctx context.Context, planID, targetID, destinationID, snapshotID, idempotencyKey string) (Drill, error)
	Get(ctx context.Context, id string) (Drill, error)
	Retire(ctx context.Context, id, reason string) (Drill, error)
	List(ctx context.Context, planID, targetID string, limit int) ([]Drill, error)
	ListPage(ctx context.Context, planID, targetID string, limit int, pageToken string) (DrillPage, error)
	RunDue(ctx context.Context) error
	Reconcile(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

type Deps struct {
	Repo          Repository
	Plans         PlanLookup
	Snapshots     SnapshotLookup
	Readiness     DestinationReadiness
	Restores      RestoreRunner
	Notifications NotificationPublisher
	Clock         Clock
	BaseContext   context.Context
	Workers       int
	PollInterval  time.Duration
	PollTimeout   time.Duration
	Logger        *log.Logger
}

// DestinationReadiness supplies the same read-only destination gate used by
// backup preflight. A successful snapshot alone is not sufficient evidence
// that a restore can safely use a filesystem destination now.
type DestinationReadiness interface {
	ReadyForDrill(context.Context, string) (bool, string, error)
}

type service struct {
	deps                    Deps
	queue                   chan Drill
	ctx                     context.Context
	cancel                  context.CancelFunc
	wg                      sync.WaitGroup
	notificationTimerWG     sync.WaitGroup
	failureStateMu          sync.Mutex
	notificationDispatchMu  sync.Mutex
	notificationTimerMu     sync.Mutex
	notificationTimer       *time.Timer
	notificationTimerClosed bool
}

const (
	drillNotificationBatchWindow  = 15 * time.Second
	pendingDrillNotificationLimit = 1000000
)

func NewService(d Deps) Service {
	if d.Clock == nil {
		d.Clock = systemClock{}
	}
	if d.BaseContext == nil {
		d.BaseContext = context.Background()
	}
	if d.Workers < 1 {
		d.Workers = 1
	}
	if d.PollInterval <= 0 {
		d.PollInterval = time.Second
	}
	if d.PollTimeout <= 0 {
		d.PollTimeout = 30 * time.Minute
	}
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	ctx, cancel := context.WithCancel(d.BaseContext)
	s := &service{deps: d, queue: make(chan Drill, 128), ctx: ctx, cancel: cancel}
	for i := 0; i < d.Workers; i++ {
		s.wg.Add(1)
		go s.worker(ctx)
	}
	return s
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

func (s *service) Preview(ctx context.Context, planID, targetID, destinationID string) (Preview, error) {
	plan, targetID, destinationID, err := s.resolveUnit(ctx, planID, targetID, destinationID)
	if err != nil {
		return Preview{}, err
	}
	preview := Preview{Eligible: true, PlanID: plan.ID, TargetID: targetID, DestinationID: destinationID}
	snapshot, ok, err := s.deps.Snapshots.LatestSuccessfulSnapshot(ctx, plan.ID, targetID, destinationID)
	if err != nil {
		return Preview{}, fmt.Errorf("find latest successful snapshot: %w", err)
	}
	if !ok || strings.TrimSpace(snapshot.ID) == "" {
		preview.Eligible = false
		preview.Reason = "no successful backup snapshot exists for this target and destination"
		preview.Warnings = []string{"run the selected backup plan successfully before starting a recovery drill"}
	} else if s.deps.Readiness != nil {
		ready, reason, readinessErr := s.deps.Readiness.ReadyForDrill(ctx, destinationID)
		if readinessErr != nil {
			preview.Eligible = false
			preview.Reason = fmt.Sprintf("destination readiness could not be verified: %v", readinessErr)
		} else if !ready {
			preview.Eligible = false
			preview.Reason = reason
		}
	}
	if plan.DrillSchedule == "" {
		preview.Warnings = append(preview.Warnings, "this plan has no automated drill schedule; manual drills remain available")
	}
	preview.SnapshotID = snapshot.ID
	return preview, nil
}

func (s *service) Run(ctx context.Context, planID, targetID, destinationID, idempotencyKey string, scheduled bool) (Drill, error) {
	plan, targetID, destinationID, err := s.resolveUnit(ctx, planID, targetID, destinationID)
	if err != nil {
		return Drill{}, err
	}
	return s.runSnapshot(ctx, plan, targetID, destinationID, "", idempotencyKey, scheduled)
}

func (s *service) RunMigrationSnapshot(ctx context.Context, planID, targetID, destinationID, snapshotID, idempotencyKey string) (Drill, error) {
	planID, targetID, destinationID, snapshotID = strings.TrimSpace(planID), strings.TrimSpace(targetID), strings.TrimSpace(destinationID), strings.TrimSpace(snapshotID)
	if planID == "" || targetID == "" || destinationID == "" || snapshotID == "" || strings.TrimSpace(idempotencyKey) == "" {
		return Drill{}, ErrInvalid{Field: "migration", Reason: "plan, target, destination, exact snapshot, and idempotency key are required"}
	}
	plan, err := s.deps.Plans.PlanForDrill(ctx, planID)
	if err != nil {
		return Drill{}, err
	}
	if !contains(plan.TargetIDs, targetID) {
		return Drill{}, ErrInvalid{Field: "target_id", Reason: "target is not a member of the source plan"}
	}
	return s.runSnapshot(ctx, plan, targetID, destinationID, snapshotID, idempotencyKey, false)
}

func (s *service) runSnapshot(ctx context.Context, plan Plan, targetID, destinationID, snapshotID, idempotencyKey string, scheduled bool) (Drill, error) {
	if key := strings.TrimSpace(idempotencyKey); key != "" {
		if existing, ok, err := s.deps.Repo.FindByIdempotency(ctx, key); err != nil {
			return Drill{}, err
		} else if ok {
			return existing, nil
		}
	}
	if existing, ok, err := s.deps.Repo.LatestForUnit(ctx, plan.ID, targetID, destinationID); err != nil {
		return Drill{}, err
	} else if ok && IsActive(existing.Status) {
		return Drill{}, ErrAlreadyActive{ID: existing.ID}
	}

	now := s.deps.Clock.Now().UTC()
	d := Drill{PlanID: plan.ID, TargetID: targetID, DestinationID: destinationID, Status: StatusRequested, Scheduled: scheduled, RequestedAt: now, IdempotencyKey: strings.TrimSpace(idempotencyKey)}
	var snapshot Snapshot
	var err error
	ok := strings.TrimSpace(snapshotID) != ""
	if ok {
		snapshot.ID = snapshotID
	} else {
		snapshot, ok, err = s.deps.Snapshots.LatestSuccessfulSnapshot(ctx, plan.ID, targetID, destinationID)
		if err != nil {
			return Drill{}, fmt.Errorf("find latest successful snapshot: %w", err)
		}
	}
	if !ok || snapshot.ID == "" {
		if err := d.TransitionStatus(EventFail); err != nil {
			return Drill{}, err
		}
		d.Error = "no successful backup snapshot exists for this target and destination"
		d.NextAction = "run the backup plan successfully, then retry the recovery drill"
		d.FinishedAt = now
		s.failureStateMu.Lock()
		created, createErr := s.deps.Repo.Create(ctx, d)
		s.failureStateMu.Unlock()
		if createErr == nil {
			if notifyErr := s.dispatchPendingNotifications(ctx); notifyErr != nil {
				s.deps.Logger.Printf("drills: dispatch failed notification %s: %v", created.ID, notifyErr)
			}
			if latest, readErr := s.deps.Repo.Get(ctx, created.ID); readErr == nil {
				created = latest
			} else {
				s.deps.Logger.Printf("drills: read failure notification state %s: %v", created.ID, readErr)
			}
		}
		return created, createErr
	}
	d.SnapshotID = snapshot.ID
	created, err := s.deps.Repo.Create(ctx, d)
	if err != nil {
		return Drill{}, fmt.Errorf("create recovery drill: %w", err)
	}
	select {
	case s.queue <- created:
	case <-ctx.Done():
		return Drill{}, ctx.Err()
	}
	return created, nil
}

func (s *service) resolveUnit(ctx context.Context, planID, targetID, destinationID string) (Plan, string, string, error) {
	planID = strings.TrimSpace(planID)
	targetID = strings.TrimSpace(targetID)
	destinationID = strings.TrimSpace(destinationID)
	if planID == "" {
		return Plan{}, "", "", ErrInvalid{Field: "plan_id", Reason: "required"}
	}
	plan, err := s.deps.Plans.PlanForDrill(ctx, planID)
	if err != nil {
		return Plan{}, "", "", err
	}
	if targetID == "" {
		if len(plan.TargetIDs) != 1 {
			return Plan{}, "", "", ErrInvalid{Field: "target_id", Reason: "required when the plan has multiple targets"}
		}
		targetID = plan.TargetIDs[0]
	}
	if destinationID == "" {
		if len(plan.DestinationIDs) != 1 {
			return Plan{}, "", "", ErrInvalid{Field: "destination_id", Reason: "required when the plan has multiple destinations"}
		}
		destinationID = plan.DestinationIDs[0]
	}
	if !contains(plan.TargetIDs, targetID) {
		return Plan{}, "", "", ErrInvalid{Field: "target_id", Reason: "target is not a member of the plan"}
	}
	if !contains(plan.DestinationIDs, destinationID) {
		return Plan{}, "", "", ErrInvalid{Field: "destination_id", Reason: "destination is not a member of the plan"}
	}
	return plan, targetID, destinationID, nil
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

func (s *service) worker(ctx context.Context) {
	defer s.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case d := <-s.queue:
			s.execute(ctx, d)
		}
	}
}

func (s *service) execute(ctx context.Context, d Drill) {
	if err := s.deps.Repo.Start(ctx, d.ID, s.deps.Clock.Now().UTC()); err != nil {
		s.deps.Logger.Printf("drills: mark %s running: %v", d.ID, err)
		s.failWithAction(ctx, d.ID, fmt.Sprintf("could not persist drill start: %v", err), "repair catalog storage, then retry the recovery drill")
		return
	}
	r, err := s.deps.Restores.VerifyTarget(ctx, d.TargetID, d.DestinationID, d.SnapshotID)
	if err != nil {
		s.fail(ctx, d.ID, fmt.Sprintf("start verified restore: %v", err))
		return
	}
	if err := s.deps.Repo.AttachRestore(ctx, d.ID, r.ID); err != nil {
		s.deps.Logger.Printf("drills: record restore %s: %v", d.ID, err)
		s.failWithAction(ctx, d.ID, fmt.Sprintf("verified restore started but drill linkage could not be persisted: %v", err), "inspect the linked restore and catalog storage before retrying the drill")
		return
	}
	deadline := time.Now().Add(s.deps.PollTimeout)
	for {
		current, err := s.deps.Restores.GetRestore(ctx, r.ID)
		if err != nil {
			s.fail(ctx, d.ID, fmt.Sprintf("get verified restore: %v", err))
			return
		}
		switch strings.ToLower(current.Status) {
		case "verified":
			if err := s.deps.Repo.Finish(ctx, d.ID, EventVerify, "", "", s.deps.Clock.Now().UTC()); err != nil {
				// A successful restore without durable drill evidence is not a
				// successful drill. Try to leave an explicit failed record rather
				// than silently abandoning the requested/running row.
				reason := fmt.Sprintf("verified restore succeeded but drill evidence could not be committed: %v", err)
				s.deps.Logger.Printf("drills: finish verified %s: %v", d.ID, err)
				s.failWithAction(ctx, d.ID, reason, "retry the drill after confirming catalog storage is healthy; the linked restore was verified but drill evidence was not committed")
			}
			return
		case "failed":
			reason := current.Error
			if reason == "" {
				reason = "verified restore failed"
			}
			s.fail(ctx, d.ID, reason)
			return
		}
		if time.Now().After(deadline) {
			s.fail(ctx, d.ID, "verified restore did not reach a terminal state before the drill timeout")
			return
		}
		t := time.NewTimer(s.deps.PollInterval)
		select {
		case <-ctx.Done():
			t.Stop()
			return
		case <-t.C:
		}
	}
}

func (s *service) fail(ctx context.Context, id, reason string) {
	s.failWithAction(ctx, id, reason, "inspect the linked restore record and repair the backup destination or source before retrying")
}

func (s *service) failWithAction(ctx context.Context, id, reason, nextAction string) {
	s.failureStateMu.Lock()
	err := s.deps.Repo.Finish(ctx, id, EventFail, reason, nextAction, s.deps.Clock.Now().UTC())
	s.failureStateMu.Unlock()
	if err != nil {
		s.deps.Logger.Printf("drills: finish %s: %v", id, err)
		return
	}
	s.dispatchPendingNotifications(ctx)
}

func (s *service) Get(ctx context.Context, id string) (Drill, error) {
	if strings.TrimSpace(id) == "" {
		return Drill{}, ErrInvalid{Field: "id", Reason: "required"}
	}
	return s.deps.Repo.Get(ctx, id)
}

func (s *service) Retire(ctx context.Context, id, reason string) (Drill, error) {
	id = strings.TrimSpace(id)
	reason = strings.TrimSpace(reason)
	if id == "" {
		return Drill{}, ErrInvalid{Field: "id", Reason: "required"}
	}
	if reason == "" {
		return Drill{}, ErrInvalid{Field: "reason", Reason: "required"}
	}
	current, err := s.deps.Repo.Get(ctx, id)
	if err != nil {
		return Drill{}, err
	}
	if !IsTerminal(current.Status) {
		return Drill{}, ErrInvalid{Field: "status", Reason: "only completed recovery drills can be retired"}
	}
	return s.deps.Repo.Retire(ctx, id, reason, s.deps.Clock.Now().UTC())
}

func (s *service) List(ctx context.Context, planID, targetID string, limit int) ([]Drill, error) {
	return s.deps.Repo.List(ctx, strings.TrimSpace(planID), strings.TrimSpace(targetID), limit)
}

func (s *service) ListPage(ctx context.Context, planID, targetID string, limit int, pageToken string) (DrillPage, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	return s.deps.Repo.ListPage(ctx, strings.TrimSpace(planID), strings.TrimSpace(targetID), limit, strings.TrimSpace(pageToken))
}

func (s *service) RunDue(ctx context.Context) error {
	if err := s.dispatchPendingNotifications(ctx); err != nil {
		return err
	}
	plans, err := s.deps.Plans.SchedulableDrillPlans(ctx)
	if err != nil {
		return err
	}
	now := s.deps.Clock.Now().UTC()
	for _, p := range plans {
		if !p.Enabled || p.DrillSchedule == "" || p.DrillSchedule == "manual" {
			continue
		}
		interval, e := time.ParseDuration(p.DrillSchedule)
		if e != nil {
			return fmt.Errorf("plan %q has invalid recovery drill schedule %q: %w", p.ID, p.DrillSchedule, e)
		}
		if interval <= 0 {
			return fmt.Errorf("plan %q has invalid recovery drill schedule %q: duration must be positive", p.ID, p.DrillSchedule)
		}
		for _, tid := range p.TargetIDs {
			for _, did := range p.DestinationIDs {
				last, ok, e := s.deps.Repo.LatestForUnit(ctx, p.ID, tid, did)
				if e != nil {
					return e
				}
				if ok {
					if IsActive(last.Status) {
						continue
					}
					at := last.FinishedAt
					if at.IsZero() {
						at = last.RequestedAt
					}
					if !at.IsZero() && now.Sub(at) < interval {
						continue
					}
				}
				key := fmt.Sprintf("scheduled:%s:%s:%s:%d", p.ID, tid, did, now.Truncate(interval).UnixNano())
				if _, e = s.Run(ctx, p.ID, tid, did, key, true); e != nil {
					if _, active := e.(ErrAlreadyActive); !active {
						return e
					}
				}
			}
		}
	}
	return nil
}

func (s *service) Reconcile(ctx context.Context) error {
	list, err := s.deps.Repo.List(ctx, "", "", 1000)
	if err != nil {
		return err
	}
	now := s.deps.Clock.Now().UTC()
	for _, d := range list {
		if IsActive(d.Status) {
			s.failureStateMu.Lock()
			err := s.deps.Repo.Finish(ctx, d.ID, EventFail, "reconciled: process restarted while recovery drill was in-flight", "retry the recovery drill after verifying service health", now)
			s.failureStateMu.Unlock()
			if err != nil {
				return err
			}
		}
	}
	return s.dispatchPendingNotifications(ctx)
}

func (s *service) dispatchPendingNotifications(ctx context.Context) error {
	if s.deps.Notifications == nil {
		return nil
	}
	// Prevent a new failure from entering the just-published time window before
	// its exact cohort is marked delivered.
	s.failureStateMu.Lock()
	defer s.failureStateMu.Unlock()
	s.notificationDispatchMu.Lock()
	defer s.notificationDispatchMu.Unlock()
	for {
		pending, err := s.deps.Repo.ListPendingNotifications(ctx, pendingDrillNotificationLimit)
		if err != nil {
			return err
		}
		if len(pending) == 0 {
			return nil
		}
		if !s.deps.Notifications.Enabled() {
			s.deps.Logger.Printf("drills: %d failed notification(s) remain pending because Vrooli Events delivery is not configured", len(pending))
			return nil
		}

		sort.Slice(pending, func(i, j int) bool {
			left, right := drillFailureTime(pending[i]), drillFailureTime(pending[j])
			if left.Equal(right) {
				return pending[i].ID < pending[j].ID
			}
			return left.Before(right)
		})
		first := pending[0]
		occurred := drillFailureTime(first)
		if occurred.IsZero() {
			occurred = time.Now().UTC()
		}
		batchThrough := occurred.Add(drillNotificationBatchWindow)
		batch := make([]Drill, 0, len(pending))
		for _, drill := range pending {
			if !drillFailureTime(drill).After(batchThrough) {
				batch = append(batch, drill)
			}
		}
		if wait := time.Until(batchThrough); wait > 0 {
			s.scheduleNotificationDispatch(wait)
			return nil
		}

		event := drillFailureNotification(first, batch, occurred, batchThrough)
		batchIDs := make([]string, 0, len(batch))
		for _, drill := range batch {
			batchIDs = append(batchIDs, drill.ID)
		}
		if err := s.deps.Notifications.PublishDomainEvent(ctx, event); err != nil {
			// The event envelope and its member window are deterministic. If the
			// hub accepted it before this call failed, republishing has the same
			// event ID and therefore cannot deliver a second message.
			s.deps.Logger.Printf("drills: publish failure notification batch %s: %v", first.ID, err)
			return nil
		}
		if err := s.deps.Repo.MarkNotificationsSent(ctx, batchIDs, s.deps.Clock.Now().UTC()); err != nil {
			s.deps.Logger.Printf("drills: mark failure notification batch %s delivered: %v", first.ID, err)
			return nil
		}
	}
}

func drillFailureTime(drill Drill) time.Time {
	if !drill.FinishedAt.IsZero() {
		return drill.FinishedAt.UTC()
	}
	return drill.RequestedAt.UTC()
}

func drillFailureNotification(first Drill, batch []Drill, occurred, batchThrough time.Time) eventbus.DomainEvent {
	title := "Recovery drill failed"
	body := fmt.Sprintf("Recovery drill %s failed for plan %s, target %s and destination %s. %s. Next action: %s.", first.ID, first.PlanID, first.TargetID, first.DestinationID, strings.TrimSpace(first.Error), strings.TrimSpace(first.NextAction))
	dedupeKey := "recovery-drill-failed:" + first.ID
	correlation := map[string]any{"drill_id": first.ID, "plan_id": first.PlanID, "target_id": first.TargetID, "destination_id": first.DestinationID}
	if first.Scheduled {
		title = "Scheduled recovery drill failed"
	}
	if len(batch) > 1 {
		allScheduled := true
		planCounts := map[string]int{}
		causeCounts := map[string]int{}
		for _, drill := range batch {
			allScheduled = allScheduled && drill.Scheduled
			planCounts[drill.PlanID]++
			cause := strings.TrimSpace(drill.Error)
			if cause == "" {
				cause = "no failure detail recorded"
			}
			causeCounts[cause]++
		}
		if allScheduled {
			title = "Scheduled recovery drills failed"
		} else {
			title = "Recovery drills failed"
		}
		body = fmt.Sprintf("%d recovery drills failed across %d plans.", len(batch), len(planCounts))
		causes := make([]string, 0, len(causeCounts))
		for cause := range causeCounts {
			causes = append(causes, cause)
		}
		sort.Strings(causes)
		for i, cause := range causes {
			if i == 8 {
				body += fmt.Sprintf("\n- %d additional failure causes are in Data Backup Manager.", len(causes)-i)
				break
			}
			body += fmt.Sprintf("\n- %d: %s", causeCounts[cause], cause)
		}
		if len(planCounts) == 1 {
			for planID, count := range planCounts {
				body += fmt.Sprintf("\nPlan %s has %d failed target/destination unit(s).", planID, count)
			}
		} else {
			body += "\nThe failures span multiple plans."
		}
		body += "\nOpen Data Backup Manager's Drills page for each target, destination, and next action."
		dedupeKey = fmt.Sprintf("recovery-drill-failure-batch:%s:%s", first.ID, batchThrough.Format(time.RFC3339Nano))
		correlation = map[string]any{
			"batch_id": first.ID, "failure_count": fmt.Sprint(len(batch)), "plan_count": fmt.Sprint(len(planCounts)),
		}
	}
	return eventbus.DomainEvent{
		Source:    "data-backup-manager",
		EventType: "data_backup_manager.backup_health.notification.v1",
		Occurred:  occurred,
		Payload: map[string]any{
			"title": title, "body": body, "category": "backup-recovery-needs-attention",
			"urgency": "high", "sensitivity": "private",
			"dedupe_key": dedupeKey, "correlation": correlation,
		},
	}
}

func (s *service) scheduleNotificationDispatch(after time.Duration) {
	if after < time.Millisecond {
		after = time.Millisecond
	}
	s.notificationTimerMu.Lock()
	if s.notificationTimerClosed || s.notificationTimer != nil {
		s.notificationTimerMu.Unlock()
		return
	}
	s.notificationTimerWG.Add(1)
	s.notificationTimer = time.AfterFunc(after, func() {
		defer s.notificationTimerWG.Done()
		s.notificationTimerMu.Lock()
		s.notificationTimer = nil
		s.notificationTimerMu.Unlock()
		if err := s.dispatchPendingNotifications(s.ctx); err != nil {
			s.deps.Logger.Printf("drills: dispatch batched failure notifications: %v", err)
		}
	})
	s.notificationTimerMu.Unlock()
}

func (s *service) stopNotificationDispatch() {
	s.notificationTimerMu.Lock()
	s.notificationTimerClosed = true
	if s.notificationTimer != nil {
		if s.notificationTimer.Stop() {
			s.notificationTimerWG.Done()
		}
		s.notificationTimer = nil
	}
	s.notificationTimerMu.Unlock()
}

func (s *service) Shutdown(ctx context.Context) error {
	s.cancel()
	s.stopNotificationDispatch()
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		s.notificationTimerWG.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
