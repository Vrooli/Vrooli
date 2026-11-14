package usageattribution

// Reach splits one window's observed increase by who a usage mode can change:
// operator sessions (no profile), runs launched under a profile, and the
// unattributed remainder (off-disk or unexplained use). Points are estimates.
type Reach struct {
	Observed, Operator, Profile, Unattributed float64
}

// ReachOf classifies a window result with the profile grouping, so the split
// matches `usage burners --by profile`.
func ReachOf(result Result, runs map[string]Run) Reach {
	reach := Reach{Observed: result.Observed, Unattributed: result.Unattributed}
	for _, burner := range Group(result, ByProfile, runs, "") {
		if burner.Key == operatorProfile {
			reach.Operator += burner.Points
		} else {
			reach.Profile += burner.Points
		}
	}
	return reach
}
