package readiness

import "fmt"

const (
	ReasonVerdictMissing = "readiness_verdict_missing"
	ReasonVerdictStale   = "readiness_verdict_stale"
	ReasonGoalOpen       = "readiness_goal_open"
	ReasonWaiverInvalid  = "readiness_waiver_invalid"
)

type ReleaseRecord struct {
	VerdictPresent   bool
	ApprovedAtCommit string
	GoalRef          string
	GoalClosed       bool
	Waiver           *Waiver
}

// CheckReleaseReadiness applies the release-side readiness contract. A waiver
// is deliberately not accepted here: waivers are separate, commit-scoped
// records and must be validated before being supplied as a gate input.
func CheckReleaseReadiness(commit string, record *ReleaseRecord) error {
	if record != nil && record.Waiver != nil {
		if record.Waiver.Reason == "" || record.Waiver.Actor == "" || record.Waiver.Commit != commit || record.Waiver.At.IsZero() {
			return fmt.Errorf("%s: waiver must name reason, actor, and the deployed commit", ReasonWaiverInvalid)
		}
		return nil
	}
	if record == nil || !record.VerdictPresent {
		return fmt.Errorf("%s", ReasonVerdictMissing)
	}
	if record.ApprovedAtCommit != commit {
		return fmt.Errorf("%s: approved_at_commit=%s deployed_commit=%s", ReasonVerdictStale, record.ApprovedAtCommit, commit)
	}
	if !record.GoalClosed {
		return fmt.Errorf("%s: goal=%s", ReasonGoalOpen, record.GoalRef)
	}
	return nil
}
