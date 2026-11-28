package modelcatalog

// DefaultFootprintBudgetMB is the declared resource budget a promoted default
// must fit, per capability. It exists because "small by default" is a product
// decision, not a preference: a default that quietly grows to a multi-gigabyte
// model makes the whole machine less usable for everything else.
//
// The budget is compared against size_mb_approx, the same declared fact the
// docs and the UI show. Raising a budget is an explicit, reviewable seed change
// (and a docs diff), not an accidental side effect of promoting a new model.
//
// Summarize is bounded at 4 GB because it runs through resource-ollama, whose
// installed tags include quantization metadata and runtime overhead on top of
// the published weight size; the STT/TTS/speaker budgets are for weights only.
// The 4B quality alternative measures 3.39 GB on disk; 9B measures 6.59 GB
// and must not fit.
var DefaultFootprintBudgetMB = map[Capability]int{
	CapabilitySTT:       600,
	CapabilityTTS:       400,
	CapabilitySummarize: 4000,
	CapabilitySpeaker:   100,
}

// FootprintBudget returns the declared budget for a capability, or 0 when no
// budget is defined.
func FootprintBudget(capability Capability) int {
	return DefaultFootprintBudgetMB[capability]
}

// UsesLocalWeights reports whether a runtime loads weights on this host. BYOK,
// gateway, and browser runtimes do not, so their declared size (often 0) is not
// a local footprint.
func UsesLocalWeights(runtime string) bool {
	switch runtime {
	case "byok", "gateway", "browser":
		return false
	default:
		return true
	}
}

// DeclaredLocalFootprintMB sums the declared size of every available record
// that runs locally. It is a catalogue total, not an install size: resources
// install their eager artifact and leave on-demand tiers unpulled.
func (c *Catalog) DeclaredLocalFootprintMB() float64 {
	total := 0.0
	for _, m := range c.All() {
		if m.Readiness != ReadinessAvailable || !UsesLocalWeights(m.Runtime) {
			continue
		}
		total += m.SizeMBApprox
	}
	return total
}

// PromotedDefaultFootprintMB sums the declared size of the resolved default
// for each capability that loads local weights. It is the weight-disk part of
// what a fresh install of the referenced resources needs, and it must stay
// within the sum of the capability budgets.
func (c *Catalog) PromotedDefaultFootprintMB() float64 {
	total := 0.0
	for _, capability := range DefaultCapabilities {
		def, err := c.DefaultFor(capability, "", Platform{})
		if err != nil || !UsesLocalWeights(def.Runtime) {
			continue
		}
		total += def.SizeMBApprox
	}
	return total
}
