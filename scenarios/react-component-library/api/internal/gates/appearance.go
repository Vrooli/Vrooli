package gates

func delegatedAppearanceResult(gate string) Result {
	return Result{Status: "delegated", Skipped: []string{"catalog handler executes " + gate + " against a version-pinned browser report"}}
}
