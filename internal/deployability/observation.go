package deployability

import "fmt"

// PlatformObservation is an observed fact supplied by a validation provider.
// It is intentionally separate from manifest declarations: owners declare
// support, while a provider reports what the target actually made available.
type PlatformObservation struct {
	OS        HostOS
	Available bool
	Reason    string
}

// ValidateObservedPlatform resolves a target for the observed host and turns
// a contradictory declaration into an explicit ineligible verdict. This is
// the seam used by portability validation providers; it never guesses a
// platform from a resource or scenario name.
func ValidateObservedPlatform(input ResolutionInput, observation PlatformObservation) Resolution {
	input.OS = observation.OS
	result := Resolve(input)
	if observation.Available {
		return result
	}
	result.Verdict = VerdictIneligible
	result.Reasons = append(result.Reasons, Reason{
		Code:    "observed_platform_unavailable",
		Message: fmt.Sprintf("observed host %s is unavailable for this target: %s", observation.OS, observation.Reason),
	})
	return result
}
