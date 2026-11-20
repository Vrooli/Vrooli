package inference

import (
	"ai-gateway/internal/providers"
	"ai-gateway/internal/rolecatalog"
)

// resolveTemperature applies the role/caller precedence for one candidate.
//
//	caller set + role overridable   -> caller value, but only if the candidate
//	                                   declares "honored"
//	caller set + role not overridable -> ErrRoleForbidsSampling
//	role declares                   -> role value, omitted only when the
//	                                   candidate declares "rejected"
//	neither                         -> nil, the resource policy default applies
//
// The two paths deliberately draw the line in different places.
//
// A caller-supplied control is a promise, so only "honored" keeps it. "ignored"
// fails the promise exactly as thoroughly as "rejected" does — the difference is
// only whether the failure is visible — so both refuse the candidate.
//
// A role-declared default is a preference, and the question there is not "will
// this be honoured" but "will sending it break the call". Only "rejected"
// breaks it. So "ignored" and "unknown" are still sent: harmless where the
// provider discards it, and correct where an undeclared provider turns out to
// honour it after all. That is what treating "unknown" as best-effort means —
// try it, rather than silently drop the role's stated intent. It also keeps a
// deterministic role deterministic against a resource that has not yet
// published its declaration.
func resolveTemperature(request ProviderRequest, role rolecatalog.InferenceRole, support providers.SamplingSupport) (*float64, error) {
	if request.Temperature != nil {
		if role.Sampling == nil || !role.Sampling.Overridable {
			return nil, ErrRoleForbidsSampling
		}
		if support != providers.SamplingHonored {
			return nil, ErrUnsupportedSampling
		}
		value := *request.Temperature
		return &value, nil
	}
	if role.Sampling == nil || role.Sampling.Temperature == nil {
		return nil, nil
	}
	if support == providers.SamplingRejected {
		return nil, nil
	}
	value := *role.Sampling.Temperature
	return &value, nil
}

// resolveOutputCap applies the caller/role precedence for the output budget and
// reports where the answer came from. Reporting the source matters as much as
// the number: "the gateway imposed nothing" and "the gateway imposed a cap you
// cannot see" are different facts, and only the first is true here when both
// the caller and the role are silent.
func resolveOutputCap(request ProviderRequest, resolved providers.ResolvedRole) (int32, OutputCapSource) {
	switch {
	case request.MaxOutputTokens > 0:
		return request.MaxOutputTokens, OutputCapRequest
	case resolved.MaxOutputTokens > 0:
		// The resource applies its own role cap when the gateway sends none, so
		// this is a report of what will happen rather than a value to transmit.
		return resolved.MaxOutputTokens, OutputCapRolePolicy
	default:
		return 0, OutputCapNoneImposed
	}
}
