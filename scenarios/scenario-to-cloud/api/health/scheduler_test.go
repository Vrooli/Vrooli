package health

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/health"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/perfbudget"
)

// TestSchedulerRunOnceObservesEachTargetAndContinuesAfterFailure [REQ:STC-P0-036]
// prevents one unreachable target from suppressing later observations.
func TestSchedulerRunOnceObservesEachTargetAndContinuesAfterFailure(t *testing.T) {
	var seen []string
	scheduler := Scheduler{
		Interval: time.Second,
		Targets: func(context.Context) ([]string, error) {
			return []string{"prod-a", "", "prod-b"}, nil
		},
		Observe: func(_ context.Context, id string) (*healthv1.HealthObservation, error) {
			seen = append(seen, id)
			if id == "prod-a" {
				return nil, errors.New("unreachable")
			}
			return &healthv1.HealthObservation{DeploymentId: id}, nil
		},
	}
	scheduler.RunOnce(context.Background())
	if !reflect.DeepEqual(seen, []string{"prod-a", "prod-b"}) {
		t.Fatalf("observed targets = %v", seen)
	}
}

func TestSchedulerRunOnceDoesNotObserveWithoutTargetsOrInterval(t *testing.T) {
	calls := 0
	scheduler := Scheduler{
		Targets: func(context.Context) ([]string, error) { return []string{"prod-a"}, nil },
		Observe: func(context.Context, string) (*healthv1.HealthObservation, error) {
			calls++
			return nil, nil
		},
	}
	scheduler.RunOnce(context.Background())
	if calls != 0 {
		t.Fatalf("observed %d targets without configured interval", calls)
	}
}

func TestSchedulerRunObservesImmediatelyAndStopsWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	scheduler := Scheduler{
		Interval: time.Hour,
		Targets:  func(context.Context) ([]string, error) { return []string{"prod-a"}, nil },
		Observe: func(context.Context, string) (*healthv1.HealthObservation, error) {
			calls++
			cancel()
			return &healthv1.HealthObservation{DeploymentId: "prod-a"}, nil
		},
	}
	scheduler.Run(ctx)
	if calls != 1 {
		t.Fatalf("calls = %d, want one immediate observation", calls)
	}
}

// TestScheduledObservationPublishesOutageAndPostRestartRecovery joins the
// production scheduler callback, typed health evaluation, persisted alert
// transition state and Vrooli Events publisher contract. Its event sink is
// local and does not prove downstream Notification Hub delivery.
func TestScheduledObservationPublishesOutageAndPostRestartRecovery(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	sink, publisher := newEventSink(t)
	state := &memoryAlertState{alerts: map[string][]perfbudget.Alert{}}
	down := true
	newAlerter := func(at time.Time) *Alerter {
		alerter := NewAlerter(publisher, DefaultDetection(), nil).WithStateStore(state)
		alerter.now = func() time.Time { return at }
		return alerter
	}
	observeWith := func(alerter *Alerter) func(context.Context, string) (*healthv1.HealthObservation, error) {
		return func(ctx context.Context, id string) (*healthv1.HealthObservation, error) {
			status := "running"
			if down {
				status = "stopped"
			}
			observedAt := now
			if !down {
				observedAt = now.Add(time.Minute)
			}
			live := fixtureLiveState(observedAt, status)
			obs, report := observationFor(t, live, nil, observedAt)
			if obs.GetDeploymentId() != id {
				t.Fatalf("scheduler id=%q but observation deployment=%q", id, obs.GetDeploymentId())
			}
			alerter.Observe(ctx, obs, report)
			return obs, nil
		}
	}
	scheduler := Scheduler{
		Interval: time.Minute,
		Targets: func(context.Context) ([]string, error) {
			return []string{fixtureDeployment(domain.StatusDeployed).ID}, nil
		},
		Observe: observeWith(newAlerter(now)),
	}
	scheduler.RunOnce(context.Background())
	openFacts := sink.facts(t)
	if len(openFacts) != 1 || openFacts[0]["status"] != "open" || openFacts[0]["reason"] != "status_unhealthy" {
		t.Fatalf("scheduled outage event = %+v", openFacts)
	}

	// Replace the observer process to prove that a later scheduled healthy
	// observation resolves the persisted incident instead of losing it.
	down = false
	scheduler.Observe = observeWith(newAlerter(now.Add(time.Minute)))
	scheduler.RunOnce(context.Background())
	allFacts := sink.facts(t)
	if len(allFacts) != 2 || allFacts[1]["status"] != "resolved" || allFacts[1]["incident_id"] != allFacts[0]["incident_id"] {
		t.Fatalf("scheduled recovery events = %+v", allFacts)
	}
}
