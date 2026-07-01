package validation

// This file is the single rule/fixer registry. Both the active rule set
// (ScanScenario) and the deterministic fixer set (BuildFixCandidates) derive
// from it, and a finding's AutofixAvailable flag is computed by asking the
// registered fixer (preview) whether it can actually produce a candidate. That
// makes "advertised == implemented" true by construction: a rule cannot claim
// autofix_available=true unless its fixer yields a candidate for that scenario
// right now. The invariant is additionally guarded by a table test.
//
// specs carries only rule identity + surfaces + eval; the fixers live in a
// separate fixerByID map. Keeping them apart matters: a fixer's body reaches
// back into specs (via ruleFiresIsolated), so storing the fixer values inside
// the specs literal would form a package-level initialization cycle.

// fixerFunc is a deterministic remediation for one rule. It reads the scenario
// tree fresh from disk (never the scan cache, so a chain of applies sees prior
// writes) and, when apply is true, performs the edit and returns Applied=true.
// It returns ok=false when there is nothing it can deterministically fix for
// this scenario (e.g. the self-contained inputs are absent).
type fixerFunc func(root string, apply bool) (Candidate, bool, error)

// ruleSpec is one registry entry: rule identity, the surfaces it requires, and
// its evaluation function.
type ruleSpec struct {
	id       string
	surfaces []surface
	eval     func(c *scanContext) (Finding, bool)
}

// specs is the ordered rule registry. Order is the finding emission order.
var specs = []ruleSpec{
	// Phase 0 — identity + visual system + assets + applied (the original seven).
	{id: "has-display-name", eval: ruleHasDisplayName},
	{id: "has-color-system", surfaces: []surface{surfaceUI}, eval: ruleHasColorSystem},
	{id: "has-typography", surfaces: []surface{surfaceUI}, eval: ruleHasTypography},
	{id: "has-logo", surfaces: []surface{surfaceUI}, eval: ruleHasLogo},
	{id: "has-favicon", surfaces: []surface{surfaceUI}, eval: ruleHasFavicon},
	{id: "wcag-aa-contrast", surfaces: []surface{surfaceUI}, eval: ruleWCAGContrast},
	// brand-markers-applied is brand-projection: a deterministic fixer needs an
	// assigned brand (palette/markers via the apply domain), which the served
	// validation path has no input for. With no brand assigned it is honestly
	// guidance-only (no fixer ⇒ autofix_available=false), fixing the prior lie.
	{id: "brand-markers-applied", surfaces: []surface{surfaceUI}, eval: ruleBrandMarkersApplied},

	// Phase 1 — accessibility / contrast depth.
	{id: "dark-mode-contrast", surfaces: []surface{surfaceUI}, eval: ruleDarkModeContrast},
	{id: "color-scheme-declared", surfaces: []surface{surfaceUI}, eval: ruleColorSchemeDeclared},
	{id: "reduced-motion-support", surfaces: []surface{surfaceUI}, eval: ruleReducedMotionSupport},

	// Phase 2 — cross-surface consistency + template residue.
	{id: "name-consistency", surfaces: []surface{surfaceUI}, eval: ruleNameConsistency},
	{id: "theme-color-consistency", surfaces: []surface{surfaceUI}, eval: ruleThemeColorConsistency},
	{id: "no-template-residue", surfaces: []surface{surfaceUI}, eval: ruleNoTemplateResidue},

	// Phase 3 — brand asset quality (detect-only; image re-encode is non-deterministic here).
	{id: "apple-touch-icon", surfaces: []surface{surfaceUI}, eval: ruleAppleTouchIconPresent},
	{id: "asset-validity", surfaces: []surface{surfaceUI}, eval: ruleAssetValidity},
	{id: "referenced-assets-exist", surfaces: []surface{surfaceUI}, eval: ruleReferencedAssetsExist},
	{id: "svg-asset-safety", surfaces: []surface{surfaceUI}, eval: ruleSVGAssetSafety},
	{id: "custom-font-loaded", surfaces: []surface{surfaceUI}, eval: ruleCustomFontLoaded},

	// Phase 4 — PWA / mobile theming.
	{id: "theme-color-present", surfaces: []surface{surfaceUI}, eval: ruleThemeColorPresent},
	{id: "standalone-capable", surfaces: []surface{surfaceUI}, eval: ruleStandaloneCapable},
	{id: "ios-statusbar-safe-area", surfaces: []surface{surfaceUI}, eval: ruleIOSStatusBarSafeArea},
	{id: "manifest-completeness", surfaces: []surface{surfaceUI}, eval: ruleManifestCompleteness},

	// Phase 5 — social / link-preview metadata.
	{id: "open-graph", surfaces: []surface{surfaceUI}, eval: ruleOpenGraph},
	{id: "twitter-card", surfaces: []surface{surfaceUI}, eval: ruleTwitterCard},
	{id: "social-preview-image", surfaces: []surface{surfaceUI}, eval: ruleSocialPreviewImage},

	// Phase 6 — CLI / API branding (detect-only; surface-conditional).
	{id: "cli-branding", surfaces: []surface{surfaceCLI}, eval: ruleCLIBranding},
	{id: "api-branding", eval: ruleAPIBranding},

	// Phase 7 — public-asset edge convention (assets served under /public/ so an
	// Access bypass can serve them to anonymous fetchers; see PUBLIC_ASSETS.md).
	{id: "public-asset-convention", surfaces: []surface{surfaceUI}, eval: rulePublicAssetConvention},
}

// fixerByID maps a rule id to its deterministic fixer. Only rules listed here
// can ever advertise autofix_available=true. Every key MUST also exist in specs
// (asserted by the invariant test).
var fixerByID = map[string]fixerFunc{
	"has-color-system":        fixColorSystem,
	"has-favicon":             fixFavicon,
	"color-scheme-declared":   fixColorScheme,
	"name-consistency":        fixNameConsistency,
	"theme-color-consistency": fixThemeColorConsistency,
	"no-template-residue":     fixTemplateResidue,
	"theme-color-present":     fixThemeColorPresent,
	"standalone-capable":      fixStandaloneCapable,
	"ios-statusbar-safe-area": fixSafeArea,
	"manifest-completeness":   fixManifestCompleteness,
	"open-graph":              fixOpenGraph,
	"twitter-card":            fixTwitterCard,
	"reduced-motion-support":  fixReducedMotion,
	"public-asset-convention": fixPublicAssetConvention,
}

// fixableRuleIDs is the registry-ordered list of rules that have a deterministic
// fixer (drives BuildFixCandidates' "fix everything" path).
var fixableRuleIDs = func() []string {
	var out []string
	for _, s := range specs {
		if _, ok := fixerByID[s.id]; ok {
			out = append(out, s.id)
		}
	}
	return out
}()

// autofixAvailable reports whether the registered fixer for ruleID can produce a
// candidate for the scenario at root right now. Rules with no fixer are always
// false. This is the SSOT for the finding's AutofixAvailable flag.
func autofixAvailable(ruleID, root string) bool {
	fixer, ok := fixerByID[ruleID]
	if !ok {
		return false
	}
	_, produced, err := fixer(root, false)
	return err == nil && produced
}
