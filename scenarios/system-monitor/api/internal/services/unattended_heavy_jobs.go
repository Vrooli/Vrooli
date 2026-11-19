package services

import (
	"context"
	"fmt"
	"time"
)

// RuleUnattendedHeavyJob is deliberately warning-only: system-monitor reports
// durable work with no live waiter, but never cancels it or dispatches an agent.
const RuleUnattendedHeavyJob = "unattended_heavy_job"

const (
	unattendedJobMinimumAge   = 15 * time.Minute
	unattendedJobMaximumStale = time.Minute
)

// HeavyJob is an owner-provided durable job observation. WaiterCount must count
// live server-side wait edges, not browser visibility or polling requests.
type HeavyJob struct {
	OwnerID     string    `json:"owner_id"`
	JobID       string    `json:"job_id"`
	Operation   string    `json:"operation"`
	State       string    `json:"state"`
	StartedAt   time.Time `json:"started_at"`
	WaiterCount int       `json:"waiter_count"`
	Heavy       bool      `json:"heavy"`
}

// HeavyJobSnapshot is complete only when the owner has reported every active
// heavy job. Incomplete snapshots cannot clear or open findings.
type HeavyJobSnapshot struct {
	ObservedAt time.Time  `json:"observed_at"`
	Complete   bool       `json:"complete"`
	Jobs       []HeavyJob `json:"jobs"`
}

// HeavyJobSource is implemented by the job-owning scenario adapter. Keeping
// waiter truth at the owner avoids guessing from process age or client polling.
type HeavyJobSource interface {
	GetHeavyJobs(context.Context) (HeavyJobSnapshot, error)
}

type unattendedHeavyJobRule struct{ source HeavyJobSource }

func NewUnattendedHeavyJobRule(source HeavyJobSource) AlertRule {
	return unattendedHeavyJobRule{source: source}
}

func (unattendedHeavyJobRule) ID() string { return RuleUnattendedHeavyJob }

func (r unattendedHeavyJobRule) Evaluate(ctx context.Context, snap AlertSnapshot) RuleEvaluation {
	if r.source == nil {
		return unsupportedRule("owner job telemetry is not configured")
	}
	observed, err := r.source.GetHeavyJobs(ctx)
	if err != nil {
		return failedRule("read owner heavy-job telemetry: " + err.Error())
	}
	if !observed.Complete || observed.ObservedAt.IsZero() || observed.ObservedAt.After(snap.Now) || snap.Now.Sub(observed.ObservedAt) > unattendedJobMaximumStale {
		return unsupportedRule("complete, fresh owner job telemetry is unavailable")
	}
	var conditions []AlertCondition
	for _, job := range observed.Jobs {
		if !job.Heavy || job.OwnerID == "" || job.JobID == "" || job.Operation == "" || job.State != "running" || job.WaiterCount != 0 || job.StartedAt.IsZero() || job.StartedAt.After(snap.Now) || snap.Now.Sub(job.StartedAt) < unattendedJobMinimumAge {
			continue
		}
		age := snap.Now.Sub(job.StartedAt)
		conditions = append(conditions, AlertCondition{
			Key: job.OwnerID + ":" + job.JobID, Severity: SeverityWarning,
			Message:    fmt.Sprintf("Heavy %s job %s has run for %s with no live waiter.", job.OwnerID, job.Operation, age.Round(time.Minute)),
			MetricName: RuleUnattendedHeavyJob, Value: age.Minutes(), Threshold: unattendedJobMinimumAge.Minutes(),
			Details: map[string]interface{}{
				"owner_id": job.OwnerID, "job_id": job.JobID, "operation": job.Operation,
				"state": job.State, "waiter_count": job.WaiterCount,
				"started_at":  job.StartedAt.UTC().Format(time.RFC3339Nano),
				"observed_at": observed.ObservedAt.UTC().Format(time.RFC3339Nano),
				"age_seconds": age.Seconds(), "report_only": true,
			},
		})
	}
	if len(conditions) == 0 {
		return clearRule()
	}
	return activeRule(conditions...)
}
