package eligibility

import "slices"

const EvaluatorVersion = "eligibility-v1"

func Evaluate(profile Profile, candidate Candidate) Decision {
	out := Decision{Status: Eligible, ProfileRevision: profile.Revision, RecipeRevision: candidate.Revision, EvaluatorVersion: EvaluatorVersion}
	for _, excluded := range profile.ExcludedGroups {
		if slices.Contains(candidate.Groups, excluded) {
			out.Status = Ineligible
			out.Reasons = append(out.Reasons, Reason{Code: "excluded_group", Rule: excluded, Reference: "recipe.groups", Message: "recipe contains an excluded food group", ViableMethodIDs: append([]string(nil), candidate.MethodIDs...)})
		}
	}
	for _, allergen := range profile.Allergies {
		state, ok := candidate.AllergenEvidence[allergen]
		if !ok || state == Unknown || state == Conflicting || state == Precautionary {
			if out.Status == Eligible {
				out.Status = NeedsInformation
			}
			out.Reasons = append(out.Reasons, Reason{Code: "allergen_evidence_unknown", Rule: allergen, Reference: "recipe.allergens." + allergen, Message: "active allergen restriction lacks decisive evidence", ViableMethodIDs: append([]string(nil), candidate.MethodIDs...)})
		} else if state == DeclaredPresent {
			out.Status = Ineligible
			out.Reasons = append(out.Reasons, Reason{Code: "allergen_present", Rule: allergen, Reference: "recipe.allergens." + allergen, Message: "recipe declares an active allergen", ViableMethodIDs: append([]string(nil), candidate.MethodIDs...)})
		}
	}
	for _, appliance := range candidate.RequiredAppliances {
		if !slices.Contains(profile.Appliances, appliance) {
			out.Status = Ineligible
			out.Reasons = append(out.Reasons, Reason{Code: "missing_appliance", Rule: appliance, Reference: "recipe.method.appliances", Message: "no selected kitchen capability can execute this method", ViableMethodIDs: append([]string(nil), candidate.MethodIDs...)})
		}
	}
	return out
}
