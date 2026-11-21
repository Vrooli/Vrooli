package policy

import (
	"fmt"
	"strings"
)

// Sampling bounds are the safe envelope every declared value is checked or
// clamped against. A misconfigured policy fails load or is clamped on read; it
// never pushes a runtime out of range.
const (
	MinTemperature = 0.0
	MaxTemperature = 2.0
	MinTopP        = 0.0
	MaxTopP        = 1.0
	MinTopK        = 0
	MaxTopK        = 1000
	MinMaxTokens   = 1
)

var (
	samplingSupportStates = map[SamplingSupportState]struct{}{
		SamplingHonored: {}, SamplingIgnored: {}, SamplingRejected: {}, SamplingUnknown: {},
	}
	samplingParameters = map[string]struct{}{
		"temperature": {}, "top_p": {}, "top_k": {}, "seed": {},
	}
)

// Resolve clamps the declared sampling into the safe envelope. A nil receiver
// (role declared no sampling_defaults) resolves to an all-absent Sampling so
// callers omit the keys and fall back to the model's own defaults.
func (s *SamplingDefaults) Resolve() Sampling {
	var out Sampling
	if s == nil {
		return out
	}
	if s.Temperature != nil {
		out.Temperature = clampFloat(*s.Temperature, MinTemperature, MaxTemperature)
		out.HasTemperature = true
	}
	if s.TopP != nil {
		out.TopP = clampFloat(*s.TopP, MinTopP, MaxTopP)
		out.HasTopP = true
	}
	if s.TopK != nil {
		out.TopK = clampInt(*s.TopK, MinTopK, MaxTopK)
		out.HasTopK = true
	}
	return out
}

// ResolveGenerate resolves the three-way precedence for the chat-request levers
// the role owns: an explicit flag wins, an absent flag falls through to the
// role's declared request_defaults, and an absent default leaves the parameter
// off the wire so the upstream provider's own default applies. temperatureFlag
// carries the -1 "unset" sentinel and maxTokensFlag carries 0.
func (d *RequestDefaults) ResolveGenerate(temperatureFlag float64, maxTokensFlag int) (*float64, int) {
	var temperature *float64
	switch {
	case temperatureFlag >= 0:
		value := temperatureFlag
		temperature = &value
	case d != nil && d.Temperature != nil:
		value := *d.Temperature
		temperature = &value
	}
	maxTokens := maxTokensFlag
	if maxTokens <= 0 && d != nil && d.MaxTokens != nil {
		maxTokens = *d.MaxTokens
	}
	return temperature, maxTokens
}

func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func validateSamplingSupport(errs *[]error, path string, support map[string]SamplingSupportState) {
	for parameter, state := range support {
		if _, ok := samplingParameters[strings.TrimSpace(parameter)]; !ok {
			*errs = append(*errs, fmt.Errorf(path+" key %q is not a known sampling parameter", parameter))
		}
		if _, ok := samplingSupportStates[state]; !ok {
			*errs = append(*errs, fmt.Errorf(path+"."+parameter+" %q is not an allowed support state", state))
		}
	}
}

func copySamplingSupport(in map[string]SamplingSupportState) map[string]SamplingSupportState {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]SamplingSupportState, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
