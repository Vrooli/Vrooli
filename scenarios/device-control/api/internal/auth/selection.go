package auth

// Why UnlockCandidate found no usable profile.
const (
	NoProfile      = "no_profile"
	NeedsReprovide = "needs_reprovision"
)

// unsafeToRetry lists last outcomes after which retrying the stored credential
// could burn attempts on a phone's lockout counter, or cannot succeed.
var unsafeToRetry = map[string]bool{
	OutcomeWrongCredential: true,
	OutcomeVerifiedFailed:  true,
	OutcomeUnconfigured:    true,
	OutcomeHumanRequired:   true,
	OutcomeProviderAbsent:  true,
}

// UnlockCandidate chooses, without being told, the active profile a one-call
// verb may use to unlock deviceID: the most recently updated active profile
// whose last outcome does not say the stored credential is wrong or missing.
// When only such profiles exist it returns the newest with reason
// NeedsReprovide so the caller can name the repair; with none, NoProfile.
func UnlockCandidate(profiles []Profile, deviceID string) (Profile, string) {
	var best, blocked *Profile
	for i := range profiles {
		p := &profiles[i]
		if p.DeviceID != deviceID || p.Status != "active" {
			continue
		}
		if unsafeToRetry[p.LastOutcome] {
			if blocked == nil || p.UpdatedAt.After(blocked.UpdatedAt) {
				blocked = p
			}
			continue
		}
		if best == nil || p.UpdatedAt.After(best.UpdatedAt) {
			best = p
		}
	}
	switch {
	case best != nil:
		return *best, ""
	case blocked != nil:
		return *blocked, NeedsReprovide
	}
	return Profile{}, NoProfile
}
