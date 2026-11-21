package evaluation

// OperationalReport deliberately has a separate type from controlled reports:
// test observations cannot establish a real-use benefit claim.
type OperationalReport struct {
	Status            string
	Reason            string
	EligibleAttempts  int
	VerifiedSuccesses int
	Failed            int
}

func MeasureOperational(observations []Observation) OperationalReport {
	out := OperationalReport{Status: "unknown", Reason: "no_eligible_operator_observations"}
	for _, observation := range observations {
		if observation.Provenance != "operator" {
			continue
		}
		out.EligibleAttempts++
		if observation.Support && observation.Coverage {
			out.VerifiedSuccesses++
		} else {
			out.Failed++
		}
	}
	if out.EligibleAttempts == 0 {
		return out
	}
	out.Status = "observed"
	out.Reason = "operator_observations_available"
	return out
}
