package jobstate

import jobsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs"

// DisplayName gives CLI output a concise name for a durable job state.
func DisplayName(state jobsv1.JobState) string {
	switch state {
	case jobsv1.JobState_JOB_STATE_SUCCEEDED:
		return "succeeded"
	case jobsv1.JobState_JOB_STATE_FAILED:
		return "failed"
	case jobsv1.JobState_JOB_STATE_CANCELED:
		return "canceled"
	case jobsv1.JobState_JOB_STATE_RUNNING:
		return "running"
	case jobsv1.JobState_JOB_STATE_QUEUED:
		return "queued"
	default:
		return "unknown"
	}
}
