package drills

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

type Service interface {
	Preview(ctx context.Context, planID, targetID, destinationID string) (Preview, error)
	Run(ctx context.Context, planID, targetID, destinationID, idempotencyKey string, scheduled bool) (Drill, error)
	Get(ctx context.Context, id string) (Drill, error)
	List(ctx context.Context, planID, targetID string, limit int) ([]Drill, error)
	RunDue(ctx context.Context) error
	Reconcile(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

type Deps struct {
	Repo         Repository
	Plans        PlanLookup
	Snapshots    SnapshotLookup
	Restores     RestoreRunner
	Clock        Clock
	BaseContext  context.Context
	Workers      int
	PollInterval time.Duration
	PollTimeout  time.Duration
	Logger       *log.Logger
}

type service struct {
	deps   Deps
	queue  chan Drill
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

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
	s := &service{deps: d, queue: make(chan Drill, 128), cancel: cancel}
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
	if key := strings.TrimSpace(idempotencyKey); key != "" {
		if existing, ok, err := s.deps.Repo.FindByIdempotency(ctx, key); err != nil {
			return Drill{}, err
		} else if ok {
			return existing, nil
		}
	}
	if existing, ok, err := s.deps.Repo.LatestForUnit(ctx, plan.ID, targetID, destinationID); err != nil {
		return Drill{}, err
	} else if ok && (existing.Status == StatusRequested || existing.Status == StatusRunning) {
		return Drill{}, ErrAlreadyActive{ID: existing.ID}
	}

	now := s.deps.Clock.Now().UTC()
	d := Drill{PlanID: plan.ID, TargetID: targetID, DestinationID: destinationID, Status: StatusRequested, Scheduled: scheduled, RequestedAt: now, IdempotencyKey: strings.TrimSpace(idempotencyKey)}
	snapshot, ok, err := s.deps.Snapshots.LatestSuccessfulSnapshot(ctx, plan.ID, targetID, destinationID)
	if err != nil {
		return Drill{}, fmt.Errorf("find latest successful snapshot: %w", err)
	}
	if !ok || snapshot.ID == "" {
		d.Status = StatusFailed
		d.Error = "no successful backup snapshot exists for this target and destination"
		d.NextAction = "run the backup plan successfully, then retry the recovery drill"
		d.FinishedAt = now
		return s.deps.Repo.Create(ctx, d)
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
	if err := s.deps.Repo.MarkRunning(ctx, d.ID, "", s.deps.Clock.Now().UTC()); err != nil {
		s.deps.Logger.Printf("drills: mark %s running: %v", d.ID, err)
		s.failWithAction(ctx, d.ID, fmt.Sprintf("could not persist drill start: %v", err), "repair catalog storage, then retry the recovery drill")
		return
	}
	r, err := s.deps.Restores.VerifyTarget(ctx, d.TargetID, d.DestinationID, d.SnapshotID)
	if err != nil {
		s.fail(ctx, d.ID, fmt.Sprintf("start verified restore: %v", err))
		return
	}
	if err := s.deps.Repo.MarkRunning(ctx, d.ID, r.ID, s.deps.Clock.Now().UTC()); err != nil {
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
			if err := s.deps.Repo.Finish(ctx, d.ID, StatusVerified, "", "", s.deps.Clock.Now().UTC()); err != nil {
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
	if err := s.deps.Repo.Finish(ctx, id, StatusFailed, reason, nextAction, s.deps.Clock.Now().UTC()); err != nil {
		s.deps.Logger.Printf("drills: finish %s: %v", id, err)
	}
}

func (s *service) Get(ctx context.Context, id string) (Drill, error) {
	if strings.TrimSpace(id) == "" {
		return Drill{}, ErrInvalid{Field: "id", Reason: "required"}
	}
	return s.deps.Repo.Get(ctx, id)
}

func (s *service) List(ctx context.Context, planID, targetID string, limit int) ([]Drill, error) {
	return s.deps.Repo.List(ctx, strings.TrimSpace(planID), strings.TrimSpace(targetID), limit)
}

func (s *service) RunDue(ctx context.Context) error {
	plans, err := s.deps.Plans.SchedulableDrillPlans(ctx)
	if err != nil {
		return err
	}
	now := s.deps.Clock.Now().UTC()
	for _, p := range plans {
		if !p.Enabled || p.DrillSchedule == "" {
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
					if last.Status == StatusRequested || last.Status == StatusRunning {
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
		if d.Status == StatusRequested || d.Status == StatusRunning {
			if err := s.deps.Repo.Finish(ctx, d.ID, StatusFailed, "reconciled: process restarted while recovery drill was in-flight", "retry the recovery drill after verifying service health", now); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *service) Shutdown(ctx context.Context) error {
	s.cancel()
	done := make(chan struct{})
	go func() { s.wg.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
