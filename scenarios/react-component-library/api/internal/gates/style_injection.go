package gates

func ValidateStyleInjection(scope Scope) (Result, error) {
	return validateActiveSourceFiles(scope, "style-injection", func(_ assetDoc, source string) defect {
		if styleTagRE.MatchString(source) {
			return defect{Message: "renders a style element from component output", Remediation: "Move the stylesheet into a module-level string and mount it with useLibraryStyleSheet so instances share one head node.", DocsRef: "docs/reference/style-ownership.md"}
		}
		return ok()
	})
}

// ValidateForeignTokenClasses is the compatibility name for the superseded
// palette-only gate. Keep it delegated so existing catalog evidence and
// calibration fixtures retain their stable gate identity.
