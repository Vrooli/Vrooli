package eligibility

import "slices"

const EvaluatorVersion = "eligibility-v2-methods"

const (
	PreparationCooking    = "cooking"
	PreparationAssembly   = "assembly"
	PreparationReadyToEat = "ready_to_eat"
)

func Evaluate(profile Profile, candidate Candidate) Decision {
	out := Decision{ProfileRevision: profile.Revision, RecipeRevision: candidate.Revision, EvaluatorVersion: EvaluatorVersion}
	globalReasons := make([]Reason, 0)
	globalStatus := Eligible
	for _, excluded := range profile.ExcludedGroups {
		if slices.Contains(candidate.Groups, excluded) {
			globalStatus = Ineligible
			globalReasons = append(globalReasons, Reason{Code: "excluded_group", Rule: excluded, Reference: "recipe.groups", Message: "recipe contains an excluded food group"})
		}
	}
	for _, allergen := range profile.Allergies {
		state, ok := candidate.AllergenEvidence[allergen]
		if !ok || state == Unknown || state == Conflicting || state == Precautionary {
			if globalStatus == Eligible {
				globalStatus = NeedsInformation
			}
			globalReasons = append(globalReasons, Reason{Code: "allergen_evidence_unknown", Rule: allergen, Reference: "recipe.allergens." + allergen, Message: "active allergen restriction lacks decisive evidence"})
		} else if state == DeclaredPresent {
			globalStatus = Ineligible
			globalReasons = append(globalReasons, Reason{Code: "allergen_present", Rule: allergen, Reference: "recipe.allergens." + allergen, Message: "recipe declares an active allergen"})
		}
	}

	if len(candidate.Methods) == 0 {
		noMethod := Reason{Code: "method_missing", Rule: "preparation method", Reference: "recipe.methods", Message: "recipe needs at least one explicit preparation method"}
		if globalStatus != Ineligible {
			globalStatus = NeedsInformation
		}
		globalReasons = append(globalReasons, noMethod)
		out.Status = globalStatus
		out.Reasons = globalReasons
		return out
	}

	out.Methods = make([]MethodDecision, 0, len(candidate.Methods))
	for _, method := range candidate.Methods {
		assessment := assessMethod(profile, method)
		for _, reason := range globalReasons {
			assessment.Reasons = append(assessment.Reasons, withMethod(reason, method.ID))
			assessment.Status = combine(assessment.Status, globalStatus)
		}
		out.Methods = append(out.Methods, assessment)
		out.Reasons = append(out.Reasons, assessment.Reasons...)
		if assessment.Status == Eligible {
			out.ViableMethodIDs = append(out.ViableMethodIDs, method.ID)
		}
	}

	switch {
	case len(out.ViableMethodIDs) > 0:
		out.Status = Eligible
	case slices.ContainsFunc(out.Methods, func(method MethodDecision) bool { return method.Status == NeedsInformation }):
		out.Status = NeedsInformation
	default:
		out.Status = Ineligible
	}
	for i := range out.Reasons {
		out.Reasons[i].ViableMethodIDs = append([]string(nil), out.ViableMethodIDs...)
	}
	return out
}

func assessMethod(profile Profile, method Method) MethodDecision {
	out := MethodDecision{MethodID: method.ID, MethodName: method.Name, Status: Eligible}
	add := func(code, rule, reference, message string, status Status) {
		out.Status = combine(out.Status, status)
		out.Reasons = append(out.Reasons, Reason{Code: code, Rule: rule, Reference: reference, Message: message, MethodID: method.ID})
	}

	switch method.PreparationMode {
	case PreparationCooking:
		if len(profile.Appliances) == 0 {
			add("appliance_free_method_not_supported", method.PreparationMode, "recipe.methods."+method.ID+".preparation_mode", "without a selected appliance capability, only explicit assembly or ready-to-eat methods can qualify", Ineligible)
		}
		if len(method.RequiredAppliances)+len(method.RequiredDeviceFunctions) == 0 {
			add("method_requirements_missing", "equipment", "recipe.methods."+method.ID, "cooking method needs explicit appliance or device-function requirements", NeedsInformation)
		}
	case PreparationAssembly, PreparationReadyToEat:
	case "":
		add("preparation_mode_missing", "preparation mode", "recipe.methods."+method.ID+".preparation_mode", "method kind must be explicit before its equipment requirements can be evaluated", NeedsInformation)
	default:
		add("preparation_mode_unknown", method.PreparationMode, "recipe.methods."+method.ID+".preparation_mode", "preparation mode is not supported", Ineligible)
	}

	check := func(values, selected []string, category, selectionPath string) {
		for _, required := range values {
			if required == "" {
				add("method_requirement_invalid", category, "recipe.methods."+method.ID+"."+category, "method requirements must use explicit non-empty IDs", NeedsInformation)
				continue
			}
			if !slices.Contains(selected, required) {
				add("method_requirement_unavailable", required, selectionPath, "this method requires a kitchen capability that is not selected", Ineligible)
			}
		}
	}
	check(method.RequiredAppliances, profile.Appliances, "appliance", "profile.appliances")
	check(method.RequiredCookware, profile.Cookware, "cookware", "profile.cookware")
	check(method.RequiredTools, profile.Tools, "tool", "profile.tools")
	check(method.RequiredDeviceFunctions, profile.DeviceFunctions, "device function", "profile.kitchen_devices.capabilities")
	return out
}

func combine(current, next Status) Status {
	if current == Ineligible || next == Ineligible {
		return Ineligible
	}
	if current == NeedsInformation || next == NeedsInformation {
		return NeedsInformation
	}
	return Eligible
}

func withMethod(reason Reason, methodID string) Reason {
	reason.MethodID = methodID
	return reason
}
