package validation

// This file holds the reduced-motion accessibility rule. A scenario that ships
// transitions/animations but declares no prefers-reduced-motion accommodation
// forces the full motion onto users who have asked the OS to reduce it
// (vestibular sensitivity). The rule fires (info) ONLY when motion is actually
// shipped in the scenario's own stylesheets AND no reduce block exists — so it
// never nags a static UI. Detection is scoped to the app's own CSS (ui/src,
// ui/public), never node_modules, so a dependency's animations do not trip it.

func ruleReducedMotionSupport(c *scanContext) (Finding, bool) {
	m := c.appCSSMatches([]string{"transition", "animation", "@keyframes", "prefers-reduced-motion"})
	hasMotion := m["transition"] || m["animation"] || m["@keyframes"]
	if !hasMotion || m["prefers-reduced-motion"] {
		return Finding{}, false
	}
	return Finding{
		Severity:               SeverityInfo,
		Title:                  "No reduced-motion accommodation for shipped animations",
		Description:            "The UI ships transitions/animations but declares no @media (prefers-reduced-motion: reduce) block.",
		FilePath:               designSystemCSSRel,
		WhyItMatters:           "Users who set 'reduce motion' (often for vestibular sensitivity) still receive the full motion, which can cause discomfort.",
		RecommendedRemediation: "Add an @media (prefers-reduced-motion: reduce) block that neutralizes non-essential animation and transition.",
		Evidence:               map[string]any{"has_motion": true},
	}, true
}
