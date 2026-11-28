package jobstate

import (
	"testing"

	jobsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs"
)

func TestDisplayName(t *testing.T) {
	cases := map[jobsv1.JobState]string{
		jobsv1.JobState_JOB_STATE_SUCCEEDED: "succeeded",
		jobsv1.JobState_JOB_STATE_FAILED:    "failed",
		jobsv1.JobState_JOB_STATE_CANCELED:  "canceled",
		jobsv1.JobState_JOB_STATE_RUNNING:   "running",
		jobsv1.JobState_JOB_STATE_QUEUED:    "queued",
		jobsv1.JobState(999):                "unknown",
	}
	for state, want := range cases {
		if got := DisplayName(state); got != want {
			t.Errorf("DisplayName(%s) = %q, want %q", state, got, want)
		}
	}
}
