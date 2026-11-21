package releases

import (
	"strings"
	"time"
)

// ReleaseHealth is a read-only operational projection. It derives standing
// from durable publication, verification, and recovery evidence; build
// completion alone cannot make a release healthy.
type ReleaseHealth struct {
	ReleaseID            string           `json:"release_id"`
	Status               string           `json:"status"`
	ObservedAt           time.Time        `json:"observed_at"`
	PublicationVerified  bool             `json:"publication_verified"`
	ClientUpdatesHealthy bool             `json:"client_updates_healthy"`
	RecoveryStanding     string           `json:"recovery_standing"`
	Alerts               []ReleaseAlert   `json:"alerts,omitempty"`
	KnownDurationsMillis map[string]int64 `json:"known_durations_millis,omitempty"`
	SupportedControls    []string         `json:"supported_controls,omitempty"`
	UnsupportedControls  []string         `json:"unsupported_controls,omitempty"`
}

type ReleaseAlert struct {
	Code       string `json:"code"`
	Severity   string `json:"severity"`
	Target     string `json:"target,omitempty"`
	Message    string `json:"message"`
	NextAction string `json:"next_action"`
}

// SigningExpiry is an owner observation used to warn before a published
// release can no longer be produced or repaired with a trusted signer.
type SigningExpiry struct {
	Platform  string
	ExpiresAt time.Time
}

func (s SigningExpiry) DaysToExpiry(now time.Time) int {
	return int(s.ExpiresAt.UTC().Sub(now.UTC()).Hours() / 24)
}

func AssessHealth(release *Release, now time.Time) ReleaseHealth {
	health := ReleaseHealth{ObservedAt: now.UTC(), SupportedControls: []string{"halt"}, UnsupportedControls: []string{"cohort_rollout"}, KnownDurationsMillis: map[string]int64{}}
	if release == nil {
		health.Status = "unknown"
		health.Alerts = []ReleaseAlert{{Code: "release_unavailable", Severity: "critical", Message: "The release record is unavailable.", NextAction: "Restore release ledger access before taking an effectful action."}}
		return health
	}
	health.ReleaseID, health.Status = release.ID, "unknown"
	if release.Status == StatusAmbiguous {
		health.Status = "attention"
		health.Alerts = append(health.Alerts, ReleaseAlert{Code: "ambiguous_effect", Severity: "critical", Target: release.ID, Message: "The release may have an external effect without a complete durable receipt.", NextAction: "Reconcile the owner before retrying or promoting another candidate."})
	}
	if release.Status == StatusFailed || release.Status == StatusVerifyFailed {
		health.Status = "attention"
		health.Alerts = append(health.Alerts, ReleaseAlert{Code: "failed_verification", Severity: "critical", Target: release.ID, Message: "Release verification or publication failed.", NextAction: "Inspect the failed stage and use an exact owner recovery action."})
	}
	if len(release.Platforms) > 0 {
		publishedTargets := make(map[string]bool, len(release.PublicationReceipts))
		for _, receipt := range release.PublicationReceipts {
			if !receipt.TrustedPublication() || !matchesReleasePublication(release, receipt) {
				continue
			}
			publishedTargets[receipt.TargetID] = true
		}
		allPublished := true
		for _, platform := range release.Platforms {
			if platform.Status != PlatformStatusPublished || !publishedTargets[platform.Platform] {
				allPublished = false
			}
			if platform.Status == PlatformStatusFailed || platform.Status == PlatformStatusVerifyFailed {
				health.Alerts = append(health.Alerts, ReleaseAlert{Code: "update_health_regression", Severity: "critical", Target: platform.Platform, Message: "A release target is not healthy.", NextAction: "Stop future offers and reconcile the affected target."})
			}
		}
		health.PublicationVerified = release.Status == StatusPublished && allPublished
	}
	if len(release.ClientUpdateReceipts) > 0 {
		health.ClientUpdatesHealthy = true
		seenTargets := make(map[string]bool, len(release.ClientUpdateReceipts))
		for _, receipt := range release.ClientUpdateReceipts {
			if !receipt.TrustedUpdate() || !matchesReleaseUpdate(release, receipt) {
				continue
			}
			seenTargets[receipt.TargetID] = true
		}
		for _, platform := range release.Platforms {
			if !seenTargets[platform.Platform] {
				health.ClientUpdatesHealthy = false
				health.Alerts = append(health.Alerts, ReleaseAlert{Code: "missing_client_update_receipt", Severity: "high", Target: platform.Platform, Message: "The installed target has no trusted receipt for the exact reviewed successor.", NextAction: "Collect the owner-produced relaunch receipt before expanding offers."})
			}
		}
	} else if len(release.VerificationEvidence) > 0 {
		health.ClientUpdatesHealthy = len(release.Platforms) > 0
		seenTargets := make(map[string]bool, len(release.VerificationEvidence))
		for _, evidence := range release.VerificationEvidence {
			validTarget := false
			for _, platform := range release.Platforms {
				if platform.Platform == evidence.Platform {
					validTarget = true
					break
				}
			}
			if !validTarget || seenTargets[evidence.Platform] || !evidence.Match || !evidence.SHA512Match {
				health.ClientUpdatesHealthy = false
				health.Alerts = append(health.Alerts, ReleaseAlert{Code: "update_health_regression", Severity: "critical", Target: evidence.Platform, Message: "A client verification receipt does not match the reviewed artifact.", NextAction: "Halt the affected channel and recover the installed target through its owner."})
				continue
			}
			seenTargets[evidence.Platform] = true
		}
		for _, platform := range release.Platforms {
			if !seenTargets[platform.Platform] {
				health.ClientUpdatesHealthy = false
				health.Alerts = append(health.Alerts, ReleaseAlert{Code: "missing_client_update_receipt", Severity: "high", Target: platform.Platform, Message: "The installed target has no trusted verification evidence for the exact reviewed successor.", NextAction: "Collect the owner-produced relaunch receipt before expanding offers."})
			}
		}
	}
	if len(release.RecoveryReceipts) > 0 {
		last := release.RecoveryReceipts[len(release.RecoveryReceipts)-1]
		health.RecoveryStanding = last.Outcome + ":" + last.Health
		switch last.Outcome {
		case "halted":
			health.Status = "halted"
		case "withdrawn":
			health.Status = "withdrawn"
		case "rolled_back", "forward_repaired":
			health.Status = "recovered"
		}
	}
	if release.PublishedAt != nil && health.PublicationVerified {
		health.KnownDurationsMillis["release_to_publication_verified"] = release.PublishedAt.Sub(release.CreatedAt).Milliseconds()
	}
	if len(release.RecoveryReceipts) > 0 {
		health.KnownDurationsMillis["release_to_recovery"] = release.RecoveryReceipts[len(release.RecoveryReceipts)-1].ObservedAt.Sub(release.CreatedAt).Milliseconds()
	}
	if health.Status == "unknown" && health.PublicationVerified && (len(release.VerificationEvidence) == 0 || health.ClientUpdatesHealthy) {
		health.Status = "healthy"
	}
	if !health.PublicationVerified && release.Status == StatusPublished {
		health.Alerts = append(health.Alerts, ReleaseAlert{Code: "missing_publication_receipt", Severity: "high", Target: release.ID, Message: "The release is marked published without complete target evidence.", NextAction: "Do not promote another cohort until public objects are independently verified."})
	}
	if release.Status == StatusPublished && len(release.VerificationEvidence) == 0 && len(release.ClientUpdateReceipts) == 0 {
		health.Status = "attention"
		health.Alerts = append(health.Alerts, ReleaseAlert{Code: "stale_required_evidence", Severity: "high", Target: release.ID, Message: "No client health receipt is attached to the published release.", NextAction: "Collect a fresh client verification receipt before expanding offers."})
	}
	return health
}

func matchesReleasePublication(release *Release, receipt PublicationReceipt) bool {
	if release.CandidateID != "" && receipt.CandidateID != release.CandidateID {
		return false
	}
	if release.DestinationRevisionID != "" && receipt.DestinationRevisionID != release.DestinationRevisionID {
		return false
	}
	// Release.ArtifactDigest identifies the complete candidate manifest. A
	// publication receipt carries the per-target bytes digest, so those values
	// are intentionally checked at the owner verification boundary rather than
	// compared as if they were the same hash domain.
	return true
}

func matchesReleaseUpdate(release *Release, receipt ClientUpdateReceipt) bool {
	if release.CandidateID != "" && receipt.CandidateID != release.CandidateID {
		return false
	}
	if strings.TrimSpace(release.CandidateID) == "" && strings.TrimSpace(release.DestinationRevisionID) == "" {
		return true
	}
	for _, publication := range release.PublicationReceipts {
		if !publication.TrustedPublication() || !matchesReleasePublication(release, publication) || publication.TargetID != receipt.TargetID {
			continue
		}
		return equivalentDigest(publication.ArtifactDigest, receipt.SuccessorDigest)
	}
	return false
}

func equivalentDigest(left, right string) bool {
	leftAlgorithm, leftValue := splitDigest(left)
	rightAlgorithm, rightValue := splitDigest(right)
	if leftValue == "" || rightValue == "" || leftValue != rightValue {
		return false
	}
	return leftAlgorithm == "" || rightAlgorithm == "" || leftAlgorithm == rightAlgorithm
}

func splitDigest(value string) (string, string) {
	value = strings.ToLower(strings.TrimSpace(value))
	parts := strings.SplitN(value, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", value
}

func addDuration(measurements map[string]int64, key string, start, end time.Time) {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return
	}
	measurements[key] = end.Sub(start).Milliseconds()
}
