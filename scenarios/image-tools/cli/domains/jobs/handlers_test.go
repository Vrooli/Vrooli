package jobs

import (
	"testing"

	jobsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs"
)

func TestJobQueueLabel(t *testing.T) {
	tests := []struct {
		name       string
		queue      jobsv1.JobQueue
		legacyLane jobsv1.JobLane
		want       string
	}{
		{name: "local serialized queue", queue: jobsv1.JobQueue_JOB_QUEUE_LOCAL_SERIAL, legacyLane: jobsv1.JobLane_JOB_LANE_GPU, want: "local-serial"},
		{name: "legacy gpu value means local serialized queue", legacyLane: jobsv1.JobLane_JOB_LANE_GPU, want: "local-serial"},
		{name: "cpu worker pool", queue: jobsv1.JobQueue_JOB_QUEUE_CPU_POOL, legacyLane: jobsv1.JobLane_JOB_LANE_CPU, want: "cpu-pool"},
		{name: "network worker pool", queue: jobsv1.JobQueue_JOB_QUEUE_NETWORK_POOL, legacyLane: jobsv1.JobLane_JOB_LANE_NETWORK, want: "network-pool"},
		{name: "unknown", want: "unspecified"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := jobQueueLabel(tt.queue, tt.legacyLane); got != tt.want {
				t.Fatalf("jobQueueLabel(%v, %v) = %q, want %q", tt.queue, tt.legacyLane, got, tt.want)
			}
		})
	}
}
